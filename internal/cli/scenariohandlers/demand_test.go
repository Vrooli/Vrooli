package scenariohandlers

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

func TestDemandOutputUsesPublicSnakeCaseWireNames(t *testing.T) {
	out := newDemandOutput(scenarioruntime.DemandLease{
		LeaseID: "lease-1", Scenario: "search-hub", ConsumerID: "session-1", Kind: "program",
		CreatedAt: time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC),
		ExpiresAt: time.Date(2026, 9, 6, 12, 10, 0, 0, time.UTC), Status: "active",
	})
	encoded, err := json.Marshal(out)
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]map[string]any
	if err := json.Unmarshal(encoded, &wire); err != nil {
		t.Fatal(err)
	}
	lease := wire["lease"]
	if lease["lease_id"] != "lease-1" || lease["consumer_id"] != "session-1" {
		t.Fatalf("wire lease = %#v", lease)
	}
	if _, legacy := lease["LeaseID"]; legacy {
		t.Fatalf("legacy Go field name leaked into wire response: %#v", lease)
	}
}

func TestDemandHistoryDeclaresBoundedEvidenceAndUsesSnakeCase(t *testing.T) {
	var output bytes.Buffer
	if err := renderDemandHistory(&output, true, []scenarioruntime.DemandTransition{{SchemaVersion: 1, EventID: "event", Operation: "renewed", Scenario: "demo", Variant: "live", LeaseID: "lease"}}); err != nil {
		t.Fatal(err)
	}
	var wire struct {
		Scope     string           `json:"scope"`
		Retention int              `json:"retention_per_scenario"`
		Events    []map[string]any `json:"events"`
	}
	if err := json.Unmarshal(output.Bytes(), &wire); err != nil {
		t.Fatal(err)
	}
	if wire.Scope != "recent_committed_transitions" || wire.Retention != scenarioruntime.DemandAuditRetention || len(wire.Events) != 1 || wire.Events[0]["lease_id"] != "lease" {
		t.Fatalf("wire=%+v", wire)
	}
	output.Reset()
	if err := renderDemandHistory(&output, true, nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), `"events":[]`) {
		t.Fatalf("empty history=%s", output.String())
	}
}
