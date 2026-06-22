package runtime

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/vrooli/binaryfetch"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

func writeFile(path string, data []byte) error { return os.WriteFile(path, data, 0o755) }

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func boolPtr(b bool) *bool { return &b }

func releaseManifest(name string, requires *hostreqspec.CapabilityRequirement) hostreqkit.ToolManifest {
	return hostreqkit.ToolManifest{
		Name:        name,
		Description: "test backend",
		Commands:    []string{name},
		VersionArgs: []string{"--version"},
		Requires:    requires,
		Source: &hostreqkit.ToolSource{
			Type: "release",
			Targets: map[string]hostreqkit.ToolSourceTarget{
				"linux/amd64": {URL: "https://example.test/" + name + ".tar.gz", SHA256: "deadbeef", Archive: "tar.gz", BinPath: "bin/" + name},
			},
		},
	}
}

func resolved(name string) hostreqspec.ResolvedRequirement {
	return hostreqspec.ResolvedRequirement{Name: name, Kind: hostreqspec.KindTool, Required: false}
}

func withFacts(t *testing.T, facts hostreqspec.CapabilityFacts) {
	t.Helper()
	prev := capabilityFactsFn
	capabilityFactsFn = func() hostreqspec.CapabilityFacts { return facts }
	t.Cleanup(func() { capabilityFactsFn = prev })
}

func withArch(t *testing.T, arch string) {
	t.Helper()
	prev := runtimeArch
	runtimeArch = func() string { return arch }
	t.Cleanup(func() { runtimeArch = prev })
}

// withUserBin points ~/.vrooli/bin at a temp dir and stubs command resolution
// so a "fetched" file there is seen as installed.
func withUserBin(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	prev := userLocalBinDir
	userLocalBinDir = func() (string, error) { return dir, nil }
	t.Cleanup(func() { userLocalBinDir = prev })
	return dir
}

func TestInspectFetch_GPURequiredSkippedOnCPUHost(t *testing.T) {
	withFacts(t, hostreqspec.CapabilityFacts{Arch: "amd64", RAMGb: 32}) // no GPU
	h := toolHandler{manifest: releaseManifest("sd", &hostreqspec.CapabilityRequirement{GPU: boolPtr(true)})}
	status := h.Inspect(hostreqkit.Host{OS: "linux"}, resolved("sd"))
	if status.ExecutionState != hostreqkit.ExecutionNotApplicable {
		t.Fatalf("state = %q, want not_applicable", status.ExecutionState)
	}
	if status.SupportClass != hostreqkit.SupportNotApplicable {
		t.Fatalf("support = %q, want not_applicable", status.SupportClass)
	}
}

func TestInspectFetch_ArchMismatchSkipped(t *testing.T) {
	withFacts(t, hostreqspec.CapabilityFacts{Arch: "arm64", RAMGb: 16})
	h := toolHandler{manifest: releaseManifest("iopaint", &hostreqspec.CapabilityRequirement{Arch: []string{"amd64"}})}
	status := h.Inspect(hostreqkit.Host{OS: "linux"}, resolved("iopaint"))
	if status.ExecutionState != hostreqkit.ExecutionNotApplicable {
		t.Fatalf("state = %q, want not_applicable", status.ExecutionState)
	}
}

func TestInspectFetch_NoTargetForHostIsUnsupported(t *testing.T) {
	withFacts(t, hostreqspec.CapabilityFacts{Arch: "arm64"})
	withArch(t, "arm64") // manifest only declares linux/amd64
	withUserBin(t)
	h := toolHandler{manifest: releaseManifest("sd", nil)}
	status := h.Inspect(hostreqkit.Host{OS: "linux"}, resolved("sd"))
	if status.ExecutionState != hostreqkit.ExecutionUnsupported {
		t.Fatalf("state = %q, want unsupported", status.ExecutionState)
	}
}

func TestApplyFetch_DryRunListsTargetURL(t *testing.T) {
	withFacts(t, hostreqspec.CapabilityFacts{Arch: "amd64"})
	withArch(t, "amd64")
	withUserBin(t)
	h := toolHandler{manifest: releaseManifest("sd", nil)}
	status := h.Inspect(hostreqkit.Host{OS: "linux"}, resolved("sd"))
	if !status.InstallSupported {
		t.Fatalf("expected InstallSupported for a host with a target")
	}
	applied, err := h.Apply(hostreqkit.Host{OS: "linux"}, status, hostreqkit.EnsureOptions{DryRun: true})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if applied.ExecutionState != hostreqkit.ExecutionWouldInstall {
		t.Fatalf("state = %q, want would_install", applied.ExecutionState)
	}
	if !notesContain(applied.Notes, "example.test/sd.tar.gz") {
		t.Fatalf("dry-run notes should name the target URL: %v", applied.Notes)
	}
}

func TestApplyFetch_SuccessViaStubFetcher(t *testing.T) {
	withFacts(t, hostreqspec.CapabilityFacts{Arch: "amd64"})
	withArch(t, "amd64")
	binDir := withUserBin(t)

	var fetched binaryfetch.Target
	prevFetch := fetchBinaryFn
	fetchBinaryFn = func(_ context.Context, spec binaryfetch.Target, destDir string, _ binaryfetch.ProgressFunc) (string, error) {
		fetched = spec
		// Simulate the binary landing in ~/.vrooli/bin.
		path := destDir + "/" + spec.Name
		if err := writeFile(path, []byte("#!/bin/sh\necho ok\n")); err != nil {
			return "", err
		}
		return path, nil
	}
	t.Cleanup(func() { fetchBinaryFn = prevFetch })

	h := toolHandler{manifest: releaseManifest("sd", nil)}
	status := h.Inspect(hostreqkit.Host{OS: "linux"}, resolved("sd"))
	applied, err := h.Apply(hostreqkit.Host{OS: "linux"}, status, hostreqkit.EnsureOptions{})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if applied.ExecutionState != hostreqkit.ExecutionInstalled {
		t.Fatalf("state = %q (notes %v), want installed", applied.ExecutionState, applied.Notes)
	}
	if fetched.URL != "https://example.test/sd.tar.gz" || fetched.SHA256 != "deadbeef" {
		t.Fatalf("fetcher received wrong spec: %+v", fetched)
	}
	if fetched.BinPath != "bin/sd" || fetched.Archive != "tar.gz" {
		t.Fatalf("fetcher spec archive/binpath wrong: %+v", fetched)
	}
	if !fileExists(binDir + "/sd") {
		t.Fatalf("expected binary written into ~/.vrooli/bin")
	}
}

func TestApplyFetch_FetchFailureIsFailed(t *testing.T) {
	withFacts(t, hostreqspec.CapabilityFacts{Arch: "amd64"})
	withArch(t, "amd64")
	withUserBin(t)
	prevFetch := fetchBinaryFn
	fetchBinaryFn = func(_ context.Context, _ binaryfetch.Target, _ string, _ binaryfetch.ProgressFunc) (string, error) {
		return "", binaryfetch.ErrChecksumMismatch
	}
	t.Cleanup(func() { fetchBinaryFn = prevFetch })

	h := toolHandler{manifest: releaseManifest("sd", nil)}
	status := h.Inspect(hostreqkit.Host{OS: "linux"}, resolved("sd"))
	applied, _ := h.Apply(hostreqkit.Host{OS: "linux"}, status, hostreqkit.EnsureOptions{})
	if applied.ExecutionState != hostreqkit.ExecutionFailed {
		t.Fatalf("state = %q, want failed", applied.ExecutionState)
	}
}

func TestInspectFetch_AlreadyPresentWhenInUserBin(t *testing.T) {
	withFacts(t, hostreqspec.CapabilityFacts{Arch: "amd64"})
	withArch(t, "amd64")
	binDir := withUserBin(t)
	if err := writeFile(binDir+"/sd", []byte("bin")); err != nil {
		t.Fatal(err)
	}
	h := toolHandler{manifest: releaseManifest("sd", nil)}
	status := h.Inspect(hostreqkit.Host{OS: "linux"}, resolved("sd"))
	if status.ExecutionState != hostreqkit.ExecutionAlreadyPresent {
		t.Fatalf("state = %q, want already_present", status.ExecutionState)
	}
}

func TestCapabilityGate_MetProceeds(t *testing.T) {
	withFacts(t, hostreqspec.CapabilityFacts{HasGPU: true, MaxVRAMGb: 8, Arch: "amd64", RAMGb: 32})
	withArch(t, "amd64")
	withUserBin(t)
	h := toolHandler{manifest: releaseManifest("sd", &hostreqspec.CapabilityRequirement{GPU: boolPtr(true), MinVRAMGb: 4})}
	status := h.Inspect(hostreqkit.Host{OS: "linux"}, resolved("sd"))
	if status.ExecutionState == hostreqkit.ExecutionNotApplicable {
		t.Fatalf("gate should pass with GPU+8GiB VRAM; got not_applicable")
	}
	if !status.InstallSupported {
		t.Fatalf("expected InstallSupported when gate met and target present")
	}
}

// dirReleaseManifest builds a dir-layout release manifest for one tool.
func dirReleaseManifest(name string) hostreqkit.ToolManifest {
	return hostreqkit.ToolManifest{
		Name:        name,
		Description: "test dir backend",
		Commands:    []string{name},
		VersionArgs: []string{"--help"},
		Source: &hostreqkit.ToolSource{
			Type: "release",
			Targets: map[string]hostreqkit.ToolSourceTarget{
				"linux/amd64": {URL: "https://example.test/" + name + ".zip", SHA256: "deadbeef", Archive: "zip", Layout: "dir", BinPath: name + "-cli"},
			},
		},
	}
}

// withUserOpt points ~/.vrooli/opt/<tool> at a temp dir.
func withUserOpt(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	prev := userLocalOptDir
	userLocalOptDir = func(tool string) (string, error) { return root + "/" + tool, nil }
	t.Cleanup(func() { userLocalOptDir = prev })
	return root
}

func TestApplyFetch_DirLayoutWritesLauncher(t *testing.T) {
	withFacts(t, hostreqspec.CapabilityFacts{Arch: "amd64"})
	withArch(t, "amd64")
	binDir := withUserBin(t)
	optRoot := withUserOpt(t)

	var gotSpec binaryfetch.Target
	var gotOptDir string
	prev := fetchDirFn
	fetchDirFn = func(_ context.Context, spec binaryfetch.Target, optDir string, _ binaryfetch.ProgressFunc) (string, error) {
		gotSpec = spec
		gotOptDir = optDir
		if err := os.MkdirAll(optDir, 0o755); err != nil {
			return "", err
		}
		entry := optDir + "/" + spec.BinPath
		if err := writeFile(entry, []byte("#!/bin/sh\necho ok\n")); err != nil {
			return "", err
		}
		return entry, nil
	}
	t.Cleanup(func() { fetchDirFn = prev })

	h := toolHandler{manifest: dirReleaseManifest("sd")}
	status := h.Inspect(hostreqkit.Host{OS: "linux"}, resolved("sd"))
	if !status.InstallSupported {
		t.Fatalf("expected InstallSupported for dir-layout target")
	}
	applied, err := h.Apply(hostreqkit.Host{OS: "linux"}, status, hostreqkit.EnsureOptions{})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if applied.ExecutionState != hostreqkit.ExecutionInstalled {
		t.Fatalf("state = %q (notes %v), want installed", applied.ExecutionState, applied.Notes)
	}
	if gotSpec.Layout != "dir" || gotSpec.BinPath != "sd-cli" {
		t.Fatalf("fetchDir received wrong spec: %+v", gotSpec)
	}
	if gotOptDir != optRoot+"/sd" {
		t.Fatalf("opt dir = %q, want %q", gotOptDir, optRoot+"/sd")
	}
	launcher := binDir + "/sd"
	data, err := os.ReadFile(launcher)
	if err != nil {
		t.Fatalf("launcher not written: %v", err)
	}
	script := string(data)
	if !strings.HasPrefix(script, "#!/bin/sh") {
		t.Fatalf("launcher missing shebang: %q", script)
	}
	if !strings.Contains(script, "LD_LIBRARY_PATH") {
		t.Fatalf("launcher must export LD_LIBRARY_PATH: %q", script)
	}
	if !strings.Contains(script, optRoot+"/sd") {
		t.Fatalf("launcher must exec the opt-dir binary: %q", script)
	}
	info, _ := os.Stat(launcher)
	if info.Mode().Perm()&0o100 == 0 {
		t.Fatalf("launcher not executable: %v", info.Mode())
	}
}

func TestApplyFetch_DirLayoutDryRunNotesOptDir(t *testing.T) {
	withFacts(t, hostreqspec.CapabilityFacts{Arch: "amd64"})
	withArch(t, "amd64")
	withUserBin(t)
	withUserOpt(t)
	h := toolHandler{manifest: dirReleaseManifest("sd")}
	status := h.Inspect(hostreqkit.Host{OS: "linux"}, resolved("sd"))
	applied, err := h.Apply(hostreqkit.Host{OS: "linux"}, status, hostreqkit.EnsureOptions{DryRun: true})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if applied.ExecutionState != hostreqkit.ExecutionWouldInstall {
		t.Fatalf("state = %q, want would_install", applied.ExecutionState)
	}
	if !notesContain(applied.Notes, "~/.vrooli/opt/sd") {
		t.Fatalf("dry-run notes should mention the opt dir: %v", applied.Notes)
	}
}

func TestInspectFetch_DirLayoutInstalledWhenLauncherPresent(t *testing.T) {
	withFacts(t, hostreqspec.CapabilityFacts{Arch: "amd64"})
	withArch(t, "amd64")
	binDir := withUserBin(t)
	withUserOpt(t)
	// The launcher's mere presence in ~/.vrooli/bin counts as installed.
	if err := writeFile(binDir+"/sd", []byte("#!/bin/sh\nexec opt/sd/sd-cli \"$@\"\n")); err != nil {
		t.Fatal(err)
	}
	h := toolHandler{manifest: dirReleaseManifest("sd")}
	status := h.Inspect(hostreqkit.Host{OS: "linux"}, resolved("sd"))
	if status.ExecutionState != hostreqkit.ExecutionAlreadyPresent {
		t.Fatalf("state = %q, want already_present", status.ExecutionState)
	}
}

func notesContain(notes []string, substr string) bool {
	for _, n := range notes {
		if strings.Contains(n, substr) {
			return true
		}
	}
	return false
}
