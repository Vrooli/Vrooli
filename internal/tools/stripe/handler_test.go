package stripe

import (
	"os"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqkit/hostreqkittest"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

var stubLookups = stripeStubLookups

func stripeStubLookups(t *testing.T) func() {
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
	Name:        "stripe",
	Description: "Stripe CLI",
	Commands:    []string{"stripe"},
	VersionArgs: []string{"version"},
	Handler:     "stripe",
	Packages:    map[string]string{"brew": "stripe/stripe-cli/stripe", "winget": "Stripe.StripeCli"},
	InstallHint: "Install the Stripe CLI",
	Platforms:   []string{"linux", "macos", "windows"},
}

var newTestHandler = stripeTestHandler

func stripeTestHandler() hostreqkit.Handler {
	return NewHandler(testManifest)
}

var baseRequirement = stripeBaseRequirement

func stripeBaseRequirement() hostreqspec.ResolvedRequirement {
	return hostreqspec.ResolvedRequirement{
		Name: "stripe",
		Kind: hostreqspec.KindTool,
	}
}

// --- Name and Kind ---

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
	if status.PackageName != "stripe" {
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
		if name == "stripe" {
			return "/usr/bin/stripe", nil
		}
		return "", os.ErrNotExist
	}
	hostreqkit.CombinedOutputFn = func(name string, args ...string) ([]byte, error) {
		return []byte("stripe version 1.29.0\n"), nil
	}

	h := newTestHandler()
	status := h.Inspect(
		hostreqkit.Host{OS: "linux", PackageManager: "apt"},
		baseRequirement(),
	)

	hostreqkittest.AssertInstalled(t, status, "stripe version 1.29.0")
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
	if !status.InstallSupported {
		t.Fatal("InstallSupported should be true for brew on macOS")
	}
	if status.PackageName != "stripe/stripe-cli/stripe" {
		t.Fatalf("PackageName = %q", status.PackageName)
	}
}

func TestInspectWindowsWinget(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	h := newTestHandler()
	status := h.Inspect(
		hostreqkit.Host{OS: "windows", PackageManager: "winget"},
		baseRequirement(),
	)

	if status.SupportClass != hostreqkit.SupportSupported {
		t.Fatalf("SupportClass = %q", status.SupportClass)
	}
	if !status.InstallSupported {
		t.Fatal("InstallSupported should be true for winget on Windows")
	}
	if status.PackageName != "Stripe.StripeCli" {
		t.Fatalf("PackageName = %q", status.PackageName)
	}
}

func TestInspectUnsupportedPlatform(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	hostreqkit.LookPathFn = func(string) (string, error) { return "", os.ErrNotExist }

	h := newTestHandler()
	status := h.Inspect(
		hostreqkit.Host{OS: "windows"},
		baseRequirement(),
	)

	if status.SupportClass != hostreqkit.SupportUnsupported {
		t.Fatalf("SupportClass = %q", status.SupportClass)
	}
	if status.ExecutionState != hostreqkit.ExecutionUnsupported {
		t.Fatalf("ExecutionState = %q", status.ExecutionState)
	}
}

func TestInspectLinuxNonAptUnsupported(t *testing.T) {
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

func TestInspectDarwinNonBrewUnsupported(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	hostreqkit.LookPathFn = func(string) (string, error) { return "", os.ErrNotExist }

	h := newTestHandler()
	status := h.Inspect(
		hostreqkit.Host{OS: "darwin", PackageManager: ""},
		baseRequirement(),
	)

	if status.SupportClass != hostreqkit.SupportUnsupported {
		t.Fatalf("SupportClass = %q", status.SupportClass)
	}
}

func TestInspectIncludesInstallHint(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	hostreqkit.LookPathFn = func(string) (string, error) { return "", os.ErrNotExist }

	h := newTestHandler()
	status := h.Inspect(
		hostreqkit.Host{OS: "linux", PackageManager: "apt-get"},
		baseRequirement(),
	)

	found := false
	for _, note := range status.Notes {
		if strings.Contains(note, "Install the Stripe CLI") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected install hint in Notes, got %v", status.Notes)
	}
}

// --- Apply tests ---

func TestApplyAlreadyInstalled(t *testing.T) {
	h := newTestHandler()
	status, err := h.Apply(
		hostreqkit.Host{OS: "linux", PackageManager: "apt-get"},
		hostreqkit.ItemStatus{Installed: true},
		hostreqkit.EnsureOptions{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if status.ExecutionState != hostreqkit.ExecutionAlreadyPresent {
		t.Fatalf("ExecutionState = %q", status.ExecutionState)
	}
}

func TestApplyNotApplicableReturnsEarly(t *testing.T) {
	h := newTestHandler()
	status, err := h.Apply(
		hostreqkit.Host{OS: "linux"},
		hostreqkit.ItemStatus{SupportClass: hostreqkit.SupportNotApplicable},
		hostreqkit.EnsureOptions{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if status.ExecutionState != hostreqkit.ExecutionNotApplicable {
		t.Fatalf("ExecutionState = %q", status.ExecutionState)
	}
}

func TestApplyLinuxDryRun(t *testing.T) {
	h := newTestHandler()
	status, err := h.Apply(
		hostreqkit.Host{OS: "linux", PackageManager: "apt-get"},
		hostreqkit.ItemStatus{SupportClass: hostreqkit.SupportSupported},
		hostreqkit.EnsureOptions{DryRun: true},
	)
	if err != nil {
		t.Fatal(err)
	}
	if status.ExecutionState != hostreqkit.ExecutionWouldInstall {
		t.Fatalf("ExecutionState = %q", status.ExecutionState)
	}
	found := false
	for _, note := range status.Notes {
		if strings.Contains(note, "dry-run") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected dry-run note, got %v", status.Notes)
	}
}

func TestApplyDarwinDryRun(t *testing.T) {
	h := newTestHandler()
	status, err := h.Apply(
		hostreqkit.Host{OS: "darwin", PackageManager: "brew"},
		hostreqkit.ItemStatus{SupportClass: hostreqkit.SupportSupported},
		hostreqkit.EnsureOptions{DryRun: true},
	)
	if err != nil {
		t.Fatal(err)
	}
	if status.ExecutionState != hostreqkit.ExecutionWouldInstall {
		t.Fatalf("ExecutionState = %q", status.ExecutionState)
	}
	found := false
	for _, note := range status.Notes {
		if strings.Contains(note, "brew install") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected brew install note, got %v", status.Notes)
	}
}

func TestApplyLinuxAptFullFlow(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	hostreqkit.LookPathFn = func(name string) (string, error) {
		switch name {
		case "sudo", "gpg":
			return "/usr/bin/" + name, nil
		case "stripe":
			return "/usr/bin/stripe", nil
		default:
			return "", os.ErrNotExist
		}
	}

	KeyDownloadFn = func() ([]byte, error) {
		return []byte("fake-gpg-key"), nil
	}

	var calls []string
	hostreqkit.RunCommandFn = func(name string, args []string, opts hostreqkit.EnsureOptions) error {
		calls = append(calls, name+" "+strings.Join(args, " "))
		return nil
	}
	hostreqkit.CombinedOutputFn = func(name string, args ...string) ([]byte, error) {
		return []byte("stripe version 1.29.0\n"), nil
	}

	h := newTestHandler()
	status, err := h.Apply(
		hostreqkit.Host{OS: "linux", PackageManager: "apt-get"},
		hostreqkit.ItemStatus{SupportClass: hostreqkit.SupportSupported},
		hostreqkit.EnsureOptions{SudoMode: "ask"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !status.Installed {
		t.Fatal("expected Installed = true")
	}
	if status.ExecutionState != hostreqkit.ExecutionInstalled {
		t.Fatalf("ExecutionState = %q", status.ExecutionState)
	}
	if status.Version != "stripe version 1.29.0" {
		t.Fatalf("Version = %q", status.Version)
	}

	// Verify key commands were run: gpg --dearmor, mkdir, install (keyring),
	// install (source), apt-get update, apt-get install
	needles := []string{
		"gpg --dearmor",
		"mkdir -p",
		"install -m 644",
		"apt-get update -qq",
		"apt-get install -y stripe",
	}
	for _, needle := range needles {
		found := false
		for _, call := range calls {
			if strings.Contains(call, needle) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected command containing %q, got %v", needle, calls)
		}
	}

	// Verify login hint in notes
	loginHint := false
	for _, note := range status.Notes {
		if strings.Contains(note, "stripe login") {
			loginHint = true
			break
		}
	}
	if !loginHint {
		t.Fatalf("expected login hint in Notes, got %v", status.Notes)
	}
}

func TestApplyLinuxInstallsGpgIfMissing(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	gpgInstalled := false
	hostreqkit.LookPathFn = func(name string) (string, error) {
		switch name {
		case "sudo":
			return "/usr/bin/sudo", nil
		case "gpg":
			if gpgInstalled {
				return "/usr/bin/gpg", nil
			}
			return "", os.ErrNotExist
		case "stripe":
			return "/usr/bin/stripe", nil
		default:
			return "", os.ErrNotExist
		}
	}

	KeyDownloadFn = func() ([]byte, error) {
		return []byte("fake-key"), nil
	}

	var calls []string
	hostreqkit.RunCommandFn = func(name string, args []string, opts hostreqkit.EnsureOptions) error {
		call := name + " " + strings.Join(args, " ")
		calls = append(calls, call)
		// After gpg install, mark it as available
		if strings.Contains(call, "apt-get install -y gpg") {
			gpgInstalled = true
		}
		return nil
	}
	hostreqkit.CombinedOutputFn = func(string, ...string) ([]byte, error) {
		return []byte("stripe version 1.29.0\n"), nil
	}

	h := newTestHandler()
	status, err := h.Apply(
		hostreqkit.Host{OS: "linux", PackageManager: "apt-get"},
		hostreqkit.ItemStatus{SupportClass: hostreqkit.SupportSupported},
		hostreqkit.EnsureOptions{SudoMode: "ask"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if status.ExecutionState != hostreqkit.ExecutionInstalled {
		t.Fatalf("ExecutionState = %q", status.ExecutionState)
	}

	// First real command should be gpg install
	foundGpgInstall := false
	for _, call := range calls {
		if strings.Contains(call, "apt-get install -y gpg") {
			foundGpgInstall = true
			break
		}
	}
	if !foundGpgInstall {
		t.Fatalf("expected gpg install command, got %v", calls)
	}
}

func TestApplyLinuxKeyDownloadFailure(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	hostreqkit.LookPathFn = func(name string) (string, error) {
		if name == "sudo" || name == "gpg" {
			return "/usr/bin/" + name, nil
		}
		return "", os.ErrNotExist
	}
	KeyDownloadFn = func() ([]byte, error) {
		return nil, os.ErrPermission
	}
	hostreqkit.RunCommandFn = func(string, []string, hostreqkit.EnsureOptions) error { return nil }

	h := newTestHandler()
	status, err := h.Apply(
		hostreqkit.Host{OS: "linux", PackageManager: "apt-get"},
		hostreqkit.ItemStatus{SupportClass: hostreqkit.SupportSupported},
		hostreqkit.EnsureOptions{SudoMode: "ask"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if status.ExecutionState != hostreqkit.ExecutionFailed {
		t.Fatalf("ExecutionState = %q, want failed", status.ExecutionState)
	}
}

func TestApplyDarwinBrewFlow(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	hostreqkit.LookPathFn = func(name string) (string, error) {
		if name == "stripe" {
			return "/usr/local/bin/stripe", nil
		}
		return "", os.ErrNotExist
	}

	var installedPkg string
	hostreqkit.RunCommandFn = func(name string, args []string, opts hostreqkit.EnsureOptions) error {
		installedPkg = name + " " + strings.Join(args, " ")
		return nil
	}
	hostreqkit.CombinedOutputFn = func(string, ...string) ([]byte, error) {
		return []byte("stripe version 1.29.0\n"), nil
	}

	h := newTestHandler()
	status, err := h.Apply(
		hostreqkit.Host{OS: "darwin", PackageManager: "brew"},
		hostreqkit.ItemStatus{SupportClass: hostreqkit.SupportSupported},
		hostreqkit.EnsureOptions{},
	)
	if err != nil {
		t.Fatal(err)
	}
	hostreqkittest.AssertExecutionState(t, status, hostreqkit.ExecutionInstalled)
	hostreqkittest.AssertSingleCommandContaining(t, []string{installedPkg}, "brew install stripe/stripe-cli/stripe")
}

func TestApplyDarwinBrewInstallFailure(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	hostreqkit.LookPathFn = func(string) (string, error) { return "", os.ErrNotExist }
	hostreqkit.RunCommandFn = func(string, []string, hostreqkit.EnsureOptions) error {
		return os.ErrPermission
	}

	h := newTestHandler()
	status, err := h.Apply(
		hostreqkit.Host{OS: "darwin", PackageManager: "brew"},
		hostreqkit.ItemStatus{SupportClass: hostreqkit.SupportSupported},
		hostreqkit.EnsureOptions{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if status.ExecutionState != hostreqkit.ExecutionFailed {
		t.Fatalf("ExecutionState = %q, want failed", status.ExecutionState)
	}
}

func TestApplyDefaultFallbackUnsupported(t *testing.T) {
	h := newTestHandler()
	status, err := h.Apply(
		hostreqkit.Host{OS: "linux", PackageManager: "dnf"},
		hostreqkit.ItemStatus{SupportClass: hostreqkit.SupportSupported},
		hostreqkit.EnsureOptions{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if status.SupportClass != hostreqkit.SupportUnsupported {
		t.Fatalf("SupportClass = %q", status.SupportClass)
	}
	if status.ExecutionState != hostreqkit.ExecutionUnsupported {
		t.Fatalf("ExecutionState = %q", status.ExecutionState)
	}
}

func TestApplyLinuxNotOnPathAfterInstall(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	hostreqkit.LookPathFn = func(name string) (string, error) {
		if name == "sudo" || name == "gpg" {
			return "/usr/bin/" + name, nil
		}
		return "", os.ErrNotExist
	}

	KeyDownloadFn = func() ([]byte, error) {
		return []byte("fake-key"), nil
	}
	hostreqkit.RunCommandFn = func(string, []string, hostreqkit.EnsureOptions) error { return nil }

	h := newTestHandler()
	status, err := h.Apply(
		hostreqkit.Host{OS: "linux", PackageManager: "apt-get"},
		hostreqkit.ItemStatus{SupportClass: hostreqkit.SupportSupported},
		hostreqkit.EnsureOptions{SudoMode: "ask"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if status.ExecutionState != hostreqkit.ExecutionFailed {
		t.Fatalf("ExecutionState = %q, want failed (not on PATH)", status.ExecutionState)
	}
	found := false
	for _, note := range status.Notes {
		if strings.Contains(note, "not available on PATH") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected PATH note, got %v", status.Notes)
	}
}
