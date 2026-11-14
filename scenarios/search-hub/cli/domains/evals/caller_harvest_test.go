package evals

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
)

func TestHarvestSelectedCallersUsesAgentManagerInventoryAndWritesPrivateCandidate(t *testing.T) {
	const runID = "123e4567-e89b-12d3-a456-426614174000"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/runs":
			if r.URL.Query().Get("limit") != "5000" || r.URL.Query().Get("offset") != "0" {
				t.Errorf("inventory query = %v", r.URL.Query())
			}
			_, _ = io.WriteString(w, `{"runs":[{"id":"`+runID+`"}],"has_more":false}`)
		case "/api/v1/runs/" + runID + "/events":
			if r.URL.Query().Get("after_sequence") != "-1" || r.URL.Query().Get("event_types") != "message,message_deleted" {
				t.Errorf("event query = %v", r.URL.Query())
			}
			// Owner protojson omits the zero-valued proto3 sequence field.
			_, _ = io.WriteString(w, `{"events":[{"id":"event-1","run_id":"`+runID+`","event_type":"RUN_EVENT_TYPE_MESSAGE","message":{"role":"user","source":"operator","content":"private caller query","conversation_id":"conversation-1"}}],"has_more":false}`)
		default:
			t.Errorf("unexpected request path %q", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	bundle, err := harvestSelectedCallers(context.Background(), server.Client(), server.URL, []string{runID})
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.Records) != 1 || bundle.Records[0].Sequence != 0 || bundle.Records[0].Status != "candidate" || bundle.Records[0].AcceptanceEligible {
		t.Fatalf("candidate bundle = %+v", bundle)
	}

	path := filepath.Join(t.TempDir(), "candidates.json")
	if err := writeCallerHarvestFile(path, bundle); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("candidate file mode = %o; want 600", info.Mode().Perm())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "private caller query") || !strings.Contains(string(data), `"acceptance_eligible": false`) {
		t.Fatalf("candidate export omitted private query or acceptance gate: %s", data)
	}
	if err := writeCallerHarvestFile(path, bundle); err == nil {
		t.Fatal("existing candidate export should not be overwritten")
	}
}

func TestParseCallerRunIDsRequiresUniqueSelectedInventoryIDs(t *testing.T) {
	ids, err := parseCallerRunIDs("run-2, run-1")
	if err != nil || len(ids) != 2 || ids[0] != "run-1" {
		t.Fatalf("parseCallerRunIDs = %v, %v", ids, err)
	}
	for _, raw := range []string{"", "run-1,run-1"} {
		if _, err := parseCallerRunIDs(raw); err == nil {
			t.Errorf("parseCallerRunIDs(%q) unexpectedly succeeded", raw)
		}
	}
}

func TestAgentManagerBaseRestrictsCallerTextReadsToLoopback(t *testing.T) {
	if _, err := configuredAgentManagerBase("https://agent-manager.example.test"); err == nil {
		t.Fatal("remote Agent Manager API should be rejected")
	}
	base, err := configuredAgentManagerBase("http://localhost:8090")
	if err != nil || base != "http://localhost:8090" {
		t.Fatalf("configured local Agent Manager base = %q, %v", base, err)
	}
}

func TestEvalsRegisterWiresHarvestCallersAsPrivateLocalAction(t *testing.T) {
	manifest, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	group, err := Register(nil, manifest)
	if err != nil {
		t.Fatal(err)
	}
	for _, command := range group.Subcommands {
		if command.Name == "harvest-callers" {
			if command.NeedsAPI || command.PrimitiveEvidence() != cliapp.PrimitiveAction {
				t.Fatalf("harvest-callers command = needsAPI:%t primitive:%q", command.NeedsAPI, command.PrimitiveEvidence())
			}
			return
		}
	}
	t.Fatal("harvest-callers was not registered")
}
