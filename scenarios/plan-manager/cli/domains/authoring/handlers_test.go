package authoring

import (
	"context"
	"net/http"
	"sync"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	authoringv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/authoring"
	authoringconnect "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/authoring/authoring_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/shared"

	"github.com/vrooli/cli-core/cliapp"
	clitest "plan-manager/cli/internal/testutil"
)

// authRecorder is a fake AuthoringService capturing the request the handler
// built and returning a canned response (or error).
type authRecorder struct {
	authoringconnect.UnimplementedAuthoringServiceHandler
	mu   sync.Mutex
	req  proto.Message
	resp proto.Message
	err  error
}

func (r *authRecorder) record(req proto.Message) {
	r.mu.Lock()
	r.req = req
	r.mu.Unlock()
}

func (r *authRecorder) lastRequest() proto.Message {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.req
}

func (r *authRecorder) StartSession(_ context.Context, req *connect.Request[authoringv1.StartSessionRequest]) (*connect.Response[authoringv1.StartSessionResponse], error) {
	r.record(req.Msg)
	if r.err != nil {
		return nil, r.err
	}
	if m, ok := r.resp.(*authoringv1.StartSessionResponse); ok && m != nil {
		return connect.NewResponse(m), nil
	}
	return connect.NewResponse(&authoringv1.StartSessionResponse{Session: &authoringv1.AuthoringSession{Id: "sess-1", Title: req.Msg.GetTitle()}}), nil
}

func (r *authRecorder) GetSection(_ context.Context, req *connect.Request[authoringv1.GetSectionRequest]) (*connect.Response[authoringv1.GetSectionResponse], error) {
	r.record(req.Msg)
	if r.err != nil {
		return nil, r.err
	}
	if m, ok := r.resp.(*authoringv1.GetSectionResponse); ok && m != nil {
		return connect.NewResponse(m), nil
	}
	return connect.NewResponse(&authoringv1.GetSectionResponse{Section: &authoringv1.Section{Key: req.Msg.GetSectionKey()}}), nil
}

func (r *authRecorder) SubmitSection(_ context.Context, req *connect.Request[authoringv1.SubmitSectionRequest]) (*connect.Response[authoringv1.SubmitSectionResponse], error) {
	r.record(req.Msg)
	if r.err != nil {
		return nil, r.err
	}
	if m, ok := r.resp.(*authoringv1.SubmitSectionResponse); ok && m != nil {
		return connect.NewResponse(m), nil
	}
	return connect.NewResponse(&authoringv1.SubmitSectionResponse{Session: &authoringv1.AuthoringSession{CurrentSectionKey: "next-key"}}), nil
}

func (r *authRecorder) Next(_ context.Context, req *connect.Request[authoringv1.NextRequest]) (*connect.Response[authoringv1.NextResponse], error) {
	r.record(req.Msg)
	if r.err != nil {
		return nil, r.err
	}
	if m, ok := r.resp.(*authoringv1.NextResponse); ok && m != nil {
		return connect.NewResponse(m), nil
	}
	return connect.NewResponse(&authoringv1.NextResponse{Section: &authoringv1.Section{Key: "purpose"}}), nil
}

func (r *authRecorder) ValidateStructure(_ context.Context, req *connect.Request[authoringv1.ValidateStructureRequest]) (*connect.Response[authoringv1.ValidateStructureResponse], error) {
	r.record(req.Msg)
	if r.err != nil {
		return nil, r.err
	}
	if m, ok := r.resp.(*authoringv1.ValidateStructureResponse); ok && m != nil {
		return connect.NewResponse(m), nil
	}
	return connect.NewResponse(&authoringv1.ValidateStructureResponse{Valid: true}), nil
}

func (r *authRecorder) Autofill(_ context.Context, req *connect.Request[authoringv1.AutofillRequest]) (*connect.Response[authoringv1.AutofillResponse], error) {
	r.record(req.Msg)
	if r.err != nil {
		return nil, r.err
	}
	if m, ok := r.resp.(*authoringv1.AutofillResponse); ok && m != nil {
		return connect.NewResponse(m), nil
	}
	return connect.NewResponse(&authoringv1.AutofillResponse{}), nil
}

func (r *authRecorder) Finalize(_ context.Context, req *connect.Request[authoringv1.FinalizeRequest]) (*connect.Response[authoringv1.FinalizeResponse], error) {
	r.record(req.Msg)
	if r.err != nil {
		return nil, r.err
	}
	if m, ok := r.resp.(*authoringv1.FinalizeResponse); ok && m != nil {
		return connect.NewResponse(m), nil
	}
	return connect.NewResponse(&authoringv1.FinalizeResponse{Plan: &sharedv1.Plan{Id: "plan-final", Slug: "pf"}}), nil
}

func (r *authRecorder) AddPhase(_ context.Context, req *connect.Request[authoringv1.AddPhaseRequest]) (*connect.Response[authoringv1.AddPhaseResponse], error) {
	r.record(req.Msg)
	if r.err != nil {
		return nil, r.err
	}
	if m, ok := r.resp.(*authoringv1.AddPhaseResponse); ok && m != nil {
		return connect.NewResponse(m), nil
	}
	return connect.NewResponse(&authoringv1.AddPhaseResponse{Phase: &authoringv1.PhaseDraft{Id: "ph1", Order: 1, Title: req.Msg.GetTitle()}}), nil
}

func (r *authRecorder) GetPhase(_ context.Context, req *connect.Request[authoringv1.GetPhaseRequest]) (*connect.Response[authoringv1.GetPhaseResponse], error) {
	r.record(req.Msg)
	if r.err != nil {
		return nil, r.err
	}
	if m, ok := r.resp.(*authoringv1.GetPhaseResponse); ok && m != nil {
		return connect.NewResponse(m), nil
	}
	return connect.NewResponse(&authoringv1.GetPhaseResponse{Phase: &authoringv1.PhaseDraft{Id: req.Msg.GetPhaseId(), Order: 1}}), nil
}

func (r *authRecorder) SubmitPhaseField(_ context.Context, req *connect.Request[authoringv1.SubmitPhaseFieldRequest]) (*connect.Response[authoringv1.SubmitPhaseFieldResponse], error) {
	r.record(req.Msg)
	if r.err != nil {
		return nil, r.err
	}
	if m, ok := r.resp.(*authoringv1.SubmitPhaseFieldResponse); ok && m != nil {
		return connect.NewResponse(m), nil
	}
	return connect.NewResponse(&authoringv1.SubmitPhaseFieldResponse{Session: &authoringv1.AuthoringSession{Id: req.Msg.GetSessionId()}}), nil
}

func (r *authRecorder) NextPhase(_ context.Context, req *connect.Request[authoringv1.NextPhaseRequest]) (*connect.Response[authoringv1.NextPhaseResponse], error) {
	r.record(req.Msg)
	if r.err != nil {
		return nil, r.err
	}
	if m, ok := r.resp.(*authoringv1.NextPhaseResponse); ok && m != nil {
		return connect.NewResponse(m), nil
	}
	return connect.NewResponse(&authoringv1.NextPhaseResponse{Phase: &authoringv1.PhaseDraft{Id: "ph1", Order: 1}}), nil
}

func newAuthFixture(t *testing.T, rec *authRecorder) (*cliapp.ScenarioApp, []cliapp.SubcommandGroup) {
	t.Helper()
	mux := http.NewServeMux()
	path, handler := authoringconnect.NewAuthoringServiceHandler(rec)
	mux.Handle(path, handler)
	app := clitest.NewTestApp(t, mux)
	group, err := Register(app, clitest.ReadManifest(t))
	require.NoError(t, err, "register author group against manifest")
	return app, []cliapp.SubcommandGroup{group}
}

// TestAuthoringRequestMapping drives every covered author verb end-to-end and
// asserts the typed request, including the comma-split --sources parsing.
func TestAuthoringRequestMapping(t *testing.T) {
	tests := []struct {
		name   string
		cmd    string
		argv   []string
		assert func(t *testing.T, req proto.Message)
	}{
		{
			name: "start maps title/slug/template flags", cmd: "start",
			argv: []string{"--title", "My Plan", "--slug", "mp", "--template", "tmpl-cli"},
			assert: func(t *testing.T, req proto.Message) {
				m := req.(*authoringv1.StartSessionRequest)
				require.Equal(t, "My Plan", m.GetTitle())
				require.Equal(t, "mp", m.GetSlug())
				require.Equal(t, "tmpl-cli", m.GetTemplateId())
			},
		},
		{
			name: "section-get maps session positional + section flag", cmd: "section-get",
			argv: []string{"sess-1", "--section", "purpose"},
			assert: func(t *testing.T, req proto.Message) {
				m := req.(*authoringv1.GetSectionRequest)
				require.Equal(t, "sess-1", m.GetSessionId())
				require.Equal(t, "purpose", m.GetSectionKey())
			},
		},
		{
			name: "section-submit maps session positional + section/content", cmd: "section-submit",
			argv: []string{"sess-1", "--section", "scope", "--content", "the scope text"},
			assert: func(t *testing.T, req proto.Message) {
				m := req.(*authoringv1.SubmitSectionRequest)
				require.Equal(t, "sess-1", m.GetSessionId())
				require.Equal(t, "scope", m.GetSectionKey())
				require.Equal(t, "the scope text", m.GetContent())
			},
		},
		{
			name: "next maps session positional", cmd: "next",
			argv: []string{"sess-1"},
			assert: func(t *testing.T, req proto.Message) {
				require.Equal(t, "sess-1", req.(*authoringv1.NextRequest).GetSessionId())
			},
		},
		{
			name: "validate maps session positional", cmd: "validate",
			argv: []string{"sess-1"},
			assert: func(t *testing.T, req proto.Message) {
				require.Equal(t, "sess-1", req.(*authoringv1.ValidateStructureRequest).GetSessionId())
			},
		},
		{
			name: "autofill maps session + comma-split sources", cmd: "autofill",
			argv: []string{"sess-1", "--sources", "regression_anchor, references , required_reading"},
			assert: func(t *testing.T, req proto.Message) {
				m := req.(*authoringv1.AutofillRequest)
				require.Equal(t, "sess-1", m.GetSessionId())
				require.Equal(t, []string{"regression_anchor", "references", "required_reading"}, m.GetSources())
			},
		},
		{
			name: "autofill with no sources sends nil (run all)", cmd: "autofill",
			argv: []string{"sess-1"},
			assert: func(t *testing.T, req proto.Message) {
				require.Empty(t, req.(*authoringv1.AutofillRequest).GetSources())
			},
		},
		{
			name: "finalize maps session positional", cmd: "finalize",
			argv: []string{"sess-1"},
			assert: func(t *testing.T, req proto.Message) {
				require.Equal(t, "sess-1", req.(*authoringv1.FinalizeRequest).GetSessionId())
			},
		},
		{
			name: "phase-add maps session title intent", cmd: "phase-add",
			argv: []string{"sess-1", "--title", "Contract", "--intent", "Add RPCs"},
			assert: func(t *testing.T, req proto.Message) {
				m := req.(*authoringv1.AddPhaseRequest)
				require.Equal(t, "sess-1", m.GetSessionId())
				require.Equal(t, "Contract", m.GetTitle())
				require.Equal(t, "Add RPCs", m.GetIntent())
			},
		},
		{
			name: "phase-get maps session and phase", cmd: "phase-get",
			argv: []string{"sess-1", "ph1"},
			assert: func(t *testing.T, req proto.Message) {
				m := req.(*authoringv1.GetPhaseRequest)
				require.Equal(t, "sess-1", m.GetSessionId())
				require.Equal(t, "ph1", m.GetPhaseId())
			},
		},
		{
			name: "phase-submit maps field content", cmd: "phase-submit",
			argv: []string{"sess-1", "ph1", "--field", "references", "--content", "[CODE: x.go]"},
			assert: func(t *testing.T, req proto.Message) {
				m := req.(*authoringv1.SubmitPhaseFieldRequest)
				require.Equal(t, "sess-1", m.GetSessionId())
				require.Equal(t, "ph1", m.GetPhaseId())
				require.Equal(t, "references", m.GetField())
				require.Equal(t, "[CODE: x.go]", m.GetContent())
			},
		},
		{
			name: "phase-next maps session", cmd: "phase-next",
			argv: []string{"sess-1"},
			assert: func(t *testing.T, req proto.Message) {
				require.Equal(t, "sess-1", req.(*authoringv1.NextPhaseRequest).GetSessionId())
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := &authRecorder{}
			app, groups := newAuthFixture(t, rec)
			cmd := clitest.FindCommand(t, groups, "author", tc.cmd)
			_, err := clitest.RunCommand(t, cmd, app, tc.argv...)
			require.NoError(t, err)
			req := rec.lastRequest()
			require.NotNil(t, req, "handler must have issued a request")
			tc.assert(t, req)
		})
	}
}

// TestAuthoringOutputRendering pins the human render for the wizard verbs.
func TestAuthoringOutputRendering(t *testing.T) {
	t.Run("start renders session summary", func(t *testing.T) {
		rec := &authRecorder{resp: &authoringv1.StartSessionResponse{Session: &authoringv1.AuthoringSession{
			Id: "sess-1", Title: "My Plan", CurrentSectionKey: "purpose",
			Sections: []*authoringv1.Section{{Key: "purpose"}, {Key: "scope"}},
		}}}
		app, groups := newAuthFixture(t, rec)
		out, err := clitest.RunCommand(t, clitest.FindCommand(t, groups, "author", "start"), app, "--title", "My Plan")
		require.NoError(t, err)
		require.Contains(t, out, `Started session sess-1 for "My Plan".`)
		require.Contains(t, out, "Seeded 2 section(s); next: purpose.")
	})

	t.Run("validate renders valid verdict", func(t *testing.T) {
		rec := &authRecorder{resp: &authoringv1.ValidateStructureResponse{Valid: true}}
		app, groups := newAuthFixture(t, rec)
		out, err := clitest.RunCommand(t, clitest.FindCommand(t, groups, "author", "validate"), app, "sess-1")
		require.NoError(t, err)
		require.Contains(t, out, "Structure is valid (0 violation(s)).")
	})

	t.Run("validate renders INVALID verdict + violations", func(t *testing.T) {
		rec := &authRecorder{resp: &authoringv1.ValidateStructureResponse{
			Valid:      false,
			Violations: []*authoringv1.StructureViolation{{SectionKey: "scope", Message: "empty"}},
		}}
		app, groups := newAuthFixture(t, rec)
		out, err := clitest.RunCommand(t, clitest.FindCommand(t, groups, "author", "validate"), app, "sess-1")
		require.NoError(t, err)
		require.Contains(t, out, "Structure is INVALID (1 violation(s)).")
		require.Contains(t, out, "scope: empty")
	})

	t.Run("autofill renders filled count", func(t *testing.T) {
		rec := &authRecorder{resp: &authoringv1.AutofillResponse{Results: []*authoringv1.AutofillResult{
			{Source: "regression_anchor", SectionKey: "regression_anchor", Filled: true},
			{Source: "references", Degraded: true, Detail: "code-facts down"},
		}}}
		app, groups := newAuthFixture(t, rec)
		out, err := clitest.RunCommand(t, clitest.FindCommand(t, groups, "author", "autofill"), app, "sess-1")
		require.NoError(t, err)
		require.Contains(t, out, "Autofilled 1 of 2 source(s).")
		require.Contains(t, out, "regression_anchor → filled regression_anchor")
		require.Contains(t, out, "references → degraded (code-facts down)")
	})

	t.Run("next signals complete", func(t *testing.T) {
		rec := &authRecorder{resp: &authoringv1.NextResponse{Complete: true}}
		app, groups := newAuthFixture(t, rec)
		out, err := clitest.RunCommand(t, clitest.FindCommand(t, groups, "author", "next"), app, "sess-1")
		require.NoError(t, err)
		require.Contains(t, out, "All mandatory sections are filled.")
	})

	t.Run("finalize renders persisted plan", func(t *testing.T) {
		rec := &authRecorder{resp: &authoringv1.FinalizeResponse{Plan: &sharedv1.Plan{
			Id: "plan-final", Slug: "pf",
			Phases:     []*sharedv1.Phase{{Id: "ph1"}},
			References: []*sharedv1.Reference{{Id: "r1"}},
		}}}
		app, groups := newAuthFixture(t, rec)
		out, err := clitest.RunCommand(t, clitest.FindCommand(t, groups, "author", "finalize"), app, "sess-1")
		require.NoError(t, err)
		require.Contains(t, out, "Finalized into plan plan-final (pf).")
		require.Contains(t, out, "Persisted 1 phase(s) and 1 reference(s).")
	})
}

// TestAuthoringErrorHandling covers server error wrapping + missing positional.
func TestAuthoringErrorHandling(t *testing.T) {
	t.Run("server error is wrapped", func(t *testing.T) {
		rec := &authRecorder{err: connect.NewError(connect.CodeNotFound, errBoom())}
		app, groups := newAuthFixture(t, rec)
		_, err := clitest.RunCommand(t, clitest.FindCommand(t, groups, "author", "validate"), app, "sess-x")
		require.Error(t, err)
		require.Contains(t, err.Error(), "validate structure")
	})

	t.Run("missing required positional is a parser error", func(t *testing.T) {
		rec := &authRecorder{}
		app, groups := newAuthFixture(t, rec)
		_, err := clitest.RunCommand(t, clitest.FindCommand(t, groups, "author", "validate"), app)
		require.Error(t, err)
		require.Contains(t, err.Error(), "session")
		require.Nil(t, rec.lastRequest())
	})
}

// --- direct unit tests of the pure helpers ---

func TestParseSources(t *testing.T) {
	require.Nil(t, parseSources(""))
	require.Nil(t, parseSources("   "))
	require.Equal(t, []string{"a"}, parseSources("a"))
	require.Equal(t, []string{"a", "b", "c"}, parseSources(" a , b ,c "))
	// trailing/empty segments are dropped
	require.Equal(t, []string{"a", "b"}, parseSources("a,,b,"))
}

func TestNextLabel(t *testing.T) {
	require.Equal(t, "complete", nextLabel(""))
	require.Equal(t, "complete", nextLabel("   "))
	require.Equal(t, "purpose", nextLabel("purpose"))
}

func TestFormatSection(t *testing.T) {
	mandatoryEmpty := formatSection(&authoringv1.Section{Key: "scope", Label: "Scope", Mandatory: true})
	require.Equal(t, "[scope] Scope (mandatory, empty)", mandatoryEmpty)

	optionalFilled := formatSection(&authoringv1.Section{Key: "notes", Label: "Notes", Filled: true})
	require.Equal(t, "[notes] Notes (optional, filled)", optionalFilled)

	autofilled := formatSection(&authoringv1.Section{Key: "anchor", Label: "Anchor", Mandatory: true, Filled: true, Autofilled: true})
	require.Equal(t, "[anchor] Anchor (mandatory, autofilled)", autofilled)
}

func errBoom() error { return &boomError{} }

type boomError struct{}

func (*boomError) Error() string { return "boom" }
