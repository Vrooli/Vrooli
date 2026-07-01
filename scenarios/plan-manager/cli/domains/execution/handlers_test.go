package execution

import (
	"context"
	"net/http"
	"sync"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	executionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/execution"
	executionconnect "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/execution/execution_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/shared"

	"github.com/vrooli/cli-core/cliapp"
	clitest "plan-manager/cli/internal/testutil"
)

// execRecorder is a fake ExecutionService capturing the request the handler
// built and returning a canned response message (or error).
type execRecorder struct {
	executionconnect.UnimplementedExecutionServiceHandler
	mu   sync.Mutex
	req  proto.Message
	resp proto.Message
	err  error
}

func (r *execRecorder) record(req proto.Message) {
	r.mu.Lock()
	r.req = req
	r.mu.Unlock()
}

func (r *execRecorder) lastRequest() proto.Message {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.req
}

func (r *execRecorder) Start(_ context.Context, req *connect.Request[executionv1.StartRequest]) (*connect.Response[executionv1.StartResponse], error) {
	r.record(req.Msg)
	if r.err != nil {
		return nil, r.err
	}
	if m, ok := r.resp.(*executionv1.StartResponse); ok && m != nil {
		return connect.NewResponse(m), nil
	}
	return connect.NewResponse(&executionv1.StartResponse{Execution: &executionv1.Execution{Id: "exec-1", PlanId: req.Msg.GetPlanId()}}), nil
}

func (r *execRecorder) GetStatus(_ context.Context, req *connect.Request[executionv1.GetStatusRequest]) (*connect.Response[executionv1.GetStatusResponse], error) {
	r.record(req.Msg)
	if r.err != nil {
		return nil, r.err
	}
	if m, ok := r.resp.(*executionv1.GetStatusResponse); ok && m != nil {
		return connect.NewResponse(m), nil
	}
	return connect.NewResponse(&executionv1.GetStatusResponse{Execution: &executionv1.Execution{Id: req.Msg.GetExecutionId()}, Context: &executionv1.PhaseContext{}}), nil
}

func (r *execRecorder) GetContext(_ context.Context, req *connect.Request[executionv1.GetContextRequest]) (*connect.Response[executionv1.GetContextResponse], error) {
	r.record(req.Msg)
	if r.err != nil {
		return nil, r.err
	}
	if m, ok := r.resp.(*executionv1.GetContextResponse); ok && m != nil {
		return connect.NewResponse(m), nil
	}
	return connect.NewResponse(&executionv1.GetContextResponse{Execution: &executionv1.Execution{Id: req.Msg.GetExecutionId()}, Context: &executionv1.PhaseContext{}}), nil
}

func (r *execRecorder) Resume(_ context.Context, req *connect.Request[executionv1.ResumeRequest]) (*connect.Response[executionv1.ResumeResponse], error) {
	r.record(req.Msg)
	if r.err != nil {
		return nil, r.err
	}
	if m, ok := r.resp.(*executionv1.ResumeResponse); ok && m != nil {
		return connect.NewResponse(m), nil
	}
	return connect.NewResponse(&executionv1.ResumeResponse{Execution: &executionv1.Execution{Id: "exec-1", PlanId: req.Msg.GetPlanOrExecution()}, Context: &executionv1.PhaseContext{}}), nil
}

func (r *execRecorder) GetNext(_ context.Context, req *connect.Request[executionv1.GetNextRequest]) (*connect.Response[executionv1.GetNextResponse], error) {
	r.record(req.Msg)
	if r.err != nil {
		return nil, r.err
	}
	if m, ok := r.resp.(*executionv1.GetNextResponse); ok && m != nil {
		return connect.NewResponse(m), nil
	}
	return connect.NewResponse(&executionv1.GetNextResponse{Context: &executionv1.PhaseContext{}}), nil
}

func (r *execRecorder) TransitionPhase(_ context.Context, req *connect.Request[executionv1.TransitionPhaseRequest]) (*connect.Response[executionv1.TransitionPhaseResponse], error) {
	r.record(req.Msg)
	if r.err != nil {
		return nil, r.err
	}
	return connect.NewResponse(&executionv1.TransitionPhaseResponse{
		Execution: &executionv1.Execution{Id: req.Msg.GetExecutionId()},
		Plan:      &sharedv1.Plan{Status: sharedv1.PlanStatus_PLAN_STATUS_ACTIVE},
	}), nil
}

func (r *execRecorder) Complete(_ context.Context, req *connect.Request[executionv1.CompleteRequest]) (*connect.Response[executionv1.CompleteResponse], error) {
	r.record(req.Msg)
	if r.err != nil {
		return nil, r.err
	}
	if m, ok := r.resp.(*executionv1.CompleteResponse); ok && m != nil {
		return connect.NewResponse(m), nil
	}
	return connect.NewResponse(&executionv1.CompleteResponse{Handoff: &sharedv1.Handoff{ExecutionId: req.Msg.GetExecutionId(), Completeness: sharedv1.Completeness_COMPLETENESS_FULL}}), nil
}

func (r *execRecorder) GetHandoff(_ context.Context, req *connect.Request[executionv1.GetHandoffRequest]) (*connect.Response[executionv1.GetHandoffResponse], error) {
	r.record(req.Msg)
	if r.err != nil {
		return nil, r.err
	}
	if m, ok := r.resp.(*executionv1.GetHandoffResponse); ok && m != nil {
		return connect.NewResponse(m), nil
	}
	return connect.NewResponse(&executionv1.GetHandoffResponse{Handoff: &sharedv1.Handoff{ExecutionId: req.Msg.GetExecutionId()}}), nil
}

func (r *execRecorder) GetVelocity(_ context.Context, req *connect.Request[executionv1.GetVelocityRequest]) (*connect.Response[executionv1.GetVelocityResponse], error) {
	r.record(req.Msg)
	if r.err != nil {
		return nil, r.err
	}
	if m, ok := r.resp.(*executionv1.GetVelocityResponse); ok && m != nil {
		return connect.NewResponse(m), nil
	}
	return connect.NewResponse(&executionv1.GetVelocityResponse{}), nil
}

func newExecFixture(t *testing.T, rec *execRecorder) (*cliapp.ScenarioApp, []cliapp.SubcommandGroup) {
	t.Helper()
	mux := http.NewServeMux()
	path, handler := executionconnect.NewExecutionServiceHandler(rec)
	mux.Handle(path, handler)
	app := clitest.NewTestApp(t, mux)
	group, err := Register(app, clitest.ReadManifest(t))
	require.NoError(t, err, "register exec group against manifest")
	return app, []cliapp.SubcommandGroup{group}
}

// TestExecRequestMapping drives every covered exec verb end-to-end. The
// transition/triage status-enum rows and the complete int-parsing rows are the
// silent-failure-prone surfaces.
func TestExecRequestMapping(t *testing.T) {
	tests := []struct {
		name   string
		cmd    string
		argv   []string
		assert func(t *testing.T, req proto.Message)
	}{
		{
			name: "start maps plan positional + run-id flag", cmd: "start",
			argv: []string{"plan-1", "--run-id", "run-42"},
			assert: func(t *testing.T, req proto.Message) {
				m := req.(*executionv1.StartRequest)
				require.Equal(t, "plan-1", m.GetPlanId())
				require.Equal(t, "run-42", m.GetRunId())
			},
		},
		{
			name: "status maps execution positional", cmd: "status",
			argv: []string{"exec-9"},
			assert: func(t *testing.T, req proto.Message) {
				require.Equal(t, "exec-9", req.(*executionv1.GetStatusRequest).GetExecutionId())
			},
		},
		{
			name: "context maps execution positional and phase flag", cmd: "context",
			argv: []string{"exec-9", "--phase", "phase-2"},
			assert: func(t *testing.T, req proto.Message) {
				m := req.(*executionv1.GetContextRequest)
				require.Equal(t, "exec-9", m.GetExecutionId())
				require.Equal(t, "phase-2", m.GetPhaseId())
			},
		},
		{
			name: "resume maps target phase and run-id", cmd: "resume",
			argv: []string{"plan-1", "--phase", "phase-2", "--run-id", "run-99"},
			assert: func(t *testing.T, req proto.Message) {
				m := req.(*executionv1.ResumeRequest)
				require.Equal(t, "plan-1", m.GetPlanOrExecution())
				require.Equal(t, "phase-2", m.GetPhaseId())
				require.Equal(t, "run-99", m.GetRunId())
			},
		},
		{
			name: "next maps execution positional", cmd: "next",
			argv: []string{"exec-9"},
			assert: func(t *testing.T, req proto.Message) {
				require.Equal(t, "exec-9", req.(*executionv1.GetNextRequest).GetExecutionId())
			},
		},
		{
			name: "transition maps status and overrides", cmd: "transition",
			argv: []string{"exec-1", "phase-3", "--status", "done", "--validation-override-reason", "offline validation accepted", "--feedback-override-reason", "feedback reviewed manually"},
			assert: func(t *testing.T, req proto.Message) {
				m := req.(*executionv1.TransitionPhaseRequest)
				require.Equal(t, "exec-1", m.GetExecutionId())
				require.Equal(t, "phase-3", m.GetPhaseId())
				require.Equal(t, sharedv1.PhaseStatus_PHASE_STATUS_DONE, m.GetToStatus())
				require.Equal(t, "offline validation accepted", m.GetValidationOverride().GetReason())
				require.Equal(t, "feedback reviewed manually", m.GetFeedbackOverride().GetReason())
			},
		},
		{
			name: "transition maps status flag to ACTIVE enum", cmd: "transition",
			argv: []string{"exec-1", "phase-3", "--status", "active"},
			assert: func(t *testing.T, req proto.Message) {
				require.Equal(t, sharedv1.PhaseStatus_PHASE_STATUS_ACTIVE, req.(*executionv1.TransitionPhaseRequest).GetToStatus())
			},
		},
		{
			name: "complete parses tokens+iterations ints", cmd: "complete",
			argv: []string{"exec-1", "--tokens", "12345", "--iterations", "7"},
			assert: func(t *testing.T, req proto.Message) {
				m := req.(*executionv1.CompleteRequest)
				require.Equal(t, "exec-1", m.GetExecutionId())
				require.Equal(t, int64(12345), m.GetTokens())
				require.Equal(t, int32(7), m.GetIterations())
			},
		},
		{
			name: "complete with no velocity flags sends zeros", cmd: "complete",
			argv: []string{"exec-1"},
			assert: func(t *testing.T, req proto.Message) {
				m := req.(*executionv1.CompleteRequest)
				require.Equal(t, int64(0), m.GetTokens())
				require.Equal(t, int32(0), m.GetIterations())
			},
		},
		{
			name: "handoff maps execution positional", cmd: "handoff",
			argv: []string{"exec-1"},
			assert: func(t *testing.T, req proto.Message) {
				require.Equal(t, "exec-1", req.(*executionv1.GetHandoffRequest).GetExecutionId())
			},
		},
		{
			name: "velocity maps plan positional", cmd: "velocity",
			argv: []string{"plan-v"},
			assert: func(t *testing.T, req proto.Message) {
				require.Equal(t, "plan-v", req.(*executionv1.GetVelocityRequest).GetPlanId())
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := &execRecorder{}
			app, groups := newExecFixture(t, rec)
			cmd := clitest.FindCommand(t, groups, "exec", tc.cmd)
			_, err := clitest.RunCommand(t, cmd, app, tc.argv...)
			require.NoError(t, err)
			req := rec.lastRequest()
			require.NotNil(t, req, "handler must have issued a request")
			tc.assert(t, req)
		})
	}
}

// TestExecOutputRendering pins the human render for the data + mutation verbs.
func TestExecOutputRendering(t *testing.T) {
	t.Run("start renders execution summary", func(t *testing.T) {
		rec := &execRecorder{resp: &executionv1.StartResponse{Execution: &executionv1.Execution{Id: "exec-1", PlanId: "plan-1", RunId: "run-9"}}}
		app, groups := newExecFixture(t, rec)
		out, err := clitest.RunCommand(t, clitest.FindCommand(t, groups, "exec", "start"), app, "plan-1")
		require.NoError(t, err)
		require.Contains(t, out, "Started execution exec-1 on plan plan-1.")
		require.Contains(t, out, "run id: run-9")
	})

	t.Run("status renders context summary", func(t *testing.T) {
		rec := &execRecorder{resp: &executionv1.GetStatusResponse{
			Execution: &executionv1.Execution{Id: "exec-1", PlanId: "plan-1"},
			Context: &executionv1.PhaseContext{
				Completeness:    sharedv1.Completeness_COMPLETENESS_PARTIAL,
				Staleness:       sharedv1.StalenessTier_STALENESS_TIER_FRESH,
				ResumePhaseId:   "phase-2",
				CurrentPhase:    &sharedv1.Phase{Title: "Build", Status: sharedv1.PhaseStatus_PHASE_STATUS_ACTIVE},
				RequiredReading: []string{"docs/X.md"},
				RelevantContext: []*sharedv1.RelevantContextItem{{
					Kind:        sharedv1.RelevantContextKind_RELEVANT_CONTEXT_KIND_COMMAND,
					Label:       "Recall prior work",
					Reason:      "Recover execution context.",
					Instruction: "Run before editing.",
					Command:     "search-hub query plan-manager --type record,doc",
					Required:    true,
				}},
			},
		}}
		app, groups := newExecFixture(t, rec)
		out, err := clitest.RunCommand(t, clitest.FindCommand(t, groups, "exec", "status"), app, "exec-1")
		require.NoError(t, err)
		require.Contains(t, out, "Execution exec-1 on plan plan-1 (completeness partial).")
		require.Contains(t, out, "Resume point: phase-2; staleness: fresh.")
		require.Contains(t, out, "Current phase: Build (active).")
		require.Contains(t, out, "context[command]: Recall prior work (required)")
		require.Contains(t, out, "command: search-hub query plan-manager --type record,doc")
		require.Contains(t, out, "read: docs/X.md")
	})

	t.Run("context renders setup heading", func(t *testing.T) {
		rec := &execRecorder{resp: &executionv1.GetContextResponse{
			Execution: &executionv1.Execution{Id: "exec-1", PlanId: "plan-1"},
			Context: &executionv1.PhaseContext{
				Completeness:  sharedv1.Completeness_COMPLETENESS_PARTIAL,
				ResumePhaseId: "phase-2",
				CurrentPhase:  &sharedv1.Phase{Title: "Build", Status: sharedv1.PhaseStatus_PHASE_STATUS_TODO},
				RelevantContext: []*sharedv1.RelevantContextItem{{
					Kind:     sharedv1.RelevantContextKind_RELEVANT_CONTEXT_KIND_SKILL,
					Label:    "Load implementation skill",
					Command:  "prompt-manager skill read api-steer",
					Required: true,
				}},
			},
		}}
		app, groups := newExecFixture(t, rec)
		out, err := clitest.RunCommand(t, clitest.FindCommand(t, groups, "exec", "context"), app, "exec-1")
		require.NoError(t, err)
		require.Contains(t, out, "Setup context")
		require.Contains(t, out, "context[skill]: Load implementation skill (required)")
		require.Contains(t, out, "command: prompt-manager skill read api-steer")
	})

	t.Run("resume renders execution and setup", func(t *testing.T) {
		rec := &execRecorder{resp: &executionv1.ResumeResponse{
			Execution: &executionv1.Execution{Id: "exec-1", PlanId: "plan-1", CurrentPhaseId: "phase-2"},
			Context: &executionv1.PhaseContext{
				CurrentPhase: &sharedv1.Phase{Title: "Build", Status: sharedv1.PhaseStatus_PHASE_STATUS_TODO},
				RelevantContext: []*sharedv1.RelevantContextItem{{
					Kind:    sharedv1.RelevantContextKind_RELEVANT_CONTEXT_KIND_SEARCH,
					Label:   "Recall prior work",
					Command: "search-hub query plan-manager --type record,doc",
				}},
			},
		}}
		app, groups := newExecFixture(t, rec)
		out, err := clitest.RunCommand(t, clitest.FindCommand(t, groups, "exec", "resume"), app, "plan-1")
		require.NoError(t, err)
		require.Contains(t, out, "Resumed execution exec-1 on plan plan-1.")
		require.Contains(t, out, "Current phase: phase-2.")
		require.Contains(t, out, "context[search]: Recall prior work")
	})

	t.Run("transition renders mutation", func(t *testing.T) {
		rec := &execRecorder{}
		app, groups := newExecFixture(t, rec)
		out, err := clitest.RunCommand(t, clitest.FindCommand(t, groups, "exec", "transition"), app, "exec-1", "phase-3", "--status", "done")
		require.NoError(t, err)
		require.Contains(t, out, "Transitioned phase phase-3 to done.")
		require.Contains(t, out, "plan status: active")
	})

	t.Run("complete renders completeness + nudges", func(t *testing.T) {
		rec := &execRecorder{resp: &executionv1.CompleteResponse{
			Handoff: &sharedv1.Handoff{ExecutionId: "exec-1", Completeness: sharedv1.Completeness_COMPLETENESS_FULL, ResumePhaseId: ""},
			Nudges:  []*executionv1.CompletionNudge{{Kind: "file_bugs", Message: "open items", Satisfied: false}},
		}}
		app, groups := newExecFixture(t, rec)
		out, err := clitest.RunCommand(t, clitest.FindCommand(t, groups, "exec", "complete"), app, "exec-1")
		require.NoError(t, err)
		require.Contains(t, out, "Completed execution exec-1 (completeness full).")
		require.Contains(t, out, "Resume point: (none).")
		require.Contains(t, out, "[file_bugs] needs attention — open items")
	})

	t.Run("velocity renders points", func(t *testing.T) {
		rec := &execRecorder{resp: &executionv1.GetVelocityResponse{Points: []*sharedv1.VelocityPoint{
			{RecordedAt: "2026-01-01", WallTimeSeconds: 30, Tokens: 100, Iterations: 2, Completeness: sharedv1.Completeness_COMPLETENESS_FULL},
		}}}
		app, groups := newExecFixture(t, rec)
		out, err := clitest.RunCommand(t, clitest.FindCommand(t, groups, "exec", "velocity"), app, "plan-v")
		require.NoError(t, err)
		require.Contains(t, out, "1 velocity point(s) for plan plan-v.")
		require.Contains(t, out, "30s wall, 100 tokens, 2 iterations (full)")
	})
}

// TestExecErrorHandling covers int-parse rejection (pre-RPC) and server errors.
func TestExecErrorHandling(t *testing.T) {
	t.Run("complete rejects non-numeric tokens before RPC", func(t *testing.T) {
		rec := &execRecorder{}
		app, groups := newExecFixture(t, rec)
		_, err := clitest.RunCommand(t, clitest.FindCommand(t, groups, "exec", "complete"), app, "exec-1", "--tokens", "abc")
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid --tokens")
		require.Nil(t, rec.lastRequest(), "RPC must not be issued when --tokens is invalid")
	})

	t.Run("complete rejects non-numeric iterations before RPC", func(t *testing.T) {
		rec := &execRecorder{}
		app, groups := newExecFixture(t, rec)
		_, err := clitest.RunCommand(t, clitest.FindCommand(t, groups, "exec", "complete"), app, "exec-1", "--iterations", "x")
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid --iterations")
		require.Nil(t, rec.lastRequest())
	})

	t.Run("server error is wrapped", func(t *testing.T) {
		rec := &execRecorder{err: connect.NewError(connect.CodeUnavailable, errBoom())}
		app, groups := newExecFixture(t, rec)
		_, err := clitest.RunCommand(t, clitest.FindCommand(t, groups, "exec", "status"), app, "exec-1")
		require.Error(t, err)
		require.Contains(t, err.Error(), "get status")
	})

	t.Run("missing required positional is a parser error", func(t *testing.T) {
		rec := &execRecorder{}
		app, groups := newExecFixture(t, rec)
		_, err := clitest.RunCommand(t, clitest.FindCommand(t, groups, "exec", "status"), app)
		require.Error(t, err)
		require.Contains(t, err.Error(), "execution")
		require.Nil(t, rec.lastRequest())
	})
}

// --- direct unit tests of the flag->enum + enum->label + int helpers ---

func TestExecPhaseStatusFlag(t *testing.T) {
	cases := map[string]sharedv1.PhaseStatus{
		"todo":    sharedv1.PhaseStatus_PHASE_STATUS_TODO,
		"active":  sharedv1.PhaseStatus_PHASE_STATUS_ACTIVE,
		"done":    sharedv1.PhaseStatus_PHASE_STATUS_DONE,
		"blocked": sharedv1.PhaseStatus_PHASE_STATUS_BLOCKED,
		" DONE ":  sharedv1.PhaseStatus_PHASE_STATUS_DONE,
		"":        sharedv1.PhaseStatus_PHASE_STATUS_UNSPECIFIED,
		"weird":   sharedv1.PhaseStatus_PHASE_STATUS_UNSPECIFIED,
	}
	for in, want := range cases {
		require.Equalf(t, want, phaseStatusFlag(in), "phaseStatusFlag(%q)", in)
	}
}

func TestExecLabels(t *testing.T) {
	require.Equal(t, "active", phaseStatusLabel(sharedv1.PhaseStatus_PHASE_STATUS_ACTIVE))
	require.Equal(t, "unspecified", phaseStatusLabel(sharedv1.PhaseStatus_PHASE_STATUS_UNSPECIFIED))
	require.Equal(t, "complete", planStatusLabel(sharedv1.PlanStatus_PLAN_STATUS_COMPLETE))
	require.Equal(t, "full", completenessLabel(sharedv1.Completeness_COMPLETENESS_FULL))
	require.Equal(t, "partial", completenessLabel(sharedv1.Completeness_COMPLETENESS_PARTIAL))
	require.Equal(t, "unspecified", completenessLabel(sharedv1.Completeness_COMPLETENESS_UNSPECIFIED))
	require.Equal(t, "fresh", stalenessLabel(sharedv1.StalenessTier_STALENESS_TIER_FRESH))
	require.Equal(t, "lightly_stale", stalenessLabel(sharedv1.StalenessTier_STALENESS_TIER_LIGHTLY_STALE))
	require.Equal(t, "definitely_stale", stalenessLabel(sharedv1.StalenessTier_STALENESS_TIER_DEFINITELY_STALE))
	require.Equal(t, "unknown", stalenessLabel(sharedv1.StalenessTier_STALENESS_TIER_UNSPECIFIED))
}

func TestParseIntFlags(t *testing.T) {
	v64, err := parseInt64Flag("")
	require.NoError(t, err)
	require.Equal(t, int64(0), v64)
	v64, err = parseInt64Flag("  42 ")
	require.NoError(t, err)
	require.Equal(t, int64(42), v64)
	_, err = parseInt64Flag("nope")
	require.Error(t, err)

	v32, err := parseInt32Flag("")
	require.NoError(t, err)
	require.Equal(t, int32(0), v32)
	v32, err = parseInt32Flag("9")
	require.NoError(t, err)
	require.Equal(t, int32(9), v32)
	_, err = parseInt32Flag("x")
	require.Error(t, err)
}

func TestOrNone(t *testing.T) {
	require.Equal(t, "(none)", orNone(""))
	require.Equal(t, "(none)", orNone("   "))
	require.Equal(t, "abc", orNone("abc"))
}

func errBoom() error { return &boomError{} }

type boomError struct{}

func (*boomError) Error() string { return "boom" }
