package capabilities

import (
	"context"
	"fmt"
)

// AggregateChecker reports the state of a capability composed from optional
// dependencies. It is intentionally explicit about partial availability: the
// audio-tools rollup remains usable when at least one local capability is
// ready, while the message exposes the exact healthy/total count.
type AggregateChecker struct {
	Checkers []Checker
}

func (c AggregateChecker) Check(ctx context.Context) (Status, string) {
	if len(c.Checkers) == 0 {
		return StatusUnavailable, "no component capability checkers configured"
	}
	available := 0
	for _, checker := range c.Checkers {
		if checker == nil {
			continue
		}
		status, _ := checker.Check(ctx)
		if status == StatusAvailable {
			available++
		}
	}
	if available == 0 {
		return StatusUnavailable, fmt.Sprintf("no component capabilities are available (0/%d)", len(c.Checkers))
	}
	return StatusAvailable, fmt.Sprintf("%d/%d component capabilities are available", available, len(c.Checkers))
}
