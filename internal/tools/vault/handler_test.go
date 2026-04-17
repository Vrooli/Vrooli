package vault

import (
	"os"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

func stubLookups(t *testing.T) func() {
	t.Helper()
	origLookPath := hostreqkit.LookPathFn
	origReadFile := hostreqkit.ReadFileFn
	origCombinedOutput := hostreqkit.CombinedOutputFn
	origRunCommand := hostreqkit.RunCommandFn
	origKeyDownload := KeyDownloadFn
	return func() {
		hostreqkit.LookPathFn = origLookPath
		hostreqkit.ReadFileFn = origReadFile
		hostreqkit.CombinedOutputFn = origCombinedOutput
		hostreqkit.RunCommandFn = origRunCommand
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

func newTestHandler() hostreqkit.Handler {
	return NewHandler(testManifest)
}

func baseRequirement() hostreqspec.ResolvedRequirement {
	return hostreqspec.ResolvedRequirement{
		Name: "vault",
		Kind: hostreqspec.KindTool,
	}
}

// --- Name and Kind ---

func TestNameAndKind(t *testing.T) {
	h := newTestHandler()
	if h.Name() != "vault" {
		t.Fatalf("Name = %q", h.Name())
	}
	if h.Kind() != hostreqspec.KindTool {
		t.Fatalf("Kind = %q", h.Kind())
	}
}

// --- Inspect tests ---

func TestInspectLinuxAptNotInstalled(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	hostreqkit.LookPathFn = func(string) (string, error) { return "", os.ErrNotExist }

	h := newTestHandler()
	status := h.Inspect(
		hostreqkit.Host{OS: "linux", PackageManager: "apt-get"},
		baseRequirement(),
	)

	if status.SupportClass != hostreqkit.SupportSupported {
		t.Fatalf("SupportClass = %q", status.SupportClass)
	}
	if !status.InstallSupported {
		t.Fatal("InstallSupported should be true for apt-based Linux")
	}
	if status.PackageName != "vault" {
		t.Fatalf("PackageName = %q", status.PackageName)
	}
	if status.Installed {
		t.Fatal("should not be installed")
	}
}

func TestInspectLinuxAptInstalled(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	hostreqkit.LookPathFn = func(name string) (string, error) {
		if name == "vault" {
			return "/usr/bin/vault", nil
		}
		return "", os.ErrNotExist
	}
	hostreqkit.CombinedOutputFn = func(name string, args ...string) ([]byte, error) {
		return []byte("Vault v1.15.4\n"), nil
	}

	h := newTestHandler()
	status := h.Inspect(
		hostreqkit.Host{OS: "linux", PackageManager: "apt"},
		baseRequirement(),
	)

	if !status.Installed {
		t.Fatal("should be installed")
	}
	if status.ExecutionState != hostreqkit.ExecutionAlreadyPresent {
		t.Fatalf("ExecutionState = %q", status.ExecutionState)
	}
	if status.Version != "Vault v1.15.4" {
		t.Fatalf("Version = %q", status.Version)
	}
}

func TestInspectDarwinBrew(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	hostreqkit.LookPathFn = func(string) (string, error) { return "", os.ErrNotExist }

	h := newTestHandler()
	status := h.Inspect(
		hostreqkit.Host{OS: "darwin", PackageManager: "brew"},
		baseRequirement(),
	)

	if status.SupportClass != hostreqkit.SupportSupported {
		t.Fatalf("SupportClass = %q", status.SupportClass)
	}
	if status.PackageName != "hashicorp/tap/vault" {
		t.Fatalf("PackageName = %q", status.PackageName)
	}
}

func TestInspectWindowsWinget(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	hostreqkit.LookPathFn = func(string) (string, error) { return "", os.ErrNotExist }

	h := newTestHandler()
	status := h.Inspect(
		hostreqkit.Host{OS: "windows", PackageManager: "winget"},
		baseRequirement(),
	)

	if status.SupportClass != hostreqkit.SupportSupported {
		t.Fatalf("SupportClass = %q", status.SupportClass)
	}
	if status.PackageName != "Hashicorp.Vault" {
		t.Fatalf("PackageName = %q", status.PackageName)
	}
}

func TestInspectUnsupportedPlatform(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	hostreqkit.LookPathFn = func(string) (string, error) { return "", os.ErrNotExist }

	h := newTestHandler()
	status := h.Inspect(
		hostreqkit.Host{OS: "linux", PackageManager: "dnf"},
		baseRequirement(),
	)

	if status.SupportClass != hostreqkit.SupportUnsupported {
		t.Fatalf("SupportClass = %q", status.SupportClass)
	}
}

func TestInspectManualRequirement(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	h := newTestHandler()
	req := baseRequirement()
	req.Manual = true
	status := h.Inspect(hostreqkit.Host{OS: "linux", PackageManager: "apt"}, req)

	if status.SupportClass != hostreqkit.SupportManualOnly {
		t.Fatalf("SupportClass = %q", status.SupportClass)
	}
}

// --- Apply tests ---

func TestApplyAlreadyInstalled(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	h := newTestHandler()
	status := hostreqkit.ItemStatus{Installed: true}
	result, err := h.Apply(hostreqkit.Host{}, status, hostreqkit.EnsureOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if result.ExecutionState != hostreqkit.ExecutionAlreadyPresent {
		t.Fatalf("ExecutionState = %q", result.ExecutionState)
	}
}

func TestApplyManualReturnsEarly(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	h := newTestHandler()
	status := hostreqkit.ItemStatus{SupportClass: hostreqkit.SupportManualOnly}
	result, err := h.Apply(hostreqkit.Host{}, status, hostreqkit.EnsureOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if result.ExecutionState != hostreqkit.ExecutionManualActionRequired {
		t.Fatalf("ExecutionState = %q", result.ExecutionState)
	}
}

func TestApplyUnsupportedReturnsEarly(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	h := newTestHandler()
	status := hostreqkit.ItemStatus{SupportClass: hostreqkit.SupportUnsupported}
	result, err := h.Apply(hostreqkit.Host{}, status, hostreqkit.EnsureOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if result.ExecutionState != hostreqkit.ExecutionUnsupported {
		t.Fatalf("ExecutionState = %q", result.ExecutionState)
	}
}

func TestApplyLinuxDryRun(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	h := newTestHandler()
	status := hostreqkit.ItemStatus{SupportClass: hostreqkit.SupportSupported}
	result, err := h.Apply(
		hostreqkit.Host{OS: "linux", PackageManager: "apt"},
		status,
		hostreqkit.EnsureOptions{DryRun: true},
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.ExecutionState != hostreqkit.ExecutionWouldInstall {
		t.Fatalf("ExecutionState = %q", result.ExecutionState)
	}
}

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

func TestApplyLinuxAptFullFlow(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	var commands []string
	hostreqkit.LookPathFn = func(name string) (string, error) {
		if name == "gpg" {
			return "/usr/bin/gpg", nil
		}
		if name == "vault" && len(commands) > 0 {
			return "/usr/bin/vault", nil
		}
		return "", os.ErrNotExist
	}
	hostreqkit.RunCommandFn = func(name string, args []string, opts hostreqkit.EnsureOptions) error {
		commands = append(commands, name+" "+strings.Join(args, " "))
		return nil
	}
	hostreqkit.CombinedOutputFn = func(name string, args ...string) ([]byte, error) {
		return []byte("Vault v1.15.4\n"), nil
	}
	KeyDownloadFn = func() ([]byte, error) {
		return []byte("fake-gpg-key"), nil
	}

	h := newTestHandler()
	status := hostreqkit.ItemStatus{SupportClass: hostreqkit.SupportSupported}
	result, err := h.Apply(
		hostreqkit.Host{OS: "linux", PackageManager: "apt-get"},
		status,
		hostreqkit.EnsureOptions{AutoInstall: true, SudoMode: "skip"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.ExecutionState != hostreqkit.ExecutionInstalled {
		t.Fatalf("ExecutionState = %q, notes = %v", result.ExecutionState, result.Notes)
	}
	if len(commands) < 5 {
		t.Fatalf("expected at least 5 commands, got %d: %v", len(commands), commands)
	}
}

func TestApplyLinuxKeyDownloadFailure(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	hostreqkit.LookPathFn = func(name string) (string, error) {
		if name == "gpg" {
			return "/usr/bin/gpg", nil
		}
		return "", os.ErrNotExist
	}
	KeyDownloadFn = func() ([]byte, error) {
		return nil, os.ErrPermission
	}

	h := newTestHandler()
	status := hostreqkit.ItemStatus{SupportClass: hostreqkit.SupportSupported}
	result, err := h.Apply(
		hostreqkit.Host{OS: "linux", PackageManager: "apt"},
		status,
		hostreqkit.EnsureOptions{AutoInstall: true, SudoMode: "skip"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.ExecutionState != hostreqkit.ExecutionFailed {
		t.Fatalf("ExecutionState = %q", result.ExecutionState)
	}
}

func TestApplyDarwinBrewFlow(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	var commands []string
	hostreqkit.LookPathFn = func(name string) (string, error) {
		if name == "vault" && len(commands) > 0 {
			return "/usr/local/bin/vault", nil
		}
		return "", os.ErrNotExist
	}
	hostreqkit.RunCommandFn = func(name string, args []string, opts hostreqkit.EnsureOptions) error {
		commands = append(commands, name+" "+strings.Join(args, " "))
		return nil
	}
	hostreqkit.CombinedOutputFn = func(name string, args ...string) ([]byte, error) {
		return []byte("Vault v1.15.4\n"), nil
	}

	h := newTestHandler()
	status := hostreqkit.ItemStatus{SupportClass: hostreqkit.SupportSupported}
	result, err := h.Apply(
		hostreqkit.Host{OS: "darwin", PackageManager: "brew"},
		status,
		hostreqkit.EnsureOptions{AutoInstall: true},
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.ExecutionState != hostreqkit.ExecutionInstalled {
		t.Fatalf("ExecutionState = %q", result.ExecutionState)
	}
	if len(commands) != 1 || !strings.Contains(commands[0], "brew install") {
		t.Fatalf("expected brew install, got %v", commands)
	}
}

func TestApplyDefaultFallbackUnsupported(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	h := newTestHandler()
	status := hostreqkit.ItemStatus{SupportClass: hostreqkit.SupportSupported}
	result, err := h.Apply(
		hostreqkit.Host{OS: "linux", PackageManager: "dnf"},
		status,
		hostreqkit.EnsureOptions{AutoInstall: true},
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.SupportClass != hostreqkit.SupportUnsupported {
		t.Fatalf("SupportClass = %q", result.SupportClass)
	}
}
