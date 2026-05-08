package notes

import "fmt"

// AttachmentUploadStatus describes the API-side lifecycle for the
// multipart attachment upload orchestration. It is intentionally about
// orchestration, not persistence: attachment metadata is still owned by
// Attachment and the repository.
type AttachmentUploadStatus string

const (
	AttachmentUploadReceived         AttachmentUploadStatus = "received"
	AttachmentUploadBytesStored      AttachmentUploadStatus = "bytes_stored"
	AttachmentUploadMetadataRecorded AttachmentUploadStatus = "metadata_recorded"
	AttachmentUploadFailed           AttachmentUploadStatus = "failed"
)

// AttachmentUploadEvent describes allowed lifecycle events during upload.
type AttachmentUploadEvent string

const (
	AttachmentUploadStoreBytes     AttachmentUploadEvent = "store_bytes"
	AttachmentUploadRecordMetadata AttachmentUploadEvent = "record_metadata"
	AttachmentUploadFail           AttachmentUploadEvent = "fail"
)

type AttachmentUploadState struct {
	Status AttachmentUploadStatus
}

func InitialAttachmentUploadState() AttachmentUploadState {
	return AttachmentUploadState{Status: AttachmentUploadReceived}
}

func AllAttachmentUploadStatuses() []AttachmentUploadStatus {
	return []AttachmentUploadStatus{
		AttachmentUploadReceived,
		AttachmentUploadBytesStored,
		AttachmentUploadMetadataRecorded,
		AttachmentUploadFailed,
	}
}

func AllAttachmentUploadEvents() []AttachmentUploadEvent {
	return []AttachmentUploadEvent{
		AttachmentUploadStoreBytes,
		AttachmentUploadRecordMetadata,
		AttachmentUploadFail,
	}
}

func TransitionAttachmentUpload(state AttachmentUploadState, event AttachmentUploadEvent) (AttachmentUploadState, error) {
	if err := CheckAttachmentUploadInvariants(state); err != nil {
		return state, err
	}
	switch event {
	case AttachmentUploadStoreBytes:
		if state.Status == AttachmentUploadReceived {
			return AttachmentUploadState{Status: AttachmentUploadBytesStored}, nil
		}
	case AttachmentUploadRecordMetadata:
		if state.Status == AttachmentUploadBytesStored {
			return AttachmentUploadState{Status: AttachmentUploadMetadataRecorded}, nil
		}
	case AttachmentUploadFail:
		if state.Status == AttachmentUploadReceived || state.Status == AttachmentUploadBytesStored {
			return AttachmentUploadState{Status: AttachmentUploadFailed}, nil
		}
	}
	return state, fmt.Errorf("cannot apply %s from %s", event, state.Status)
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
