// Package measures exposes BAS execution quality measures over persisted
// execution and UX telemetry rows.
package measures

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"connectrpc.com/connect"
	"github.com/go-chi/chi/v5"
	"github.com/vrooli/api-core/connectx"
	coredb "github.com/vrooli/api-core/database"
	measurelib "github.com/vrooli/measures-go"
	basmeasuresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/measures"
	basmeasuresconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/measures/measuresv1connect"
	sharedmeasuresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/measures/v1"
)

const (
	ExecutionSuccessRate = "executions.pass-rate"
	ExecutionDurationP95 = "executions.p95-duration"
	StepFailureRate      = "execution_metrics.step-failure-rate"
	SelectorFailureRate  = "telemetry.selector-failure-rate"
)

// Metrics is the narrow aggregate seam used by both transports.
type Metrics interface {
	Aggregate(context.Context, time.Time, time.Time) (Aggregate, error)
}

type Aggregate struct {
	TerminalExecutions   int64
	SuccessfulExecutions int64
	DurationP95Ms        float64
	StepCount            int64
	FailedSteps          int64
	SelectorTraces       int64
	FailedSelectors      int64
}

type SQLRepository struct{ db *coredb.RoutedDB }

func NewSQLRepository(db *coredb.RoutedDB) *SQLRepository { return &SQLRepository{db: db} }

// aggregateSQL computes all four readings in one database round trip. The
// scalar subqueries remain SQL-side aggregates; no row listing or filtering is
// performed in Go.
const aggregateSQL = `WITH executions_window AS (
  SELECT status, (julianday(completed_at) - julianday(started_at)) * 86400000.0 AS duration_ms
  FROM executions
  WHERE datetime(created_at) >= datetime(?) AND datetime(created_at) < datetime(?)
), metrics_window AS (
  SELECT COALESCE(SUM(step_count), 0) AS step_count,
         COALESCE(SUM(failed_steps), 0) AS failed_steps
  FROM ux_execution_metrics
  WHERE datetime(computed_at) >= datetime(?) AND datetime(computed_at) < datetime(?)
), traces_window AS (
  SELECT COUNT(*) AS trace_count,
         COALESCE(SUM(CASE WHEN success = 0 THEN 1 ELSE 0 END), 0) AS failed_count
  FROM ux_interaction_traces
  WHERE datetime(timestamp) >= datetime(?) AND datetime(timestamp) < datetime(?)
), duration_ranked AS (
  SELECT duration_ms, ROW_NUMBER() OVER (ORDER BY duration_ms) AS rn,
         COUNT(*) OVER () AS total
  FROM executions_window
  WHERE duration_ms IS NOT NULL
)
SELECT
  (SELECT COUNT(*) FROM executions_window WHERE status IN ('completed', 'failed')),
  (SELECT COUNT(*) FROM executions_window WHERE status = 'completed'),
  COALESCE((SELECT duration_ms FROM duration_ranked WHERE rn >= CAST(0.95 * total AS INTEGER) ORDER BY rn LIMIT 1), 0),
  (SELECT step_count FROM metrics_window),
  (SELECT failed_steps FROM metrics_window),
  (SELECT trace_count FROM traces_window),
  (SELECT failed_count FROM traces_window)`

func (r *SQLRepository) Aggregate(ctx context.Context, from, to time.Time) (Aggregate, error) {
	if r == nil || r.db == nil {
		return Aggregate{}, fmt.Errorf("BAS measures repository is not configured")
	}
	lo, hi := from.UTC().Format(time.RFC3339Nano), to.UTC().Format(time.RFC3339Nano)
	var a Aggregate
	err := r.db.QueryRowContext(ctx, aggregateSQL, lo, hi, lo, hi, lo, hi).Scan(
		&a.TerminalExecutions, &a.SuccessfulExecutions, &a.DurationP95Ms,
		&a.StepCount, &a.FailedSteps, &a.SelectorTraces, &a.FailedSelectors,
	)
	return a, err
}

func ratio(n, d int64) float64 {
	if d == 0 {
		return 0
	}
	return float64(n) / float64(d)
}

type spec struct {
	decl  measurelib.MeasureDeclaration
	value func(Aggregate) string
}

func specs() []spec {
	params := func() map[string]measurelib.Param {
		return map[string]measurelib.Param{"window": {Name: "window", Type: measurelib.ParamTypeTimeWindow, Default: string(measurelib.TokenThisWeek)}}
	}
	return []spec{
		{measurelib.MeasureDeclaration{Name: ExecutionSuccessRate, Scenario: "browser-automation-studio", Domain: "executions", Intent: "Fraction of terminal browser executions that completed successfully.", Questions: []string{"what is the browser execution pass rate", "how often do browser executions succeed", "what fraction of executions passed"}, Params: params(), Result: measurelib.Result{Kind: measurelib.ResultScalar, ValueField: "rate", Unit: "ratio", SummaryTemplate: "{rate} execution pass rate ({window})"}, Effect: measurelib.EffectRead, RunEligible: true}, func(a Aggregate) string {
			return strconv.FormatFloat(ratio(a.SuccessfulExecutions, a.TerminalExecutions), 'f', -1, 64)
		}},
		{measurelib.MeasureDeclaration{Name: ExecutionDurationP95, Scenario: "browser-automation-studio", Domain: "executions", Intent: "95th percentile duration of completed browser executions.", Questions: []string{"what is the p95 browser execution duration", "how long do browser executions take", "what is the slowest typical browser execution"}, Params: params(), Result: measurelib.Result{Kind: measurelib.ResultScalar, ValueField: "durationMs", Unit: "milliseconds", SummaryTemplate: "{durationMs} ms p95 execution duration ({window})"}, Effect: measurelib.EffectRead, RunEligible: true}, func(a Aggregate) string { return strconv.FormatFloat(a.DurationP95Ms, 'f', -1, 64) }},
		{measurelib.MeasureDeclaration{Name: StepFailureRate, Scenario: "browser-automation-studio", Domain: "execution_metrics", Intent: "Fraction of recorded execution steps that failed.", Questions: []string{"what is the browser step failure rate", "how many execution steps fail", "are browser workflow steps failing"}, Params: params(), Result: measurelib.Result{Kind: measurelib.ResultScalar, ValueField: "rate", Unit: "ratio", SummaryTemplate: "{rate} step failure rate ({window})"}, Effect: measurelib.EffectRead, RunEligible: true}, func(a Aggregate) string { return strconv.FormatFloat(ratio(a.FailedSteps, a.StepCount), 'f', -1, 64) }},
		{measurelib.MeasureDeclaration{Name: SelectorFailureRate, Scenario: "browser-automation-studio", Domain: "telemetry", Intent: "Fraction of interaction traces whose selector interaction failed.", Questions: []string{"what is the selector failure rate", "how often do browser selectors fail", "are locator interactions failing"}, Params: params(), Result: measurelib.Result{Kind: measurelib.ResultScalar, ValueField: "rate", Unit: "ratio", SummaryTemplate: "{rate} selector failure rate ({window})"}, Effect: measurelib.EffectRead, RunEligible: true}, func(a Aggregate) string {
			return strconv.FormatFloat(ratio(a.FailedSelectors, a.SelectorTraces), 'f', -1, 64)
		}},
	}
}

func resolveWindow(tw *sharedmeasuresv1.TimeWindow, now time.Time) (measurelib.Range, error) {
	if tw == nil || tw.GetWindow() == nil {
		return measurelib.ResolveToken(measurelib.TokenThisWeek, now, time.UTC)
	}
	return measurelib.ResolveTimeWindow(tw, now, time.UTC)
}

func declarationRegistry(m Metrics, now func() time.Time) (*measurelib.Registry, error) {
	if now == nil {
		now = time.Now
	}
	r := measurelib.NewRegistry(measurelib.WithClock(now))
	for _, s := range specs() {
		s := s
		if err := r.Register(s.decl, func(ctx context.Context, req measurelib.MeasureRequest) (measurelib.MeasureResult, error) {
			rng, err := measurelib.ResolveToken(measurelib.TimeWindowToken(req.Params["window"]), now(), time.UTC)
			if err != nil {
				return measurelib.MeasureResult{}, err
			}
			a, err := m.Aggregate(ctx, rng.From, rng.To)
			if err != nil {
				return measurelib.MeasureResult{}, err
			}
			return measurelib.MeasureResult{Value: s.value(a), Provenance: measurelib.Provenance{ExecutedQuery: "BAS execution/metrics/telemetry aggregate over [from,to)"}}, nil
		}); err != nil {
			return nil, err
		}
	}
	return r, nil
}

func Module(db *coredb.RoutedDB, now func() time.Time) (connectx.ServiceMount, error) {
	metrics := NewSQLRepository(db)
	if now == nil {
		now = time.Now
	}
	path, handler := basmeasuresconnect.NewMeasuresServiceHandler(&Handler{metrics: metrics, now: now})
	return connectx.ServiceMount{Path: path, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Connect routes are registered by the caller; this branch is replaced
		// below by the mux mount wrapper when the scenario registers services.
		handler.ServeHTTP(w, r)
	})}, nil
}

// Handler is the typed Connect implementation. The serve registry is exposed
// by Mount, which keeps both transports on the same Metrics seam.
type Handler struct {
	metrics Metrics
	now     func() time.Time
}

func (h *Handler) value(ctx context.Context, name string, tw *sharedmeasuresv1.TimeWindow) (float64, error) {
	rng, err := resolveWindow(tw, h.now())
	if err != nil {
		return 0, connect.NewError(connect.CodeInvalidArgument, err)
	}
	a, err := h.metrics.Aggregate(ctx, rng.From, rng.To)
	if err != nil {
		return 0, connect.NewError(connect.CodeInternal, err)
	}
	for _, s := range specs() {
		if s.decl.Name == name {
			v, e := strconv.ParseFloat(s.value(a), 64)
			return v, e
		}
	}
	return 0, connect.NewError(connect.CodeNotFound, fmt.Errorf("unknown measure %q", name))
}

func (h *Handler) ExecutionSuccessRate(ctx context.Context, req *connect.Request[basmeasuresv1.MeasureRequest]) (*connect.Response[basmeasuresv1.RateResponse], error) {
	v, e := h.value(ctx, ExecutionSuccessRate, req.Msg.GetWindow())
	if e != nil {
		return nil, e
	}
	return connect.NewResponse(&basmeasuresv1.RateResponse{Rate: v}), nil
}
func (h *Handler) ExecutionDurationP95(ctx context.Context, req *connect.Request[basmeasuresv1.MeasureRequest]) (*connect.Response[basmeasuresv1.DurationResponse], error) {
	v, e := h.value(ctx, ExecutionDurationP95, req.Msg.GetWindow())
	if e != nil {
		return nil, e
	}
	return connect.NewResponse(&basmeasuresv1.DurationResponse{DurationMs: v}), nil
}
func (h *Handler) StepFailureRate(ctx context.Context, req *connect.Request[basmeasuresv1.MeasureRequest]) (*connect.Response[basmeasuresv1.RateResponse], error) {
	v, e := h.value(ctx, StepFailureRate, req.Msg.GetWindow())
	if e != nil {
		return nil, e
	}
	return connect.NewResponse(&basmeasuresv1.RateResponse{Rate: v}), nil
}
func (h *Handler) SelectorFailureRate(ctx context.Context, req *connect.Request[basmeasuresv1.MeasureRequest]) (*connect.Response[basmeasuresv1.RateResponse], error) {
	v, e := h.value(ctx, SelectorFailureRate, req.Msg.GetWindow())
	if e != nil {
		return nil, e
	}
	return connect.NewResponse(&basmeasuresv1.RateResponse{Rate: v}), nil
}

func ServeMount(db *coredb.RoutedDB, now func() time.Time) (connectx.ServiceMount, error) {
	if now == nil {
		now = time.Now
	}
	metrics := NewSQLRepository(db)
	path, connectHandler := basmeasuresconnect.NewMeasuresServiceHandler(&Handler{metrics: metrics, now: now})
	return connectx.ServiceMount{Path: path, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { connectHandler.ServeHTTP(w, r) })}, nil
}

// MountHTTP adds the framework-agnostic /measures registry to a chi router.
func MountHTTP(r chi.Router, db *coredb.RoutedDB, now func() time.Time) error {
	if now == nil {
		now = time.Now
	}
	registry, err := declarationRegistry(NewSQLRepository(db), now)
	if err != nil {
		return err
	}
	r.Mount("/measures", http.StripPrefix("/measures", registry.Handler()))
	return nil
}
