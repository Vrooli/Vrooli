package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListDraftsHitsExpectedEndpoint(t *testing.T) {
	app := newTestApp(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			fmt.Fprint(w, `{"status":"healthy","readiness":true}`)
			return
		}
		if r.URL.Path != "/api/v1/drafts" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		fmt.Fprint(w, `{"drafts":[],"total":0}`)
	}))
	t.Cleanup(server.Close)
	t.Setenv("PRD_CONTROL_TOWER_API_BASE", server.URL)

	if err := app.Run([]string{"list-drafts", "--json"}); err != nil {
		t.Fatalf("list-drafts failed: %v", err)
	}
}

func TestGeneratePRDCallsAIGenerateDraftEndpoint(t *testing.T) {
	app := newTestApp(t)

	seen := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			fmt.Fprint(w, `{"status":"healthy","readiness":true}`)
			return
		}
		if r.URL.Path != "/api/v1/drafts/ai/generate" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		var payload map[string]interface{}
		body := new(bytes.Buffer)
		_, _ = body.ReadFrom(r.Body)
		_ = r.Body.Close()
		if err := json.Unmarshal(body.Bytes(), &payload); err != nil {
			t.Fatalf("parse request: %v", err)
		}
		if payload["entity_type"] != "scenario" {
			t.Fatalf("expected entity_type=scenario, got %v", payload["entity_type"])
		}
		if payload["entity_name"] != "demo-scenario" {
			t.Fatalf("expected entity_name=demo-scenario, got %v", payload["entity_name"])
		}
		if payload["section"] != "🎯 Full PRD" {
			t.Fatalf("expected section full prd, got %v", payload["section"])
		}
		if !strings.Contains(fmt.Sprintf("%v", payload["context"]), "some context") {
			t.Fatalf("expected context to contain 'some context', got %v", payload["context"])
		}
		seen = true

		fmt.Fprint(w, `{"draft_id":"d1","entity_type":"scenario","entity_name":"demo-scenario","section":"🎯 Full PRD","generated_text":"# PRD","model":"test","saved_to_draft":true,"success":true}`)
	}))
	t.Cleanup(server.Close)
	t.Setenv("PRD_CONTROL_TOWER_API_BASE", server.URL)

	if err := app.Run([]string{"generate-prd", "demo-scenario", "--context", "some context", "--json"}); err != nil {
		t.Fatalf("generate-prd failed: %v", err)
	}
	if !seen {
		t.Fatalf("expected server to receive generate request")
	}
}

func TestGeneratePRDWithTemplatePublishes(t *testing.T) {
	app := newTestApp(t)

	step := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			fmt.Fprint(w, `{"status":"healthy","readiness":true}`)
			return
		}
		switch step {
		case 0:
			if r.URL.Path != "/api/v1/drafts/ai/generate" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			step++
			fmt.Fprint(w, `{"draft_id":"d1","entity_type":"scenario","entity_name":"demo","section":"🎯 Full PRD","generated_text":"# PRD","model":"test","saved_to_draft":true,"success":true}`)
			return
		case 1:
			if r.URL.Path != "/api/v1/drafts/d1/publish" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			var payload map[string]interface{}
			body := new(bytes.Buffer)
			_, _ = body.ReadFrom(r.Body)
			_ = r.Body.Close()
			if err := json.Unmarshal(body.Bytes(), &payload); err != nil {
				t.Fatalf("parse request: %v", err)
			}
			template := payload["template"].(map[string]interface{})
			if template["name"] != "my-template" {
				t.Fatalf("expected template name my-template, got %v", template["name"])
			}
			vars := template["variables"].(map[string]interface{})
			if vars["SCENARIO_ID"] != "demo" {
				t.Fatalf("expected SCENARIO_ID demo, got %v", vars["SCENARIO_ID"])
			}
			step++
			fmt.Fprint(w, `{"success":true,"message":"ok","published_to":"/tmp/PRD.md","published_at":"now","draft_removed":true,"created_scenario":true,"scenario_id":"demo","scenario_type":"scenario","scenario_path":"/tmp/demo"}`)
			return
		default:
			t.Fatalf("unexpected extra request: %s", r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)
	t.Setenv("PRD_CONTROL_TOWER_API_BASE", server.URL)

	if err := app.Run([]string{"generate-prd", "demo", "--context", "ctx", "--template", "my-template", "--json"}); err != nil {
		t.Fatalf("generate-prd with template failed: %v", err)
	}
	if step != 2 {
		t.Fatalf("expected 2 requests, got %d", step)
	}
}

func TestGeneratePRDWithPublishFlagPublishes(t *testing.T) {
	app := newTestApp(t)

	step := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			fmt.Fprint(w, `{"status":"healthy","readiness":true}`)
			return
		}
		switch step {
		case 0:
			if r.URL.Path != "/api/v1/drafts/ai/generate" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			step++
			fmt.Fprint(w, `{"draft_id":"d1","entity_type":"scenario","entity_name":"demo","section":"🎯 Full PRD","generated_text":"# PRD","model":"test","saved_to_draft":true,"success":true}`)
			return
		case 1:
			if r.URL.Path != "/api/v1/drafts/d1/publish" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			step++
			fmt.Fprint(w, `{"success":true,"message":"ok","published_to":"/tmp/PRD.md","published_at":"now","draft_removed":true,"created_scenario":false,"scenario_id":"demo","scenario_type":"scenario","scenario_path":"/tmp/demo"}`)
			return
		default:
			t.Fatalf("unexpected extra request: %s", r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)
	t.Setenv("PRD_CONTROL_TOWER_API_BASE", server.URL)

	if err := app.Run([]string{"generate-prd", "demo", "--context", "ctx", "--publish", "--json"}); err != nil {
		t.Fatalf("generate-prd with publish failed: %v", err)
	}
	if step != 2 {
		t.Fatalf("expected 2 requests, got %d", step)
	}
}
