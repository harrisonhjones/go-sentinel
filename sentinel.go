// Sentinel provides a convenient way to periodically call a function, handle the success or failure of that
// function and then cleanup when done. Periodic triggering Sentinel happens automatically if desired and it can be
// manually triggered when needed. All called functions are optional. Sentinel could be used to periodically ping a
// web resource to determine if it has changed. If so, stop polling and perform some action and cleanup. If not, keep
// polling. Alternately Sentinel could we used to indefinitely poll a remote resource, update a local cache, and only
// cleanup if an error occurs multiple times.
// TODO: More documentation
package sentinel

import (
	"context"
	"time"

	"github.com/pkg/errors"
)

// New returns a new inactive Sentinel which, when Started, is automatically triggered on the specified duration.
// If a zero duration is provided the Sentinel will not automatically trigger and must be triggered manually.
func New(ctx context.Context, duration time.Duration, functions Functions) *sentinel {
	c := make(chan bool, 1)
	t := make(chan interface{}, 1)

	// If duration is positive create a new ticker to automatically trigger the Sentinel when started.
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

// Start starts the Sentinel which allows it to be triggered automatically and/or manually.
// If the Sentinel is already active an error will be returned.
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

// Stop stops the Sentinel, triggering the provided FinallyFunction to run, and marks it inactive.
// If the Sentinel is already inactive or stopping an error is returned.
// The Sentinel can be restarted by calling the Start function again.
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

// IsActive returns with the active state of the Sentinel.
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

	// Cleanly shutdown the Sentinel
	s.shutdown()
}

func (s *sentinel) shutdown() {
	s.lock.Lock()
	defer s.lock.Unlock()

	// Drain the manual stop channel
	select {
	case <-s.stop:
	default:
	}

	// Mark the Sentinel as inactive
	s.active = false

	// Signal that the Sentinel has stopped
	select {
	case s.c <- true:
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
