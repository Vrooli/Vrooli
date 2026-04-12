package runtime

import (
	"os"
	goruntime "runtime"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreq"
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

func TestInspectRejectsImplicitLegacyResolution(t *testing.T) {
	if _, err := Inspect("development"); err == nil || !strings.Contains(err.Error(), "explicit host requirements") {
		t.Fatalf("Inspect error = %v", err)
	}
}

func TestEnsureRejectsImplicitLegacyResolution(t *testing.T) {
	_, err := Ensure(EnsureOptions{Environment: "development"})
	if err == nil || !strings.Contains(err.Error(), "explicit host requirements") {
		t.Fatalf("Ensure error = %v", err)
	}
}

func TestEnsureRequirementsSupportsDeclaredToolAndSafeguard(t *testing.T) {
	restore := stubRuntimeLookups(t)
	defer restore()

	lookPathFn = func(name string) (string, error) {
		switch name {
		case "docker":
			return "/usr/bin/docker", nil
		default:
			return "", os.ErrNotExist
		}
	}

	report, err := EnsureRequirements(EnsureOptions{
		Environment: "development",
		DryRun:      true,
		AutoInstall: true,
		SudoMode:    "skip",
	}, hostreq.Resolution{
		Tools: []hostreq.ResolvedRequirement{
			{Name: "tmux", Kind: hostreq.KindTool, Required: true, Reasons: []string{"scenario tmux"}},
			{Name: "sqlite", Kind: hostreq.KindTool, Required: false, Manual: true, Reasons: []string{"manual sqlite"}},
		},
		Safeguards: []hostreq.ResolvedRequirement{
			{Name: "remote_session_protection", Kind: hostreq.KindSafeguard, Required: false, Reasons: []string{"linux guard"}},
		},
	})
	if err == nil {
		t.Fatal("expected missing required tmux error")
	}

	tmux := findStatus(t, report.Tools, "tmux")
	if tmux.SupportClass != SupportSupported {
		t.Fatalf("tmux support class = %q", tmux.SupportClass)
	}
	if !containsNote(tmux.Notes, "dry-run: would run") {
		t.Fatalf("tmux notes = %v", tmux.Notes)
	}

	sqlite := findStatus(t, report.Tools, "sqlite")
	if sqlite.SupportClass != SupportManualOnly {
		t.Fatalf("sqlite support class = %q", sqlite.SupportClass)
	}

	safeguard := findStatus(t, report.Safeguards, "remote_session_protection")
	if safeguard.Kind != hostreq.KindSafeguard {
		t.Fatalf("safeguard kind = %q", safeguard.Kind)
	}
}

func TestInspectRequirementsMarksUnknownHandlerUnsupported(t *testing.T) {
	report, err := InspectRequirements("development", hostreq.Resolution{
		Tools: []hostreq.ResolvedRequirement{
			{Name: "missing-tool", Kind: hostreq.KindTool, Required: true},
		},
	})
	if err != nil {
		t.Fatalf("InspectRequirements: %v", err)
	}
	status := findStatus(t, report.Tools, "missing-tool")
	if status.SupportClass != SupportUnsupported {
		t.Fatalf("support class = %q", status.SupportClass)
	}
}

func TestInspectRequirementsIncludesNewCoreHandlers(t *testing.T) {
	restore := stubRuntimeLookups(t)
	defer restore()

	lookPathFn = func(name string) (string, error) {
		switch name {
		case "git", "curl":
			return "/usr/bin/" + name, nil
		default:
			return "", os.ErrNotExist
		}
	}
	combinedOutputFn = func(name string, args ...string) ([]byte, error) {
		return []byte(name + " version\n"), nil
	}

	report, err := InspectRequirements("development", hostreq.Resolution{
		Tools: []hostreq.ResolvedRequirement{
			{Name: "git", Kind: hostreq.KindTool, Required: true},
			{Name: "curl", Kind: hostreq.KindTool, Required: true},
			{Name: "jq", Kind: hostreq.KindTool, Required: true},
		},
	})
	if err != nil {
		t.Fatalf("InspectRequirements: %v", err)
	}

	git := findStatus(t, report.Tools, "git")
	if !git.Installed || git.SupportClass != SupportSupported {
		t.Fatalf("git status = %+v", git)
	}
	jq := findStatus(t, report.Tools, "jq")
	if jq.SupportClass != SupportSupported || jq.PackageName != "jq" {
		t.Fatalf("jq status = %+v", jq)
	}
	if !contains(report.MissingRequired, "jq") {
		t.Fatalf("missing required = %v", report.MissingRequired)
	}
}

func TestRemoteSessionProtectionClassifiesUnsupportedAndNotApplicable(t *testing.T) {
	unsupported := inspectRequirement(Host{
		OS: "darwin",
	}, hostreq.ResolvedRequirement{
		Name: "remote_session_protection",
		Kind: hostreq.KindSafeguard,
	})
	if unsupported.SupportClass != SupportUnsupported {
		t.Fatalf("unsupported class = %q", unsupported.SupportClass)
	}

	notApplicable := inspectRequirement(Host{
		OS:              "linux",
		SupportsSysctl:  false,
		SupportsSystemd: false,
	}, hostreq.ResolvedRequirement{
		Name: "remote_session_protection",
		Kind: hostreq.KindSafeguard,
	})
	if notApplicable.SupportClass != SupportNotApplicable {
		t.Fatalf("notApplicable class = %q", notApplicable.SupportClass)
	}
}

func TestInstallCommandMappings(t *testing.T) {
	restore := stubRuntimeLookups(t)
	defer restore()

	lookPathFn = func(name string) (string, error) {
		if name == "sudo" {
			return "/usr/bin/sudo", nil
		}
		return "", os.ErrNotExist
	}

	command, args, err := installCommand(Host{OS: "linux", PackageManager: "apt-get"}, "jq", "ask")
	if err != nil {
		t.Fatalf("linux installCommand: %v", err)
	}
	if command != "sudo" || strings.Join(args, " ") != "apt-get install -y jq" {
		t.Fatalf("linux install command = %s %v", command, args)
	}

	command, args, err = installCommand(Host{OS: "darwin", PackageManager: "brew"}, "jq", "ask")
	if err != nil {
		t.Fatalf("darwin installCommand: %v", err)
	}
	if command != "brew" || strings.Join(args, " ") != "install jq" {
		t.Fatalf("darwin install command = %s %v", command, args)
	}
}

func stubRuntimeLookups(t *testing.T) func() {
	t.Helper()
	originalLookPathFn := lookPathFn
	originalCombinedOutputFn := combinedOutputFn
	return func() {
		lookPathFn = originalLookPathFn
		combinedOutputFn = originalCombinedOutputFn
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

func containsNote(values []string, target string) bool {
	for _, value := range values {
		if strings.Contains(value, target) {
			return true
		}
	}
	return false
}

func findStatus(t *testing.T, items []ItemStatus, name string) ItemStatus {
	t.Helper()
	for _, item := range items {
		if item.Name == name {
			return item
		}
	}
	t.Fatalf("status %q not found", name)
	return ItemStatus{}
}
