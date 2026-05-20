package flow

import (
	"fmt"

	"ui-health/internal/notes/flow/generated"
)

type (
	AttachmentUploadStatus = generated.AttachmentUploadStatus
	AttachmentUploadEvent  = generated.AttachmentUploadEvent
)

const (
	AttachmentUploadReceived         = generated.AttachmentUploadReceived
	AttachmentUploadBytesStored      = generated.AttachmentUploadBytesStored
	AttachmentUploadMetadataRecorded = generated.AttachmentUploadMetadataRecorded
	AttachmentUploadFailed           = generated.AttachmentUploadFailed
)

const (
	AttachmentUploadStoreBytes     = generated.AttachmentUploadStoreBytes
	AttachmentUploadRecordMetadata = generated.AttachmentUploadRecordMetadata
	AttachmentUploadFail           = generated.AttachmentUploadFail
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
	next, err := generated.TransitionAttachmentUploadStatus(state.Status, event)
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
