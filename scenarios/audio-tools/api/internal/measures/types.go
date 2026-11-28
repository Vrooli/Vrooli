// Package measures owns stable product contracts for aggregate operational
// measures. Lab repositories may implement these read contracts.
package measures

import (
	"context"
	"time"
)

// ExperimentCreatedCounts is an aggregate-only measure, not experiment state.
type ExperimentCreatedCounts struct {
	Experiments int64
	Runs        int64
}

// ExperimentCounter is the product read contract implemented by the
// experiment persistence service.
type ExperimentCounter interface {
	CountCreatedInRange(context.Context, time.Time, time.Time) (ExperimentCreatedCounts, error)
}
