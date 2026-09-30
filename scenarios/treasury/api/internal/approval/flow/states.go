package flow

import "treasury/internal/approval/flow/generated"

type (
	ApprovalStatus = generated.ApprovalStatus
	ApprovalEvent  = generated.ApprovalEvent
)

const (
	ApprovalQueued   = generated.ApprovalQueued
	ApprovalApproved = generated.ApprovalApproved
	ApprovalDeclined = generated.ApprovalDeclined
	ApprovalExpired  = generated.ApprovalExpired
	ApprovalApprove  = generated.ApprovalApprove
	ApprovalDecline  = generated.ApprovalDecline
	ApprovalExpire   = generated.ApprovalExpire
)

type ApprovalState struct{ Status ApprovalStatus }

func InitialApprovalState() ApprovalState { return ApprovalState{Status: ApprovalQueued} }
