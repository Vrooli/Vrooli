package vault

import (
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
)

var stubLookups = vaultStubLookups

func vaultStubLookups(t *testing.T) func() {
	t.Helper()
	origLookPath := hostreqkit.LookPathFn
	origReadFile := hostreqkit.ReadFileFn
	origCombinedOutput := hostreqkit.CombinedOutputFn
	origRunCommand := hostreqkit.RunCommandFn
	origElevationFacts := hostreqkit.ElevationFactsFn
	origKeyDownload := KeyDownloadFn
	hostreqkit.ElevationFactsFn = func() hostreqkit.ElevationFacts {
		return hostreqkit.ElevationFacts{Platform: "linux", Elevated: true, CanElevate: true, Mechanism: "test"}
	}
	return func() {
		hostreqkit.LookPathFn = origLookPath
		hostreqkit.ReadFileFn = origReadFile
		hostreqkit.CombinedOutputFn = origCombinedOutput
		hostreqkit.RunCommandFn = origRunCommand
		hostreqkit.ElevationFactsFn = origElevationFacts
		KeyDownloadFn = origKeyDownload
	}
}

var testManifest = hostreqkit.ToolManifest{
	Name:        "vault",
	Description: "HashiCorp Vault CLI",
	Commands:    []string{"vault"},
	VersionArgs: []string{"version"},
	Handler:     "vault",
	Packages:    map[string]string{"brew": "hashicorp/tap/vault", "winget": "Hashicorp.Vault"},
	InstallHint: "Install the Vault CLI",
	Platforms:   []string{"linux", "macos", "windows"},
}

var newTestHandler = vaultTestHandler

func vaultTestHandler() hostreqkit.Handler {
	return NewHandler(testManifest)
}

// --- Name and Kind ---

// --- Inspect tests ---

// --- Apply tests ---

func TestApplyDarwinDryRun(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	h := newTestHandler()
	status := hostreqkit.ItemStatus{SupportClass: hostreqkit.SupportSupported}
	result, err := h.Apply(
		hostreqkit.Host{OS: "darwin", PackageManager: "brew"},
		status,
		hostreqkit.EnsureOptions{DryRun: true},
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.ExecutionState != hostreqkit.ExecutionWouldInstall {
		t.Fatalf("ExecutionState = %q", result.ExecutionState)
	}
	found := false
	for _, note := range result.Notes {
		if strings.Contains(note, "brew install") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected brew install note, got %v", result.Notes)
	}
}
