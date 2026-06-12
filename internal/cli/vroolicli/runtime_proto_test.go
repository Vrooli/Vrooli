package vroolicli

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/runtimesupervisor"
)

// TestCliVersionJSONContract pins the `vrooli --version --json` wire shape.
func TestCliVersionJSONContract(t *testing.T) {
	var buf bytes.Buffer
	if err := writeCliVersionJSON(&buf, versionOutput{
		CLIVersion:      "1.4.2",
		PlatformVersion: "1.4.3",
		Root:            "/srv/vrooli",
	}); err != nil {
		t.Fatalf("writeCliVersionJSON: %v", err)
	}

	got := decode(t, buf.Bytes())
	if _, ok := got["success"]; ok {
		t.Errorf("version output must not carry a success envelope: %v", got)
	}
	if got["cli_version"] != "1.4.2" {
		t.Errorf("cli_version: %v", got["cli_version"])
	}
	if got["platform_version"] != "1.4.3" {
		t.Errorf("platform_version: %v", got["platform_version"])
	}
	if got["root"] != "/srv/vrooli" {
		t.Errorf("root: %v", got["root"])
	}
}

// TestCliSupervisorStatusJSONContract pins the `vrooli runtime supervisor
// status --json` wire shape, including a sparse zero-time / nil-PID case.
func TestCliSupervisorStatusJSONContract(t *testing.T) {
	pid := 4242
	report := runtimesupervisor.StatusReport{
		SupervisorID:                  "sup-1",
		Status:                        "running",
		StatusReason:                  "healthy",
		HostBootID:                    "boot-1",
		HostSessionID:                 "sess-1",
		PID:                           &pid,
		LastHeartbeatAt:               time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC),
		HeartbeatDeadlineAt:           time.Time{}, // sparse: zero -> ""
		SupervisedInstanceCount:       3,
		UnverifiedInstanceCount:       1,
		EffectiveRenewInterval:        10 * time.Second,
		EffectiveLeaseTTL:             45 * time.Second,
		EffectiveHealthInterval:       45 * time.Second,
		EffectiveMaxHealthConcurrency: 16,
		EffectiveBatchSize:            250,
		LastTick: runtimesupervisor.TickReport{
			SupervisorID:     "sup-1",
			Renewed:          5,
			Expired:          2,
			Unverified:       1,
			HealthProbeCount: 7,
		},
	}

	var buf bytes.Buffer
	if err := writeCliSupervisorStatusJSON(&buf, report); err != nil {
		t.Fatalf("writeCliSupervisorStatusJSON: %v", err)
	}

	got := decode(t, buf.Bytes())
	if _, ok := got["success"]; ok {
		t.Errorf("status output must not carry a success envelope: %v", got)
	}
	if got["supervisor_id"] != "sup-1" || got["status"] != "running" {
		t.Errorf("supervisor_id/status mismatch: %v", got)
	}
	if got["host_boot_id"] != "boot-1" || got["host_session_id"] != "sess-1" {
		t.Errorf("host ids mismatch: %v", got)
	}
	if got["last_heartbeat_at"] != "2026-06-11T12:00:00Z" {
		t.Errorf("last_heartbeat_at: %v", got["last_heartbeat_at"])
	}
	if got["heartbeat_deadline_at"] != "" {
		t.Errorf("zero deadline must map to empty string, got %v", got["heartbeat_deadline_at"])
	}

	// int32 fields must be JSON numbers (float64), not strings.
	if v, ok := got["pid"].(float64); !ok || v != 4242 {
		t.Errorf("pid must be a JSON number 4242, got %T %v", got["pid"], got["pid"])
	}
	if v, ok := got["supervised_instance_count"].(float64); !ok || v != 3 {
		t.Errorf("supervised_instance_count must be a JSON number 3, got %T %v", got["supervised_instance_count"], got["supervised_instance_count"])
	}
	if v, ok := got["effective_max_health_concurrency"].(float64); !ok || v != 16 {
		t.Errorf("effective_max_health_concurrency must be a number, got %T %v", got["effective_max_health_concurrency"], got["effective_max_health_concurrency"])
	}

	// int64 duration fields: protojson serializes int64 as a JSON STRING.
	if got["effective_renew_interval"] != "10000000000" {
		t.Errorf("effective_renew_interval (int64 ns): %v", got["effective_renew_interval"])
	}
	if got["effective_lease_ttl"] != "45000000000" {
		t.Errorf("effective_lease_ttl (int64 ns): %v", got["effective_lease_ttl"])
	}

	tick, ok := got["last_tick"].(map[string]any)
	if !ok {
		t.Fatalf("last_tick missing/wrong type: %v", got["last_tick"])
	}
	if v, ok := tick["renewed"].(float64); !ok || v != 5 {
		t.Errorf("last_tick.renewed must be number 5, got %T %v", tick["renewed"], tick["renewed"])
	}
	if v, ok := tick["health_probe_count"].(float64); !ok || v != 7 {
		t.Errorf("last_tick.health_probe_count must be number 7, got %v", tick["health_probe_count"])
	}
}

// TestCliSupervisorServiceResultJSONContract pins the install/uninstall shape.
func TestCliSupervisorServiceResultJSONContract(t *testing.T) {
	var buf bytes.Buffer
	if err := writeCliSupervisorServiceResultJSON(&buf, runtimesupervisor.ServiceInstallResult{
		UnitName: "vrooli-runtime-supervisor.service",
		UnitPath: "/home/u/.config/systemd/user/vrooli-runtime-supervisor.service",
		Scope:    "user",
		Active:   true,
	}); err != nil {
		t.Fatalf("writeCliSupervisorServiceResultJSON: %v", err)
	}

	got := decode(t, buf.Bytes())
	if _, ok := got["success"]; ok {
		t.Errorf("service result must not carry a success envelope: %v", got)
	}
	if got["unit_name"] != "vrooli-runtime-supervisor.service" {
		t.Errorf("unit_name: %v", got["unit_name"])
	}
	if got["unit_path"] == "" || got["scope"] != "user" {
		t.Errorf("unit_path/scope mismatch: %v", got)
	}
	if got["active"] != true {
		t.Errorf("active: want true, got %v", got["active"])
	}
}

func decode(t *testing.T, b []byte) map[string]any {
	t.Helper()
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, string(b))
	}
	return got
}
