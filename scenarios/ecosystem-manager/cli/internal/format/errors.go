// Package format provides CLI output helpers for structured API errors and mutation results.
package format

import (
	"errors"
	"fmt"

	"github.com/vrooli/cli-core/cliutil"
)

// WrapAPIError wraps an error with a prefix. If the error is a structured APIError,
// it uses FormatConcise() for rich recovery output; otherwise falls back to simple wrapping.
func WrapAPIError(prefix string, err error) error {
	var apiErr *cliutil.APIError
	if errors.As(err, &apiErr) && apiErr.IsStructured() {
		return fmt.Errorf("%s\n%s", prefix, apiErr.FormatConcise())
	}
	return fmt.Errorf("%s: %w", prefix, err)
}
