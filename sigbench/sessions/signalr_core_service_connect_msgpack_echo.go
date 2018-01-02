package sessions

import (
	//"encoding/json"
	"errors"
	//"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/vmihailenco/msgpack"
)

type SignalRServiceConnMsgPackEcho struct {
	SignalRCoreBase
}

func (s *SignalRServiceConnMsgPackEcho) Name() string {
	return "SignalRServiceConnMsgPackCore:ConnectEcho"
}

func (s *SignalRServiceConnMsgPackEcho) Execute(ctx *UserContext) error {
	s.logInProgress(1)

	host := ctx.Params[ParamHost]
	lazySending := ctx.Params[ParamLazySending]
	c, err := s.signalrCoreServiceConnect(host, "chat",
			"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789", host+"/client/", ctx.UserId)
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
			msg, err := decodeSignalRBinary(msgWithTerm)
			if err != nil {
				s.logError("Fail to decode msgpack", err)
				return
			}
			var content MsgpackInvocation
			err = msgpack.Unmarshal(msg, &content)
			if err != nil {
				s.logError("Fail to decode incoming message", err)
				return
			}

			s.logMsgRecvCount(1)
			s.logMsgRecvSize(int64(len(msgWithTerm)))
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
	}()

	err = c.WriteMessage(websocket.TextMessage, []byte("{\"protocol\":\"messagepack\"}\x1e"))
	if err != nil {
		s.logError("Fail to set protocol", err)
		return err
	}
	// waiting until receiving "start" command
	if lazySending == "true" {
		<-startSend
	}
	// Send message
	sendMessage := func() error {
		return s.sendMsgPack(c,
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
