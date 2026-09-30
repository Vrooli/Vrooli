package autofiler

import (
	"context"
	"fmt"
	"time"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/backlogstatus"
	"swarm-manager/internal/eventlog"
	"swarm-manager/internal/settings"
)

type BacklogReader interface {
	LoadAll(kinds []backlog.BacklogKind) ([]backlog.BacklogItem, error)
}

type TransitionCounter interface {
	CountStatusTransitionsInRange(ctx context.Context, eventType eventlog.EventType, toStatus string, from, to time.Time) (int, error)
}

type BrakeState struct {
	WindowDays     int
	Minimum        int
	Observed       int
	Braked         bool
	WindowStartUTC time.Time
	WindowEndUTC   time.Time
}

func CountOpenAutoFiled(reader BacklogReader) (int, error) {
	items, err := reader.LoadAll(nil)
	if err != nil {
		return 0, err
	}
	return len(OpenAutoFiled(items)), nil
}

func RemainingBudget(reader BacklogReader, maxOpen int) (int, error) {
	if maxOpen <= 0 {
		return 0, nil
	}
	open, err := CountOpenAutoFiled(reader)
	if err != nil {
		return 0, err
	}
	if open >= maxOpen {
		return 0, nil
	}
	return maxOpen - open, nil
}

func VelocityBrake(ctx context.Context, counter TransitionCounter, cfg settings.AutoFilerSettings, now time.Time) (BrakeState, error) {
	windowDays := cfg.VelocityWindowDays
	if windowDays <= 0 {
		windowDays = settings.DefaultSettings().AutoFiler.VelocityWindowDays
	}
	minimum := cfg.MinVelocityTransitions
	if minimum < 0 {
		minimum = 0
	}
	end := now.UTC()
	start := end.Add(-time.Duration(windowDays) * 24 * time.Hour)
	state := BrakeState{
		WindowDays:     windowDays,
		Minimum:        minimum,
		WindowStartUTC: start,
		WindowEndUTC:   end,
	}
	if minimum == 0 {
		return state, nil
	}
	if counter == nil {
		return state, fmt.Errorf("transition counter is required")
	}
	// Every status a suggestion can advance into. Derived from the SSOT so a
	// newly added status is counted rather than silently missing from the
	// adoption metrics; `suggested` and `backlog` are excluded because they are
	// the origin states, not forward motion.
	var forwardStatuses []backlog.BacklogStatus
	for _, s := range backlogstatus.All() {
		if s == backlogstatus.Suggested || s == backlogstatus.Backlog {
			continue
		}
		forwardStatuses = append(forwardStatuses, backlog.BacklogStatus(s))
	}
	for _, status := range forwardStatuses {
		count, err := counter.CountStatusTransitionsInRange(ctx, eventlog.EventBacklogStatusChanged, string(status), start, end)
		if err != nil {
			return state, err
		}
		state.Observed += count
	}
	state.Braked = state.Observed < minimum
	return state, nil
}
