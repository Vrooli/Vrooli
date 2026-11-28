package tts

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type trackedSpeechBody struct {
	io.Reader
	closed bool
}

func (b *trackedSpeechBody) Close() error { b.closed = true; return nil }

type staticSpeechDoer struct{ response *http.Response }

func (d staticSpeechDoer) Do(*http.Request) (*http.Response, error) { return d.response, nil }

func TestSpeechSynthesizerTransfersResponseBodyOwnership(t *testing.T) {
	body := &trackedSpeechBody{Reader: strings.NewReader("wav bytes")}
	synth := &SpeechSynthesizer{
		BaseURL: "http://tts.local", Model: "kale-tts-1-nano", SpeedMode: SpeedModeNative,
		Doer: staticSpeechDoer{response: &http.Response{
			StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"audio/wav"}},
				Body:       body,
		}},
	}

	result, contentType, err := synth.Synthesize(context.Background(), SynthesizeRequest{Input: "hello", Voice: "Kenzie", ResponseFormat: "wav"})
	if err != nil {
		t.Fatal(err)
	}
	if body.closed {
		t.Fatal("synthesizer closed the body before transferring it to its caller")
	}
	got, err := io.ReadAll(result)
	if err != nil {
		t.Fatal(err)
	}
	if err := result.Close(); err != nil {
		t.Fatal(err)
	}
	if string(got) != "wav bytes" || contentType != "audio/wav" || !body.closed {
		t.Fatalf("result=%q contentType=%q bodyClosed=%v", got, contentType, body.closed)
	}
}

func TestSpeechSynthesizerKaleRequestsWAVAndTranscodesOutput(t *testing.T) {
	var request map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/audio/speech" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "audio/wav")
		_, _ = w.Write([]byte("wav bytes"))
	}))
	defer server.Close()

	transcoded := false
	synth := &SpeechSynthesizer{BaseURL: server.URL, Model: "kale-tts-1-nano", SpeedMode: SpeedModePostprocess, Doer: server.Client(), Transcode: func(_ context.Context, data []byte, in, out string) ([]byte, error) {
		transcoded = true
		if string(data) != "wav bytes" || in != "wav" || out != "mp3" {
			t.Fatalf("transcode input = %q, %q -> %q", data, in, out)
		}
		return []byte("mp3 bytes"), nil
	}}
	body, contentType, err := synth.Synthesize(context.Background(), SynthesizeRequest{Input: "hello", Voice: "Kenzie", ResponseFormat: "mp3", Speed: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer body.Close()
	got, err := io.ReadAll(body)
	if err != nil {
		t.Fatal(err)
	}
	if request["model"] != "kale-tts-1-nano" || request["response_format"] != "wav" {
		t.Fatalf("request = %#v", request)
	}
	if _, ok := request["speed"]; ok {
		t.Fatalf("Kale runtime does not accept speed; got request %#v", request)
	}
	if string(got) != "mp3 bytes" || contentType != "audio/mpeg" || !transcoded {
		t.Fatalf("result = %q %q transcoded=%v", got, contentType, transcoded)
	}
}

func TestSpeechSynthesizerAppliesKaleSpeedLocally(t *testing.T) {
	var request map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "audio/wav")
		_, _ = w.Write([]byte("base wav"))
	}))
	defer server.Close()

	transformed := false
	synth := &SpeechSynthesizer{
		BaseURL: server.URL, Model: "kale-tts-1-nano", SpeedMode: SpeedModePostprocess, Doer: server.Client(),
		SpeedTransform: func(_ context.Context, data []byte, speed float64) ([]byte, error) {
			transformed = true
			if string(data) != "base wav" || speed != 1.5 {
				t.Fatalf("speed transform input = %q at %v", data, speed)
			}
			return []byte("sped wav"), nil
		},
	}
	body, contentType, err := synth.Synthesize(context.Background(), SynthesizeRequest{
		Input: "hello", Voice: "Kenzie", ResponseFormat: "wav", Speed: 1.5,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer body.Close()
	got, err := io.ReadAll(body)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := request["speed"]; ok {
		t.Fatalf("Kale runtime does not accept speed; got request %#v", request)
	}
	if !transformed || string(got) != "sped wav" || contentType != "audio/wav" {
		t.Fatalf("result = %q %q transformed=%v", got, contentType, transformed)
	}
}

func TestSpeechSynthesizerUsesDeclaredSpeedModeInsteadOfModelID(t *testing.T) {
	var request map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "audio/wav")
		_, _ = w.Write([]byte("base wav"))
	}))
	defer server.Close()

	transformed := false
	synth := &SpeechSynthesizer{
		BaseURL: server.URL, Model: "declared-model-id", SpeedMode: SpeedModePostprocess, Doer: server.Client(),
		SpeedTransform: func(_ context.Context, data []byte, speed float64) ([]byte, error) {
			transformed = true
			return []byte("sped wav"), nil
		},
	}
	body, _, err := synth.Synthesize(context.Background(), SynthesizeRequest{Input: "hello", Voice: "Kenzie", ResponseFormat: "wav", Speed: 1.25})
	if err != nil {
		t.Fatal(err)
	}
	defer body.Close()
	got, err := io.ReadAll(body)
	if err != nil {
		t.Fatal(err)
	}
	if request["model"] != "declared-model-id" || request["response_format"] != "wav" {
		t.Fatalf("request = %#v", request)
	}
	if _, ok := request["speed"]; ok || !transformed || string(got) != "sped wav" {
		t.Fatalf("declared postprocess mode not honored: request=%#v transformed=%v response=%q", request, transformed, got)
	}
}

func TestSpeechSynthesizerRejectsUnknownDeclaredSpeedMode(t *testing.T) {
	synth := &SpeechSynthesizer{SpeedMode: "magic"}
	if _, _, err := synth.Synthesize(context.Background(), SynthesizeRequest{}); err == nil {
		t.Fatal("unknown speed mode should fail closed")
	}
}
