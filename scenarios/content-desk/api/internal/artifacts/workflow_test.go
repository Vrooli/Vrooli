package artifacts

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDraftTransitionMatrix(t *testing.T) {
	statuses := []DraftStatus{DraftRequested, DraftDrafting, DraftDrafted, DraftChecking, DraftBlocked, DraftReviewed, DraftApproved, DraftPublished, DraftAbandoned}
	events := []DraftEvent{DraftBegin, DraftComplete, DraftCheck, DraftBlock, DraftReviewPass, DraftApprove, DraftPublish, DraftAbandon}
	valid := map[[2]string]string{
		{string(DraftRequested), string(DraftBegin)}: string(DraftDrafting), {string(DraftDrafting), string(DraftComplete)}: string(DraftDrafted), {string(DraftDrafted), string(DraftCheck)}: string(DraftChecking), {string(DraftChecking), string(DraftBlock)}: string(DraftBlocked), {string(DraftChecking), string(DraftReviewPass)}: string(DraftReviewed), {string(DraftBlocked), string(DraftCheck)}: string(DraftChecking), {string(DraftReviewed), string(DraftBlock)}: string(DraftBlocked), {string(DraftReviewed), string(DraftApprove)}: string(DraftApproved), {string(DraftApproved), string(DraftPublish)}: string(DraftPublished),
	}
	for _, status := range statuses {
		for _, event := range events {
			t.Run(string(status)+"/"+string(event), func(t *testing.T) {
				next, err := TransitionDraft(DraftState{Status: status}, event)
				want, isValid := valid[[2]string{string(status), string(event)}]
				abandonAllowed := event == DraftAbandon && status != DraftApproved && status != DraftPublished && status != DraftAbandoned
				if isValid {
					require.NoError(t, err)
					require.Equal(t, DraftStatus(want), next.Status)
					return
				}
				if abandonAllowed {
					require.NoError(t, err)
					require.Equal(t, DraftAbandoned, next.Status)
					return
				}
				require.Error(t, err)
				require.Equal(t, status, next.Status)
			})
		}
	}
}

func TestDraftCannotSkipReviewOrLeavePublished(t *testing.T) {
	_, err := TransitionDraft(DraftState{Status: DraftDrafted}, DraftApprove)
	require.Error(t, err)
	_, err = TransitionDraft(DraftState{Status: DraftPublished}, DraftCheck)
	require.Error(t, err)
}
