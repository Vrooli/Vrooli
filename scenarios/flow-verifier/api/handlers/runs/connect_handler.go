package runs

import (
	"context"
	"log"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	"flow-verifier/internal/runs"

	runsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/runs"
)

// Deps wires the seams the runs Connect handler needs.
type Deps struct {
	Service *runs.Service
	Logger  *log.Logger
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

func (h *connectHandler) ListRuns(ctx context.Context, req *connect.Request[runsv1.ListRunsRequest]) (*connect.Response[runsv1.ListRunsResponse], error) {
	q := runs.ListQuery{FlowID: req.Msg.GetFlowId(), Limit: int(req.Msg.GetLimit())}
	rows, err := h.deps.Service.List(ctx, q)
	if err != nil {
		h.deps.Logger.Printf("runs.ListRuns: %v", err)
		return nil, runs.ToConnectError(err)
	}
	out := &runsv1.ListRunsResponse{Runs: make([]*runsv1.Run, 0, len(rows))}
	for _, r := range rows {
		out.Runs = append(out.Runs, RunToProto(r))
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) GetRun(ctx context.Context, req *connect.Request[runsv1.GetRunRequest]) (*connect.Response[runsv1.GetRunResponse], error) {
	row, err := h.deps.Service.Get(ctx, req.Msg.GetId())
	if err != nil {
		connectErr := runs.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("runs.GetRun(%q): %v", req.Msg.GetId(), err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(&runsv1.GetRunResponse{Run: RunToProto(row)}), nil
}

// RunToProto is exported because the verifications handler also
// serialises runs.Run rows into its StartVerificationResponse.
func RunToProto(r runs.Run) *runsv1.Run {
	missing := append([]string(nil), r.MissingArtifacts...)
	return &runsv1.Run{
		Id:               r.ID,
		FlowId:           r.FlowID,
		FlowPath:         r.FlowPath,
		Root:             r.Root,
		SourceSha256:     r.SourceSHA256,
		ModelSha256:      r.ModelSHA256,
		GenSha256:        r.GenSHA256,
		Mode:             modeToProto(r.Mode),
		Status:           statusToProto(r.Status),
		Counterexample:   r.Counterexample,
		ErrorMessage:     r.ErrorMessage,
		FailureReason:    failureReasonToProto(r.FailureReason),
		MissingArtifacts: missing,
		Output:           r.Output,
		StartedAt:        timeToProto(r.StartedAt),
		FinishedAt:       timeToProto(r.FinishedAt),
		DurationMs:       r.DurationMs,
	}
}

func timeToProto(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}
	return timestamppb.New(t)
}

func modeToProto(m runs.Mode) runsv1.RunMode {
	switch m {
	case runs.ModeRun:
		return runsv1.RunMode_RUN_MODE_RUN
	case runs.ModeCheck:
		return runsv1.RunMode_RUN_MODE_CHECK
	}
	return runsv1.RunMode_RUN_MODE_UNSPECIFIED
}

func statusToProto(s runs.Status) runsv1.RunStatus {
	switch s {
	case runs.StatusPassed:
		return runsv1.RunStatus_RUN_STATUS_PASSED
	case runs.StatusFailed:
		return runsv1.RunStatus_RUN_STATUS_FAILED
	case runs.StatusError:
		return runsv1.RunStatus_RUN_STATUS_ERROR
	}
	return runsv1.RunStatus_RUN_STATUS_UNSPECIFIED
}

func failureReasonToProto(s string) runsv1.FailureReason {
	switch s {
	case "missing_artifacts":
		return runsv1.FailureReason_FAILURE_REASON_MISSING_ARTIFACTS
	case "stale_artifacts":
		return runsv1.FailureReason_FAILURE_REASON_STALE_ARTIFACTS
	case "counterexample":
		return runsv1.FailureReason_FAILURE_REASON_COUNTEREXAMPLE
	case "lint":
		return runsv1.FailureReason_FAILURE_REASON_LINT
	case "quint_failure":
		return runsv1.FailureReason_FAILURE_REASON_QUINT_FAILURE
	case "io":
		return runsv1.FailureReason_FAILURE_REASON_IO
	}
	return runsv1.FailureReason_FAILURE_REASON_UNSPECIFIED
}
