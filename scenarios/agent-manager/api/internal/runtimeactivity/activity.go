// Package runtimeactivity projects current execution observations for ordinary
// product telemetry. It does not expose admission fences or lifecycle authority.
package runtimeactivity

import (
	"context"
	"fmt"
	"strings"
	"time"

	"agent-manager/internal/maintenance"
)

type Observer interface {
	ObserveActivity(context.Context) (maintenance.Inventory, error)
}

// Snapshot counts physically observed execution scopes, not logical run states.
// A missing ExecutingRuns is unknown, even if ObservedExecutingRuns is zero.
// LogicalQueuedRuns is independent pending/starting metadata, never work pulse.
type Snapshot struct {
	ObservedAt            time.Time
	Complete              bool
	ExecutingRuns         *uint32
	ObservedExecutingRuns uint32
	LogicalQueuedRuns     uint32
	Unknown               []string
}

type Reader struct{ observer Observer }

func New(observer Observer) *Reader { return &Reader{observer: observer} }

func (r *Reader) Read(ctx context.Context) Snapshot {
	result := Snapshot{Unknown: []string{}}
	if r == nil || r.observer == nil {
		result.Unknown = append(result.Unknown, "physical execution observer unavailable")
		return result
	}
	inventory, err := r.observer.ObserveActivity(ctx)
	result.ObservedAt = inventory.ObservedAt
	for _, reason := range inventory.Unknown {
		result.addUnknown(reason)
	}
	if err != nil {
		result.addUnknown(err.Error())
	}
	if ctx.Err() != nil {
		result.addUnknown(ctx.Err().Error())
	}
	if inventory.ObservedExecutors < 0 || inventory.LogicalQueuedRuns < 0 || uint64(inventory.ObservedExecutors) > uint64(^uint32(0)) || uint64(inventory.LogicalQueuedRuns) > uint64(^uint32(0)) {
		result.addUnknown("invalid runtime observation counts")
		return result
	}
	result.ObservedExecutingRuns = uint32(inventory.ObservedExecutors)
	result.LogicalQueuedRuns = uint32(inventory.LogicalQueuedRuns)
	if result.ObservedAt.IsZero() {
		result.addUnknown("physical observation timestamp unavailable")
	}
	if err == nil && !inventory.PhysicalComplete && len(result.Unknown) < 64 {
		result.addUnknown("physical execution scope coverage is incomplete")
	}
	if err == nil && inventory.PhysicalComplete && len(result.Unknown) == 0 {
		result.Complete = true
		count := result.ObservedExecutingRuns
		result.ExecutingRuns = &count
	}
	return result
}

func (s *Snapshot) addUnknown(reason string) {
	if len(s.Unknown) >= 64 {
		return
	}
	reason = strings.ToValidUTF8(reason, "?")
	text := []rune(reason)
	if len(text) > 512 {
		reason = string(text[:512]) + " (truncated)"
	}
	s.Unknown = append(s.Unknown, reason)
}

func (s Snapshot) Reason() string {
	if s.Complete {
		return "Complete canonical physical execution-scope observation; queued records are not executing work."
	}
	if s.ObservedExecutingRuns > 0 {
		return fmt.Sprintf("Physical execution is present (at least %d scopes); total coverage is incomplete.", s.ObservedExecutingRuns)
	}
	return "Physical execution coverage is incomplete; zero observed is not zero activity."
}
