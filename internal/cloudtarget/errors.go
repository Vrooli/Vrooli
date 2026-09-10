package cloudtarget

import (
	"errors"
	"fmt"
)

// Exit codes follow the cloud CLI contract: 0 ok, 1 failed, 2 refused.
const (
	ExitFailed  = 1
	ExitRefused = 2
)

// Stable reason codes. The cloud side maps these onto its typed error model;
// do not rename without updating the certification matrix.
const (
	CodeFenceStale              = "fence_stale"
	CodeReceiptInputMismatch    = "receipt_input_mismatch"
	CodeInvalidArgument         = "invalid_argument"
	CodeReleaseDigestMismatch   = "release_digest_mismatch"
	CodeBundleSHA256Mismatch    = "bundle_sha256_mismatch"
	CodeNativeCLIMismatch       = "native_cli_mismatch"
	CodeArchiveTraversal        = "archive_traversal"
	CodeArchiveSymlinkEscape    = "archive_symlink_escape"
	CodeArchiveTooLarge         = "archive_too_large"
	CodeArchiveEntryLimit       = "archive_entry_limit"
	CodeArchiveUnsupportedEntry = "archive_unsupported_entry"
	CodeArchiveTimeBudget       = "archive_time_budget"
	CodeArchiveUnreadable       = "archive_unreadable"
	CodeManifestInvalid         = "release_manifest_invalid"
	CodeReleaseNotStaged        = "release_not_staged"
	CodeReleaseIncomplete       = "release_incomplete"
	CodeReleaseHasNoScenarios   = "release_has_no_scenarios"
	CodeStageFailed             = "stage_failed"
	CodeActivationFailed        = "activation_failed"
	CodeRollbackNotEligible     = "rollback_not_eligible"
	CodeActionNotAllowed        = "action_not_allowed"
	CodeBrokerUnavailable       = "privilege_broker_unavailable"
	CodeHostActionFailed        = "host_action_failed"
	CodeReceiptNotFound         = "receipt_not_found"
	CodeEdgeSpecInvalid         = "edge_spec_invalid"
	CodeEdgePrivateListener     = "edge_private_listener"
	CodeEdgeSnippetOutOfScope   = "edge_snippet_out_of_scope"
	CodeEdgeConfigInvalid       = "edge_config_invalid"
	CodeEdgeReloadFailed        = "edge_reload_failed"
	CodeEdgeRollbackNotEligible = "edge_rollback_not_eligible"
	CodeEdgeUnrelatedChanged    = "edge_unrelated_config_changed"
	CodeDataBindingConflict     = "data_binding_conflict"
	CodeStoreIO                 = "store_io_error"
)

// Error is the typed failure every verb returns. Exit selects the CLI exit
// code; Details carries structured context that is safe to print.
type Error struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Exit    int            `json:"-"`
	Details map[string]any `json:"details,omitempty"`
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Message == "" {
		return e.Code
	}
	return e.Code + ": " + e.Message
}

// ExitCode lets the root CLI boundary honour the typed exit code.
func (e *Error) ExitCode() int {
	if e == nil || e.Exit == 0 {
		return ExitFailed
	}
	return e.Exit
}

func refuse(code, format string, args ...any) *Error {
	return &Error{Code: code, Message: fmt.Sprintf(format, args...), Exit: ExitRefused}
}

func fail(code, format string, args ...any) *Error {
	return &Error{Code: code, Message: fmt.Sprintf(format, args...), Exit: ExitFailed}
}

func (e *Error) withDetails(details map[string]any) *Error {
	e.Details = details
	return e
}

// AsError extracts the typed error from err, wrapping an untyped one as a
// failed verb so callers always see a code.
func AsError(err error) *Error {
	if err == nil {
		return nil
	}
	var typed *Error
	if errors.As(err, &typed) {
		return typed
	}
	return &Error{Code: "verb_failed", Message: err.Error(), Exit: ExitFailed}
}
