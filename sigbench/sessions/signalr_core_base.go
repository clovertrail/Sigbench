package sessions

import (
	//"encoding/json"
	"bytes"
	//"fmt"
	"log"
	"strconv"
	"sync/atomic"

	//"github.com/vmihailenco/msgpack"
	"github.com/gorilla/websocket"
)

type SignalRCoreBase struct {
	cntInProgress    int64
	cntEstablished int64
	cntError         int64
	cntSuccess       int64
	messageSendCount int64
	messageRecvCount int64
	msgSendSize int64
	msgRecvSize int64
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

func (s *SignalRCoreBase) concatStr(v1 string, v2 string) string {
	var buffer bytes.Buffer

	buffer.WriteString(v1)
	buffer.WriteString(v2)
	return buffer.String()
}

var invocationId int = 0
func (s *SignalRCoreBase) sendJsonMsg(c *websocket.Conn, target string, arguments []string) error {
	msg, err := SerializeSignalRCoreMessage(&SignalRCoreInvocation{
		Type:         1,
		InvocationId: strconv.Itoa(invocationId),
		Target:       target,
		Arguments: arguments,
	})
	err = c.WriteMessage(websocket.TextMessage, msg)
	if err != nil {
		s.logError("Fail to send echo", err)
		return err
	}
	invocationId++
	s.logMsgSendCount(1)
	s.logMsgSendSize(int64(len(msg)))
	return nil
}

func (s *SignalRCoreBase) Counters() map[string]int64 {
	tag := s.Name()
	counters := map[string]int64{
		s.concatStr(tag, ":established"): atomic.LoadInt64(&s.cntEstablished),
		s.concatStr(tag, ":inprogress"): atomic.LoadInt64(&s.cntInProgress),
		s.concatStr(tag, ":success"):      atomic.LoadInt64(&s.cntSuccess),
		s.concatStr(tag, ":error"):        atomic.LoadInt64(&s.cntError),
		s.concatStr(tag, ":msgsendcount"): atomic.LoadInt64(&s.messageSendCount),
		s.concatStr(tag, ":msgrecvcount"): atomic.LoadInt64(&s.messageRecvCount),
		s.concatStr(tag, ":msgsendsize"): atomic.LoadInt64(&s.msgSendSize),
		s.concatStr(tag, ":msgrecvsize"): atomic.LoadInt64(&s.msgRecvSize),
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
