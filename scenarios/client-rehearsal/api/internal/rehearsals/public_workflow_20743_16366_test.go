package rehearsals

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestPublic20743And16366SanitizedFixtureProvenance(t *testing.T) {
	candidates := []struct {
		id        string
		hash      string
		nodes     int
		connNodes int
	}{
		{id: "20743", hash: "1dc9fc72487b2e30cb82ccc5ab6e448fdeb14d801d466bfd9fe3633c6662a5b1", nodes: 22, connNodes: 19},
		{id: "16366", hash: "821630e52f5359aed8a77d6705e7a6b653f9a1cf10de88c11855c8fdca82114e", nodes: 18, connNodes: 15},
	}
	for _, candidate := range candidates {
		t.Run(candidate.id, func(t *testing.T) {
			fixture := readFixture(t, "public-workflows/"+candidate.id+".json")
			if got := fmt.Sprintf("%x", sha256.Sum256([]byte(fixture))); got != candidate.hash {
				t.Fatalf("sanitized fixture changed without provenance update: %s", got)
			}
			for _, forbidden := range []string{"gmail.com", "google.com", "slack.com", "1lqLTqsEEBeKH0huGm6KAlAaUsFv4YMPNFeNU57mwCU0", "1vyOm_vm8l0ba0Mb2ug35jcNUE1sjzNiSSjm9JelcjuA", "YOUR_COMPANY_ID", "YOUR_GOOGLE_SHEET_ID", "MDQ5NjkxNjA1OTI5NDc0NjYzNjQ6MDow"} {
				if strings.Contains(fixture, forbidden) {
					t.Errorf("sanitized fixture retains a live/source destination %q", forbidden)
				}
			}
			if !strings.Contains(fixture, `"active":false`) || strings.Contains(fixture, `"credentials"`) {
				t.Fatal("fixture must be inactive and contain no credential references")
			}
			var graph struct {
				Nodes       []map[string]any `json:"nodes"`
				Connections map[string]any   `json:"connections"`
			}
			if err := json.Unmarshal([]byte(fixture), &graph); err != nil {
				t.Fatalf("decode sanitized graph: %v", err)
			}
			if len(graph.Nodes) != candidate.nodes || len(graph.Connections) != candidate.connNodes {
				t.Fatalf("sanitization changed executable graph shape: nodes=%d connection sources=%d", len(graph.Nodes), len(graph.Connections))
			}
		})
	}
}

func TestPublic20743And16366FullGraphImportAttempt(t *testing.T) {
	for _, id := range []string{"20743", "16366"} {
		t.Run(id, func(t *testing.T) {
			workflow, _, _, err := normalizeWorkflow(readFixture(t, "public-workflows/"+id+".json"))
			if err != nil {
				t.Logf("no simulator credit: full sanitized graph import fails closed: %v", err)
				return
			}
			t.Logf("full graph imported (%d nodes); still no credit until the whole workflow runs in-process against synthetic profiles", len(workflow.Nodes))
		})
	}
}

func TestPublic16366SyntheticServiceProfileLoads(t *testing.T) {
	if _, err := NewDescriptionSimulator(readFixture(t, "public-workflow-16366-profile.json"), false); err != nil {
		t.Fatalf("load synthetic Sheets, Google Tasks, Slack, Gmail, and AI profile: %v", err)
	}
}
