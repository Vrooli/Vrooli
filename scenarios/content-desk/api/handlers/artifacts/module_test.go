package artifacts

import (
	"context"
	"testing"

	assetstudio "content-desk/integrations/assetstudio"
	channelmanager "content-desk/integrations/channelmanager"
	internalartifacts "content-desk/internal/artifacts"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	artifactsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/content-desk/v1/artifacts"
)

type repositoryStub struct {
	internalartifacts.Repository
	draft      internalartifacts.Draft
	target     internalartifacts.ReleaseTarget
	attachment internalartifacts.Attachment
	current    internalartifacts.CurrentRevision
}

func (r repositoryStub) GetCurrentRevision(context.Context, string) (internalartifacts.CurrentRevision, error) {
	return r.current, nil
}

func (r *repositoryStub) Attach(_ context.Context, attachment internalartifacts.Attachment) (internalartifacts.Attachment, error) {
	attachment.ID = "attachment-1"
	r.attachment = attachment
	return attachment, nil
}

func (r *repositoryStub) ListAttachments(_ context.Context, _ string) ([]internalartifacts.Attachment, error) {
	if r.attachment.ID == "" {
		return nil, nil
	}
	return []internalartifacts.Attachment{r.attachment}, nil
}

func (r repositoryStub) Get(context.Context, string) (internalartifacts.Draft, error) {
	return r.draft, nil
}

func (r repositoryStub) RevalidateForRelease(context.Context, string) (internalartifacts.Draft, error) {
	if r.draft.Status != internalartifacts.DraftApproved {
		return internalartifacts.Draft{}, context.Canceled
	}
	return r.draft, nil
}

func (r *repositoryStub) RecordEligibility(_ context.Context, _ string, target internalartifacts.ReleaseTarget) error {
	r.target = target
	return nil
}

func (r repositoryStub) GetReleaseTarget(_ context.Context, _ string) (internalartifacts.ReleaseTarget, bool, error) {
	if r.target.Eligibility == "" {
		return internalartifacts.ReleaseTarget{}, false, nil
	}
	return r.target, true, nil
}

type submitterStub struct {
	receipt channelmanager.Receipt
	request channelmanager.Submission
	err     error
}

type eligibilityStub struct {
	result string
	err    error
}

type assetStub struct {
	reference assetstudio.Reference
	err       error
}

func (s assetStub) ResolveReleasedAsset(context.Context, string) (assetstudio.Reference, error) {
	return s.reference, s.err
}

func (s eligibilityStub) CheckEligibility(context.Context, string, string) (string, error) {
	return s.result, s.err
}

func (s *submitterStub) SubmitRelease(_ context.Context, request channelmanager.Submission) (channelmanager.Receipt, error) {
	s.request = request
	return s.receipt, s.err
}

// [REQ:CONTENTD-P1-006]
func TestSubmitReleaseDraftDelegatesOnlyApprovedDrafts(t *testing.T) {
	submitter := &submitterStub{receipt: channelmanager.Receipt{ID: "release-1", ActionID: "action-1", Status: "scheduled"}}
	repo := &repositoryStub{draft: internalartifacts.Draft{ID: "draft-1", Status: internalartifacts.DraftApproved}, attachment: internalartifacts.Attachment{ID: "attachment-1", DraftID: "draft-1", AssetID: "asset-1"}}
	h := handler{repo: repo, submitter: submitter}
	response, err := h.SubmitReleaseDraft(context.Background(), connect.NewRequest(&artifactsv1.SubmitReleaseDraftRequest{Id: "draft-1", IdentityId: "identity-1", Lane: "main", IdempotencyKey: "release-key", DisclosureVisible: true}))
	require.NoError(t, err)
	require.Equal(t, "release-1", response.Msg.ReleaseId)
	require.Equal(t, channelmanager.Submission{IdentityID: "identity-1", Lane: "main", DraftID: "draft-1", IdempotencyKey: "release-key", AssetIDs: []string{"asset-1"}, DisclosureVisible: true}, submitter.request)
}

func TestSubmitReleaseDraftFailsClosedForNonApprovedDraft(t *testing.T) {
	h := handler{repo: &repositoryStub{draft: internalartifacts.Draft{ID: "draft-1", Status: internalartifacts.DraftReviewed}}, submitter: &submitterStub{}}
	_, err := h.SubmitReleaseDraft(context.Background(), connect.NewRequest(&artifactsv1.SubmitReleaseDraftRequest{Id: "draft-1", IdentityId: "identity-1", Lane: "main", IdempotencyKey: "release-key"}))
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
}

// A draft approved for a specific account must release to that account. A
// request that names a different identity or lane is refused before any Channel
// Manager dispatch, so approval and release cannot diverge.
func TestSubmitReleaseDraftRejectsTargetThatDiffersFromApproved(t *testing.T) {
	submitter := &submitterStub{}
	repo := &repositoryStub{draft: internalartifacts.Draft{ID: "draft-1", Status: internalartifacts.DraftApproved}, target: internalartifacts.ReleaseTarget{IdentityID: "identity-1", Lane: "main", Eligibility: "eligible"}}
	h := handler{repo: repo, submitter: submitter}
	_, err := h.SubmitReleaseDraft(context.Background(), connect.NewRequest(&artifactsv1.SubmitReleaseDraftRequest{Id: "draft-1", IdentityId: "identity-2", Lane: "main", IdempotencyKey: "release-key"}))
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	require.Empty(t, submitter.request.IdentityID)
}

// When the request omits the target, the recorded approved target is used
// instead of failing on an empty identity.
func TestSubmitReleaseDraftUsesApprovedTargetWhenRequestOmitsIt(t *testing.T) {
	submitter := &submitterStub{receipt: channelmanager.Receipt{ID: "release-1"}}
	repo := &repositoryStub{draft: internalartifacts.Draft{ID: "draft-1", Status: internalartifacts.DraftApproved}, target: internalartifacts.ReleaseTarget{IdentityID: "identity-1", Lane: "main", Eligibility: "eligible"}}
	h := handler{repo: repo, submitter: submitter}
	if _, err := h.SubmitReleaseDraft(context.Background(), connect.NewRequest(&artifactsv1.SubmitReleaseDraftRequest{Id: "draft-1", IdempotencyKey: "release-key"})); err != nil {
		t.Fatal(err)
	}
	require.Equal(t, "identity-1", submitter.request.IdentityID)
	require.Equal(t, "main", submitter.request.Lane)
}

// A targetless approval cannot release without naming both an identity and a
// lane, and a recorded ineligible target is never released.
func TestSubmitReleaseDraftFailsClosedWithoutUsableTarget(t *testing.T) {
	approved := internalartifacts.Draft{ID: "draft-1", Status: internalartifacts.DraftApproved}
	h := handler{repo: &repositoryStub{draft: approved}, submitter: &submitterStub{}}
	_, err := h.SubmitReleaseDraft(context.Background(), connect.NewRequest(&artifactsv1.SubmitReleaseDraftRequest{Id: "draft-1", IdempotencyKey: "release-key"}))
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))

	ineligible := handler{repo: &repositoryStub{draft: approved, target: internalartifacts.ReleaseTarget{IdentityID: "identity-1", Lane: "main", Eligibility: "not_eligible"}}, submitter: &submitterStub{}}
	_, err = ineligible.SubmitReleaseDraft(context.Background(), connect.NewRequest(&artifactsv1.SubmitReleaseDraftRequest{Id: "draft-1", IdentityId: "identity-1", Lane: "main", IdempotencyKey: "release-key"}))
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
}

// [REQ:CONTENTD-P1-005]
func TestTargetedApprovalRequiresEligibleChannelManagerIdentity(t *testing.T) {
	repo := &repositoryStub{draft: internalartifacts.Draft{ID: "draft-1", Status: internalartifacts.DraftReviewed}}
	h := handler{repo: repo, eligibility: eligibilityStub{result: "not_eligible"}}
	_, err := h.ApproveDraft(context.Background(), connect.NewRequest(&artifactsv1.ApproveDraftRequest{Id: "draft-1", IdentityId: "identity-1", Lane: "main"}))
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	require.Equal(t, "not_eligible", repo.target.Eligibility)
}

// [REQ:CONTENTD-P1-009]
func TestAttachReleasedAssetStoresOnlyValidatedMetadataReference(t *testing.T) {
	repo := &repositoryStub{}
	h := handler{repo: repo, assets: assetStub{reference: assetstudio.Reference{ID: "asset-1", AltText: "Asset Studio alt", MediaType: "image/png", Width: 1600, Height: 900}}}
	response, err := h.AttachReleasedAsset(context.Background(), connect.NewRequest(&artifactsv1.AttachReleasedAssetRequest{DraftId: "draft-1", AssetId: "asset-1", Role: "hero", AspectRatio: "16:9", AltText: "Operator-provided accessible description", Position: 0}))
	require.NoError(t, err)
	require.Equal(t, "asset-1", response.Msg.Attachment.AssetId)
	require.Equal(t, "Operator-provided accessible description", repo.attachment.AltText)
	require.Empty(t, response.Msg.Attachment.ProtoReflect().GetUnknown())
}

// The current-revision read must report only authority that still applies to the
// effective body: a revision that invalidated the prior review/approval resolves
// to has_* = false, so a consumer cannot release under stale authority.
func TestGetDraftCurrentRevisionExposesOnlyApplicableAuthority(t *testing.T) {
	repo := &repositoryStub{current: internalartifacts.CurrentRevision{
		DraftID:    "draft-1",
		Status:     internalartifacts.DraftBlocked,
		Body:       "effective body",
		RevisionID: "rev-2",
		ActorKind:  "operator",
		Capacity:   "editor",
	}}
	h := handler{repo: repo}
	response, err := h.GetDraftCurrentRevision(context.Background(), connect.NewRequest(&artifactsv1.GetDraftCurrentRevisionRequest{Id: "draft-1"}))
	require.NoError(t, err)
	current := response.Msg.CurrentRevision
	require.Equal(t, "draft-1", current.DraftId)
	require.Equal(t, "effective body", current.Body)
	require.Equal(t, "rev-2", current.RevisionId)
	require.True(t, current.HasStoredRevision)
	require.False(t, current.HasApplicableReview)
	require.False(t, current.HasApplicableApproval)
	require.Empty(t, current.ReviewRunId)
	require.Empty(t, current.ApprovalActorKind)
}

// A still-applicable review and approval are reported with their stored
// provenance, not collapsed into a boolean.
func TestGetDraftCurrentRevisionReportsApplicableAuthority(t *testing.T) {
	repo := &repositoryStub{current: internalartifacts.CurrentRevision{
		DraftID:           "draft-1",
		Status:            internalartifacts.DraftApproved,
		Body:              "approved body",
		RevisionID:        "rev-1",
		ReviewRunID:       "run-1",
		ApprovalActorKind: "operator",
		ApprovalCapacity:  "editor",
	}}
	h := handler{repo: repo}
	response, err := h.GetDraftCurrentRevision(context.Background(), connect.NewRequest(&artifactsv1.GetDraftCurrentRevisionRequest{Id: "draft-1"}))
	require.NoError(t, err)
	current := response.Msg.CurrentRevision
	require.True(t, current.HasStoredRevision)
	require.True(t, current.HasApplicableReview)
	require.True(t, current.HasApplicableApproval)
	require.Equal(t, "run-1", current.ReviewRunId)
	require.Equal(t, "operator", current.ApprovalActorKind)
}
