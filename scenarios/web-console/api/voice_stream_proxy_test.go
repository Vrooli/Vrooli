package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/websocket"
)

type proxyURLResolver struct {
	url string
	err error
}

func (r proxyURLResolver) Resolve() (string, error) { return r.url, r.err }

func TestVoiceStreamProxyURLAndUnavailablePaths(t *testing.T) {
	got, err := buildUpstreamWS("https://audio.example/base/", "format=webm")
	if err != nil || got != "wss://audio.example/base/api/v1/voice/stream?format=webm" {
		t.Fatalf("upstream URL = %q/%v", got, err)
	}
	got, err = buildUpstreamWS("http://audio.example", "")
	if err != nil || got != "ws://audio.example/api/v1/voice/stream" {
		t.Fatalf("plain upstream URL = %q/%v", got, err)
	}
	if _, err := buildUpstreamWS("://bad", ""); err == nil {
		t.Fatal("invalid upstream URL unexpectedly parsed")
	}

	for _, proxy := range []*voiceStreamProxy{
		nil,
		newVoiceStreamProxy(nil),
		newVoiceStreamProxy(proxyURLResolver{err: errors.New("offline")}),
	} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/voice/stream", nil)
		proxy.ServeHTTP(rec, req)
		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("proxy unavailable status = %d", rec.Code)
		}
	}
	if isExpectedWSClose(&websocket.CloseError{Code: websocket.CloseNormalClosure}) == false {
		t.Fatal("normal WebSocket close was not classified as expected")
	}
	if isExpectedWSClose(errors.New("unexpected read failure")) {
		t.Fatal("generic WebSocket failure was classified as expected")
	}
}
