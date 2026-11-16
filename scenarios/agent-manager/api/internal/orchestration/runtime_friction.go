package orchestration

import (
	"context"
	"fmt"
	"strings"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/domain"
	"agent-manager/internal/eventlog"
	"agent-manager/internal/invocationreadmodel"
	"agent-manager/internal/repository"
	"agent-manager/internal/runsignal"
	"agent-manager/internal/toolfriction"

	"github.com/google/uuid"
)

const runtimeFrictionRunLimit = 5000

// RuntimeFrictionReport returns a bounded, read-only view of run duration and
// friction evidence. Imported transcript runs use the same repository and
// event projection as native runs and are included in the report.
func (o *Orchestrator) RuntimeFrictionReport(ctx context.Context, since, until time.Time) (toolfriction.RuntimeReport, error) {
	if o.runs == nil {
		return toolfriction.RuntimeReport{}, fmt.Errorf("run repository is not configured")
	}
	if since.IsZero() || until.IsZero() || !until.After(since) || until.Sub(since) > 30*24*time.Hour {
		return toolfriction.RuntimeReport{}, fmt.Errorf("runtime friction window must be positive and no longer than 30 days")
	}
	runs, err := o.runs.List(ctx, repository.RunListFilter{
		ListFilter: repository.ListFilter{Limit: runtimeFrictionRunLimit + 1},
	})
	if err != nil {
		return toolfriction.RuntimeReport{}, fmt.Errorf("list runs for runtime friction: %w", err)
	}
	truncated := len(runs) > runtimeFrictionRunLimit
	if truncated {
		runs = runs[:runtimeFrictionRunLimit]
	}

	inputs := make([]toolfriction.RuntimeRun, 0, len(runs))
	failedEventReads := 0
	for _, run := range runs {
		if run == nil || !runInRuntimeWindow(run, since, until) {
			continue
		}
		events, eventErr := o.allRunEvents(ctx, run.ID, event.GetOptions{AfterSequence: -1})
		if eventErr != nil {
			failedEventReads++
			events = nil
		}
		accounting, hasAccounting := o.runtimeTimeAccounting(ctx, run.ID, events, run.StartedAt, run.EndedAt)
		inputs = append(inputs, runtimeRunInput(run, events, accounting, hasAccounting))
	}
	window := toolfriction.RuntimeWindow{Since: since, Until: until}
	report := toolfriction.AnalyzeRuntime(inputs, window)
	if failedEventReads > 0 || truncated {
		report.Coverage.State = "partial"
		reasons := make([]string, 0, 2)
		if failedEventReads > 0 {
			reasons = append(reasons, fmt.Sprintf("events unavailable for %d runs", failedEventReads))
		}
		if truncated {
			reasons = append(reasons, fmt.Sprintf("run listing was capped at %d", runtimeFrictionRunLimit))
		}
		report.Coverage.Reason = strings.Join(reasons, "; ")
	}
	return report, nil
}

func (o *Orchestrator) runtimeTimeAccounting(ctx context.Context, runID uuid.UUID, events []*domain.RunEvent, startedAt, endedAt *time.Time) (runsignal.TimeAccounting, bool) {
	if startedAt == nil || endedAt == nil || !endedAt.After(*startedAt) {
		return runsignal.TimeAccounting{}, false
	}
	if projection, ok := o.invocationReadModel.(invocationreadmodel.ProjectionStore); ok {
		if accounting, present, err := projection.TimeAccountingForRun(ctx, runID.String()); err == nil && present && accounting.DurationMS() == endedAt.Sub(*startedAt).Milliseconds() {
			return accounting, true
		}
	}
	if len(events) == 0 {
		return runsignal.TimeAccounting{}, false
	}
	return runsignal.DeriveTimeAccounting(events, startedAt, endedAt), true
}

func runInRuntimeWindow(run *domain.Run, since, until time.Time) bool {
	anchor := run.EndedAt
	if anchor == nil {
		anchor = run.StartedAt
	}
	if anchor == nil {
		anchor = &run.CreatedAt
	}
	return !anchor.Before(since) && anchor.Before(until)
}

func runtimeRunInput(run *domain.Run, events []*domain.RunEvent, accounting runsignal.TimeAccounting, hasAccounting bool) toolfriction.RuntimeRun {
	out := toolfriction.RuntimeRun{
		ID:            run.ID.String(),
		TaskID:        run.TaskID.String(),
		Status:        string(run.Status),
		TerminalClass: string(run.TerminalClass),
		StopReason:    string(run.StopReason),
		Imported:      run.ExecutionMode.Normalized() == domain.ExecutionModeImported,
		StartedAt:     run.StartedAt,
		EndedAt:       run.EndedAt,
		Time:          accounting,
		HasTime:       hasAccounting,
	}
	if len(events) == 0 {
		return out
	}
	byID := make(map[string]*domain.RunEvent, len(events))
	for _, event := range events {
		if event != nil {
			byID[event.ID.String()] = event
		}
	}
	for _, fact := range runsignal.DeriveInvocationFacts(events) {
		source := byID[fact.CallEventID]
		if source == nil {
			continue
		}
		signal := toolfriction.RuntimeSignal{
			At:            source.Timestamp,
			Kind:          "invocation",
			Check:         fact.CommandPath,
			CommandPath:   fact.CommandPath,
			Outcome:       fact.Outcome,
			SemanticsKind: fact.SemanticsKind,
			Retry:         fact.RetryOfCallEventID != "",
		}
		if fact.SemanticsKind == "verify" {
			signal.CheckID = fact.CommandPath
		}
		if fact.DurationMS != nil {
			signal.DurationMS = *fact.DurationMS
			signal.DurationKnown = true
		}
		if strings.HasPrefix(fact.CommandPath, "agent-manager ") || fact.Executable == "agent-manager" {
			signal.Owner = "agent-manager"
		} else if strings.HasPrefix(fact.CommandPath, "workspace-sandbox ") || fact.Executable == "workspace-sandbox" {
			signal.Owner = "workspace-sandbox"
		}
		out.Events = append(out.Events, signal)
	}
	for _, source := range events {
		if source == nil {
			continue
		}
		switch payload := source.Data.(type) {
		case *domain.MessageEventData:
			if payload.Role == "assistant" && !payload.EvidenceOnly {
				text := payload.Content
				if len(text) > 512 {
					text = text[:512]
				}
				out.Events = append(out.Events, toolfriction.RuntimeSignal{At: source.Timestamp, Kind: "message", Text: text})
			}
		default:
			raw, ok := source.Data.(*domain.TypedEventData)
			if !ok {
				continue
			}
			decoded, decodeErr := eventlog.Decode(source.EventType, source.SchemaVersion, raw.Body)
			if decodeErr != nil {
				continue
			}
			switch value := decoded.(type) {
			case *eventlog.SandboxOperationPayload:
				if !value.Success {
					signal := toolfriction.RuntimeSignal{At: source.Timestamp, Kind: toolfriction.RuntimeSandboxEnvironment}
					if value.DurationMS > 0 && run.StartedAt != nil && source.Timestamp.After(*run.StartedAt) && (run.EndedAt == nil || !source.Timestamp.After(*run.EndedAt)) {
						signal.DurationMS = value.DurationMS
						signal.DurationKnown = true
						signal.DurationBasis = "unattributed"
					}
					out.Events = append(out.Events, signal)
				}
			case *eventlog.CheckpointFailurePayload, *eventlog.HeartbeatMissPayload, *eventlog.RunnerFallbackExhaustedPayload, *eventlog.ModelFallbackExhaustedPayload:
				out.Events = append(out.Events, toolfriction.RuntimeSignal{At: source.Timestamp, Kind: toolfriction.RuntimeAgentManagerFailure})
			case *eventlog.RetryAttemptPayload:
				out.Events = append(out.Events, toolfriction.RuntimeSignal{At: source.Timestamp, Kind: toolfriction.RuntimeRework})
			}
		}
	}
	return out
}
