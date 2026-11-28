//go:build !windows

package lifecycle

import (
	"os"
	"os/signal"
	"syscall"
)

// brokenPipeSignals receives SIGPIPE while a hold is active. Nothing reads it:
// the subscription is what matters, and a signal sent to a full channel is
// dropped.
var brokenPipeSignals = make(chan os.Signal, 1)

func subscribeBrokenPipe() { signal.Notify(brokenPipeSignals, syscall.SIGPIPE) }

func unsubscribeBrokenPipe() { signal.Stop(brokenPipeSignals) }
