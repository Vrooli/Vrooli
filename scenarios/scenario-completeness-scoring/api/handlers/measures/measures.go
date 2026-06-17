// Package measures exposes scenario-completeness-scoring's declared analytical
// measures over the persisted score_snapshots table.
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
	gomeasures "github.com/vrooli/measures-go"

	measuresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/measures/v1"
	scsmeasuresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-completeness-scoring/v1/measures"
	scsmeasuresconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-completeness-scoring/v1/measures/measures_v1connect"

	"scenario-completeness-scoring/internal/module"
	"scenario-completeness-scoring/internal/scoring"
)

const (
	MeasureFleetBelowRung   = "scoring.fleet-below-rung"
	MeasureAverageComposite = "scoring.average-composite"
	MeasureScoreSeries      = "scoring.score-series"
)

// Store is the narrow snapshot aggregate seam used by the measures package.
type Store interface {
	CountLatestBelowRung(ctx context.Context, thresholdRank int, window scoring.MeasureWindow) (int64, error)
	AverageLatestComposite(ctx context.Context, window scoring.MeasureWindow) (float64, bool, error)
	FleetScoreSeries(ctx context.Context, window scoring.MeasureWindow) ([]scoring.ScoreSeriesPoint, error)
}

type computeFn func(ctx context.Context, store Store, window scoring.MeasureWindow, params map[string]string) (gomeasures.MeasureResult, error)

type spec struct {
	decl    gomeasures.MeasureDeclaration
	compute computeFn
}

func specs() []spec {
	return []spec{
		{
			decl: windowDecl(
				MeasureFleetBelowRung, "CountFleetBelowRung",
				"How many scenarios are below a maturity rung threshold in a time window.",
				[]string{
					"how many scenarios are below rung R2",
					"count scenarios still below R3 this month",
					"how many scenarios are under R1 in the last 30 days",
				},
				gomeasures.ResultScalar, "count", "scenarios", "{count} scenarios below {rung} ({window})",
				map[string]gomeasures.Param{
					"window": {Name: "window", Type: gomeasures.ParamTypeTimeWindow, Default: string(gomeasures.TokenThisWeek)},
					"rung": {
						Name:       "rung",
						Type:       gomeasures.ParamTypeEnum,
						Default:    "RUNG_THRESHOLD_R2",
						EnumValues: rungEnumValues(),
					},
				},
			),
			compute: countFleetBelowRung,
		},
		{
			decl: windowDecl(
				MeasureAverageComposite, "AverageComposite",
				"Average latest composite score across scenarios scored in a time window.",
				[]string{
					"what is the average scenario completeness score this week",
					"average composite score this month",
					"fleet average completeness in the last 30 days",
				},
				gomeasures.ResultScalar, "average", "score", "{average} average composite score ({window})",
				map[string]gomeasures.Param{
					"window": {Name: "window", Type: gomeasures.ParamTypeTimeWindow, Default: string(gomeasures.TokenThisWeek)},
				},
			),
			compute: averageComposite,
		},
		{
			decl: windowDecl(
				MeasureScoreSeries, "ScoreSeries",
				"Fleet-average scenario completeness score series over a time window.",
				[]string{
					"show scenario completeness score trend this month",
					"fleet score history over the last 30 days",
					"score series for scenario completeness this week",
				},
				gomeasures.ResultSeries, "average", "score", "fleet score series ({window})",
				map[string]gomeasures.Param{
					"window": {Name: "window", Type: gomeasures.ParamTypeTimeWindow, Default: string(gomeasures.TokenThisWeek)},
				},
			),
			compute: scoreSeries,
		},
	}
}

func windowDecl(name, method, intent string, questions []string, kind gomeasures.ResultKind, valueField, unit, summary string, params map[string]gomeasures.Param) gomeasures.MeasureDeclaration {
	return gomeasures.MeasureDeclaration{
		Name:        name,
		Scenario:    "scenario-completeness-scoring",
		Domain:      "scoring",
		Intent:      intent,
		Questions:   questions,
		Params:      params,
		Result:      gomeasures.Result{Kind: kind, ValueField: valueField, Unit: unit, SummaryTemplate: summary},
		Effect:      gomeasures.EffectRead,
		RunEligible: true,
		Service:     "vrooli.scenario_completeness_scoring.v1.measures.MeasuresService",
		Method:      method,
	}
}

func rungEnumValues() []string {
	return []string{
		scsmeasuresv1.RungThreshold_RUNG_THRESHOLD_R0.String(),
		scsmeasuresv1.RungThreshold_RUNG_THRESHOLD_R1.String(),
		scsmeasuresv1.RungThreshold_RUNG_THRESHOLD_R2.String(),
		scsmeasuresv1.RungThreshold_RUNG_THRESHOLD_R3.String(),
		scsmeasuresv1.RungThreshold_RUNG_THRESHOLD_R4.String(),
	}
}

func Declarations() []gomeasures.MeasureDeclaration {
	ss := specs()
	out := make([]gomeasures.MeasureDeclaration, 0, len(ss))
	for _, s := range ss {
		out = append(out, s.decl)
	}
	return out
}

func NewRegistry(store Store, now func() time.Time) (*gomeasures.Registry, error) {
	if now == nil {
		now = time.Now
	}
	reg := gomeasures.NewRegistry(gomeasures.WithClock(now))
	for _, s := range specs() {
		s := s
		err := reg.Register(s.decl, func(ctx context.Context, req gomeasures.MeasureRequest) (gomeasures.MeasureResult, error) {
			window, err := resolveToken(req.Params["window"], now())
			if err != nil {
				return gomeasures.MeasureResult{}, err
			}
			return s.compute(ctx, store, window, req.Params)
		})
		if err != nil {
			return nil, fmt.Errorf("measures: register %s: %w", s.decl.Name, err)
		}
	}
	return reg, nil
}

func MeasuresHandler(store Store, now func() time.Time) (http.Handler, error) {
	reg, err := NewRegistry(store, now)
	if err != nil {
		return nil, err
	}
	return reg.Handler(), nil
}

func resolveToken(token string, now time.Time) (scoring.MeasureWindow, error) {
	t := gomeasures.TimeWindowToken(token)
	if t == "" {
		t = gomeasures.TokenThisWeek
	}
	rng, err := gomeasures.ResolveToken(t, now, time.UTC)
	if err != nil {
		return scoring.MeasureWindow{}, err
	}
	return scoring.MeasureWindow{From: rng.From, To: rng.To}, nil
}

func resolveProtoWindow(tw *measuresv1.TimeWindow, now time.Time) (scoring.MeasureWindow, error) {
	if tw == nil || tw.GetWindow() == nil {
		return resolveToken("", now)
	}
	rng, err := gomeasures.ResolveTimeWindow(tw, now, time.UTC)
	if err != nil {
		return scoring.MeasureWindow{}, err
	}
	return scoring.MeasureWindow{From: rng.From, To: rng.To}, nil
}

func countFleetBelowRung(ctx context.Context, store Store, window scoring.MeasureWindow, params map[string]string) (gomeasures.MeasureResult, error) {
	rung := scsmeasuresv1.RungThreshold_RUNG_THRESHOLD_R2.String()
	if params != nil && params["rung"] != "" {
		rung = params["rung"]
	}
	rank := rungRankFromName(rung)
	n, err := store.CountLatestBelowRung(ctx, rank, window)
	if err != nil {
		return gomeasures.MeasureResult{}, err
	}
	return gomeasures.MeasureResult{
		Value: strconv.FormatInt(n, 10),
		Provenance: gomeasures.Provenance{ExecutedQuery: fmt.Sprintf(
			"latest score_snapshots per scenario where created_at >= %q and created_at < %q and working_rung rank < %d",
			window.From.UTC().Format(time.RFC3339Nano), window.To.UTC().Format(time.RFC3339Nano), rank,
		)},
	}, nil
}

func averageComposite(ctx context.Context, store Store, window scoring.MeasureWindow, _ map[string]string) (gomeasures.MeasureResult, error) {
	avg, ok, err := store.AverageLatestComposite(ctx, window)
	if err != nil {
		return gomeasures.MeasureResult{}, err
	}
	value := "0"
	if ok {
		value = strconv.FormatFloat(avg, 'f', 2, 64)
	}
	return gomeasures.MeasureResult{
		Value: value,
		Provenance: gomeasures.Provenance{ExecutedQuery: fmt.Sprintf(
			"AVG(composite) over latest score_snapshots per scenario where created_at >= %q and created_at < %q",
			window.From.UTC().Format(time.RFC3339Nano), window.To.UTC().Format(time.RFC3339Nano),
		)},
	}, nil
}

func scoreSeries(ctx context.Context, store Store, window scoring.MeasureWindow, _ map[string]string) (gomeasures.MeasureResult, error) {
	points, err := store.FleetScoreSeries(ctx, window)
	if err != nil {
		return gomeasures.MeasureResult{}, err
	}
	fields := make([]map[string]string, 0, len(points))
	for _, point := range points {
		fields = append(fields, map[string]string{
			"bucket":  point.Bucket.UTC().Format("2006-01-02"),
			"average": strconv.FormatFloat(point.Score, 'f', 2, 64),
			"count":   strconv.Itoa(point.Count),
		})
	}
	return gomeasures.MeasureResult{
		Fields: fields,
		Provenance: gomeasures.Provenance{ExecutedQuery: fmt.Sprintf(
			"daily AVG(composite) from score_snapshots where created_at >= %q and created_at < %q group by day",
			window.From.UTC().Format(time.RFC3339Nano), window.To.UTC().Format(time.RFC3339Nano),
		)},
	}, nil
}

func rungRankFromName(name string) int {
	switch name {
	case scsmeasuresv1.RungThreshold_RUNG_THRESHOLD_R0.String():
		return 0
	case scsmeasuresv1.RungThreshold_RUNG_THRESHOLD_R1.String():
		return 1
	case scsmeasuresv1.RungThreshold_RUNG_THRESHOLD_R2.String():
		return 2
	case scsmeasuresv1.RungThreshold_RUNG_THRESHOLD_R3.String():
		return 3
	case scsmeasuresv1.RungThreshold_RUNG_THRESHOLD_R4.String():
		return 4
	default:
		return 2
	}
}

// Handler implements vrooli.scenario_completeness_scoring.v1.measures.MeasuresService.
type Handler struct {
	store Store
	now   func() time.Time
}

func NewHandler(store Store, now func() time.Time) *Handler {
	if now == nil {
		now = time.Now
	}
	return &Handler{store: store, now: now}
}

func (h *Handler) CountFleetBelowRung(ctx context.Context, req *connect.Request[scsmeasuresv1.CountFleetBelowRungRequest]) (*connect.Response[scsmeasuresv1.CountFleetBelowRungResponse], error) {
	window, err := resolveProtoWindow(req.Msg.GetWindow(), h.now())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	n, err := h.store.CountLatestBelowRung(ctx, rungRank(req.Msg.GetRung()), window)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&scsmeasuresv1.CountFleetBelowRungResponse{Count: n}), nil
}

func (h *Handler) AverageComposite(ctx context.Context, req *connect.Request[scsmeasuresv1.AverageCompositeRequest]) (*connect.Response[scsmeasuresv1.AverageCompositeResponse], error) {
	window, err := resolveProtoWindow(req.Msg.GetWindow(), h.now())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	avg, _, err := h.store.AverageLatestComposite(ctx, window)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&scsmeasuresv1.AverageCompositeResponse{Average: avg}), nil
}

func (h *Handler) ScoreSeries(ctx context.Context, req *connect.Request[scsmeasuresv1.ScoreSeriesRequest]) (*connect.Response[scsmeasuresv1.ScoreSeriesResponse], error) {
	window, err := resolveProtoWindow(req.Msg.GetWindow(), h.now())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	points, err := h.store.FleetScoreSeries(ctx, window)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := make([]*scsmeasuresv1.ScoreSeriesPoint, 0, len(points))
	for _, point := range points {
		out = append(out, &scsmeasuresv1.ScoreSeriesPoint{
			Bucket:  point.Bucket.UTC().Format("2006-01-02"),
			Average: point.Score,
			Count:   int64(point.Count),
		})
	}
	return connect.NewResponse(&scsmeasuresv1.ScoreSeriesResponse{Points: out}), nil
}

func rungRank(r scsmeasuresv1.RungThreshold) int {
	if r == scsmeasuresv1.RungThreshold_RUNG_THRESHOLD_UNSPECIFIED {
		r = scsmeasuresv1.RungThreshold_RUNG_THRESHOLD_R2
	}
	return int(r) - 1
}

func RegisterRoutes(router *mux.Router, store Store, now func() time.Time) {
	path, handler := scsmeasuresconnect.NewMeasuresServiceHandler(NewHandler(store, now))
	connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: handler})
}

func Module(store Store, now func() time.Time) module.Module {
	return module.Module{
		Name: "measures",
		Mount: func(r *mux.Router) {
			RegisterRoutes(r, store, now)
			measuresHandler, err := MeasuresHandler(store, now)
			if err != nil {
				panic(err)
			}
			r.PathPrefix("/measures/").Handler(http.StripPrefix("/measures", measuresHandler))
		},
		Endpoints: Endpoints,
	}
}

var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "measures_count_fleet_below_rung",
		Path:        scsmeasuresconnect.MeasuresServiceCountFleetBelowRungProcedure,
		Method:      "POST",
		Summary:     "Count scenarios below a maturity rung",
		Description: "Counts latest score snapshots per scenario inside a time window whose working rung is below the requested threshold.",
		Category:    "measures",
		CLIMapping: &module.CLIMapping{
			Command: "scenario-completeness-scoring scoring fleet-below-rung",
			Args:    []string{"--window", "<window>", "--rung", "<rung>"},
		},
	},
	{
		ID:          "measures_average_composite",
		Path:        scsmeasuresconnect.MeasuresServiceAverageCompositeProcedure,
		Method:      "POST",
		Summary:     "Average fleet composite score",
		Description: "Averages the latest composite score per scenario inside a time window.",
		Category:    "measures",
		CLIMapping: &module.CLIMapping{
			Command: "scenario-completeness-scoring scoring average-composite",
			Args:    []string{"--window", "<window>"},
		},
	},
	{
		ID:          "measures_score_series",
		Path:        scsmeasuresconnect.MeasuresServiceScoreSeriesProcedure,
		Method:      "POST",
		Summary:     "Fleet score series",
		Description: "Returns a daily fleet-average completeness score series over persisted score snapshots.",
		Category:    "measures",
		CLIMapping: &module.CLIMapping{
			Command: "scenario-completeness-scoring scoring score-series",
			Args:    []string{"--window", "<window>"},
		},
	},
	{
		ID:          "measures_declarations",
		Path:        "/measures/declarations",
		Method:      "GET",
		Summary:     "List declared measures",
		Description: "Measures-go registry declarations harvested by measures-health and search-hub.",
		Category:    "measures",
		RESTException: &module.RESTException{
			Reason: module.RESTReasonOpsProbe,
			Note:   "measures-go serves a framework-neutral harvest endpoint consumed without a generated client.",
		},
	},
	{
		ID:          "measures_execute",
		Path:        "/measures/execute",
		Method:      "POST",
		Summary:     "Execute a declared measure",
		Description: "Measures-go registry execution endpoint used by measures-health behavioral probes and search-hub federation.",
		Category:    "measures",
		RESTException: &module.RESTException{
			Reason: module.RESTReasonOpsProbe,
			Note:   "measures-go serves a uniform JSON execution endpoint shared across scenarios.",
		},
	},
}
