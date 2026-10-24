package scenario

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/portspec"
)

func writeFileForTest(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}

func intPtr(v int) *int { return &v }

// fakeEphemeral installs a fixed ephemeral range for a subtest and restores
// the prior probe on cleanup.
func fakeEphemeral(t *testing.T, min, max int) {
	t.Helper()
	prev := portEphemeral
	portEphemeral = func(_ context.Context) portspec.EphemeralRange {
		return portspec.EphemeralRange{Min: min, Max: max, Source: "test"}
	}
	t.Cleanup(func() { portEphemeral = prev })
}

func TestValidateManifestPorts_FixedInEphemeral(t *testing.T) {
	fakeEphemeral(t, 32768, 60999)
	err := validateManifestPorts("/x.json", map[string]Port{
		"ui": {EnvVar: "UI_PORT", Port: intPtr(36234)},
	})
	if err == nil {
		t.Fatal("expected error for fixed port inside ephemeral range")
	}
	if !strings.Contains(err.Error(), "ephemeral range") {
		t.Errorf("message missing 'ephemeral range': %v", err)
	}
	if !strings.Contains(err.Error(), "vrooli-ports-migrate") {
		t.Errorf("message missing migration hint: %v", err)
	}
}

func TestValidateManifestPorts_FixedAboveCanonical(t *testing.T) {
	fakeEphemeral(t, 49152, 65535) // macOS-style
	err := validateManifestPorts("/x.json", map[string]Port{
		"ui": {EnvVar: "UI_PORT", Port: intPtr(40000)},
	})
	if err == nil || !strings.Contains(err.Error(), "canonical max") {
		t.Fatalf("expected canonical-max error; got %v", err)
	}
}

func TestValidateManifestPorts_FixedSafe(t *testing.T) {
	fakeEphemeral(t, 32768, 60999)
	if err := validateManifestPorts("/x.json", map[string]Port{
		"ui": {EnvVar: "UI_PORT", Port: intPtr(21234)},
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateManifestPorts_RangeInEphemeral(t *testing.T) {
	fakeEphemeral(t, 32768, 60999)
	err := validateManifestPorts("/x.json", map[string]Port{
		"ui": {EnvVar: "UI_PORT", Range: "35000-39999"},
	})
	if err == nil || !strings.Contains(err.Error(), "ephemeral range") {
		t.Fatalf("expected ephemeral-overlap error; got %v", err)
	}
}

func TestValidateManifestPorts_RangeSafe(t *testing.T) {
	fakeEphemeral(t, 32768, 60999)
	if err := validateManifestPorts("/x.json", map[string]Port{
		"ui":  {EnvVar: "UI_PORT", Range: "20000-24999"},
		"api": {EnvVar: "API_PORT", Range: "15000-19999"},
		"ws":  {EnvVar: "WS_PORT", Range: "25000-29999"},
	}); err != nil {
		t.Fatalf("unexpected error for canonical bands: %v", err)
	}
}

func TestValidateManifestPorts_RangePartialOverlap(t *testing.T) {
	fakeEphemeral(t, 32768, 60999)
	err := validateManifestPorts("/x.json", map[string]Port{
		"ui": {EnvVar: "UI_PORT", Range: "30000-40000"},
	})
	if err == nil {
		t.Fatal("expected error for range crossing ephemeral floor")
	}
	// 30000-32767 is headroom, but 32768-40000 is ephemeral; must flag.
	if !strings.Contains(err.Error(), "ephemeral") && !strings.Contains(err.Error(), "canonical max") {
		t.Errorf("message should mention ephemeral or canonical max: %v", err)
	}
}

func TestValidateManifestPorts_MalformedRange(t *testing.T) {
	fakeEphemeral(t, 32768, 60999)
	err := validateManifestPorts("/x.json", map[string]Port{
		"ui": {EnvVar: "UI_PORT", Range: "notarange"},
	})
	if err == nil || !strings.Contains(err.Error(), "invalid range") {
		t.Fatalf("expected invalid-range error; got %v", err)
	}
}

func TestValidateManifestPorts_InvertedRange(t *testing.T) {
	fakeEphemeral(t, 32768, 60999)
	err := validateManifestPorts("/x.json", map[string]Port{
		"ui": {EnvVar: "UI_PORT", Range: "24000-20000"},
	})
	if err == nil || !strings.Contains(err.Error(), "invalid range") {
		t.Fatalf("expected invalid-range error; got %v", err)
	}
}

func TestValidateManifestPorts_EmptyMap(t *testing.T) {
	fakeEphemeral(t, 32768, 60999)
	if err := validateManifestPorts("/x.json", nil); err != nil {
		t.Fatalf("expected nil for empty ports; got %v", err)
	}
}

func TestValidateManifestPorts_EscapeHatch(t *testing.T) {
	t.Setenv("VROOLI_PORT_VALIDATION", "off")
	fakeEphemeral(t, 32768, 60999)
	if err := validateManifestPorts("/x.json", map[string]Port{
		"ui": {EnvVar: "UI_PORT", Port: intPtr(36234)},
	}); err != nil {
		t.Fatalf("escape hatch should bypass validation; got %v", err)
	}
}

// TestReadService_DoesNotEnforcePortPolicy proves that loading a manifest
// with ephemeral-range ports succeeds, so Stop/Status/List paths remain
// usable during the transition. Policy is enforced at Start time via
// ValidateManifestPorts.
func TestReadService_DoesNotEnforcePortPolicy(t *testing.T) {
	fakeEphemeral(t, 32768, 60999)

	tmp := t.TempDir()
	path := tmp + "/service.json"
	content := `{
  "service": { "name": "sample" },
  "ports": {
    "ui": { "env_var": "UI_PORT", "port": 36234 }
  }
}
`
	if err := writeFileForTest(path, content); err != nil {
		t.Fatalf("write: %v", err)
	}
	manifest, err := ReadService(path)
	if err != nil {
		t.Fatalf("ReadService must tolerate ephemeral-range ports: %v", err)
	}
	if len(manifest.Ports) != 1 {
		t.Errorf("manifest.Ports = %+v", manifest.Ports)
	}
	// But the exported validator, called explicitly, does flag it.
	if err := ValidateManifestPorts(path, manifest.Ports); err == nil {
		t.Errorf("ValidateManifestPorts should reject ephemeral-range port")
	}
}

func TestValidateManifestPorts_StableErrorOrder(t *testing.T) {
	fakeEphemeral(t, 32768, 60999)
	err := validateManifestPorts("/x.json", map[string]Port{
		"zebra": {EnvVar: "Z", Port: intPtr(36234)},
		"alpha": {EnvVar: "A", Port: intPtr(36235)},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	// Expect alpha listed before zebra.
	if strings.Index(err.Error(), "\"alpha\"") > strings.Index(err.Error(), "\"zebra\"") {
		t.Errorf("errors not sorted by port name:\n%s", err.Error())
	}
}
