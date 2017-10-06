package sentinel

import (
	"context"
	"time"

	"github.com/pkg/errors"
)

func New(ctx context.Context, duration time.Duration, functions Functions) *sentinel {
	c := make(chan bool)
	t := make(chan interface{})

	var trigger <-chan time.Time
	if duration > 0 {
		trigger = time.NewTicker(duration).C
	} else {
		trigger = nil
	}

	return &sentinel{
		Functions: functions,
		T:         t,
		C:         c,

		active:   false,
		iTrigger: trigger,
		stop:     make(chan bool, 1),
		t:        t,
		c:        c,
	}
}

func (s *sentinel) Start() error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.active {
		return errors.New("already started")
	}

	s.active = true
	go s.work()

	return nil
}

func (s *sentinel) Stop() error {
	s.lock.RLock()
	defer s.lock.RUnlock()
	if !s.active {
		return errors.New("must be active to stop")
	}

	select {
	case s.stop <- true:
		return nil
	default:
		return errors.New("stop already requested")
	}
}

func (s *sentinel) IsActive() bool {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.active
}

func (s *sentinel) work() {
	var reason StopReason = AutomaticStop
Loop:
	for {
		select {
		case <-s.stop:
			// fmt.Printf("worker: signaled to stop\n")
			reason = ManualStop
			break Loop
		case tData := <-s.iTrigger:
			// fmt.Printf("worker: internal trigger\n")
			if done := s.trigger(InternalTrigger, tData); done {
				break Loop
			}
		case tData := <-s.t:
			// fmt.Printf("worker: external trigger\n")
			if done := s.trigger(ExternalTrigger, tData); done {
				break Loop
			}
		}
	}

	if s.Finally != nil {
		// fmt.Printf("worker: finally\n")
		s.Finally(s.ctx, reason)
	}

	// fmt.Printf("worker: signaling done\n")
	s.lock.Lock()
	defer s.lock.Unlock()
	s.active = false
	select {
	case s.stop <- true:
		s.c <- true
	default:
	}
}

func (s *sentinel) trigger(tReason TriggerReason, tData interface{}) bool {

	if s.Every == nil {
		// fmt.Printf("trigger: every is nil, signaling done\n")
		return true
	}

	data, done, err := s.Every(s.ctx, tReason, tData)
	if err != nil {
		// fmt.Printf("trigger: every returned with an error: %v\n", err)

		if s.Failure == nil {
			// fmt.Printf("trigger: failure is nil, signaling done\n")
			return true
		}

		if done := s.Failure(s.ctx, err); done == true {
			// fmt.Printf("trigger: failure signaled done\n")
			return true
		}
		return false
	}

	if s.Success == nil {
		// fmt.Printf("trigger: success is nil, signaling done\n")
		return true
	}

	if done := s.Success(s.ctx, data); done == true {
		// fmt.Printf("trigger: success signaled done\n")
		return true
	}

	if done == true {
		// fmt.Printf("trigger: every signaled done\n")
		return true
	}

	return false
}
