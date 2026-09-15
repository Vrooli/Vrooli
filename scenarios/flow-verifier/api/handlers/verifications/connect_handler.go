package verifications

import (
	"context"
	"errors"
	"log"

	"connectrpc.com/connect"

	runsH "flow-verifier/handlers/runs"
	"flow-verifier/internal/pipeline"
	"flow-verifier/internal/runs"

	runsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/runs"
	verificationsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/verifications"
)

// Deps wires the verifications Connect handler's dependencies.
type Deps struct {
	Runs   *runs.Service
	Logger *log.Logger
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

func (h *connectHandler) StartVerification(ctx context.Context, req *connect.Request[verificationsv1.StartVerificationRequest]) (*connect.Response[verificationsv1.StartVerificationResponse], error) {
	root := req.Msg.GetRoot()
	if root == "" {
		root = "."
	}
	var mode pipeline.Mode
	switch req.Msg.GetMode() {
	case verificationsv1.VerificationMode_VERIFICATION_MODE_GENERATE:
		mode = pipeline.ModeGenerate
	case verificationsv1.VerificationMode_VERIFICATION_MODE_CHECK,
		verificationsv1.VerificationMode_VERIFICATION_MODE_UNSPECIFIED:
		mode = pipeline.ModeCheck
	default:
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid verification mode"))
	}

	rec := &runsRecorder{svc: h.deps.Runs}
	_, runErr := pipeline.Verify(ctx, pipeline.VerifyOptions{
		Root:     root,
		FlowID:   req.Msg.GetFlowId(),
		Mode:     mode,
		Recorder: rec,
	})

	resp := &verificationsv1.StartVerificationResponse{
		Runs: make([]*runsv1.Run, 0, len(rec.captured)),
	}
	for _, row := range rec.captured {
		resp.Runs = append(resp.Runs, runsH.RunToProto(row))
	}
	if runErr != nil {
		resp.Status = "failed"
		resp.ErrorMessage = runErr.Error()
	} else {
		resp.Status = "passed"
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) GetVerification(ctx context.Context, req *connect.Request[verificationsv1.GetVerificationRequest]) (*connect.Response[verificationsv1.GetVerificationResponse], error) {
	row, err := h.deps.Runs.Get(ctx, req.Msg.GetRunId())
	if err != nil {
		connectErr := runs.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("verifications.GetVerification(%q): %v", req.Msg.GetRunId(), err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(&verificationsv1.GetVerificationResponse{Run: runsH.RunToProto(row)}), nil
}

// runsRecorder adapts *runs.Service to pipeline.Recorder and captures
// the inserted rows so the StartVerification response carries them.
type runsRecorder struct {
	svc      *runs.Service
	captured []runs.Run
}

func (r *runsRecorder) Record(ctx context.Context, entry pipeline.RunEntry) error {
	row := runs.Run{
		FlowID:           entry.FlowID,
		FlowPath:         entry.FlowPath,
		Root:             entry.Root,
		Mode:             pipelineModeToRunsMode(entry.Mode),
		Status:           runs.Status(entry.Status),
		Output:           entry.Output,
		ErrorMessage:     entry.ErrorMessage,
		FailureReason:    entry.FailureReason,
		MissingArtifacts: entry.MissingArtifacts,
		StartedAt:        entry.StartedAt,
		FinishedAt:       entry.FinishedAt,
	}
	inserted, err := r.svc.Record(ctx, row)
	if err != nil {
		return err
	}
	r.captured = append(r.captured, inserted)
	return nil
}

func pipelineModeToRunsMode(m pipeline.Mode) runs.Mode {
	if m == pipeline.ModeGenerate {
		return runs.ModeRun
	}
	return runs.ModeCheck
}
