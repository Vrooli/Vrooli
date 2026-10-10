// The pinned nonmodel probe only remains alive until its fixed deadline or
// termination signal in continuing mode. First mode observes only fixed nonsecret
// canary markers and starts the same pinned helper descendant, without sockets
// or model calls.
package main

import (
	"context"
	"errors"
	"github.com/vrooli/vrooli/internal/nativeisolation/probeprotocol"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func deadline(args []string, now time.Time) (time.Time, error) {
	if len(args) != 2 || args[0] != "--until" {
		return time.Time{}, errors.New("invalid fixed probe arguments")
	}
	d, e := time.Parse(time.RFC3339Nano, args[1])
	if e != nil || args[1] != d.UTC().Format(time.RFC3339Nano) || !now.Before(d) || d.Sub(now) > 8*time.Hour {
		return time.Time{}, errors.New("invalid fixed probe deadline")
	}
	return d, nil
}

func main() {
	args := os.Args[1:]
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if len(args) == 3 && args[0] == "--first" && args[1] == "--contract" {
		o, e := probeprotocol.Decode(args[2], time.Now().UTC())
		if e != nil || observe(ctx, o, time.Now().UTC(), fixedObservationEffects{}) != nil {
			os.Exit(2)
		}
		return
	}
	descendant := len(args) == 3 && args[0] == "--descendant"
	if descendant {
		args = args[1:]
	}
	d, e := deadline(args, time.Now().UTC())
	if e != nil || (descendant && d.Sub(time.Now().UTC()) > 5*time.Minute) {
		os.Exit(2)
	}
	timer := time.NewTimer(time.Until(d))
	defer timer.Stop()
	select {
	case <-ctx.Done():
	case <-timer.C:
	}
}
