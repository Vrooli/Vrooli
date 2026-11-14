package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"landing-page-business-suite/cli/domains/remoteprofiles"
)

func TestRemoteProfilesCreateRejectsAPIBaseWithoutV1Suffix(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() error: %v", err)
	}

	err = remoteprofiles.RunCreate(app.dependencies(), []string{
		"--tag", "prod",
		"--api-base", "https://vrooli.com",
	})
	if err == nil {
		t.Fatal("expected input validation error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "Status: INVALID INPUT") {
		t.Fatalf("expected status header, got: %v", err)
	}
	if !strings.Contains(msg, "api_base_format") {
		t.Fatalf("expected api_base_format triage, got: %v", err)
	}
	if !strings.Contains(msg, "Next Steps:") {
		t.Fatalf("expected next-step guidance, got: %v", err)
	}
}

func TestRemoteProfilesCreateAcceptsCanonicalAPIBase(t *testing.T) {
	var sawCreate bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/admin/remote-profiles":
			sawCreate = true
			var payload map[string]string
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode create payload: %v", err)
			}
			if payload["api_base"] != "https://vrooli.com/api/v1" {
				t.Fatalf("unexpected api_base: %q", payload["api_base"])
			}
			if payload["tag"] != "prod" {
				t.Fatalf("unexpected tag: %q", payload["tag"])
			}
			_, _ = w.Write([]byte(`{"id":42,"tag":"prod","api_base":"https://vrooli.com/api/v1"}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	withAPIBase(t, server.URL)
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() error: %v", err)
	}
	withAdminSession(t, app, server.URL)

	if err := remoteprofiles.RunCreate(app.dependencies(), []string{
		"--tag", "prod",
		"--label", "Production",
		"--api-base", "https://vrooli.com/api/v1/",
	}); err != nil {
		t.Fatalf("cmdRemoteProfilesCreate returned error: %v", err)
	}
	if !sawCreate {
		t.Fatal("expected remote profile create request")
	}
}
