package rehearsals

import (
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"testing"
)

func TestPublic21013NativeProfileBindsWebhookLeadCaptureToSyntheticGmailAndSheets(t *testing.T) {
	raw := readFixture(t, "public-workflows/21013.json")
	workflow, _, _, err := normalizeWorkflow(raw)
	if err != nil {
		t.Fatalf("normalize public workflow 21013: %v", err)
	}
	simulator, err := NewDescriptionSimulator(readFixture(t, "public-workflow-21013-native-profile.json"), false)
	if err != nil {
		t.Fatalf("load reusable Gmail/Sheets API profile: %v", err)
	}
	profile, err := simulator.ExportVirtualAPIProfile()
	if err != nil {
		t.Fatalf("export task-scoped native API profile: %v", err)
	}
	if len(profile.Hosts) != 2 || !slices.Contains(profile.Hosts, "gmail.googleapis.com") || !slices.Contains(profile.Hosts, "sheets.googleapis.com") {
		t.Fatalf("native API profile hosts=%v; want exact Gmail and Sheets hosts", profile.Hosts)
	}
	if !workflowUsesExactVirtualAPIRoutes(workflow, profile) {
		t.Fatal("public workflow's Gmail and Sheets native operations are not covered by exact bound routes")
	}
	want := map[string]bool{
		"n8n-nodes-base.googlesheets|sheet|append": false,
		"n8n-nodes-base.gmail|message|send":        false,
	}
	for _, route := range profile.Routes {
		if route.Status != http.StatusOK || route.ContentType != "application/json" || len(route.Bindings) != 1 {
			t.Fatalf("route is not bounded to one JSON native operation: %#v", route)
		}
		for _, binding := range route.Bindings {
			key := strings.ToLower(binding.NodeType) + "|" + strings.ToLower(binding.Resource) + "|" + strings.ToLower(binding.Operation)
			if _, exists := want[key]; !exists {
				t.Fatalf("unexpected native binding %q", key)
			}
			want[key] = true
			if key == "n8n-nodes-base.googlesheets|sheet|append" {
				if binding.InputMatch["documentId"] != "synthetic-sheet-id" || binding.InputMatch["sheetName"] != "Leads" {
					t.Fatalf("Sheets route is not bound to synthetic document and sheet selectors: %#v", binding.InputMatch)
				}
			}
		}
		if route.Host == "sheets.googleapis.com" && route.Path != "/v4/spreadsheets/synthetic-sheet-id/values/Leads:append" {
			t.Fatalf("Sheets append route lost its exact API path: %#v", route)
		}
		if route.Host == "gmail.googleapis.com" && route.Path != "/gmail/v1/users/me/messages/send" {
			t.Fatalf("Gmail send route lost its exact API path: %#v", route)
		}
	}
	for key, found := range want {
		if !found {
			t.Errorf("public workflow native API profile does not bind %s", key)
		}
	}
}

func TestPublic21013NativeImportHasPinnedNodeVersionsAndEditorPositions(t *testing.T) {
	raw := readFixture(t, "public-workflows/21013.json")
	var source struct {
		Nodes []struct {
			Type        string         `json:"type"`
			TypeVersion float64        `json:"typeVersion"`
			Position    []float64      `json:"position"`
			Parameters  map[string]any `json:"parameters"`
		} `json:"nodes"`
	}
	if err := json.Unmarshal([]byte(raw), &source); err != nil {
		t.Fatal(err)
	}
	want := map[string]float64{
		"n8n-nodes-base.webhook": 2.1, "n8n-nodes-base.code": 2,
		"n8n-nodes-base.if": 2.2, "n8n-nodes-base.respondToWebhook": 1.5,
		"n8n-nodes-base.googleSheets": 4.7, "n8n-nodes-base.gmail": 2.1,
	}
	for i, node := range source.Nodes {
		version, supported := want[node.Type]
		if !supported || node.TypeVersion != version {
			t.Fatalf("node %d has unsupported pinned import version: type=%q version=%v", i, node.Type, node.TypeVersion)
		}
		if len(node.Position) != 2 {
			t.Fatalf("node %q lacks the required two-dimensional n8n editor position", node.Type)
		}
		if node.Type == "n8n-nodes-base.googleSheets" && node.Parameters["resource"] != "sheet" {
			t.Fatalf("Google Sheets node must declare the native Sheet resource, got %#v", node.Parameters["resource"])
		}
		if node.Type == "n8n-nodes-base.gmail" {
			for field, want := range map[string]any{"sendTo": "lead@example.invalid", "subject": "Synthetic form received", "message": "Thanks, we received your message.", "emailType": "text"} {
				if node.Parameters[field] != want {
					t.Errorf("Gmail node %s = %#v, want bounded synthetic value %#v", field, node.Parameters[field], want)
				}
			}
		}
	}
}
