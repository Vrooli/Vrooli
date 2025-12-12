package handlers

import (
	"net/http"

	"github.com/vrooli/browser-automation-studio/internal/apierror"
)

// APIError is re-exported from internal/apierror for backward compatibility.
// All error types are now centralized in the apierror package to avoid duplication.
type APIError = apierror.APIError

// respondError writes a JSON error response with the appropriate status code.
func (h *Handler) respondError(w http.ResponseWriter, err *APIError) {
	apierror.RespondError(w, err)
}

// respondSuccess writes a JSON success response.
func (h *Handler) respondSuccess(w http.ResponseWriter, statusCode int, data any) {
	apierror.RespondSuccess(w, statusCode, data)
}

// RespondError is re-exported for use in subpackages.
var RespondError = apierror.RespondError

// RespondSuccess is re-exported for use in subpackages.
var RespondSuccess = apierror.RespondSuccess

// Re-export all predefined errors from apierror package for backward compatibility.
// 400 Bad Request errors
var (
	ErrInvalidRequest           = apierror.ErrInvalidRequest
	ErrInvalidWorkflowID        = apierror.ErrInvalidWorkflowID
	ErrInvalidProjectID         = apierror.ErrInvalidProjectID
	ErrInvalidExecutionID       = apierror.ErrInvalidExecutionID
	ErrInvalidScheduleID        = apierror.ErrInvalidScheduleID
	ErrInvalidCronExpression    = apierror.ErrInvalidCronExpression
	ErrInvalidTimezone          = apierror.ErrInvalidTimezone
	ErrMissingRequiredField     = apierror.ErrMissingRequiredField
	ErrInvalidWorkflowPayload   = apierror.ErrInvalidWorkflowPayload
	ErrWorkflowValidationFailed = apierror.ErrWorkflowValidationFailed
	ErrCaseExpectationMissing   = apierror.ErrCaseExpectationMissing
)

// 404 Not Found errors
var (
	ErrWorkflowNotFound        = apierror.ErrWorkflowNotFound
	ErrProjectNotFound         = apierror.ErrProjectNotFound
	ErrProjectFileNotFound     = apierror.ErrProjectFileNotFound
	ErrWorkflowVersionNotFound = apierror.ErrWorkflowVersionNotFound
	ErrExecutionNotFound       = apierror.ErrExecutionNotFound
	ErrScreenshotNotFound      = apierror.ErrScreenshotNotFound
	ErrScheduleNotFound        = apierror.ErrScheduleNotFound
)

// 409 Conflict errors
var (
	ErrProjectAlreadyExists  = apierror.ErrProjectAlreadyExists
	ErrWorkflowAlreadyExists = apierror.ErrWorkflowAlreadyExists
	ErrConflict              = apierror.ErrConflict
)

// 500 Internal Server Error
var (
	ErrInternalServer          = apierror.ErrInternalServer
	ErrDatabaseError           = apierror.ErrDatabaseError
	ErrWorkflowExecutionFailed = apierror.ErrWorkflowExecutionFailed
)

// Other status codes
var (
	ErrAIServiceError     = apierror.ErrAIServiceError
	ErrServiceUnavailable = apierror.ErrServiceUnavailable
	ErrRequestTooLarge    = apierror.ErrRequestTooLarge
	ErrRequestTimeout     = apierror.ErrRequestTimeout
)
