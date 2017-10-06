package sentinel

import (
	"context"
	"time"

	"github.com/pkg/errors"
)

func New(ctx context.Context, config Config) *sentinel {
	c := make(chan bool)
	return &sentinel{
		Config: config,
		active: false,
		ticker: time.NewTicker(config.Duration),
		stop:   make(chan bool, 1),
		c:      c,
		Done:   c,
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
	s.lock.Lock()
	defer s.lock.Unlock()

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

func (s *sentinel) work() {
	var manuallyStopped = false
Loop:
	for {
		select {
		case <-s.stop:
			// fmt.Printf("worker: signaled to stop\n")
			manuallyStopped = true
			break Loop
		case <-s.ticker.C:
			// fmt.Printf("worker: tick\n")

			data, done, err := s.Every(s.ctx)
			if err != nil {
				// fmt.Printf("worker: every returned with an error: %v\n", err)
				if done := s.Failure(s.ctx, err); done == true {
					// fmt.Printf("worker: failure signaled done\n")
					break
				}
				continue Loop
			}

			if done := s.Success(s.ctx, data); done == true {
				// fmt.Printf("worker: success signaled done\n")
				break Loop
			}

			if done == true {
				// fmt.Printf("worker: every signaled done\n")
				break Loop
			}
		}

	}

	// fmt.Printf("worker: finally\n")
	s.Finally(s.ctx, manuallyStopped)

	// fmt.Printf("worker: signaling done\n")
	s.c <- true
}
