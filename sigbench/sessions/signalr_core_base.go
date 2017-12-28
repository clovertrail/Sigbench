package sessions

import (
	//"encoding/json"
	"bytes"
	//"fmt"
	"log"
	"strconv"
	"sync/atomic"

	//"github.com/vmihailenco/msgpack"
)

type SignalRCoreBase struct {
	cntInProgress    int64
	cntError         int64
	cntSuccess       int64
	messageSendCount int64
	latency          [LatencyArrayLen]int64
}

func (s *SignalRCoreBase) logLatency(latency int64) {
	// log.Println("Latency: ", latency)
	index := int(latency / LatencyStep)
	if index > LatencyArrayLen-1 {
		index = LatencyArrayLen - 1
	}
	atomic.AddInt64(&s.latency[index], 1)
}

func (s *SignalRCoreBase) Setup(map[string]string) error {
	s.cntInProgress = 0
	s.cntError = 0
	s.cntSuccess = 0
	s.messageSendCount = 0
	return nil
}

func (s *SignalRCoreBase) logError(msg string, err error) {
	log.Println("Error: ", msg, " due to ", err)
	atomic.AddInt64(&s.cntError, 1)
}

func (s *SignalRCoreBase) Counters() map[string]int64 {
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
