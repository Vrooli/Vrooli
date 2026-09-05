package validation

import (
	"context"
	"net/http"
	"sync"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/shared"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/validation"
	validationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/validation/validation_v1connect"

	"github.com/vrooli/cli-core/cliapp"
	clitest "github.com/vrooli/cli-core/cliapptest"
)

// validationRecorder is a fake ValidationService capturing the request the
// handler built and returning a canned response (or error).
type validationRecorder struct {
	validationconnect.UnimplementedValidationServiceHandler
	mu        sync.Mutex
	req       proto.Message
	resp      proto.Message
	err       error
	waitErr   error
	waitCalls int
}

func (r *validationRecorder) record(req proto.Message) {
	r.mu.Lock()
	r.req = req
	r.mu.Unlock()
}

func (r *validationRecorder) lastRequest() proto.Message {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.req
}

func (r *validationRecorder) ResolveReferences(_ context.Context, req *connect.Request[validationv1.ResolveReferencesRequest]) (*connect.Response[validationv1.ResolveReferencesResponse], error) {
	r.record(req.Msg)
	if r.err != nil {
		return nil, r.err
	}
	if m, ok := r.resp.(*validationv1.ResolveReferencesResponse); ok && m != nil {
		return connect.NewResponse(m), nil
	}
	return connect.NewResponse(&validationv1.ResolveReferencesResponse{}), nil
}

func (r *validationRecorder) ComputeStaleness(_ context.Context, req *connect.Request[validationv1.ComputeStalenessRequest]) (*connect.Response[validationv1.ComputeStalenessResponse], error) {
	r.record(req.Msg)
	if r.err != nil {
		return nil, r.err
	}
	if m, ok := r.resp.(*validationv1.ComputeStalenessResponse); ok && m != nil {
		return connect.NewResponse(m), nil
	}
	return connect.NewResponse(&validationv1.ComputeStalenessResponse{}), nil
}

func (r *validationRecorder) DeriveBaselineScope(_ context.Context, req *connect.Request[validationv1.DeriveBaselineScopeRequest]) (*connect.Response[validationv1.DeriveBaselineScopeResponse], error) {
	r.record(req.Msg)
	if r.err != nil {
		return nil, r.err
	}
	if m, ok := r.resp.(*validationv1.DeriveBaselineScopeResponse); ok && m != nil {
		return connect.NewResponse(m), nil
	}
	return connect.NewResponse(&validationv1.DeriveBaselineScopeResponse{}), nil
}

func (r *validationRecorder) RunValidation(_ context.Context, req *connect.Request[validationv1.RunValidationRequest]) (*connect.Response[validationv1.RunValidationResponse], error) {
	r.record(req.Msg)
	if r.err != nil {
		return nil, r.err
	}
	if m, ok := r.resp.(*validationv1.RunValidationResponse); ok && m != nil {
		return connect.NewResponse(m), nil
	}
	return connect.NewResponse(&validationv1.RunValidationResponse{Result: &sharedv1.ValidationResult{}}), nil
}

func (r *validationRecorder) StartValidation(_ context.Context, req *connect.Request[validationv1.StartValidationRequest]) (*connect.Response[validationv1.StartValidationResponse], error) {
	r.record(req.Msg)
	if r.err != nil {
		return nil, r.err
	}
	if m, ok := r.resp.(*validationv1.StartValidationResponse); ok && m != nil {
		return connect.NewResponse(m), nil
	}
	return connect.NewResponse(&validationv1.StartValidationResponse{Operation: &validationv1.ValidationOperation{}}), nil
}

func (r *validationRecorder) GetValidationOperation(_ context.Context, req *connect.Request[validationv1.GetValidationOperationRequest]) (*connect.Response[validationv1.GetValidationOperationResponse], error) {
	return r.validationOperation(req)
}

func (r *validationRecorder) WaitValidationOperation(_ context.Context, req *connect.Request[validationv1.GetValidationOperationRequest]) (*connect.Response[validationv1.GetValidationOperationResponse], error) {
	r.mu.Lock()
	r.waitCalls++
	call := r.waitCalls
	waitErr := r.waitErr
	r.mu.Unlock()
	if call == 1 && waitErr != nil {
		r.record(req.Msg)
		return nil, waitErr
	}
	return r.validationOperation(req)
}

func (r *validationRecorder) ResumeValidationOperation(_ context.Context, req *connect.Request[validationv1.GetValidationOperationRequest]) (*connect.Response[validationv1.GetValidationOperationResponse], error) {
	return r.validationOperation(req)
}

func (r *validationRecorder) validationOperation(req *connect.Request[validationv1.GetValidationOperationRequest]) (*connect.Response[validationv1.GetValidationOperationResponse], error) {
	r.record(req.Msg)
	if r.err != nil {
		return nil, r.err
	}
	if m, ok := r.resp.(*validationv1.GetValidationOperationResponse); ok && m != nil {
		return connect.NewResponse(m), nil
	}
	return connect.NewResponse(&validationv1.GetValidationOperationResponse{Operation: &validationv1.ValidationOperation{}}), nil
}

func (r *validationRecorder) VerifyDefinitionOfDone(_ context.Context, req *connect.Request[validationv1.VerifyDefinitionOfDoneRequest]) (*connect.Response[validationv1.VerifyDefinitionOfDoneResponse], error) {
	r.record(req.Msg)
	if r.err != nil {
		return nil, r.err
	}
	if m, ok := r.resp.(*validationv1.VerifyDefinitionOfDoneResponse); ok && m != nil {
		return connect.NewResponse(m), nil
	}
	return connect.NewResponse(&validationv1.VerifyDefinitionOfDoneResponse{Result: &sharedv1.ValidationResult{}}), nil
}

func newValidationFixture(t *testing.T, rec *validationRecorder) (*cliapp.ScenarioApp, []cliapp.SubcommandGroup) {
	t.Helper()
	mux := http.NewServeMux()
	path, handler := validationconnect.NewValidationServiceHandler(rec)
	mux.Handle(path, handler)
	app := clitest.NewTestApp(t, mux)
	group, err := Register(app, clitest.ReadManifest(t))
	require.NoError(t, err, "register validate group against manifest")
	return app, []cliapp.SubcommandGroup{group}
}

// TestValidationRequestMapping drives every validate verb end-to-end and asserts
// the plan positional + optional phase flag land on the right request fields.
func TestValidationRequestMapping(t *testing.T) {
	tests := []struct {
		name   string
		cmd    string
		argv   []string
		assert func(t *testing.T, req proto.Message)
	}{
		{
			name: "references maps plan positional + phase flag", cmd: "references",
			argv: []string{"plan-1", "--phase", "phase-2"},
			assert: func(t *testing.T, req proto.Message) {
				m := req.(*validationv1.ResolveReferencesRequest)
				require.Equal(t, "plan-1", m.GetPlanId())
				require.Equal(t, "phase-2", m.GetPhaseId())
			},
		},
		{
			name: "references with no phase flag", cmd: "references",
			argv: []string{"plan-1"},
			assert: func(t *testing.T, req proto.Message) {
				m := req.(*validationv1.ResolveReferencesRequest)
				require.Equal(t, "plan-1", m.GetPlanId())
				require.Equal(t, "", m.GetPhaseId())
			},
		},
		{
			name: "staleness maps plan positional + phase flag", cmd: "staleness",
			argv: []string{"plan-1", "--phase", "phase-3"},
			assert: func(t *testing.T, req proto.Message) {
				m := req.(*validationv1.ComputeStalenessRequest)
				require.Equal(t, "plan-1", m.GetPlanId())
				require.Equal(t, "phase-3", m.GetPhaseId())
			},
		},
		{
			name: "baseline-scope maps plan positional + phase flag", cmd: "baseline-scope",
			argv: []string{"plan-1", "--phase", "phase-4"},
			assert: func(t *testing.T, req proto.Message) {
				m := req.(*validationv1.DeriveBaselineScopeRequest)
				require.Equal(t, "plan-1", m.GetPlanId())
				require.Equal(t, "phase-4", m.GetPhaseId())
			},
		},
		{
			name: "start maps plan, phase, and idempotency key", cmd: "start",
			argv: []string{"plan-1", "--phase", "phase-5", "--idempotency-key", "retry-1"},
			assert: func(t *testing.T, req proto.Message) {
				m := req.(*validationv1.StartValidationRequest)
				require.Equal(t, "plan-1", m.GetPlanId())
				require.Equal(t, "phase-5", m.GetPhaseId())
				require.Equal(t, "retry-1", m.GetIdempotencyKey())
			},
		},
		{
			name: "show maps operation without wait", cmd: "show", argv: []string{"op-1"},
			assert: func(t *testing.T, req proto.Message) {
				m := req.(*validationv1.GetValidationOperationRequest)
				require.Equal(t, "op-1", m.GetOperationId())
				require.False(t, m.GetWait())
			},
		},
		{
			name: "wait is a legacy inspection alias", cmd: "wait", argv: []string{"op-2"},
			assert: func(t *testing.T, req proto.Message) {
				m := req.(*validationv1.GetValidationOperationRequest)
				require.Equal(t, "op-2", m.GetOperationId())
				require.False(t, m.GetWait())
			},
		},
		{
			name: "resume is a legacy inspection alias", cmd: "resume", argv: []string{"op-3"},
			assert: func(t *testing.T, req proto.Message) {
				m := req.(*validationv1.GetValidationOperationRequest)
				require.Equal(t, "op-3", m.GetOperationId())
				require.False(t, m.GetWait())
			},
		},
		{
			name: "run maps plan positional + phase flag", cmd: "run",
			argv: []string{"plan-1", "--phase", "phase-5"},
			assert: func(t *testing.T, req proto.Message) {
				m := req.(*validationv1.RunValidationRequest)
				require.Equal(t, "plan-1", m.GetPlanId())
				require.Equal(t, "phase-5", m.GetPhaseId())
			},
		},
		{
			name: "verify-dod maps plan positional", cmd: "verify-dod",
			argv: []string{"plan-1"},
			assert: func(t *testing.T, req proto.Message) {
				require.Equal(t, "plan-1", req.(*validationv1.VerifyDefinitionOfDoneRequest).GetPlanId())
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := &validationRecorder{}
			app, groups := newValidationFixture(t, rec)
			cmd := clitest.FindCommand(t, groups, "validate", tc.cmd)
			_, err := clitest.RunCommand(t, cmd, app, tc.argv...)
			require.NoError(t, err)
			req := rec.lastRequest()
			require.NotNil(t, req, "handler must have issued a request")
			tc.assert(t, req)
		})
	}
}

// TestValidationOutputRendering pins the human render, including the enum->label
// projections (verdict, staleness, resolution) that drive the output strings.
func TestValidationOutputRendering(t *testing.T) {
	t.Run("references renders resolved + degraded note", func(t *testing.T) {
		rec := &validationRecorder{resp: &validationv1.ResolveReferencesResponse{
			Degraded: true,
			References: []*sharedv1.Reference{
				{Kind: sharedv1.ReferenceKind_REFERENCE_KIND_REQ, Target: "OT-P0-001", Resolution: sharedv1.ReferenceResolution_REFERENCE_RESOLUTION_RESOLVED, Staleness: sharedv1.StalenessTier_STALENESS_TIER_FRESH},
				{Kind: sharedv1.ReferenceKind_REFERENCE_KIND_CODE, Target: "foo.go", Resolution: sharedv1.ReferenceResolution_REFERENCE_RESOLUTION_MISSING, Staleness: sharedv1.StalenessTier_STALENESS_TIER_DEFINITELY_STALE, Note: "moved"},
			},
		}}
		app, groups := newValidationFixture(t, rec)
		out, err := clitest.RunCommand(t, clitest.FindCommand(t, groups, "validate", "references"), app, "plan-1")
		require.NoError(t, err)
		require.Contains(t, out, "Resolved 2 reference(s). (degraded: code-facts unavailable)")
		require.Contains(t, out, "[REQ: OT-P0-001] resolution=resolved staleness=fresh")
		require.Contains(t, out, "[CODE: foo.go] resolution=missing staleness=definitely_stale (moved)")
	})

	t.Run("staleness renders overall tier", func(t *testing.T) {
		rec := &validationRecorder{resp: &validationv1.ComputeStalenessResponse{
			Overall: sharedv1.StalenessTier_STALENESS_TIER_LIGHTLY_STALE,
		}}
		app, groups := newValidationFixture(t, rec)
		out, err := clitest.RunCommand(t, clitest.FindCommand(t, groups, "validate", "staleness"), app, "plan-1")
		require.NoError(t, err)
		require.Contains(t, out, "Overall staleness: lightly_stale.")
	})

	t.Run("baseline-scope renders commands + locations count", func(t *testing.T) {
		rec := &validationRecorder{resp: &validationv1.DeriveBaselineScopeResponse{
			Commands:  []string{"make test", "go vet ./..."},
			Locations: []string{"scenarios/plan-manager"},
		}}
		app, groups := newValidationFixture(t, rec)
		out, err := clitest.RunCommand(t, clitest.FindCommand(t, groups, "validate", "baseline-scope"), app, "plan-1")
		require.NoError(t, err)
		require.Contains(t, out, "Derived 2 command(s) across 1 location(s).")
		require.Contains(t, out, "make test")
		require.Contains(t, out, "go vet ./...")
	})

	t.Run("run renders verdict + staleness", func(t *testing.T) {
		rec := &validationRecorder{resp: &validationv1.RunValidationResponse{Result: &sharedv1.ValidationResult{
			Verdict: sharedv1.ValidationVerdict_VALIDATION_VERDICT_PASS, Staleness: sharedv1.StalenessTier_STALENESS_TIER_FRESH, Detail: "all green",
		}}}
		app, groups := newValidationFixture(t, rec)
		out, err := clitest.RunCommand(t, clitest.FindCommand(t, groups, "validate", "run"), app, "plan-1")
		require.NoError(t, err)
		require.Contains(t, out, "Verdict: pass (staleness fresh).")
		require.Contains(t, out, "all green")
	})

	t.Run("start renders durable id and producer-sync guidance", func(t *testing.T) {
		rec := &validationRecorder{resp: &validationv1.StartValidationResponse{
			Operation:    &validationv1.ValidationOperation{Id: "op-123", Status: validationv1.ValidationOperationStatus_VALIDATION_OPERATION_STATUS_QUEUED, ScopeFingerprint: "sha256:scope", QueueReason: "awaiting scheduler claim"},
			Deduplicated: true,
		}}
		app, groups := newValidationFixture(t, rec)
		out, err := clitest.RunCommand(t, clitest.FindCommand(t, groups, "validate", "start"), app, "plan-1", "--idempotency-key", "same")
		require.NoError(t, err)
		require.Contains(t, out, "Validation operation: op-123")
		require.Contains(t, out, "no child work was duplicated")
		require.Contains(t, out, "Scope fingerprint: sha256:scope.")
		require.Contains(t, out, "Queue reason: awaiting scheduler claim.")
		require.Contains(t, out, "plan-manager validate sync op-123")
	})

	t.Run("wait renders terminal durable result", func(t *testing.T) {
		rec := &validationRecorder{resp: &validationv1.GetValidationOperationResponse{Operation: &validationv1.ValidationOperation{
			Id: "op-terminal", Status: validationv1.ValidationOperationStatus_VALIDATION_OPERATION_STATUS_TERMINAL,
			Attempt: 1, Result: &sharedv1.ValidationResult{Verdict: sharedv1.ValidationVerdict_VALIDATION_VERDICT_PASS, Staleness: sharedv1.StalenessTier_STALENESS_TIER_FRESH, Detail: "oracle clean"},
		}}}
		app, groups := newValidationFixture(t, rec)
		out, err := clitest.RunCommand(t, clitest.FindCommand(t, groups, "validate", "wait"), app, "op-terminal")
		require.NoError(t, err)
		require.Contains(t, out, "Status: terminal")
		require.Contains(t, out, "Verdict: pass")
		require.Contains(t, out, "oracle clean")
	})

	t.Run("verify-dod renders met verdict", func(t *testing.T) {
		rec := &validationRecorder{resp: &validationv1.VerifyDefinitionOfDoneResponse{
			DodMet: true,
			Result: &sharedv1.ValidationResult{Verdict: sharedv1.ValidationVerdict_VALIDATION_VERDICT_PASS, Detail: "baseline exit-0"},
		}}
		app, groups := newValidationFixture(t, rec)
		out, err := clitest.RunCommand(t, clitest.FindCommand(t, groups, "validate", "verify-dod"), app, "plan-1")
		require.NoError(t, err)
		require.Contains(t, out, "Definition of Done met (verdict pass).")
	})

	t.Run("verify-dod renders NOT met verdict", func(t *testing.T) {
		rec := &validationRecorder{resp: &validationv1.VerifyDefinitionOfDoneResponse{
			DodMet: false,
			Result: &sharedv1.ValidationResult{Verdict: sharedv1.ValidationVerdict_VALIDATION_VERDICT_FAIL, Detail: "diff non-empty"},
		}}
		app, groups := newValidationFixture(t, rec)
		out, err := clitest.RunCommand(t, clitest.FindCommand(t, groups, "validate", "verify-dod"), app, "plan-1")
		require.NoError(t, err)
		require.Contains(t, out, "Definition of Done NOT met (verdict fail).")
	})
}

// TestValidationErrorHandling covers server error wrapping + missing positional.
func TestValidationErrorHandling(t *testing.T) {
	t.Run("server error is wrapped", func(t *testing.T) {
		rec := &validationRecorder{err: connect.NewError(connect.CodeInternal, errBoom())}
		app, groups := newValidationFixture(t, rec)
		_, err := clitest.RunCommand(t, clitest.FindCommand(t, groups, "validate", "run"), app, "plan-1")
		require.Error(t, err)
		require.Contains(t, err.Error(), "run validation")
		require.Contains(t, err.Error(), "plan-manager author validate <session>")
	})

	t.Run("missing required positional is a parser error", func(t *testing.T) {
		rec := &validationRecorder{}
		app, groups := newValidationFixture(t, rec)
		_, err := clitest.RunCommand(t, clitest.FindCommand(t, groups, "validate", "run"), app)
		require.Error(t, err)
		require.Contains(t, err.Error(), "plan")
		require.Nil(t, rec.lastRequest())
	})
}

// --- direct unit tests of the enum->label + reference formatting helpers ---

func TestVerdictLabel(t *testing.T) {
	require.Equal(t, "pass", verdictLabel(sharedv1.ValidationVerdict_VALIDATION_VERDICT_PASS))
	require.Equal(t, "fail", verdictLabel(sharedv1.ValidationVerdict_VALIDATION_VERDICT_FAIL))
	require.Equal(t, "unknown", verdictLabel(sharedv1.ValidationVerdict_VALIDATION_VERDICT_UNKNOWN))
	require.Equal(t, "unspecified", verdictLabel(sharedv1.ValidationVerdict_VALIDATION_VERDICT_UNSPECIFIED))
}

func TestStalenessLabel(t *testing.T) {
	require.Equal(t, "fresh", stalenessLabel(sharedv1.StalenessTier_STALENESS_TIER_FRESH))
	require.Equal(t, "lightly_stale", stalenessLabel(sharedv1.StalenessTier_STALENESS_TIER_LIGHTLY_STALE))
	require.Equal(t, "definitely_stale", stalenessLabel(sharedv1.StalenessTier_STALENESS_TIER_DEFINITELY_STALE))
	require.Equal(t, "unknown", stalenessLabel(sharedv1.StalenessTier_STALENESS_TIER_UNSPECIFIED))
}

func TestResolutionLabel(t *testing.T) {
	require.Equal(t, "resolved", resolutionLabel(sharedv1.ReferenceResolution_REFERENCE_RESOLUTION_RESOLVED))
	require.Equal(t, "unresolved", resolutionLabel(sharedv1.ReferenceResolution_REFERENCE_RESOLUTION_UNRESOLVED))
	require.Equal(t, "future", resolutionLabel(sharedv1.ReferenceResolution_REFERENCE_RESOLUTION_FUTURE))
	require.Equal(t, "missing", resolutionLabel(sharedv1.ReferenceResolution_REFERENCE_RESOLUTION_MISSING))
	require.Equal(t, "unspecified", resolutionLabel(sharedv1.ReferenceResolution_REFERENCE_RESOLUTION_UNSPECIFIED))
}

func TestFormatReference(t *testing.T) {
	req := formatReference(&sharedv1.Reference{
		Kind: sharedv1.ReferenceKind_REFERENCE_KIND_REQ, Target: "OT-1",
		Resolution: sharedv1.ReferenceResolution_REFERENCE_RESOLUTION_RESOLVED,
		Staleness:  sharedv1.StalenessTier_STALENESS_TIER_FRESH,
	})
	require.Equal(t, "[REQ: OT-1] resolution=resolved staleness=fresh", req)

	doc := formatReference(&sharedv1.Reference{
		Kind: sharedv1.ReferenceKind_REFERENCE_KIND_DOC, Target: "docs/x.md",
		Resolution: sharedv1.ReferenceResolution_REFERENCE_RESOLUTION_UNRESOLVED,
		Staleness:  sharedv1.StalenessTier_STALENESS_TIER_LIGHTLY_STALE, Note: "renamed",
	})
	require.Equal(t, "[DOC: docs/x.md] resolution=unresolved staleness=lightly_stale (renamed)", doc)

	// The default marker for an unmarked/CODE reference kind is "CODE".
	code := formatReference(&sharedv1.Reference{
		Kind: sharedv1.ReferenceKind_REFERENCE_KIND_CODE, Target: "foo.go",
		Resolution: sharedv1.ReferenceResolution_REFERENCE_RESOLUTION_MISSING,
		Staleness:  sharedv1.StalenessTier_STALENESS_TIER_DEFINITELY_STALE,
	})
	require.Equal(t, "[CODE: foo.go] resolution=missing staleness=definitely_stale", code)
}

func errBoom() error { return &boomError{} }

type boomError struct{}

func (*boomError) Error() string { return "boom" }
