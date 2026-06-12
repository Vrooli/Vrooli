package projectcli

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/control"
	"github.com/vrooli/vrooli/internal/maintenance"
	"github.com/vrooli/vrooli/internal/network"
	"github.com/vrooli/vrooli/internal/orchestrator"
	"github.com/vrooli/vrooli/internal/project"
	"github.com/vrooli/vrooli/internal/resources"
	"github.com/vrooli/vrooli/internal/templatevalidation"
)

func decodeJSON(t *testing.T, b []byte) map[string]any {
	t.Helper()
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, string(b))
	}
	return got
}

// TestStatusJSONContract pins the `vrooli status --json` wire shape.
func TestStatusJSONContract(t *testing.T) {
	healthy := true
	resp := StatusResponse{
		Report: project.StatusReport{
			Resources: []resources.Status{
				{
					Resource:  resources.Resource{Name: "postgres", Enabled: true},
					Installed: true,
					Running:   true,
					Healthy:   &healthy,
					Health:    "healthy",
					Raw:       json.RawMessage(`{"k":1}`),
				},
			},
			Scenarios: []orchestrator.ScenarioView{
				{
					Name:      "web-console",
					Status:    "running",
					Processes: 2,
					Runtime:   "native",
					Ports:     map[string]int{"api": 5329},
					Health:    "ok",
				},
			},
			Maintenance: &maintenance.ProcessSnapshot{
				TrackedProcesses: 3,
				OrphanProcesses:  1,
				Orphans:          []maintenance.SystemProcess{{PID: 42, PPID: 1, Command: "node"}},
			},
			Summary: map[string]int{"resources_total": 1},
		},
	}

	var buf bytes.Buffer
	if err := RenderStatusReport(&buf, cliout.FormatJSON, resp); err != nil {
		t.Fatalf("RenderStatusReport: %v", err)
	}
	got := decodeJSON(t, buf.Bytes())
	if got["success"] != true {
		t.Errorf("success: want true, got %v", got["success"])
	}
	status := got["status"].(map[string]any)
	rs := status["resources"].([]any)
	if len(rs) != 1 {
		t.Fatalf("resources: want 1, got %v", status["resources"])
	}
	r0 := rs[0].(map[string]any)
	if r0["status_code"] == nil { // snake_case key present (EmitUnpopulated)
		t.Errorf("status_code key missing (snake_case?): %v", r0)
	}
	if _, ok := r0["resource"].(map[string]any); !ok {
		t.Errorf("nested resource missing: %v", r0)
	}
	if r0["raw"] == nil {
		t.Errorf("raw should decode the json.RawMessage payload, got nil")
	}
	scs := status["scenarios"].([]any)
	s0 := scs[0].(map[string]any)
	if _, ok := s0["processes"].(float64); !ok {
		t.Errorf("processes must be a JSON number, got %T (%v)", s0["processes"], s0["processes"])
	}
	if _, ok := s0["health_status"]; !ok {
		t.Errorf("health_status key missing: %v", s0)
	}
	mnt := status["maintenance"].(map[string]any)
	if _, ok := mnt["tracked_processes"].(float64); !ok {
		t.Errorf("tracked_processes must be a JSON number, got %T", mnt["tracked_processes"])
	}
	summary := status["summary"].(map[string]any)
	if v, ok := summary["resources_total"].(float64); !ok || v != 1 {
		t.Errorf("summary.resources_total must be number 1, got %v", summary["resources_total"])
	}
}

// TestDoctorJSONContract pins the `vrooli doctor --json` wire shape.
func TestDoctorJSONContract(t *testing.T) {
	var buf bytes.Buffer
	report := project.DoctorReport{Checks: []project.DoctorCheck{{Name: "go", Status: "ok", Message: "found"}}}
	if err := RenderDoctorReport(&buf, cliout.FormatJSON, report); err != nil {
		t.Fatalf("RenderDoctorReport: %v", err)
	}
	got := decodeJSON(t, buf.Bytes())
	if got["success"] != true {
		t.Errorf("success: want true, got %v", got["success"])
	}
	checks := got["checks"].([]any)
	if len(checks) != 1 || checks[0].(map[string]any)["name"] != "go" {
		t.Errorf("checks mismatch: %v", got["checks"])
	}
}

// TestStopJSONContract pins the stop/clean "data" envelope shape.
func TestStopJSONContract(t *testing.T) {
	var buf bytes.Buffer
	report := control.StopReport{
		Stopped: []control.ResultItem{{Name: "postgres", Message: "stopped"}},
		Failed:  []control.ResultItem{{Name: "redis", Error: "boom"}},
		Message: "Stopped 1 targets, 1 failed",
	}
	if err := RenderStopReport(&buf, cliout.FormatJSON, report); err != nil {
		t.Fatalf("RenderStopReport: %v", err)
	}
	got := decodeJSON(t, buf.Bytes())
	if got["success"] != true {
		t.Errorf("success: want true, got %v", got["success"])
	}
	data := got["data"].(map[string]any)
	if data["message"] != "Stopped 1 targets, 1 failed" {
		t.Errorf("data.message mismatch: %v", data["message"])
	}
	if len(data["stopped"].([]any)) != 1 || len(data["failed"].([]any)) != 1 {
		t.Errorf("stopped/failed mismatch: %v", data)
	}
}

// TestOrphansJSONContract pins both orphan list shapes (live + dry-run).
func TestOrphansJSONContract(t *testing.T) {
	list := []maintenance.SystemProcess{{PID: 7, PPID: 1, Command: "vite"}}

	var live bytes.Buffer
	if err := RenderOrphansResponse(&live, cliout.FormatJSON, OrphansResponse{List: list}); err != nil {
		t.Fatalf("RenderOrphansResponse live: %v", err)
	}
	liveGot := decodeJSON(t, live.Bytes())
	if liveGot["success"] != true {
		t.Errorf("live success: %v", liveGot["success"])
	}
	orphans := liveGot["orphans"].([]any)
	if pid, ok := orphans[0].(map[string]any)["pid"].(float64); !ok || pid != 7 {
		t.Errorf("orphans[0].pid must be number 7, got %v", orphans[0])
	}

	var dry bytes.Buffer
	if err := RenderOrphansResponse(&dry, cliout.FormatJSON, OrphansResponse{List: list, DryRun: true}); err != nil {
		t.Fatalf("RenderOrphansResponse dry: %v", err)
	}
	dryGot := decodeJSON(t, dry.Bytes())
	dr := dryGot["dry_run"].(map[string]any)
	if len(dr["orphans"].([]any)) != 1 {
		t.Errorf("dry_run.orphans mismatch: %v", dr)
	}
}

// TestLocksJSONContract pins the `vrooli locks --json` list-mode shape.
func TestLocksJSONContract(t *testing.T) {
	var buf bytes.Buffer
	resp := LocksResponse{
		List: []maintenance.LockInfo{{Port: 5329, Scenario: "web", PID: 99, Path: "/tmp/lock", Stale: true}},
		RuntimeClaims: []maintenance.RuntimeClaimInfo{{
			ClaimID:    "c1",
			Scenario:   "web",
			Port:       5329,
			Generation: 5,
			CreatedAt:  time.Unix(1700000000, 0).UTC(),
		}},
	}
	if err := RenderLocksResponse(&buf, cliout.FormatJSON, resp); err != nil {
		t.Fatalf("RenderLocksResponse: %v", err)
	}
	got := decodeJSON(t, buf.Bytes())
	if got["success"] != true {
		t.Errorf("success: %v", got["success"])
	}
	locks := got["locks"].([]any)
	if port, ok := locks[0].(map[string]any)["port"].(float64); !ok || port != 5329 {
		t.Errorf("locks[0].port must be number 5329, got %v", locks[0])
	}
	claims := got["registry_claims"].([]any)
	c0 := claims[0].(map[string]any)
	if c0["claim_id"] != "c1" {
		t.Errorf("registry_claims[0].claim_id mismatch: %v", c0)
	}
	if c0["created_at"] == "" || c0["created_at"] == nil {
		t.Errorf("created_at should be RFC3339Nano string, got %v", c0["created_at"])
	}
	// generation is int64 -> protojson STRING-encodes it; documented exception.
	if _, ok := c0["generation"].(string); !ok {
		t.Errorf("generation (int64) expected JSON string, got %T", c0["generation"])
	}
}

// TestTemplateCleanupJSONContract pins the camelCase cleanup envelope.
func TestTemplateCleanupJSONContract(t *testing.T) {
	var buf bytes.Buffer
	result := templatevalidation.CleanupResult{
		CleanupPlan: templatevalidation.CleanupPlan{
			DryRun:    true,
			OlderThan: time.Hour,
			RunID:     "r1",
			Eligible: []templatevalidation.Run{{
				MarkerPath: "/tmp/m",
				Marker:     templatevalidation.RunMarker{RunID: "r1", Template: "react-vite", CreatorPID: 12},
				Age:        "1h",
			}},
		},
		Message: "ok",
	}
	resp := TemplateValidationCleanupResponse{Result: result}
	if err := RenderTemplateValidationCleanupResponse(&buf, cliout.FormatJSON, resp); err != nil {
		t.Fatalf("RenderTemplateValidationCleanupResponse: %v", err)
	}
	got := decodeJSON(t, buf.Bytes())
	if got["success"] != true { // len(Failures)==0
		t.Errorf("success: want true, got %v", got["success"])
	}
	cleanup := got["cleanup"].(map[string]any)
	// camelCase keys (json_name) — NOT snake_case.
	if _, ok := cleanup["dryRun"]; !ok {
		t.Errorf("expected camelCase key dryRun, got keys %v", cleanup)
	}
	if cleanup["runId"] != "r1" {
		t.Errorf("runId mismatch: %v", cleanup["runId"])
	}
	elig := cleanup["eligible"].([]any)
	m0 := elig[0].(map[string]any)["marker"].(map[string]any)
	if pid, ok := m0["creatorPid"].(float64); !ok || pid != 12 {
		t.Errorf("marker.creatorPid must be number 12, got %v", m0["creatorPid"])
	}
}

// TestPortDiagnosticJSONContract pins the `vrooli diagnose-port --json` shape.
func TestPortDiagnosticJSONContract(t *testing.T) {
	var buf bytes.Buffer
	listenerPID := 88
	diag := maintenance.PortDiagnostic{
		Port:     5329,
		Scenario: "web",
		InUse:    true,
		Listeners: []maintenance.PortListener{
			{PID: 88, Command: "node", Zombie: false},
		},
		ListenerInspection: network.ListenerInspection{Available: true, Tool: "ss"},
		Lock:               &maintenance.LockInfo{Port: 5329, Path: "/tmp/lock"},
		RegistryClaims:     []maintenance.RuntimeClaimInfo{{ClaimID: "c1", Scenario: "web", Port: 5329}},
		RegistryProcesses:  []maintenance.RuntimeProcessRefInfo{{RefID: "p1", PID: &listenerPID}},
		HostOrphanCount:    2,
		Recommendations:    []string{"do thing"},
		PortPolicy:         maintenance.PortPolicyReport{EphemeralMin: 32768, EphemeralMax: 60999, CanonicalBand: "api"},
	}
	if err := RenderPortDiagnostic(&buf, cliout.FormatJSON, diag); err != nil {
		t.Fatalf("RenderPortDiagnostic: %v", err)
	}
	got := decodeJSON(t, buf.Bytes())
	if got["success"] != true {
		t.Errorf("success: %v", got["success"])
	}
	d := got["diagnostic"].(map[string]any)
	if port, ok := d["port"].(float64); !ok || port != 5329 {
		t.Errorf("diagnostic.port must be number 5329, got %v", d["port"])
	}
	if _, ok := d["host_orphan_count"].(float64); !ok {
		t.Errorf("host_orphan_count must be a JSON number, got %T", d["host_orphan_count"])
	}
	pp := d["port_policy"].(map[string]any)
	if _, ok := pp["ephemeral_min"].(float64); !ok {
		t.Errorf("port_policy.ephemeral_min must be a number, got %T", pp["ephemeral_min"])
	}
	li := d["listener_inspection"].(map[string]any)
	if li["tool"] != "ss" {
		t.Errorf("listener_inspection.tool mismatch: %v", li)
	}
	procs := d["registry_processes"].([]any)
	if pid, ok := procs[0].(map[string]any)["pid"].(float64); !ok || pid != 88 {
		t.Errorf("registry_processes[0].pid (structpb number) must be 88, got %v", procs[0])
	}
}
