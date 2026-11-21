package incidents

import (
	"context"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
)

// One watchdog report that carries a cpu-pressure and a fork-rate finding
// opens two incidents, each fingerprinted on its finding, and the fork-rate
// incident names the attributed parent.
func TestIncidentFingerprintSeparatesPressureFindings(t *testing.T) {
	store := &memoryStore{}
	service := NewService(store)
	result := checks.Result{
		CheckID:   "system-emergency-watchdog-report",
		Status:    checks.StatusCritical,
		Message:   "emergency watchdog reported 2 findings",
		Timestamp: time.Date(2026, 9, 2, 10, 14, 0, 0, time.UTC),
		Details: map[string]any{
			"findings": []map[string]any{
				{"name": "cpu-pressure", "reason": "CPU pressure 96.0% meets or exceeds SB14 bar"},
				{"name": "fork-rate", "reason": "2481.0 forks/s exceeds SB16 bar", "attribution": map[string]any{
					"top_parent": map[string]any{"pid": 4242, "name": "claude", "children": 300, "delta": 280, "scope": "/user.slice/user-1000.slice/user@1000.service/vrooli-agents.slice/vrooli-agent-abc.scope"},
				}},
			},
		},
	}
	created, err := service.UpsertAllFromCheckResult(context.Background(), result)
	if err != nil {
		t.Fatalf("UpsertAllFromCheckResult() error = %v", err)
	}
	if created != 2 || len(store.inputs) != 2 {
		t.Fatalf("created = %d, inputs = %d, want two incidents", created, len(store.inputs))
	}
	if store.inputs[0].Fingerprint == store.inputs[1].Fingerprint {
		t.Fatalf("both findings share fingerprint %s", store.inputs[0].Fingerprint)
	}
	var forkTitle string
	for _, input := range store.inputs {
		if input.Type != TypeHostPressure {
			t.Fatalf("incident type = %s, want host_pressure", input.Type)
		}
		if stringDetail(input.Evidence, "findingKey") == "fork-rate" {
			forkTitle = input.Title
		}
	}
	if forkTitle != "fork storm from claude pid 4242 in scope vrooli-agent-abc.scope" {
		t.Fatalf("fork-rate title = %q", forkTitle)
	}
	single := checks.Result{CheckID: "system-emergency-watchdog-report", Status: checks.StatusCritical, Message: "one", Details: map[string]any{
		"findings": []map[string]any{{"name": "fork-rate", "reason": "r"}},
	}}
	rule, ok := classifyResult(withFinding(single, single.Details["findings"].([]map[string]any)[0]))
	if !ok || rule.title != "fork storm (unattributed)" {
		t.Fatalf("unattributed rule = %+v %v", rule, ok)
	}
}

// [REQ:STORM-002] One check owns several live incidents when it reports
// several findings; upserting one finding must not resolve the others as
// "superseded". (Before 2026-09-02 the 190 workload findings and the fork
// storm resolved each other on every tick, so no storm incident survived
// long enough to be thawed against.)
func TestPerFindingIncidentsAreNotSupersededByEachOther(t *testing.T) {
	store := &memoryStore{}
	service := NewService(store)
	result := checks.Result{
		CheckID: "system-emergency-watchdog-report", Status: checks.StatusCritical, Message: "two findings",
		Details: map[string]any{"findings": []map[string]any{
			{"name": "fork-rate", "reason": "2481.0 forks/s exceeds SB16 bar"},
			{"name": "unmanaged-workload:x.service", "reason": "matches no Vrooli declaration"},
		}},
	}
	if _, err := service.UpsertAllFromCheckResult(context.Background(), result); err != nil {
		t.Fatal(err)
	}
	store.list = []Incident{
		{ID: "inc-fork", Fingerprint: store.inputs[0].Fingerprint, Status: StatusOpen, Severity: SeverityCritical, SourceCheckIDs: []string{result.CheckID}},
		{ID: "inc-x", Fingerprint: store.inputs[1].Fingerprint, Status: StatusOpen, Severity: SeverityCritical, SourceCheckIDs: []string{result.CheckID}},
	}
	if _, err := service.UpsertAllFromCheckResult(context.Background(), result); err != nil {
		t.Fatal(err)
	}
	if len(store.updates) != 0 {
		t.Fatalf("per-finding upserts superseded siblings: %v %v", store.updateIDs, store.updates)
	}
	// A whole-check result whose fingerprint moved still supersedes its own
	// prior incident: the narrow rule keeps the old one-incident-per-check
	// behavior for every other check.
	store.list = []Incident{{ID: "inc-old", Fingerprint: "stale", Status: StatusOpen, Severity: SeverityCritical, SourceCheckIDs: []string{"host-runtime-integrity"}}}
	if _, _, err := service.UpsertFromCheckResult(context.Background(), checks.Result{CheckID: "host-runtime-integrity", Status: checks.StatusCritical, Message: "runtime failed"}); err != nil {
		t.Fatal(err)
	}
	if len(store.updateIDs) != 1 || store.updateIDs[0] != "inc-old" || store.updates[0] != StatusResolved {
		t.Fatalf("whole-check supersede lost: %v %v", store.updateIDs, store.updates)
	}
}
