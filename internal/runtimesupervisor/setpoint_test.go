package runtimesupervisor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/vrooli/internal/setpoint"
)

// The supervisor's memory and CPU pressure gates read the setpoint file; the
// environment overrides the file, and the compiled fallback is the setpoint
// package's, not a constant of this package.
func TestSupervisorPressureProviderReadsSetpoint(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "setpoint.json")
	doc := `{"schema_version":"1.0.0","confidence":{"level":"SKETCH","rationale":"test","recorded_on":"2026-09-02"},"bars":[
	 {"id":"cpu","cell_ref":"substrate/SB14","projection":"substrate","target_kind":"cpu","deadband":"d","sustain":"10m","actuator":"a","decision_ref":"r","unit":"percent","max":77,"gradeable":true},
	 {"id":"mem","cell_ref":"substrate/SB20","projection":"substrate","target_kind":"mem","deadband":"d","sustain":"one read","actuator":"a","decision_ref":"r","unit":"percent","max":33,"gradeable":true}]}`
	if err := os.WriteFile(path, []byte(doc), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(setpoint.PathEnv, path)
	t.Setenv("VROOLI_RUNTIME_PRESSURE_SOME_AVG10", "")
	t.Setenv("VROOLI_RUNTIME_PRESSURE_CPU_SOME_AVG10", "")
	cfg := EnvConfig()
	provider, ok := cfg.PressureProvider.(*HostPressureProvider)
	if !ok {
		t.Fatalf("PressureProvider = %T", cfg.PressureProvider)
	}
	if provider.someAvg10Threshold != 33 || provider.cpuSomeAvg10Threshold != 77 || cfg.PressureSomeAvg10 != 33 {
		t.Fatalf("thresholds = mem %v cpu %v cfg %v, want 33/77/33 from the setpoint", provider.someAvg10Threshold, provider.cpuSomeAvg10Threshold, cfg.PressureSomeAvg10)
	}
	t.Setenv("VROOLI_RUNTIME_PRESSURE_SOME_AVG10", "12")
	if got := EnvConfig().PressureProvider.(*HostPressureProvider).someAvg10Threshold; got != 12 {
		t.Fatalf("environment override = %v, want 12", got)
	}
	fallback := NewHostPressureProviderWithCPU(0, 0)
	want := setpoint.Fallback()
	if fallback.someAvg10Threshold != want.Max(setpoint.CellMemoryPSI, -1) || fallback.cpuSomeAvg10Threshold != want.Max(setpoint.CellCPUPressure, -1) {
		t.Fatalf("fallback thresholds %v/%v are not the setpoint's", fallback.someAvg10Threshold, fallback.cpuSomeAvg10Threshold)
	}
}
