package stats

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	statsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/stats"
	statsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/stats/stats_v1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// connectService exposes the producer-owned portfolio projection through the
// generated Connect contract. The REST stats endpoint remains available for
// compatibility and for its richer, uncontracted analytical payload.
type connectService struct {
	engine *Engine
}

func (s *connectService) GetPortfolioStats(ctx context.Context, _ *connect.Request[statsv1.GetPortfolioStatsRequest]) (*connect.Response[statsv1.PortfolioStats], error) {
	if s.engine == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("stats engine is unavailable"))
	}
	if err := s.engine.Refresh(ctx); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("refresh stats: %w", err))
	}
	resp, err := s.engine.GetStatsForParams(ctx, Params{})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build portfolio stats: %w", err))
	}
	return connect.NewResponse(&statsv1.PortfolioStats{
		ObservedAt:          timestamppb.New(resp.GeneratedAt),
		SwarmThroughput:     int64(resp.Throughput.CompletedLast7Days),
		ThroughputStats:     int64(resp.Throughput.CreatedLast7Days),
		SwarmActiveAgents:   int64(resp.Agent.TotalExecutions),
		AgentStats:          resp.Agent.SuccessRate,
		TimingStats:         resp.Timing.AvgExecutionMinutes,
		BlockingStats:       int64(resp.Blocking.CurrentlyBlocked),
		DashboardStats:      int64(resp.Dashboard.TotalBacklogSize),
		CompositeThroughput: int64(resp.Dashboard.TotalCompletedAllTime),
		ReviewStats:         int64(resp.Review.RoundsCompleted),
		ScopeStats:          int64(len(resp.Scope.Goals)),
	}), nil
}

// RegisterConnectRoutes registers the stable typed portfolio projection.
func RegisterConnectRoutes(router *mux.Router, engine *Engine) {
	path, handler := statsconnect.NewStatsServiceHandler(&connectService{engine: engine})
	connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: handler})
}
