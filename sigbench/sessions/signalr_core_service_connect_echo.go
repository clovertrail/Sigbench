package sessions

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	//"log"
	//"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

type SignalRServiceConnCoreEcho struct {
	SignalRCoreBase
	client2ServiceInternalLat  [InternalLatencyLength]int64
	service2ServerExternalLat  [ExternalLatencyLength]int64
	server2ServiceExternalLat  [ExternalLatencyLength]int64
	service2ServiceExternalLat [ExternalLatencyLength]int64
	service2ClientInternalLat  [InternalLatencyLength]int64
	serverInternalLat          [InternalLatencyLength]int64

	enableMetrics bool
}

func (s *SignalRServiceConnCoreEcho) logClient2ServiceInternalLat(latency int64) {
	index := int(latency / InternalLatencyStep)
	if index > InternalLatencyLength-1 {
		index = InternalLatencyLength - 1
	}
	atomic.AddInt64(&s.client2ServiceInternalLat[index], 1)
}

func (s *SignalRServiceConnCoreEcho) logService2ClientInternalLat(latency int64) {
	index := int(latency / InternalLatencyStep)
	if index > InternalLatencyLength-1 {
		index = InternalLatencyLength - 1
	}
	atomic.AddInt64(&s.service2ClientInternalLat[index], 1)
}

func (s *SignalRServiceConnCoreEcho) logServerInternalLat(latency int64) {
	index := int(latency / InternalLatencyStep)
	if index > InternalLatencyLength-1 {
		index = InternalLatencyLength - 1
	}
	atomic.AddInt64(&s.serverInternalLat[index], 1)
}

func (s *SignalRServiceConnCoreEcho) logService2ServerExternalLat(latency int64) {
	index := int(latency / ExternalLatencyStep)
	if index > ExternalLatencyLength-1 {
		index = ExternalLatencyLength - 1
	} else if index < 0 {
		fmt.Printf("Negative latency %d\n", latency)
		return
	}
	atomic.AddInt64(&s.service2ServerExternalLat[index], 1)
}

func (s *SignalRServiceConnCoreEcho) logServer2ServiceExternalLat(latency int64) {
	index := int(latency / ExternalLatencyStep)
	if index > ExternalLatencyLength-1 {
		index = ExternalLatencyLength - 1
	}
	atomic.AddInt64(&s.server2ServiceExternalLat[index], 1)
}

func (s *SignalRServiceConnCoreEcho) logService2ServiceExternalLat(latency int64) {
	index := int(latency / ExternalLatencyStep)
	if index > ExternalLatencyLength-1 {
		index = ExternalLatencyLength - 1
	}
	atomic.AddInt64(&s.service2ServiceExternalLat[index], 1)
}

func (s *SignalRServiceConnCoreEcho) Name() string {
	return "SignalRCoreService:ConnectEcho"
}

func (s *SignalRServiceConnCoreEcho) Execute(ctx *UserContext) error {
	s.logInProgress(1)

	lazySending := ctx.Params[ParamLazySending]
	appHost := ctx.Params[ParamAppHost]
	if ctx.Params[ParamEnableMetrics] == "true" {
		s.enableMetrics = true
	}
	c, err := s.signalrCoreServiceConnectApp(appHost)
	defer c.Close()

	startSend := make(chan int)
	doneChan := make(chan struct{})

	go func() {
		defer close(doneChan)
		established := false
		for {
			_, msgWithTerm, err := c.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure) {
					s.logError("Fail to read incoming message", err)
				}
				return
			}

			if !established {
				s.logEstablished(1)
				s.logInProgress(-1)
				established = true
			}
			msg := msgWithTerm[:len(msgWithTerm)-1]
			var content SignalRCoreInvocation
			err = json.Unmarshal(msg, &content)
			if err != nil {
				fmt.Printf("%s\n", content)
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
					s.logMsgRecvCount(1)
					s.logMsgRecvSize(int64(len(msgWithTerm)))
					startTime, _ := strconv.ParseInt(content.Arguments[1], 10, 64)
					s.logLatency((time.Now().UnixNano() - startTime) / 1000000)
				}
			} else if content.Type == 3 {
				var complete SignalRCoreServiceCompletion
				err = json.Unmarshal(msg, &complete)
				if err != nil {
					s.logError("Fail to decode incoming message", err)
					return
				}
				if complete.Meta != nil && complete.Meta["A"] != "" && complete.Meta["B"] != "" &&
					complete.Meta["C"] != "" && complete.Meta["D"] != "" &&
					complete.Meta["E"] != "" && complete.Meta["F"] != "" {
					timeServiceRecvClient, _ := strconv.ParseInt(complete.Meta["A"], 10, 64)
					timeServiceSendServer, _ := strconv.ParseInt(complete.Meta["B"], 10, 64)
					timeServerRecvService, _ := strconv.ParseInt(complete.Meta["C"], 10, 64)
					timeServerSendService, _ := strconv.ParseInt(complete.Meta["D"], 10, 64)
					timeServiceRecvServer, _ := strconv.ParseInt(complete.Meta["E"], 10, 64)
					timeServiceSendClient, _ := strconv.ParseInt(complete.Meta["F"], 10, 64)
					s.logClient2ServiceInternalLat((timeServiceSendServer - timeServiceRecvClient))
					s.logService2ServiceExternalLat((timeServiceRecvServer - timeServiceSendServer))
					s.logService2ServerExternalLat((timeServerRecvService - timeServiceSendServer))
					s.logServer2ServiceExternalLat((timeServiceRecvServer - timeServerSendService))
					s.logService2ClientInternalLat((timeServiceSendClient - timeServiceRecvServer))
				}
				if complete.Meta != nil && complete.Meta["C"] != "" && complete.Meta["D"] != "" {
					timeServerRecvService, _ := strconv.ParseInt(complete.Meta["C"], 10, 64)
					timeServerSendService, _ := strconv.ParseInt(complete.Meta["D"], 10, 64)
					s.logServerInternalLat((timeServerSendService - timeServerRecvService))
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
	sendMessage := func() error {
		return s.sendJsonMsg(c,
			"echo",
			[]string{
				ctx.UserId,
				strconv.FormatInt(time.Now().UnixNano(), 10),
			})
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

func (s *SignalRServiceConnCoreEcho) Counters() map[string]int64 {
	counters := s.SignalRCoreBase.Counters()
	if s.enableMetrics {
		tag := s.counterTag()
		var buffer bytes.Buffer
		var displayLabel int
		var step int = int(InternalLatencyStep)
		for i := 0; i < InternalLatencyLength; i++ {
			buffer.Reset()
			buffer.WriteString(tag)
			buffer.WriteString(":latency:")
			buffer.WriteString("client2serviceIn")
			if i < InternalLatencyLength-1 {
				displayLabel = int(i*step + step)
				buffer.WriteString("lt_")
			} else {
				displayLabel = int(i * step)
				buffer.WriteString("ge_")
			}
			buffer.WriteString(strconv.Itoa(displayLabel))
			counters[buffer.String()] = s.client2ServiceInternalLat[i]

			buffer.Reset()
			buffer.WriteString(tag)
			buffer.WriteString(":latency:")
			buffer.WriteString("service2clientIn")
			if i < InternalLatencyLength-1 {
				displayLabel = int(i*step + step)
				buffer.WriteString("lt_")
			} else {
				displayLabel = int(i * step)
				buffer.WriteString("ge_")
			}
			buffer.WriteString(strconv.Itoa(displayLabel))
			counters[buffer.String()] = s.service2ClientInternalLat[i]

			buffer.Reset()
			buffer.WriteString(tag)
			buffer.WriteString(":latency:")
			buffer.WriteString("serverIn")
			if i < InternalLatencyLength-1 {
				displayLabel = int(i*step + step)
				buffer.WriteString("lt_")
			} else {
				displayLabel = int(i * step)
				buffer.WriteString("ge_")
			}
			buffer.WriteString(strconv.Itoa(displayLabel))
			counters[buffer.String()] = s.serverInternalLat[i]
		}

		step = int(ExternalLatencyStep)
		for i := 0; i < ExternalLatencyLength; i++ {
			buffer.Reset()
			buffer.WriteString(tag)
			buffer.WriteString(":latency:")
			buffer.WriteString("service2serverExt")
			if i < ExternalLatencyLength-1 {
				displayLabel = int(i*step + step)
				buffer.WriteString("lt_")
			} else {
				displayLabel = int(i * step)
				buffer.WriteString("ge_")
			}
			buffer.WriteString(strconv.Itoa(displayLabel))
			counters[buffer.String()] = s.service2ServerExternalLat[i]

			buffer.Reset()
			buffer.WriteString(tag)
			buffer.WriteString(":latency:")
			buffer.WriteString("server2serviceExt")
			if i < ExternalLatencyLength-1 {
				displayLabel = int(i*step + step)
				buffer.WriteString("lt_")
			} else {
				displayLabel = int(i * step)
				buffer.WriteString("ge_")
			}
			buffer.WriteString(strconv.Itoa(displayLabel))
			counters[buffer.String()] = s.server2ServiceExternalLat[i]

			buffer.Reset()
			buffer.WriteString(tag)
			buffer.WriteString(":latency:")
			buffer.WriteString("service2serviceExt")
			if i < ExternalLatencyLength-1 {
				displayLabel = int(i*step + step)
				buffer.WriteString("lt_")
			} else {
				displayLabel = int(i * step)
				buffer.WriteString("ge_")
			}
			buffer.WriteString(strconv.Itoa(displayLabel))
			counters[buffer.String()] = s.service2ServiceExternalLat[i]
		}
	}
	return counters
}
