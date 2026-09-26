package forecasts

import (
	"context"
	"log"

	"connectrpc.com/connect"
	v "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/forecasts"
	f "personal-planner/internal/forecasts"
)

type (
	Deps struct {
		Service f.Service
		Logger  *log.Logger
	}
	connectHandler struct{ deps Deps }
)

func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) GetForecast(ctx context.Context, req *connect.Request[v.GetForecastRequest]) (*connect.Response[v.GetForecastResponse], error) {
	x, err := h.deps.Service.Get(ctx, req.Msg.LocalDate, req.Msg.Timezone, int(req.Msg.HorizonDays))
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&v.GetForecastResponse{Forecast: toProto(x)}), nil
}

func (h *connectHandler) ListForecastSnapshots(ctx context.Context, req *connect.Request[v.ListForecastSnapshotsRequest]) (*connect.Response[v.ListForecastSnapshotsResponse], error) {
	items, err := h.deps.Service.List(ctx, int(req.Msg.Limit))
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	snapshots := make([]*v.ForecastSnapshotSummary, 0, len(items))
	for _, x := range items {
		snapshots = append(snapshots, &v.ForecastSnapshotSummary{Id: x.ID, GeneratedAt: x.GeneratedAt, InputFingerprint: x.InputFingerprint, HorizonStart: x.HorizonStart, HorizonEnd: x.HorizonEnd, CentralFinish: x.CentralFinish, CautiousFinish: x.CautiousFinish, ResultState: x.ResultState, RiskState: x.RiskState, Explanation: x.Explanation, PreviousSnapshotId: x.PreviousSnapshotID, ChangeExplanation: x.ChangeExplanation})
	}
	return connect.NewResponse(&v.ListForecastSnapshotsResponse{Snapshots: snapshots}), nil
}

func toProto(x f.Forecast) *v.Forecast {
	commitments := make([]*v.CommitmentOutlook, 0, len(x.CommitmentOutlooks))
	for _, c := range x.CommitmentOutlooks {
		commitments = append(commitments, &v.CommitmentOutlook{Id: c.ID, Result: c.Result, PromisedBoundary: c.PromisedBoundary, ForecastFinish: c.ForecastFinish, RiskState: c.RiskState, Explanation: c.Explanation})
	}
	return &v.Forecast{GeneratedAt: x.GeneratedAt, Freshness: x.Freshness, InputFingerprint: x.InputFingerprint, HorizonStart: x.HorizonStart, HorizonEnd: x.HorizonEnd, AlgorithmVersion: x.AlgorithmVersion, CentralFinish: x.CentralFinish, CautiousFinish: x.CautiousFinish, ResultState: x.ResultState, RiskState: x.RiskState, Explanation: x.Explanation, KnownWorkMinutes: x.KnownWorkMinutes, AvailableMinutes: x.AvailableMinutes, ReserveMinutes: x.ReserveMinutes, UnresolvedWorkCount: x.UnresolvedWorkCount, CommitmentOutlooks: commitments, SnapshotId: x.SnapshotID, PreviousSnapshotId: x.PreviousSnapshotID, ChangeExplanation: x.ChangeExplanation}
}
