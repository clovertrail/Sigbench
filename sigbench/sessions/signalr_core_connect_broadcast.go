package sessions

import (
	//"bytes"
	"encoding/json"
	"errors"
	//"log"
	//"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

type SignalRConnCoreBroadcast struct {
	SignalRCoreBase
}

func (s *SignalRConnCoreBroadcast) Name() string {
	return "SignalRCore:ConnectBroadcast"
}

func (s *SignalRConnCoreBroadcast) Execute(ctx *UserContext) error {
	s.logInProgress(1)
	var target string
	target = "broadcastMessage"
	useNego := ctx.Params[ParamUseNego]
	host := ctx.Params[ParamHost]
	lazySending := ctx.Params[ParamLazySending]
	hub := ctx.Params[ParamHub]
	c, err := s.signalrCoreConnect(host, hub, useNego == "true")
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
				s.logError("Fail to decode incoming message", err)
				return
			}

			if content.Type == 1 {
				if lazySending == "true" {
					if content.Target == "start" {
						startSend <- 1
					}
				}
				if content.Target == target {
					s.logMsgRecvCount(1)
					s.logMsgRecvSize(int64(len(msgWithTerm)))
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
	sendMessage := func() error {
		return s.sendJsonMsg(c,
			target,
			[]string {
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
