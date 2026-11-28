package installgateway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

const (
	testWorkspaceYAML = "packages:\n  - .\nminimumReleaseAge: 10080\n"
	testBaseLock      = "lockfileVersion: '9.0'\nimporters:\n  .:\n    dependencies:\n      alpha:\n        specifier: 1.0.0\n        version: 1.0.0\npackages:\n  alpha@1.0.0:\n    resolution: {integrity: sha512-AAAA}\nsnapshots:\n  alpha@1.0.0: {}\n"
	testNextLock      = "lockfileVersion: '9.0'\nimporters:\n  .:\n    dependencies:\n      alpha:\n        specifier: 1.0.0\n        version: 1.0.0\n      beta:\n        specifier: 2.0.0\n        version: 2.0.0\npackages:\n  alpha@1.0.0:\n    resolution: {integrity: sha512-AAAA}\n  beta@2.0.0:\n    resolution: {integrity: sha512-BBBB}\nsnapshots:\n  alpha@1.0.0: {}\n  beta@2.0.0: {}\n"
	testBaseManifest  = `{"name":"demo-ui","packageManager":"pnpm@10.34.9","dependencies":{"alpha":"1.0.0"}}`
	testNextManifest  = `{"name":"demo-ui","packageManager":"pnpm@10.34.9","dependencies":{"alpha":"1.0.0","beta":"2.0.0"}}`
)

// lockPlus returns testBaseLock with extra package entries inside packages:.
func lockPlus(extra string) string {
	return strings.Replace(testBaseLock, "snapshots:", extra+"snapshots:", 1)
}

type pnpmCall struct {
	Name string
	Kind string
	Dir  string
	Args []string
	Env  []string
}

// fakePnpm replaces the pnpm process. Its add handler plays the resolver by
// rewriting the staged inputs; install handlers play the link step.
type fakePnpm struct {
	mu       sync.Mutex
	store    string
	calls    []pnpmCall
	add      func(call pnpmCall) ([]byte, error)
	update   func(call pnpmCall) ([]byte, error)
	install  func(call pnpmCall, attempt int) ([]byte, error)
	installs int
}

func useFakePnpm(t *testing.T, fake *fakePnpm) {
	t.Helper()
	if fake.store == "" {
		fake.store = filepath.Join(t.TempDir(), "store")
	}
	previous := runPackageManager
	runPackageManager = func(_ context.Context, name, dir string, env []string, args ...string) ([]byte, error) {
		kind := args[0]
		if name == "npm" && containsArg(args, "--package-lock-only") {
			kind = "add" // npm resolves with `install|uninstall --package-lock-only`
		}
		if kind == "remove" {
			kind = "add" // the fake resolver rewrites the staged inputs for add and remove alike
		}
		call := pnpmCall{Name: name, Kind: kind, Dir: dir, Args: append([]string(nil), args...), Env: env}
		fake.mu.Lock()
		fake.calls = append(fake.calls, call)
		fake.mu.Unlock()
		switch kind {
		case "store":
			base := fake.store
			for _, item := range env {
				if value, ok := strings.CutPrefix(item, "npm_config_store_dir="); ok {
					base = value // pnpm honours an explicit store directory
				}
			}
			return []byte(filepath.Join(base, "v10") + "\n"), nil
		case "add":
			if fake.add == nil {
				return nil, fmt.Errorf("unexpected pnpm add")
			}
			return fake.add(call)
		case "update":
			if fake.update == nil {
				return nil, fmt.Errorf("unexpected pnpm update")
			}
			return fake.update(call)
		case "install":
			fake.mu.Lock()
			fake.installs++
			attempt := fake.installs
			fake.mu.Unlock()
			if fake.install == nil {
				return []byte("installed"), nil
			}
			return fake.install(call, attempt)
		}
		return nil, fmt.Errorf("unexpected pnpm %v", args)
	}
	t.Cleanup(func() { runPackageManager = previous })
}

func (f *fakePnpm) commands(name string) []pnpmCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []pnpmCall
	for _, call := range f.calls {
		if call.Kind == name {
			out = append(out, call)
		}
	}
	return out
}

// resolvesTo makes the fake resolver rewrite the staged manifest and lockfile.
func resolvesTo(manifest, lock string) func(pnpmCall) ([]byte, error) {
	return func(call pnpmCall) ([]byte, error) {
		if err := os.WriteFile(filepath.Join(call.Dir, "package.json"), []byte(manifest), 0o600); err != nil {
			return nil, err
		}
		return []byte("resolved"), os.WriteFile(filepath.Join(call.Dir, "pnpm-lock.yaml"), []byte(lock), 0o600)
	}
}

type fakeSizer struct {
	mu       sync.Mutex
	registry map[string]artifactSize
	tarballs map[string]artifactSize
	asked    []string
}

func useFakeSizer(t *testing.T, sizer *fakeSizer) {
	t.Helper()
	previous := newRegistrySizer
	newRegistrySizer = func() (registrySizer, error) { return sizer, nil }
	t.Cleanup(func() { newRegistrySizer = previous })
}

func (f *fakeSizer) Registry(_ context.Context, name, version, _ string) (artifactSize, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.asked = append(f.asked, name+"@"+version)
	size, ok := f.registry[name+"@"+version]
	if !ok {
		return artifactSize{}, fmt.Errorf("no fake size for %s@%s", name, version)
	}
	return size, nil
}

func (f *fakeSizer) Tarball(_ context.Context, rawURL, integrity string) (artifactSize, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.asked = append(f.asked, rawURL+"#"+integrity)
	size, ok := f.tarballs[rawURL]
	if !ok {
		return artifactSize{}, fmt.Errorf("no fake tarball size for %s", rawURL)
	}
	return size, nil
}

type pnpmFixture struct {
	repo, surface string
	resolution    Resolution
}

func newPnpmFixture(t *testing.T, manifest, lock, packageName, spec string) pnpmFixture {
	t.Helper()
	repo := t.TempDir()
	surface := filepath.Join(repo, "scenarios", "demo", "ui")
	if err := os.MkdirAll(surface, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(repo, "packages"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{"package.json": manifest, "pnpm-lock.yaml": lock, "pnpm-workspace.yaml": testWorkspaceYAML} {
		if err := os.WriteFile(filepath.Join(surface, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return pnpmFixture{repo: repo, surface: surface, resolution: Resolution{
		RepositoryRoot: repo, SurfaceRoot: surface, ManifestPath: filepath.Join(surface, "package.json"),
		PackageManager: "pnpm", PackageName: packageName,
		Argv:    []string{"pnpm", "add", "--ignore-scripts", "--workspace-root", packageName + "@" + spec},
		Profile: InstallProfile{LifecycleMode: "scripts-disabled-by-default"},
	}}
}

// installedLock marks node_modules as installed from exactly lock.
func (f pnpmFixture) installedLock(t *testing.T, lock string, virtualEntries ...string) {
	t.Helper()
	dir := filepath.Join(f.surface, "node_modules", ".pnpm")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "lock.yaml"), []byte(lock), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, entry := range virtualEntries {
		if err := os.MkdirAll(filepath.Join(dir, entry), 0o755); err != nil {
			t.Fatal(err)
		}
	}
}

func (f pnpmFixture) read(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(f.surface, name))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func stageLeftovers(t *testing.T, fixture pnpmFixture) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Dir(fixture.surface))
	if err != nil {
		t.Fatal(err)
	}
	var left []string
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".sda-stage-") {
			left = append(left, entry.Name())
		}
	}
	return left
}

func TestStageSizesOnlyPackagesNodeModulesLacks(t *testing.T) {
	fixture := newPnpmFixture(t, testBaseManifest, testBaseLock, "beta", "2.0.0")
	fixture.installedLock(t, testBaseLock)
	fake := &fakePnpm{add: resolvesTo(testNextManifest, testNextLock)}
	useFakePnpm(t, fake)
	sizer := &fakeSizer{registry: map[string]artifactSize{"beta@2.0.0": {Unpacked: 4000, Archive: 1000}}}
	useFakeSizer(t, sizer)

	stage, err := stageJSResolution(context.Background(), fixture.resolution)
	if err != nil {
		t.Fatal(err)
	}
	defer stage.Release()
	if strings.Join(sizer.asked, ",") != "beta@2.0.0" {
		t.Fatalf("sized %v; only the package the lock adds should be sized", sizer.asked)
	}
	if stage.Measured.PackageCount != 1 || stage.Measured.PackageBytes != 5000 || stage.Measured.ProbeBytes <= 5000 {
		t.Fatalf("measurement = %+v", stage.Measured)
	}
	if fixture.read(t, "package.json") != testBaseManifest || fixture.read(t, "pnpm-lock.yaml") != testBaseLock {
		t.Fatal("staging changed the surface")
	}
	adds := fake.commands("add")
	if len(adds) != 1 || !strings.Contains(strings.Join(adds[0].Args, " "), "add --lockfile-only --ignore-scripts --workspace-root beta@2.0.0") {
		t.Fatalf("resolver calls = %+v", adds)
	}
	if adds[0].Dir == fixture.surface || filepath.Dir(adds[0].Dir) != filepath.Dir(fixture.surface) {
		t.Fatalf("resolver ran in %s; it must run in a sibling of %s so relative file: paths keep their depth", adds[0].Dir, fixture.surface)
	}
}

func TestStageSizesWholeLockWhenNodeModulesIsAbsent(t *testing.T) {
	fixture := newPnpmFixture(t, testBaseManifest, testBaseLock, "beta", "2.0.0")
	useFakePnpm(t, &fakePnpm{add: resolvesTo(testNextManifest, testNextLock)})
	sizer := &fakeSizer{registry: map[string]artifactSize{
		"alpha@1.0.0": {Unpacked: 100, Archive: 10}, "beta@2.0.0": {Unpacked: 4000, Archive: 1000},
	}}
	useFakeSizer(t, sizer)
	stage, err := stageJSResolution(context.Background(), fixture.resolution)
	if err != nil {
		t.Fatal(err)
	}
	defer stage.Release()
	if stage.Measured.PackageCount != 2 || stage.Measured.PackageBytes != 5110 {
		t.Fatalf("a surface without node_modules must be sized for the whole lock: %+v", stage.Measured)
	}
}

func TestStageSizesDriftedNodeModulesByWhatTheVirtualStoreLacks(t *testing.T) {
	fixture := newPnpmFixture(t, testBaseManifest, testBaseLock, "beta", "2.0.0")
	fixture.installedLock(t, "lockfileVersion: '9.0'\n# drifted\n", "alpha@1.0.0")
	useFakePnpm(t, &fakePnpm{add: resolvesTo(testNextManifest, testNextLock)})
	sizer := &fakeSizer{registry: map[string]artifactSize{"beta@2.0.0": {Unpacked: 4000, Archive: 1000}}}
	useFakeSizer(t, sizer)
	stage, err := stageJSResolution(context.Background(), fixture.resolution)
	if err != nil {
		t.Fatal(err)
	}
	defer stage.Release()
	if strings.Join(sizer.asked, ",") != "beta@2.0.0" || stage.Measured.PackageCount != 1 {
		t.Fatalf("drift sizing asked %v, measured %+v; alpha is already in the virtual store", sizer.asked, stage.Measured)
	}
}

func localPackageFixture(t *testing.T, repo, name, packageName string, files map[string]string) {
	t.Helper()
	root := filepath.Join(repo, "packages", name)
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	files["package.json"] = fmt.Sprintf(`{"name":%q,"version":"1.0.0"}`, packageName)
	for file, body := range files {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(root, file)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, file), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestStageMeasuresSharedLocalPackageFromItsOwnSourceNotItsDependencyClosure(t *testing.T) {
	spec := "file:../../../packages/viewer"
	nextManifest := `{"name":"demo-ui","packageManager":"pnpm@10.34.9","dependencies":{"alpha":"1.0.0","@vrooli/viewer":"` + spec + `"}}`
	// The shared package depends on a large closure that the surface already
	// holds, so the lock adds only the package itself.
	nextLock := lockPlus("  '@vrooli/viewer@" + spec + "':\n    resolution: {directory: ../../../packages/viewer, type: directory}\n")
	fixture := newPnpmFixture(t, testBaseManifest, testBaseLock, "@vrooli/viewer", spec)
	fixture.installedLock(t, testBaseLock)
	localPackageFixture(t, fixture.repo, "viewer", "@vrooli/viewer", map[string]string{"src/index.js": strings.Repeat("x", 3000), "node_modules/huge/index.js": strings.Repeat("y", 1<<20)})
	useFakePnpm(t, &fakePnpm{add: resolvesTo(nextManifest, nextLock)})
	sizer := &fakeSizer{}
	useFakeSizer(t, sizer)
	if err := ApplyExactVersion(&fixture.resolution, spec); err != nil {
		t.Fatal(err)
	}
	stage, err := stageJSResolution(context.Background(), fixture.resolution)
	if err != nil {
		t.Fatal(err)
	}
	defer stage.Release()
	if len(sizer.asked) != 0 {
		t.Fatalf("registry was consulted for %v; nothing but a local directory was added", sizer.asked)
	}
	if stage.Measured.PackageCount != 1 || stage.Measured.PackageBytes < 3000 || stage.Measured.PackageBytes > 4000 {
		t.Fatalf("local package measured as %+v; node_modules inside the package must not count", stage.Measured)
	}
}

func TestStageRefusesLocalPackageOutsidePackagesDirectory(t *testing.T) {
	spec := "file:../../../elsewhere"
	nextLock := lockPlus("  '@vrooli/viewer@" + spec + "':\n    resolution: {directory: ../../../elsewhere, type: directory}\n")
	fixture := newPnpmFixture(t, testBaseManifest, testBaseLock, "@vrooli/viewer", spec)
	fixture.installedLock(t, testBaseLock)
	if err := os.MkdirAll(filepath.Join(fixture.repo, "elsewhere"), 0o755); err != nil {
		t.Fatal(err)
	}
	useFakePnpm(t, &fakePnpm{add: resolvesTo(testNextManifest, nextLock)})
	useFakeSizer(t, &fakeSizer{})
	_, err := stageJSResolution(context.Background(), fixture.resolution)
	if err == nil || !strings.Contains(err.Error(), "beneath the repository packages directory") {
		t.Fatalf("local package outside packages/ = %v", err)
	}
	if left := stageLeftovers(t, fixture); len(left) != 0 {
		t.Fatalf("failed staging left %v behind", left)
	}
}

const (
	sheetjsURL = "https://cdn.sheetjs.com/xlsx-0.20.3/xlsx-0.20.3.tgz"
	sheetjsKey = "xlsx@" + sheetjsURL
)

func tarballLock(integrity string) string {
	resolution := "{tarball: " + sheetjsURL + "}"
	if integrity != "" {
		resolution = "{integrity: " + integrity + ", tarball: " + sheetjsURL + "}"
	}
	return lockPlus("  " + sheetjsKey + ":\n    resolution: " + resolution + "\n    version: 0.20.3\n")
}

func TestStageGivesTarballURLDependencyAnIntegrityIdentity(t *testing.T) {
	fixture := newPnpmFixture(t, testBaseManifest, testBaseLock, "xlsx", sheetjsURL)
	fixture.installedLock(t, testBaseLock)
	if err := ApplyExactVersion(&fixture.resolution, sheetjsURL); err != nil || !fixture.resolution.ExactVersion {
		t.Fatalf("a versioned tarball URL must be accepted as exact: exact=%t err=%v", fixture.resolution.ExactVersion, err)
	}
	useFakePnpm(t, &fakePnpm{add: resolvesTo(testNextManifest, tarballLock("sha512-XLSX"))})
	sizer := &fakeSizer{tarballs: map[string]artifactSize{sheetjsURL: {Unpacked: 9000, Archive: 3000}}}
	useFakeSizer(t, sizer)
	stage, err := stageJSResolution(context.Background(), fixture.resolution)
	if err != nil {
		t.Fatal(err)
	}
	defer stage.Release()
	if strings.Join(sizer.asked, ",") != sheetjsURL+"#sha512-XLSX" || stage.Measured.PackageBytes != 12000 {
		t.Fatalf("tarball sized as %v / %+v", sizer.asked, stage.Measured)
	}
	if len(stage.Notes) != 1 || !strings.Contains(stage.Notes[0], "sha512-XLSX") || !strings.Contains(stage.Notes[0], sheetjsURL) {
		t.Fatalf("the reviewer must see the URL and integrity that identify the artifact: %v", stage.Notes)
	}
}

func TestApplyExactVersionRejectsMutableTarballURLs(t *testing.T) {
	for _, spec := range []string{
		"http://cdn.example/x-1.0.0.tgz", "https://cdn.example/x.tgz?token=abc", "https://user:pw@cdn.example/x-1.0.0.tgz",
		"https://cdn.example/x-1.0.0.zip", "https://cdn.example/x-1.0.0.tgz#frag",
	} {
		resolution := Resolution{PackageManager: "pnpm", PackageName: "x"}
		if err := ApplyExactVersion(&resolution, spec); err == nil {
			t.Errorf("mutable tarball spec %q was accepted as exact", spec)
		}
	}
}

func TestStageRefusesTarballURLThatWasNotRequested(t *testing.T) {
	fixture := newPnpmFixture(t, testBaseManifest, testBaseLock, "beta", "2.0.0")
	fixture.installedLock(t, testBaseLock)
	useFakePnpm(t, &fakePnpm{add: resolvesTo(testNextManifest, tarballLock("sha512-XLSX"))})
	useFakeSizer(t, &fakeSizer{})
	_, err := stageJSResolution(context.Background(), fixture.resolution)
	if err == nil || !strings.Contains(err.Error(), "tarball URL dependency") {
		t.Fatalf("a transitive tarball URL must be refused, got %v", err)
	}
}

func TestStageRepairsLockEntryPnpmRefusesForMissingIntegrity(t *testing.T) {
	brokenLock := tarballLock("")
	fixture := newPnpmFixture(t, testBaseManifest, brokenLock, "xlsx", sheetjsURL)
	fixture.installedLock(t, brokenLock)
	var seen []string
	fake := &fakePnpm{}
	fake.add = func(call pnpmCall) ([]byte, error) {
		lock, _ := os.ReadFile(filepath.Join(call.Dir, "pnpm-lock.yaml"))
		seen = append(seen, string(lock))
		if strings.Contains(string(lock), sheetjsKey) {
			// pnpm 10.34 words this with a thin space (U+2009).
			return []byte("ERR_PNPM_MISSING_TARBALL_INTEGRITY  Cannot install package \"" + sheetjsKey + "\": its lockfile entry has no \"integrity\" field"), fmt.Errorf("exit status 1")
		}
		// Real pnpm re-resolves the entry but, on `remove` and in older versions,
		// records no integrity for it; the gateway pins one from the bytes.
		return resolvesTo(testNextManifest, tarballLock(""))(call)
	}
	useFakePnpm(t, fake)
	useFakeSizer(t, &fakeSizer{tarballs: map[string]artifactSize{sheetjsURL: {Unpacked: 10, Archive: 5, Integrity: "sha512-XLSX"}}})
	stage, err := stageJSResolution(context.Background(), fixture.resolution)
	if err != nil {
		t.Fatal(err)
	}
	defer stage.Release()
	if len(seen) != 2 || strings.Contains(seen[1], "xlsx") {
		t.Fatalf("the retry must run on a lock without the refused entry, saw %d lock(s)", len(seen))
	}
	if notes := strings.Join(stage.Notes, "\n"); !strings.Contains(notes, "re-resolved") || !strings.Contains(notes, "pinned to sha512-XLSX") || !strings.Contains(string(stage.next["pnpm-lock.yaml"]), "integrity: sha512-XLSX") {
		t.Fatalf("repair and pin were not recorded or not published: notes=%v", stage.Notes)
	}
	if fixture.read(t, "pnpm-lock.yaml") != brokenLock {
		t.Fatal("repairing in the stage changed the surface lock before publish")
	}
}

func TestStaleStoreLayoutPublishesInputsWithoutTouchingNodeModules(t *testing.T) {
	// 61 of the fleet's pnpm surfaces were linked by pnpm 9 (store/v3); pnpm 10
	// resolves store/v3/v10 and would recreate node_modules to switch. The
	// governed change is the manifest and lockfile, so it proceeds and leaves
	// node_modules to the lifecycle instead of refusing or purging it.
	fixture := newPnpmFixture(t, testBaseManifest, testBaseLock, "beta", "2.0.0")
	fixture.installedLock(t, testBaseLock)
	modules := filepath.Join(fixture.surface, "node_modules", ".modules.yaml")
	if err := os.WriteFile(modules, []byte("packageManager: pnpm@9.12.1\nstoreDir: /elsewhere/store/v3\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	fake := &fakePnpm{add: resolvesTo(testNextManifest, testNextLock)}
	useFakePnpm(t, fake)
	sizer := &fakeSizer{}
	useFakeSizer(t, sizer)
	plan, err := PlanDiskAdmission(context.Background(), fixture.resolution)
	if err != nil || plan.Unsupported != "" {
		t.Fatalf("plan = %+v, %v", plan, err)
	}
	defer plan.Release()
	if len(sizer.asked) != 0 || plan.JSStage.LinkSkipped == "" || !strings.Contains(strings.Join(plan.Notes, "\n"), "node_modules will not be relinked") {
		t.Fatalf("sized %v, LinkSkipped=%q, notes=%v", sizer.asked, plan.JSStage.LinkSkipped, plan.Notes)
	}
	resolution := fixture.resolution
	resolution.DiskJSStage = plan.JSStage
	out, err := (ExecInstaller{}).Install(context.Background(), resolution)
	if err != nil || !strings.Contains(out, "not relinked") {
		t.Fatalf("install = %q, %v", out, err)
	}
	if fixture.read(t, "package.json") != testNextManifest || fixture.read(t, "pnpm-lock.yaml") != testNextLock {
		t.Fatal("the dependency inputs must still be published")
	}
	if len(fake.commands("install")) != 0 {
		t.Fatal("pnpm install must not run against node_modules linked to another store")
	}
	if data, _ := os.ReadFile(modules); !strings.Contains(string(data), "/elsewhere/store/v3") {
		t.Fatal("node_modules was modified")
	}
}

func TestPnpmEnvironmentNeverRunsInCIModeWhichPurgesNodeModules(t *testing.T) {
	// Observed with pnpm 10 on a pnpm 9 surface: CI=true recreates node_modules
	// without asking; CI=false aborts and leaves it. An inherited CI=true must
	// not reach pnpm.
	t.Setenv("CI", "true")
	env := "\n" + strings.Join(pnpmEnvironment(Resolution{}, "/data/store"), "\n") + "\n"
	if strings.Contains(env, "\nCI=true\n") || !strings.Contains(env, "\nCI=false\n") {
		t.Fatalf("pnpm would run in CI mode: %s", env)
	}
}

func TestPnpmEnvironmentUsesTheResolvedStoreAndLifecycleSettings(t *testing.T) {
	env := pnpmEnvironment(Resolution{}, "/data/store")
	joined := "\n" + strings.Join(env, "\n") + "\n"
	for _, want := range []string{"\nnpm_config_store_dir=/data/store\n", "\nCI=false\n", "\nnpm_config_registry=https://registry.npmjs.org/\n"} {
		if !strings.Contains(joined, want) {
			t.Errorf("pnpm environment lacks %q", strings.TrimSpace(want))
		}
	}
}

func TestPublishLinksFrozenLockAndKeepsStageContent(t *testing.T) {
	fixture := newPnpmFixture(t, testBaseManifest, testBaseLock, "beta", "2.0.0")
	fixture.installedLock(t, testBaseLock)
	fake := &fakePnpm{add: resolvesTo(testNextManifest, testNextLock)}
	useFakePnpm(t, fake)
	useFakeSizer(t, &fakeSizer{registry: map[string]artifactSize{"beta@2.0.0": {Unpacked: 10, Archive: 5}}})
	resolution := fixture.resolution
	resolution.Argv = append([]string{"pnpm", "add", "--ignore-scripts", "--ignore-workspace"}, "beta@2.0.0")
	plan, err := PlanDiskAdmission(context.Background(), resolution)
	if err != nil || plan.Unsupported != "" {
		t.Fatalf("plan = %+v, %v", plan, err)
	}
	if plan.CachePath != fake.store || len(plan.Paths) != 3 || plan.JSStage == nil {
		t.Fatalf("pnpm admission must reserve the resolved store %s: %+v", fake.store, plan)
	}
	resolution.DiskJSStage, resolution.DiskCachePath = plan.JSStage, plan.CachePath
	if _, err := (ExecInstaller{}).Install(context.Background(), resolution); err != nil {
		t.Fatal(err)
	}
	plan.Release()
	if fixture.read(t, "package.json") != testNextManifest || fixture.read(t, "pnpm-lock.yaml") != testNextLock {
		t.Fatal("apply must publish exactly the stage that admission sized")
	}
	installs := fake.commands("install")
	if len(installs) != 1 || strings.Join(installs[0].Args, " ") != "install --frozen-lockfile --ignore-scripts --ignore-workspace" || installs[0].Dir != fixture.surface {
		t.Fatalf("link step = %+v", installs)
	}
	if len(fake.commands("add")) != 1 {
		t.Fatal("apply re-resolved instead of publishing the admitted stage")
	}
	if left := stageLeftovers(t, fixture); len(left) != 0 {
		t.Fatalf("staging left %v", left)
	}
}

func TestFailedLinkRestoresInputsAndRelinksPreviousLock(t *testing.T) {
	fixture := newPnpmFixture(t, testBaseManifest, testBaseLock, "beta", "2.0.0")
	fixture.installedLock(t, testBaseLock)
	fake := &fakePnpm{add: resolvesTo(testNextManifest, testNextLock)}
	var locksAtInstall []string
	fake.install = func(call pnpmCall, attempt int) ([]byte, error) {
		lock, _ := os.ReadFile(filepath.Join(call.Dir, "pnpm-lock.yaml"))
		locksAtInstall = append(locksAtInstall, string(lock))
		if attempt == 1 {
			return []byte("ERR_PNPM_FETCH_404 beta"), fmt.Errorf("exit status 1")
		}
		return []byte("relinked"), nil
	}
	useFakePnpm(t, fake)
	useFakeSizer(t, &fakeSizer{registry: map[string]artifactSize{"beta@2.0.0": {Unpacked: 10, Archive: 5}}})
	out, err := (ExecInstaller{}).Install(context.Background(), fixture.resolution)
	if err == nil || !strings.Contains(err.Error(), "previous package.json, lockfile and node_modules were restored") {
		t.Fatalf("failed link = %v / %s", err, out)
	}
	if fixture.read(t, "package.json") != testBaseManifest || fixture.read(t, "pnpm-lock.yaml") != testBaseLock {
		t.Fatal("a failed link must restore the previous manifest and lockfile")
	}
	if len(locksAtInstall) != 2 || locksAtInstall[0] != testNextLock || locksAtInstall[1] != testBaseLock {
		t.Fatalf("the second link must run against the restored lock, saw %d runs", len(locksAtInstall))
	}
	if left := stageLeftovers(t, fixture); len(left) != 0 {
		t.Fatalf("staging left %v", left)
	}
}

func TestPublishRefusesWhenInputsChangedAfterAdmission(t *testing.T) {
	fixture := newPnpmFixture(t, testBaseManifest, testBaseLock, "beta", "2.0.0")
	fixture.installedLock(t, testBaseLock)
	fake := &fakePnpm{add: resolvesTo(testNextManifest, testNextLock)}
	useFakePnpm(t, fake)
	useFakeSizer(t, &fakeSizer{registry: map[string]artifactSize{"beta@2.0.0": {Unpacked: 10, Archive: 5}}})
	stage, err := stageJSResolution(context.Background(), fixture.resolution)
	if err != nil {
		t.Fatal(err)
	}
	defer stage.Release()
	concurrent := `{"name":"demo-ui","packageManager":"pnpm@10.34.9","dependencies":{"alpha":"1.0.0","gamma":"3.0.0"}}`
	if err := os.WriteFile(filepath.Join(fixture.surface, "package.json"), []byte(concurrent), 0o644); err != nil {
		t.Fatal(err)
	}
	resolution := fixture.resolution
	resolution.DiskJSStage = stage
	_, err = (ExecInstaller{}).Install(context.Background(), resolution)
	if err == nil || !strings.Contains(err.Error(), "changed after the install was admitted") {
		t.Fatalf("publish over concurrent work = %v", err)
	}
	if fixture.read(t, "package.json") != concurrent || len(fake.commands("install")) != 0 {
		t.Fatal("publish overwrote concurrent edits or linked anyway")
	}
}

func TestPublishBootstrapsAnExternalWorkspaceWithoutALock(t *testing.T) {
	root := t.TempDir()
	manifest := `{"name":"task","packageManager":"pnpm@10.34.9"}`
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	fake := &fakePnpm{add: resolvesTo(testNextManifest, testNextLock)}
	useFakePnpm(t, fake)
	useFakeSizer(t, &fakeSizer{registry: map[string]artifactSize{"alpha@1.0.0": {Unpacked: 10, Archive: 5}, "beta@2.0.0": {Unpacked: 10, Archive: 5}}})
	resolution := Resolution{
		ExternalWorkspace: true, RepositoryRoot: t.TempDir(), SurfaceRoot: root, ManifestPath: filepath.Join(root, "package.json"),
		PackageManager: "pnpm", PackageName: "beta", ExactVersion: true,
		Argv:    []string{"pnpm", "add", "--ignore-scripts", "--ignore-pnpmfile", "--ignore-workspace", "--save-exact", "beta@2.0.0"},
		Profile: InstallProfile{LifecycleMode: "scripts-disabled-by-default"},
	}
	if _, err := (ExecInstaller{}).Install(context.Background(), resolution); err != nil {
		t.Fatal(err)
	}
	if lock, err := os.ReadFile(filepath.Join(root, "pnpm-lock.yaml")); err != nil || string(lock) != testNextLock {
		t.Fatalf("external workspace lock = %q, %v", lock, err)
	}
	if _, err := os.Stat(filepath.Join(root, "pnpm-workspace.yaml")); !os.IsNotExist(err) {
		t.Fatal("external workspaces must not gain workspace configuration")
	}
	for _, call := range fake.commands("add") {
		if filepath.Dir(call.Dir) != os.TempDir() {
			t.Fatalf("external staging should live under the temp directory, ran in %s", call.Dir)
		}
		if !strings.Contains(strings.Join(call.Env, "\n"), "npm_config_ignore_pnpmfile=true") {
			t.Fatal("external workspaces keep the pnpmfile ban")
		}
	}
}

func TestRealJSStagesAndPublishesALocalPackage(t *testing.T) {
	if _, err := exec.LookPath("pnpm"); err != nil {
		t.Skip("pnpm is not installed")
	}
	store := t.TempDir()
	t.Setenv("npm_config_store_dir", store)
	repo := t.TempDir()
	surface := filepath.Join(repo, "scenarios", "demo", "ui")
	if err := os.MkdirAll(surface, 0o755); err != nil {
		t.Fatal(err)
	}
	spec := "file:../../../packages/viewer"
	localPackageFixture(t, repo, "viewer", "@vrooli/viewer", map[string]string{"index.js": "module.exports = 1;\n"})
	for name, body := range map[string]string{
		"package.json":        `{"name":"demo-ui","version":"1.0.0","private":true}`,
		"pnpm-workspace.yaml": testWorkspaceYAML,
	} {
		if err := os.WriteFile(filepath.Join(surface, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	resolution, err := Resolve(repo, "demo", "ui", "npm", "@vrooli/viewer", spec)
	if err != nil {
		// Resolve needs lockfile evidence for the manager; a first install has none.
		if err := os.WriteFile(filepath.Join(surface, "pnpm-lock.yaml"), []byte("lockfileVersion: '9.0'\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if resolution, err = Resolve(repo, "demo", "ui", "npm", "@vrooli/viewer", spec); err != nil {
			t.Fatal(err)
		}
	}
	if err := ApplyExactVersion(&resolution, spec); err != nil {
		t.Fatal(err)
	}
	plan, err := PlanDiskAdmission(context.Background(), resolution)
	if err != nil || plan.Unsupported != "" {
		t.Fatalf("real pnpm admission = %+v, %v", plan, err)
	}
	defer plan.Release()
	if plan.CachePath != store {
		t.Fatalf("admission reserved %s but pnpm uses %s", plan.CachePath, store)
	}
	if lock, _ := os.ReadFile(filepath.Join(surface, "pnpm-lock.yaml")); !bytes.Equal(lock, []byte("lockfileVersion: '9.0'\n")) {
		t.Fatal("planning changed the surface lock")
	}
	resolution.DiskJSStage, resolution.DiskCachePath = plan.JSStage, plan.CachePath
	if out, err := (ExecInstaller{}).Install(context.Background(), resolution); err != nil {
		t.Fatalf("install: %v\n%s", err, out)
	}
	var published struct{ Dependencies map[string]string }
	manifest, _ := os.ReadFile(filepath.Join(surface, "package.json"))
	if err := json.Unmarshal(manifest, &published); err != nil || published.Dependencies["@vrooli/viewer"] != spec {
		t.Fatalf("manifest not published: %s", manifest)
	}
	if lock, _ := os.ReadFile(filepath.Join(surface, "pnpm-lock.yaml")); !strings.Contains(string(lock), "directory: ../../../packages/viewer") {
		t.Fatalf("lock not published: %s", lock)
	}
	if _, err := os.Stat(filepath.Join(surface, "node_modules", "@vrooli", "viewer", "index.js")); err != nil {
		t.Fatalf("package was not linked: %v", err)
	}
}

const (
	npmBaseManifest = `{"name":"demo-ui","dependencies":{"old":"1.0.0"}}`
	npmBaseLock     = `{"name":"demo-ui","lockfileVersion":3,"packages":{"":{"dependencies":{"old":"1.0.0"}},"node_modules/old":{"version":"1.0.0","resolved":"https://registry.npmjs.org/old/-/old-1.0.0.tgz","integrity":"sha512-OLD"}}}`
	npmNextManifest = `{"name":"demo-ui","dependencies":{"old":"1.0.0","new-leaf":"1.2.3"}}`
	npmNextLock     = `{"name":"demo-ui","lockfileVersion":3,"packages":{"":{"dependencies":{"old":"1.0.0","new-leaf":"1.2.3"}},"node_modules/old":{"version":"1.0.0","resolved":"https://registry.npmjs.org/old/-/old-1.0.0.tgz","integrity":"sha512-OLD"},"node_modules/new-leaf":{"version":"1.2.3","resolved":"https://registry.npmjs.org/new-leaf/-/new-leaf-1.2.3.tgz","integrity":"sha512-NEW"},"node_modules/@esbuild/win32-x64":{"version":"0.1.0","resolved":"https://registry.npmjs.org/@esbuild/win32-x64/-/win32-x64-0.1.0.tgz","integrity":"sha512-WIN","os":["win32"],"cpu":["x64"]},"node_modules/linked":{"resolved":"../../packages/linked","link":true}}}`
)

func newNpmFixture(t *testing.T) pnpmFixture {
	t.Helper()
	repo := t.TempDir()
	surface := filepath.Join(repo, "scenarios", "legacy", "ui")
	if err := os.MkdirAll(filepath.Join(surface, "node_modules", "old"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{
		"package.json": npmBaseManifest, "package-lock.json": npmBaseLock, "node_modules/old/package.json": `{"name":"old","version":"1.0.0"}`,
	} {
		if err := os.WriteFile(filepath.Join(surface, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return pnpmFixture{repo: repo, surface: surface, resolution: Resolution{
		RepositoryRoot: repo, SurfaceRoot: surface, ManifestPath: filepath.Join(surface, "package.json"),
		PackageManager: "npm", PackageName: "new-leaf", Argv: []string{"npm", "install", "--ignore-scripts", "new-leaf@1.2.3"},
		Profile: InstallProfile{LifecycleMode: "scripts-disabled-by-default"},
	}}
}

func TestStageNpmSizesOnlyLockedPackagesMissingFromNodeModules(t *testing.T) {
	// A legacy npm surface that already declares dependencies used to be refused
	// outright; its resolved lock shows exactly what the install adds.
	fixture := newNpmFixture(t)
	fake := &fakePnpm{add: resolvesNpm(npmNextManifest, npmNextLock)}
	useFakePnpm(t, fake)
	sizer := &fakeSizer{registry: map[string]artifactSize{"new-leaf@1.2.3": {Unpacked: 700, Archive: 300}}}
	useFakeSizer(t, sizer)
	plan, err := PlanDiskAdmission(context.Background(), fixture.resolution)
	if err != nil || plan.Unsupported != "" {
		t.Fatalf("npm admission = %+v, %v", plan, err)
	}
	defer plan.Release()
	if strings.Join(sizer.asked, ",") != "new-leaf@1.2.3" {
		t.Fatalf("sized %v; the installed package, the linked package and the win32-only package must be skipped", sizer.asked)
	}
	if plan.JS.PackageCount != 1 || plan.JS.PackageBytes != 1000 || len(plan.Paths) != 3 {
		t.Fatalf("npm plan = %+v", plan)
	}
	if fixture.read(t, "package.json") != npmBaseManifest || fixture.read(t, "package-lock.json") != npmBaseLock {
		t.Fatal("npm planning changed the surface")
	}
	args := strings.Join(fake.commands("add")[0].Args, " ")
	if !strings.HasSuffix(args, "install --ignore-scripts new-leaf@1.2.3 --package-lock-only") {
		t.Fatalf("npm resolve command = %q", args)
	}
	if !strings.Contains(strings.Join(fake.commands("add")[0].Env, "\n"), "npm_config_cache="+filepath.Join(fake.commands("add")[0].Dir, ".sda-metadata-cache")) {
		t.Fatal("npm metadata cache must be isolated inside the stage")
	}
}

func resolvesNpm(manifest, lock string) func(pnpmCall) ([]byte, error) {
	return func(call pnpmCall) ([]byte, error) {
		if err := os.WriteFile(filepath.Join(call.Dir, "package.json"), []byte(manifest), 0o600); err != nil {
			return nil, err
		}
		return []byte("resolved"), os.WriteFile(filepath.Join(call.Dir, "package-lock.json"), []byte(lock), 0o600)
	}
}

func TestNpmPublishRestoresInputsWhenLinkFails(t *testing.T) {
	fixture := newNpmFixture(t)
	fake := &fakePnpm{add: resolvesNpm(npmNextManifest, npmNextLock)}
	fake.install = func(call pnpmCall, attempt int) ([]byte, error) {
		if attempt == 1 {
			return []byte("npm error code E500"), fmt.Errorf("exit status 1")
		}
		return []byte("pruned"), nil
	}
	useFakePnpm(t, fake)
	useFakeSizer(t, &fakeSizer{registry: map[string]artifactSize{"new-leaf@1.2.3": {Unpacked: 700, Archive: 300}}})
	_, err := (ExecInstaller{}).Install(context.Background(), fixture.resolution)
	if err == nil || !strings.Contains(err.Error(), "were restored") {
		t.Fatalf("npm link failure = %v", err)
	}
	if fixture.read(t, "package.json") != npmBaseManifest || fixture.read(t, "package-lock.json") != npmBaseLock {
		t.Fatal("a failed npm link must restore the manifest and lockfile")
	}
	installs := fake.commands("install")
	if len(installs) != 2 || strings.Join(installs[0].Args, " ") != "install --ignore-scripts --no-audit --no-fund" {
		t.Fatalf("npm link calls = %+v", installs)
	}
}

func TestNpmEnvironmentDoesNotLoadOneConfigAsUserAndGlobal(t *testing.T) {
	// npm exits with "double-loading config" when both slots name /dev/null.
	env := strings.Join(npmEnvironment(Resolution{DiskCachePath: "/data/npm-cache"}), "\n")
	if !strings.Contains(env, "npm_config_userconfig="+os.DevNull) {
		t.Fatalf("user config not neutralized: %s", env)
	}
	if strings.Contains(env, "npm_config_globalconfig="+os.DevNull) {
		t.Fatal("npm rejects /dev/null as both the user and the global config")
	}
}

func TestStageSkipsPnpmPackagesLockedForAnotherPlatform(t *testing.T) {
	fixture := newPnpmFixture(t, testBaseManifest, testBaseLock, "beta", "2.0.0")
	nextLock := lockPlus("  beta@2.0.0:\n    resolution: {integrity: sha512-BBBB}\n  '@esbuild/win32-x64@0.1.0':\n    resolution: {integrity: sha512-WIN}\n    cpu: [x64]\n    os: [win32]\n")
	useFakePnpm(t, &fakePnpm{add: resolvesTo(testNextManifest, nextLock)})
	sizer := &fakeSizer{registry: map[string]artifactSize{"alpha@1.0.0": {Unpacked: 100, Archive: 10}, "beta@2.0.0": {Unpacked: 4000, Archive: 1000}}}
	useFakeSizer(t, sizer)
	stage, err := stageJSResolution(context.Background(), fixture.resolution)
	if err != nil {
		t.Fatal(err)
	}
	defer stage.Release()
	for _, asked := range sizer.asked {
		if strings.Contains(asked, "esbuild") {
			t.Fatalf("a win32-only package was sized on this host: %v", sizer.asked)
		}
	}
}

func TestStageSweepsAbandonedStagesButNotLiveOnes(t *testing.T) {
	fixture := newPnpmFixture(t, testBaseManifest, testBaseLock, "beta", "2.0.0")
	fixture.installedLock(t, testBaseLock)
	parent := filepath.Dir(fixture.surface)
	abandoned := filepath.Join(parent, ".sda-stage-111")
	live := filepath.Join(parent, ".sda-stage-222")
	unrelated := filepath.Join(parent, ".sda-stage-notes")
	for _, dir := range []string{abandoned, live, unrelated} {
		if err := os.MkdirAll(filepath.Join(dir, "cache"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	old := time.Now().Add(-3 * time.Hour)
	for _, dir := range []string{abandoned, unrelated} {
		if err := os.Chtimes(dir, old, old); err != nil {
			t.Fatal(err)
		}
	}
	useFakePnpm(t, &fakePnpm{add: resolvesTo(testNextManifest, testNextLock)})
	useFakeSizer(t, &fakeSizer{registry: map[string]artifactSize{"beta@2.0.0": {Unpacked: 10, Archive: 5}}})
	stage, err := stageJSResolution(context.Background(), fixture.resolution)
	if err != nil {
		t.Fatal(err)
	}
	stage.Release()
	for dir, wantExists := range map[string]bool{abandoned: false, live: true, unrelated: true} {
		if _, err := os.Stat(dir); (err == nil) != wantExists {
			t.Errorf("%s exists=%t, want %t", filepath.Base(dir), err == nil, wantExists)
		}
	}
}

// The web-console evidence: a lock written by an older pnpm holds a tarball
// URL entry with no integrity. Removing an unrelated dependency used to leave it
// that way, so every later governed run hit the same refusal. The lock now
// identifies the bytes, with a one-line diff and pnpm's own formatting.
func TestRemovalPinsTheIntegrityOfAnIntegrityLessTarballEntry(t *testing.T) {
	lock := tarballLock("")
	fixture := newPnpmFixture(t, testBaseManifest, lock, "alpha", "")
	fixture.installedLock(t, lock)
	fixture.resolution.Argv = []string{"pnpm", "remove", "--workspace-root", "alpha"}
	fixture.resolution.RemovedPackages = []string{"alpha"}
	useFakePnpm(t, &fakePnpm{add: resolvesTo(`{"name":"demo-ui"}`, lock+"# resolved\n")})
	sizer := &fakeSizer{tarballs: map[string]artifactSize{sheetjsURL: {Unpacked: 90, Archive: 30, Integrity: "sha512-PIN"}}}
	useFakeSizer(t, sizer)

	stage, err := stageJSResolution(context.Background(), fixture.resolution)
	if err != nil {
		t.Fatal(err)
	}
	defer stage.Release()
	pinned := string(stage.next["pnpm-lock.yaml"])
	want := strings.Replace(lock+"# resolved\n", "resolution: {tarball:", "resolution: {integrity: sha512-PIN, tarball:", 1)
	if pinned != want {
		t.Fatalf("pinning must change exactly the one resolution line.\n got: %q\nwant: %q", pinned, want)
	}
	if len(sizer.asked) != 1 || !strings.HasPrefix(sizer.asked[0], sheetjsURL) {
		t.Fatalf("the tarball must be downloaded once, for pinning and sizing together: %v", sizer.asked)
	}
	if !strings.Contains(strings.Join(stage.Notes, "\n"), "pinned to sha512-PIN") {
		t.Fatalf("the reviewer must see the pinned digest: %v", stage.Notes)
	}
}

func TestSetPnpmLockIntegrityRefusesAnEntryItCannotEditSafely(t *testing.T) {
	if _, err := setPnpmLockIntegrity([]byte(tarballLock("")), "nope@https://x/y.tgz", "sha512-X"); err == nil {
		t.Fatal("an unknown key must be refused")
	}
	already := tarballLock("sha512-OLD")
	if _, err := setPnpmLockIntegrity([]byte(already), sheetjsKey, "sha512-NEW"); err == nil {
		t.Fatal("an entry that already has an integrity must not be overwritten")
	}
}

func immatureError(name, version string) []byte {
	return []byte(" ERR_PNPM_NO_MATURE_MATCHING_VERSION  Version " + version + " (released 47 hours ago) of " + name + " does not meet the minimumReleaseAge constraint")
}

// notification-hub/ui evidence: a raw pnpm run locked a native package at a
// version younger than the seven-day policy, so every governed change failed.
// The policy stays; pnpm re-resolves that package to a mature version.
func TestStageReResolvesALockedVersionYoungerThanTheReleaseAgePolicy(t *testing.T) {
	young := lockPlus("  beta@2.0.0:\n    resolution: {integrity: sha512-BETA}\n")
	fixture := newPnpmFixture(t, testBaseManifest, young, "gamma", "3.0.0")
	fixture.installedLock(t, young)
	fixture.resolution.Argv = []string{"pnpm", "add", "--ignore-scripts", "--workspace-root", "gamma@3.0.0"}
	mature := lockPlus("  beta@1.9.0:\n    resolution: {integrity: sha512-OLDER}\n")
	var order []string
	fake := &fakePnpm{}
	fake.add = func(call pnpmCall) ([]byte, error) {
		lock, _ := os.ReadFile(filepath.Join(call.Dir, "pnpm-lock.yaml"))
		if strings.Contains(string(lock), "beta@2.0.0") {
			order = append(order, "add-refused")
			return immatureError("beta", "2.0.0"), fmt.Errorf("exit status 1")
		}
		order = append(order, "add")
		return resolvesTo(testNextManifest, mature+"# gamma\n")(call)
	}
	fake.update = func(call pnpmCall) ([]byte, error) {
		order = append(order, "update "+strings.Join(call.Args, " "))
		return []byte("updated"), os.WriteFile(filepath.Join(call.Dir, "pnpm-lock.yaml"), []byte(mature), 0o600)
	}
	useFakePnpm(t, fake)
	useFakeSizer(t, &fakeSizer{registry: map[string]artifactSize{"beta@1.9.0": {Unpacked: 10, Archive: 5}, "gamma@3.0.0": {Unpacked: 10, Archive: 5}}})

	stage, err := stageJSResolution(context.Background(), fixture.resolution)
	if err != nil {
		t.Fatal(err)
	}
	defer stage.Release()
	if got := strings.Join(order, " | "); got != "add-refused | update update beta --lockfile-only --workspace-root | add" {
		t.Fatalf("sequence = %q", got)
	}
	if notes := strings.Join(stage.Notes, "\n"); !strings.Contains(notes, "beta@2.0.0") || !strings.Contains(notes, "younger than the release-age policy") {
		t.Fatalf("the reviewer must be told what was re-resolved and why: %v", stage.Notes)
	}
}

func TestStageDoesNotRelaxThePolicyForARequestedPackageThatIsTooNew(t *testing.T) {
	fixture := newPnpmFixture(t, testBaseManifest, testBaseLock, "gamma", "3.0.0")
	fake := &fakePnpm{add: func(pnpmCall) ([]byte, error) { return immatureError("gamma", "3.0.0"), fmt.Errorf("exit status 1") }}
	fake.update = func(pnpmCall) ([]byte, error) {
		t.Error("update must not run for a package the lock does not hold")
		return nil, nil
	}
	useFakePnpm(t, fake)
	_, err := stageJSResolution(context.Background(), fixture.resolution)
	if err == nil || !strings.Contains(err.Error(), "release-age policy") {
		t.Fatalf("err = %v", err)
	}
}

// packages/file-viewer: node_modules was linked against store/v10 while the user
// config selects store/v3/v10. pnpm can select the recorded store, so the
// change links against it; nothing is recreated and nothing is skipped.
func TestStageFollowsTheStoreNodeModulesIsLinkedAgainstWhenPnpmCanSelectIt(t *testing.T) {
	fixture := newPnpmFixture(t, testBaseManifest, testBaseLock, "beta", "2.0.0")
	fixture.installedLock(t, testBaseLock)
	recordedBase := filepath.Join(t.TempDir(), "store")
	modules := filepath.Join(fixture.surface, "node_modules", ".modules.yaml")
	if err := os.WriteFile(modules, []byte("{\n  \"storeDir\": \""+filepath.Join(recordedBase, "v10")+"\"\n}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	fake := &fakePnpm{add: resolvesTo(testNextManifest, testNextLock)}
	useFakePnpm(t, fake)
	useFakeSizer(t, &fakeSizer{registry: map[string]artifactSize{"beta@2.0.0": {Unpacked: 10, Archive: 5}}})
	stage, err := stageJSResolution(context.Background(), fixture.resolution)
	if err != nil {
		t.Fatal(err)
	}
	defer stage.Release()
	if stage.LinkSkipped != "" {
		t.Fatalf("a store pnpm can select must not skip the link: %s", stage.LinkSkipped)
	}
	if !strings.Contains(strings.Join(stage.Notes, "\n"), "uses the recorded store") {
		t.Fatalf("the reviewer must see which store is used: %v", stage.Notes)
	}
	resolve := fake.commands("add")[0]
	if !containsEnv(resolve.Env, "npm_config_store_dir="+recordedBase) {
		t.Fatalf("the resolver must run against the recorded store %s: %v", recordedBase, resolve.Env)
	}
}

func containsEnv(env []string, want string) bool {
	for _, item := range env {
		if item == want {
			return true
		}
	}
	return false
}

// Flags are appended after the package argument (-D, --save-exact,
// --ignore-workspace), so the requested specifier cannot be "the last argument".
func TestRequestedSpecIsFoundByNameWhenFlagsFollowThePackage(t *testing.T) {
	for _, argv := range [][]string{
		{"pnpm", "add", "--ignore-scripts", "xlsx@" + sheetjsURL},
		{"pnpm", "add", "--ignore-scripts", "xlsx@" + sheetjsURL, "-D"},
		{"pnpm", "add", "xlsx@" + sheetjsURL, "-D", "--ignore-workspace"},
	} {
		if got := requestedSpec(Resolution{PackageName: "xlsx", Argv: argv}); got != sheetjsURL {
			t.Errorf("%v: spec = %q", argv, got)
		}
	}
	if got := requestedSpec(Resolution{PackageName: "xlsx", Argv: []string{"pnpm", "add", "xlsx"}}); got != "" {
		t.Errorf("a request without a version has no specifier, got %q", got)
	}
}

// A lock-only request publishes the manifest and lockfile and does nothing else:
// no size probe, no link, node_modules and the package store untouched, so its
// plan is the small input-sized one even where a link would have been possible.
func TestLockOnlyRequestPublishesInputsAndNeverLinksOrSizesPackages(t *testing.T) {
	fixture := newPnpmFixture(t, testBaseManifest, testBaseLock, "beta", "2.0.0")
	fixture.installedLock(t, testBaseLock)
	fake := &fakePnpm{add: resolvesTo(testNextManifest, testNextLock)}
	useFakePnpm(t, fake)
	sizer := &fakeSizer{}
	useFakeSizer(t, sizer)
	resolution := fixture.resolution
	resolution.LockOnly = true

	plan, err := PlanDiskAdmission(context.Background(), resolution)
	if err != nil || plan.Unsupported != "" {
		t.Fatalf("plan = %+v, %v", plan, err)
	}
	defer plan.Release()
	if len(sizer.asked) != 0 || plan.JSStage.LinkSkipped == "" || !strings.Contains(strings.Join(plan.Notes, "\n"), "lock-only") {
		t.Fatalf("a lock-only request must not size or link packages: sized %v, LinkSkipped=%q, notes=%v", sizer.asked, plan.JSStage.LinkSkipped, plan.Notes)
	}
	resolution.DiskJSStage = plan.JSStage
	out, err := (ExecInstaller{}).Install(context.Background(), resolution)
	if err != nil || !strings.Contains(out, "not relinked") {
		t.Fatalf("install = %q, %v", out, err)
	}
	if fixture.read(t, "package.json") != testNextManifest || fixture.read(t, "pnpm-lock.yaml") != testNextLock {
		t.Fatal("the lock-only update must still publish the dependency inputs")
	}
	for _, call := range fake.calls {
		if len(call.Args) > 0 && call.Args[0] == "install" {
			t.Fatalf("a lock-only update must not run a linking install: %v", call.Args)
		}
	}
}

// An override is part of the staged resolution: the stage's own package.json
// carries it when the package manager resolves, and publishing writes that
// manifest together with the lockfile it produced. (Writing it to the real
// manifest after admission made the publish see a changed base and refuse.)
func TestOverrideIsResolvedInTheStageAndPublishedWithItsLock(t *testing.T) {
	fixture := newPnpmFixture(t, testBaseManifest, testBaseLock, "beta", "2.0.0")
	fixture.installedLock(t, testBaseLock)
	var seenByResolver string
	fake := &fakePnpm{add: func(call pnpmCall) ([]byte, error) {
		manifest, err := os.ReadFile(filepath.Join(call.Dir, "package.json"))
		if err != nil {
			return nil, err
		}
		seenByResolver = string(manifest)
		return resolvesTo(string(manifest), testNextLock)(call)
	}}
	useFakePnpm(t, fake)
	useFakeSizer(t, &fakeSizer{})
	resolution := fixture.resolution
	resolution.LockOnly = true
	resolution.NpmOverride = &NpmOverride{Selector: "beta", Version: "2.0.1"}

	plan, err := PlanDiskAdmission(context.Background(), resolution)
	if err != nil || plan.Unsupported != "" {
		t.Fatalf("plan = %+v, %v", plan, err)
	}
	defer plan.Release()
	if !strings.Contains(seenByResolver, `"beta": "2.0.1"`) {
		t.Fatalf("the resolver must see the override in the staged manifest: %s", seenByResolver)
	}
	if fixture.read(t, "package.json") != testBaseManifest {
		t.Fatal("planning must not touch the real manifest")
	}
	resolution.DiskJSStage = plan.JSStage
	if out, err := (ExecInstaller{}).Install(context.Background(), resolution); err != nil {
		t.Fatalf("install = %q, %v", out, err)
	}
	if !strings.Contains(fixture.read(t, "package.json"), `"beta": "2.0.1"`) || fixture.read(t, "pnpm-lock.yaml") != testNextLock {
		t.Fatalf("the override and its lock must be published together:\n%s", fixture.read(t, "package.json"))
	}
}

func TestOverrideSelectorsSetExactlyThatEntryAndNameTheirPackage(t *testing.T) {
	for selector, want := range map[string]string{
		"brace-expansion": "brace-expansion", "brace-expansion@1": "brace-expansion", "@scope/pkg@^2": "@scope/pkg",
		"minimatch@3>brace-expansion": "brace-expansion", "express>@scope/pkg@1": "@scope/pkg",
	} {
		if err := ValidateOverrideSelector(selector); err != nil || OverridePackage(selector) != want {
			t.Errorf("%q: package %q (%v), want %q", selector, OverridePackage(selector), err, want)
		}
	}
	for _, bad := range []string{"", "a b", "a;b", "../x", "a>>b", "a>", "$(x)"} {
		if ValidateOverrideSelector(bad) == nil {
			t.Errorf("%q must be refused", bad)
		}
	}

	manifest := filepath.Join(t.TempDir(), "package.json")
	const original = `{"name":"x","pnpm":{"overrides":{"brace-expansion@1":"1.1.18","brace-expansion@2":"2.1.4","minimatch@3>brace-expansion":"1.1.18"}}}`
	if err := os.WriteFile(manifest, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := SetNpmOverride(manifest, "brace-expansion@1", "1.1.21"); err != nil {
		t.Fatal(err)
	}
	if err := SetNpmOverride(manifest, "minimatch@3>brace-expansion", "1.1.21"); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(manifest)
	var got struct {
		Pnpm struct{ Overrides map[string]string }
	}
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"brace-expansion@1": "1.1.21", "brace-expansion@2": "2.1.4", "minimatch@3>brace-expansion": "1.1.21"}
	if len(got.Pnpm.Overrides) != 3 || got.Pnpm.Overrides["brace-expansion@1"] != want["brace-expansion@1"] || got.Pnpm.Overrides["brace-expansion@2"] != "2.1.4" || got.Pnpm.Overrides["minimatch@3>brace-expansion"] != "1.1.21" {
		t.Fatalf("an explicit selector must change only its own entry: %v", got.Pnpm.Overrides)
	}
	if err := SetNpmOverride(manifest, "brace-expansion", "1.1.21"); err != nil {
		t.Fatal(err)
	}
	data, _ = os.ReadFile(manifest)
	if strings.Contains(string(data), "brace-expansion@") && !strings.Contains(string(data), "minimatch@3>brace-expansion") {
		t.Fatalf("a package-wide override keeps parent-qualified entries: %s", data)
	}
	if strings.Contains(string(data), `"brace-expansion@1"`) || strings.Contains(string(data), `"brace-expansion@2"`) {
		t.Fatalf("a package-wide override retires the version-specific entries: %s", data)
	}
}
