package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"landing-page-business-suite/cli/domains/remoteprofiles"
)

func TestRemoteProfilesProxyAcceptsProfileTagSelector(t *testing.T) {
	var sawList, sawProxy bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/admin/remote-profiles":
			sawList = true
			_, _ = w.Write([]byte(`{"profiles":[{"id":12,"tag":"prod"}]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/admin/remote-profiles/12/proxy":
			sawProxy = true
			var payload map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode proxy payload: %v", err)
			}
			if payload["method"] != "GET" {
				t.Fatalf("unexpected method: %v", payload["method"])
			}
			if payload["path"] != "/admin/download-apps" {
				t.Fatalf("unexpected path: %v", payload["path"])
			}
			_, _ = w.Write([]byte(`{"ok":true}`))
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

	if err := app.Run([]string{
		"remote-profiles-proxy",
		"--profile-tag", "prod",
		"--method", "GET",
		"--path", "/admin/download-apps",
		"--json",
	}); err != nil {
		t.Fatalf("cmdRemoteProfilesProxy returned error: %v", err)
	}
	if !sawList {
		t.Fatal("expected remote profiles list request for tag resolution")
	}
	if !sawProxy {
		t.Fatal("expected remote proxy request")
	}
}

func TestRemoteProfilesProxyRejectsMixingIDAndProfileTag(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() error: %v", err)
	}
	err = remoteprofiles.RunProxy(app.dependencies(), []string{
		"12",
		"--profile-tag", "prod",
		"--method", "GET",
		"--path", "/admin/download-apps",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "use either --profile-tag/--tag or an explicit profile id") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRemoteProfilesProxyPositionalIDStillWorks(t *testing.T) {
	var sawProxy bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/admin/remote-profiles/12/proxy" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		sawProxy = true
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	withAPIBase(t, server.URL)
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() error: %v", err)
	}
	withAdminSession(t, app, server.URL)

	if err := app.Run([]string{
		"remote-profiles-proxy",
		"12",
		"--method", "GET",
		"--path", "/admin/download-apps",
		"--json",
	}); err != nil {
		t.Fatalf("cmdRemoteProfilesProxy returned error: %v", err)
	}
	if !sawProxy {
		t.Fatal("expected remote proxy request")
	}
}

func TestRemoteProfilesDownloadStorageTest_UsesProxyEndpoint(t *testing.T) {
	var sawList, sawProxy bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/admin/remote-profiles":
			sawList = true
			_, _ = w.Write([]byte(`{"profiles":[{"id":12,"tag":"prod"}]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/admin/remote-profiles/12/proxy":
			sawProxy = true
			var payload map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode proxy payload: %v", err)
			}
			if payload["method"] != "POST" {
				t.Fatalf("unexpected method: %v", payload["method"])
			}
			if payload["path"] != "/admin/download-storage/test" {
				t.Fatalf("unexpected path: %v", payload["path"])
			}
			_, _ = w.Write([]byte(`{"ok":true}`))
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

	if err := app.Run([]string{
		"remote-profiles-download-storage-test",
		"--profile-tag", "prod",
		"--json",
	}); err != nil {
		t.Fatalf("cmdRemoteProfilesDownloadStorageTest returned error: %v", err)
	}
	if !sawList {
		t.Fatal("expected remote profiles list request for tag resolution")
	}
	if !sawProxy {
		t.Fatal("expected remote proxy request")
	}
}

func TestRemoteProfilesDownloadAppsList_UsesProxyEndpoint(t *testing.T) {
	var sawProxy bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/admin/remote-profiles/12/proxy" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		sawProxy = true
		var payload map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode proxy payload: %v", err)
		}
		if payload["method"] != "GET" {
			t.Fatalf("unexpected method: %v", payload["method"])
		}
		if payload["path"] != "/admin/download-apps" {
			t.Fatalf("unexpected path: %v", payload["path"])
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	withAPIBase(t, server.URL)
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() error: %v", err)
	}
	withAdminSession(t, app, server.URL)

	if err := app.Run([]string{
		"remote-profiles-download-apps-list",
		"12",
		"--json",
	}); err != nil {
		t.Fatalf("cmdRemoteProfilesDownloadAppsList returned error: %v", err)
	}
	if !sawProxy {
		t.Fatal("expected remote proxy request")
	}
}
