package sessions

import (
	"log"
	"sync/atomic"
)

type Session interface {
	Name() string
	Setup() error
	Execute(*SessionContext) error
	Counters() map[string]int64
}

var SessionMap = map[string]Session{
	"signalrcore:echo":             &SignalRCoreEcho{},
	"signalrcore:broadcast:sender": &SignalRCoreBroadcastSender{},
}

type DummySession struct {
	counterA int64
	counterB int64
}

func (s *DummySession) Name() string {
	return "Dummy"
}

func (s *DummySession) Setup() error {
	log.Println("> Dummy setup")
	s.counterA = 0
	s.counterB = 0
	return nil
}

func (s *DummySession) Execute(ctx *SessionContext) error {
	log.Println("> Dummy at phase: " + ctx.Phase)
	atomic.AddInt64(&s.counterA, 1)
	atomic.AddInt64(&s.counterB, 2)
	return nil
}

func (s *DummySession) Counters() map[string]int64 {
	return map[string]int64{
		"a": atomic.LoadInt64(&s.counterA),
		"b": atomic.LoadInt64(&s.counterB),
	}
}
