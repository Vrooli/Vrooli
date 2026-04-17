package cloudflared

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
	Name:        "cloudflared",
	Description: "Cloudflare Tunnel client",
	Commands:    []string{"cloudflared"},
	VersionArgs: []string{"--version"},
	Handler:     "cloudflared",
	Packages:    map[string]string{"brew": "cloudflare/cloudflare/cloudflared", "winget": "Cloudflare.cloudflared"},
	InstallHint: "Install cloudflared",
	Platforms:   []string{"linux", "macos", "windows"},
}

func newTestHandler() hostreqkit.Handler {
	return NewHandler(testManifest)
}

func baseRequirement() hostreqspec.ResolvedRequirement {
	return hostreqspec.ResolvedRequirement{
		Name: "cloudflared",
		Kind: hostreqspec.KindTool,
	}
}

// --- Name and Kind ---

func TestNameAndKind(t *testing.T) {
	h := newTestHandler()
	if h.Name() != "cloudflared" {
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
	if status.PackageName != "cloudflared" {
		t.Fatalf("PackageName = %q", status.PackageName)
	}
}

func TestInspectLinuxAptInstalled(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	hostreqkit.LookPathFn = func(name string) (string, error) {
		if name == "cloudflared" {
			return "/usr/bin/cloudflared", nil
		}
		return "", os.ErrNotExist
	}
	hostreqkit.CombinedOutputFn = func(name string, args ...string) ([]byte, error) {
		return []byte("cloudflared version 2024.2.1\n"), nil
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
	if status.Version != "cloudflared version 2024.2.1" {
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
	if status.PackageName != "cloudflare/cloudflare/cloudflared" {
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
	if status.PackageName != "Cloudflare.cloudflared" {
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

func TestApplyLinuxAptFullFlow(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	var commands []string
	hostreqkit.LookPathFn = func(name string) (string, error) {
		if name == "cloudflared" && len(commands) > 0 {
			return "/usr/bin/cloudflared", nil
		}
		return "", os.ErrNotExist
	}
	hostreqkit.RunCommandFn = func(name string, args []string, opts hostreqkit.EnsureOptions) error {
		commands = append(commands, name+" "+strings.Join(args, " "))
		return nil
	}
	hostreqkit.CombinedOutputFn = func(name string, args ...string) ([]byte, error) {
		return []byte("cloudflared version 2024.2.1\n"), nil
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
	// Expected: mkdir keyrings, mkdir sources.list.d, install key, install source, apt-get update, apt-get install
	if len(commands) < 5 {
		t.Fatalf("expected at least 5 commands, got %d: %v", len(commands), commands)
	}
}

func TestApplyLinuxKeyDownloadFailure(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

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
		if name == "cloudflared" && len(commands) > 0 {
			return "/usr/local/bin/cloudflared", nil
		}
		return "", os.ErrNotExist
	}
	hostreqkit.RunCommandFn = func(name string, args []string, opts hostreqkit.EnsureOptions) error {
		commands = append(commands, name+" "+strings.Join(args, " "))
		return nil
	}
	hostreqkit.CombinedOutputFn = func(name string, args ...string) ([]byte, error) {
		return []byte("cloudflared version 2024.2.1\n"), nil
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
