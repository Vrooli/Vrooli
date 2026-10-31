package policybridge

import (
	"encoding/json"
	"testing"
	"time"
)

func TestBuildSnapshotDeclaresRouteAndArgvEvidence(t *testing.T) {
	data, err := BuildSnapshot(time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got["provider_id"] != "scenario-dependency-analyzer" || got["evidence_state"] != "clean" {
		t.Fatalf("snapshot = %+v", got)
	}
}
