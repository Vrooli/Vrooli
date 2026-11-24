package drills

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFailureDrillDriverCallersShareJSONTransportOwner(t *testing.T) {
	type observedRequest struct {
		method, path, contentType, adminSecret, body string
	}
	var requests []observedRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
		}
		requests = append(requests, observedRequest{
			method: r.Method, path: r.URL.Path, contentType: r.Header.Get("Content-Type"),
			adminSecret: r.Header.Get("X-Playwright-Admin-Secret"), body: string(body),
		})
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			_, _ = io.WriteString(w, `{"accepted":true}`)
			return
		}
		_, _ = io.WriteString(w, `{"faults":[],"audit":[]}`)
	}))
	defer server.Close()

	svc := &service{deps: Deps{DriverURL: server.URL, AdminSecret: "test-local-secret", HTTPClient: server.Client()}}
	var postOutput struct {
		Accepted bool `json:"accepted"`
	}
	if err := svc.post(context.Background(), "/test-control/faults/arm", map[string]string{"token": "local-token"}, &postOutput); err != nil {
		t.Fatalf("post() error = %v", err)
	}
	snapshot, err := svc.snapshot(context.Background())
	if err != nil {
		t.Fatalf("snapshot() error = %v", err)
	}
	if !postOutput.Accepted || len(snapshot.Faults) != 0 || len(snapshot.Audit) != 0 {
		t.Fatalf("unexpected decoded callers: post=%+v snapshot=%+v", postOutput, snapshot)
	}
	if len(requests) != 2 {
		t.Fatalf("request count = %d, want 2", len(requests))
	}
	if got := requests[0]; got.method != http.MethodPost || got.path != "/test-control/faults/arm" || got.contentType != "application/json" || got.adminSecret != "test-local-secret" || !strings.Contains(got.body, `"token":"local-token"`) {
		t.Fatalf("unexpected POST transport: %+v", got)
	}
	if got := requests[1]; got.method != http.MethodGet || got.path != "/test-control/faults" || got.adminSecret != "test-local-secret" || got.contentType != "" {
		t.Fatalf("unexpected snapshot transport: %+v", got)
	}
	var posted map[string]string
	if err := json.Unmarshal([]byte(requests[0].body), &posted); err != nil || posted["token"] != "local-token" {
		t.Fatalf("POST payload = %q, decode error = %v", requests[0].body, err)
	}
}
