// Package usage hosts the UsageService Connect-RPC handler.
package usage

import (
	"context"
	"errors"
	"log"
	"time"

	"audio-tools/internal/module"
	"audio-tools/internal/store"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	usagev1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/usage"
	usageconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/usage/usage_v1connect"
)

type Deps struct {
	Logger *log.Logger
	Store  *store.UsageStore
}

type connectHandler struct{ deps Deps }

func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

var Endpoints = []module.EndpointDescriptor{
	{ID: "usage.list_recent", Path: "/vrooli.audio_tools.v1.usage.UsageService/ListRecent", Method: "POST", Category: "usage"},
	{ID: "usage.get_summary", Path: "/vrooli.audio_tools.v1.usage.UsageService/GetSummary", Method: "POST", Category: "usage"},
}

func Module(d Deps) module.Module {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	connectPath, h := usageconnect.NewUsageServiceHandler(NewConnectHandler(d))
	return module.Module{
		Name: "usage",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: h})
		},
		Endpoints: Endpoints,
	}
}

func Schema() string { return "" }

func (h *connectHandler) ListRecent(ctx context.Context, req *connect.Request[usagev1.ListRecentRequest]) (*connect.Response[usagev1.ListRecentResponse], error) {
	if h.deps.Store == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("usage store not configured"))
	}
	m := req.Msg
	since := resolveSince(m.GetSinceSeconds())
	rows, err := h.deps.Store.ListRecent(ctx, since, int(m.GetLimit()), m.GetCapability(), m.GetProviderTier())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &usagev1.ListRecentResponse{Rows: make([]*usagev1.UsageRow, 0, len(rows))}
	for _, r := range rows {
		out.Rows = append(out.Rows, rowToProto(r))
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) GetSummary(ctx context.Context, req *connect.Request[usagev1.GetSummaryRequest]) (*connect.Response[usagev1.GetSummaryResponse], error) {
	if h.deps.Store == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("usage store not configured"))
	}
	since := resolveSince(req.Msg.GetSinceSeconds())
	sum, err := h.deps.Store.Summary(ctx, since, req.Msg.GetCapability())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &usagev1.Summary{
		Since:           sum.Since.UTC().Format(time.RFC3339),
		Until:           sum.Until.UTC().Format(time.RFC3339),
		OperationsTotal: sum.OperationsTotal,
		CreditsTotal:    sum.CreditsTotal,
		ErrorCount:      sum.ErrorCount,
	}
	for _, d := range sum.Distribution {
		out.Distribution = append(out.Distribution, &usagev1.ProviderDistribution{
			ProviderTier: d.ProviderTier, ProviderId: d.ProviderID,
			Count: d.Count, CreditsTotal: d.CreditsTotal, AvgLatencyMs: d.AvgLatencyMs,
		})
	}
	for _, f := range sum.FallbackReasons {
		out.FallbackReasons = append(out.FallbackReasons, &usagev1.FallbackReason{Reason: f.Reason, Count: f.Count})
	}
	return connect.NewResponse(&usagev1.GetSummaryResponse{Summary: out}), nil
}

func resolveSince(sinceSeconds int64) time.Time {
	if sinceSeconds <= 0 {
		sinceSeconds = 86400
	}
	return time.Now().UTC().Add(-time.Duration(sinceSeconds) * time.Second)
}

func rowToProto(r store.UsageRow) *usagev1.UsageRow {
	return &usagev1.UsageRow{
		OperationId:          r.OperationID,
		EmittedAt:            r.EmittedAt.UTC().Format(time.RFC3339Nano),
		Capability:           r.Capability,
		Operation:            r.Operation,
		ProviderTier:         r.ProviderTier,
		ProviderId:           r.ProviderID,
		ModelId:              r.ModelID,
		LatencyMs:            r.LatencyMs,
		CreditsCharged:       r.CreditsCharged,
		PromptTokens:         r.PromptTokens,
		OutputTokens:         r.OutputTokens,
		AudioDurationSeconds: r.AudioDurationSeconds,
		Error:                r.Error,
		FallbackReason:       r.FallbackReason,
		UserIdentity:         r.UserIdentity,
	}
}
