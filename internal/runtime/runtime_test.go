package runtime

import (
	"os"
	goruntime "runtime"
	"strings"
	"testing"
)

// AI_CHECK: GO_MIGRATION_TEST_QUALITY=3 | LAST: 2026-04-11

func TestCurrentMatchesRuntimeGOOS(t *testing.T) {
	host := Current()
	if host.OS != goruntime.GOOS {
		t.Fatalf("host.OS = %q, want %q", host.OS, goruntime.GOOS)
	}
}

func TestValidateSupportMatchesCapabilityFlags(t *testing.T) {
	host := Current()
	if host.SupportsSetup {
		if err := host.ValidateSetup(); err != nil {
			t.Fatalf("ValidateSetup: %v", err)
		}
	} else if err := host.ValidateSetup(); err == nil {
		t.Fatal("ValidateSetup succeeded on unsupported host")
	}

	if host.SupportsDevelop {
		if err := host.ValidateDevelop(); err != nil {
			t.Fatalf("ValidateDevelop: %v", err)
		}
	} else if err := host.ValidateDevelop(); err == nil {
		t.Fatal("ValidateDevelop succeeded on unsupported host")
	}
}

func TestUnsupportedHostErrorsIncludePlatformNotes(t *testing.T) {
	host := Host{
		OS:              "darwin",
		SupportsSetup:   false,
		SupportsDevelop: false,
		Notes:           []string{"project-level setup/develop are native, but resource and scenario lifecycle support still assumes Linux-oriented tooling"},
	}

	setupErr := host.ValidateSetup()
	if setupErr == nil || !strings.Contains(setupErr.Error(), "not supported on darwin") {
		t.Fatalf("ValidateSetup error = %v", setupErr)
	}
	if !strings.Contains(setupErr.Error(), "Linux-oriented tooling") {
		t.Fatalf("ValidateSetup error missing note: %v", setupErr)
	}

	developErr := host.ValidateDevelop()
	if developErr == nil || !strings.Contains(developErr.Error(), "not supported on darwin") {
		t.Fatalf("ValidateDevelop error = %v", developErr)
	}
}

func TestInspectReportsKnownToolSurface(t *testing.T) {
	restore := stubRuntimeLookups(t)
	defer restore()

	lookPathFn = func(name string) (string, error) {
		switch name {
		case "docker", "go", "node":
			return "/usr/bin/" + name, nil
		default:
			return "", os.ErrNotExist
		}
	}
	combinedOutputFn = func(name string, args ...string) ([]byte, error) {
		return []byte(name + " version\n"), nil
	}

	report, err := Inspect("development")
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	if report.Environment != "development" {
		t.Fatalf("environment = %q", report.Environment)
	}
	if len(report.Tools) != 5 {
		t.Fatalf("tool count = %d", len(report.Tools))
	}
	if !contains(report.MissingRequired, "python") {
		t.Fatalf("missing required = %v", report.MissingRequired)
	}
	if !contains(report.MissingOptional, "helm") {
		t.Fatalf("missing optional = %v", report.MissingOptional)
	}
}

func TestEnsureDryRunLeavesMissingToolUninstalled(t *testing.T) {
	restore := stubRuntimeLookups(t)
	defer restore()

	lookPathFn = func(name string) (string, error) {
		return "", os.ErrNotExist
	}

	report, err := Ensure(EnsureOptions{
		Environment: "minimal",
		DryRun:      true,
		AutoInstall: true,
		SudoMode:    "skip",
	})
	if err == nil {
		t.Fatal("expected missing required tools error")
	}
	if !contains(report.MissingRequired, "docker") {
		t.Fatalf("missing required = %v", report.MissingRequired)
	}
}

func stubRuntimeLookups(t *testing.T) func() {
	t.Helper()
	originalLookPathFn := lookPathFn
	originalCombinedOutputFn := combinedOutputFn
	originalInstallToolFn := installToolFn
	return func() {
		lookPathFn = originalLookPathFn
		combinedOutputFn = originalCombinedOutputFn
		installToolFn = originalInstallToolFn
	}
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
