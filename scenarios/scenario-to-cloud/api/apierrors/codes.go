package apierrors

import "net/http"

// Stable error codes. These strings are the contract shared by the API, the
// CLI and the UI; never rename one without a deprecation cycle. The wire body
// never carries the HTTP status, so the status is derived from the code here.
const (
	CodeInvalidJSON                 = "invalid_json"
	CodeInvalidRequest              = "invalid_request"
	CodeInternal                    = "internal"
	CodeUnauthenticated             = "unauthenticated"
	CodeForbiddenScope              = "forbidden_scope"
	CodeForbiddenTarget             = "forbidden_target"
	CodeForbiddenRevoked            = "forbidden_revoked"
	CodeForbiddenOrigin             = "forbidden_origin"
	CodeForbiddenHost               = "forbidden_host"
	CodeRequestTooLarge             = "request_too_large"
	CodeTooManyRequests             = "too_many_requests"
	CodeDeploymentNotFound          = "deployment_not_found"
	CodeDeploymentSelectorAmbiguous = "deployment_selector_ambiguous"
	CodeDeploymentSelectorInvalid   = "deployment_selector_invalid"
	CodeDeploymentIdentityConflict  = "deployment_identity_conflict"
	CodeManifestInvalid             = "manifest_invalid"
	CodeRequestKeyConflict          = "request_key_conflict"
	CodePlanStale                   = "plan_stale"
	CodePlanDigestMismatch          = "plan_digest_mismatch"
	CodeOperationConflict           = "operation_conflict"
	CodeOperationNotFound           = "operation_not_found"
	CodeFenceStale                  = "fence_stale"
	CodeClosureUnavailable          = "closure_unavailable"
	CodeUnsupportedCapability       = "unsupported_capability"
	CodeUnsupportedSchemaVersion    = "unsupported_schema_version"
	CodeNeedsInput                  = "needs_input"
	CodeReleaseVerificationFailed   = "release_verification_failed"
	CodeReachUnavailable            = "reach_unavailable"
	CodeReachScopeMissing           = "reach_scope_missing"
	CodeReachProtocolUnsupported    = "reach_protocol_unsupported"
	CodeTargetOffline               = "target_offline"
	CodeEnrollmentRevoked           = "enrollment_revoked"
	CodeHealthUnknown               = "health_unknown"
	CodePublicationRefused          = "publication_refused"
	CodeGovernanceUnavailable       = "governance_unavailable"
	CodeReceiptInvalid              = "receipt_invalid"
	CodePreviewRequired             = "preview_required"
	// Data recovery (P12). The target-side codes are the recoverypoint
	// engine's so the cloud side, the target and the certification matrix
	// share one name.
	CodeRecoveryPointNotFound     = "recovery_point_not_found"
	CodeRecoveryPointConflict     = "recovery_point_conflict"
	CodeRecoveryPointCorrupt      = "recovery_point_corrupt"
	CodeRecoveryPointProtected    = "recovery_point_protected"
	CodeRecoveryPointRequired     = "recovery_point_required"
	CodeRecoveryKeyUnavailable    = "recovery_key_unavailable"
	CodeRestoreTargetNotClean     = "restore_target_not_clean"
	CodeBackupProviderUnavailable = "backup_provider_unavailable"
	CodeBackupFailed              = "backup_failed"
	CodeRestoreFailed             = "restore_failed"
	CodeRollbackIncompatible      = "rollback_incompatible"
)

// statusByCode is the single mapping from a stable code to an HTTP status.
var statusByCode = map[string]int{
	CodeInvalidJSON:                 http.StatusBadRequest,
	CodeInvalidRequest:              http.StatusBadRequest,
	CodeInternal:                    http.StatusInternalServerError,
	CodeUnauthenticated:             http.StatusUnauthorized,
	CodeForbiddenScope:              http.StatusForbidden,
	CodeForbiddenTarget:             http.StatusForbidden,
	CodeForbiddenRevoked:            http.StatusForbidden,
	CodeForbiddenOrigin:             http.StatusForbidden,
	CodeForbiddenHost:               http.StatusForbidden,
	CodeRequestTooLarge:             http.StatusRequestEntityTooLarge,
	CodeTooManyRequests:             http.StatusTooManyRequests,
	CodeDeploymentNotFound:          http.StatusNotFound,
	CodeDeploymentSelectorAmbiguous: http.StatusConflict,
	CodeDeploymentSelectorInvalid:   http.StatusBadRequest,
	CodeDeploymentIdentityConflict:  http.StatusConflict,
	CodeManifestInvalid:             http.StatusUnprocessableEntity,
	CodeRequestKeyConflict:          http.StatusConflict,
	CodePlanStale:                   http.StatusConflict,
	CodePlanDigestMismatch:          http.StatusConflict,
	CodeOperationConflict:           http.StatusConflict,
	CodeOperationNotFound:           http.StatusNotFound,
	CodeReachScopeMissing:           http.StatusForbidden,
	CodeReachProtocolUnsupported:    http.StatusConflict,
	CodeTargetOffline:               http.StatusServiceUnavailable,
	CodeEnrollmentRevoked:           http.StatusForbidden,
	CodeFenceStale:                  http.StatusConflict,
	CodeClosureUnavailable:          http.StatusFailedDependency,
	CodeUnsupportedCapability:       http.StatusNotImplemented,
	CodeUnsupportedSchemaVersion:    http.StatusBadRequest,
	CodeNeedsInput:                  http.StatusPreconditionRequired,
	CodeReleaseVerificationFailed:   http.StatusUnprocessableEntity,
	CodeReachUnavailable:            http.StatusServiceUnavailable,
	CodeHealthUnknown:               http.StatusServiceUnavailable,
	CodePublicationRefused:          http.StatusConflict,
	CodeGovernanceUnavailable:       http.StatusServiceUnavailable,
	CodeReceiptInvalid:              http.StatusUnprocessableEntity,
	CodePreviewRequired:             http.StatusPreconditionFailed,
	CodeRecoveryPointNotFound:       http.StatusNotFound,
	CodeRecoveryPointConflict:       http.StatusConflict,
	CodeRecoveryPointCorrupt:        http.StatusUnprocessableEntity,
	CodeRecoveryPointProtected:      http.StatusConflict,
	CodeRecoveryPointRequired:       http.StatusPreconditionRequired,
	CodeRecoveryKeyUnavailable:      http.StatusFailedDependency,
	CodeRestoreTargetNotClean:       http.StatusConflict,
	CodeBackupProviderUnavailable:   http.StatusServiceUnavailable,
	CodeBackupFailed:                http.StatusFailedDependency,
	CodeRestoreFailed:               http.StatusFailedDependency,
	CodeRollbackIncompatible:        http.StatusConflict,
}

// StatusFor returns the HTTP status for a stable code. Unknown codes are
// reported as 500 so a typo in a code can never masquerade as a client fault.
func StatusFor(code string) int {
	if status, ok := statusByCode[code]; ok {
		return status
	}
	return http.StatusInternalServerError
}

// Exit codes the CLI derives from an error code.
const (
	ExitOK              = 0
	ExitFailed          = 1
	ExitRefused         = 2
	ExitPending         = 3
	ExitObserverTimeout = 124
)

// ExitCodeFor maps a stable error code to the CLI exit code contract:
// 0 ok, 1 failed, 2 refused/conflict, 3 pending/needs-input.
func ExitCodeFor(code string) int {
	switch code {
	case "":
		return ExitOK
	case CodeNeedsInput:
		return ExitPending
	case CodeUnauthenticated, CodeForbiddenScope, CodeForbiddenTarget,
		CodeForbiddenRevoked, CodeForbiddenOrigin, CodeForbiddenHost,
		CodeDeploymentSelectorAmbiguous, CodeDeploymentIdentityConflict,
		CodeRequestKeyConflict, CodePlanStale, CodePlanDigestMismatch,
		CodeOperationConflict, CodeFenceStale, CodeUnsupportedCapability,
		CodeUnsupportedSchemaVersion, CodePublicationRefused, CodeReceiptInvalid,
		CodePreviewRequired, CodeRecoveryPointConflict, CodeRecoveryPointCorrupt,
		CodeRecoveryPointProtected, CodeRecoveryPointRequired, CodeRecoveryKeyUnavailable,
		CodeRestoreTargetNotClean, CodeBackupProviderUnavailable, CodeRollbackIncompatible,
		CodeReachScopeMissing, CodeReachProtocolUnsupported, CodeEnrollmentRevoked:
		return ExitRefused
	default:
		return ExitFailed
	}
}
