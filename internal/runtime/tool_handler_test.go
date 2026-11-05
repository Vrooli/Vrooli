//nolint:goconst // test data deliberately reuses stable version fixtures.
package runtime

import (
	"context"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/binaryfetch"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
	"github.com/vrooli/vrooli/internal/shell/shelltest"
	"github.com/vrooli/vrooli/internal/tools"
)

const fetchTestTool = "vrooli-runtime-fetch-test-tool"

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
		Acquisition: &hostreqkit.ToolSource{
			Kind:    "url",
			Targets: []hostreqkit.ToolSourceTarget{{When: map[string]string{"os": "linux", "arch": "amd64"}, URL: "https://example.test/" + name + ".tar.gz", SHA256: "deadbeef", Archive: "tar.gz", BinPath: "bin/" + name}},
		},
	}
}

func hybridReleaseManifest(name string) hostreqkit.ToolManifest {
	manifest := releaseManifest(name, nil)
	manifest.Packages = map[string]string{
		"apt":    "distro-" + name,
		"brew":   "homebrew-" + name,
		"winget": "winget-" + name,
	}
	return manifest
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

func withoutCommandOnPath(t *testing.T) {
	t.Helper()
	previous := hostreqkit.LookPathFn
	hostreqkit.LookPathFn = func(string) (string, error) { return "", os.ErrNotExist }
	t.Cleanup(func() { hostreqkit.LookPathFn = previous })
}

func bufManifest(t *testing.T) hostreqkit.ToolManifest {
	t.Helper()
	data, err := fs.ReadFile(tools.Manifests, "buf/tool.json")
	if err != nil {
		t.Fatalf("read buf manifest: %v", err)
	}
	var manifest hostreqkit.ToolManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parse buf manifest: %v", err)
	}
	return manifest
}

func TestBufManifestUsesVerifiedGenericReleaseTargets(t *testing.T) {
	manifest := bufManifest(t)
	if manifest.Handler != "" {
		t.Fatalf("buf handler = %q; generic runtime handler must own URL acquisition", manifest.Handler)
	}
	if manifest.Acquisition == nil || manifest.Acquisition.Kind != "url" {
		t.Fatalf("buf acquisition = %#v; want verified URL source", manifest.Acquisition)
	}
	for target, wantURLFragment := range map[string]string{
		"linux/amd64":  "buf-Linux-x86_64",
		"linux/arm64":  "buf-Linux-aarch64",
		"darwin/amd64": "buf-Darwin-x86_64",
		"darwin/arm64": "buf-Darwin-arm64",
	} {
		parts := strings.SplitN(target, "/", 2)
		spec, ok := hostreqkit.TargetFor(manifest.Acquisition, parts[0], parts[1])
		if !ok {
			t.Errorf("missing required Buf acquisition target %q", target)
			continue
		}
		if !strings.Contains(spec.URL, wantURLFragment) || len(spec.SHA256) != 64 {
			t.Errorf("target %s = %#v; want URL and SHA-256", target, spec)
		}
	}
	if _, ok := hostreqkit.TargetFor(manifest.Acquisition, "darwin", "386"); ok {
		t.Fatal("unsupported Darwin architecture must not silently select an acquisition target")
	}
}

func TestBufGenericInstallerUsesInvokingUserBinAndConverges(t *testing.T) {
	withFacts(t, hostreqspec.CapabilityFacts{Arch: "arm64"})
	withArch(t, "arm64")
	binDir := withUserBin(t)
	withoutCommandOnPath(t)
	manifest := bufManifest(t)
	h := toolHandler{manifest: manifest}
	host := hostreqkit.Host{OS: "darwin", PackageManager: "brew"}
	status := h.Inspect(host, resolved("buf"))
	if !status.InstallSupported || !notesContain(status.Notes, "~/.vrooli/bin") {
		t.Fatalf("Buf status = %#v; expected user-local generic install", status)
	}
	prevFetch := fetchBinaryFn
	fetchBinaryFn = func(_ context.Context, spec binaryfetch.Target, dest string, _ binaryfetch.ProgressFunc) (string, error) {
		if spec.SHA256 == "" || dest != binDir {
			t.Fatalf("fetch spec/destination = %#v, %q", spec, dest)
		}
		path := dest + "/" + spec.Name
		return path, writeFile(path, []byte("buf"))
	}
	t.Cleanup(func() { fetchBinaryFn = prevFetch })
	applied, err := h.Apply(host, status, hostreqkit.EnsureOptions{})
	if err != nil || applied.ExecutionState != hostreqkit.ExecutionInstalled {
		t.Fatalf("first apply = %#v, %v", applied, err)
	}
	repeated, err := h.Apply(host, h.Inspect(host, resolved("buf")), hostreqkit.EnsureOptions{})
	if err != nil || repeated.ExecutionState != hostreqkit.ExecutionAlreadyPresent {
		t.Fatalf("repeat apply = %#v, %v; want convergence", repeated, err)
	}
}

func TestBufGenericInstallerRejectsUndeclaredArchitecture(t *testing.T) {
	withFacts(t, hostreqspec.CapabilityFacts{Arch: "386"})
	withArch(t, "386")
	withUserBin(t)
	withoutCommandOnPath(t)
	status := toolHandler{manifest: bufManifest(t)}.Inspect(hostreqkit.Host{OS: "darwin", PackageManager: "brew"}, resolved("buf"))
	if status.ExecutionState != hostreqkit.ExecutionUnsupported || status.InstallSupported {
		t.Fatalf("status = %#v; undeclared Darwin architecture must be unsupported", status)
	}
}

func TestInspectFetch_GPURequiredSkippedOnCPUHost(t *testing.T) {
	withFacts(t, hostreqspec.CapabilityFacts{Arch: "amd64", RAMGb: 32}) // no GPU
	h := toolHandler{manifest: releaseManifest(fetchTestTool, &hostreqspec.CapabilityRequirement{GPU: boolPtr(true)})}
	status := h.Inspect(hostreqkit.Host{OS: "linux"}, resolved(fetchTestTool))
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
	h := toolHandler{manifest: releaseManifest("source-only-no-target-test-command", nil)}
	status := h.Inspect(hostreqkit.Host{OS: "linux"}, resolved("source-only-no-target-test-command"))
	if status.ExecutionState != hostreqkit.ExecutionUnsupported {
		t.Fatalf("state = %q, want unsupported", status.ExecutionState)
	}
}

func TestInspectHybrid_MatchingReleaseTargetPreferredOverPackage(t *testing.T) {
	withFacts(t, hostreqspec.CapabilityFacts{Arch: "amd64"})
	withArch(t, "amd64")
	withUserBin(t)
	h := toolHandler{manifest: hybridReleaseManifest("hybrid-target-preference")}

	status := h.Inspect(hostreqkit.Host{OS: "linux", PackageManager: "apt"}, resolved("hybrid-target-preference"))

	if !status.InstallSupported {
		t.Fatal("expected matching release target to be installable")
	}
	if status.PackageName != "" {
		t.Fatalf("package name = %q, want empty when release target is preferred", status.PackageName)
	}
	if !notesContain(status.Notes, "example.test/hybrid-target-preference.tar.gz") {
		t.Fatalf("release target URL missing from notes: %v", status.Notes)
	}
}

func TestInspectHybrid_MissingReleaseTargetFallsBackToHostPackage(t *testing.T) {
	withFacts(t, hostreqspec.CapabilityFacts{Arch: "arm64"})
	withArch(t, "arm64")
	withUserBin(t)
	h := toolHandler{manifest: hybridReleaseManifest("hybrid-package-fallback")}

	cases := []struct {
		name        string
		host        hostreqkit.Host
		wantPackage string
	}{
		{name: "macos", host: hostreqkit.Host{OS: "darwin", PackageManager: "brew"}, wantPackage: "homebrew-hybrid-package-fallback"},
		{name: "windows", host: hostreqkit.Host{OS: "windows", PackageManager: "winget"}, wantPackage: "winget-hybrid-package-fallback"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status := h.Inspect(tc.host, resolved("hybrid-package-fallback"))
			if status.ExecutionState == hostreqkit.ExecutionUnsupported {
				t.Fatalf("state = %q, want package fallback to remain supported", status.ExecutionState)
			}
			if !status.InstallSupported {
				t.Fatal("expected package fallback to be installable")
			}
			if status.PackageName != tc.wantPackage {
				t.Fatalf("package name = %q, want %q", status.PackageName, tc.wantPackage)
			}
		})
	}
}

func TestApplyHybrid_MissingReleaseTargetUsesHostPackage(t *testing.T) {
	withFacts(t, hostreqspec.CapabilityFacts{Arch: "arm64"})
	withArch(t, "arm64")
	withUserBin(t)
	h := toolHandler{manifest: hybridReleaseManifest("hybrid-package-apply")}
	host := hostreqkit.Host{OS: "darwin", PackageManager: "brew"}
	status := h.Inspect(host, resolved("hybrid-package-apply"))

	applied, err := h.Apply(host, status, hostreqkit.EnsureOptions{DryRun: true})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if applied.ExecutionState != hostreqkit.ExecutionWouldInstall {
		t.Fatalf("state = %q (notes %v), want would_install", applied.ExecutionState, applied.Notes)
	}
	if !notesContain(applied.Notes, "brew install homebrew-hybrid-package-apply") {
		t.Fatalf("package install command missing from notes: %v", applied.Notes)
	}
}

func TestApplyFetch_DryRunListsTargetURL(t *testing.T) {
	withFacts(t, hostreqspec.CapabilityFacts{Arch: "amd64"})
	withArch(t, "amd64")
	withUserBin(t)
	h := toolHandler{manifest: releaseManifest(fetchTestTool, nil)}
	status := h.Inspect(hostreqkit.Host{OS: "linux"}, resolved(fetchTestTool))
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
	if !notesContain(applied.Notes, "example.test/"+fetchTestTool+".tar.gz") {
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
		if err := writeFile(path, []byte(shelltest.POSIXShebang()+"echo ok\n")); err != nil {
			return "", err
		}
		return path, nil
	}
	t.Cleanup(func() { fetchBinaryFn = prevFetch })

	h := toolHandler{manifest: releaseManifest(fetchTestTool, nil)}
	status := h.Inspect(hostreqkit.Host{OS: "linux"}, resolved(fetchTestTool))
	applied, err := h.Apply(hostreqkit.Host{OS: "linux"}, status, hostreqkit.EnsureOptions{})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if applied.ExecutionState != hostreqkit.ExecutionInstalled {
		t.Fatalf("state = %q (notes %v), want installed", applied.ExecutionState, applied.Notes)
	}
	if fetched.URL != "https://example.test/"+fetchTestTool+".tar.gz" || fetched.SHA256 != "deadbeef" {
		t.Fatalf("fetcher received wrong spec: %+v", fetched)
	}
	if fetched.BinPath != "bin/"+fetchTestTool || fetched.Archive != "tar.gz" {
		t.Fatalf("fetcher spec archive/binpath wrong: %+v", fetched)
	}
	if !fileExists(binDir + "/" + fetchTestTool) {
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

	h := toolHandler{manifest: releaseManifest(fetchTestTool, nil)}
	status := h.Inspect(hostreqkit.Host{OS: "linux"}, resolved(fetchTestTool))
	applied, _ := h.Apply(hostreqkit.Host{OS: "linux"}, status, hostreqkit.EnsureOptions{})
	if applied.ExecutionState != hostreqkit.ExecutionFailed {
		t.Fatalf("state = %q, want failed", applied.ExecutionState)
	}
}

func TestInspectFetch_AlreadyPresentWhenInUserBin(t *testing.T) {
	withFacts(t, hostreqspec.CapabilityFacts{Arch: "amd64"})
	withArch(t, "amd64")
	binDir := withUserBin(t)
	if err := writeFile(binDir+"/"+fetchTestTool, []byte("bin")); err != nil {
		t.Fatal(err)
	}
	h := toolHandler{manifest: releaseManifest(fetchTestTool, nil)}
	status := h.Inspect(hostreqkit.Host{OS: "linux"}, resolved(fetchTestTool))
	if status.ExecutionState != hostreqkit.ExecutionAlreadyPresent {
		t.Fatalf("state = %q, want already_present", status.ExecutionState)
	}
}

func TestCapabilityGate_MetProceeds(t *testing.T) {
	withFacts(t, hostreqspec.CapabilityFacts{HasGPU: true, MaxVRAMGb: 8, Arch: "amd64", RAMGb: 32})
	withArch(t, "amd64")
	withUserBin(t)
	h := toolHandler{manifest: releaseManifest(fetchTestTool, &hostreqspec.CapabilityRequirement{GPU: boolPtr(true), MinVRAMGb: 4})}
	status := h.Inspect(hostreqkit.Host{OS: "linux"}, resolved(fetchTestTool))
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
		Acquisition: &hostreqkit.ToolSource{
			Kind:    "url",
			Targets: []hostreqkit.ToolSourceTarget{{When: map[string]string{"os": "linux", "arch": "amd64"}, URL: "https://example.test/" + name + ".zip", SHA256: "deadbeef", Archive: "zip", Layout: "dir", BinPath: name + "-cli"}},
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
		if err := writeFile(entry, []byte(shelltest.POSIXShebang()+"echo ok\n")); err != nil {
			return "", err
		}
		return entry, nil
	}
	t.Cleanup(func() { fetchDirFn = prev })

	h := toolHandler{manifest: dirReleaseManifest(fetchTestTool)}
	status := h.Inspect(hostreqkit.Host{OS: "linux"}, resolved(fetchTestTool))
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
	if gotSpec.Layout != "dir" || gotSpec.BinPath != fetchTestTool+"-cli" {
		t.Fatalf("fetchDir received wrong spec: %+v", gotSpec)
	}
	if gotOptDir != optRoot+"/"+fetchTestTool {
		t.Fatalf("opt dir = %q, want %q", gotOptDir, optRoot+"/"+fetchTestTool)
	}
	launcher := binDir + "/" + fetchTestTool
	data, err := os.ReadFile(launcher)
	if err != nil {
		t.Fatalf("launcher not written: %v", err)
	}
	script := string(data)
	if !strings.HasPrefix(script, shelltest.POSIXShebang()) {
		t.Fatalf("launcher missing shebang: %q", script)
	}
	if !strings.Contains(script, "LD_LIBRARY_PATH") {
		t.Fatalf("launcher must export LD_LIBRARY_PATH: %q", script)
	}
	if !strings.Contains(script, optRoot+"/"+fetchTestTool) {
		t.Fatalf("launcher must exec the opt-dir binary: %q", script)
	}
	info, _ := os.Stat(launcher)
	if info.Mode().Perm()&0o100 == 0 {
		t.Fatalf("launcher not executable: %v", info.Mode())
	}
}

func TestWriteLauncherResolvesRuntimeEnvironmentWithinToolDirectory(t *testing.T) {
	binDir := t.TempDir()
	optDir := t.TempDir()
	entry := filepath.Join(optDir, "tool", "bin", fetchTestTool)
	if err := os.MkdirAll(filepath.Dir(entry), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeLauncher(binDir, fetchTestTool, optDir, entry, map[string]string{"GOROOT": "tool"}); err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(filepath.Join(binDir, fetchTestTool))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(contents), "export GOROOT='"+filepath.Join(optDir, "tool")+"'") {
		t.Fatalf("launcher runtime environment = %q", contents)
	}
	if err := writeLauncher(binDir, fetchTestTool, optDir, entry, map[string]string{"GOROOT": "../outside"}); err == nil {
		t.Fatal("expected launcher runtime environment traversal to be rejected")
	}
}

func TestApplyFetch_DirLayoutDryRunNotesOptDir(t *testing.T) {
	withFacts(t, hostreqspec.CapabilityFacts{Arch: "amd64"})
	withArch(t, "amd64")
	withUserBin(t)
	withUserOpt(t)
	h := toolHandler{manifest: dirReleaseManifest(fetchTestTool)}
	status := h.Inspect(hostreqkit.Host{OS: "linux"}, resolved(fetchTestTool))
	applied, err := h.Apply(hostreqkit.Host{OS: "linux"}, status, hostreqkit.EnsureOptions{DryRun: true})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if applied.ExecutionState != hostreqkit.ExecutionWouldInstall {
		t.Fatalf("state = %q, want would_install", applied.ExecutionState)
	}
	if !notesContain(applied.Notes, "~/.vrooli/opt/"+fetchTestTool) {
		t.Fatalf("dry-run notes should mention the opt dir: %v", applied.Notes)
	}
}

// A pinned tool whose version probe cannot execute — the Go toolchain refuses to
// start when the working directory is unreadable, for instance — must not be
// reported as a version mismatch. The observed version is unknown, not wrong,
// and re-fetching the identical working release cannot change the outcome.
func TestFetchToolUnrunnableVersionProbeIsNotAMismatch(t *testing.T) {
	withFacts(t, hostreqspec.CapabilityFacts{Arch: "amd64"})
	withArch(t, "amd64")
	binDir := withUserBin(t)
	withoutCommandOnPath(t)

	manifest := dirReleaseManifest(fetchTestTool)
	manifest.Version = "1.25.12"
	h := toolHandler{manifest: manifest}

	const probeFailure = "go: cannot determine current directory: stat .: permission denied"
	if err := writeFile(binDir+"/"+fetchTestTool, []byte(shelltest.POSIXShebang()+"echo '"+probeFailure+"' >&2\nexit 1\n")); err != nil {
		t.Fatal(err)
	}

	status := h.Inspect(hostreqkit.Host{OS: "linux"}, resolved(fetchTestTool))

	if status.Installed {
		t.Fatalf("status = %+v, want an unverifiable pin to stay unsatisfied", status)
	}
	if status.InstallSupported {
		t.Fatalf("status = %+v, want no install remediation for a probe the environment broke", status)
	}
	if status.BlockingReason != hostreqkit.BlockingProbeFailed {
		t.Fatalf("blocking reason = %q, want %q", status.BlockingReason, hostreqkit.BlockingProbeFailed)
	}
	if notesContain(status.Notes, "version mismatch") {
		t.Fatalf("status notes = %v, want no version-mismatch claim when nothing was observed", status.Notes)
	}
	if !notesContain(status.Notes, probeFailure) {
		t.Fatalf("status notes = %v, want the underlying probe failure surfaced", status.Notes)
	}
}

func TestFetchToolExactVersionMismatchIsRepairable(t *testing.T) {
	withFacts(t, hostreqspec.CapabilityFacts{Arch: "amd64"})
	withArch(t, "amd64")
	binDir := withUserBin(t)
	withoutCommandOnPath(t)

	manifest := dirReleaseManifest(fetchTestTool)
	manifest.Version = "1.25.12"
	h := toolHandler{manifest: manifest}
	host := hostreqkit.Host{OS: "linux"}

	// A stale release on the managed path must be treated as not ready, but
	// still repairable by the verified release target.
	if err := writeFile(binDir+"/"+fetchTestTool, []byte(shelltest.POSIXShebang()+"echo go version go1.25.0 linux/amd64\n")); err != nil {
		t.Fatal(err)
	}
	status := h.Inspect(host, resolved(fetchTestTool))
	if status.Installed || !status.InstallSupported || status.ExecutionState != hostreqkit.ExecutionPending {
		t.Fatalf("stale pinned tool status = %+v, want repairable pending status", status)
	}
	if !notesContain(status.Notes, "version mismatch") {
		t.Fatalf("status notes = %v, want version mismatch", status.Notes)
	}

	prev := fetchDirFn
	fetchDirFn = func(_ context.Context, spec binaryfetch.Target, optDir string, _ binaryfetch.ProgressFunc) (string, error) {
		entry := optDir + "/" + spec.BinPath
		if err := os.MkdirAll(filepath.Dir(entry), 0o755); err != nil {
			return "", err
		}
		if err := writeFile(entry, []byte(shelltest.POSIXShebang()+"echo go version go1.25.12 linux/amd64\n")); err != nil {
			return "", err
		}
		return entry, nil
	}
	t.Cleanup(func() { fetchDirFn = prev })
	withUserOpt(t)

	applied, err := h.Apply(host, status, hostreqkit.EnsureOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !applied.Installed || applied.ExecutionState != hostreqkit.ExecutionInstalled {
		t.Fatalf("repaired pinned tool status = %+v", applied)
	}
}

func TestFetchToolRepairsLauncherMissingDeclaredRuntimeEnvironment(t *testing.T) {
	withFacts(t, hostreqspec.CapabilityFacts{Arch: "amd64"})
	withArch(t, "amd64")
	binDir := withUserBin(t)
	optRoot := withUserOpt(t)
	withoutCommandOnPath(t)

	manifest := dirReleaseManifest(fetchTestTool)
	manifest.Version = "1.25.12"
	manifest.Acquisition.Targets[0] = hostreqkit.ToolSourceTarget{
		URL: "https://example.test/tool.zip", SHA256: "deadbeef", Archive: "zip", Layout: "dir", BinPath: "go/bin/go", RuntimeEnv: map[string]string{"GOROOT": "go"},
	}
	entry := filepath.Join(optRoot, fetchTestTool, "go/bin/go")
	if err := os.MkdirAll(filepath.Dir(entry), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeFile(entry, []byte(shelltest.POSIXShebang()+"echo go version go1.25.12 linux/amd64\n")); err != nil {
		t.Fatal(err)
	}
	if err := writeFile(filepath.Join(binDir, fetchTestTool), []byte(shelltest.POSIXShebang()+"exec '"+entry+"' \"$@\"\n")); err != nil {
		t.Fatal(err)
	}

	h := toolHandler{manifest: manifest}
	status := h.Inspect(hostreqkit.Host{OS: "linux"}, resolved(fetchTestTool))
	if status.Installed || !status.InstallSupported || !notesContain(status.Notes, "does not set required runtime environment GOROOT") {
		t.Fatalf("legacy launcher status = %+v, want a repairable runtime-environment mismatch", status)
	}
	if optRoot == "" {
		t.Fatal("temporary opt root must be populated")
	}
}

func TestInspectFetch_DirLayoutInstalledWhenLauncherPresent(t *testing.T) {
	withFacts(t, hostreqspec.CapabilityFacts{Arch: "amd64"})
	withArch(t, "amd64")
	binDir := withUserBin(t)
	optRoot := withUserOpt(t)
	entry := filepath.Join(optRoot, fetchTestTool, fetchTestTool+"-cli")
	if err := os.MkdirAll(filepath.Dir(entry), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeFile(entry, []byte(shelltest.POSIXShebang()+"exit 0\n")); err != nil {
		t.Fatal(err)
	}
	if err := writeFile(binDir+"/"+fetchTestTool, []byte(shelltest.POSIXShebang()+"exec '"+entry+"' \"$@\"\n")); err != nil {
		t.Fatal(err)
	}
	h := toolHandler{manifest: dirReleaseManifest(fetchTestTool)}
	status := h.Inspect(hostreqkit.Host{OS: "linux"}, resolved(fetchTestTool))
	if status.ExecutionState != hostreqkit.ExecutionAlreadyPresent {
		t.Fatalf("state = %q, want already_present", status.ExecutionState)
	}
}

func TestInspectFetch_DirLayoutRejectsDanglingLauncher(t *testing.T) {
	withFacts(t, hostreqspec.CapabilityFacts{Arch: "amd64"})
	withArch(t, "amd64")
	binDir := withUserBin(t)
	withUserOpt(t)
	if err := writeFile(binDir+"/"+fetchTestTool, []byte(shelltest.POSIXShebang()+"exec /missing/payload \"$@\"\n")); err != nil {
		t.Fatal(err)
	}
	h := toolHandler{manifest: dirReleaseManifest(fetchTestTool)}
	status := h.Inspect(hostreqkit.Host{OS: "linux"}, resolved(fetchTestTool))
	if status.ExecutionState != hostreqkit.ExecutionPending || status.Installed {
		t.Fatalf("state = %#v, want repairable pending status", status)
	}
	if !notesContain(status.Notes, "managed tool payload is missing or not executable") {
		t.Fatalf("status notes = %v", status.Notes)
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
