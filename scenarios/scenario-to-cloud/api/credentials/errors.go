package credentials

import (
	"net/http"

	"scenario-to-cloud/apierrors"
)

// Stable codes owned by the credential lifecycle. They extend the shared
// registry in apierrors without renaming anything there; HTTP statuses are
// set explicitly so the shared code→status map need not know them.
const (
	CodeDescriptorCollision  = "credential_descriptor_collision"
	CodeBindingAmbiguous     = "credential_binding_ambiguous"
	CodeBindingNotFound      = "credential_binding_not_found"
	CodeRotationNotFound     = "credential_rotation_not_found"
	CodeRotationConflict     = "credential_rotation_conflict"
	CodeRotationRefused      = "credential_rotation_refused"
	CodeStoreLocked          = "credential_store_locked"
	CodeDistributionFailed   = "credential_distribution_failed"
	CodeRevocationIncomplete = "revocation_incomplete"
	CodePendingOperatorInput = "pending_operator_input"
	CodeVerificationFailed   = "credential_verification_failed"
	CodeConfirmationRequired = "break_glass_confirmation_required"
	CodeRecoveryFailed       = "credential_recovery_failed"
)

var statusByCode = map[string]int{
	CodeDescriptorCollision:  http.StatusConflict,
	CodeBindingAmbiguous:     http.StatusConflict,
	CodeBindingNotFound:      http.StatusNotFound,
	CodeRotationNotFound:     http.StatusNotFound,
	CodeRotationConflict:     http.StatusConflict,
	CodeRotationRefused:      http.StatusUnprocessableEntity,
	CodeStoreLocked:          http.StatusServiceUnavailable,
	CodeDistributionFailed:   http.StatusBadGateway,
	CodeRevocationIncomplete: http.StatusAccepted,
	CodePendingOperatorInput: http.StatusPreconditionRequired,
	CodeVerificationFailed:   http.StatusUnprocessableEntity,
	CodeConfirmationRequired: http.StatusPreconditionRequired,
	CodeRecoveryFailed:       http.StatusBadGateway,
}

// HTTPStatusFor reports the lifecycle's HTTP status for one of its codes.
func HTTPStatusFor(code string) (int, bool) {
	status, ok := statusByCode[code]
	return status, ok
}

// newError builds a typed error with the lifecycle's HTTP status.
func newError(code, message string) *apierrors.Error {
	e := apierrors.New(code, message)
	if status, ok := statusByCode[code]; ok {
		e.HTTPStatus = status
	}
	return e
}

// StoreLocked is the fail-closed answer when the target credential store
// cannot accept or reveal material. The next action is the operator unlock.
func StoreLocked(target string) *apierrors.Error {
	return newError(CodeStoreLocked, "the target credential store is locked; nothing was written").
		WithDetail("target", target).
		WithNextAction(apierrors.NextAction{Owner: "vrooli", Kind: "command", Reference: "vrooli credentials doctor", Label: "Unlock the target credential store, then retry"})
}
