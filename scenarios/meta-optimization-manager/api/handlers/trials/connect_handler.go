package trials

import (
	"context"
	"log"

	internaltrials "meta-optimization-manager/internal/trials"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	trialsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/meta-optimization-manager/v1/trials"
)

// Deps wires the seams the Connect trials handler needs.
type Deps struct {
	Service internaltrials.Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler constructs the TrialsService Connect handler.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) ListTrialTasks(ctx context.Context, req *connect.Request[trialsv1.ListTrialTasksRequest]) (*connect.Response[trialsv1.ListTrialTasksResponse], error) {
	tasks, err := h.deps.Service.ListTrialTasks(ctx, req.Msg.GetSuite())
	if err != nil {
		h.deps.Logger.Printf("trials.ListTrialTasks: %v", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp := &trialsv1.ListTrialTasksResponse{Tasks: make([]*trialsv1.TrialTask, 0, len(tasks))}
	for _, t := range tasks {
		resp.Tasks = append(resp.Tasks, taskToProto(t))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) RunTrials(ctx context.Context, req *connect.Request[trialsv1.RunTrialsRequest]) (*connect.Response[trialsv1.RunTrialsResponse], error) {
	runs, err := h.deps.Service.RunTrials(ctx, req.Msg.GetSuite(), req.Msg.GetTaskId(), req.Msg.GetModel())
	if err != nil {
		h.deps.Logger.Printf("trials.RunTrials: %v", err)
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	resp := &trialsv1.RunTrialsResponse{Runs: make([]*trialsv1.TrialRun, 0, len(runs))}
	for _, r := range runs {
		resp.Runs = append(resp.Runs, runToProto(r))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) GetTrialHistory(ctx context.Context, req *connect.Request[trialsv1.GetTrialHistoryRequest]) (*connect.Response[trialsv1.GetTrialHistoryResponse], error) {
	hist, err := h.deps.Service.GetTrialHistory(ctx, req.Msg.GetTaskId(), req.Msg.GetSuite())
	if err != nil {
		h.deps.Logger.Printf("trials.GetTrialHistory: %v", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp := &trialsv1.GetTrialHistoryResponse{
		Points:     make([]*trialsv1.TrialHistoryPoint, 0, len(hist.Points)),
		RecentRuns: make([]*trialsv1.TrialRun, 0, len(hist.RecentRuns)),
	}
	for _, p := range hist.Points {
		resp.Points = append(resp.Points, &trialsv1.TrialHistoryPoint{
			At:               timestamppb.New(p.At),
			SuccessRate:      p.SuccessRate,
			MedianTokens:     p.MedianTokens,
			MedianDurationMs: p.MedianDurationMs,
			RunCount:         int32(p.RunCount),
		})
	}
	for _, r := range hist.RecentRuns {
		resp.RecentRuns = append(resp.RecentRuns, runToProto(r))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) GetTrialRun(ctx context.Context, req *connect.Request[trialsv1.GetTrialRunRequest]) (*connect.Response[trialsv1.GetTrialRunResponse], error) {
	run, err := h.deps.Service.GetTrialRun(ctx, req.Msg.GetId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewResponse(&trialsv1.GetTrialRunResponse{Run: runToProto(run)}), nil
}

func (h *connectHandler) GetGateCoverage(ctx context.Context, _ *connect.Request[trialsv1.GetGateCoverageRequest]) (*connect.Response[trialsv1.GetGateCoverageResponse], error) {
	gc, err := h.deps.Service.GetGateCoverage(ctx)
	if err != nil {
		h.deps.Logger.Printf("trials.GetGateCoverage: %v", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&trialsv1.GetGateCoverageResponse{
		GuideTasksTotal:    int32(gc.GuideTasksTotal),
		GuideTasksWithGate: int32(gc.GuideTasksWithGate),
		GateCoverageRatio:  gc.Ratio,
	}), nil
}

func taskToProto(t internaltrials.TrialTask) *trialsv1.TrialTask {
	return &trialsv1.TrialTask{
		Id:          t.ID,
		Suite:       t.Suite,
		GuideTaskId: t.GuideTaskID,
		Description: t.Description,
		Negative:    t.Negative,
	}
}

func runToProto(r internaltrials.TrialRun) *trialsv1.TrialRun {
	out := &trialsv1.TrialRun{
		Id:             r.ID,
		TaskId:         r.TaskID,
		Suite:          r.Suite,
		Model:          r.Model,
		Verdict:        verdictToProto(r.Verdict),
		Tokens:         r.Tokens,
		DurationMs:     r.DurationMs,
		SandboxDiffRef: r.SandboxDiffRef,
	}
	if !r.At.IsZero() {
		out.At = timestamppb.New(r.At)
	}
	return out
}
