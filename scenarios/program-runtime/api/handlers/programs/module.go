package programs

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	programsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs"
	programsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs/programs_v1connect"
	"program-runtime/internal/budgets"
	"program-runtime/internal/module"
	internalprograms "program-runtime/internal/programs"
)

type handler struct {
	programsconnect.UnimplementedProgramServiceHandler
	service   *internalprograms.Service
	authoring internalprograms.AuthoringDeps
}

func Module(service *internalprograms.Service, authoring internalprograms.AuthoringDeps) module.Module {
	return module.Module{Name: "programs", Mount: func(r *mux.Router) {
		path, h := programsconnect.NewProgramServiceHandler(&handler{service: service, authoring: authoring})
		connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: h})
	}, Endpoints: Endpoints}
}

func (h *handler) SubmitProgram(ctx context.Context, req *connect.Request[programsv1.SubmitProgramRequest]) (*connect.Response[programsv1.SubmitProgramResponse], error) {
	p, diagnostics, err := h.service.SubmitWithDiagnostics(ctx, req.Msg.SessionId, req.Msg.Source, req.Msg.Provenance, req.Msg.IncludeMaterialized, req.Msg.Explain, req.Msg.Async)
	if err != nil {
		// A synchronous submission that outran its bound is DeadlineExceeded,
		// not InvalidArgument: the request was well-formed and the program is
		// still running. The accepted program travels with the error so the
		// caller has an id to wait on rather than a lost submission.
		var deadline *internalprograms.SyncDeadlineExceededError
		if errors.As(err, &deadline) {
			return nil, connect.NewError(connect.CodeDeadlineExceeded, err)
		}
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&programsv1.SubmitProgramResponse{Program: p, Diagnostics: diagnostics}), nil
}

// WaitForProgram blocks server-side until the program is terminal or the
// bounded deadline elapses. It replaces a client-side GetProgram loop that ran
// at twenty requests a second and existed only in the CLI.
func (h *handler) WaitForProgram(ctx context.Context, req *connect.Request[programsv1.WaitForProgramRequest]) (*connect.Response[programsv1.WaitForProgramResponse], error) {
	waited := budgets.BoundWait(time.Duration(req.Msg.GetTimeoutMillis()) * time.Millisecond)
	p, terminal, err := h.service.Wait(ctx, req.Msg.GetId(), waited)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewResponse(&programsv1.WaitForProgramResponse{
		Program:      p,
		Terminal:     terminal,
		WaitedMillis: waited.Milliseconds(),
	}), nil
}

func (h *handler) GetProgram(ctx context.Context, req *connect.Request[programsv1.GetProgramRequest]) (*connect.Response[programsv1.GetProgramResponse], error) {
	p, err := h.service.Get(ctx, req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewResponse(&programsv1.GetProgramResponse{Program: p}), nil
}

func (h *handler) ListPrograms(ctx context.Context, req *connect.Request[programsv1.ListProgramsRequest]) (*connect.Response[programsv1.ListProgramsResponse], error) {
	return connect.NewResponse(&programsv1.ListProgramsResponse{Programs: h.service.List(ctx, req.Msg.SessionId, req.Msg.IncludeOperator)}), nil
}

func (h *handler) MineFailures(ctx context.Context, req *connect.Request[programsv1.MineFailuresRequest]) (*connect.Response[programsv1.MineFailuresResponse], error) {
	shapes := h.service.MineFailures(ctx, req.Msg.IncludeOperator)
	return connect.NewResponse(&programsv1.MineFailuresResponse{Shapes: shapes, Count: int64(len(shapes))}), nil
}

func (h *handler) MineRefusals(ctx context.Context, req *connect.Request[programsv1.MineRefusalsRequest]) (*connect.Response[programsv1.MineRefusalsResponse], error) {
	shapes := h.service.MineRefusals(ctx, req.Msg.IncludeOperator)
	return connect.NewResponse(&programsv1.MineRefusalsResponse{Shapes: shapes, Count: int64(len(shapes))}), nil
}

func (h *handler) MineUnresolvedBindings(ctx context.Context, _ *connect.Request[programsv1.MineUnresolvedBindingsRequest]) (*connect.Response[programsv1.MineUnresolvedBindingsResponse], error) {
	shapes := h.service.MineUnresolvedBindings(ctx)
	return connect.NewResponse(&programsv1.MineUnresolvedBindingsResponse{Shapes: shapes, Count: int64(len(shapes))}), nil
}

// RunAuthoringEval measures first-attempt authoring correctness against the
// versioned corpus.
//
// It reports `unavailable` with a stated reason only when a dependency is
// genuinely missing. It must never return a fixed unavailable response: that is
// indistinguishable from an honest degradation to every caller, and it passes
// the floor gate trivially because a floor comparison is skipped when nothing
// was measured.
func (h *handler) RunAuthoringEval(ctx context.Context, req *connect.Request[programsv1.RunAuthoringEvalRequest]) (*connect.Response[programsv1.RunAuthoringEvalResponse], error) {
	deps := h.authoring
	if suite := req.Msg.GetSuite(); suite != "" {
		deps.SuitePath = suite
	}
	// The eval must carry a deadline, not merely respect one it happens to be
	// given. Go's http.Server write timeout does not create a context deadline,
	// so without this the run's per-case reserve had nothing to measure against:
	// it would author twelve programs at up to three minutes each and be severed
	// mid-write — the `unexpected EOF` this repair exists to remove. Bounded
	// here so the eval always returns a result: `measured` when it finished,
	// `partial` when it stopped early and said how much it skipped.
	ctx, cancel := context.WithTimeout(ctx, budgets.AuthoringEval)
	defer cancel()
	result := internalprograms.RunAuthoringEval(ctx, deps)
	cases := make([]*programsv1.AuthoringCaseResult, 0, len(result.Cases))
	for _, item := range result.Cases {
		cases = append(cases, &programsv1.AuthoringCaseResult{
			CaseId:         item.CaseID,
			Authored:       item.Authored,
			FirstAttemptOk: item.FirstAttempt,
			Cause:          item.Cause,
			AgentBytes:     item.AgentBytes,
			Model:          item.Model,
		})
	}
	return connect.NewResponse(&programsv1.RunAuthoringEvalResponse{
		Suite:        result.Suite,
		Status:       result.Status,
		Reason:       result.Reason,
		Floor:        result.Floor,
		Met:          result.Met,
		Missed:       result.Missed,
		WrongResult:  result.WrongResult,
		Unavailable:  result.Unavailable,
		Cases:        int32(len(result.Cases)),
		Results:      cases,
		NotAttempted: result.NotAttempted,
		HarnessStamp: result.HarnessStamp,
	}), nil
}
