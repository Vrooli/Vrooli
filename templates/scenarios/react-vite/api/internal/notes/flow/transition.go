package notes

import (
	"fmt"

	"{{SCENARIO_ID}}/internal/notes/generated/attachmentupload"
)

type AttachmentUploadStatus = attachmentupload.AttachmentUploadStatus
type AttachmentUploadEvent = attachmentupload.AttachmentUploadEvent

const (
	AttachmentUploadReceived         = attachmentupload.AttachmentUploadReceived
	AttachmentUploadBytesStored      = attachmentupload.AttachmentUploadBytesStored
	AttachmentUploadMetadataRecorded = attachmentupload.AttachmentUploadMetadataRecorded
	AttachmentUploadFailed           = attachmentupload.AttachmentUploadFailed
)

const (
	AttachmentUploadStoreBytes     = attachmentupload.AttachmentUploadStoreBytes
	AttachmentUploadRecordMetadata = attachmentupload.AttachmentUploadRecordMetadata
	AttachmentUploadFail           = attachmentupload.AttachmentUploadFail
)

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
	next, err := attachmentupload.TransitionAttachmentUploadStatus(state.Status, event)
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
