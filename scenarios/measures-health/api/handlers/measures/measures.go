// Package measures exposes measures-health's own declared analytical measures —
// the gold-star dogfood: full-tier measures over its persisted validation_run
// history, surfaced two ways over a SINGLE shared compute path so the two can
// never disagree:
//
//   - the packages/measures-go serve registry, mounted at /measures
//     (GET /measures/declarations + POST /measures/execute) — the contract the
//     behavioral probe and the search-hub central index call;
//   - the Connect-RPC MeasuresService (RegisterRoutes) — the typed CLI/UI surface.
//
// Counts are real SQL aggregates over the validation_runs table
// (internal/runhistory), never a list-and-filter.
//
// Proto: packages/proto/schemas/measures-health/v1/measures/measures.proto
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

	mhmeasuresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/measures-health/v1/measures"
	mhmeasuresconnect "github.com/vrooli/vrooli/packages/proto/gen/go/measures-health/v1/measures/measures_v1connect"
	measuresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/measures/v1"
)

// Counter is the narrow validation_runs substrate the measures compute against.
// The production *runhistory.Repository satisfies it; tests inject a fake.
type Counter interface {
	CountFailed(ctx context.Context, from, to time.Time) (int64, error)
	CountPassing(ctx context.Context, from, to time.Time) (int64, error)
}

// computeFn computes one measure's scalar value over [from, to) and returns the
// value plus the executed-query provenance. Shared by the serve registry and the
// Connect RPC so a measure and its RPC report identical numbers.
type computeFn func(ctx context.Context, c Counter, from, to time.Time) (int64, string, error)

type spec struct {
	decl    measures.MeasureDeclaration
	compute computeFn
}

// Measure names — the conventional "<domain>.<command>" identifiers. They MUST
// equal the manifest's domain.command so the behavioral probe's /execute call
// resolves them.
const (
	MeasureValidationFailed   = "validation_run.failed"
	MeasureValidationCoverage = "validation_run.coverage"
)

// specs returns the full measure set in a stable order.
func specs() []spec {
	return []spec{
		{
			decl: windowDecl(
				MeasureValidationFailed, "validation_run", "CountFailedValidations",
				"How many scenarios failed measures validation in a time window.",
				[]string{
					"how many scenarios failed measures validation this week",
					"measures validation failures last month",
					"how many scenarios failed the measures gate in the last 7 days",
				},
				"scenarios", "{count} scenarios failed measures validation ({window})",
			),
			compute: func(ctx context.Context, c Counter, from, to time.Time) (int64, string, error) {
				n, err := c.CountFailed(ctx, from, to)
				return n, rangeQuery("passed = 0", from, to), err
			},
		},
		{
			decl: windowDecl(
				MeasureValidationCoverage, "validation_run", "CountValidationCoverage",
				"How many scenarios passed measures validation in a time window (fleet measure-coverage over time).",
				[]string{
					"how many scenarios passed measures validation this week",
					"measures coverage this month",
					"how many scenarios had clean measures validation in the last 30 days",
				},
				"scenarios", "{count} scenarios passed measures validation ({window})",
			),
			compute: func(ctx context.Context, c Counter, from, to time.Time) (int64, string, error) {
				n, err := c.CountPassing(ctx, from, to)
				return n, rangeQuery("passed = 1", from, to), err
			},
		},
	}
}

// windowDecl builds a read-only, run-eligible scalar measure with a single
// canonical time_window param defaulting to this_week (→ full tier).
func windowDecl(name, domain, method, intent string, questions []string, unit, summary string) measures.MeasureDeclaration {
	return measures.MeasureDeclaration{
		Name:      name,
		Scenario:  "measures-health",
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

func rangeQuery(cond string, from, to time.Time) string {
	return fmt.Sprintf(
		"SELECT COUNT(*) FROM validation_runs WHERE %s AND ran_at >= %q AND ran_at < %q",
		cond, from.UTC().Format(time.RFC3339), to.UTC().Format(time.RFC3339),
	)
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

func resolveToken(token string, now time.Time) (measures.Range, error) {
	t := measures.TimeWindowToken(token)
	if t == "" {
		t = measures.TokenThisWeek
	}
	return measures.ResolveToken(t, now, time.UTC)
}

func resolveProtoWindow(tw *measuresv1.TimeWindow, now time.Time) (measures.Range, error) {
	if tw == nil || tw.GetWindow() == nil {
		return measures.ResolveToken(measures.TokenThisWeek, now, time.UTC)
	}
	return measures.ResolveTimeWindow(tw, now, time.UTC)
}

// -----------------------------------------------------------------------------
// Connect-RPC MeasuresService — the typed surface, sharing the compute path.
// -----------------------------------------------------------------------------

// Handler implements vrooli.measures_health.v1.measures.MeasuresService.
type Handler struct {
	counter Counter
	now     func() time.Time
	byName  map[string]computeFn
}

// NewHandler constructs the Connect handler. now anchors window resolution; nil = time.Now.
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

func (h *Handler) CountFailedValidations(ctx context.Context, req *connect.Request[mhmeasuresv1.CountFailedValidationsRequest]) (*connect.Response[mhmeasuresv1.CountFailedValidationsResponse], error) {
	n, err := h.count(ctx, MeasureValidationFailed, req.Msg.GetWindow())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&mhmeasuresv1.CountFailedValidationsResponse{Count: n}), nil
}

func (h *Handler) CountValidationCoverage(ctx context.Context, req *connect.Request[mhmeasuresv1.CountValidationCoverageRequest]) (*connect.Response[mhmeasuresv1.CountValidationCoverageResponse], error) {
	n, err := h.count(ctx, MeasureValidationCoverage, req.Msg.GetWindow())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&mhmeasuresv1.CountValidationCoverageResponse{Count: n}), nil
}

// RegisterRoutes mounts the Connect MeasuresService on the given mux router.
func RegisterRoutes(router *mux.Router, c Counter, now func() time.Time) {
	path, handler := mhmeasuresconnect.NewMeasuresServiceHandler(NewHandler(c, now))
	connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: handler})
}
