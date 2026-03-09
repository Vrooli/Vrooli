package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ──────────────────────────────────────────────────────────────────────────────
// prd generate --path
// ──────────────────────────────────────────────────────────────────────────────

func TestPRDGeneratePassesCustomPath(t *testing.T) {
	app := newTestApp(t)

	var capturedPayload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			fmt.Fprint(w, `{"status":"healthy","readiness":true}`)
			return
		}
		if r.URL.Path == "/api/v1/drafts/ai/generate" {
			body := new(bytes.Buffer)
			_, _ = body.ReadFrom(r.Body)
			_ = r.Body.Close()
			_ = json.Unmarshal(body.Bytes(), &capturedPayload)

			fmt.Fprint(w, `{"draft_id":"d1","entity_type":"scenario","entity_name":"demo","section":"🎯 Full PRD","generated_text":"# PRD","model":"test","saved_to_draft":true,"success":true}`)
			return
		}
		t.Fatalf("unexpected path: %s", r.URL.Path)
	}))
	t.Cleanup(server.Close)
	t.Setenv("PRD_CONTROL_TOWER_API_BASE", server.URL)

	if err := app.Run([]string{"prd", "generate", "demo", "--context", "ctx", "--path", "/custom/dir", "--json"}); err != nil {
		t.Fatalf("prd generate failed: %v", err)
	}

	if capturedPayload["custom_path"] != "/custom/dir" {
		t.Errorf("expected custom_path=/custom/dir, got %v", capturedPayload["custom_path"])
	}
}

func TestPRDGenerateWithPathAndPublishPassesCustomPathToBoth(t *testing.T) {
	app := newTestApp(t)

	var genPayload, pubPayload map[string]interface{}
	step := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			fmt.Fprint(w, `{"status":"healthy","readiness":true}`)
			return
		}
		switch step {
		case 0:
			body := new(bytes.Buffer)
			_, _ = body.ReadFrom(r.Body)
			_ = r.Body.Close()
			_ = json.Unmarshal(body.Bytes(), &genPayload)
			step++
			fmt.Fprint(w, `{"draft_id":"d1","entity_type":"scenario","entity_name":"demo","section":"🎯 Full PRD","generated_text":"# PRD","model":"test","saved_to_draft":true,"success":true}`)
		case 1:
			body := new(bytes.Buffer)
			_, _ = body.ReadFrom(r.Body)
			_ = r.Body.Close()
			_ = json.Unmarshal(body.Bytes(), &pubPayload)
			step++
			fmt.Fprint(w, `{"success":true,"message":"ok","published_to":"/custom/dir/PRD.md","published_at":"now"}`)
		default:
			t.Fatalf("unexpected extra request: %s", r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)
	t.Setenv("PRD_CONTROL_TOWER_API_BASE", server.URL)

	if err := app.Run([]string{"prd", "generate", "demo", "--context", "ctx", "--path", "/custom/dir", "--publish", "--json"}); err != nil {
		t.Fatalf("failed: %v", err)
	}
	if step != 2 {
		t.Fatalf("expected 2 requests, got %d", step)
	}
	if genPayload["custom_path"] != "/custom/dir" {
		t.Errorf("generate request missing custom_path, got %v", genPayload["custom_path"])
	}
	if pubPayload["custom_path"] != "/custom/dir" {
		t.Errorf("publish request missing custom_path, got %v", pubPayload["custom_path"])
	}
}

func TestPRDGenerateWithPathAndTemplatePassesCustomPathToBoth(t *testing.T) {
	app := newTestApp(t)

	var genPayload, pubPayload map[string]interface{}
	step := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			fmt.Fprint(w, `{"status":"healthy","readiness":true}`)
			return
		}
		switch step {
		case 0:
			body := new(bytes.Buffer)
			_, _ = body.ReadFrom(r.Body)
			_ = r.Body.Close()
			_ = json.Unmarshal(body.Bytes(), &genPayload)
			step++
			fmt.Fprint(w, `{"draft_id":"d1","entity_type":"scenario","entity_name":"demo","section":"🎯 Full PRD","generated_text":"# PRD","model":"test","saved_to_draft":true,"success":true}`)
		case 1:
			body := new(bytes.Buffer)
			_, _ = body.ReadFrom(r.Body)
			_ = r.Body.Close()
			_ = json.Unmarshal(body.Bytes(), &pubPayload)
			step++
			fmt.Fprint(w, `{"success":true,"message":"ok","published_to":"/custom/dir/PRD.md","published_at":"now","created_scenario":true,"scenario_id":"demo","scenario_type":"scenario","scenario_path":"/tmp/demo"}`)
		default:
			t.Fatalf("unexpected extra request: %s", r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)
	t.Setenv("PRD_CONTROL_TOWER_API_BASE", server.URL)

	if err := app.Run([]string{"prd", "generate", "demo", "--context", "ctx", "--path", "/custom/dir", "--template", "go-cli", "--json"}); err != nil {
		t.Fatalf("failed: %v", err)
	}
	if genPayload["custom_path"] != "/custom/dir" {
		t.Errorf("generate request missing custom_path")
	}
	if pubPayload["custom_path"] != "/custom/dir" {
		t.Errorf("publish request missing custom_path")
	}
}

func TestPRDGenerateWithoutPathOmitsCustomPath(t *testing.T) {
	app := newTestApp(t)

	var capturedPayload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			fmt.Fprint(w, `{"status":"healthy","readiness":true}`)
			return
		}
		body := new(bytes.Buffer)
		_, _ = body.ReadFrom(r.Body)
		_ = r.Body.Close()
		_ = json.Unmarshal(body.Bytes(), &capturedPayload)
		fmt.Fprint(w, `{"draft_id":"d1","entity_type":"scenario","entity_name":"demo","section":"🎯 Full PRD","generated_text":"# PRD","model":"test","saved_to_draft":true,"success":true}`)
	}))
	t.Cleanup(server.Close)
	t.Setenv("PRD_CONTROL_TOWER_API_BASE", server.URL)

	if err := app.Run([]string{"prd", "generate", "demo", "--context", "ctx", "--json"}); err != nil {
		t.Fatalf("failed: %v", err)
	}

	// custom_path should be empty or absent
	cp, _ := capturedPayload["custom_path"].(string)
	if cp != "" {
		t.Errorf("expected empty custom_path when --path not given, got %q", cp)
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// prd validate --path
// ──────────────────────────────────────────────────────────────────────────────

func TestPRDValidatePassesCustomPath(t *testing.T) {
	app := newTestApp(t)

	var capturedQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			fmt.Fprint(w, `{"status":"healthy","readiness":true}`)
			return
		}
		capturedQuery = r.URL.Query().Get("custom_path")
		fmt.Fprint(w, `{"entity_type":"scenario","entity_name":"demo","status":"healthy","violations":[],"generated_at":"2024-01-01T00:00:00Z"}`)
	}))
	t.Cleanup(server.Close)
	t.Setenv("PRD_CONTROL_TOWER_API_BASE", server.URL)

	if err := app.Run([]string{"prd", "validate", "demo", "--path", "/my/custom/dir", "--json"}); err != nil {
		t.Fatalf("failed: %v", err)
	}

	if capturedQuery != "/my/custom/dir" {
		t.Errorf("expected custom_path query param=/my/custom/dir, got %q", capturedQuery)
	}
}

func TestPRDValidateWithoutPathOmitsCustomPath(t *testing.T) {
	app := newTestApp(t)

	var capturedQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			fmt.Fprint(w, `{"status":"healthy","readiness":true}`)
			return
		}
		capturedQuery = r.URL.Query().Get("custom_path")
		fmt.Fprint(w, `{"entity_type":"scenario","entity_name":"demo","status":"healthy","violations":[],"generated_at":"2024-01-01T00:00:00Z"}`)
	}))
	t.Cleanup(server.Close)
	t.Setenv("PRD_CONTROL_TOWER_API_BASE", server.URL)

	if err := app.Run([]string{"prd", "validate", "demo", "--json"}); err != nil {
		t.Fatalf("failed: %v", err)
	}

	if capturedQuery != "" {
		t.Errorf("expected empty custom_path when --path not given, got %q", capturedQuery)
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// prd fix --path
// ──────────────────────────────────────────────────────────────────────────────

func TestPRDFixPassesCustomPath(t *testing.T) {
	app := newTestApp(t)

	var validateQuery string
	var genPayload, pubPayload map[string]interface{}
	step := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			fmt.Fprint(w, `{"status":"healthy","readiness":true}`)
			return
		}
		switch step {
		case 0:
			// validate call
			validateQuery = r.URL.Query().Get("custom_path")
			step++
			fmt.Fprint(w, `{"entity_type":"scenario","entity_name":"demo","status":"warning","violations":[{"rule_id":"r1","severity":"high","title":"Missing section"}],"generated_at":"2024-01-01T00:00:00Z"}`)
		case 1:
			// AI generate call
			body := new(bytes.Buffer)
			_, _ = body.ReadFrom(r.Body)
			_ = r.Body.Close()
			_ = json.Unmarshal(body.Bytes(), &genPayload)
			step++
			fmt.Fprint(w, `{"draft_id":"d1","entity_type":"scenario","entity_name":"demo","section":"🎯 Full PRD","generated_text":"# Fixed PRD","model":"test","saved_to_draft":true,"success":true}`)
		case 2:
			// publish call
			body := new(bytes.Buffer)
			_, _ = body.ReadFrom(r.Body)
			_ = r.Body.Close()
			_ = json.Unmarshal(body.Bytes(), &pubPayload)
			step++
			fmt.Fprint(w, `{"success":true,"message":"ok","published_to":"/custom/dir/PRD.md","published_at":"now"}`)
		default:
			t.Fatalf("unexpected extra request: %s", r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)
	t.Setenv("PRD_CONTROL_TOWER_API_BASE", server.URL)

	if err := app.Run([]string{"prd", "fix", "demo", "--path", "/custom/dir", "--auto", "--json"}); err != nil {
		t.Fatalf("failed: %v", err)
	}

	if validateQuery != "/custom/dir" {
		t.Errorf("validate custom_path = %q, want /custom/dir", validateQuery)
	}
	if genPayload["custom_path"] != "/custom/dir" {
		t.Errorf("generate custom_path = %v, want /custom/dir", genPayload["custom_path"])
	}
	if pubPayload["custom_path"] != "/custom/dir" {
		t.Errorf("publish custom_path = %v, want /custom/dir", pubPayload["custom_path"])
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// requirements generate --path
// ──────────────────────────────────────────────────────────────────────────────

func TestRequirementsGeneratePassesCustomPath(t *testing.T) {
	app := newTestApp(t)

	var capturedPayload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			fmt.Fprint(w, `{"status":"healthy","readiness":true}`)
			return
		}
		if r.URL.Path != "/api/v1/requirements/generate" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		body := new(bytes.Buffer)
		_, _ = body.ReadFrom(r.Body)
		_ = r.Body.Close()
		_ = json.Unmarshal(body.Bytes(), &capturedPayload)
		fmt.Fprint(w, `{"entity_type":"scenario","entity_name":"demo","success":true,"message":"Generated 3 requirements","requirement_count":3,"p0_count":1,"p1_count":2,"p2_count":0,"files_created":[],"model":"test","generated_at":"2024-01-01T00:00:00Z"}`)
	}))
	t.Cleanup(server.Close)
	t.Setenv("PRD_CONTROL_TOWER_API_BASE", server.URL)

	if err := app.Run([]string{"requirements", "generate", "demo", "--path", "/my/reqs/path", "--json"}); err != nil {
		t.Fatalf("failed: %v", err)
	}

	if capturedPayload["custom_path"] != "/my/reqs/path" {
		t.Errorf("expected custom_path=/my/reqs/path, got %v", capturedPayload["custom_path"])
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// requirements validate --path
// ──────────────────────────────────────────────────────────────────────────────

func TestRequirementsValidatePassesCustomPath(t *testing.T) {
	app := newTestApp(t)

	var capturedQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			fmt.Fprint(w, `{"status":"healthy","readiness":true}`)
			return
		}
		capturedQuery = r.URL.Query().Get("custom_path")
		fmt.Fprint(w, `{"entity_type":"scenario","entity_name":"demo","status":"healthy","requirement_count":5,"target_count":5,"linked_count":5,"violations":[],"generated_at":"2024-01-01T00:00:00Z"}`)
	}))
	t.Cleanup(server.Close)
	t.Setenv("PRD_CONTROL_TOWER_API_BASE", server.URL)

	if err := app.Run([]string{"requirements", "validate", "demo", "--path", "/validate/path", "--json"}); err != nil {
		t.Fatalf("failed: %v", err)
	}

	if capturedQuery != "/validate/path" {
		t.Errorf("expected custom_path=/validate/path, got %q", capturedQuery)
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// requirements fix --path
// ──────────────────────────────────────────────────────────────────────────────

func TestRequirementsFixPassesCustomPath(t *testing.T) {
	app := newTestApp(t)

	var capturedPayload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			fmt.Fprint(w, `{"status":"healthy","readiness":true}`)
			return
		}
		if r.URL.Path != "/api/v1/requirements/fix" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		body := new(bytes.Buffer)
		_, _ = body.ReadFrom(r.Body)
		_ = r.Body.Close()
		_ = json.Unmarshal(body.Bytes(), &capturedPayload)
		fmt.Fprint(w, `{"entity_type":"scenario","entity_name":"demo","success":true,"message":"Fixed 1 target","targets_fixed":1,"requirements_added":2,"total_requirements":7,"remaining_violations":0,"files_modified":[],"model":"test","fixed_at":"2024-01-01T00:00:00Z"}`)
	}))
	t.Cleanup(server.Close)
	t.Setenv("PRD_CONTROL_TOWER_API_BASE", server.URL)

	if err := app.Run([]string{"requirements", "fix", "demo", "--path", "/fix/path", "--json"}); err != nil {
		t.Fatalf("failed: %v", err)
	}

	if capturedPayload["custom_path"] != "/fix/path" {
		t.Errorf("expected custom_path=/fix/path, got %v", capturedPayload["custom_path"])
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Verify --path is accepted but not required (backward compatibility)
// ──────────────────────────────────────────────────────────────────────────────

func TestAllCommandsWorkWithoutPath(t *testing.T) {
	// Verify that all 6 subcommands work without --path (backward compat)
	commands := []struct {
		name string
		args []string
		resp string
	}{
		{
			name: "prd generate",
			args: []string{"prd", "generate", "demo", "--context", "ctx", "--json"},
			resp: `{"draft_id":"d1","entity_type":"scenario","entity_name":"demo","section":"🎯 Full PRD","generated_text":"# PRD","model":"test","saved_to_draft":true,"success":true}`,
		},
		{
			name: "requirements generate",
			args: []string{"requirements", "generate", "demo", "--json"},
			resp: `{"entity_type":"scenario","entity_name":"demo","success":true,"requirement_count":0,"p0_count":0,"p1_count":0,"p2_count":0,"files_created":[],"model":"test","generated_at":"now"}`,
		},
		{
			name: "requirements fix",
			args: []string{"requirements", "fix", "demo", "--json"},
			resp: `{"entity_type":"scenario","entity_name":"demo","success":true,"message":"ok","targets_fixed":0,"requirements_added":0,"total_requirements":0,"remaining_violations":0,"files_modified":[],"fixed_at":"now"}`,
		},
	}

	for _, tt := range commands {
		t.Run(tt.name, func(t *testing.T) {
			app := newTestApp(t)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/health" {
					fmt.Fprint(w, `{"status":"healthy","readiness":true}`)
					return
				}
				fmt.Fprint(w, tt.resp)
			}))
			t.Cleanup(server.Close)
			t.Setenv("PRD_CONTROL_TOWER_API_BASE", server.URL)

			if err := app.Run(tt.args); err != nil {
				t.Fatalf("%s without --path failed: %v", tt.name, err)
			}
		})
	}
}
