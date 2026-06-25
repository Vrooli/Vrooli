package authoring

import (
	"context"
	"errors"
	"io"
	"log"
	"testing"

	internalauthoring "plan-manager/internal/authoring"
	internalplans "plan-manager/internal/plans"

	"connectrpc.com/connect"

	"github.com/stretchr/testify/require"

	authoringv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/authoring"
)

// fakeAuthoringService is a minimal stand-in for internalauthoring.Service.
type fakeAuthoringService struct {
	session    internalauthoring.Session
	section    internalauthoring.Section
	violations []internalauthoring.StructureViolation
	results    []internalauthoring.AutofillResult
	valid      bool
	complete   bool
	plan       internalplans.Plan
	err        error

	gotTitle      string
	gotSlug       string
	gotTemplateID string
	gotSessionID  string
	gotSectionKey internalauthoring.SectionKey
	gotContent    string
	gotSources    []internalauthoring.AutofillSource
}

func (f *fakeAuthoringService) StartSession(_ context.Context, title, slug, templateID string) (internalauthoring.Session, error) {
	f.gotTitle, f.gotSlug, f.gotTemplateID = title, slug, templateID
	return f.session, f.err
}

func (f *fakeAuthoringService) GetSection(_ context.Context, sessionID string, key internalauthoring.SectionKey) (internalauthoring.Section, error) {
	f.gotSessionID, f.gotSectionKey = sessionID, key
	return f.section, f.err
}

func (f *fakeAuthoringService) SubmitSection(_ context.Context, sessionID string, key internalauthoring.SectionKey, content string) (internalauthoring.Session, []internalauthoring.StructureViolation, error) {
	f.gotSessionID, f.gotSectionKey, f.gotContent = sessionID, key, content
	return f.session, f.violations, f.err
}

func (f *fakeAuthoringService) Next(_ context.Context, sessionID string) (internalauthoring.Section, bool, error) {
	f.gotSessionID = sessionID
	return f.section, f.complete, f.err
}

func (f *fakeAuthoringService) ValidateStructure(_ context.Context, sessionID string) (bool, []internalauthoring.StructureViolation, error) {
	f.gotSessionID = sessionID
	return f.valid, f.violations, f.err
}

func (f *fakeAuthoringService) Autofill(_ context.Context, sessionID string, sources []internalauthoring.AutofillSource) (internalauthoring.Session, []internalauthoring.AutofillResult, error) {
	f.gotSessionID, f.gotSources = sessionID, sources
	return f.session, f.results, f.err
}

func (f *fakeAuthoringService) Finalize(_ context.Context, sessionID string) (internalplans.Plan, error) {
	f.gotSessionID = sessionID
	return f.plan, f.err
}

var _ internalauthoring.Service = (*fakeAuthoringService)(nil)

func newAuthoringHandler(svc internalauthoring.Service) *connectHandler {
	return NewConnectHandler(Deps{Service: svc, Logger: log.New(io.Discard, "", 0)})
}

func TestStartSessionSuccess(t *testing.T) {
	svc := &fakeAuthoringService{session: internalauthoring.Session{ID: "s1", Title: "My Plan"}}
	h := newAuthoringHandler(svc)

	resp, err := h.StartSession(context.Background(), connect.NewRequest(&authoringv1.StartSessionRequest{
		Title:      "My Plan",
		Slug:       "my-plan",
		TemplateId: "cli",
	}))
	require.NoError(t, err)
	require.Equal(t, "s1", resp.Msg.GetSession().GetId())
	require.Equal(t, "My Plan", resp.Msg.GetSession().GetTitle())
	require.Equal(t, "My Plan", svc.gotTitle)
	require.Equal(t, "my-plan", svc.gotSlug)
	require.Equal(t, "cli", svc.gotTemplateID)
}

func TestGetSectionSuccess(t *testing.T) {
	svc := &fakeAuthoringService{section: internalauthoring.Section{Key: internalauthoring.SectionPurpose, Label: "Purpose", Content: "why"}}
	h := newAuthoringHandler(svc)

	resp, err := h.GetSection(context.Background(), connect.NewRequest(&authoringv1.GetSectionRequest{
		SessionId:  "s1",
		SectionKey: "purpose",
	}))
	require.NoError(t, err)
	require.Equal(t, "purpose", resp.Msg.GetSection().GetKey())
	require.Equal(t, "why", resp.Msg.GetSection().GetContent())
	require.Equal(t, "s1", svc.gotSessionID)
	require.Equal(t, internalauthoring.SectionKey("purpose"), svc.gotSectionKey, "handler must wrap the raw key in a SectionKey")
}

func TestSubmitSectionSuccess(t *testing.T) {
	svc := &fakeAuthoringService{
		session:    internalauthoring.Session{ID: "s1"},
		violations: []internalauthoring.StructureViolation{{SectionKey: internalauthoring.SectionPurpose, Message: "empty"}},
	}
	h := newAuthoringHandler(svc)

	resp, err := h.SubmitSection(context.Background(), connect.NewRequest(&authoringv1.SubmitSectionRequest{
		SessionId:  "s1",
		SectionKey: "purpose",
		Content:    "the purpose",
	}))
	require.NoError(t, err)
	require.Equal(t, "s1", resp.Msg.GetSession().GetId())
	require.Len(t, resp.Msg.GetViolations(), 1)
	require.Equal(t, "purpose", resp.Msg.GetViolations()[0].GetSectionKey())
	require.Equal(t, "the purpose", svc.gotContent)
}

func TestNextIncludesSectionWhenIncomplete(t *testing.T) {
	svc := &fakeAuthoringService{section: internalauthoring.Section{Key: internalauthoring.SectionScope, Label: "Scope"}, complete: false}
	h := newAuthoringHandler(svc)

	resp, err := h.Next(context.Background(), connect.NewRequest(&authoringv1.NextRequest{SessionId: "s1"}))
	require.NoError(t, err)
	require.False(t, resp.Msg.GetComplete())
	require.NotNil(t, resp.Msg.GetSection(), "an incomplete Next must carry the next section")
	require.Equal(t, "scope", resp.Msg.GetSection().GetKey())
}

func TestNextOmitsSectionWhenComplete(t *testing.T) {
	svc := &fakeAuthoringService{section: internalauthoring.Section{Key: internalauthoring.SectionScope}, complete: true}
	h := newAuthoringHandler(svc)

	resp, err := h.Next(context.Background(), connect.NewRequest(&authoringv1.NextRequest{SessionId: "s1"}))
	require.NoError(t, err)
	require.True(t, resp.Msg.GetComplete())
	require.Nil(t, resp.Msg.GetSection(), "a complete Next must not carry a section")
}

func TestValidateStructureSuccess(t *testing.T) {
	svc := &fakeAuthoringService{valid: false, violations: []internalauthoring.StructureViolation{
		{SectionKey: internalauthoring.SectionRegressionAnchor, Message: "no anchor"},
	}}
	h := newAuthoringHandler(svc)

	resp, err := h.ValidateStructure(context.Background(), connect.NewRequest(&authoringv1.ValidateStructureRequest{SessionId: "s1"}))
	require.NoError(t, err)
	require.False(t, resp.Msg.GetValid())
	require.Len(t, resp.Msg.GetViolations(), 1)
	require.Equal(t, "regression_anchor", resp.Msg.GetViolations()[0].GetSectionKey())
}

func TestAutofillSuccess(t *testing.T) {
	svc := &fakeAuthoringService{
		session: internalauthoring.Session{ID: "s1"},
		results: []internalauthoring.AutofillResult{
			{Source: internalauthoring.AutofillReferences, SectionKey: internalauthoring.SectionReferences, Degraded: true, Detail: "down"},
		},
	}
	h := newAuthoringHandler(svc)

	resp, err := h.Autofill(context.Background(), connect.NewRequest(&authoringv1.AutofillRequest{
		SessionId: "s1",
		Sources:   []string{"references"},
	}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.GetResults(), 1)
	require.True(t, resp.Msg.GetResults()[0].GetDegraded())
	require.Equal(t, []internalauthoring.AutofillSource{internalauthoring.AutofillReferences}, svc.gotSources)
}

func TestFinalizeSuccess(t *testing.T) {
	svc := &fakeAuthoringService{plan: internalplans.Plan{ID: "plan-1", Status: internalplans.PlanStatusDraft}}
	h := newAuthoringHandler(svc)

	resp, err := h.Finalize(context.Background(), connect.NewRequest(&authoringv1.FinalizeRequest{SessionId: "s1"}))
	require.NoError(t, err)
	require.Equal(t, "plan-1", resp.Msg.GetPlan().GetId())
	require.Equal(t, "s1", svc.gotSessionID)
}

// TestAuthoringErrorMapping asserts each authoring/plans sentinel maps to the
// documented Connect code (see internal/authoring/errors.go ToConnectError).
func TestAuthoringErrorMapping(t *testing.T) {
	t.Run("session_not_found_is_not_found", func(t *testing.T) {
		h := newAuthoringHandler(&fakeAuthoringService{err: internalauthoring.ErrSessionNotFound{ID: "x"}})
		_, err := h.GetSection(context.Background(), connect.NewRequest(&authoringv1.GetSectionRequest{SessionId: "x", SectionKey: "purpose"}))
		require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
	})
	t.Run("section_not_found_is_not_found", func(t *testing.T) {
		h := newAuthoringHandler(&fakeAuthoringService{err: internalauthoring.ErrSectionNotFound{SessionID: "s", SectionKey: "bogus"}})
		_, err := h.GetSection(context.Background(), connect.NewRequest(&authoringv1.GetSectionRequest{SessionId: "s", SectionKey: "bogus"}))
		require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
	})
	t.Run("invalid_session_is_invalid_argument", func(t *testing.T) {
		h := newAuthoringHandler(&fakeAuthoringService{err: internalauthoring.ErrInvalidSession{Reason: "title is required"}})
		_, err := h.StartSession(context.Background(), connect.NewRequest(&authoringv1.StartSessionRequest{}))
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	})
	t.Run("structure_gate_is_failed_precondition", func(t *testing.T) {
		h := newAuthoringHandler(&fakeAuthoringService{err: internalauthoring.ErrStructureGate{
			Violations: []internalauthoring.StructureViolation{{SectionKey: internalauthoring.SectionPurpose, Message: "empty"}},
		}})
		_, err := h.Finalize(context.Background(), connect.NewRequest(&authoringv1.FinalizeRequest{SessionId: "s1"}))
		require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	})
	t.Run("plan_invalid_is_invalid_argument", func(t *testing.T) {
		h := newAuthoringHandler(&fakeAuthoringService{err: internalplans.ErrInvalidPlan{Reason: "title is required"}})
		_, err := h.Finalize(context.Background(), connect.NewRequest(&authoringv1.FinalizeRequest{SessionId: "s1"}))
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	})
	t.Run("unknown_error_is_internal", func(t *testing.T) {
		h := newAuthoringHandler(&fakeAuthoringService{err: errors.New("boom")})
		_, err := h.Finalize(context.Background(), connect.NewRequest(&authoringv1.FinalizeRequest{SessionId: "s1"}))
		require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
	})
}
