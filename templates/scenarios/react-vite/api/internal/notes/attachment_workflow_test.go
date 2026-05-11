package notes_test

import (
	"testing"
	"{{SCENARIO_ID}}/internal/notes"

	"github.com/stretchr/testify/require"
)

func TestAttachmentUploadWorkflow_RejectsUnknownState(t *testing.T) {
	err := notes.CheckAttachmentUploadInvariants(notes.AttachmentUploadState{Status: "ghost"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown attachment upload status")
}
