package cliapp

import (
	"errors"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
)

func TestDecodeEnvelopeEmpty(t *testing.T) {
	if _, ok := DecodeEnvelope(nil); ok {
		t.Error("nil body should not decode")
	}
	if _, ok := DecodeEnvelope([]byte{}); ok {
		t.Error("empty body should not decode")
	}
}

func TestDecodeEnvelopeNotJSON(t *testing.T) {
	if _, ok := DecodeEnvelope([]byte("plain text")); ok {
		t.Error("non-JSON body should not decode")
	}
}

func TestDecodeEnvelopeMissingCode(t *testing.T) {
	body := []byte(`{"message": "something failed"}`)
	if _, ok := DecodeEnvelope(body); ok {
		t.Error("body without code should not decode")
	}
}

func TestDecodeEnvelopeValid(t *testing.T) {
	body := []byte(`{"code":"not_found","message":"note 42 not found","details":{"id":"42"}}`)
	env, ok := DecodeEnvelope(body)
	if !ok {
		t.Fatal("expected envelope to decode")
	}
	if env.Code != "not_found" {
		t.Errorf("code: got %q", env.Code)
	}
	if env.Message != "note 42 not found" {
		t.Errorf("message: got %q", env.Message)
	}
	if env.Details["id"] != "42" {
		t.Errorf("details: got %v", env.Details)
	}
}

func TestWrapAPIErrorNil(t *testing.T) {
	if err := WrapAPIError("foo", nil, nil); err != nil {
		t.Errorf("nil err should produce nil, got %v", err)
	}
}

func TestWrapAPIErrorWithBodyEnvelope(t *testing.T) {
	body := []byte(`{"code":"invalid_request","message":"title required"}`)
	err := WrapAPIError("create note", errors.New("transport"), body)
	if err == nil || !strings.Contains(err.Error(), "create note: invalid_request: title required") {
		t.Errorf("unexpected: %v", err)
	}
}

func TestWrapAPIErrorWithAPIErrorEnvelope(t *testing.T) {
	apiErr := &cliutil.APIError{
		StatusCode:  404,
		Message:     "fallback",
		RawResponse: []byte(`{"code":"not_found","message":"note 42 not found"}`),
	}
	err := WrapAPIError("get note", apiErr, nil)
	if err == nil || !strings.Contains(err.Error(), "not_found: note 42 not found") {
		t.Errorf("expected envelope-derived message, got %v", err)
	}
}

func TestWrapAPIErrorWithAPIErrorNoEnvelope(t *testing.T) {
	apiErr := &cliutil.APIError{
		StatusCode:  500,
		Message:     "internal server error",
		RawResponse: []byte("plain text"),
	}
	err := WrapAPIError("list notes", apiErr, nil)
	if err == nil || !strings.Contains(err.Error(), "list notes:") {
		t.Errorf("expected wrapped APIError, got %v", err)
	}
	if !errors.Is(err, apiErr) {
		t.Error("expected errors.Is to find the APIError")
	}
}

func TestWrapAPIErrorPlainError(t *testing.T) {
	plain := errors.New("network down")
	err := WrapAPIError("ping", plain, nil)
	if !errors.Is(err, plain) {
		t.Error("expected errors.Is to find plain error")
	}
	if !strings.Contains(err.Error(), "ping: network down") {
		t.Errorf("got %v", err)
	}
}

func TestWrapAPIErrorEmptyAction(t *testing.T) {
	err := WrapAPIError("", errors.New("boom"), nil)
	if !strings.HasPrefix(err.Error(), "request:") {
		t.Errorf("empty action should default to 'request', got %v", err)
	}
}
