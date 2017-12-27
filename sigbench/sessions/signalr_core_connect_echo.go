package sessions

import (
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
	cntInProgress int64
	cntError      int64
	cntSuccess    int64
	cntLatencyLessThan100ms  int64
	cntLatencyLessThan500ms  int64
	cntLatencyLessThan1000ms int64
	cntLatencyMoreThan1000ms int64
}

func (s *SignalRConnCoreEcho) Name() string {
	return "SignalRCore:ConnectEcho"
}

func (s *SignalRConnCoreEcho) logLatency(latency int64) {
	// log.Println("Latency: ", latency)
	if latency < 100 {
		atomic.AddInt64(&s.cntLatencyLessThan100ms, 1)
	} else if latency < 500 {
		atomic.AddInt64(&s.cntLatencyLessThan500ms, 1)
	} else if latency < 1000 {
		atomic.AddInt64(&s.cntLatencyLessThan1000ms, 1)
	} else {
		atomic.AddInt64(&s.cntLatencyMoreThan1000ms, 1)
	}
}

func (s *SignalRConnCoreEcho) Setup(map[string]string) error {
	s.cntInProgress = 0
	s.cntError = 0
	s.cntSuccess = 0
	s.cntLatencyLessThan100ms = 0
	s.cntLatencyLessThan500ms = 0
	s.cntLatencyLessThan1000ms = 0
	s.cntLatencyMoreThan1000ms = 0
	return nil
}

func (s *SignalRConnCoreEcho) logError(msg string, err error) {
	log.Println("Error: ", msg, " due to ", err)
	atomic.AddInt64(&s.cntError, 1)
}

func (s *SignalRConnCoreEcho) Execute(ctx *UserContext) error {
	atomic.AddInt64(&s.cntInProgress, 1)
	defer atomic.AddInt64(&s.cntInProgress, -1)

	host := ctx.Params[ParamHost]
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
	wsUrl := "ws://" + host + "/chat?id=" + handshakeContent.ConnectionId
	c, _, err := websocket.DefaultDialer.Dial(wsUrl, nil)
	if err != nil {
		s.logError("Fail to connect to websocket", err)
		return err
	}
	defer c.Close()

	startSend := make(chan int)
	doneChan := make(chan struct{})
	recvChan := make(chan int64)

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
				if content.Target == "start" {
					startSend <- 1
				}
				if content.Target == "echo" {
					startTime, _ := strconv.ParseInt(content.Arguments[1], 10, 64)
					recvChan <- (time.Now().UnixNano() - startTime) / 1000000
				}
			}
		}
	}()

	err = c.WriteMessage(websocket.TextMessage, []byte("{\"protocol\":\"json\"}\x1e"))
	if err != nil {
		s.logError("Fail to set protocol", err)
		return err
	}
	<-startSend
	//log.Println("Server informs to send")
	msg, err := SerializeSignalRCoreMessage(&SignalRCoreInvocation{
		Type:         1,
		InvocationId: "0",
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

	// Wait echo response
	select {
	case <-time.After(1 * time.Minute):
		s.logError("Fail to receive echo within timeout", nil)
		return errors.New("fail to receive echo within timeout")
	case latency := <-recvChan:
		s.logLatency(latency)
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
	return map[string]int64{
		"signalrcore:echo:inprogress": atomic.LoadInt64(&s.cntInProgress),
		"signalrcore:echo:success":    atomic.LoadInt64(&s.cntSuccess),
		"signalrcore:echo:error":      atomic.LoadInt64(&s.cntError),
		"signalrcore:echo:latency:lt_100":   atomic.LoadInt64(&s.cntLatencyLessThan100ms),
		"signalrcore:echo:latency:lt_500":   atomic.LoadInt64(&s.cntLatencyLessThan500ms),
		"signalrcore:echo:latency:lt_1000":  atomic.LoadInt64(&s.cntLatencyLessThan1000ms),
		"signalrcore:echo:latency:ge_1000": atomic.LoadInt64(&s.cntLatencyMoreThan1000ms),
	}
}
