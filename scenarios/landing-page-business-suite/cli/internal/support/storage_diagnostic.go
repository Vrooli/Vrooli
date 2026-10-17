package support

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

// StorageDiagnostic mirrors the delivery package's safe, serializable storage
// failure explanation. It carries no secret material and lets the CLI report
// the exact cause and remediation instead of a generic denial sentence.
type StorageDiagnostic struct {
	Code        string `json:"code"`
	Summary     string `json:"summary"`
	Operation   string `json:"operation,omitempty"`
	Bucket      string `json:"bucket,omitempty"`
	Region      string `json:"region,omitempty"`
	Remediation string `json:"remediation,omitempty"`
	Retryable   bool   `json:"retryable"`
	RequestID   string `json:"request_id,omitempty"`
	HTTPStatus  int    `json:"http_status,omitempty"`
}

type storageTestEnvelope struct {
	Error      string             `json:"error"`
	ErrorType  string             `json:"error_type"`
	Diagnostic *StorageDiagnostic `json:"diagnostic"`
	Validation *struct {
		Ready      bool               `json:"ready"`
		Diagnostic *StorageDiagnostic `json:"diagnostic"`
	} `json:"validation"`
}

// StorageDiagnosticFromError extracts the structured delivery-storage
// diagnostic from an admin API error. It returns false when the error carries
// no structured diagnostic, so callers can fall back to the plain message.
func StorageDiagnosticFromError(err error) (*StorageDiagnostic, bool) {
	if err == nil {
		return nil, false
	}
	var apiErr *cliutil.APIError
	if !errors.As(err, &apiErr) || len(apiErr.RawResponse) == 0 {
		return nil, false
	}
	var envelope storageTestEnvelope
	if jsonErr := json.Unmarshal(apiErr.RawResponse, &envelope); jsonErr != nil {
		return nil, false
	}
	if envelope.Validation != nil && envelope.Validation.Diagnostic != nil {
		return envelope.Validation.Diagnostic, true
	}
	if envelope.Diagnostic != nil {
		return envelope.Diagnostic, true
	}
	return nil, false
}

// DescribeStorageFailure renders a storage test failure with its machine
// code, operation, HTTP status, request id, and remediation. It falls back to
// the error text when no structured diagnostic is present.
func DescribeStorageFailure(err error) string {
	diagnostic, ok := StorageDiagnosticFromError(err)
	if !ok {
		if err != nil {
			return err.Error()
		}
		return "delivery storage check failed"
	}
	parts := []string{diagnostic.Summary}
	if diagnostic.Code != "" {
		parts = append(parts, fmt.Sprintf("code=%s", diagnostic.Code))
	}
	if diagnostic.Operation != "" {
		parts = append(parts, fmt.Sprintf("operation=%s", diagnostic.Operation))
	}
	if diagnostic.HTTPStatus != 0 {
		parts = append(parts, fmt.Sprintf("http_status=%d", diagnostic.HTTPStatus))
	}
	if diagnostic.RequestID != "" {
		parts = append(parts, fmt.Sprintf("request_id=%s", diagnostic.RequestID))
	}
	if diagnostic.Remediation != "" {
		parts = append(parts, diagnostic.Remediation)
	}
	return strings.Join(parts, " | ")
}
