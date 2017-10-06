package sentinel

import (
	"context"
	"sync"
	"time"
)

const (
	ManualStop StopReason = iota
	AutomaticStop
)

const (
	InternalTrigger TriggerReason = iota
	ExternalTrigger
)

type StopReason int
type TriggerReason int

type EveryFunction func(ctx context.Context, tReason TriggerReason, tData interface{}) (data interface{}, done bool, err error)
type SuccessFunction func(ctx context.Context, data interface{}) (done bool)
type FailFunction func(ctx context.Context, err error) (done bool)
type FinallyFunction func(ctx context.Context, sReason StopReason)

type sentinel struct {
	Functions
	T chan<- interface{} // For manually triggering
	C <-chan bool        // When the worker is done true is pushed on this channel

	ctx      context.Context // Context
	lock     sync.RWMutex    // Make access to internal data concurrently safe
	iTrigger <-chan time.Time
	stop     chan bool // Tells the worker to stop
	active   bool      // State of the sentinel
	t        chan interface{}
	c        chan bool // Alias for C channel which can be written to internally
}

type Functions struct {
	Every   EveryFunction
	Success SuccessFunction
	Failure FailFunction
	Finally FinallyFunction
}
