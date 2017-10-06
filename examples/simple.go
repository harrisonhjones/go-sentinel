package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"harrisonhjones.com/sentinel"
)

/* This example sets up a sentinel which is triggered every 100 ms and keeps a running count of how many times its been
triggered. Once it reaches 21 times it signals that it's done. However the main application stops the sentinel after ~1
second. The finally function is called with indication that the sentinel was manually stopped.
*/
func main() {
	counter := 0

	s := sentinel.New(context.TODO(), sentinel.Config{
		Duration: time.Millisecond * 100,
		Every: func(ctx context.Context) (data interface{}, done bool, err error) {
			fmt.Printf("Every: %d\n", counter)
			counter++

			if counter > 20 {
				return "oh my god", true, nil
			}

			return "you pass butter", false, nil
		},
		Success: func(ctx context.Context, data interface{}) (done bool) {
			fmt.Printf("Success: %d\n", counter)
			fmt.Printf("\tData: %v\n", data)
			return false
		},
		Failure: func(ctx context.Context, err error) (done bool) {
			fmt.Printf("Failure: %d\n", counter)
			return false
		},
		Finally: func(ctx context.Context, stopped bool) {
			fmt.Printf("Finally: %d\n", counter)
			if stopped {
				fmt.Printf("\tSentinel was manually stopped\n")
				return
			}
			fmt.Printf("\tSentinel was automatically stopped\n")
		},
	})

	fmt.Printf("Started the sentinel\n")
	if err := s.Start(); err != nil {
		fmt.Printf("failed to start: %v", err)
		os.Exit(-1)
	}

	fmt.Printf("Sleeping for 1 second\n")
	time.Sleep(time.Second * 1)

	fmt.Printf("Stopping the sentinel early\n")
	// Stop the sentinel early
	if err := s.Stop(); err != nil {
		fmt.Printf("failed to stop: %v", err)
		os.Exit(-2)
	}

	<-s.Done
	fmt.Printf("Done!")
}
