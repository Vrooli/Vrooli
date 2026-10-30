// Package runreport builds the bounded, diagnostic projection consumed by run
// inspection, investigations, and human-facing detail views.
package runreport

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"agent-manager/internal/adapters/sandbox"
	sharedavailability "agent-manager/internal/availability"
	"agent-manager/internal/domain"
	"agent-manager/internal/runsignal"

	"github.com/google/uuid"
	eventdomain "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
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
	Count          int
	UnmatchedCount int `json:"unmatchedCount"`
	EventIDs       []string
	Calls          []CrossScenarioCall
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
	MatchConfidence     string         `json:"matchConfidence,omitempty"`
	MatchReason         string         `json:"matchReason,omitempty"`
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
	Name           string `json:"name"`
	Calls          int    `json:"calls"`
	Successes      int    `json:"successes"`
	Failures       int    `json:"failures"`
	Unresolved     int    `json:"unresolved"`
	Returned       int    `json:"returned"`
	UnknownOutcome int    `json:"unknownOutcome"`
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
	Tokens                 int64                       `json:"tokens"`
	CostUSD                float64                     `json:"costUsd"`
	CostSource             string                      `json:"costSource,omitempty"`
	ChargeReason           string                      `json:"chargeReason,omitempty"`
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
	// HiddenQuarantinedEpisodes counts episodes of quarantined or informational
	// patterns left out of Episodes.
	HiddenQuarantinedEpisodes int                          `json:"hiddenQuarantinedEpisodes,omitempty"`
	SelfReportSpans           []runsignal.SelfReportSpan   `json:"selfReportSpans,omitempty"`
	HelpRecoveries            int                          `json:"helpRecoveries"`
	UnknownInvocations        int                          `json:"unknownInvocations"`
	TimeAccounting            runsignal.TimeAccounting     `json:"timeAccounting"`
	Goal                      GoalProgress                 `json:"goal"`
	InvocationValidity        EvidenceValidity             `json:"invocationValidity"`
	WorkReferences            []*eventdomain.WorkReference `json:"workReferences,omitempty"`
}

func Build(ctx context.Context, source Source, runID uuid.UUID) (*RunReport, error) {
	if source == nil {
		return nil, fmt.Errorf("run report source is unavailable")
	}
	run, err := source.Run(ctx, runID)
	if err != nil {
		return nil, err
	}
	r := newReport(run)
	imported := run.ExecutionMode.Normalized() == domain.ExecutionModeImported
	if imported {
		r.Diff.Available = Availability{State: AvailabilityUnavailable, Reason: "imported run has no sandbox"}
		r.ReceiptsAvailability = Availability{State: AvailabilityUnavailable, Reason: "imported run predates run identity correlation"}
		r.LedgerAvailability = Availability{State: AvailabilityUnavailable, Reason: "imported run predates run identity correlation"}
	}
	r.foldRun(run)
	events, err := source.Events(ctx, runID)
	if err != nil {
		r.EventsAvailability = Availability{State: AvailabilityUnavailable, Reason: err.Error()}
	} else if err := r.foldEventEvidence(ctx, source, run, events, imported); err != nil {
		return nil, err
	}
	if !imported {
		if diff, err := source.Diff(ctx, runID); err == nil && diff != nil {
			r.Diff = DiffSummary{Files: len(diff.Files), Bytes: run.TotalSizeBytes, Available: Availability{State: AvailabilityAvailable}}
		} else if err != nil {
			r.Diff.Available.Reason = err.Error()
		}
	}
	if receipts, ok := source.(ReceiptSource); ok {
		if err := r.foldReceipts(ctx, source, receipts, runID, events); err != nil {
			return nil, err
		}
	}
	return r, nil
}

// newReport is the report before any evidence is read: every availability
// starts unavailable and each evidence step upgrades what it observed.
func newReport(run *domain.Run) *RunReport {
	r := &RunReport{RunID: run.ID, Status: run.Status, ExitCode: run.ExitCode, Error: run.ErrorMsg, Events: map[string]int{}, EventsAvailability: Availability{State: AvailabilityAvailable}, ReceiptsAvailability: Availability{State: AvailabilityUnavailable, Reason: "receipt reader is not configured"}, LedgerAvailability: Availability{State: AvailabilityUnavailable, Reason: "receipt reader is not configured"}, ProjectionAvailability: Availability{State: AvailabilityUnavailable, Reason: "receipt reader is not configured"}, RequestedModel: run.RequestedModel, ActualModel: run.ActualModel, WorkReferences: run.WorkReferences, Diff: DiffSummary{Files: run.ChangedFiles, Bytes: run.TotalSizeBytes, Available: Availability{State: AvailabilityUnavailable}}}
	r.Goal = deriveGoalProgress(run, nil, nil)
	r.InvocationValidity = invocationValidity(nil)
	return r
}

// foldRun copies the run record's own duration, summary and result fields.
func (r *RunReport) foldRun(run *domain.Run) {
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
}

// foldEventEvidence derives everything that reads the complete event stream:
// tool/time accounting, invocation facts, episodes, self-report spans and goal
// progress. Durable projections win over replay when the source has them.
func (r *RunReport) foldEventEvidence(ctx context.Context, source Source, run *domain.Run, events []*domain.RunEvent, imported bool) error {
	runID := run.ID
	r.foldEvents(events)
	// Source.Events is the complete stream, so the run's one usage account
	// over it is the token total whenever the stream retains usage.
	if hasUsage(events) {
		r.Tokens = runsignal.AccountUsage(events).TotalTokens()
	}
	r.TimeAccounting = runsignal.DeriveTimeAccounting(events, run.StartedAt, run.EndedAt)
	if durable, ok := source.(DurableTimeAccountingSource); ok {
		accounting, present, err := durable.DurableTimeAccounting(ctx, runID)
		if err != nil {
			return fmt.Errorf("read durable time accounting: %w", err)
		}
		if present {
			r.TimeAccounting = accounting
		}
	}
	ownershipFromProjection := false
	r.InvocationFacts = nil
	if durable, ok := source.(DurableInvocationFactSource); ok {
		facts, present, err := durable.DurableInvocationFacts(ctx, runID)
		if err != nil {
			return fmt.Errorf("read durable invocation facts: %w", err)
		}
		r.InvocationFacts, ownershipFromProjection = facts, present
	}
	if !ownershipFromProjection {
		r.InvocationFacts = runsignal.DeriveInvocationFacts(events)
	}
	r.foldInvocationFacts(ownershipFromProjection)
	r.InvocationValidity = invocationValidity(r.InvocationFacts)
	episodes, present := []runsignal.FrictionEpisode(nil), false
	if durable, ok := source.(DurableEpisodeSource); ok {
		var err error
		if episodes, present, err = durable.DurableEpisodes(ctx, runID); err != nil {
			return fmt.Errorf("read durable episodes: %w", err)
		}
	}
	if present {
		r.Episodes = episodes
	} else {
		r.Episodes = runsignal.DeriveEpisodes(r.InvocationFacts, events, runsignal.WithGoalRun(strings.TrimSpace(run.GoalID) != ""))
	}
	if imported {
		for index := range r.Episodes {
			r.Episodes[index].OwnerConfidence = "manifest-derived-historical"
		}
	}
	spans, present := []runsignal.SelfReportSpan(nil), false
	if durable, ok := source.(DurableSelfReportSpanSource); ok {
		var err error
		if spans, present, err = durable.DurableSelfReportSpans(ctx, runID); err != nil {
			return fmt.Errorf("read durable self-report spans: %w", err)
		}
	}
	if present {
		r.SelfReportSpans = spans
	} else {
		r.SelfReportSpans = runsignal.DeriveSelfReportSpans(events)
	}
	raiseEpisodeSeverityForRepeatedRules(r.Episodes, r.SelfReportSpans, events)
	r.Goal = deriveGoalProgress(run, events, r.Episodes)
	r.Episodes, r.HiddenQuarantinedEpisodes = runsignal.ShownEpisodes(r.Episodes)
	return nil
}

// foldInvocationFacts rolls the facts up into retry, help-recovery and unknown
// counts. When the facts are the durable projection, they are also the
// runner-aware ownership authority, so imported Codex wrapper calls do not fall
// back to tool-name heuristics.
func (r *RunReport) foldInvocationFacts(ownershipFromProjection bool) {
	for _, fact := range r.InvocationFacts {
		if fact.RetryOfCallEventID != "" {
			r.RepeatedToolCalls++
		}
		if fact.HelpRecovery {
			r.HelpRecoveries++
		}
		if fact.Ownership == "unknown" || fact.Availability == "unknown" {
			r.UnknownInvocations++
		}
	}
	if !ownershipFromProjection {
		return
	}
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

// foldReceipts joins the run's cross-scenario receipts and persists the ledger
// and receipt evidence back through the source's optional stores.
func (r *RunReport) foldReceipts(ctx context.Context, source Source, receipts ReceiptSource, runID uuid.UUID, events []*domain.RunEvent) error {
	summary, err := receipts.Receipts(ctx, runID)
	if err != nil {
		r.ReceiptsAvailability = Availability{State: AvailabilityDegraded, Reason: err.Error()}
	} else {
		r.ReceiptsAvailability = Availability{State: summary.State, Reason: summary.Reason}
		r.LedgerAvailability = Availability{State: summary.State, Reason: summary.Reason}
		r.ReceiptCount = summary.Count
		if summary.UnmatchedCount > 0 {
			r.ReceiptsAvailability.Reason = fmt.Sprintf("%d receipt candidates were not safely matched", summary.UnmatchedCount)
		}
		r.ReceiptEvidenceIDs = append([]string(nil), summary.EventIDs...)
		r.CrossScenarioCalls = append([]CrossScenarioCall(nil), summary.Calls...)
		r.LedgerTargetRollups = ledgerTargetRollups(r.CrossScenarioCalls)
		r.ProjectionAvailability = projectionAvailability(r.CrossScenarioCalls)
		r.Episodes = UpgradeEpisodeOwnership(r.Episodes, events, r.CrossScenarioCalls, r.LedgerAvailability)
		if store, ok := source.(LedgerStore); ok {
			if err := store.ReplaceCrossScenarioCalls(ctx, runID, string(r.LedgerAvailability.State), r.CrossScenarioCalls); err != nil {
				return fmt.Errorf("persist cross-scenario ledger: %w", err)
			}
		}
	}
	if store, ok := source.(ReceiptJoinStore); ok {
		if err := store.ReplaceReceiptEvidence(ctx, runID, string(r.ReceiptsAvailability.State), r.ReceiptEvidenceIDs); err != nil {
			return fmt.Errorf("persist receipt evidence: %w", err)
		}
	}
	return nil
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
	if !firstMetAt.IsZero() {
		progress.TokenCostBeforeFirstMet = runsignal.AccountUsage(events).Between(events, time.Time{}, firstMetAt)
	}
	if !metSeen {
		progress.Availability = Availability{State: AvailabilityDegraded, Reason: "goal is declared but no first-met marker is available"}
	}
	for i, event := range events {
		if event == nil || (!firstMetAt.IsZero() && event.Timestamp.After(firstMetAt)) {
			continue
		}
		if _, ok := event.Data.(*domain.MessageEventData); !ok || domain.MessageSource(event) != domain.MessageSourceOperator {
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
	fileReads := map[string]int{}
	callNames := map[string]string{}
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
				if call.ToolCallID != "" {
					callNames[call.ToolCallID] = call.ToolName
				}
				if path := runsignal.ReadPath(call); path != "" {
					fileReads[path]++
					if fileReads[path] == 2 {
						r.FilesReadMoreThanOnce++
					}
				}
			}
			// Historical output rows may omit the name; bind by the existing call ID.
			if result, ok := event.Data.(*domain.ToolResultEventData); ok && result.ToolName == "" && result.ToolCallID != "" {
				copyEvent, copyData := *event, *result
				copyData.ToolName = callNames[result.ToolCallID]
				copyEvent.Data = &copyData
				event = &copyEvent
			}
			costSeen = r.foldEvent(event, tools, costSeen)
		}
	}
	for _, v := range tools {
		v.Unresolved = max(0, v.Calls-v.Returned)
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
			r.CostUSD = 0
			costSeen = true
		}
	case *domain.ChargeEventData:
		if data.AmountMicroUSD != nil {
			r.CostUSD += float64(*data.AmountMicroUSD) / 1_000_000
		}
		if r.CostSource == "" || r.CostSource == "unknown" {
			r.CostSource = string(data.Basis)
		}
		if r.ChargeReason == "" {
			r.ChargeReason = data.ChargeReason
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
	summary.Returned++
	switch runsignal.ToolResultOutcome(data) {
	case "success":
		summary.Successes++
	case "failure":
		summary.Failures++
	case "unknown":
		summary.UnknownOutcome++
	}
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
	fmt.Fprintf(&b, "Duration: %s | heartbeat gap: %s\nTurns: %d | tokens: %d | cost: %s\n", r.Duration.Round(0), r.HeartbeatGap.Round(0), r.Turns, r.Tokens, costText(r))
	fmt.Fprintf(&b, "Final output: %s (%s), candidates=%d\n", r.Result.SelectionStatus, r.Result.SelectionRule, r.Result.CandidateCount)
	if r.Result.StructuredStatus != "" {
		fmt.Fprintf(&b, "Structured result: %s (%s) diagnostics=%s\n", r.Result.StructuredStatus, r.Result.StructuredMethod, strings.Join(r.Result.DiagnosticCodes, ","))
	}
	fmt.Fprintf(&b, "Model: requested=%s actual=%s fallbacks=%d\n", unavailable(r.RequestedModel), unavailable(r.ActualModel), r.FallbackCount)
	fmt.Fprintf(&b, "Tools: project-owned=%d external=%d\n", r.ProjectOwnedToolCalls, r.ExternalToolCalls)
	fmt.Fprintf(&b, "Efficiency: recognized command retries=%d files reread=%d longest event gap=%s\n", r.RepeatedToolCalls, r.FilesReadMoreThanOnce, r.LongestEventGap.Round(0))
	for _, tool := range r.Tools {
		fmt.Fprintf(&b, "  %s: calls=%d returned=%d success=%d failed=%d unknown=%d unresolved=%d\n", tool.Name, tool.Calls, tool.Returned, tool.Successes, tool.Failures, tool.UnknownOutcome, tool.Unresolved)
	}
	keys := make([]string, 0, len(r.Events))
	for key := range r.Events {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	b.WriteString("Events:")
	if r.EventsAvailability.State != AvailabilityAvailable && r.EventsAvailability.State != AvailabilityEmpty {
		fmt.Fprintf(&b, " unavailable (%s)", r.EventsAvailability.State)
	}
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

func costText(r *RunReport) string {
	switch strings.ToLower(strings.TrimSpace(r.CostSource)) {
	case "subscription":
		return "subscription (allocated outside per-run charge)"
	case "local":
		return "local (no monetary charge)"
	case "unpriced", "unknown", "":
		if r.ChargeReason != "" {
			return "unpriced (" + r.ChargeReason + ")"
		}
		return "unpriced"
	default:
		return fmt.Sprintf("$%.4f (%s)", r.CostUSD, r.CostSource)
	}
}

func hasUsage(events []*domain.RunEvent) bool {
	for _, event := range events {
		if event != nil {
			if _, ok := event.Data.(*domain.UsageEventData); ok {
				return true
			}
		}
	}
	return false
}
