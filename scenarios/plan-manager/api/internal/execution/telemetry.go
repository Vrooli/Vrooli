package execution

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"
)

// TelemetryKind is intentionally closed: reports must distinguish lifecycle
// facts instead of inferring them from log text.
type TelemetryKind string

const (
	TelemetryRequest      TelemetryKind = "request"
	TelemetryAdmission    TelemetryKind = "admission"
	TelemetryExecution    TelemetryKind = "execution"
	TelemetryIntervention TelemetryKind = "intervention"
	TelemetryRetry        TelemetryKind = "retry"
	TelemetryRecovery     TelemetryKind = "recovery"
	TelemetryCompletion   TelemetryKind = "completion"
	TelemetryAbandonment  TelemetryKind = "abandonment"
	TelemetryReuse        TelemetryKind = "reuse"
)

type ExecutionTelemetryEvent struct {
	ID              string
	Kind            TelemetryKind
	OccurredAt      time.Time
	TaskID          string
	PlanID          string
	FamilyID        string
	ChildID         string
	ValidationID    string
	AttemptID       string
	ParentEventID   string
	State           string
	Reason          string
	PolicyIdentity  string
	ContentIdentity string
	Duration        time.Duration
}

func telemetryTime(value string) time.Time {
	parsed, _ := time.Parse(time.RFC3339Nano, value)
	return parsed
}

func (e ExecutionTelemetryEvent) Validate() error {
	if strings.TrimSpace(e.ID) == "" || e.OccurredAt.IsZero() || strings.TrimSpace(e.TaskID) == "" || strings.TrimSpace(e.PlanID) == "" {
		return errors.New("telemetry event requires id, occurred_at, task_id, and plan_id")
	}
	if e.Kind == "" {
		return errors.New("telemetry event kind is required")
	}
	return nil
}

type TelemetrySink interface {
	Append(ExecutionTelemetryEvent) error
}

type noopTelemetrySink struct{}

func (noopTelemetrySink) Append(ExecutionTelemetryEvent) error { return nil }

type repositoryTelemetrySink struct{ repo Repository }

// NewRepositoryTelemetrySink adapts the durable execution repository to the
// owner telemetry append seam used by the validation and execution domains.
func NewRepositoryTelemetrySink(repo Repository) TelemetrySink {
	return repositoryTelemetrySink{repo: repo}
}

func (s repositoryTelemetrySink) Append(event ExecutionTelemetryEvent) error {
	return s.repo.SaveTelemetry(context.Background(), event)
}

type MemoryTelemetrySink struct {
	mu     sync.Mutex
	events []ExecutionTelemetryEvent
}

func (s *MemoryTelemetrySink) Append(event ExecutionTelemetryEvent) error {
	if err := event.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, event)
	return nil
}

func (s *MemoryTelemetrySink) Events() []ExecutionTelemetryEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := append([]ExecutionTelemetryEvent(nil), s.events...)
	sort.SliceStable(result, func(i, j int) bool { return result[i].OccurredAt.Before(result[j].OccurredAt) })
	return result
}

type ValidationTelemetry struct {
	Requested int
	Executed  int
	Reused    int
	Unknown   int
	Duration  time.Duration
}

type CoordinationTelemetry struct {
	Productive time.Duration
	Waiting    time.Duration
	Unknown    time.Duration
}

type FamilyCriticalPathTelemetry struct {
	Families int
	Known    int
	Unknown  int
	Duration time.Duration
}

type TelemetryReport struct {
	Validation     ValidationTelemetry
	Coordination   CoordinationTelemetry
	ManualRecovery int
	CriticalPath   FamilyCriticalPathTelemetry
}

// BuildTelemetryReport derives only from typed events. Missing lifecycle
// boundaries are counted as unknown; no duration is fabricated from wall-clock
// report time or from adjacent unrelated events.
func BuildTelemetryReport(events []ExecutionTelemetryEvent) TelemetryReport {
	var report TelemetryReport
	byValidation := map[string]ExecutionTelemetryEvent{}
	byTask := map[string][]ExecutionTelemetryEvent{}
	byFamily := map[string][]ExecutionTelemetryEvent{}
	for _, event := range events {
		byTask[event.TaskID] = append(byTask[event.TaskID], event)
		if event.FamilyID != "" {
			byFamily[event.FamilyID] = append(byFamily[event.FamilyID], event)
		}
		switch event.Kind {
		case TelemetryRequest:
			report.Validation.Requested++
		case TelemetryExecution:
			if event.ValidationID != "" {
				byValidation[event.ValidationID] = event
			}
		case TelemetryReuse:
			if event.ValidationID != "" {
				byValidation[event.ValidationID] = event
			}
		case TelemetryRecovery:
			report.ManualRecovery++
		}
	}
	for _, event := range byValidation {
		if event.Kind == TelemetryReuse {
			report.Validation.Reused++
		} else {
			report.Validation.Executed++
		}
		if event.Duration <= 0 {
			report.Validation.Unknown++
		} else {
			report.Validation.Duration += event.Duration
		}
	}
	for _, taskEvents := range byTask {
		sort.Slice(taskEvents, func(i, j int) bool { return taskEvents[i].OccurredAt.Before(taskEvents[j].OccurredAt) })
		for i := 1; i < len(taskEvents); i++ {
			interval := taskEvents[i].OccurredAt.Sub(taskEvents[i-1].OccurredAt)
			if interval <= 0 {
				continue
			}
			switch taskEvents[i-1].Kind {
			case TelemetryExecution, TelemetryCompletion:
				report.Coordination.Productive += interval
			case TelemetryAdmission, TelemetryIntervention, TelemetryRecovery, TelemetryRetry:
				report.Coordination.Waiting += interval
			default:
				report.Coordination.Unknown += interval
			}
		}
	}
	for _, familyEvents := range byFamily {
		report.CriticalPath.Families++
		var first, last time.Time
		for _, event := range familyEvents {
			if first.IsZero() || event.OccurredAt.Before(first) {
				first = event.OccurredAt
			}
			if event.OccurredAt.After(last) {
				last = event.OccurredAt
			}
		}
		if first.IsZero() || last.IsZero() || !last.After(first) {
			report.CriticalPath.Unknown++
			continue
		}
		report.CriticalPath.Known++
		report.CriticalPath.Duration += last.Sub(first)
	}
	return report
}
