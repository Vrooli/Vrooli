package buf

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

var testManifest = hostreqkit.ToolManifest{
	Name:        "buf",
	Description: "Buf CLI",
	Commands:    []string{"buf"},
	VersionArgs: []string{"--version"},
	Handler:     "buf",
	Version:     "1.37.0",
	Packages:    map[string]string{"brew": "bufbuild/buf/buf", "winget": "Bufbuild.Buf"},
	Platforms:   []string{"linux", "macos", "windows"},
}

func newHandler() hostreqkit.Handler { return NewHandler(testManifest) }

func baseRequirement() hostreqspec.ResolvedRequirement {
	return hostreqspec.ResolvedRequirement{Name: "buf", Kind: hostreqspec.KindTool}
}

func stub(t *testing.T) (restore func()) {
	t.Helper()
	origLookPath := hostreqkit.LookPathFn
	origCombinedOutput := hostreqkit.CombinedOutputFn
	origRunCommand := hostreqkit.RunCommandFn
	origDownload := DownloadFn
	origUserHome := UserHomeDirFn
	return func() {
		hostreqkit.LookPathFn = origLookPath
		hostreqkit.CombinedOutputFn = origCombinedOutput
		hostreqkit.RunCommandFn = origRunCommand
		DownloadFn = origDownload
		UserHomeDirFn = origUserHome
	}
}

func TestNameAndKind(t *testing.T) {
	h := newHandler()
	if h.Name() != "buf" {
		t.Fatalf("Name = %q", h.Name())
	}
	if h.Kind() != hostreqspec.KindTool {
		t.Fatalf("Kind = %q", h.Kind())
	}
}

func TestInspectAlreadyInstalled(t *testing.T) {
	defer stub(t)()
	hostreqkit.LookPathFn = func(name string) (string, error) {
		if name == "buf" {
			return "/usr/local/bin/buf", nil
		}
		return "", os.ErrNotExist
	}
	hostreqkit.CombinedOutputFn = func(string, ...string) ([]byte, error) {
		return []byte("1.37.0\n"), nil
	}
	status := newHandler().Inspect(hostreqkit.Host{OS: "linux", PackageManager: "apt"}, baseRequirement())
	if !status.Installed || status.Version == "" {
		t.Fatalf("expected installed+version; got Installed=%v Version=%q", status.Installed, status.Version)
	}
	if status.ExecutionState != hostreqkit.ExecutionAlreadyPresent {
		t.Fatalf("ExecutionState = %q", status.ExecutionState)
	}
}

func TestInspectLinuxAssignsReleaseAsset(t *testing.T) {
	defer stub(t)()
	hostreqkit.LookPathFn = func(string) (string, error) { return "", os.ErrNotExist }
	status := newHandler().Inspect(hostreqkit.Host{OS: "linux", PackageManager: "apt"}, baseRequirement())
	// The asset name embeds runtime.GOARCH translation, so we only assert
	// the platform prefix (Linux) and that the package name was set.
	if !strings.HasPrefix(status.PackageName, "buf-Linux-") {
		t.Fatalf("PackageName = %q; want buf-Linux-<arch>", status.PackageName)
	}
	if !status.InstallSupported {
		t.Fatal("InstallSupported should be true on linux")
	}
}

func TestInspectDarwinAssignsReleaseAsset(t *testing.T) {
	defer stub(t)()
	hostreqkit.LookPathFn = func(string) (string, error) { return "", os.ErrNotExist }
	status := newHandler().Inspect(hostreqkit.Host{OS: "darwin", PackageManager: "brew"}, baseRequirement())
	if !strings.HasPrefix(status.PackageName, "buf-Darwin-") {
		t.Fatalf("PackageName = %q; want buf-Darwin-<arch>", status.PackageName)
	}
}

func TestInspectWindowsUsesWinget(t *testing.T) {
	defer stub(t)()
	hostreqkit.LookPathFn = func(string) (string, error) { return "", os.ErrNotExist }
	status := newHandler().Inspect(hostreqkit.Host{OS: "windows", PackageManager: "winget"}, baseRequirement())
	if status.PackageName != "Bufbuild.Buf" {
		t.Fatalf("PackageName = %q", status.PackageName)
	}
}

func TestInspectUnsupportedOS(t *testing.T) {
	defer stub(t)()
	hostreqkit.LookPathFn = func(string) (string, error) { return "", os.ErrNotExist }
	status := newHandler().Inspect(hostreqkit.Host{OS: "freebsd"}, baseRequirement())
	if status.SupportClass != hostreqkit.SupportUnsupported {
		t.Fatalf("SupportClass = %q", status.SupportClass)
	}
}

func TestApplyDryRunReportsURL(t *testing.T) {
	defer stub(t)()
	hostreqkit.LookPathFn = func(string) (string, error) { return "", os.ErrNotExist }
	status := newHandler().Inspect(hostreqkit.Host{OS: "linux", PackageManager: "apt"}, baseRequirement())
	out, err := newHandler().Apply(hostreqkit.Host{OS: "linux", PackageManager: "apt"}, status, hostreqkit.EnsureOptions{DryRun: true})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if out.ExecutionState != hostreqkit.ExecutionWouldInstall {
		t.Fatalf("ExecutionState = %q", out.ExecutionState)
	}
	matched := false
	for _, n := range out.Notes {
		if strings.Contains(n, "github.com/bufbuild/buf/releases") && strings.Contains(n, "v1.37.0") {
			matched = true
		}
	}
	if !matched {
		t.Fatalf("dry-run note should reference the release URL with v1.37.0; got %v", out.Notes)
	}
}

func TestApplyInstallsToHomeWithoutSudo(t *testing.T) {
	defer stub(t)()
	tmp := t.TempDir()
	UserHomeDirFn = func() (string, error) { return tmp, nil }
	// No sudo — chooseInstallDir falls back to ~/.local/bin
	hostreqkit.LookPathFn = func(name string) (string, error) {
		switch name {
		case "buf":
			candidate := filepath.Join(tmp, ".local", "bin", "buf")
			if _, err := os.Stat(candidate); err == nil {
				return candidate, nil
			}
		}
		return "", os.ErrNotExist
	}
	hostreqkit.CombinedOutputFn = func(string, ...string) ([]byte, error) {
		return []byte("1.37.0\n"), nil
	}
	DownloadFn = func(url string) ([]byte, error) {
		if !strings.Contains(url, "buf-") || !strings.Contains(url, "v1.37.0") {
			return nil, errors.New("unexpected URL " + url)
		}
		return []byte("#!/bin/sh\necho 1.37.0\n"), nil
	}

	status := newHandler().Inspect(hostreqkit.Host{OS: "linux", PackageManager: "apt"}, baseRequirement())
	out, err := newHandler().Apply(hostreqkit.Host{OS: "linux", PackageManager: "apt"}, status, hostreqkit.EnsureOptions{SudoMode: "skip"})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if out.ExecutionState != hostreqkit.ExecutionInstalled {
		t.Fatalf("ExecutionState = %q (notes=%v)", out.ExecutionState, out.Notes)
	}
	binPath := filepath.Join(tmp, ".local", "bin", "buf")
	if _, err := os.Stat(binPath); err != nil {
		t.Fatalf("buf not installed at %s: %v", binPath, err)
	}
}

func TestPinnedVersionMatchesManifest(t *testing.T) {
	if testManifest.Version != defaultVersion {
		t.Fatalf("test manifest version %q drifted from defaultVersion %q", testManifest.Version, defaultVersion)
	}
}

func TestReleaseFilenameSanity(t *testing.T) {
	h := handler{manifest: testManifest}
	if h.releaseFilename(hostreqkit.Host{OS: "linux"}) == "" && (runtime.GOARCH == "amd64" || runtime.GOARCH == "arm64") {
		t.Fatal("expected non-empty release filename on supported linux arch")
	}
	if h.releaseFilename(hostreqkit.Host{OS: "freebsd"}) != "" {
		t.Fatal("expected empty release filename on unsupported OS")
	}
}
