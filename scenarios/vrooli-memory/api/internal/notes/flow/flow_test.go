package flow

import (
	"testing"

	"vrooli-memory/internal/notes/flow/generated"
)

func TestAttachmentUploadFormalReplay(t *testing.T) {
	generated.RunReplay(t, func(s generated.AttachmentUploadStatus, e generated.AttachmentUploadEvent) (generated.AttachmentUploadStatus, error) {
		next, err := TransitionAttachmentUpload(AttachmentUploadState{Status: s}, e)
		return next.Status, err
	})
}
