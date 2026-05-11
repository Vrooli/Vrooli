package notes

import "fmt"

// AttachmentUploadStatus describes the API-side lifecycle for the
// multipart attachment upload orchestration. It is intentionally about
// orchestration, not persistence: attachment metadata is still owned by
// Attachment and the repository.
type AttachmentUploadStatus string

// AttachmentUploadEvent describes allowed lifecycle events during upload.
type AttachmentUploadEvent string

type AttachmentUploadState struct {
	Status AttachmentUploadStatus
}

func InitialAttachmentUploadState() AttachmentUploadState {
	return AttachmentUploadState{Status: AttachmentUploadReceived}
}

func TransitionAttachmentUpload(state AttachmentUploadState, event AttachmentUploadEvent) (AttachmentUploadState, error) {
	if err := CheckAttachmentUploadInvariants(state); err != nil {
		return state, err
	}
	next, err := TransitionAttachmentUploadStatus(state.Status, event)
	return AttachmentUploadState{Status: next}, err
}

func CheckAttachmentUploadInvariants(state AttachmentUploadState) error {
	switch state.Status {
	case AttachmentUploadReceived,
		AttachmentUploadBytesStored,
		AttachmentUploadMetadataRecorded,
		AttachmentUploadFailed:
		return nil
	default:
		return fmt.Errorf("unknown attachment upload status %q", state.Status)
	}
}
