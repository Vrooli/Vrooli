package flow

import "treasury/internal/authorization/flow/generated"

type (
	AuthorizationStatus = generated.AuthorizationStatus
	AuthorizationEvent  = generated.AuthorizationEvent
)

const (
	AuthorizationEvaluating      = generated.AuthorizationEvaluating
	AuthorizationRefused         = generated.AuthorizationRefused
	AuthorizationPending         = generated.AuthorizationPending
	AuthorizationApproved        = generated.AuthorizationApproved
	AuthorizationReleased        = generated.AuthorizationReleased
	AuthorizationSettled         = generated.AuthorizationSettled
	AuthorizationRefuse          = generated.AuthorizationRefuse
	AuthorizationRequireApproval = generated.AuthorizationRequireApproval
	AuthorizationApprove         = generated.AuthorizationApprove
	AuthorizationRelease         = generated.AuthorizationRelease
	AuthorizationSettle          = generated.AuthorizationSettle
)

type AuthorizationState struct{ Status AuthorizationStatus }

func InitialAuthorizationState() AuthorizationState {
	return AuthorizationState{Status: AuthorizationEvaluating}
}
