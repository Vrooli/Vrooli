package authoring

import (
	"context"
	"encoding/json"
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
	return connect.NewResponse(&authoringv1.SubmitSectionResponse{
		Summary:  &authoringv1.AuthoringMutationSummary{ObjectKind: "section", ObjectId: req.Msg.GetSectionKey(), Summary: "submitted section"},
		Progress: &authoringv1.AuthoringProgress{SessionId: req.Msg.GetSessionId(), CurrentSectionKey: "next-key"},
	}), nil
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

func (r *authRecorder) SubmitRelevantContextItem(_ context.Context, req *connect.Request[authoringv1.SubmitRelevantContextItemRequest]) (*connect.Response[authoringv1.SubmitRelevantContextItemResponse], error) {
	r.record(req.Msg)
	if r.err != nil {
		return nil, r.err
	}
	if m, ok := r.resp.(*authoringv1.SubmitRelevantContextItemResponse); ok && m != nil {
		return connect.NewResponse(m), nil
	}
	return connect.NewResponse(&authoringv1.SubmitRelevantContextItemResponse{Item: req.Msg.GetItem()}), nil
}

func (r *authRecorder) ListRelevantContext(_ context.Context, req *connect.Request[authoringv1.ListRelevantContextRequest]) (*connect.Response[authoringv1.ListRelevantContextResponse], error) {
	r.record(req.Msg)
	if r.err != nil {
		return nil, r.err
	}
	if m, ok := r.resp.(*authoringv1.ListRelevantContextResponse); ok && m != nil {
		return connect.NewResponse(m), nil
	}
	return connect.NewResponse(&authoringv1.ListRelevantContextResponse{}), nil
}

func (r *authRecorder) GetSession(_ context.Context, req *connect.Request[authoringv1.GetSessionRequest]) (*connect.Response[authoringv1.GetSessionResponse], error) {
	r.record(req.Msg)
	if r.err != nil {
		return nil, r.err
	}
	if m, ok := r.resp.(*authoringv1.GetSessionResponse); ok && m != nil {
		return connect.NewResponse(m), nil
	}
	return connect.NewResponse(&authoringv1.GetSessionResponse{Session: &authoringv1.AuthoringSession{Id: req.Msg.GetSessionId()}}), nil
}

func (r *authRecorder) UpdateRelevantContextItem(_ context.Context, req *connect.Request[authoringv1.UpdateRelevantContextItemRequest]) (*connect.Response[authoringv1.UpdateRelevantContextItemResponse], error) {
	r.record(req.Msg)
	if r.err != nil {
		return nil, r.err
	}
	if m, ok := r.resp.(*authoringv1.UpdateRelevantContextItemResponse); ok && m != nil {
		return connect.NewResponse(m), nil
	}
	return connect.NewResponse(&authoringv1.UpdateRelevantContextItemResponse{Item: req.Msg.GetItem(), Summary: &authoringv1.AuthoringMutationSummary{ObjectKind: "context"}}), nil
}

func (r *authRecorder) RemoveRelevantContextItem(_ context.Context, req *connect.Request[authoringv1.RemoveRelevantContextItemRequest]) (*connect.Response[authoringv1.RemoveRelevantContextItemResponse], error) {
	r.record(req.Msg)
	if r.err != nil {
		return nil, r.err
	}
	if m, ok := r.resp.(*authoringv1.RemoveRelevantContextItemResponse); ok && m != nil {
		return connect.NewResponse(m), nil
	}
	return connect.NewResponse(&authoringv1.RemoveRelevantContextItemResponse{Summary: &authoringv1.AuthoringMutationSummary{ObjectKind: "context", ObjectId: req.Msg.GetItemId()}}), nil
}

func (r *authRecorder) DiscoverContextCandidates(_ context.Context, req *connect.Request[authoringv1.DiscoverContextCandidatesRequest]) (*connect.Response[authoringv1.DiscoverContextCandidatesResponse], error) {
	r.record(req.Msg)
	if r.err != nil {
		return nil, r.err
	}
	if m, ok := r.resp.(*authoringv1.DiscoverContextCandidatesResponse); ok && m != nil {
		return connect.NewResponse(m), nil
	}
	return connect.NewResponse(&authoringv1.DiscoverContextCandidatesResponse{}), nil
}

func (r *authRecorder) AcceptContextCandidate(_ context.Context, req *connect.Request[authoringv1.AcceptContextCandidateRequest]) (*connect.Response[authoringv1.AcceptContextCandidateResponse], error) {
	r.record(req.Msg)
	if r.err != nil {
		return nil, r.err
	}
	if m, ok := r.resp.(*authoringv1.AcceptContextCandidateResponse); ok && m != nil {
		return connect.NewResponse(m), nil
	}
	return connect.NewResponse(&authoringv1.AcceptContextCandidateResponse{Candidate: &authoringv1.ContextCandidate{Id: req.Msg.GetCandidateId(), Status: "accepted"}}), nil
}

func (r *authRecorder) RejectContextCandidate(_ context.Context, req *connect.Request[authoringv1.RejectContextCandidateRequest]) (*connect.Response[authoringv1.RejectContextCandidateResponse], error) {
	r.record(req.Msg)
	if r.err != nil {
		return nil, r.err
	}
	if m, ok := r.resp.(*authoringv1.RejectContextCandidateResponse); ok && m != nil {
		return connect.NewResponse(m), nil
	}
	return connect.NewResponse(&authoringv1.RejectContextCandidateResponse{Candidate: &authoringv1.ContextCandidate{Id: req.Msg.GetCandidateId(), Status: "rejected", RejectionReason: req.Msg.GetReason()}}), nil
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

func (r *authRecorder) MovePhase(_ context.Context, req *connect.Request[authoringv1.MovePhaseRequest]) (*connect.Response[authoringv1.MovePhaseResponse], error) {
	r.record(req.Msg)
	if r.err != nil {
		return nil, r.err
	}
	if m, ok := r.resp.(*authoringv1.MovePhaseResponse); ok && m != nil {
		return connect.NewResponse(m), nil
	}
	return connect.NewResponse(&authoringv1.MovePhaseResponse{
		Phase:    &authoringv1.PhaseDraft{Id: req.Msg.GetPhaseId(), Order: 1},
		Summary:  &authoringv1.AuthoringMutationSummary{ObjectKind: "phase", ObjectId: req.Msg.GetPhaseId(), Field: "order", Summary: "moved phase"},
		Progress: &authoringv1.AuthoringProgress{SessionId: req.Msg.GetSessionId()},
	}), nil
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
	return connect.NewResponse(&authoringv1.SubmitPhaseFieldResponse{
		Phase:    &authoringv1.PhaseDraft{Id: req.Msg.GetPhaseId()},
		Summary:  &authoringv1.AuthoringMutationSummary{ObjectKind: "phase", ObjectId: req.Msg.GetPhaseId(), Field: req.Msg.GetField(), Summary: "submitted phase field"},
		Progress: &authoringv1.AuthoringProgress{SessionId: req.Msg.GetSessionId()},
	}), nil
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
			name: "autofill maps session + comma-split default sources", cmd: "autofill",
			argv: []string{"sess-1", "--sources", "regression_anchor, references"},
			assert: func(t *testing.T, req proto.Message) {
				m := req.(*authoringv1.AutofillRequest)
				require.Equal(t, "sess-1", m.GetSessionId())
				require.Equal(t, []string{"regression_anchor", "references"}, m.GetSources())
			},
		},
		{
			name: "autofill can still request legacy required reading explicitly", cmd: "autofill",
			argv: []string{"sess-1", "--sources", "required_reading"},
			assert: func(t *testing.T, req proto.Message) {
				m := req.(*authoringv1.AutofillRequest)
				require.Equal(t, "sess-1", m.GetSessionId())
				require.Equal(t, []string{"required_reading"}, m.GetSources())
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
			name: "context-submit maps structured context flags", cmd: "context-submit",
			argv: []string{"sess-1", "--phase", "ph1", "--kind", "command", "--label", "Recall", "--reason", "prior work", "--instruction", "run recall", "--command", "search-hub query plan-manager", "--required", "--repeat", "phase_entry"},
			assert: func(t *testing.T, req proto.Message) {
				m := req.(*authoringv1.SubmitRelevantContextItemRequest)
				require.Equal(t, "sess-1", m.GetSessionId())
				require.Equal(t, "ph1", m.GetPhaseId())
				require.Equal(t, sharedv1.RelevantContextKind_RELEVANT_CONTEXT_KIND_COMMAND, m.GetItem().GetKind())
				require.Equal(t, "Recall", m.GetItem().GetLabel())
				require.Equal(t, []string{"search-hub", "query", "plan-manager"}, m.GetItem().GetArgv())
				require.True(t, m.GetItem().GetRequired())
				require.Equal(t, sharedv1.RelevantContextRepeatPolicy_RELEVANT_CONTEXT_REPEAT_POLICY_PHASE_ENTRY, m.GetItem().GetRepeatPolicy())
			},
		},
		{
			name: "context-submit parses quoted command argv", cmd: "context-submit",
			argv: []string{"sess-1", "--kind", "command", "--label", "Recall", "--command", "search-hub query 'shared drift hygiene' --type record,doc --json"},
			assert: func(t *testing.T, req proto.Message) {
				m := req.(*authoringv1.SubmitRelevantContextItemRequest)
				require.Equal(t, []string{"search-hub", "query", "shared drift hygiene", "--type", "record,doc", "--json"}, m.GetItem().GetArgv())
			},
		},
		{
			name: "context-submit prefers argv-json", cmd: "context-submit",
			argv: []string{"sess-1", "--kind", "command", "--label", "Recall", "--command", "ignored fallback", "--argv-json", `["search-hub","query","shared drift hygiene","--type","record,doc","--json"]`},
			assert: func(t *testing.T, req proto.Message) {
				m := req.(*authoringv1.SubmitRelevantContextItemRequest)
				require.Equal(t, []string{"search-hub", "query", "shared drift hygiene", "--type", "record,doc", "--json"}, m.GetItem().GetArgv())
			},
		},
		{
			name: "context-submit without --repeat sends UNSPECIFIED so the server picks the scope default", cmd: "context-submit",
			argv: []string{"sess-1", "--phase", "ph1", "--kind", "skill", "--label", "Steer", "--reason", "phase setup", "--instruction", "load it", "--target", "api-steer"},
			assert: func(t *testing.T, req proto.Message) {
				m := req.(*authoringv1.SubmitRelevantContextItemRequest)
				require.Equal(t, sharedv1.RelevantContextRepeatPolicy_RELEVANT_CONTEXT_REPEAT_POLICY_UNSPECIFIED, m.GetItem().GetRepeatPolicy(),
					"an unset --repeat must not hard-code once_per_execution; the server applies the scope-appropriate default")
			},
		},
		{
			name: "context-list maps session and phase", cmd: "context-list",
			argv: []string{"sess-1", "--phase", "ph1"},
			assert: func(t *testing.T, req proto.Message) {
				m := req.(*authoringv1.ListRelevantContextRequest)
				require.Equal(t, "sess-1", m.GetSessionId())
				require.Equal(t, "ph1", m.GetPhaseId())
			},
		},
		{
			name: "context-discover maps concepts and complexity", cmd: "context-discover",
			argv: []string{"sess-1", "--concepts", "a,b", "--complexity", "architectural"},
			assert: func(t *testing.T, req proto.Message) {
				m := req.(*authoringv1.DiscoverContextCandidatesRequest)
				require.Equal(t, "sess-1", m.GetSessionId())
				require.Equal(t, []string{"a", "b"}, m.GetConcepts())
				require.Equal(t, "architectural", m.GetComplexity())
			},
		},
		{
			name: "context-accept maps candidate and phase", cmd: "context-accept",
			argv: []string{"sess-1", "cand1", "--phase", "ph1"},
			assert: func(t *testing.T, req proto.Message) {
				m := req.(*authoringv1.AcceptContextCandidateRequest)
				require.Equal(t, "sess-1", m.GetSessionId())
				require.Equal(t, "cand1", m.GetCandidateId())
				require.Equal(t, "ph1", m.GetPhaseId())
			},
		},
		{
			name: "context-reject maps candidate and reason", cmd: "context-reject",
			argv: []string{"sess-1", "cand1", "--reason", "duplicate"},
			assert: func(t *testing.T, req proto.Message) {
				m := req.(*authoringv1.RejectContextCandidateRequest)
				require.Equal(t, "sess-1", m.GetSessionId())
				require.Equal(t, "cand1", m.GetCandidateId())
				require.Equal(t, "duplicate", m.GetReason())
			},
		},
		{
			name: "get-session maps session positional", cmd: "get-session",
			argv: []string{"sess-1"},
			assert: func(t *testing.T, req proto.Message) {
				require.Equal(t, "sess-1", req.(*authoringv1.GetSessionRequest).GetSessionId())
			},
		},
		{
			name: "context-update maps session/item/phase and structured flags", cmd: "context-update",
			argv: []string{"sess-1", "ctx-9", "--phase", "ph1", "--kind", "note", "--label", "fixed", "--instruction", "do the right thing"},
			assert: func(t *testing.T, req proto.Message) {
				m := req.(*authoringv1.UpdateRelevantContextItemRequest)
				require.Equal(t, "sess-1", m.GetSessionId())
				require.Equal(t, "ctx-9", m.GetItemId())
				require.Equal(t, "ph1", m.GetPhaseId())
				require.Equal(t, "ctx-9", m.GetItem().GetId())
				require.Equal(t, sharedv1.RelevantContextKind_RELEVANT_CONTEXT_KIND_NOTE, m.GetItem().GetKind())
				require.Equal(t, "fixed", m.GetItem().GetLabel())
			},
		},
		{
			name: "context-remove maps session/item/phase", cmd: "context-remove",
			argv: []string{"sess-1", "ctx-9", "--phase", "ph1"},
			assert: func(t *testing.T, req proto.Message) {
				m := req.(*authoringv1.RemoveRelevantContextItemRequest)
				require.Equal(t, "sess-1", m.GetSessionId())
				require.Equal(t, "ctx-9", m.GetItemId())
				require.Equal(t, "ph1", m.GetPhaseId())
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
			name: "phase-move maps before target", cmd: "phase-move",
			argv: []string{"sess-1", "ph3", "--before", "ph2"},
			assert: func(t *testing.T, req proto.Message) {
				m := req.(*authoringv1.MovePhaseRequest)
				require.Equal(t, "sess-1", m.GetSessionId())
				require.Equal(t, "ph3", m.GetPhaseId())
				require.Equal(t, "ph2", m.GetBeforePhaseId())
				require.Empty(t, m.GetAfterPhaseId())
			},
		},
		{
			name: "phase-move maps after target", cmd: "phase-move",
			argv: []string{"sess-1", "ph1", "--after", "ph3"},
			assert: func(t *testing.T, req proto.Message) {
				m := req.(*authoringv1.MovePhaseRequest)
				require.Equal(t, "sess-1", m.GetSessionId())
				require.Equal(t, "ph1", m.GetPhaseId())
				require.Empty(t, m.GetBeforePhaseId())
				require.Equal(t, "ph3", m.GetAfterPhaseId())
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
			Id: "sess-1", PlanSlug: "my-plan", Title: "My Plan", CurrentSectionKey: "purpose",
			Sections: []*authoringv1.Section{{Key: "purpose"}, {Key: "scope"}},
		}}}
		app, groups := newAuthFixture(t, rec)
		out, err := clitest.RunCommand(t, clitest.FindCommand(t, groups, "author", "start"), app, "--title", "My Plan")
		require.NoError(t, err)
		require.Contains(t, out, `Started session my-plan for "My Plan".`)
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
			Id: "plan-final", Slug: "pf", Title: "Readable Plan", Status: sharedv1.PlanStatus_PLAN_STATUS_DRAFT,
			Mirror:     &sharedv1.RenderedPlanMirror{RelativePath: "pf.md", Status: sharedv1.RenderedMirrorStatus_RENDERED_MIRROR_STATUS_FRESH},
			Phases:     []*sharedv1.Phase{{Id: "ph1"}},
			References: []*sharedv1.Reference{{Id: "r1"}},
		}}}
		app, groups := newAuthFixture(t, rec)
		out, err := clitest.RunCommand(t, clitest.FindCommand(t, groups, "author", "finalize"), app, "sess-1")
		require.NoError(t, err)
		require.Contains(t, out, "Finalized plan pf.")
		require.Contains(t, out, "id: plan-final")
		require.Contains(t, out, "slug: pf")
		require.Contains(t, out, "title: Readable Plan")
		require.Contains(t, out, "status: draft")
		require.Contains(t, out, "mirror_status: fresh")
		require.Contains(t, out, "mirror_path: pf.md")
		require.NotContains(t, out, "phases: 1")

		full, err := clitest.RunCommand(t, clitest.FindCommand(t, groups, "author", "finalize"), app, "sess-1", "--full")
		require.NoError(t, err)
		require.Contains(t, full, "phases: 1")
		require.Contains(t, full, "references: 1")

		compactJSON, err := clitest.RunCommand(t, clitest.FindCommand(t, groups, "author", "finalize"), app, "sess-1", "--json")
		require.NoError(t, err)
		var compact map[string]any
		require.NoError(t, json.Unmarshal([]byte(compactJSON), &compact))
		require.Equal(t, "plan-final", compact["id"])
		require.Equal(t, "pf", compact["slug"])
		require.Equal(t, "Readable Plan", compact["title"])
		require.Equal(t, "draft", compact["status"])
		require.Equal(t, "fresh", compact["mirror_status"])
		require.Equal(t, "pf.md", compact["mirror_path"])
		require.NotContains(t, compact, "plan")

		fullJSON, err := clitest.RunCommand(t, clitest.FindCommand(t, groups, "author", "finalize"), app, "sess-1", "--full", "--json")
		require.NoError(t, err)
		var fullPayload map[string]any
		require.NoError(t, json.Unmarshal([]byte(fullJSON), &fullPayload))
		require.Contains(t, fullPayload, "plan")
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
