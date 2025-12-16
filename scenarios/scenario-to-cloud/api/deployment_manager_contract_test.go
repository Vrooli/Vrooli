package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPackagerAPISurface_DeploymentManagerContract(t *testing.T) {
	// [REQ:STC-P0-007] packager API surface for deployment-manager
	//
	// This test avoids executing real SSH/SCP/network operations; it validates route presence,
	// request/response JSON envelopes, and stable HTTP status semantics for orchestration.
	t.Setenv("API_PORT", "0")
	srv, err := NewServer()
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	ts := httptest.NewServer(srv.router)
	defer ts.Close()

	manifest := map[string]interface{}{
		"version": "1.0.0",
		"target": map[string]interface{}{
			"type": "vps",
			"vps": map[string]interface{}{
				"host": "203.0.113.10",
			},
		},
		"scenario": map[string]interface{}{
			"id": "landing-page-business-suite",
		},
		"dependencies": map[string]interface{}{
			"scenarios": []string{"landing-page-business-suite"},
			"resources": []string{},
			"analyzer": map[string]interface{}{
				"tool": "scenario-dependency-analyzer",
			},
		},
		"bundle": map[string]interface{}{
			"include_packages": true,
			"include_autoheal": true,
			"scenarios":        []string{"landing-page-business-suite", "vrooli-autoheal"},
		},
		"ports": map[string]interface{}{
			"ui":  3000,
			"api": 3001,
			"ws":  3002,
		},
		"edge": map[string]interface{}{
			"domain": "example.com",
			"caddy": map[string]interface{}{
				"enabled": true,
				"email":   "ops@example.com",
			},
		},
	}

	postJSON := func(path string, payload any) (*http.Response, []byte) {
		t.Helper()
		var body io.Reader
		if payload == nil {
			body = bytes.NewReader([]byte("not-json"))
		} else {
			b, err := json.Marshal(payload)
			if err != nil {
				t.Fatalf("marshal %s: %v", path, err)
			}
			body = bytes.NewReader(b)
		}
		resp, err := http.Post(ts.URL+path, "application/json", body)
		if err != nil {
			t.Fatalf("post %s: %v", path, err)
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		return resp, b
	}

	assertInvalidJSON := func(path string) {
		t.Helper()
		resp, body := postJSON(path, nil)
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("%s status: got %d body=%s", path, resp.StatusCode, string(body))
		}
		var env APIErrorEnvelope
		if err := json.Unmarshal(body, &env); err != nil {
			t.Fatalf("%s unmarshal error envelope: %v body=%s", path, err, string(body))
		}
		if env.Error.Code != "invalid_json" {
			t.Fatalf("%s expected invalid_json, got %+v", path, env.Error)
		}
	}

	// Consumer-side contract: validate and plan accept the manifest directly.
	resp, body := postJSON("/api/v1/manifest/validate", manifest)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("validate status: got %d body=%s", resp.StatusCode, string(body))
	}
	resp, body = postJSON("/api/v1/plan", manifest)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("plan status: got %d body=%s", resp.StatusCode, string(body))
	}

	// Packager operations: build bundle, preflight, setup/deploy/inspect plan+apply.
	// Setup/deploy/inspect plan endpoints must accept the canonical wrapper payloads.
	resp, body = postJSON("/api/v1/vps/setup/plan", map[string]any{
		"manifest":    manifest,
		"bundle_path": "/tmp/mini-vrooli.tar.gz",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("vps/setup/plan status: got %d body=%s", resp.StatusCode, string(body))
	}
	resp, body = postJSON("/api/v1/vps/deploy/plan", map[string]any{
		"manifest": manifest,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("vps/deploy/plan status: got %d body=%s", resp.StatusCode, string(body))
	}
	resp, body = postJSON("/api/v1/vps/inspect/plan", map[string]any{
		"manifest": manifest,
		"options": map[string]any{
			"tail_lines": 20,
		},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("vps/inspect/plan status: got %d body=%s", resp.StatusCode, string(body))
	}

	// For endpoints that could perform real local/remote operations, assert they reject invalid JSON.
	assertInvalidJSON("/api/v1/bundle/build")
	assertInvalidJSON("/api/v1/preflight")
	assertInvalidJSON("/api/v1/vps/setup/apply")
	assertInvalidJSON("/api/v1/vps/deploy/apply")
	assertInvalidJSON("/api/v1/vps/inspect/apply")
}
