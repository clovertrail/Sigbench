package sessions

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

type SignalRConnCoreEcho struct {
	cntInProgress    int64
	cntError         int64
	cntSuccess       int64
	messageSendCount int64
	latency          [LatencyArrayLen]int64
}

func (s *SignalRConnCoreEcho) Name() string {
	return "SignalRCore:ConnectEcho"
}

func (s *SignalRConnCoreEcho) logLatency(latency int64) {
	// log.Println("Latency: ", latency)
	index := int(latency / LatencyStep)
	if index > LatencyArrayLen-1 {
		index = LatencyArrayLen - 1
	}
	atomic.AddInt64(&s.latency[index], 1)
}

func (s *SignalRConnCoreEcho) Setup(map[string]string) error {
	s.cntInProgress = 0
	s.cntError = 0
	s.cntSuccess = 0
	s.messageSendCount = 0
	return nil
}

func (s *SignalRConnCoreEcho) logError(msg string, err error) {
	log.Println("Error: ", msg, " due to ", err)
	atomic.AddInt64(&s.cntError, 1)
}

func (s *SignalRConnCoreEcho) Execute(ctx *UserContext) error {
	atomic.AddInt64(&s.cntInProgress, 1)
	defer atomic.AddInt64(&s.cntInProgress, -1)

	useNego := ctx.Params[ParamUseNego]
	host := ctx.Params[ParamHost]
	lazySending := ctx.Params[ParamLazySending]
	negotiateResponse, err := http.Post("http://"+host+"/chat/negotiate", "text/plain;charset=UTF-8", nil)
	if err != nil {
		s.logError("Failed to negotiate with the server", err)
		return err
	}
	defer negotiateResponse.Body.Close()

	decoder := json.NewDecoder(negotiateResponse.Body)
	var handshakeContent SignalRCoreHandshakeResp
	err = decoder.Decode(&handshakeContent)
	if err != nil {
		s.logError("Fail to obtain connection id", err)
		return err
	}
	var wsUrl string
	if useNego == "true" {
		negotiateResponse, err := http.Post("http://"+host+"/chat/negotiate", "text/plain;charset=UTF-8", nil)
		if err != nil {
			s.logError("Failed to negotiate with the server", err)
			return err
		}
		defer negotiateResponse.Body.Close()

		decoder := json.NewDecoder(negotiateResponse.Body)
		var handshakeContent SignalRCoreHandshakeResp
		err = decoder.Decode(&handshakeContent)
		if err != nil {
			s.logError("Fail to obtain connection id", err)
			return err
		}
		wsUrl = "ws://" + host + "/chat?id=" + handshakeContent.ConnectionId
	} else {
		wsUrl = "ws://" + host + "/chat"
	}
	c, _, err := websocket.DefaultDialer.Dial(wsUrl, nil)
	if err != nil {
		s.logError("Fail to connect to websocket", err)
		return err
	}
	defer c.Close()

	startSend := make(chan int)
	doneChan := make(chan struct{})

	go func() {
		defer close(doneChan)
		for {
			_, msgWithTerm, err := c.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure) {
					s.logError("Fail to read incoming message", err)
				}
				return
			}

			msg := msgWithTerm[:len(msgWithTerm)-1]
			var content SignalRCoreInvocation
			err = json.Unmarshal(msg, &content)
			if err != nil {
				s.logError("Fail to decode incoming message", err)
				return
			}

			if content.Type == 1 {
				if lazySending == "true" {
					if content.Target == "start" {
						startSend <- 1
					}
				}
				if content.Target == "echo" {
					startTime, _ := strconv.ParseInt(content.Arguments[1], 10, 64)
					s.logLatency((time.Now().UnixNano() - startTime) / 1000000)
				}
			}
		}
	}()

	err = c.WriteMessage(websocket.TextMessage, []byte("{\"protocol\":\"json\"}\x1e"))
	if err != nil {
		s.logError("Fail to set protocol", err)
		return err
	}
	// waiting until receiving "start" command
	if lazySending == "true" {
		<-startSend
	}

	invocationId := 0
	sendMessage := func() error {
		msg, err := SerializeSignalRCoreMessage(&SignalRCoreInvocation{
			Type:         1,
			InvocationId: strconv.Itoa(invocationId),
			Target:       "echo",
			Arguments: []string{
				ctx.UserId,
				strconv.FormatInt(time.Now().UnixNano(), 10),
			},
		})
		err = c.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			s.logError("Fail to send echo", err)
			return err
		}
		invocationId++
		atomic.AddInt64(&s.messageSendCount, 1)
		return nil
	}
	if err = sendMessage(); err != nil {
		return err
	}
	repeatEcho := ctx.Params[ParamRepeatEcho]
	if repeatEcho == "true" {
		ticker := time.NewTicker(time.Second)
		for range ticker.C {
			if err = sendMessage(); err != nil {
				ticker.Stop()
				return err
			}
		}
	}

	// close websocket
	err = c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		s.logError("Fail to close websocket gracefully", err)
		return err
	}
	// Wait close response
	select {
	case <-time.After(1 * time.Minute):
		s.logError("Fail to receive close message", nil)
		return errors.New("fail to receive close message")
	case <-doneChan:
		atomic.AddInt64(&s.cntSuccess, 1)
		return nil
	}
}

func (s *SignalRConnCoreEcho) Counters() map[string]int64 {
	counters := map[string]int64{
		"signalrcore:echo:inprogress":   atomic.LoadInt64(&s.cntInProgress),
		"signalrcore:echo:success":      atomic.LoadInt64(&s.cntSuccess),
		"signalrcore:echo:error":        atomic.LoadInt64(&s.cntError),
		"signalrcore:echo:msgsendcount": atomic.LoadInt64(&s.messageSendCount),
	}
	var buffer bytes.Buffer
	var displayLabel int
	var step int = int(LatencyStep)
	for i := 0; i < LatencyArrayLen; i++ {
		buffer.Reset()
		buffer.WriteString("signalrcore:echo:latency:")
		if i < LatencyArrayLen-1 {
			displayLabel = int(i*step + step)
			buffer.WriteString("lt_")
		} else {
			displayLabel = int(i * step)
			buffer.WriteString("ge_")
		}
		buffer.WriteString(strconv.Itoa(displayLabel))
		counters[buffer.String()] = s.latency[i]
	}

	return counters
}
