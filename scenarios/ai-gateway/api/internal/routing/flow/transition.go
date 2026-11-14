package flow

import (
	"fmt"

	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing"

	"ai-gateway/internal/routing/flow/generated"
)

const OutcomeUnknownCode = "outcome_unknown"

type ReceiptState struct {
	Status    routingv1.MediaExecutionStatus
	ErrorCode string
}

// TransitionMedia delegates receipt topology to the generated owner contract.
func TransitionMedia(status generated.MediaStatus, event generated.MediaEvent) (generated.MediaStatus, error) {
	return generated.TransitionMediaStatus(status, event)
}

// TransitionReceipt adapts the public receipt representation to the domain
// lifecycle. Unknown outcome is a distinct state encoded by FAILED plus code.
func TransitionReceipt(status routingv1.MediaExecutionStatus, code string, event generated.MediaEvent) (ReceiptState, error) {
	state, err := State(status, code)
	if err != nil {
		return ReceiptState{Status: status, ErrorCode: code}, err
	}
	next, err := TransitionMedia(state, event)
	if err != nil {
		return ReceiptState{Status: status, ErrorCode: code}, err
	}
	return Representation(next)
}

func Representation(state generated.MediaStatus) (ReceiptState, error) {
	var status routingv1.MediaExecutionStatus
	var code string
	switch state {
	case generated.MediaQueued:
		status = routingv1.MediaExecutionStatus_MEDIA_EXECUTION_STATUS_QUEUED
	case generated.MediaRunning:
		status = routingv1.MediaExecutionStatus_MEDIA_EXECUTION_STATUS_RUNNING
	case generated.MediaSucceeded:
		status = routingv1.MediaExecutionStatus_MEDIA_EXECUTION_STATUS_SUCCEEDED
	case generated.MediaFailed, generated.MediaOutcomeUnknown:
		status = routingv1.MediaExecutionStatus_MEDIA_EXECUTION_STATUS_FAILED
		if state == generated.MediaOutcomeUnknown {
			code = OutcomeUnknownCode
		}
	case generated.MediaCancelled:
		status = routingv1.MediaExecutionStatus_MEDIA_EXECUTION_STATUS_CANCELLED
	default:
		return ReceiptState{}, fmt.Errorf("unsupported media state %q", state)
	}
	return ReceiptState{Status: status, ErrorCode: code}, nil
}

func State(status routingv1.MediaExecutionStatus, code string) (generated.MediaStatus, error) {
	if code == OutcomeUnknownCode && status != routingv1.MediaExecutionStatus_MEDIA_EXECUTION_STATUS_FAILED {
		return "", fmt.Errorf("unknown media outcome requires failed status")
	}
	switch status {
	case routingv1.MediaExecutionStatus_MEDIA_EXECUTION_STATUS_QUEUED:
		return generated.MediaQueued, nil
	case routingv1.MediaExecutionStatus_MEDIA_EXECUTION_STATUS_RUNNING:
		return generated.MediaRunning, nil
	case routingv1.MediaExecutionStatus_MEDIA_EXECUTION_STATUS_SUCCEEDED:
		return generated.MediaSucceeded, nil
	case routingv1.MediaExecutionStatus_MEDIA_EXECUTION_STATUS_FAILED:
		if code == OutcomeUnknownCode {
			return generated.MediaOutcomeUnknown, nil
		}
		return generated.MediaFailed, nil
	case routingv1.MediaExecutionStatus_MEDIA_EXECUTION_STATUS_CANCELLED:
		return generated.MediaCancelled, nil
	default:
		return "", fmt.Errorf("unsupported media execution status %s", status)
	}
}
