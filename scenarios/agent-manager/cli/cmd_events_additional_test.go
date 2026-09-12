package main

import (
	"encoding/json"
	"testing"
)

func TestHTTPToWSURLPreservesEndpointAndMapsSupportedSchemes(t *testing.T) {
	cases := map[string]string{
		"http://example.test/api/v1/ws?token=x": "ws://example.test/api/v1/ws?token=x",
		"https://example.test/api/v1/ws":        "wss://example.test/api/v1/ws",
		"ws://example.test/api/v1/ws":           "ws://example.test/api/v1/ws",
		"wss://example.test/api/v1/ws":          "wss://example.test/api/v1/ws",
	}
	for in, want := range cases {
		got, err := httpToWSURL(in)
		if err != nil || got != want {
			t.Fatalf("httpToWSURL(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	if _, err := httpToWSURL("ftp://example.test/events"); err == nil {
		t.Fatal("unsupported scheme unexpectedly succeeded")
	}
}

func TestHandleWSMessageFiltersOtherRunsAndRejectsMalformedKnownPayloads(t *testing.T) {
	app := &App{}
	if err := app.handleWSMessage(WebSocketMessage{Type: "run_event", RunID: "other", Payload: json.RawMessage(`{`)}, "target"); err != nil {
		t.Fatalf("message for another run should be ignored: %v", err)
	}
	for _, kind := range []string{"run_event", "run_progress", "run_status"} {
		if err := app.handleWSMessage(WebSocketMessage{Type: kind, Payload: json.RawMessage(`{`)}, "target"); err == nil {
			t.Fatalf("malformed %s payload unexpectedly succeeded", kind)
		}
	}
	for _, kind := range []string{"connected", "pong"} {
		if err := app.handleWSMessage(WebSocketMessage{Type: kind}, "target"); err != nil {
			t.Fatalf("%s = %v", kind, err)
		}
	}
}

func TestHandleWSMessageDisplaysEachSupportedPayloadShape(t *testing.T) {
	app := &App{}
	cases := []WebSocketMessage{
		{Type: "run_event", RunID: "target", Payload: json.RawMessage(`{"sequence":3,"eventType":"message","data":{"content":"hello"}}`)},
		{Type: "run_progress", RunID: "target", Payload: json.RawMessage(`{"percentComplete":50,"phase":"executing","currentAction":"working"}`)},
		{Type: "run_status", RunID: "target", Payload: json.RawMessage(`{"status":"running"}`)},
		{Type: "run_status", RunID: "target", Payload: json.RawMessage(`{"status":"completed"}`)},
		{Type: "diagnostic", RunID: "target", Payload: json.RawMessage(`{"detail":"trace"}`)},
	}
	for _, msg := range cases {
		if err := app.handleWSMessage(msg, "target"); err != nil {
			t.Fatalf("handleWSMessage(%s): %v", msg.Type, err)
		}
	}
}
