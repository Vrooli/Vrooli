// Package measures exposes swarm-manager's declared analytical measures: the
// granular, individually addressable replacement for the monolithic /stats blob.
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
	"fmt"
	"net/http"
	"strconv"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	measures "github.com/vrooli/measures-go"

	measuresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/measures/v1"
	smmeasuresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/measures"
	smmeasuresconnect "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/measures/measures_v1connect"

	"swarm-manager/internal/eventlog"
)

// Counter is the narrow event-log substrate the measures compute against. The
// production *eventlog.SQLiteRepository satisfies it; tests inject a fake.
type Counter interface {
	CountEventsInRange(ctx context.Context, eventType eventlog.EventType, from, to time.Time) (int, error)
	CountStatusTransitionsInRange(ctx context.Context, eventType eventlog.EventType, toStatus string, from, to time.Time) (int, error)
}

// computeFn computes one measure's scalar value over [from, to) and returns the
// value plus the executed-query provenance string. Shared by the serve registry
// and the Connect RPC so a measure and its RPC report identical numbers.
type computeFn func(ctx context.Context, c Counter, from, to time.Time) (int64, string, error)

// spec is one measure: its declaration (the SSOT identity surfaced to the index
// and the probe) plus its compute path.
type spec struct {
	decl    measures.MeasureDeclaration
	compute computeFn
}

// Measure names — the conventional "<domain>.<command>" identifiers. They MUST
// equal the manifest's domain.command so the behavioral probe's /execute call
// resolves them.
const (
	MeasureBacklogCompleted    = "backlog.completed"
	MeasureBacklogCreated      = "backlog.created"
	MeasureExecutionCompleted  = "execution.completed"
	MeasureGoalCreated         = "goal.created"
	MeasureAgentSessionCreated = "agent_session.created"
)

// specs returns the full measure set in a stable order. now() anchors relative
// time-window resolution (injected for deterministic tests).
func specs() []spec {
	return []spec{
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
func NewRegistry(c Counter, now func() time.Time) (*measures.Registry, error) {
	if now == nil {
		now = time.Now
	}
	reg := measures.NewRegistry(measures.WithClock(now))
	for _, s := range specs() {
		s := s
		err := reg.Register(s.decl, func(ctx context.Context, req measures.MeasureRequest) (measures.MeasureResult, error) {
			rng, err := resolveToken(req.Params["window"], now())
			if err != nil {
				return measures.MeasureResult{}, err
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
func MeasuresHandler(c Counter, now func() time.Time) (http.Handler, error) {
	reg, err := NewRegistry(c, now)
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
	now     func() time.Time
	byName  map[string]computeFn
}

// NewHandler constructs the Connect handler. now() anchors window resolution;
// nil means time.Now.
func NewHandler(c Counter, now func() time.Time) *Handler {
	if now == nil {
		now = time.Now
	}
	byName := map[string]computeFn{}
	for _, s := range specs() {
		byName[s.decl.Name] = s.compute
	}
	return &Handler{counter: c, now: now, byName: byName}
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

// RegisterRoutes mounts the Connect MeasuresService on the given mux router.
func RegisterRoutes(router *mux.Router, c Counter, now func() time.Time) {
	path, handler := smmeasuresconnect.NewMeasuresServiceHandler(NewHandler(c, now))
	connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: handler})
}
