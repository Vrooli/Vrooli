package stt

import (
	"context"
	"strings"
	"testing"

	"connectrpc.com/connect"

	"audio-tools/internal/ai/sttchain"
	sttpkg "audio-tools/internal/stt"
	sttpipeline "audio-tools/internal/stt/pipeline"

	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
)

// A transient backend-down maps to Unavailable; an operator-action one to
// FailedPrecondition. Neither leaks transport detail (plan L2).
func TestMapChainErrorBackendDown(t *testing.T) {
	transient := &sttpipeline.STTBackendError{Resource: "whisper", Transient: true, Message: "Speech backend (whisper) is starting — please try again in a moment."}
	if code := connect.CodeOf(mapChainError(transient)); code != connect.CodeUnavailable {
		t.Errorf("transient backend-down code = %s, want Unavailable", code)
	}
	operator := &sttpipeline.STTBackendError{Resource: "whisper", Transient: false, Message: "Speech backend (whisper) is unavailable — run `vrooli resource start whisper` and try again."}
	if code := connect.CodeOf(mapChainError(operator)); code != connect.CodeFailedPrecondition {
		t.Errorf("operator-action backend-down code = %s, want FailedPrecondition", code)
	}
	degraded := &sttpipeline.STTBackendError{Resource: "whisper", Transient: true, State: sttpipeline.STTBackendStateDegraded, Message: "Speech backend (whisper) is slow or degraded — please try again shortly."}
	if code := connect.CodeOf(mapChainError(degraded)); code != connect.CodeUnavailable {
		t.Errorf("degraded backend code = %s, want Unavailable", code)
	}
	for _, e := range []error{transient, operator, degraded} {
		if msg := mapChainError(e).Error(); strings.Contains(msg, "dial") || strings.Contains(msg, "tcp") {
			t.Errorf("mapped connect error leaked transport detail: %q", msg)
		}
	}
}

func TestResponseFromResultProjectsProviderAndPolicyDetails(t *testing.T) {
	h := &connectHandler{}
	res := &sttchain.Result{Text: "hello", DetectedLanguage: "en", DurationSeconds: 2, ProviderID: "whisper", ModelID: "medium"}
	resp := h.responseFromResult(context.Background(), res, []byte("audio"), sttpkg.StreamConfig{EngineID: "kyutai-stt"})
	if resp.GetText() != "hello" || resp.GetProviderId() != "whisper" || resp.GetModelId() != "medium" {
		t.Fatalf("response = %+v", resp)
	}
	if resp.GetPolicyDetails()["selected_engine"] != "kyutai-stt" || resp.GetPolicyDetails()["engine_substitution"] != "true" {
		t.Fatalf("policy details = %#v", resp.GetPolicyDetails())
	}
}

// The streaming-error event carries a distinct code and a clean message for the
// typed backend error (the UI keys off the code for a "starting…" affordance).
func TestProtoForEventBackendDown(t *testing.T) {
	transient := &sttpipeline.STTBackendError{Resource: "whisper", Transient: true, Message: "Speech backend (whisper) is starting — please try again in a moment."}
	ev := protoForEvent(sttchain.StreamEvent{Kind: sttchain.StreamEventError, Error: transient})
	se := ev.GetError()
	if se == nil {
		t.Fatal("expected a StreamError event")
	}
	if se.GetCode() != "backend_starting" {
		t.Errorf("code = %q, want backend_starting", se.GetCode())
	}
	if strings.Contains(se.GetMessage(), "dial") || strings.Contains(se.GetMessage(), "tcp") {
		t.Errorf("stream error message leaked transport detail: %q", se.GetMessage())
	}

	operator := &sttpipeline.STTBackendError{Resource: "whisper", Transient: false, Message: "Speech backend (whisper) is unavailable — run `vrooli resource start whisper` and try again."}
	ev2 := protoForEvent(sttchain.StreamEvent{Kind: sttchain.StreamEventError, Error: operator})
	if got := ev2.GetError().GetCode(); got != "backend_unavailable" {
		t.Errorf("code = %q, want backend_unavailable", got)
	}
	degraded := &sttpipeline.STTBackendError{Resource: "whisper", Transient: true, State: sttpipeline.STTBackendStateDegraded, Message: "Speech backend (whisper) is slow or degraded — please try again shortly."}
	ev3 := protoForEvent(sttchain.StreamEvent{Kind: sttchain.StreamEventError, Error: degraded})
	if got := ev3.GetError().GetCode(); got != "backend_degraded" {
		t.Errorf("code = %q, want backend_degraded", got)
	}

	// A generic provider error keeps the legacy code (no regression).
	generic := protoForEvent(sttchain.StreamEvent{Kind: sttchain.StreamEventError, Error: errString("boom")})
	if got := generic.GetError().GetCode(); got != "provider_failure" {
		t.Errorf("generic error code = %q, want provider_failure", got)
	}
}

// errString is a minimal error wrapper used to assert non-typed errors fall
// through to the legacy code.
type errString string

func (e errString) Error() string { return string(e) }

// guard against unused import when proto helpers change shape.
var _ = sttv1.StreamError{}
