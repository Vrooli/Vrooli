package main

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	ttsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/tts"
)

// mockSynthesizer implements TTSSynthesizer for testing.
type mockSynthesizer struct {
	body        string
	contentType string
	err         error
}

func (m *mockSynthesizer) Synthesize(_ context.Context, _ SynthesizeRequest) (io.ReadCloser, string, error) {
	if m.err != nil {
		return nil, "", m.err
	}
	return io.NopCloser(strings.NewReader(m.body)), m.contentType, nil
}

func newSynthesizeTestServer(synth TTSSynthesizer, capAvailable bool) *Server {
	srv := newFakeTestServer()
	srv.ttsSynthesizer = synth
	srv.ttsConfig = DefaultTTSConfig()

	checker := &fakeChecker{status: StatusUnavailable, message: "down"}
	if capAvailable {
		checker.status = StatusAvailable
		checker.message = "ok"
	}
	srv.capabilities = NewCapabilityRegistry(
		[]CapabilityDef{{ID: "kokoro-tts", Name: "Kokoro TTS"}},
		map[string]StatusChecker{"kokoro-tts": checker},
		time.Minute,
	)
	return srv
}

func TestHandleTTSSynthesize_HappyPath(t *testing.T) {
	srv := newSynthesizeTestServer(&mockSynthesizer{
		body:        "fake-audio-bytes",
		contentType: "audio/mpeg",
	}, true)

	resp, err := callTTSSynthesize(t, srv, &ttsv1.SynthesizeRequest{
		Input: "Hello world", Voice: "af_heart", ResponseFormat: "mp3",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetContentType() != "audio/mpeg" {
		t.Errorf("expected audio/mpeg, got %s", resp.GetContentType())
	}
	if string(resp.GetAudio()) != "fake-audio-bytes" {
		t.Errorf("unexpected body: %s", string(resp.GetAudio()))
	}
}

func TestHandleTTSSynthesize_503WhenCapabilityUnavailable(t *testing.T) {
	srv := newSynthesizeTestServer(&mockSynthesizer{body: "audio"}, false)

	_, err := callTTSSynthesize(t, srv, &ttsv1.SynthesizeRequest{Input: "Hello"})
	if connectCode(err) != connect.CodeUnavailable {
		t.Fatalf("expected CodeUnavailable, got %v (err=%v)", connectCode(err), err)
	}
}

func TestHandleTTSSynthesize_EmptyInput(t *testing.T) {
	srv := newSynthesizeTestServer(&mockSynthesizer{body: "audio"}, true)

	_, err := callTTSSynthesize(t, srv, &ttsv1.SynthesizeRequest{Input: "  ", Voice: "af_heart"})
	if connectCode(err) != connect.CodeInvalidArgument {
		t.Fatalf("expected CodeInvalidArgument, got %v (err=%v)", connectCode(err), err)
	}
}

func TestHandleTTSSynthesize_InputTooLong(t *testing.T) {
	srv := newSynthesizeTestServer(&mockSynthesizer{body: "audio"}, true)

	longInput := strings.Repeat("a", maxSynthesizeInputLength+1)
	_, err := callTTSSynthesize(t, srv, &ttsv1.SynthesizeRequest{Input: longInput})
	if connectCode(err) != connect.CodeInvalidArgument {
		t.Fatalf("expected CodeInvalidArgument, got %v (err=%v)", connectCode(err), err)
	}
}

func TestHandleTTSSynthesize_OnSynthesizerError(t *testing.T) {
	srv := newSynthesizeTestServer(&mockSynthesizer{err: errors.New("backend down")}, true)

	_, err := callTTSSynthesize(t, srv, &ttsv1.SynthesizeRequest{Input: "Hello"})
	if connectCode(err) != connect.CodeInternal {
		t.Fatalf("expected CodeInternal, got %v (err=%v)", connectCode(err), err)
	}
}

func TestHandleTTSSynthesize_DefaultVoiceFromConfig(t *testing.T) {
	var capturedReq SynthesizeRequest
	synth := &capturingSynthesizer{
		body:        "audio",
		contentType: "audio/mpeg",
		captured:    &capturedReq,
	}
	srv := newSynthesizeTestServer(synth, true)
	srv.ttsConfig.KokoroVoice = "bf_emma"

	_, err := callTTSSynthesize(t, srv, &ttsv1.SynthesizeRequest{Input: "Hello"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedReq.Voice != "bf_emma" {
		t.Errorf("expected voice bf_emma from config, got %s", capturedReq.Voice)
	}
}

func TestHandleTTSSynthesize_InvalidFormat(t *testing.T) {
	srv := newSynthesizeTestServer(&mockSynthesizer{body: "audio"}, true)

	_, err := callTTSSynthesize(t, srv, &ttsv1.SynthesizeRequest{Input: "Hello", ResponseFormat: "aac"})
	if connectCode(err) != connect.CodeInvalidArgument {
		t.Fatalf("expected CodeInvalidArgument, got %v (err=%v)", connectCode(err), err)
	}
}

func TestHandleTTSSynthesize_NilSynthesizer(t *testing.T) {
	srv := newSynthesizeTestServer(nil, true)
	srv.ttsSynthesizer = nil

	_, err := callTTSSynthesize(t, srv, &ttsv1.SynthesizeRequest{Input: "Hello"})
	if connectCode(err) == connect.CodeUnknown {
		t.Fatal("expected a Connect error when synthesizer is nil")
	}
}

func TestHandleTTSSynthesize_SpeedClamped(t *testing.T) {
	var capturedReq SynthesizeRequest
	synth := &capturingSynthesizer{
		body:        "audio",
		contentType: "audio/mpeg",
		captured:    &capturedReq,
	}
	srv := newSynthesizeTestServer(synth, true)

	_, err := callTTSSynthesize(t, srv, &ttsv1.SynthesizeRequest{Input: "Hello", Speed: 100})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedReq.Speed != 4.0 {
		t.Errorf("expected speed clamped to 4.0, got %f", capturedReq.Speed)
	}
}

// capturingSynthesizer captures the request for inspection.
type capturingSynthesizer struct {
	body        string
	contentType string
	captured    *SynthesizeRequest
}

func (c *capturingSynthesizer) Synthesize(_ context.Context, req SynthesizeRequest) (io.ReadCloser, string, error) {
	*c.captured = req
	return io.NopCloser(strings.NewReader(c.body)), c.contentType, nil
}
