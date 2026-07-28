package flow

import (
	"testing"

	"content-desk/internal/artifacts"
	"github.com/stretchr/testify/require"
)

func TestDraftLifecycleReplay(t *testing.T) {
	status := artifacts.DraftRequested
	for _, event := range []artifacts.DraftEvent{artifacts.DraftBegin, artifacts.DraftComplete, artifacts.DraftCheck, artifacts.DraftReviewPass, artifacts.DraftApprove, artifacts.DraftPublish} {
		next, err := TransitionDraftLifecycle(status, event)
		require.NoError(t, err)
		status = next
	}
	require.Equal(t, artifacts.DraftPublished, status)
}
