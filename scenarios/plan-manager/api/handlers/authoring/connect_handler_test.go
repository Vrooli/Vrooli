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
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/shared"
)

// fakeAuthoringService is a minimal stand-in for internalauthoring.Service.
type fakeAuthoringService struct {
	session          internalauthoring.Session
	section          internalauthoring.Section
	violations       []internalauthoring.StructureViolation
	results          []internalauthoring.AutofillResult
	phase            internalauthoring.PhaseDraft
	contextItem      internalplans.RelevantContextItem
	contextItems     []internalplans.RelevantContextItem
	candidate        internalauthoring.ContextCandidate
	candidates       []internalauthoring.ContextCandidate
	disposition      internalauthoring.ContextDispositionSummary
	refCandidate     internalauthoring.ReferenceCandidate
	refCandidates    []internalauthoring.ReferenceCandidate
	refDisposition   internalauthoring.ReferenceDispositionSummary
	step             internalauthoring.GuidedStep
	valid            bool
	complete         bool
	plan             internalplans.Plan
	previewMarkdown  string
	alreadyFinalized bool
	finalizedAt      string
	storePath        string
	err              error

	gotTitle         string
	gotSlug          string
	gotTemplateID    string
	gotSessionID     string
	gotWorkspaceRoot string
	gotFieldWrites   []internalauthoring.FieldWrite
	fieldResults     []internalauthoring.FieldWriteResult
	gotSectionKey    internalauthoring.SectionKey
	gotContent       string
	gotSources       []internalauthoring.AutofillSource
	gotPhaseID       string
	gotBeforePhase   string
	gotAfterPhase    string
	gotPhaseField    internalauthoring.PhaseField
	gotContextItem   internalplans.RelevantContextItem
	gotItemID        string
	gotCandidateID   string
	gotBatchID       string
	gotTakes         []internalauthoring.ContextDispositionTake
	gotDrops         []internalauthoring.ContextDispositionDrop
	gotRefTakes      []internalauthoring.ReferenceDispositionTake
	gotRefDrops      []internalauthoring.ReferenceDispositionDrop
	gotSweepNote     string
	gotTakeAll       bool
	gotConcepts      []string
	gotComplexity    string
	gotRefresh       bool
	gotReason        string
}

func (f *fakeAuthoringService) StartSession(_ context.Context, title, slug, templateID string) (internalauthoring.Session, internalauthoring.GuidedStep, error) {
	f.gotTitle, f.gotSlug, f.gotTemplateID = title, slug, templateID
	return f.session, f.step, f.err
}

func (f *fakeAuthoringService) GetSession(_ context.Context, sessionID string) (internalauthoring.Session, internalauthoring.GuidedStep, error) {
	f.gotSessionID = sessionID
	return f.session, f.step, f.err
}

func (f *fakeAuthoringService) GetSection(_ context.Context, sessionID string, key internalauthoring.SectionKey) (internalauthoring.Section, internalauthoring.GuidedStep, error) {
	f.gotSessionID, f.gotSectionKey = sessionID, key
	return f.section, f.step, f.err
}

func (f *fakeAuthoringService) SubmitFields(_ context.Context, sessionID string, writes []internalauthoring.FieldWrite) (internalauthoring.Session, []internalauthoring.FieldWriteResult, internalauthoring.GuidedStep, error) {
	f.gotSessionID = sessionID
	f.gotFieldWrites = append([]internalauthoring.FieldWrite(nil), writes...)
	return f.session, f.fieldResults, f.step, f.err
}

func (f *fakeAuthoringService) SubmitSection(_ context.Context, sessionID string, key internalauthoring.SectionKey, content string) (internalauthoring.Session, []internalauthoring.StructureViolation, internalauthoring.GuidedStep, error) {
	f.gotSessionID, f.gotSectionKey, f.gotContent = sessionID, key, content
	return f.session, f.violations, f.step, f.err
}

func (f *fakeAuthoringService) Next(_ context.Context, sessionID string) (internalauthoring.Section, internalauthoring.GuidedStep, bool, error) {
	f.gotSessionID = sessionID
	return f.section, f.step, f.complete, f.err
}

func (f *fakeAuthoringService) ContinueAuthoring(_ context.Context, sessionID string) (internalauthoring.Session, internalauthoring.Section, internalauthoring.PhaseDraft, bool, []internalauthoring.StructureViolation, internalauthoring.GuidedStep, error) {
	f.gotSessionID = sessionID
	return f.session, f.section, f.phase, f.valid, f.violations, f.step, f.err
}

func (f *fakeAuthoringService) ValidateStructure(_ context.Context, sessionID string) (bool, []internalauthoring.StructureViolation, internalauthoring.GuidedStep, error) {
	f.gotSessionID = sessionID
	return f.valid, f.violations, f.step, f.err
}

func (f *fakeAuthoringService) Autofill(_ context.Context, sessionID string, sources []internalauthoring.AutofillSource) (internalauthoring.Session, []internalauthoring.AutofillResult, internalauthoring.GuidedStep, error) {
	f.gotSessionID, f.gotSources = sessionID, sources
	return f.session, f.results, f.step, f.err
}

func (f *fakeAuthoringService) SubmitRelevantContextItem(_ context.Context, sessionID, phaseID string, item internalplans.RelevantContextItem) (internalauthoring.Session, internalplans.RelevantContextItem, []internalauthoring.StructureViolation, internalauthoring.GuidedStep, error) {
	f.gotSessionID, f.gotPhaseID, f.gotContextItem = sessionID, phaseID, item
	return f.session, f.contextItem, f.violations, f.step, f.err
}

func (f *fakeAuthoringService) ListRelevantContext(_ context.Context, sessionID, phaseID string) ([]internalplans.RelevantContextItem, internalauthoring.GuidedStep, error) {
	f.gotSessionID, f.gotPhaseID = sessionID, phaseID
	return f.contextItems, f.step, f.err
}

func (f *fakeAuthoringService) UpdateRelevantContextItem(_ context.Context, sessionID, phaseID, itemID string, item internalplans.RelevantContextItem) (internalauthoring.Session, internalplans.RelevantContextItem, []internalauthoring.StructureViolation, internalauthoring.GuidedStep, error) {
	f.gotSessionID, f.gotPhaseID, f.gotItemID, f.gotContextItem = sessionID, phaseID, itemID, item
	return f.session, f.contextItem, f.violations, f.step, f.err
}

func (f *fakeAuthoringService) RemoveRelevantContextItem(_ context.Context, sessionID, phaseID, itemID string) (internalauthoring.Session, []internalauthoring.StructureViolation, internalauthoring.GuidedStep, error) {
	f.gotSessionID, f.gotPhaseID, f.gotItemID = sessionID, phaseID, itemID
	return f.session, f.violations, f.step, f.err
}

func (f *fakeAuthoringService) DiscoverContextCandidates(_ context.Context, sessionID string, concepts []string, complexity string, refresh bool) (internalauthoring.Session, []internalauthoring.ContextCandidate, internalauthoring.GuidedStep, error) {
	f.gotSessionID, f.gotConcepts, f.gotComplexity, f.gotRefresh = sessionID, append([]string(nil), concepts...), complexity, refresh
	return f.session, f.candidates, f.step, f.err
}

func (f *fakeAuthoringService) AcceptContextCandidate(_ context.Context, sessionID, candidateID, phaseID string) (internalauthoring.Session, internalauthoring.ContextCandidate, internalplans.RelevantContextItem, []internalauthoring.StructureViolation, internalauthoring.GuidedStep, error) {
	f.gotSessionID, f.gotCandidateID, f.gotPhaseID = sessionID, candidateID, phaseID
	return f.session, f.candidate, f.contextItem, f.violations, f.step, f.err
}

func (f *fakeAuthoringService) RejectContextCandidate(_ context.Context, sessionID, candidateID, reason string) (internalauthoring.Session, internalauthoring.ContextCandidate, internalauthoring.GuidedStep, error) {
	f.gotSessionID, f.gotCandidateID, f.gotReason = sessionID, candidateID, reason
	return f.session, f.candidate, f.step, f.err
}

func (f *fakeAuthoringService) ApplyContextDisposition(_ context.Context, sessionID, batchID string, takes []internalauthoring.ContextDispositionTake, drops []internalauthoring.ContextDispositionDrop, sweepNote string, takeAll bool) (internalauthoring.Session, internalauthoring.ContextDispositionSummary, []internalauthoring.StructureViolation, internalauthoring.GuidedStep, error) {
	f.gotSessionID, f.gotBatchID = sessionID, batchID
	f.gotTakes, f.gotDrops = append([]internalauthoring.ContextDispositionTake(nil), takes...), append([]internalauthoring.ContextDispositionDrop(nil), drops...)
	f.gotSweepNote, f.gotTakeAll = sweepNote, takeAll
	return f.session, f.disposition, f.violations, f.step, f.err
}

func (f *fakeAuthoringService) SuggestReferences(_ context.Context, sessionID string) (internalauthoring.Session, []internalauthoring.ReferenceCandidate, internalauthoring.GuidedStep, error) {
	f.gotSessionID = sessionID
	return f.session, f.refCandidates, f.step, f.err
}

func (f *fakeAuthoringService) ListReferenceCandidates(_ context.Context, sessionID string) ([]internalauthoring.ReferenceCandidate, internalauthoring.GuidedStep, error) {
	f.gotSessionID = sessionID
	return f.refCandidates, f.step, f.err
}

func (f *fakeAuthoringService) AcceptReferenceCandidate(_ context.Context, sessionID, candidateID string, edit *internalplans.Reference) (internalauthoring.Session, internalauthoring.ReferenceCandidate, []internalauthoring.StructureViolation, internalauthoring.GuidedStep, error) {
	f.gotSessionID, f.gotCandidateID = sessionID, candidateID
	return f.session, f.refCandidate, f.violations, f.step, f.err
}

func (f *fakeAuthoringService) RejectReferenceCandidate(_ context.Context, sessionID, candidateID, reason string) (internalauthoring.Session, internalauthoring.ReferenceCandidate, internalauthoring.GuidedStep, error) {
	f.gotSessionID, f.gotCandidateID, f.gotReason = sessionID, candidateID, reason
	return f.session, f.refCandidate, f.step, f.err
}

func (f *fakeAuthoringService) ApplyReferenceDisposition(_ context.Context, sessionID, batchID string, takes []internalauthoring.ReferenceDispositionTake, drops []internalauthoring.ReferenceDispositionDrop, sweepNote string, takeAll bool) (internalauthoring.Session, internalauthoring.ReferenceDispositionSummary, []internalauthoring.StructureViolation, internalauthoring.GuidedStep, error) {
	f.gotSessionID, f.gotBatchID = sessionID, batchID
	f.gotRefTakes, f.gotRefDrops = append([]internalauthoring.ReferenceDispositionTake(nil), takes...), append([]internalauthoring.ReferenceDispositionDrop(nil), drops...)
	f.gotSweepNote, f.gotTakeAll = sweepNote, takeAll
	return f.session, f.refDisposition, f.violations, f.step, f.err
}

func (f *fakeAuthoringService) AddPhase(_ context.Context, sessionID string, title, intent string) (internalauthoring.Session, internalauthoring.PhaseDraft, []internalauthoring.StructureViolation, internalauthoring.GuidedStep, error) {
	f.gotSessionID, f.gotContent = sessionID, title+"|"+intent
	return f.session, f.phase, f.violations, f.step, f.err
}

func (f *fakeAuthoringService) MovePhase(_ context.Context, sessionID, phaseID, beforePhaseID, afterPhaseID string) (internalauthoring.Session, internalauthoring.PhaseDraft, []internalauthoring.StructureViolation, internalauthoring.GuidedStep, error) {
	f.gotSessionID, f.gotPhaseID, f.gotBeforePhase, f.gotAfterPhase = sessionID, phaseID, beforePhaseID, afterPhaseID
	return f.session, f.phase, f.violations, f.step, f.err
}

func (f *fakeAuthoringService) GetPhase(_ context.Context, sessionID, phaseID string) (internalauthoring.PhaseDraft, internalauthoring.GuidedStep, error) {
	f.gotSessionID, f.gotPhaseID = sessionID, phaseID
	return f.phase, f.step, f.err
}

func (f *fakeAuthoringService) SubmitPhaseField(_ context.Context, sessionID, phaseID string, field internalauthoring.PhaseField, content string) (internalauthoring.Session, []internalauthoring.StructureViolation, internalauthoring.GuidedStep, error) {
	f.gotSessionID, f.gotPhaseID, f.gotPhaseField, f.gotContent = sessionID, phaseID, field, content
	return f.session, f.violations, f.step, f.err
}

func (f *fakeAuthoringService) NextPhase(_ context.Context, sessionID string) (internalauthoring.PhaseDraft, internalauthoring.GuidedStep, bool, error) {
	f.gotSessionID = sessionID
	return f.phase, f.step, f.complete, f.err
}

func (f *fakeAuthoringService) PreviewPlan(_ context.Context, sessionID string) (string, internalauthoring.GuidedStep, error) {
	f.gotSessionID = sessionID
	return f.previewMarkdown, f.step, f.err
}

func (f *fakeAuthoringService) Finalize(_ context.Context, sessionID string, opts internalauthoring.FinalizeOptions) (internalauthoring.FinalizeResult, internalauthoring.GuidedStep, error) {
	f.gotSessionID = sessionID
	f.gotWorkspaceRoot = opts.WorkspaceRoot
	return internalauthoring.FinalizeResult{
		Plan:             f.plan,
		Mirror:           f.plan.Mirror,
		AlreadyFinalized: f.alreadyFinalized,
		FinalizedAt:      f.finalizedAt,
		StorePath:        f.storePath,
	}, f.step, f.err
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

func TestGetSessionReturnsFullState(t *testing.T) {
	svc := &fakeAuthoringService{session: internalauthoring.Session{ID: "s1", Title: "My Plan"}}
	h := newAuthoringHandler(svc)

	resp, err := h.GetSession(context.Background(), connect.NewRequest(&authoringv1.GetSessionRequest{SessionId: "s1"}))
	require.NoError(t, err)
	require.Equal(t, "s1", resp.Msg.GetSession().GetId())
	require.Equal(t, "My Plan", resp.Msg.GetSession().GetTitle())
	require.Equal(t, "s1", svc.gotSessionID)
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
	// Focused contract: no full session is echoed; the response carries a
	// mutation summary + compact progress instead.
	require.Equal(t, "s1", resp.Msg.GetProgress().GetSessionId())
	require.Equal(t, "section", resp.Msg.GetSummary().GetObjectKind())
	require.Equal(t, "purpose", resp.Msg.GetSummary().GetObjectId())
	require.Len(t, resp.Msg.GetViolations(), 1)
	require.Equal(t, "purpose", resp.Msg.GetViolations()[0].GetSectionKey())
	require.Equal(t, "the purpose", svc.gotContent)
}

func TestNextIncludesSectionWhenIncomplete(t *testing.T) {
	svc := &fakeAuthoringService{
		section:  internalauthoring.Section{Key: internalauthoring.SectionScope, Label: "Scope"},
		step:     internalauthoring.GuidedStep{StepKind: "scope", Summary: "Draw the boundary."},
		complete: false,
	}
	h := newAuthoringHandler(svc)

	resp, err := h.Next(context.Background(), connect.NewRequest(&authoringv1.NextRequest{SessionId: "s1"}))
	require.NoError(t, err)
	require.False(t, resp.Msg.GetComplete())
	require.NotNil(t, resp.Msg.GetSection(), "an incomplete Next must carry the next section")
	require.Equal(t, "scope", resp.Msg.GetSection().GetKey())
	require.Equal(t, "scope", resp.Msg.GetStep().GetStepKind())
}

func TestNextOmitsSectionWhenComplete(t *testing.T) {
	svc := &fakeAuthoringService{section: internalauthoring.Section{Key: internalauthoring.SectionScope}, complete: true}
	h := newAuthoringHandler(svc)

	resp, err := h.Next(context.Background(), connect.NewRequest(&authoringv1.NextRequest{SessionId: "s1"}))
	require.NoError(t, err)
	require.True(t, resp.Msg.GetComplete())
	require.Nil(t, resp.Msg.GetSection(), "a complete Next must not carry a section")
}

func TestRelevantContextHandlers(t *testing.T) {
	t.Run("submit context item", func(t *testing.T) {
		svc := &fakeAuthoringService{
			session: internalauthoring.Session{ID: "s1"},
			contextItem: internalplans.RelevantContextItem{
				ID:           "ctx1",
				Kind:         internalplans.RelevantContextCommand,
				Scope:        internalplans.RelevantContextScopePhase,
				PhaseID:      "ph1",
				Label:        "Recall records",
				Command:      "search-hub query plan-manager --type record",
				Required:     true,
				RepeatPolicy: internalplans.RelevantContextPhaseEntry,
				Source:       internalplans.RelevantContextSourceAuthored,
				Status:       internalplans.RelevantContextStatusReady,
			},
		}
		h := newAuthoringHandler(svc)
		resp, err := h.SubmitRelevantContextItem(context.Background(), connect.NewRequest(&authoringv1.SubmitRelevantContextItemRequest{
			SessionId: "s1",
			PhaseId:   "ph1",
			Item: &sharedv1.RelevantContextItem{
				Kind:    sharedv1.RelevantContextKind_RELEVANT_CONTEXT_KIND_COMMAND,
				Label:   "Recall records",
				Command: "search-hub query plan-manager --type record",
			},
		}))
		require.NoError(t, err)
		require.Equal(t, "ctx1", resp.Msg.GetItem().GetId())
		require.Equal(t, sharedv1.RelevantContextKind_RELEVANT_CONTEXT_KIND_COMMAND, resp.Msg.GetItem().GetKind())
		require.Equal(t, internalplans.RelevantContextCommand, svc.gotContextItem.Kind)
		require.Equal(t, "ph1", svc.gotPhaseID)
	})

	t.Run("list context items", func(t *testing.T) {
		svc := &fakeAuthoringService{
			contextItems: []internalplans.RelevantContextItem{{
				ID:     "ctx1",
				Kind:   internalplans.RelevantContextDoc,
				Scope:  internalplans.RelevantContextScopeGlobal,
				Target: "docs/concepts/PLAN-MODEL.md",
			}},
		}
		h := newAuthoringHandler(svc)
		resp, err := h.ListRelevantContext(context.Background(), connect.NewRequest(&authoringv1.ListRelevantContextRequest{
			SessionId: "s1",
		}))
		require.NoError(t, err)
		require.Len(t, resp.Msg.GetItems(), 1)
		require.Equal(t, "docs/concepts/PLAN-MODEL.md", resp.Msg.GetItems()[0].GetTarget())
		require.Equal(t, "s1", svc.gotSessionID)
	})

	t.Run("update context item", func(t *testing.T) {
		svc := &fakeAuthoringService{
			session: internalauthoring.Session{ID: "s1"},
			contextItem: internalplans.RelevantContextItem{
				ID:    "ctx1",
				Kind:  internalplans.RelevantContextDoc,
				Scope: internalplans.RelevantContextScopeGlobal,
				Label: "Updated label",
			},
		}
		h := newAuthoringHandler(svc)
		resp, err := h.UpdateRelevantContextItem(context.Background(), connect.NewRequest(&authoringv1.UpdateRelevantContextItemRequest{
			SessionId: "s1",
			ItemId:    "ctx1",
			Item:      &sharedv1.RelevantContextItem{Kind: sharedv1.RelevantContextKind_RELEVANT_CONTEXT_KIND_DOC, Label: "Updated label"},
		}))
		require.NoError(t, err)
		require.Equal(t, "ctx1", resp.Msg.GetItem().GetId())
		require.Equal(t, "context", resp.Msg.GetSummary().GetObjectKind())
		require.Equal(t, "s1", resp.Msg.GetProgress().GetSessionId())
		require.Equal(t, "ctx1", svc.gotItemID)
	})

	t.Run("remove context item", func(t *testing.T) {
		svc := &fakeAuthoringService{session: internalauthoring.Session{ID: "s1"}}
		h := newAuthoringHandler(svc)
		resp, err := h.RemoveRelevantContextItem(context.Background(), connect.NewRequest(&authoringv1.RemoveRelevantContextItemRequest{
			SessionId: "s1",
			PhaseId:   "ph1",
			ItemId:    "ctx1",
		}))
		require.NoError(t, err)
		require.Equal(t, "context", resp.Msg.GetSummary().GetObjectKind())
		require.Equal(t, "ctx1", resp.Msg.GetSummary().GetObjectId())
		require.Equal(t, "s1", resp.Msg.GetProgress().GetSessionId())
		require.Equal(t, "ph1", svc.gotPhaseID)
		require.Equal(t, "ctx1", svc.gotItemID)
	})

	t.Run("discover context candidates", func(t *testing.T) {
		svc := &fakeAuthoringService{
			session: internalauthoring.Session{ID: "s1"},
			candidates: []internalauthoring.ContextCandidate{{
				ID:      "cand1",
				Concept: "plan-manager context",
				Source:  "search-hub-recall",
				Status:  internalauthoring.ContextCandidatePending,
				Item: internalplans.RelevantContextItem{
					ID:      "ctx1",
					Kind:    internalplans.RelevantContextSearch,
					Command: "search-hub query plan-manager --type record,skill,doc",
				},
			}},
		}
		h := newAuthoringHandler(svc)
		resp, err := h.DiscoverContextCandidates(context.Background(), connect.NewRequest(&authoringv1.DiscoverContextCandidatesRequest{
			SessionId:  "s1",
			Concepts:   []string{"plan-manager context"},
			Complexity: "architectural",
		}))
		require.NoError(t, err)
		require.Len(t, resp.Msg.GetCandidates(), 1)
		require.Equal(t, "cand1", resp.Msg.GetCandidates()[0].GetId())
		require.Equal(t, []string{"plan-manager context"}, svc.gotConcepts)
		require.Equal(t, "architectural", svc.gotComplexity)
	})

	t.Run("accept context candidate", func(t *testing.T) {
		svc := &fakeAuthoringService{
			session: internalauthoring.Session{ID: "s1"},
			candidate: internalauthoring.ContextCandidate{
				ID:     "cand1",
				Status: internalauthoring.ContextCandidateAccepted,
			},
			contextItem: internalplans.RelevantContextItem{
				ID:      "ctx1",
				Kind:    internalplans.RelevantContextSearch,
				Scope:   internalplans.RelevantContextScopePhase,
				PhaseID: "ph1",
			},
		}
		h := newAuthoringHandler(svc)
		resp, err := h.AcceptContextCandidate(context.Background(), connect.NewRequest(&authoringv1.AcceptContextCandidateRequest{
			SessionId:   "s1",
			CandidateId: "cand1",
			PhaseId:     "ph1",
		}))
		require.NoError(t, err)
		require.Equal(t, "ctx1", resp.Msg.GetItem().GetId())
		require.Equal(t, "cand1", svc.gotCandidateID)
		require.Equal(t, "ph1", svc.gotPhaseID)
	})

	t.Run("reject context candidate", func(t *testing.T) {
		svc := &fakeAuthoringService{
			session: internalauthoring.Session{ID: "s1"},
			candidate: internalauthoring.ContextCandidate{
				ID:              "cand1",
				Status:          internalauthoring.ContextCandidateRejected,
				RejectionReason: "duplicate",
			},
		}
		h := newAuthoringHandler(svc)
		resp, err := h.RejectContextCandidate(context.Background(), connect.NewRequest(&authoringv1.RejectContextCandidateRequest{
			SessionId:   "s1",
			CandidateId: "cand1",
			Reason:      "duplicate",
		}))
		require.NoError(t, err)
		require.Equal(t, "duplicate", resp.Msg.GetCandidate().GetRejectionReason())
		require.Equal(t, "duplicate", svc.gotReason)
	})
}

func TestPhaseNativeHandlers(t *testing.T) {
	t.Run("add phase", func(t *testing.T) {
		svc := &fakeAuthoringService{
			session: internalauthoring.Session{ID: "s1"},
			phase:   internalauthoring.PhaseDraft{ID: "ph1", Order: 1, Title: "Contract"},
			step:    internalauthoring.GuidedStep{StepKind: "phase_references"},
		}
		h := newAuthoringHandler(svc)
		resp, err := h.AddPhase(context.Background(), connect.NewRequest(&authoringv1.AddPhaseRequest{
			SessionId: "s1",
			Title:     "Contract",
			Intent:    "Add RPCs",
		}))
		require.NoError(t, err)
		require.Equal(t, "ph1", resp.Msg.GetPhase().GetId())
		require.Equal(t, "s1", svc.gotSessionID)
		require.Equal(t, "Contract|Add RPCs", svc.gotContent)
	})

	t.Run("submit phase field", func(t *testing.T) {
		svc := &fakeAuthoringService{
			// The session carries the changed phase draft so the handler can echo
			// the single updated phase (read from the returned session).
			session: internalauthoring.Session{ID: "s1", PhaseDrafts: []internalauthoring.PhaseDraft{{
				ID: "ph1", Order: 1, Title: "Contract",
				References: []internalplans.Reference{{Kind: internalplans.ReferenceCode, Target: "x.go"}},
			}}},
			step: internalauthoring.GuidedStep{StepKind: "phase_acceptance"},
		}
		h := newAuthoringHandler(svc)
		resp, err := h.SubmitPhaseField(context.Background(), connect.NewRequest(&authoringv1.SubmitPhaseFieldRequest{
			SessionId: "s1",
			PhaseId:   "ph1",
			Field:     "references",
			Content:   "[CODE: x.go]",
		}))
		require.NoError(t, err)
		// Focused contract: the single updated phase + progress, never a full session.
		require.Equal(t, "ph1", resp.Msg.GetPhase().GetId())
		require.Equal(t, "s1", resp.Msg.GetProgress().GetSessionId())
		require.Equal(t, "phase", resp.Msg.GetSummary().GetObjectKind())
		require.Equal(t, "references", resp.Msg.GetSummary().GetField())
		require.Equal(t, internalauthoring.PhaseFieldReferences, svc.gotPhaseField)
		require.Equal(t, "[CODE: x.go]", svc.gotContent)
	})

	t.Run("next phase", func(t *testing.T) {
		svc := &fakeAuthoringService{
			phase: internalauthoring.PhaseDraft{ID: "ph1", Order: 1, Title: "Contract"},
			step:  internalauthoring.GuidedStep{StepKind: "phase_references"},
		}
		h := newAuthoringHandler(svc)
		resp, err := h.NextPhase(context.Background(), connect.NewRequest(&authoringv1.NextPhaseRequest{SessionId: "s1"}))
		require.NoError(t, err)
		require.False(t, resp.Msg.GetComplete())
		require.Equal(t, "ph1", resp.Msg.GetPhase().GetId())
		require.Equal(t, "phase_references", resp.Msg.GetStep().GetStepKind())
	})
}

func TestContinueAuthoringSuccess(t *testing.T) {
	svc := &fakeAuthoringService{
		session: internalauthoring.Session{ID: "s1", Title: "My Plan"},
		phase:   internalauthoring.PhaseDraft{ID: "ph1", Order: 1, Title: "Phase"},
		step:    internalauthoring.GuidedStep{StepKind: "phase_relevant_context", NextActions: []internalauthoring.NextAction{{ID: "submit-context", Kind: internalauthoring.NextActionRecommended}}},
	}
	h := newAuthoringHandler(svc)

	resp, err := h.ContinueAuthoring(context.Background(), connect.NewRequest(&authoringv1.ContinueAuthoringRequest{SessionId: "s1"}))
	require.NoError(t, err)
	require.Equal(t, "s1", svc.gotSessionID)
	// continue returns focused progress + the single current work item, not the
	// full session graph.
	require.Equal(t, "s1", resp.Msg.GetProgress().GetSessionId())
	require.Equal(t, "ph1", resp.Msg.GetPhase().GetId())
	require.Equal(t, "phase_relevant_context", resp.Msg.GetStep().GetStepKind())
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
			{Source: internalauthoring.AutofillRegressionAnchor, SectionKey: internalauthoring.SectionRegressionAnchor, Degraded: true, Detail: "down"},
		},
	}
	h := newAuthoringHandler(svc)

	resp, err := h.Autofill(context.Background(), connect.NewRequest(&authoringv1.AutofillRequest{
		SessionId: "s1",
		Sources:   []string{"regression_anchor"},
	}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.GetResults(), 1)
	require.True(t, resp.Msg.GetResults()[0].GetDegraded())
	require.Equal(t, []internalauthoring.AutofillSource{internalauthoring.AutofillRegressionAnchor}, svc.gotSources)
}

func TestSuggestReferencesSuccess(t *testing.T) {
	svc := &fakeAuthoringService{
		session: internalauthoring.Session{ID: "s1"},
		refCandidates: []internalauthoring.ReferenceCandidate{
			{ID: "rc1", Reference: internalplans.Reference{Kind: internalplans.ReferenceCode, Target: "x.go"}, Source: "code-symbol", Confidence: 0.9, Status: internalauthoring.ReferenceCandidatePending},
		},
		step: internalauthoring.GuidedStep{StepKind: "reference_candidates"},
	}
	h := newAuthoringHandler(svc)

	resp, err := h.SuggestReferences(context.Background(), connect.NewRequest(&authoringv1.SuggestReferencesRequest{SessionId: "s1"}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.GetCandidates(), 1)
	require.Equal(t, "x.go", resp.Msg.GetCandidates()[0].GetReference().GetTarget())
	require.Equal(t, "code-symbol", resp.Msg.GetCandidates()[0].GetSource())
	require.Equal(t, "reference_candidates", resp.Msg.GetStep().GetStepKind())
}

func TestAcceptReferenceCandidateSuccess(t *testing.T) {
	svc := &fakeAuthoringService{
		session:      internalauthoring.Session{ID: "s1"},
		refCandidate: internalauthoring.ReferenceCandidate{ID: "rc1", Reference: internalplans.Reference{Kind: internalplans.ReferenceCode, Target: "x.go"}, Status: internalauthoring.ReferenceCandidateAccepted},
	}
	h := newAuthoringHandler(svc)

	resp, err := h.AcceptReferenceCandidate(context.Background(), connect.NewRequest(&authoringv1.AcceptReferenceCandidateRequest{SessionId: "s1", CandidateId: "rc1"}))
	require.NoError(t, err)
	require.Equal(t, "accepted", resp.Msg.GetCandidate().GetStatus())
	require.Equal(t, "rc1", svc.gotCandidateID)
	require.Equal(t, "reference_candidate", resp.Msg.GetSummary().GetObjectKind())
}

func TestRejectReferenceCandidateSuccess(t *testing.T) {
	svc := &fakeAuthoringService{
		session:      internalauthoring.Session{ID: "s1"},
		refCandidate: internalauthoring.ReferenceCandidate{ID: "rc1", Status: internalauthoring.ReferenceCandidateRejected, RejectionReason: "noise"},
	}
	h := newAuthoringHandler(svc)

	resp, err := h.RejectReferenceCandidate(context.Background(), connect.NewRequest(&authoringv1.RejectReferenceCandidateRequest{SessionId: "s1", CandidateId: "rc1", Reason: "noise"}))
	require.NoError(t, err)
	require.Equal(t, "rejected", resp.Msg.GetCandidate().GetStatus())
	require.Equal(t, "noise", svc.gotReason)
}

func TestApplyReferenceDispositionSuccess(t *testing.T) {
	svc := &fakeAuthoringService{
		session: internalauthoring.Session{ID: "s1"},
		refDisposition: internalauthoring.ReferenceDispositionSummary{
			Batch: internalauthoring.DiscoveryBatch{ID: "rb1", Status: internalauthoring.DiscoveryBatchApplied},
			Results: []internalauthoring.ReferenceDispositionResult{{
				Candidate: internalauthoring.ReferenceCandidate{ID: "rc1", Handle: "r1", Reference: internalplans.Reference{Kind: internalplans.ReferenceCode, Target: "x.go"}, Status: internalauthoring.ReferenceCandidateAccepted},
				Reference: internalplans.Reference{Kind: internalplans.ReferenceCode, Target: "x.go"},
				Action:    "take",
				Accepted:  true,
			}},
		},
	}
	h := newAuthoringHandler(svc)

	resp, err := h.ApplyReferenceDisposition(context.Background(), connect.NewRequest(&authoringv1.ApplyReferenceDispositionRequest{
		SessionId: "s1",
		BatchId:   "rb1",
		Take:      []*authoringv1.ReferenceDispositionTake{{Candidate: "r1"}},
		Drop:      []*authoringv1.ReferenceDispositionDrop{{Candidate: "r2", Reason: "noise"}},
		SweepNote: "reviewed",
		TakeAll:   true,
	}))
	require.NoError(t, err)
	require.Equal(t, "rb1", svc.gotBatchID)
	require.Equal(t, []internalauthoring.ReferenceDispositionTake{{CandidateID: "r1"}}, svc.gotRefTakes)
	require.Equal(t, []internalauthoring.ReferenceDispositionDrop{{CandidateID: "r2", Reason: "noise"}}, svc.gotRefDrops)
	require.True(t, svc.gotTakeAll)
	require.Equal(t, "reviewed", svc.gotSweepNote)
	require.Equal(t, "applied", resp.Msg.GetBatch().GetStatus())
	require.Len(t, resp.Msg.GetResults(), 1)
	require.Equal(t, "r1", resp.Msg.GetResults()[0].GetCandidate().GetHandle())
}

// TestSubmitRelevantContextItemAcceptedFlag: the response says explicitly
// whether the item entered the session — a violation-rejected submission must
// never look success-shaped.
func TestSubmitRelevantContextItemAcceptedFlag(t *testing.T) {
	t.Run("accepted", func(t *testing.T) {
		h := newAuthoringHandler(&fakeAuthoringService{})
		resp, err := h.SubmitRelevantContextItem(context.Background(), connect.NewRequest(&authoringv1.SubmitRelevantContextItemRequest{SessionId: "s1"}))
		require.NoError(t, err)
		require.True(t, resp.Msg.GetAccepted())
	})
	t.Run("rejected", func(t *testing.T) {
		h := newAuthoringHandler(&fakeAuthoringService{violations: []internalauthoring.StructureViolation{
			{SectionKey: internalauthoring.SectionRelevantContext, Message: "reason must not be empty"},
		}})
		resp, err := h.SubmitRelevantContextItem(context.Background(), connect.NewRequest(&authoringv1.SubmitRelevantContextItemRequest{SessionId: "s1"}))
		require.NoError(t, err)
		require.False(t, resp.Msg.GetAccepted())
		require.NotEmpty(t, resp.Msg.GetViolations())
	})
}

func TestSubmitFieldsRoundTrip(t *testing.T) {
	svc := &fakeAuthoringService{
		fieldResults: []internalauthoring.FieldWriteResult{
			{Index: 0, Accepted: true, Summary: "submitted section \"purpose\""},
			{Index: 1, Accepted: false, Summary: "unknown phase field \"bogus\"", Violations: []internalauthoring.StructureViolation{{SectionKey: internalauthoring.SectionPhases, Message: "unknown phase field"}}},
		},
	}
	h := newAuthoringHandler(svc)

	resp, err := h.SubmitFields(context.Background(), connect.NewRequest(&authoringv1.SubmitFieldsRequest{
		SessionId: "s1",
		Items: []*authoringv1.FieldWrite{
			{Scope: &authoringv1.FieldWrite_SectionKey{SectionKey: "purpose"}, Content: "Purpose."},
			{Scope: &authoringv1.FieldWrite_Phase{Phase: &authoringv1.PhaseFieldRef{PhaseRef: "1", Field: "bogus"}}, Content: "x"},
		},
	}))
	require.NoError(t, err)
	require.Equal(t, "s1", svc.gotSessionID)
	require.Len(t, svc.gotFieldWrites, 2)
	require.Equal(t, internalauthoring.SectionKey("purpose"), svc.gotFieldWrites[0].SectionKey)
	require.Equal(t, "1", svc.gotFieldWrites[1].PhaseRef)
	require.Equal(t, internalauthoring.PhaseField("bogus"), svc.gotFieldWrites[1].PhaseField)

	results := resp.Msg.GetResults()
	require.Len(t, results, 2)
	require.True(t, results[0].GetAccepted())
	require.False(t, results[1].GetAccepted())
	require.Equal(t, int32(1), results[1].GetIndex())
	require.NotEmpty(t, results[1].GetViolations())
}

func TestFinalizeSuccess(t *testing.T) {
	svc := &fakeAuthoringService{
		plan: internalplans.Plan{
			ID:            "plan-1",
			Status:        internalplans.PlanStatusDraft,
			WorkspaceRoot: "/repo/root",
			Mirror: internalplans.RenderedPlanMirror{
				Path:   "/home/user/.vrooli/plans/plan-1.md",
				Status: internalplans.RenderedMirrorStatusFresh,
			},
		},
		storePath:   "/data/plan-manager.db",
		finalizedAt: "2026-07-02T00:00:00Z",
	}
	h := newAuthoringHandler(svc)

	resp, err := h.Finalize(context.Background(), connect.NewRequest(&authoringv1.FinalizeRequest{
		SessionId:     "s1",
		WorkspaceRoot: "/repo/root",
	}))
	require.NoError(t, err)
	require.Equal(t, "plan-1", resp.Msg.GetPlan().GetId())
	require.Equal(t, "s1", svc.gotSessionID)
	require.Equal(t, "/repo/root", svc.gotWorkspaceRoot, "workspace root must reach the service")
	require.Equal(t, "/data/plan-manager.db", resp.Msg.GetStorePath())
	require.Equal(t, "/repo/root", resp.Msg.GetWorkspaceRoot())
	require.Equal(t, "2026-07-02T00:00:00Z", resp.Msg.GetFinalizedAt())
	require.False(t, resp.Msg.GetAlreadyFinalized())
	require.Equal(t, sharedv1.RenderedMirrorStatus_RENDERED_MIRROR_STATUS_FRESH, resp.Msg.GetMirror().GetStatus())
	require.Equal(t, "/home/user/.vrooli/plans/plan-1.md", resp.Msg.GetMirror().GetPath())
}

func TestFinalizeAlreadyFinalizedFlag(t *testing.T) {
	svc := &fakeAuthoringService{
		plan:             internalplans.Plan{ID: "plan-1", Status: internalplans.PlanStatusDraft},
		alreadyFinalized: true,
	}
	h := newAuthoringHandler(svc)

	resp, err := h.Finalize(context.Background(), connect.NewRequest(&authoringv1.FinalizeRequest{SessionId: "s1"}))
	require.NoError(t, err)
	require.True(t, resp.Msg.GetAlreadyFinalized())
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
