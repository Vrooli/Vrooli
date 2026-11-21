package flow

import "treasury/internal/mandate/flow/generated"

type (
	MandateStatus = generated.MandateStatus
	MandateEvent  = generated.MandateEvent
)

const (
	MandateDraft       = generated.MandateDraft
	MandateLive        = generated.MandateLive
	MandateExhausted   = generated.MandateExhausted
	MandateExpired     = generated.MandateExpired
	MandateRevoked     = generated.MandateRevoked
	MandateIssue       = generated.MandateIssue
	MandateExhaust     = generated.MandateExhaust
	MandateReachExpiry = generated.MandateReachExpiry
	MandateRevoke      = generated.MandateRevoke
)

type MandateState struct {
	Status MandateStatus
}

func InitialMandateState() MandateState {
	return MandateState{Status: MandateDraft}
}
