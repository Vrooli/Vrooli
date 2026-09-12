package cliapp

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliutil"
)

// ErrorEnvelope is the canonical non-2xx body shape across all scenarios.
//
// Scenarios define this in proto (vrooli.<scenario>.v1.errors.ErrorEnvelope),
// but cli-core can't import scenario-specific generated code. Instead, we
// decode the wire JSON directly into this struct — the protojson wire format
// uses the same field names (snake_case `code`, `message`, `details`), so
// this round-trips faithfully without dragging proto into cli-core.
type ErrorEnvelope struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
}

// DecodeEnvelope attempts to decode body as an ErrorEnvelope. Returns the
// envelope and true on success; nil and false when body is empty, not JSON,
// or missing the `code` field.
//
// Use WrapAPIError instead unless you need the typed envelope directly.
func DecodeEnvelope(body []byte) (*ErrorEnvelope, bool) {
	if len(body) == 0 {
		return nil, false
	}
	var env ErrorEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, false
	}
	if env.Code == "" {
		return nil, false
	}
	return &env, true
}

// WrapAPIError produces a human-readable, envelope-aware error string for a
// failed API call. action is a short verb phrase ("create note", "list
// tasks") used as the leading context.
//
// Lookup order for the envelope:
//  1. The body the caller passed (from a 2xx response with body, rare).
//  2. The *cliutil.APIError's RawResponse (the common non-2xx path).
//  3. Neither — wrap the underlying error.
//
// This is the common helper that replaced per-scenario apiError helpers.
func WrapAPIError(action string, err error, body []byte) error {
	if err == nil {
		return nil
	}
	action = strings.TrimSpace(action)
	if action == "" {
		action = "request"
	}

	if env, ok := DecodeEnvelope(body); ok {
		return fmt.Errorf("%s: %s: %s", action, env.Code, env.Message)
	}

	var connectErr *connect.Error
	if errors.As(err, &connectErr) {
		return fmt.Errorf("%s: %s: %s", action, connectErr.Code().String(), connectErr.Message())
	}

	if apiErr, ok := err.(*cliutil.APIError); ok {
		if env, ok := DecodeEnvelope(apiErr.RawResponse); ok {
			return fmt.Errorf("%s: %s: %s", action, env.Code, env.Message)
		}
		return fmt.Errorf("%s: %w", action, apiErr)
	}
	return fmt.Errorf("%s: %w", action, err)
}
