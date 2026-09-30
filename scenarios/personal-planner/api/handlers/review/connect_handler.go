package review

import (
	"context"
	"log"

	"connectrpc.com/connect"
	v "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/review"
	r "personal-planner/internal/review"
)

type (
	Deps struct {
		Service r.Service
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

func (h *connectHandler) GetDailySummary(c context.Context, req *connect.Request[v.GetDailySummaryRequest]) (*connect.Response[v.GetDailySummaryResponse], error) {
	x, e := h.deps.Service.Daily(c, req.Msg.LocalDate)
	if e != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, e)
	}
	return connect.NewResponse(&v.GetDailySummaryResponse{Summary: toProto(x)}), nil
}

func (h *connectHandler) GetWeeklySummary(c context.Context, req *connect.Request[v.GetWeeklySummaryRequest]) (*connect.Response[v.GetWeeklySummaryResponse], error) {
	x, e := h.deps.Service.Weekly(c, req.Msg.WeekStartLocalDate)
	if e != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, e)
	}
	return connect.NewResponse(&v.GetWeeklySummaryResponse{Summary: weeklyToProto(x)}), nil
}

func (h *connectHandler) GetReflection(c context.Context, req *connect.Request[v.GetReflectionRequest]) (*connect.Response[v.GetReflectionResponse], error) {
	x, e := h.deps.Service.Reflection(c, req.Msg.LocalDate)
	if e != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, e)
	}
	return connect.NewResponse(&v.GetReflectionResponse{Reflection: reflectionToProto(x)}), nil
}

func (h *connectHandler) SaveReflection(c context.Context, req *connect.Request[v.SaveReflectionRequest]) (*connect.Response[v.SaveReflectionResponse], error) {
	x, e := h.deps.Service.SaveReflection(c, req.Msg.LocalDate, req.Msg.Text)
	if e != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, e)
	}
	return connect.NewResponse(&v.SaveReflectionResponse{Reflection: reflectionToProto(x)}), nil
}
