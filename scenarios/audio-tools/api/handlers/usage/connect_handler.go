package usage

import (
	"context"
	"errors"
	"log"

	"connectrpc.com/connect"

	"audio-tools/internal/clock"
	"audio-tools/internal/protomap"

	usagev1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/usage"
)

type connectHandler struct{ deps Deps }

func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	if d.Clock == nil {
		d.Clock = clock.System{}
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) ListRecent(ctx context.Context, req *connect.Request[usagev1.ListRecentRequest]) (*connect.Response[usagev1.ListRecentResponse], error) {
	if h.deps.Store == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("usage store not configured"))
	}
	m := req.Msg
	since := resolveSince(h.deps.Clock, m.GetSinceSeconds())
	rows, err := h.deps.Store.ListRecent(ctx, since, int(m.GetLimit()), m.GetCapability(), protomap.ProviderTierFromProto(m.GetProviderTier()))
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
	since := resolveSince(h.deps.Clock, req.Msg.GetSinceSeconds())
	sum, err := h.deps.Store.Summary(ctx, since, req.Msg.GetCapability())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &usagev1.Summary{
		Since:           protomap.TimeToProto(sum.Since),
		Until:           protomap.TimeToProto(sum.Until),
		OperationsTotal: sum.OperationsTotal,
		CreditsTotal:    sum.CreditsTotal,
		ErrorCount:      sum.ErrorCount,
	}
	for _, d := range sum.Distribution {
		out.Distribution = append(out.Distribution, &usagev1.ProviderDistribution{
			ProviderTier: protomap.ProviderTierToProto(d.ProviderTier),
			ProviderId:   d.ProviderID,
			Count:        d.Count,
			CreditsTotal: d.CreditsTotal,
			AvgLatencyMs: d.AvgLatencyMs,
		})
	}
	for _, f := range sum.FallbackReasons {
		out.FallbackReasons = append(out.FallbackReasons, &usagev1.FallbackReason{Reason: f.Reason, Count: f.Count})
	}
	return connect.NewResponse(&usagev1.GetSummaryResponse{Summary: out}), nil
}
