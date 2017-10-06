package sentinel

import (
	"context"
	"sync"
	"time"
)

type EveryFunc func(ctx context.Context) (data interface{}, done bool, err error)
type SuccessFunc func(ctx context.Context, data interface{}) (done bool)
type FailFunc func(ctx context.Context, err error) (done bool)
type FinallyFunc func(ctx context.Context, stopped bool)

type sentinel struct {
	Config
	Done <-chan bool // When the worker is done true is pushed on this channel

	ctx    context.Context // Context
	lock   sync.Mutex      // Make access to internal data concurrently safe
	ticker *time.Ticker    // Triggers the Every func
	stop   chan bool       // Tells the worker to stop
	active bool            // State of the sentinel
	c      chan bool       // Alias for Done channel which can be written to internally
}

type Config struct {
	Duration time.Duration
	Every    EveryFunc
	Success  SuccessFunc
	Failure  FailFunc
	Finally  FinallyFunc
}
