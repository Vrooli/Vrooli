package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPackagerAPISurface_ValidateAndPlanEndpointsExist(t *testing.T) {
	// [REQ:STC-P0-007] packager API surface for deployment-manager
	t.Setenv("API_PORT", "0")
	srv, err := NewServer()
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	ts := httptest.NewServer(srv.router)
	defer ts.Close()

	payload := map[string]interface{}{
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
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	resp, err := http.Post(ts.URL+"/api/v1/manifest/validate", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("post validate: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("validate status: %d", resp.StatusCode)
	}

	resp, err = http.Post(ts.URL+"/api/v1/plan", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("post plan: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("plan status: %d", resp.StatusCode)
	}
}
