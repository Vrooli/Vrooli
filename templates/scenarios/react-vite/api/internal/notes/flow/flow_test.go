package notes

import (
	"testing"

	"{{SCENARIO_ID}}/internal/notes/generated/attachmentupload"
)

func TestAttachmentUploadFormalReplay(t *testing.T) {
	attachmentupload.RunReplay(t, func(s attachmentupload.AttachmentUploadStatus, e attachmentupload.AttachmentUploadEvent) (attachmentupload.AttachmentUploadStatus, error) {
		next, err := TransitionAttachmentUpload(AttachmentUploadState{Status: s}, e)
		return next.Status, err
	})
}
