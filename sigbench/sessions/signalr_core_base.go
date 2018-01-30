package sessions

import (
	"bytes"
	"encoding/json"
	//"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/websocket"
	"github.com/vmihailenco/msgpack"
)

type SignalRCoreBase struct {
	cntInProgress    int64
	cntEstablished   int64
	cntError         int64
	cntSuccess       int64
	messageSendCount int64
	messageRecvCount int64
	msgSendSize      int64
	msgRecvSize      int64
	latency          [LatencyArrayLen]int64
}

func (s *SignalRCoreBase) logLatency(latency int64) {
	index := int(latency / LatencyStep)
	if index > LatencyArrayLen-1 {
		index = LatencyArrayLen - 1
	}
	atomic.AddInt64(&s.latency[index], 1)
}

func (s *SignalRCoreBase) logInProgress(c int64) {
	atomic.AddInt64(&s.cntInProgress, c)
}

func (s *SignalRCoreBase) logEstablished(c int64) {
	atomic.AddInt64(&s.cntEstablished, c)
}

func (s *SignalRCoreBase) logMsgSendCount(c int64) {
	atomic.AddInt64(&s.messageSendCount, c)
}

func (s *SignalRCoreBase) logMsgRecvCount(c int64) {
	atomic.AddInt64(&s.messageRecvCount, c)
}

func (s *SignalRCoreBase) logMsgSendSize(c int64) {
	atomic.AddInt64(&s.msgSendSize, c)
}

func (s *SignalRCoreBase) logMsgRecvSize(c int64) {
	atomic.AddInt64(&s.msgRecvSize, c)
}

func (s *SignalRCoreBase) Setup(map[string]string) error {
	s.cntInProgress = 0
	s.cntEstablished = 0
	s.cntError = 0
	s.cntSuccess = 0
	s.messageSendCount = 0
	s.messageRecvCount = 0
	s.msgSendSize = 0
	s.msgRecvSize = 0
	return nil
}

func (s *SignalRCoreBase) logError(msg string, err error) {
	log.Println("Error: ", msg, " due to ", err)
	atomic.AddInt64(&s.cntError, 1)
}

func (s *SignalRCoreBase) Name() string {
	return "SignalRCore:Base"
}

func (s *SignalRCoreBase) counterTag() string {
	return "signalrcore:echo"
}

func (s *SignalRCoreBase) concatStr(v1 string, v2 string) string {
	var buffer bytes.Buffer

	buffer.WriteString(v1)
	buffer.WriteString(v2)
	return buffer.String()
}

var invocationId int = 0

func (s *SignalRCoreBase) sendMsgCore(c *websocket.Conn, msgType int, msg []byte) error {
	time.Sleep(time.Millisecond * time.Duration(rand.Int()%1000))
	err := c.WriteMessage(msgType, msg)
	if err != nil {
		s.logError("Fail to send msg", err)
		return err
	}
	invocationId++
	s.logMsgSendCount(1)
	s.logMsgSendSize(int64(len(msg)))
	return nil
}

func (s *SignalRCoreBase) sendJsonMsg(c *websocket.Conn, target string, arguments []string) error {
	msg, err := SerializeSignalRCoreMessage(&SignalRCoreInvocation{
		Type:         1,
		InvocationId: strconv.Itoa(invocationId),
		Target:       target,
		Arguments:    arguments,
	})
	err = s.sendMsgCore(c, websocket.TextMessage, msg)
	if err != nil {
		return err
	}
	return nil
}

func (s *SignalRCoreBase) sendServiceMsgPackWithNonBlocking(c *websocket.Conn, target string, arguments []string) error {
	var meta map[string]string
	invocation := ServiceMsgpackInvocationWithNonblocking{
		MessageType:  1,
		InvocationId: strconv.Itoa(invocationId),
		NonBlocking:  false,
		Target:       target,
		Arguments:    arguments,
		Meta:         meta,
	}
	msg, err := msgpack.Marshal(&invocation)
	if err != nil {
		s.logError("Fail to pack signalr core message", err)
		return err
	}
	msgPack, err := encodeSignalRBinary(msg)
	if err != nil {
		s.logError("Fail to encode message", err)
		return err
	}

	err = s.sendMsgCore(c, websocket.BinaryMessage, msgPack)
	if err != nil {
		return err
	}
	return nil
}

func (s *SignalRCoreBase) sendMsgPackWithNonBlocking(c *websocket.Conn, target string, arguments []string) error {
	invocation := MsgpackInvocationWithNonblocking{
		MessageType:  1,
		InvocationId: strconv.Itoa(invocationId),
		NonBlocking:  false,
		Target:       target,
		Arguments:    arguments,
	}
	msg, err := msgpack.Marshal(&invocation)
	if err != nil {
		s.logError("Fail to pack signalr core message", err)
		return err
	}
	msgPack, err := encodeSignalRBinary(msg)
	if err != nil {
		s.logError("Fail to encode message", err)
		return err
	}
	err = s.sendMsgCore(c, websocket.BinaryMessage, msgPack)
	if err != nil {
		return err
	}
	return nil
}

func (s *SignalRCoreBase) sendMsgPack(c *websocket.Conn, target string, arguments []string) error {
	invocation := MsgpackInvocation{
		MessageType:  1,
		InvocationId: strconv.Itoa(invocationId),
		Target:       target,
		Arguments:    arguments,
	}
	msg, err := msgpack.Marshal(&invocation)
	if err != nil {
		s.logError("Fail to pack signalr core message", err)
		return err
	}
	msgPack, err := encodeSignalRBinary(msg)
	if err != nil {
		s.logError("Fail to encode message", err)
		return err
	}
	err = s.sendMsgCore(c, websocket.BinaryMessage, msgPack)
	if err != nil {
		return err
	}
	return nil
}

func (s *SignalRCoreBase) signalrCoreConnect(host string, hub string, nego bool) (*websocket.Conn, error) {
	var wsUrl string
	if nego {
		negotiateResponse, err := http.Post("http://"+host+"/"+hub+"/negotiate", "text/plain;charset=UTF-8", nil)
		if err != nil {
			s.logError("Failed to negotiate with the server", err)
			return nil, err
		}
		defer negotiateResponse.Body.Close()

		decoder := json.NewDecoder(negotiateResponse.Body)
		var handshakeContent SignalRCoreHandshakeResp
		err = decoder.Decode(&handshakeContent)
		if err != nil {
			s.logError("Fail to obtain connection id", err)
			return nil, err
		}
		wsUrl = "ws://" + host + "/chat?id=" + handshakeContent.ConnectionId
	} else {
		wsUrl = "ws://" + host + "/" + hub
	}
	c, _, err := websocket.DefaultDialer.Dial(wsUrl, nil)
	if err != nil {
		s.logError("Fail to connect to websocket", err)
		return nil, err
	}
	return c, nil
}

func (s *SignalRCoreBase) signalrCoreServiceConnect(
	endpoint string,
	hub string,
	key string,
	audience string,
	uid string) (*websocket.Conn, error) {
	mySigningKey := []byte(key)
	t := time.Now().Add(time.Second * 30)
	// Create the Claims
	claims := &jwt.StandardClaims{
		ExpiresAt: t.Unix(),
		Audience:  audience,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	ss, _ := token.SignedString(mySigningKey)
	var wsUrl = "ws://" + endpoint + "/client/" + hub + "?signalRTokenHeader=" + ss + "&uid=" + uid
	c, _, err := websocket.DefaultDialer.Dial(wsUrl, nil)
	if err != nil {
		s.logError("Fail to connect to websocket", err)
		return nil, err
	}
	return c, nil
}

func (s *SignalRCoreBase) Counters() map[string]int64 {
	tag := s.counterTag()
	counters := map[string]int64{
		s.concatStr(tag, ":established"):  atomic.LoadInt64(&s.cntEstablished),
		s.concatStr(tag, ":inprogress"):   atomic.LoadInt64(&s.cntInProgress),
		s.concatStr(tag, ":success"):      atomic.LoadInt64(&s.cntSuccess),
		s.concatStr(tag, ":error"):        atomic.LoadInt64(&s.cntError),
		s.concatStr(tag, ":msgsendcount"): atomic.LoadInt64(&s.messageSendCount),
		s.concatStr(tag, ":msgrecvcount"): atomic.LoadInt64(&s.messageRecvCount),
		s.concatStr(tag, ":msgsendsize"):  atomic.LoadInt64(&s.msgSendSize),
		s.concatStr(tag, ":msgrecvsize"):  atomic.LoadInt64(&s.msgRecvSize),
	}
	var buffer bytes.Buffer
	var displayLabel int
	var step int = int(LatencyStep)
	for i := 0; i < LatencyArrayLen; i++ {
		buffer.Reset()
		buffer.WriteString(tag)
		buffer.WriteString(":latency:")
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
