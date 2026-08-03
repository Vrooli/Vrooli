// Package runreport builds the bounded, diagnostic projection consumed by run
// inspection, investigations, and human-facing detail views.
package runreport

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"agent-manager/internal/adapters/sandbox"
	sharedavailability "agent-manager/internal/availability"
	"agent-manager/internal/domain"
	"agent-manager/internal/runsignal"

	"github.com/google/uuid"
)

// Source is the narrow read seam required to build a report. It deliberately
// exposes projections only; no report builder can mutate a run or launch work.
type Source interface {
	Run(context.Context, uuid.UUID) (*domain.Run, error)
	Events(context.Context, uuid.UUID) ([]*domain.RunEvent, error)
	Diff(context.Context, uuid.UUID) (*sandbox.DiffResult, error)
}

// DurableInvocationFactSource supplies a completed durable projection. The
// boolean distinguishes a valid empty projection from an absent projection.
type DurableInvocationFactSource interface {
	DurableInvocationFacts(context.Context, uuid.UUID) ([]runsignal.InvocationFact, bool, error)
}

// DurableEpisodeSource and DurableSelfReportSpanSource mirror the invocation
// fact seam. They distinguish an intentionally empty projected collection
// from a run that predates the durable projection.
type DurableEpisodeSource interface {
	DurableEpisodes(context.Context, uuid.UUID) ([]runsignal.FrictionEpisode, bool, error)
}

type DurableSelfReportSpanSource interface {
	DurableSelfReportSpans(context.Context, uuid.UUID) ([]runsignal.SelfReportSpan, bool, error)
}

type DurableTimeAccountingSource interface {
	DurableTimeAccounting(context.Context, uuid.UUID) (runsignal.TimeAccounting, bool, error)
}

// ReceiptSummary is deliberately bounded: the report needs availability and
// count to route an investigation, while the receipts command owns payload
// disclosure.
type ReceiptSummary struct {
	Availability
	Count    int
	EventIDs []string
	Calls    []CrossScenarioCall
}

type CrossScenarioCall struct {
	OccurredAt          time.Time      `json:"occurredAt,omitempty"`
	TargetScenario      string         `json:"targetScenario"`
	Operation           string         `json:"operation"`
	Outcome             string         `json:"outcome"`
	StatusCode          uint32         `json:"statusCode"`
	DurationMS          uint64         `json:"durationMs"`
	ReceiptEventID      string         `json:"receiptEventId"`
	Verified            bool           `json:"verified"`
	Projection          map[string]any `json:"projection,omitempty"`
	ProjectionDropCount int            `json:"projectionDropCount,omitempty"`
	PolicyVersion       string         `json:"policyVersion,omitempty"`
}

// LedgerTargetRollup summarizes one target without hiding individual receipt
// evidence from callers that need to inspect it.
type LedgerTargetRollup struct {
	TargetScenario   string `json:"targetScenario"`
	Calls            int    `json:"calls"`
	Failures         int    `json:"failures"`
	TotalDurationMS  uint64 `json:"totalDurationMs"`
	MedianDurationMS uint64 `json:"medianDurationMs"`
}

func BoundProjection(projection map[string]any) (map[string]any, int) {
	if len(projection) == 0 {
		return nil, 0
	}
	out := map[string]any{}
	dropped := 0
	for key, value := range projection {
		if len(out) >= 16 {
			dropped++
			continue
		}
		candidate := make(map[string]any, len(out)+1)
		for k, v := range out {
			candidate[k] = v
		}
		candidate[key] = value
		body, _ := json.Marshal(candidate)
		if len(body) > 2048 {
			dropped++
			continue
		}
		out[key] = value
	}
	return out, dropped
}

// ReceiptSource is optional so fixture-only report sources stay lightweight.
// Production composition supplies it through orchestration.
type ReceiptSource interface {
	Receipts(context.Context, uuid.UUID) (ReceiptSummary, error)
}

// Availability is the shared reporting vocabulary. It remains an alias so
// existing report consumers keep the same package-level API.
type (
	Availability      = sharedavailability.Availability
	AvailabilityState = sharedavailability.State
)

const (
	AvailabilityAvailable    = sharedavailability.Available
	AvailabilityUnavailable  = sharedavailability.Unavailable
	AvailabilityDegraded     = sharedavailability.Degraded
	AvailabilityUnobserved   = sharedavailability.Unobserved
	AvailabilityUnknown      = sharedavailability.Unknown
	AvailabilityResolved     = sharedavailability.Resolved
	AvailabilityPolicyAbsent = sharedavailability.PolicyAbsent
	AvailabilityOversized    = sharedavailability.Oversized
	AvailabilityNotCaptured  = sharedavailability.NotCaptured
	AvailabilityExternal     = sharedavailability.External
	AvailabilityEmpty        = sharedavailability.Empty
	AvailabilityComplete     = sharedavailability.Complete
	AvailabilityUnreliable   = sharedavailability.Unreliable
)

type ToolSummary struct {
	Name       string `json:"name"`
	Calls      int    `json:"calls"`
	Successes  int    `json:"successes"`
	Failures   int    `json:"failures"`
	Unresolved int    `json:"unresolved"`
}
type ResultSummary struct {
	SelectionStatus  domain.FinalOutputSelectionStatus `json:"selectionStatus"`
	SelectionRule    string                            `json:"selectionRule,omitempty"`
	CandidateCount   int                               `json:"candidateCount"`
	StructuredStatus domain.StructuredResultStatus     `json:"structuredStatus,omitempty"`
	StructuredMethod string                            `json:"structuredMethod,omitempty"`
	DiagnosticCodes  []string                          `json:"diagnosticCodes,omitempty"`
}

type GoalProgress struct {
	Availability            Availability  `json:"availability"`
	GoalID                  string        `json:"goalId,omitempty"`
	FirstMetEventIndex      int           `json:"firstMetEventIndex,omitempty"`
	FirstMetTurnIndex       int           `json:"firstMetTurnIndex,omitempty"`
	TimeToFirstMet          time.Duration `json:"timeToFirstMet,omitempty"`
	TokenCostBeforeFirstMet int64         `json:"tokenCostBeforeFirstMet,omitempty"`
	InterventionsBeforeMet  int           `json:"interventionsBeforeMet,omitempty"`
	HandoffsBeforeMet       int           `json:"handoffsBeforeMet,omitempty"`
	Regression              bool          `json:"regression"`
	Restatement             bool          `json:"restatement"`
}

type EvidenceValidity struct {
	Availability      Availability `json:"availability"`
	TotalCalls        int          `json:"totalCalls"`
	ClassifiedBase    int          `json:"classifiedBase"`
	UnclassifiedCount int          `json:"unclassifiedCount"`
	UnclassifiedShare float64      `json:"unclassifiedShare"`
}
type DiffSummary struct {
	Files     int          `json:"files"`
	Bytes     int64        `json:"bytes"`
	Available Availability `json:"available"`
}
type RunReport struct {
	RunID                  uuid.UUID                   `json:"runId"`
	Status                 domain.RunStatus            `json:"status"`
	ExitCode               *int                        `json:"exitCode,omitempty"`
	Error                  string                      `json:"error,omitempty"`
	Duration               time.Duration               `json:"duration,omitempty"`
	HeartbeatGap           time.Duration               `json:"heartbeatGap,omitempty"`
	Turns                  int                         `json:"turns"`
	Tokens                 int                         `json:"tokens"`
	CostUSD                float64                     `json:"costUsd"`
	Result                 ResultSummary               `json:"result"`
	Events                 map[string]int              `json:"events"`
	Tools                  []ToolSummary               `json:"tools"`
	ProjectOwnedToolCalls  int                         `json:"projectOwnedToolCalls"`
	ExternalToolCalls      int                         `json:"externalToolCalls"`
	RequestedModel         string                      `json:"requestedModel,omitempty"`
	ActualModel            string                      `json:"actualModel,omitempty"`
	FallbackCount          int                         `json:"fallbackCount"`
	Diff                   DiffSummary                 `json:"diff"`
	EventsAvailability     Availability                `json:"eventsAvailability"`
	ReceiptsAvailability   Availability                `json:"receiptsAvailability"`
	LedgerAvailability     Availability                `json:"ledgerAvailability"`
	ProjectionAvailability Availability                `json:"projectionAvailability"`
	ReceiptCount           int                         `json:"receiptCount"`
	ReceiptEvidenceIDs     []string                    `json:"receiptEvidenceIds,omitempty"`
	CrossScenarioCalls     []CrossScenarioCall         `json:"crossScenarioCalls,omitempty"`
	LedgerTargetRollups    []LedgerTargetRollup        `json:"ledgerTargetRollups,omitempty"`
	RepeatedToolCalls      int                         `json:"repeatedToolCalls"`
	FilesReadMoreThanOnce  int                         `json:"filesReadMoreThanOnce"`
	LongestEventGap        time.Duration               `json:"longestEventGap"`
	InvocationFacts        []runsignal.InvocationFact  `json:"-"`
	Episodes               []runsignal.FrictionEpisode `json:"episodes,omitempty"`
	SelfReportSpans        []runsignal.SelfReportSpan  `json:"selfReportSpans,omitempty"`
	HelpRecoveries         int                         `json:"helpRecoveries"`
	UnknownInvocations     int                         `json:"unknownInvocations"`
	TimeAccounting         runsignal.TimeAccounting    `json:"timeAccounting"`
	Goal                   GoalProgress                `json:"goal"`
	InvocationValidity     EvidenceValidity            `json:"invocationValidity"`
}

func Build(ctx context.Context, source Source, runID uuid.UUID) (*RunReport, error) {
	if source == nil {
		return nil, fmt.Errorf("run report source is unavailable")
	}
	run, err := source.Run(ctx, runID)
	if err != nil {
		return nil, err
	}
	r := &RunReport{RunID: run.ID, Status: run.Status, ExitCode: run.ExitCode, Error: run.ErrorMsg, Events: map[string]int{}, EventsAvailability: Availability{State: AvailabilityAvailable}, ReceiptsAvailability: Availability{State: AvailabilityUnavailable, Reason: "receipt reader is not configured"}, LedgerAvailability: Availability{State: AvailabilityUnavailable, Reason: "receipt reader is not configured"}, ProjectionAvailability: Availability{State: AvailabilityUnavailable, Reason: "receipt reader is not configured"}, RequestedModel: run.RequestedModel, ActualModel: run.ActualModel, Diff: DiffSummary{Files: run.ChangedFiles, Bytes: run.TotalSizeBytes, Available: Availability{State: AvailabilityUnavailable}}}
	r.Goal = deriveGoalProgress(run, nil, nil)
	r.InvocationValidity = invocationValidity(nil)
	ownershipFromProjection := false
	imported := run.ExecutionMode.Normalized() == domain.ExecutionModeImported
	if imported {
		r.Diff.Available = Availability{State: AvailabilityUnavailable, Reason: "imported run has no sandbox"}
		r.ReceiptsAvailability = Availability{State: AvailabilityUnavailable, Reason: "imported run predates run identity correlation"}
		r.LedgerAvailability = Availability{State: AvailabilityUnavailable, Reason: "imported run predates run identity correlation"}
	}
	if run.StartedAt != nil && run.EndedAt != nil {
		r.Duration = run.EndedAt.Sub(*run.StartedAt)
	}
	if run.LastHeartbeat != nil && run.EndedAt != nil {
		r.HeartbeatGap = run.EndedAt.Sub(*run.LastHeartbeat)
	}
	if run.Summary != nil {
		r.Turns, r.Tokens, r.CostUSD = run.Summary.TurnsUsed, run.Summary.TokensUsed, run.Summary.CostEstimate
	}
	if run.Result != nil {
		r.Result.SelectionStatus, r.Result.SelectionRule, r.Result.CandidateCount = run.Result.Selection.Status, run.Result.Selection.Rule, len(run.Result.Candidates)
		if run.Result.Structured != nil {
			r.Result.StructuredStatus, r.Result.StructuredMethod = run.Result.Structured.Status, run.Result.Structured.Method
			for _, d := range run.Result.Structured.Diagnostics {
				r.Result.DiagnosticCodes = append(r.Result.DiagnosticCodes, d.Code)
			}
		}
	}
	events, err := source.Events(ctx, runID)
	if err != nil {
		r.EventsAvailability = Availability{State: AvailabilityUnavailable, Reason: err.Error()}
	} else {
		r.foldEvents(events)
		r.TimeAccounting = runsignal.DeriveTimeAccounting(events, run.StartedAt, run.EndedAt)
		if durable, ok := source.(DurableTimeAccountingSource); ok {
			accounting, present, durableErr := durable.DurableTimeAccounting(ctx, runID)
			if durableErr != nil {
				return nil, fmt.Errorf("read durable time accounting: %w", durableErr)
			}
			if present {
				r.TimeAccounting = accounting
			}
		}
		if durable, ok := source.(DurableInvocationFactSource); ok {
			facts, present, durableErr := durable.DurableInvocationFacts(ctx, runID)
			if durableErr != nil {
				return nil, fmt.Errorf("read durable invocation facts: %w", durableErr)
			}
			if present {
				r.InvocationFacts = facts
				ownershipFromProjection = true
			} else {
				r.InvocationFacts = runsignal.DeriveInvocationFacts(events)
			}
		} else {
			r.InvocationFacts = runsignal.DeriveInvocationFacts(events)
		}
		for _, fact := range r.InvocationFacts {
			if fact.HelpRecovery {
				r.HelpRecoveries++
			}
			if fact.Ownership == "unknown" || fact.Availability == "unknown" {
				r.UnknownInvocations++
			}
		}
		if ownershipFromProjection {
			// The durable invocation projection is the runner-aware ownership
			// authority. Reconcile the human-facing rollup with it so imported
			// Codex wrapper calls do not fall back to tool-name heuristics.
			r.ProjectOwnedToolCalls, r.ExternalToolCalls = 0, 0
			for _, fact := range r.InvocationFacts {
				switch fact.Ownership {
				case runsignal.OwnershipResolved, "project":
					r.ProjectOwnedToolCalls++
				case runsignal.OwnershipExternal:
					r.ExternalToolCalls++
				}
			}
		}
		r.InvocationValidity = invocationValidity(r.InvocationFacts)
		if durable, ok := source.(DurableEpisodeSource); ok {
			episodes, present, durableErr := durable.DurableEpisodes(ctx, runID)
			if durableErr != nil {
				return nil, fmt.Errorf("read durable episodes: %w", durableErr)
			}
			if present {
				r.Episodes = episodes
			} else {
				r.Episodes = runsignal.DeriveEpisodes(r.InvocationFacts, events)
			}
		} else {
			r.Episodes = runsignal.DeriveEpisodes(r.InvocationFacts, events)
		}
		if imported {
			for index := range r.Episodes {
				r.Episodes[index].OwnerConfidence = "manifest-derived-historical"
			}
		}
		if durable, ok := source.(DurableSelfReportSpanSource); ok {
			spans, present, durableErr := durable.DurableSelfReportSpans(ctx, runID)
			if durableErr != nil {
				return nil, fmt.Errorf("read durable self-report spans: %w", durableErr)
			}
			if present {
				r.SelfReportSpans = spans
			} else {
				r.SelfReportSpans = runsignal.DeriveSelfReportSpans(events)
			}
		} else {
			r.SelfReportSpans = runsignal.DeriveSelfReportSpans(events)
		}
		raiseEpisodeSeverityForRepeatedRules(r.Episodes, r.SelfReportSpans, events)
		r.Goal = deriveGoalProgress(run, events, r.Episodes)
	}
	if !imported {
		if diff, err := source.Diff(ctx, runID); err == nil && diff != nil {
			r.Diff = DiffSummary{Files: len(diff.Files), Bytes: run.TotalSizeBytes, Available: Availability{State: AvailabilityAvailable}}
		} else if err != nil {
			r.Diff.Available.Reason = err.Error()
		}
	}
	if !imported {
		if receipts, ok := source.(ReceiptSource); ok {
			summary, err := receipts.Receipts(ctx, runID)
			if err != nil {
				r.ReceiptsAvailability = Availability{State: AvailabilityDegraded, Reason: err.Error()}
			} else {
				r.ReceiptsAvailability = Availability{State: summary.State, Reason: summary.Reason}
				r.LedgerAvailability = Availability{State: summary.State, Reason: summary.Reason}
				r.ReceiptCount = summary.Count
				r.ReceiptEvidenceIDs = append([]string(nil), summary.EventIDs...)
				r.CrossScenarioCalls = append([]CrossScenarioCall(nil), summary.Calls...)
				r.LedgerTargetRollups = ledgerTargetRollups(r.CrossScenarioCalls)
				r.ProjectionAvailability = projectionAvailability(r.CrossScenarioCalls)
				r.Episodes = UpgradeEpisodeOwnership(r.Episodes, events, r.CrossScenarioCalls, r.LedgerAvailability)
				if store, ok := source.(LedgerStore); ok {
					if err := store.ReplaceCrossScenarioCalls(ctx, runID, string(r.LedgerAvailability.State), r.CrossScenarioCalls); err != nil {
						return nil, fmt.Errorf("persist cross-scenario ledger: %w", err)
					}
				}
			}
			if store, ok := source.(ReceiptJoinStore); ok {
				if err := store.ReplaceReceiptEvidence(ctx, runID, string(r.ReceiptsAvailability.State), r.ReceiptEvidenceIDs); err != nil {
					return nil, fmt.Errorf("persist receipt evidence: %w", err)
				}
			}
		}
	}
	return r, nil
}

func invocationValidity(facts []runsignal.InvocationFact) EvidenceValidity {
	validity := EvidenceValidity{Availability: Availability{State: AvailabilityAvailable}, TotalCalls: len(facts)}
	for _, fact := range facts {
		if fact.Ownership == "resolved" || fact.Ownership == "project" || fact.Ownership == "external" {
			validity.ClassifiedBase++
		} else {
			validity.UnclassifiedCount++
		}
	}
	if validity.TotalCalls == 0 {
		validity.Availability = Availability{State: AvailabilityUnavailable, Reason: "no invocation facts are available"}
		return validity
	}
	validity.UnclassifiedShare = float64(validity.UnclassifiedCount) / float64(validity.TotalCalls)
	if float64(validity.ClassifiedBase)/float64(validity.TotalCalls) < 0.90 {
		validity.Availability = Availability{State: AvailabilityUnreliable, Reason: "classified invocation share is below the minimum 90%"}
	}
	return validity
}

func deriveGoalProgress(run *domain.Run, events []*domain.RunEvent, episodes []runsignal.FrictionEpisode) GoalProgress {
	if run == nil || strings.TrimSpace(run.GoalID) == "" {
		return GoalProgress{Availability: Availability{State: AvailabilityUnavailable, Reason: "run carries no declared goal"}}
	}
	progress := GoalProgress{Availability: Availability{State: AvailabilityAvailable}, GoalID: run.GoalID, FirstMetEventIndex: -1, FirstMetTurnIndex: -1}
	var firstMetAt time.Time
	var markerTexts []string
	metSeen := false
	for index, event := range events {
		if event == nil {
			continue
		}
		message, ok := event.Data.(*domain.MessageEventData)
		if !ok {
			if usage, usageOK := event.Data.(*domain.UsageEventData); usageOK && !firstMetAt.IsZero() && !event.Timestamp.After(firstMetAt) {
				progress.TokenCostBeforeFirstMet += int64(usage.InputTokens + usage.OutputTokens + usage.CacheCreationTokens + usage.CacheReadTokens)
			}
			continue
		}
		text := strings.TrimSpace(message.Content)
		lower := strings.ToLower(text)
		isMet := strings.Contains(lower, "goal met") || strings.Contains(lower, "goal achieved") || strings.Contains(lower, "goal: met")
		isUnmet := strings.Contains(lower, "goal unmet") || strings.Contains(lower, "goal: unmet")
		if strings.Contains(lower, "goal") {
			markerTexts = append(markerTexts, lower)
		}
		if isMet && firstMetAt.IsZero() {
			firstMetAt = event.Timestamp
			progress.FirstMetEventIndex = index
			metSeen = true
			if run.StartedAt != nil && !event.Timestamp.Before(*run.StartedAt) {
				progress.TimeToFirstMet = event.Timestamp.Sub(*run.StartedAt)
			}
		}
		if isUnmet && metSeen {
			progress.Regression = true
		}
		if strings.EqualFold(message.Role, "assistant") && !firstMetAt.IsZero() && !event.Timestamp.After(firstMetAt) {
			progress.FirstMetTurnIndex++
		}
	}
	if firstMetAt.IsZero() && strings.EqualFold(run.GoalStatus, "met") {
		progress.FirstMetEventIndex = len(events) - 1
		progress.FirstMetTurnIndex = 0
		if run.EndedAt != nil && run.StartedAt != nil && !run.EndedAt.Before(*run.StartedAt) {
			progress.TimeToFirstMet = run.EndedAt.Sub(*run.StartedAt)
		}
		metSeen = true
	}
	if !firstMetAt.IsZero() {
		for _, event := range events {
			if event == nil || event.Timestamp.After(firstMetAt) {
				continue
			}
			if usage, ok := event.Data.(*domain.UsageEventData); ok {
				progress.TokenCostBeforeFirstMet += int64(usage.InputTokens + usage.OutputTokens + usage.CacheCreationTokens + usage.CacheReadTokens)
			}
		}
	}
	if !metSeen {
		progress.Availability = Availability{State: AvailabilityDegraded, Reason: "goal is declared but no first-met marker is available"}
	}
	for i, event := range events {
		if event == nil || (!firstMetAt.IsZero() && event.Timestamp.After(firstMetAt)) {
			continue
		}
		message, ok := event.Data.(*domain.MessageEventData)
		if !ok || !strings.EqualFold(message.Role, "user") {
			continue
		}
		for _, later := range events[i+1:] {
			if later == nil || (!firstMetAt.IsZero() && later.Timestamp.After(firstMetAt)) {
				continue
			}
			if _, assistant := later.Data.(*domain.MessageEventData); assistant {
				progress.InterventionsBeforeMet++
				break
			}
			if _, tool := later.Data.(*domain.ToolCallEventData); tool {
				progress.InterventionsBeforeMet++
				break
			}
		}
	}
	for _, episode := range episodes {
		if episode.Pattern != "handoff-continuation" || firstMetAt.IsZero() {
			continue
		}
		for _, event := range events {
			if event != nil && event.ID.String() == episode.StartEventID && !event.Timestamp.After(firstMetAt) {
				progress.HandoffsBeforeMet++
				break
			}
		}
	}
	for i := 1; i < len(markerTexts); i++ {
		if markerTexts[i] != markerTexts[i-1] {
			progress.Restatement = true
			break
		}
	}
	return progress
}

func ledgerTargetRollups(calls []CrossScenarioCall) []LedgerTargetRollup {
	byTarget := map[string]*LedgerTargetRollup{}
	durations := map[string][]uint64{}
	for _, call := range calls {
		target := call.TargetScenario
		if target == "" {
			target = "unknown"
		}
		rollup := byTarget[target]
		if rollup == nil {
			rollup = &LedgerTargetRollup{TargetScenario: target}
			byTarget[target] = rollup
		}
		rollup.Calls++
		if call.Outcome != "success" {
			rollup.Failures++
		}
		rollup.TotalDurationMS += call.DurationMS
		durations[target] = append(durations[target], call.DurationMS)
	}
	out := make([]LedgerTargetRollup, 0, len(byTarget))
	for target, rollup := range byTarget {
		values := durations[target]
		sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
		rollup.MedianDurationMS = values[len(values)/2]
		out = append(out, *rollup)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TargetScenario < out[j].TargetScenario })
	return out
}

func projectionAvailability(calls []CrossScenarioCall) Availability {
	if len(calls) == 0 {
		return Availability{State: AvailabilityPolicyAbsent, Reason: "vrooli-events receipt projection policy produced no receipts"}
	}
	for _, call := range calls {
		if call.ProjectionDropCount > 0 {
			return Availability{State: AvailabilityOversized, Reason: "receipt projection exceeded bounded storage caps"}
		}
		if len(call.Projection) > 0 {
			return Availability{State: AvailabilityAvailable}
		}
		if call.PolicyVersion != "" {
			return Availability{State: AvailabilityEmpty, Reason: "receipt projection policy returned no fields"}
		}
	}
	return Availability{State: AvailabilityPolicyAbsent, Reason: "vrooli-events receipt projection policy is absent"}
}

func raiseEpisodeSeverityForRepeatedRules(episodes []runsignal.FrictionEpisode, spans []runsignal.SelfReportSpan, events []*domain.RunEvent) {
	position := make(map[string]int, len(events))
	for index, event := range events {
		if event != nil {
			position[event.ID.String()] = index
		}
	}
	for index := range episodes {
		start, startOK := position[episodes[index].StartEventID]
		end, endOK := position[episodes[index].EndEventID]
		if !startOK || !endOK || end < start {
			continue
		}
		seen := map[string]bool{}
		for _, span := range spans {
			at, ok := position[span.EventID]
			if !ok || at < start || at > end {
				continue
			}
			if seen[span.RuleID] {
				episodes[index].Severity = "recurring"
				break
			}
			seen[span.RuleID] = true
		}
	}
}

func (r *RunReport) foldEvents(events []*domain.RunEvent) {
	tools := map[string]*ToolSummary{}
	toolCalls := map[string]int{}
	fileReads := map[string]int{}
	costSeen := false
	var previous time.Time
	for _, event := range events {
		if event != nil {
			if !previous.IsZero() && event.Timestamp.After(previous) && event.Timestamp.Sub(previous) > r.LongestEventGap {
				r.LongestEventGap = event.Timestamp.Sub(previous)
			}
			if !event.Timestamp.IsZero() {
				previous = event.Timestamp
			}
			if call, ok := event.Data.(*domain.ToolCallEventData); ok {
				toolCalls[call.ToolName]++
				if toolCalls[call.ToolName] > 1 {
					r.RepeatedToolCalls++
				}
				if path := runsignal.ReadPath(call); path != "" {
					fileReads[path]++
					if fileReads[path] == 2 {
						r.FilesReadMoreThanOnce++
					}
				}
			}
			costSeen = r.foldEvent(event, tools, costSeen)
		}
	}
	for _, v := range tools {
		v.Unresolved = max(0, v.Calls-v.Successes-v.Failures)
		r.Tools = append(r.Tools, *v)
	}
	sort.Slice(r.Tools, func(i, j int) bool { return r.Tools[i].Name < r.Tools[j].Name })
}

func (r *RunReport) foldEvent(event *domain.RunEvent, tools map[string]*ToolSummary, costSeen bool) bool {
	r.Events[string(event.EventType)]++
	if event.EventType == domain.EventTypeModelFallbackAttempted {
		r.FallbackCount++
	}
	switch data := event.Data.(type) {
	case *domain.UsageEventData:
		if !costSeen {
			r.CostUSD, r.Tokens = 0, 0
			costSeen = true
		}
		r.Tokens += data.InputTokens + data.OutputTokens + data.CacheReadTokens + data.CacheCreationTokens
	case *domain.ChargeEventData:
		if data.AmountMicroUSD != nil {
			r.CostUSD += float64(*data.AmountMicroUSD) / 1_000_000
		}
	case *domain.ToolCallEventData:
		r.foldToolCall(data, toolSummary(tools, data.ToolName))
	case *domain.ToolResultEventData:
		r.foldToolResult(data, toolSummary(tools, data.ToolName))
	}
	return costSeen
}

func (r *RunReport) foldToolCall(data *domain.ToolCallEventData, summary *ToolSummary) {
	summary.Calls++
	if projectOwned(data.ToolName, data.Input) {
		r.ProjectOwnedToolCalls++
		return
	}
	r.ExternalToolCalls++
}

func (r *RunReport) foldToolResult(data *domain.ToolResultEventData, summary *ToolSummary) {
	if data.Success {
		summary.Successes++
		return
	}
	summary.Failures++
}

func toolSummary(tools map[string]*ToolSummary, name string) *ToolSummary {
	if summary := tools[name]; summary != nil {
		return summary
	}
	summary := &ToolSummary{Name: name}
	tools[name] = summary
	return summary
}

func projectOwned(name string, input map[string]interface{}) bool {
	if strings.Contains(strings.ToLower(name), "shell") || strings.Contains(strings.ToLower(name), "bash") {
		if command, ok := input["command"].(string); ok {
			return runsignal.ResolveCatalog(command).State == AvailabilityResolved
		}
	}
	return runsignal.ResolveCatalog(name).State == AvailabilityResolved
}

// Text is the bounded human projection shared by the investigation seed and
// render package. Keeping it beside the projection lets orchestration depend
// on data, not on a presentation package.
func Text(r *RunReport) string {
	if r == nil {
		return "Run report unavailable\n"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Run %s\nStatus: %s\n", r.RunID, r.Status)
	if r.ExitCode != nil {
		fmt.Fprintf(&b, "Exit code: %d\n", *r.ExitCode)
	}
	if r.Error != "" {
		fmt.Fprintf(&b, "Error: %s\n", r.Error)
	}
	fmt.Fprintf(&b, "Duration: %s | heartbeat gap: %s\nTurns: %d | tokens: %d | cost: $%.4f\n", r.Duration.Round(0), r.HeartbeatGap.Round(0), r.Turns, r.Tokens, r.CostUSD)
	fmt.Fprintf(&b, "Final output: %s (%s), candidates=%d\n", r.Result.SelectionStatus, r.Result.SelectionRule, r.Result.CandidateCount)
	if r.Result.StructuredStatus != "" {
		fmt.Fprintf(&b, "Structured result: %s (%s) diagnostics=%s\n", r.Result.StructuredStatus, r.Result.StructuredMethod, strings.Join(r.Result.DiagnosticCodes, ","))
	}
	fmt.Fprintf(&b, "Model: requested=%s actual=%s fallbacks=%d\n", unavailable(r.RequestedModel), unavailable(r.ActualModel), r.FallbackCount)
	fmt.Fprintf(&b, "Tools: project-owned=%d external=%d\n", r.ProjectOwnedToolCalls, r.ExternalToolCalls)
	fmt.Fprintf(&b, "Efficiency: repeated tool calls=%d files reread=%d longest event gap=%s\n", r.RepeatedToolCalls, r.FilesReadMoreThanOnce, r.LongestEventGap.Round(0))
	for _, tool := range r.Tools {
		fmt.Fprintf(&b, "  %s: calls=%d success=%d failed=%d\n", tool.Name, tool.Calls, tool.Successes, tool.Failures)
	}
	keys := make([]string, 0, len(r.Events))
	for key := range r.Events {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	b.WriteString("Events:")
	for _, key := range keys {
		fmt.Fprintf(&b, " %s=%d", key, r.Events[key])
	}
	b.WriteByte('\n')
	fmt.Fprintf(&b, "Diff: files=%d bytes=%d (%s)\n", r.Diff.Files, r.Diff.Bytes, r.Diff.Available.State)
	fmt.Fprintf(&b, "Receipts: %s (%d)\n", r.ReceiptsAvailability.State, r.ReceiptCount)
	fmt.Fprintf(&b, "Next: agent-manager run result %s; agent-manager run events %s; agent-manager run tools %s --failed; agent-manager run diff %s; agent-manager run receipts %s\n", r.RunID, r.RunID, r.RunID, r.RunID, r.RunID)
	return b.String()
}

func unavailable(value string) string {
	if value == "" {
		return "unavailable"
	}
	return value
}
