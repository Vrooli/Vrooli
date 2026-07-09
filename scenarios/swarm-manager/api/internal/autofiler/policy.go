package autofiler

import (
	"context"
	"fmt"
	"time"

	"swarm-manager/internal/backlog"
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
	forwardStatuses := []backlog.BacklogStatus{
		backlog.StatusReady,
		backlog.StatusQueued,
		backlog.StatusInProgress,
		backlog.StatusInReview,
		backlog.StatusReviewPending,
		backlog.StatusCompleted,
		backlog.StatusFailed,
		backlog.StatusNeedsFollowup,
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
