package impact

import (
	"context"
	"fmt"
	"log"

	"connectrpc.com/connect"

	internal "proto-health/internal/impact"

	impactv1 "github.com/vrooli/vrooli/packages/proto/gen/go/proto-health/v1/impact"
)

type Reporter interface {
	GetImpact(ctx context.Context, scenario, against string) (*impactv1.ImpactReport, error)
}

type Deps struct {
	Logger   *log.Logger
	Reporter Reporter
}

type connectHandler struct {
	deps Deps
}

func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) GetImpact(ctx context.Context, req *connect.Request[impactv1.GetImpactRequest]) (*connect.Response[impactv1.GetImpactResponse], error) {
	if h.deps.Reporter == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("impact reporter is not wired"))
	}
	report, err := h.deps.Reporter.GetImpact(ctx, req.Msg.GetScenario(), req.Msg.GetAgainst())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&impactv1.GetImpactResponse{Report: report}), nil
}

var _ Reporter = (*internal.Service)(nil)
