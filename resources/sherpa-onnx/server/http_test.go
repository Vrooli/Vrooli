package main

import (
	"context"
	"encoding/binary"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeTTS struct{}

func (fakeTTS) Synthesize(context.Context, string, int, float32) (Audio, error) {
	return Audio{SampleRate: 24000, Samples: []float32{0, 0.5, -0.5, 0}}, nil
}

func (fakeTTS) Close() {}

type fakeEncoder struct {
	data        []byte
	contentType string
	err         error
}

func (f fakeEncoder) Encode(context.Context, Audio, string) ([]byte, string, error) {
	return f.data, f.contentType, f.err
}

func TestVoicesPreserveKokoroContract(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/v1/audio/voices", nil)
	w := httptest.NewRecorder()
	newHandlerWithEncoder(fakeTTS{}, fakeEncoder{data: []byte("encoded"), contentType: "audio/wav"}).ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	body, _ := io.ReadAll(w.Result().Body)
	if string(body) != "[\"af_heart\",\"af_bella\",\"af_nicole\",\"af_sarah\",\"af_sky\",\"am_adam\",\"am_michael\",\"bf_emma\",\"bf_isabella\",\"bm_george\",\"bm_lewis\"]\n" {
		t.Fatalf("voices = %s", body)
	}
}

func TestSpeechReturnsValidWAV(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/v1/audio/speech", strings.NewReader(`{"model":"kokoro","input":"hello","voice":"af_heart","response_format":"wav"}`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	newHandler(fakeTTS{}).ServeHTTP(w, r)
	if w.Code != http.StatusOK || w.Header().Get("Content-Type") != "audio/wav" {
		t.Fatalf("status/content type = %d/%q", w.Code, w.Header().Get("Content-Type"))
	}
	body, _ := io.ReadAll(w.Result().Body)
	if string(body[:4]) != "RIFF" || string(body[8:12]) != "WAVE" || binary.LittleEndian.Uint32(body[40:44]) != 8 {
		t.Fatalf("invalid WAV header: %x", body[:44])
	}
}

func TestSpeechRefusesUnencodedFormats(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/v1/audio/speech", strings.NewReader(`{"input":"hello","voice":"af_heart","response_format":"mp3"}`))
	w := httptest.NewRecorder()
	newHandlerWithEncoder(fakeTTS{}, fakeEncoder{err: context.Canceled}).ServeHTTP(w, r)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestCodecArgsCoverKokoroFormats(t *testing.T) {
	for _, tc := range []struct {
		format      string
		contentType string
	}{
		{format: "mp3", contentType: "audio/mpeg"},
		{format: "opus", contentType: "audio/ogg"},
		{format: "flac", contentType: "audio/flac"},
	} {
		_, contentType, ok := codecArgsFor(tc.format)
		if !ok || contentType != tc.contentType {
			t.Fatalf("codecArgsFor(%q) = ok=%v content-type=%q", tc.format, ok, contentType)
		}
	}
}
