// Package measures exposes swarm-manager's declared analytical measures: the
// granular, individually addressable replacement for the retired aggregate endpoint.
//
// Each measure answers ONE question-family ("how many backlog items were
// completed this week"), parameterized by the shared canonical
// vrooli.measures.v1.TimeWindow. Every measure is surfaced two ways over a
// SINGLE shared compute path so the two can never disagree:
//
//   - the packages/measures-go serve registry, mounted at /measures
//     (GET /measures/declarations + POST /measures/execute) — the contract the
//     measures-health behavioral probe and the search-hub central index call;
//   - the Connect-RPC MeasuresService (RegisterRoutes) — the typed surface the
//     CLI/UI consume.
//
// All counts are computed over the append-only event log (internal/eventlog)
// via real SQL aggregates, never a list-and-filter.
//
// Proto: packages/proto/schemas/swarm-manager/v1/measures/measures.proto
package measures

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	measures "github.com/vrooli/measures-go"

	measuresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/measures/v1"
	smmeasuresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/measures"
	smmeasuresconnect "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/measures/measures_v1connect"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/eventlog"
)

// Counter is the narrow event-log substrate the measures compute against. The
// production *eventlog.SQLiteRepository satisfies it; tests inject a fake.
type Counter interface {
	CountEventsInRange(ctx context.Context, eventType eventlog.EventType, from, to time.Time) (int, error)
	CountStatusTransitionsInRange(ctx context.Context, eventType eventlog.EventType, toStatus string, from, to time.Time) (int, error)
}

// PlanRefStore is the durable backlog projection needed by the plan_ref gauge.
// It is deliberately narrower than backlog.Store so the measures boundary owns
// no backlog mutations.
type PlanRefStore interface {
	LoadAll(kinds []backlog.BacklogKind) ([]backlog.BacklogItem, error)
}

// computeFn computes one measure's scalar value over [from, to) and returns the
// value plus the executed-query provenance string. Shared by the serve registry
// and the Connect RPC so a measure and its RPC report identical numbers.
type computeFn func(ctx context.Context, c Counter, from, to time.Time) (int64, string, error)
type rateComputeFn func(ctx context.Context, c Counter, from, to time.Time) (float64, int64, string, error)
type durationComputeFn func(ctx context.Context, c Counter, from, to time.Time) (durationSummary, string, error)

// durationSummary preserves the three values that the former stats response
// exposed for duration measures. The registry publishes Average as its scalar
// answer; the typed RPC also exposes Median and SampleSize.
type durationSummary struct {
	Average    float64
	Median     float64
	SampleSize int64
}

// spec is one measure: its declaration (the SSOT identity surfaced to the index
// and the probe) plus its compute path.
type spec struct {
	decl            measures.MeasureDeclaration
	compute         computeFn
	rate            rateComputeFn
	duration        durationComputeFn
	durableGauge    bool
	milestoneHealth bool
}

// Measure names — the conventional "<domain>.<command>" identifiers. They MUST
// equal the manifest's domain.command so the behavioral probe's /execute call
// resolves them.
const (
	MeasureBacklogCompleted         = "backlog.completed"
	MeasureBacklogCreated           = "backlog.created"
	MeasureBacklogNetDelta          = "backlog.net_delta"
	MeasureBacklogOpen              = "backlog.open"
	MeasureBacklogBlocked           = "backlog.blocked"
	MeasureBacklogLeadTime          = "backlog.lead_time"
	MeasureExecutionCompleted       = "execution.completed"
	MeasureExecutionDuration        = "execution.duration"
	MeasureExecutionReviewRate      = "execution.review_rate"
	MeasureGoalMilestoneHealth      = "goal.milestone_health"
	MeasureGoalCreated              = "goal.created"
	MeasureAgentSessionCreated      = "agent_session.created"
	MeasurePlanRefCount             = "plan_ref.count"
	MeasureAgentSessionProposalRate = "agent_session.proposal_rate"
	MeasureExecutionSuccessRate     = "execution.success_rate"
)

// specs returns the full measure set in a stable order. now() anchors relative
// time-window resolution (injected for deterministic tests).
func specs() []spec {
	return []spec{
		{
			decl:            milestoneHealthDecl(),
			milestoneHealth: true,
		},
		{
			decl: windowDecl(
				MeasureBacklogCompleted, "backlog", "CountBacklogCompleted",
				"How many backlog items were completed in a time window.",
				[]string{
					"how many backlog items did we complete this week",
					"backlog items closed last month",
					"how many work items were finished in the last 7 days",
				},
				"items", "{count} backlog items completed ({window})",
			),
			compute: countEvents(eventlog.EventBacklogStatusChanged, "completed", true),
		},
		{
			decl: windowDecl(
				MeasureBacklogCreated, "backlog", "CountBacklogCreated",
				"How many backlog items were created in a time window.",
				[]string{
					"how many backlog items were created this week",
					"new work items added last month",
					"how many backlog items did we add in the last 30 days",
				},
				"items", "{count} backlog items created ({window})",
			),
			compute: countEvents(eventlog.EventBacklogCreated, "", false),
		},
		{
			decl: windowDurationDecl(
				MeasureBacklogLeadTime, "backlog", "BacklogLeadTime",
				"How long completed backlog items took from creation to completion.",
				[]string{
					"what is the average backlog lead time this week",
					"how long did completed work items take last month",
					"median time from backlog creation to completion",
				},
				"hours", "{average} average backlog lead-time hours ({window})",
			),
			duration: backlogLeadTime,
		},
		{
			decl: windowDecl(
				MeasureBacklogNetDelta, "backlog", "CountBacklogNetDelta",
				"By how many items did the backlog grow or shrink in a time window.",
				[]string{
					"did the backlog grow or shrink this week",
					"what is the net change in backlog items last month",
					"how much work was added minus completed in the last 30 days",
				},
				"items", "{count} net backlog item change ({window})",
			),
			compute: backlogNetDelta,
		},
		{
			decl: gaugeDecl(
				MeasureBacklogOpen, "backlog", "CountBacklogOpen",
				"How many actionable backlog items are currently open.",
				[]string{
					"how many backlog items are open",
					"what work remains actionable in the backlog",
					"current open work item count",
				},
				"items", "{count} actionable backlog items are open",
			),
			durableGauge: true,
		},
		{
			decl: windowRateDecl(
				MeasureAgentSessionProposalRate, "agent_session", "AgentSessionProposalRate",
				"What proportion of agent-session proposals were applied in a time window.",
				[]string{
					"what percentage of agent session proposals were applied",
					"agent session proposal conversion rate this week",
					"how often did session proposals get applied last month",
				},
				"ratio", "{rate} proposal apply rate ({window})",
			),
			rate: proposalApplyRate,
		},
		{
			decl: windowRateDecl(
				MeasureExecutionSuccessRate, "execution", "ExecutionSuccessRate",
				"What proportion of terminal executions completed successfully in a time window.",
				[]string{
					"what is the execution success rate this week",
					"how often did agent executions succeed last month",
					"percentage of successful runs in the last 30 days",
				},
				"ratio", "{rate} execution success rate ({window})",
			),
			rate: executionSuccessRate,
		},
		{
			decl: gaugeDecl(
				MeasureBacklogBlocked, "backlog", "CountBacklogBlocked",
				"How many backlog items are currently blocked by unresolved dependencies.",
				[]string{
					"how many backlog items are blocked",
					"what work is blocked by dependencies",
					"current blocked backlog item count",
				},
				"items", "{count} backlog items are blocked",
			),
			durableGauge: true,
		},
		{
			decl: windowDecl(
				MeasureExecutionCompleted, "execution", "CountExecutionsCompleted",
				"How many executions completed in a time window.",
				[]string{
					"how many executions completed this week",
					"agent executions finished last month",
					"how many runs completed in the last 7 days",
				},
				"executions", "{count} executions completed ({window})",
			),
			compute: countEvents(eventlog.EventExecutionCompleted, "", false),
		},
		{
			decl: windowDurationDecl(
				MeasureExecutionDuration, "execution", "ExecutionDuration",
				"How long completed executions ran.",
				[]string{
					"what is the average execution duration this week",
					"how long did agent executions run last month",
					"median execution duration in the last 30 days",
				},
				"minutes", "{average} average execution minutes ({window})",
			),
			duration: executionDuration,
		},
		{
			decl: windowRateDecl(
				MeasureExecutionReviewRate, "execution", "ExecutionReviewRate",
				"What proportion of terminal executions completed at least one review round.",
				[]string{
					"what percentage of executions received a completed review this week",
					"execution review rate last month",
					"how often do terminal runs reach a review round",
				},
				"ratio", "{rate} execution review rate ({window})",
			),
			rate: executionReviewRate,
		},
		{
			decl: windowDecl(
				MeasureGoalCreated, "goal", "CountGoalsCreated",
				"How many goals were created in a time window.",
				[]string{
					"how many goals were created this week",
					"new goals started last month",
					"how many goals did we open this quarter",
				},
				"goals", "{count} goals created ({window})",
			),
			compute: countEvents(eventlog.EventGoalCreated, "", false),
		},
		{
			decl: windowDecl(
				MeasureAgentSessionCreated, "agent_session", "CountAgentSessionsCreated",
				"How many agent sessions were created in a time window.",
				[]string{
					"how many agent sessions were created this week",
					"new conversations started last month",
					"how many agent sessions did we open in the last 7 days",
				},
				"sessions", "{count} agent sessions created ({window})",
			),
			compute: countEvents(eventlog.EventAgentSessionCreated, "", false),
		},
		{
			decl: gaugeDecl(
				MeasurePlanRefCount, "plan_ref", "CountPlanRefs",
				"How many backlog items currently reference a canonical plan.",
				[]string{
					"how many backlog items have a plan reference",
					"count work items linked to plan-manager plans",
					"how many canonical plans are attached to backlog work",
				},
				"items", "{count} backlog items have a plan reference",
			),
			durableGauge: true,
		},
	}
}

func milestoneHealthDecl() measures.MeasureDeclaration {
	return measures.MeasureDeclaration{
		Name:     MeasureGoalMilestoneHealth,
		Scenario: "swarm-manager",
		Domain:   "goal",
		Intent:   "What is the current delivery health of each milestone.",
		Questions: []string{
			"which milestones have blocked work",
			"current milestone completion health",
			"how much work remains in each milestone",
		},
		Result: measures.Result{Kind: measures.ResultTable, ValueField: "milestone", Unit: "milestones", SummaryTemplate: "current health for {count} milestones"},
		Effect: measures.EffectRead, RunEligible: true, Service: "MeasuresService", Method: "GoalMilestoneHealth",
	}
}

// windowDecl builds a read-only, run-eligible scalar measure declaration with a
// single canonical time_window param defaulting to this_week (→ full tier).
func windowDecl(name, domain, method, intent string, questions []string, unit, summary string) measures.MeasureDeclaration {
	return measures.MeasureDeclaration{
		Name:      name,
		Scenario:  "swarm-manager",
		Domain:    domain,
		Intent:    intent,
		Questions: questions,
		Params: map[string]measures.Param{
			"window": {
				Name:    "window",
				Type:    measures.ParamTypeTimeWindow,
				Default: string(measures.TokenThisWeek),
			},
		},
		Result: measures.Result{
			Kind:            measures.ResultScalar,
			ValueField:      "count",
			Unit:            unit,
			SummaryTemplate: summary,
		},
		Effect:      measures.EffectRead,
		RunEligible: true,
		Service:     "MeasuresService",
		Method:      method,
	}
}

func gaugeDecl(name, domain, method, intent string, questions []string, unit, summary string) measures.MeasureDeclaration {
	return measures.MeasureDeclaration{
		Name:      name,
		Scenario:  "swarm-manager",
		Domain:    domain,
		Intent:    intent,
		Questions: questions,
		Result: measures.Result{
			Kind:            measures.ResultScalar,
			ValueField:      "count",
			Unit:            unit,
			SummaryTemplate: summary,
		},
		Effect:      measures.EffectRead,
		RunEligible: true,
		Service:     "MeasuresService",
		Method:      method,
	}
}

func windowRateDecl(name, domain, method, intent string, questions []string, unit, summary string) measures.MeasureDeclaration {
	d := windowDecl(name, domain, method, intent, questions, unit, summary)
	d.Result.ValueField = "rate"
	return d
}

func windowDurationDecl(name, domain, method, intent string, questions []string, unit, summary string) measures.MeasureDeclaration {
	d := windowDecl(name, domain, method, intent, questions, unit, summary)
	d.Result.ValueField = "average"
	return d
}

// countEvents builds a compute that counts events of eventType in the window.
// When transition is true it counts status_changed events whose `to` == status
// (the "completed" family); otherwise it is a bare event-type count.
func countEvents(eventType eventlog.EventType, status string, transition bool) computeFn {
	return func(ctx context.Context, c Counter, from, to time.Time) (int64, string, error) {
		var (
			n     int
			err   error
			query string
		)
		if transition {
			n, err = c.CountStatusTransitionsInRange(ctx, eventType, status, from, to)
			query = fmt.Sprintf(
				"SELECT COUNT(*) FROM events WHERE event_type=%q AND json_extract(metadata,'$.to')=%q AND timestamp >= %q AND timestamp < %q",
				eventType, status, from.UTC().Format(time.RFC3339Nano), to.UTC().Format(time.RFC3339Nano),
			)
		} else {
			n, err = c.CountEventsInRange(ctx, eventType, from, to)
			query = fmt.Sprintf(
				"SELECT COUNT(*) FROM events WHERE event_type=%q AND timestamp >= %q AND timestamp < %q",
				eventType, from.UTC().Format(time.RFC3339Nano), to.UTC().Format(time.RFC3339Nano),
			)
		}
		if err != nil {
			return 0, "", err
		}
		return int64(n), query, nil
	}
}

// backlogNetDelta is created minus completed over the same half-open window.
// It deliberately delegates both sides to the Counter methods used by the
// individual measures, keeping the analytical definition transport-neutral.
func backlogNetDelta(ctx context.Context, c Counter, from, to time.Time) (int64, string, error) {
	created, err := c.CountEventsInRange(ctx, eventlog.EventBacklogCreated, from, to)
	if err != nil {
		return 0, "", err
	}
	completed, err := c.CountStatusTransitionsInRange(ctx, eventlog.EventBacklogStatusChanged, "completed", from, to)
	if err != nil {
		return 0, "", err
	}
	query := fmt.Sprintf("count(backlog.created) - count(backlog.status_changed to completed) for timestamp >= %q AND timestamp < %q", from.UTC().Format(time.RFC3339Nano), to.UTC().Format(time.RFC3339Nano))
	return int64(created - completed), query, nil
}

func proposalApplyRate(ctx context.Context, c Counter, from, to time.Time) (float64, int64, string, error) {
	created, err := c.CountEventsInRange(ctx, eventlog.EventAgentSessionProposalCreated, from, to)
	if err != nil {
		return 0, 0, "", err
	}
	applied, err := c.CountEventsInRange(ctx, eventlog.EventAgentSessionProposalApplied, from, to)
	if err != nil {
		return 0, 0, "", err
	}
	var rate float64
	if created > 0 {
		rate = float64(applied) / float64(created)
	}
	query := fmt.Sprintf("count(agent_session.proposal_applied) / count(agent_session.proposal_created) for timestamp >= %q AND timestamp < %q", from.UTC().Format(time.RFC3339Nano), to.UTC().Format(time.RFC3339Nano))
	return rate, int64(created), query, nil
}

func executionSuccessRate(ctx context.Context, c Counter, from, to time.Time) (float64, int64, string, error) {
	completed, err := c.CountEventsInRange(ctx, eventlog.EventExecutionCompleted, from, to)
	if err != nil {
		return 0, 0, "", err
	}
	failed, err := c.CountEventsInRange(ctx, eventlog.EventExecutionFailed, from, to)
	if err != nil {
		return 0, 0, "", err
	}
	sample := int64(completed + failed)
	var rate float64
	if sample > 0 {
		rate = float64(completed) / float64(sample)
	}
	query := fmt.Sprintf("count(execution.completed) / (count(execution.completed) + count(execution.failed)) for timestamp >= %q AND timestamp < %q", from.UTC().Format(time.RFC3339Nano), to.UTC().Format(time.RFC3339Nano))
	return rate, sample, query, nil
}

// eventReader is deliberately optional: count-only measures can use SQL
// aggregates, while the three metadata-derived measures need event payloads.
// The production event repository satisfies it; narrow counter fakes remain
// valid for the count measure tests.
type eventReader interface {
	All(context.Context) ([]eventlog.Event, error)
}

func readEvents(ctx context.Context, c Counter) ([]eventlog.Event, error) {
	r, ok := c.(eventReader)
	if !ok {
		return nil, fmt.Errorf("metadata-derived measure requires event reader")
	}
	return r.All(ctx)
}

func inWindow(ts, from, to time.Time) bool {
	return !ts.Before(from) && ts.Before(to)
}

func median(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sort.Float64s(values)
	middle := len(values) / 2
	if len(values)%2 == 1 {
		return values[middle]
	}
	return (values[middle-1] + values[middle]) / 2
}

func summarizeDuration(values []float64) durationSummary {
	var total float64
	for _, value := range values {
		total += value
	}
	summary := durationSummary{SampleSize: int64(len(values)), Median: median(values)}
	if summary.SampleSize > 0 {
		summary.Average = total / float64(summary.SampleSize)
	}
	return summary
}

func backlogLeadTime(ctx context.Context, c Counter, from, to time.Time) (durationSummary, string, error) {
	events, err := readEvents(ctx, c)
	if err != nil {
		return durationSummary{}, "", err
	}
	created := make(map[string]time.Time)
	for _, event := range events {
		if event.EventType == eventlog.EventBacklogCreated {
			created[event.EntityID] = event.Timestamp
		}
	}
	values := make([]float64, 0)
	for _, event := range events {
		if event.EventType != eventlog.EventBacklogStatusChanged || !inWindow(event.Timestamp, from, to) {
			continue
		}
		var payload eventlog.StatusChangePayload
		if json.Unmarshal(event.Metadata, &payload) != nil || payload.To != string(backlog.StatusCompleted) {
			continue
		}
		if started, ok := created[event.EntityID]; ok && !event.Timestamp.Before(started) {
			values = append(values, event.Timestamp.Sub(started).Hours())
		}
	}
	query := fmt.Sprintf("average and median(timestamp(backlog.status_changed to completed) - timestamp(backlog.created)) for completions in [%q, %q)", from.UTC().Format(time.RFC3339Nano), to.UTC().Format(time.RFC3339Nano))
	return summarizeDuration(values), query, nil
}

func executionDuration(ctx context.Context, c Counter, from, to time.Time) (durationSummary, string, error) {
	events, err := readEvents(ctx, c)
	if err != nil {
		return durationSummary{}, "", err
	}
	values := make([]float64, 0)
	for _, event := range events {
		if event.EventType != eventlog.EventExecutionCompleted || !inWindow(event.Timestamp, from, to) {
			continue
		}
		var payload eventlog.ExecutionCompletedPayload
		if json.Unmarshal(event.Metadata, &payload) == nil {
			values = append(values, payload.DurationSeconds/60)
		}
	}
	query := fmt.Sprintf("average and median(metadata.duration_seconds / 60) for execution.completed in [%q, %q)", from.UTC().Format(time.RFC3339Nano), to.UTC().Format(time.RFC3339Nano))
	return summarizeDuration(values), query, nil
}

func executionReviewRate(ctx context.Context, c Counter, from, to time.Time) (float64, int64, string, error) {
	events, err := readEvents(ctx, c)
	if err != nil {
		return 0, 0, "", err
	}
	terminal := make(map[string]struct{})
	reviewed := make(map[string]struct{})
	for _, event := range events {
		if !inWindow(event.Timestamp, from, to) {
			continue
		}
		switch event.EventType {
		case eventlog.EventExecutionCompleted, eventlog.EventExecutionFailed:
			terminal[event.EntityID] = struct{}{}
		case eventlog.EventReviewRoundCompleted:
			reviewed[event.EntityID] = struct{}{}
		}
	}
	var numerator int64
	for executionID := range terminal {
		if _, ok := reviewed[executionID]; ok {
			numerator++
		}
	}
	sample := int64(len(terminal))
	var rate float64
	if sample > 0 {
		rate = float64(numerator) / float64(sample)
	}
	query := fmt.Sprintf("distinct terminal executions with review.round_completed / distinct terminal executions for timestamps in [%q, %q)", from.UTC().Format(time.RFC3339Nano), to.UTC().Format(time.RFC3339Nano))
	return rate, sample, query, nil
}

// Declarations returns the measure declarations (no compute) for harvesting.
func Declarations() []measures.MeasureDeclaration {
	ss := specs()
	out := make([]measures.MeasureDeclaration, 0, len(ss))
	for _, s := range ss {
		out = append(out, s.decl)
	}
	return out
}

// NewRegistry builds the measures-go serve registry over the given counter.
// now() anchors relative time-window resolution; nil means time.Now.
func NewRegistry(c Counter, plans PlanRefStore, now func() time.Time) (*measures.Registry, error) {
	if now == nil {
		now = time.Now
	}
	reg := measures.NewRegistry(measures.WithClock(now))
	for _, s := range specs() {
		s := s
		err := reg.Register(s.decl, func(ctx context.Context, req measures.MeasureRequest) (measures.MeasureResult, error) {
			if s.durableGauge {
				count, query, err := countGauge(s.decl.Name, plans)
				if err != nil {
					return measures.MeasureResult{}, err
				}
				return measures.MeasureResult{
					Value:      strconv.FormatInt(count, 10),
					Provenance: measures.Provenance{ExecutedQuery: query},
				}, nil
			}
			if s.milestoneHealth {
				fields, query, err := milestoneHealth(plans)
				if err != nil {
					return measures.MeasureResult{}, err
				}
				return measures.MeasureResult{Fields: fields, Provenance: measures.Provenance{ExecutedQuery: query}}, nil
			}
			rng, err := resolveToken(req.Params["window"], now())
			if err != nil {
				return measures.MeasureResult{}, err
			}
			if s.rate != nil {
				rate, _, query, err := s.rate(ctx, c, rng.From, rng.To)
				if err != nil {
					return measures.MeasureResult{}, err
				}
				return measures.MeasureResult{Value: strconv.FormatFloat(rate, 'f', -1, 64), Provenance: measures.Provenance{ExecutedQuery: query}}, nil
			}
			if s.duration != nil {
				summary, query, err := s.duration(ctx, c, rng.From, rng.To)
				if err != nil {
					return measures.MeasureResult{}, err
				}
				return measures.MeasureResult{Value: strconv.FormatFloat(summary.Average, 'f', -1, 64), Provenance: measures.Provenance{ExecutedQuery: query}}, nil
			}
			val, query, err := s.compute(ctx, c, rng.From, rng.To)
			if err != nil {
				return measures.MeasureResult{}, err
			}
			return measures.MeasureResult{
				Value:      strconv.FormatInt(val, 10),
				Provenance: measures.Provenance{ExecutedQuery: query},
			}, nil
		})
		if err != nil {
			return nil, fmt.Errorf("measures: register %s: %w", s.decl.Name, err)
		}
	}
	return reg, nil
}

// MeasuresHandler returns the http.Handler serving GET /declarations and
// POST /execute (mounted under /measures by the caller).
func MeasuresHandler(c Counter, plans PlanRefStore, now func() time.Time) (http.Handler, error) {
	reg, err := NewRegistry(c, plans, now)
	if err != nil {
		return nil, err
	}
	return reg.Handler(), nil
}

// resolveToken resolves a string time-window token (this_week default) to a
// concrete [from, to) range in UTC.
func resolveToken(token string, now time.Time) (measures.Range, error) {
	t := measures.TimeWindowToken(token)
	if t == "" {
		t = measures.TokenThisWeek
	}
	return measures.ResolveToken(t, now, time.UTC)
}

// resolveProtoWindow resolves a proto TimeWindow (nil/unset → this_week) for the
// Connect RPCs.
func resolveProtoWindow(tw *measuresv1.TimeWindow, now time.Time) (measures.Range, error) {
	if tw == nil || tw.GetWindow() == nil {
		return measures.ResolveToken(measures.TokenThisWeek, now, time.UTC)
	}
	return measures.ResolveTimeWindow(tw, now, time.UTC)
}

// -----------------------------------------------------------------------------
// Connect-RPC MeasuresService — the typed surface, sharing the compute path.
// -----------------------------------------------------------------------------

// Handler implements vrooli.swarm_manager.v1.measures.MeasuresService.
type Handler struct {
	counter Counter
	plans   PlanRefStore
	now     func() time.Time
	byName  map[string]computeFn
}

// NewHandler constructs the Connect handler. now() anchors window resolution;
// nil means time.Now.
func NewHandler(c Counter, plans PlanRefStore, now func() time.Time) *Handler {
	if now == nil {
		now = time.Now
	}
	byName := map[string]computeFn{}
	for _, s := range specs() {
		if s.compute != nil {
			byName[s.decl.Name] = s.compute
		}
	}
	return &Handler{counter: c, plans: plans, now: now, byName: byName}
}

// count is the shared body for every Count* RPC: resolve the window, run the
// named measure's compute, return the scalar.
func (h *Handler) count(ctx context.Context, name string, tw *measuresv1.TimeWindow) (int64, error) {
	rng, err := resolveProtoWindow(tw, h.now())
	if err != nil {
		return 0, connect.NewError(connect.CodeInvalidArgument, err)
	}
	val, _, err := h.byName[name](ctx, h.counter, rng.From, rng.To)
	if err != nil {
		return 0, connect.NewError(connect.CodeInternal, err)
	}
	return val, nil
}

func (h *Handler) CountBacklogCompleted(ctx context.Context, req *connect.Request[smmeasuresv1.CountBacklogCompletedRequest]) (*connect.Response[smmeasuresv1.CountBacklogCompletedResponse], error) {
	n, err := h.count(ctx, MeasureBacklogCompleted, req.Msg.GetWindow())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&smmeasuresv1.CountBacklogCompletedResponse{Count: n}), nil
}

func (h *Handler) CountBacklogCreated(ctx context.Context, req *connect.Request[smmeasuresv1.CountBacklogCreatedRequest]) (*connect.Response[smmeasuresv1.CountBacklogCreatedResponse], error) {
	n, err := h.count(ctx, MeasureBacklogCreated, req.Msg.GetWindow())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&smmeasuresv1.CountBacklogCreatedResponse{Count: n}), nil
}

func (h *Handler) CountBacklogNetDelta(ctx context.Context, req *connect.Request[smmeasuresv1.CountBacklogNetDeltaRequest]) (*connect.Response[smmeasuresv1.CountBacklogNetDeltaResponse], error) {
	n, err := h.count(ctx, MeasureBacklogNetDelta, req.Msg.GetWindow())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&smmeasuresv1.CountBacklogNetDeltaResponse{Count: n}), nil
}

func (h *Handler) CountExecutionsCompleted(ctx context.Context, req *connect.Request[smmeasuresv1.CountExecutionsCompletedRequest]) (*connect.Response[smmeasuresv1.CountExecutionsCompletedResponse], error) {
	n, err := h.count(ctx, MeasureExecutionCompleted, req.Msg.GetWindow())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&smmeasuresv1.CountExecutionsCompletedResponse{Count: n}), nil
}

func (h *Handler) CountGoalsCreated(ctx context.Context, req *connect.Request[smmeasuresv1.CountGoalsCreatedRequest]) (*connect.Response[smmeasuresv1.CountGoalsCreatedResponse], error) {
	n, err := h.count(ctx, MeasureGoalCreated, req.Msg.GetWindow())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&smmeasuresv1.CountGoalsCreatedResponse{Count: n}), nil
}

func (h *Handler) CountAgentSessionsCreated(ctx context.Context, req *connect.Request[smmeasuresv1.CountAgentSessionsCreatedRequest]) (*connect.Response[smmeasuresv1.CountAgentSessionsCreatedResponse], error) {
	n, err := h.count(ctx, MeasureAgentSessionCreated, req.Msg.GetWindow())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&smmeasuresv1.CountAgentSessionsCreatedResponse{Count: n}), nil
}

func (h *Handler) CountPlanRefs(_ context.Context, _ *connect.Request[smmeasuresv1.CountPlanRefsRequest]) (*connect.Response[smmeasuresv1.CountPlanRefsResponse], error) {
	n, err := countPlanRefs(h.plans)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&smmeasuresv1.CountPlanRefsResponse{Count: n}), nil
}

func (h *Handler) GoalMilestoneHealth(_ context.Context, _ *connect.Request[smmeasuresv1.GoalMilestoneHealthRequest]) (*connect.Response[smmeasuresv1.GoalMilestoneHealthResponse], error) {
	fields, _, err := milestoneHealth(h.plans)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	response := &smmeasuresv1.GoalMilestoneHealthResponse{Milestones: make([]*smmeasuresv1.MilestoneHealth, 0, len(fields))}
	for _, field := range fields {
		parse := func(name string) int64 {
			value, _ := strconv.ParseInt(field[name], 10, 64)
			return value
		}
		response.Milestones = append(response.Milestones, &smmeasuresv1.MilestoneHealth{
			Milestone: field["milestone"], Total: parse("total"), Completed: parse("completed"), InProgress: parse("in_progress"), Blocked: parse("blocked"),
		})
	}
	return connect.NewResponse(response), nil
}

func (h *Handler) AgentSessionProposalRate(ctx context.Context, req *connect.Request[smmeasuresv1.AgentSessionProposalRateRequest]) (*connect.Response[smmeasuresv1.AgentSessionProposalRateResponse], error) {
	rng, err := resolveProtoWindow(req.Msg.GetWindow(), h.now())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	rate, sample, _, err := proposalApplyRate(ctx, h.counter, rng.From, rng.To)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&smmeasuresv1.AgentSessionProposalRateResponse{Rate: rate, SampleSize: sample}), nil
}

func (h *Handler) ExecutionSuccessRate(ctx context.Context, req *connect.Request[smmeasuresv1.ExecutionSuccessRateRequest]) (*connect.Response[smmeasuresv1.ExecutionSuccessRateResponse], error) {
	rng, err := resolveProtoWindow(req.Msg.GetWindow(), h.now())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	rate, sample, _, err := executionSuccessRate(ctx, h.counter, rng.From, rng.To)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&smmeasuresv1.ExecutionSuccessRateResponse{Rate: rate, SampleSize: sample}), nil
}

func (h *Handler) BacklogLeadTime(ctx context.Context, req *connect.Request[smmeasuresv1.BacklogLeadTimeRequest]) (*connect.Response[smmeasuresv1.BacklogLeadTimeResponse], error) {
	rng, err := resolveProtoWindow(req.Msg.GetWindow(), h.now())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	summary, _, err := backlogLeadTime(ctx, h.counter, rng.From, rng.To)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&smmeasuresv1.BacklogLeadTimeResponse{AverageHours: summary.Average, MedianHours: summary.Median, SampleSize: summary.SampleSize}), nil
}

func (h *Handler) ExecutionDuration(ctx context.Context, req *connect.Request[smmeasuresv1.ExecutionDurationRequest]) (*connect.Response[smmeasuresv1.ExecutionDurationResponse], error) {
	rng, err := resolveProtoWindow(req.Msg.GetWindow(), h.now())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	summary, _, err := executionDuration(ctx, h.counter, rng.From, rng.To)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&smmeasuresv1.ExecutionDurationResponse{AverageMinutes: summary.Average, MedianMinutes: summary.Median, SampleSize: summary.SampleSize}), nil
}

func (h *Handler) ExecutionReviewRate(ctx context.Context, req *connect.Request[smmeasuresv1.ExecutionReviewRateRequest]) (*connect.Response[smmeasuresv1.ExecutionReviewRateResponse], error) {
	rng, err := resolveProtoWindow(req.Msg.GetWindow(), h.now())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	rate, sample, _, err := executionReviewRate(ctx, h.counter, rng.From, rng.To)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&smmeasuresv1.ExecutionReviewRateResponse{Rate: rate, SampleSize: sample}), nil
}

func (h *Handler) CountBacklogOpen(_ context.Context, _ *connect.Request[smmeasuresv1.CountBacklogOpenRequest]) (*connect.Response[smmeasuresv1.CountBacklogOpenResponse], error) {
	n, _, err := countGauge(MeasureBacklogOpen, h.plans)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&smmeasuresv1.CountBacklogOpenResponse{Count: n}), nil
}

func (h *Handler) CountBacklogBlocked(_ context.Context, _ *connect.Request[smmeasuresv1.CountBacklogBlockedRequest]) (*connect.Response[smmeasuresv1.CountBacklogBlockedResponse], error) {
	n, _, err := countGauge(MeasureBacklogBlocked, h.plans)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&smmeasuresv1.CountBacklogBlockedResponse{Count: n}), nil
}

func countGauge(name string, plans PlanRefStore) (int64, string, error) {
	switch name {
	case MeasurePlanRefCount:
		n, err := countPlanRefs(plans)
		return n, "count backlog items with non-null plan_ref from durable backlog projection", err
	case MeasureBacklogOpen:
		n, err := countOpenBacklog(plans)
		return n, "count non-archived, non-terminal backlog items from durable backlog projection", err
	case MeasureBacklogBlocked:
		n, err := countBlockedBacklog(plans)
		return n, "count backlog items blocked by unresolved dependencies from durable backlog projection", err
	default:
		return 0, "", fmt.Errorf("unknown durable measure gauge %q", name)
	}
}

func countPlanRefs(plans PlanRefStore) (int64, error) {
	if plans == nil {
		return 0, fmt.Errorf("plan reference measure requires backlog store")
	}
	items, err := plans.LoadAll(nil)
	if err != nil {
		return 0, err
	}
	var count int64
	for _, item := range items {
		if item.PlanRef != nil {
			count++
		}
	}
	return count, nil
}

func countOpenBacklog(plans PlanRefStore) (int64, error) {
	if plans == nil {
		return 0, fmt.Errorf("open backlog measure requires backlog store")
	}
	items, err := plans.LoadAll(nil)
	if err != nil {
		return 0, err
	}
	var count int64
	for _, item := range items {
		if !backlog.IsArchived(item) && !backlog.IsTerminalStatus(item.Status) {
			count++
		}
	}
	return count, nil
}

func countBlockedBacklog(plans PlanRefStore) (int64, error) {
	if plans == nil {
		return 0, fmt.Errorf("blocked backlog measure requires backlog store")
	}
	items, err := plans.LoadAll(nil)
	if err != nil {
		return 0, err
	}
	return int64(len(backlog.ComputeListBlockingInfo(items))), nil
}

type milestoneHealthSummary struct {
	Name       string
	Total      int64
	Completed  int64
	InProgress int64
	Blocked    int64
}

func milestoneHealth(plans PlanRefStore) ([]map[string]string, string, error) {
	if plans == nil {
		return nil, "", fmt.Errorf("milestone health measure requires backlog store")
	}
	items, err := plans.LoadAll(nil)
	if err != nil {
		return nil, "", err
	}
	blockedByRef := make(map[string]bool)
	for ref, info := range backlog.ComputeListBlockingInfo(items) {
		blockedByRef[ref] = info.Blocked
	}
	byMilestone := make(map[string]*milestoneHealthSummary)
	for _, item := range items {
		if item.Milestone == "" || backlog.IsArchived(item) {
			continue
		}
		summary := byMilestone[item.Milestone]
		if summary == nil {
			summary = &milestoneHealthSummary{Name: item.Milestone}
			byMilestone[item.Milestone] = summary
		}
		summary.Total++
		if backlog.IsTerminalStatus(item.Status) {
			summary.Completed++
		}
		if backlog.IsInFlightStatus(item.Status) {
			summary.InProgress++
		}
		if blockedByRef[string(item.Kind)+"/"+item.Name] {
			summary.Blocked++
		}
	}
	names := make([]string, 0, len(byMilestone))
	for name := range byMilestone {
		names = append(names, name)
	}
	sort.Strings(names)
	fields := make([]map[string]string, 0, len(names))
	for _, name := range names {
		summary := byMilestone[name]
		fields = append(fields, map[string]string{
			"milestone":   summary.Name,
			"total":       strconv.FormatInt(summary.Total, 10),
			"completed":   strconv.FormatInt(summary.Completed, 10),
			"in_progress": strconv.FormatInt(summary.InProgress, 10),
			"blocked":     strconv.FormatInt(summary.Blocked, 10),
		})
	}
	return fields, "group non-archived durable backlog items by milestone and count terminal, in-flight, and dependency-blocked items", nil
}

// RegisterRoutes mounts the Connect MeasuresService on the given mux router.
func RegisterRoutes(router *mux.Router, c Counter, plans PlanRefStore, now func() time.Time) {
	path, handler := smmeasuresconnect.NewMeasuresServiceHandler(NewHandler(c, plans, now))
	connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: handler})
}
