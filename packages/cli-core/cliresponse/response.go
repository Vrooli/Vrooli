// Package cliresponse decodes the response-body contract shared by Vrooli CLIs.
package cliresponse

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Decode unmarshals body as the caller's concrete shape. Bodies with a
// non-nil success field and data payload are unwrapped; other bodies are used
// directly.
func Decode(body []byte, dest interface{}) error {
	if raw, ok := unwrapEnvelope(body); ok {
		if err := json.Unmarshal(raw, dest); err != nil {
			return fmt.Errorf("parse response data: %w", err)
		}
		return nil
	}
	if err := json.Unmarshal(body, dest); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}
	return nil
}

// EnvelopeMessage returns the trimmed message from a success/data envelope.
func EnvelopeMessage(body []byte) string {
	raw, ok := unwrapEnvelope(body)
	if !ok {
		return ""
	}
	var wrapper struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(raw, &wrapper); err != nil {
		return ""
	}
	return strings.TrimSpace(wrapper.Message)
}

func unwrapEnvelope(body []byte) (json.RawMessage, bool) {
	trimmed := strings.TrimSpace(string(body))
	if !strings.HasPrefix(trimmed, "{") {
		return nil, false
	}
	var envelope struct {
		Success *bool           `json:"success"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, false
	}
	if envelope.Success == nil || len(envelope.Data) == 0 {
		return nil, false
	}
	return envelope.Data, true
}
