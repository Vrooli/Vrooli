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

	"github.com/vrooli/vrooli/scenarios/template-manager/api/internal/catalog"

	measuresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/measures/v1"
	tmmeasuresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/template-manager/v1/measures"
	tmmeasuresconnect "github.com/vrooli/vrooli/packages/proto/gen/go/template-manager/v1/measures/measures_v1connect"
)

const (
	MeasureOpenDebtCount             = "template.open-debt-count"
	MeasureDeepValidateGreenStreak   = "template.deep-validate-green-streak"
	MeasureFleetStandingDistribution = "template.fleet-standing-distribution"
	MeasureMaxVersionLag             = "template.max-version-lag"
)

type Store interface {
	CountOpenDebt(ctx context.Context, window catalog.MeasureWindow) (int64, error)
	DeepValidateGreenStreak(ctx context.Context, templateID string) (int64, error)
	FleetStandingDistribution(ctx context.Context) ([]catalog.StandingBucket, error)
	MaxVersionLag(ctx context.Context) (int64, error)
}

type computeFn func(context.Context, Store, catalog.MeasureWindow, map[string]string) (gomeasures.MeasureResult, error)

type spec struct {
	decl    gomeasures.MeasureDeclaration
	compute computeFn
}

func specs() []spec {
	return []spec{
		{
			decl: windowScalarDecl(
				MeasureOpenDebtCount,
				"OpenDebtCount",
				"How many inherited template debt entries are open in a time window.",
				[]string{"open template debt this week", "how many template defects are still open", "template inherited debt count"},
				"count",
				"entries",
				"{count} open debt entries ({window})",
				nil,
			),
			compute: openDebtCount,
		},
		{
			decl: windowScalarDecl(
				MeasureDeepValidateGreenStreak,
				"DeepValidateGreenStreak",
				"Consecutive passing deep-validation runs, newest first.",
				[]string{"deep validate green streak for react-vite", "how many deep validations passed in a row", "template deep validation streak"},
				"streak",
				"runs",
				"{streak} consecutive passing deep validations",
				map[string]gomeasures.Param{
					"template_id": {
						Name:       "template_id",
						Type:       gomeasures.ParamTypeEnum,
						Default:    "react-vite",
						EnumValues: []string{"react-vite", "landing-page-react-vite"},
					},
				},
			),
			compute: deepValidateGreenStreak,
		},
		{
			decl:    distributionDecl(),
			compute: fleetStandingDistribution,
		},
		{
			decl: windowScalarDecl(
				MeasureMaxVersionLag,
				"MaxVersionLag",
				"Maximum template version lag across the governed registry.",
				[]string{"max template version lag", "largest template migration lag", "which template is most behind"},
				"lag",
				"versions",
				"{lag} maximum version lag",
				nil,
			),
			compute: maxVersionLag,
		},
	}
}

func windowScalarDecl(name, method, intent string, questions []string, valueField, unit, summary string, extra map[string]gomeasures.Param) gomeasures.MeasureDeclaration {
	params := map[string]gomeasures.Param{
		"window": {Name: "window", Type: gomeasures.ParamTypeTimeWindow, Default: string(gomeasures.TokenThisWeek)},
	}
	for key, value := range extra {
		params[key] = value
	}
	return gomeasures.MeasureDeclaration{
		Name:        name,
		Scenario:    "template-manager",
		Domain:      "template",
		Intent:      intent,
		Questions:   questions,
		Params:      params,
		Result:      gomeasures.Result{Kind: gomeasures.ResultScalar, ValueField: valueField, Unit: unit, SummaryTemplate: summary},
		Effect:      gomeasures.EffectRead,
		RunEligible: true,
		Service:     "vrooli.template_manager.v1.measures.MeasuresService",
		Method:      method,
	}
}

func distributionDecl() gomeasures.MeasureDeclaration {
	return gomeasures.MeasureDeclaration{
		Name:      MeasureFleetStandingDistribution,
		Scenario:  "template-manager",
		Domain:    "template",
		Intent:    "Distribution of template standing buckets across governed templates.",
		Questions: []string{"template fleet standing distribution", "how many templates have drift", "template registry standing buckets"},
		Result: gomeasures.Result{
			Kind:            gomeasures.ResultSeries,
			ValueField:      "count",
			Unit:            "templates",
			SummaryTemplate: "template standing distribution",
		},
		Effect:      gomeasures.EffectRead,
		RunEligible: true,
		Service:     "vrooli.template_manager.v1.measures.MeasuresService",
		Method:      "FleetStandingDistribution",
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
			return nil, fmt.Errorf("register measure %s: %w", s.decl.Name, err)
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

func resolveToken(token string, now time.Time) (catalog.MeasureWindow, error) {
	t := gomeasures.TimeWindowToken(token)
	if t == "" {
		t = gomeasures.TokenThisWeek
	}
	rng, err := gomeasures.ResolveToken(t, now, time.UTC)
	if err != nil {
		return catalog.MeasureWindow{}, err
	}
	return catalog.MeasureWindow{From: rng.From, To: rng.To}, nil
}

func resolveProtoWindow(tw *measuresv1.TimeWindow, now time.Time) (catalog.MeasureWindow, error) {
	if tw == nil || tw.GetWindow() == nil {
		return resolveToken("", now)
	}
	rng, err := gomeasures.ResolveTimeWindow(tw, now, time.UTC)
	if err != nil {
		return catalog.MeasureWindow{}, err
	}
	return catalog.MeasureWindow{From: rng.From, To: rng.To}, nil
}

func openDebtCount(ctx context.Context, store Store, window catalog.MeasureWindow, _ map[string]string) (gomeasures.MeasureResult, error) {
	n, err := store.CountOpenDebt(ctx, window)
	if err != nil {
		return gomeasures.MeasureResult{}, err
	}
	return scalarResult(n, fmt.Sprintf("COUNT open debt_entries with last_seen_at >= %q and < %q", window.From.UTC().Format(time.RFC3339Nano), window.To.UTC().Format(time.RFC3339Nano))), nil
}

func deepValidateGreenStreak(ctx context.Context, store Store, _ catalog.MeasureWindow, params map[string]string) (gomeasures.MeasureResult, error) {
	templateID := params["template_id"]
	n, err := store.DeepValidateGreenStreak(ctx, templateID)
	if err != nil {
		return gomeasures.MeasureResult{}, err
	}
	return scalarResult(n, "newest-first deep validation_runs until first non-green status"), nil
}

func fleetStandingDistribution(ctx context.Context, store Store, _ catalog.MeasureWindow, _ map[string]string) (gomeasures.MeasureResult, error) {
	buckets, err := store.FleetStandingDistribution(ctx)
	if err != nil {
		return gomeasures.MeasureResult{}, err
	}
	fields := make([]map[string]string, 0, len(buckets))
	for _, bucket := range buckets {
		fields = append(fields, map[string]string{
			"standing": bucket.Standing,
			"count":    strconv.FormatInt(bucket.Count, 10),
		})
	}
	return gomeasures.MeasureResult{
		Fields:     fields,
		Provenance: gomeasures.Provenance{ExecutedQuery: "template_records left joined to latest drift_snapshots and open debt_entries"},
	}, nil
}

func maxVersionLag(ctx context.Context, store Store, _ catalog.MeasureWindow, _ map[string]string) (gomeasures.MeasureResult, error) {
	n, err := store.MaxVersionLag(ctx)
	if err != nil {
		return gomeasures.MeasureResult{}, err
	}
	return scalarResult(n, "MAX(template_records.lag_count)"), nil
}

func scalarResult(value int64, query string) gomeasures.MeasureResult {
	return gomeasures.MeasureResult{
		Value:      strconv.FormatInt(value, 10),
		Provenance: gomeasures.Provenance{ExecutedQuery: query},
	}
}

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

func (h *Handler) OpenDebtCount(ctx context.Context, req *connect.Request[tmmeasuresv1.OpenDebtCountRequest]) (*connect.Response[tmmeasuresv1.OpenDebtCountResponse], error) {
	window, err := resolveProtoWindow(req.Msg.GetWindow(), h.now())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	n, err := h.store.CountOpenDebt(ctx, window)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&tmmeasuresv1.OpenDebtCountResponse{Count: n}), nil
}

func (h *Handler) DeepValidateGreenStreak(ctx context.Context, req *connect.Request[tmmeasuresv1.DeepValidateGreenStreakRequest]) (*connect.Response[tmmeasuresv1.DeepValidateGreenStreakResponse], error) {
	n, err := h.store.DeepValidateGreenStreak(ctx, req.Msg.GetTemplateId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&tmmeasuresv1.DeepValidateGreenStreakResponse{Streak: n}), nil
}

func (h *Handler) FleetStandingDistribution(ctx context.Context, _ *connect.Request[tmmeasuresv1.FleetStandingDistributionRequest]) (*connect.Response[tmmeasuresv1.FleetStandingDistributionResponse], error) {
	buckets, err := h.store.FleetStandingDistribution(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := make([]*tmmeasuresv1.FleetStandingBucket, 0, len(buckets))
	for _, bucket := range buckets {
		out = append(out, &tmmeasuresv1.FleetStandingBucket{Standing: bucket.Standing, Count: bucket.Count})
	}
	return connect.NewResponse(&tmmeasuresv1.FleetStandingDistributionResponse{Buckets: out}), nil
}

func (h *Handler) MaxVersionLag(ctx context.Context, _ *connect.Request[tmmeasuresv1.MaxVersionLagRequest]) (*connect.Response[tmmeasuresv1.MaxVersionLagResponse], error) {
	lag, err := h.store.MaxVersionLag(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&tmmeasuresv1.MaxVersionLagResponse{Lag: lag}), nil
}

func RegisterRoutes(router *mux.Router, store Store, now func() time.Time) {
	path, handler := tmmeasuresconnect.NewMeasuresServiceHandler(NewHandler(store, now))
	connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: handler})
}
