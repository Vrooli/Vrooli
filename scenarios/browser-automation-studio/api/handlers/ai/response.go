package ai

import (
	"github.com/vrooli/browser-automation-studio/internal/apierror"
)

// Re-export from shared apierror package for backward compatibility within this package.
// This eliminates the previous code duplication while maintaining the same API surface.

type APIError = apierror.APIError

var (
	ErrInvalidRequest       = apierror.ErrInvalidRequest
	ErrMissingRequiredField = apierror.ErrMissingRequiredField
	ErrInternalServer       = apierror.ErrInternalServer
)

var RespondError = apierror.RespondError
var RespondSuccess = apierror.RespondSuccess
