// Package scheduler exposes the control-plane name for the shared periodic
// runner. Scenario modules consume the equivalent public packages/scheduler
// path because Go's internal import boundary intentionally excludes nested
// scenario modules.
package scheduler

import (
	"context"
	"time"

	shared "github.com/vrooli/vrooli/packages/scheduler"
)

type (
	Cycle  = shared.Cycle
	Stats  = shared.Stats
	Runner = shared.Runner
)

func New(interval time.Duration, run func(context.Context) error) *Runner {
	return shared.New(interval, run)
}
