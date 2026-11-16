package installgateway

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func removalSurface(t *testing.T, manifest, lockName, lock string) (repo, surface string) {
	t.Helper()
	repo = t.TempDir()
	surface = filepath.Join(repo, "scenarios", "demo", "ui")
	if err := os.MkdirAll(surface, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{"package.json": manifest, lockName: lock} {
		if err := os.WriteFile(filepath.Join(surface, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return repo, surface
}

func TestResolveRemovalBuildsAManagerCommandForDeclaredDependencies(t *testing.T) {
	manifest := `{"name":"demo-ui","dependencies":{"three":"^0.170.0","yaml":"^2.0.0"},"devDependencies":{"@types/three":"^0.170.0"},"optionalDependencies":{"fsevents":"^2.0.0"}}`
	repo, _ := removalSurface(t, manifest, "pnpm-lock.yaml", "lockfileVersion: '9.0'\n")
	r, err := ResolveRemoval(repo, "demo", "ui", "js", []string{"three", "@types/three", "fsevents", "three"})
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(r.Argv, " "); got != "pnpm remove three @types/three fsevents" {
		t.Fatalf("argv = %q (dev, optional and runtime dependencies are all removable; duplicates collapse)", got)
	}
	if r.PackageManager != "pnpm" || len(r.RemovedPackages) != 3 {
		t.Fatalf("resolution = %+v", r)
	}

	repo, _ = removalSurface(t, manifest, "package-lock.json", "{}")
	r, err = ResolveRemoval(repo, "demo", "ui", "npm", []string{"yaml"})
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(r.Argv, " "); got != "npm uninstall --ignore-scripts yaml" || !r.Profile.ScriptsDisabled {
		t.Fatalf("npm argv = %q profile=%+v", got, r.Profile)
	}
}

func TestResolveRemovalRefusesWhatItCannotRemoveAndSaysWhy(t *testing.T) {
	manifest := `{"name":"demo-ui","dependencies":{"three":"^0.170.0"}}`
	repo, _ := removalSurface(t, manifest, "pnpm-lock.yaml", "lockfileVersion: '9.0'\n")
	yarnRepo, _ := removalSurface(t, manifest, "yarn.lock", "")
	for _, test := range []struct {
		name                               string
		repo, surface, ecosystem, wantText string
		packages                           []string
	}{
		{"transitive or misspelled", repo, "ui", "npm", "not a direct dependency", []string{"three", "threejs"}},
		{"version attached", repo, "ui", "npm", "carries a version", []string{"three@0.170.0"}},
		{"no packages", repo, "ui", "npm", "at least one package", nil},
		{"go ecosystem", repo, "ui", "go", "supports npm packages", []string{"three"}},
		{"yarn surface", yarnRepo, "ui", "npm", "supports pnpm and npm surfaces", []string{"three"}},
		{"missing surface", repo, "api", "npm", "surface directory not found", []string{"three"}},
		{"template surface", repo, "template/vanilla", "npm", "does not apply", []string{"three"}},
		{"flag as name", repo, "ui", "npm", "cannot contain shell syntax or flags", []string{"--force"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := ResolveRemoval(test.repo, "demo", test.surface, test.ecosystem, test.packages)
			if err == nil || !strings.Contains(err.Error(), test.wantText) {
				t.Fatalf("err = %v, want it to contain %q", err, test.wantText)
			}
		})
	}
	if _, err := ResolveRemoval(repo, "demo", "ui", "npm", []string{"threejs"}); err == nil || !strings.Contains(err.Error(), "threejs") || strings.Contains(err.Error(), "three,") {
		t.Fatalf("the refusal must name only the undeclared package: %v", err)
	}
}

const (
	removeBaseManifest = `{"name":"demo-ui","packageManager":"pnpm@10.34.9","dependencies":{"alpha":"1.0.0","beta":"2.0.0"}}`
	removeNextManifest = `{"name":"demo-ui","packageManager":"pnpm@10.34.9","dependencies":{"alpha":"1.0.0"}}`
)

func TestStageRemovalPreviewsTheLockDeltaAndPublishesWithoutGrowth(t *testing.T) {
	fixture := newPnpmFixture(t, removeBaseManifest, testNextLock, "beta", "2.0.0")
	fixture.installedLock(t, testNextLock)
	fixture.resolution.Argv = []string{"pnpm", "remove", "--workspace-root", "beta"}
	fixture.resolution.RemovedPackages = []string{"beta"}
	fake := &fakePnpm{add: resolvesTo(removeNextManifest, testBaseLock)}
	useFakePnpm(t, fake)
	sizer := &fakeSizer{}
	useFakeSizer(t, sizer)

	plan, err := PlanDiskAdmission(context.Background(), fixture.resolution)
	if err != nil || plan.Unsupported != "" {
		t.Fatalf("removal admission = %+v, %v", plan, err)
	}
	defer plan.Release()
	if len(sizer.asked) != 0 || plan.JSStage.Measured.PackageCount != 0 {
		t.Fatalf("a removal adds nothing to size: asked=%v measured=%+v", sizer.asked, plan.JSStage.Measured)
	}
	removed, added, err := plan.JSStage.LockDelta()
	if err != nil || strings.Join(removed, ",") != "beta@2.0.0" || len(added) != 0 {
		t.Fatalf("lock delta = removed %v added %v (%v)", removed, added, err)
	}
	call := fake.commands("add")[0]
	if got := strings.Join(call.Args, " "); got != "remove --lockfile-only --workspace-root beta" {
		t.Fatalf("resolver args = %q", got)
	}
	if fixture.read(t, "package.json") != removeBaseManifest {
		t.Fatal("planning a removal changed the surface")
	}

	fixture.resolution.DiskJSStage = plan.JSStage
	if _, err := (ExecInstaller{}).Install(context.Background(), fixture.resolution); err != nil {
		t.Fatal(err)
	}
	if fixture.read(t, "package.json") != removeNextManifest || fixture.read(t, "pnpm-lock.yaml") != testBaseLock {
		t.Fatal("the staged removal was not published")
	}
	if links := fake.commands("install"); len(links) != 1 || !strings.Contains(strings.Join(links[0].Args, " "), "--frozen-lockfile") {
		t.Fatalf("node_modules must be relinked from the frozen lock: %+v", links)
	}
}

func TestStageRemovalOfAPackageTheLockCannotDropExplainsTheRefusal(t *testing.T) {
	fixture := newPnpmFixture(t, removeBaseManifest, testNextLock, "beta", "2.0.0")
	fixture.resolution.Argv = []string{"pnpm", "remove", "beta"}
	fixture.resolution.RemovedPackages = []string{"beta"}
	fake := &fakePnpm{add: func(pnpmCall) ([]byte, error) {
		return []byte(" ERR_PNPM_CANNOT_REMOVE_MISSING_DEPS  Cannot remove 'beta': no such dependency found"), os.ErrInvalid
	}}
	useFakePnpm(t, fake)
	_, err := stageJSResolution(context.Background(), fixture.resolution)
	if err == nil || !strings.Contains(err.Error(), "could not resolve the removal of beta") || !strings.Contains(err.Error(), "the surface was not changed") {
		t.Fatalf("err = %v", err)
	}
}

// Real pnpm: install two local shared packages through the gateway, then remove
// one. node_modules must lose only the removed package, and the manifest and
// lockfile must stop naming it.
func TestRealJSRemovalPrunesNodeModulesAndKeepsTheRest(t *testing.T) {
	if _, err := exec.LookPath("pnpm"); err != nil {
		t.Skip("pnpm is not installed")
	}
	t.Setenv("npm_config_store_dir", t.TempDir())
	repo, surface := removalSurface(t, `{"name":"demo-ui","version":"1.0.0","private":true}`, "pnpm-lock.yaml", "lockfileVersion: '9.0'\n")
	if err := os.WriteFile(filepath.Join(surface, "pnpm-workspace.yaml"), []byte(testWorkspaceYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, pkg := range []string{"viewer", "extra"} {
		localPackageFixture(t, repo, pkg, "@vrooli/"+pkg, map[string]string{"index.js": "module.exports = 1;\n"})
		spec := "file:../../../packages/" + pkg
		resolution, err := Resolve(repo, "demo", "ui", "npm", "@vrooli/"+pkg, spec)
		if err != nil {
			t.Fatal(err)
		}
		if err := ApplyExactVersion(&resolution, spec); err != nil {
			t.Fatal(err)
		}
		plan, err := PlanDiskAdmission(context.Background(), resolution)
		if err != nil || plan.Unsupported != "" {
			t.Fatalf("install %s: %+v %v", pkg, plan, err)
		}
		resolution.DiskJSStage = plan.JSStage
		if out, err := (ExecInstaller{}).Install(context.Background(), resolution); err != nil {
			t.Fatalf("install %s: %v\n%s", pkg, err, out)
		}
		plan.Release()
	}
	linked := func(pkg string) bool {
		_, err := os.Stat(filepath.Join(surface, "node_modules", "@vrooli", pkg, "index.js"))
		return err == nil
	}
	if !linked("viewer") || !linked("extra") {
		t.Fatal("fixture setup did not link both packages")
	}

	resolution, err := ResolveRemoval(repo, "demo", "ui", "npm", []string{"@vrooli/viewer"})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := PlanDiskAdmission(context.Background(), resolution)
	if err != nil || plan.Unsupported != "" {
		t.Fatalf("removal admission: %+v %v", plan, err)
	}
	defer plan.Release()
	removed, added, err := plan.JSStage.LockDelta()
	if err != nil || len(removed) != 1 || !strings.HasPrefix(removed[0], "@vrooli/viewer@") || len(added) != 0 {
		t.Fatalf("lock delta = %v %v (%v)", removed, added, err)
	}
	if !linked("viewer") {
		t.Fatal("planning a removal touched node_modules")
	}
	resolution.DiskJSStage = plan.JSStage
	if out, err := (ExecInstaller{}).Install(context.Background(), resolution); err != nil {
		t.Fatalf("remove: %v\n%s", err, out)
	}
	if linked("viewer") || !linked("extra") {
		t.Fatalf("after removal viewer linked=%t extra linked=%t", linked("viewer"), linked("extra"))
	}
	var manifest struct{ Dependencies map[string]string }
	data, _ := os.ReadFile(filepath.Join(surface, "package.json"))
	if err := json.Unmarshal(data, &manifest); err != nil || len(manifest.Dependencies) != 1 || manifest.Dependencies["@vrooli/extra"] == "" {
		t.Fatalf("manifest after removal: %s", data)
	}
	if lock, _ := os.ReadFile(filepath.Join(surface, "pnpm-lock.yaml")); strings.Contains(string(lock), "packages/viewer") {
		t.Fatalf("the lockfile still names the removed package:\n%s", lock)
	}
}

func TestResolveLockUpdateBuildsAnUpdateCommandForLockedPackages(t *testing.T) {
	manifest := `{"name":"demo-ui","dependencies":{"express":"^4.22.1"}}`
	lock := "lockfileVersion: '9.0'\n\npackages:\n\n  body-parser@1.20.5:\n    resolution: {integrity: sha512-x}\n\n  '@scope/pkg@1.0.0':\n    resolution: {integrity: sha512-y}\n"
	repo, _ := removalSurface(t, manifest, "pnpm-lock.yaml", lock)

	r, err := ResolveLockUpdate(repo, "demo", "ui", "npm", []string{"body-parser", "@scope/pkg", "body-parser"})
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(r.Argv, " "); got != "pnpm update body-parser @scope/pkg" {
		t.Fatalf("argv = %q (a transitive package is updatable; duplicates collapse)", got)
	}
	if len(r.UpdatedPackages) != 2 || len(r.RemovedPackages) != 0 {
		t.Fatalf("resolution = %+v", r)
	}

	npmLock := `{"lockfileVersion":3,"packages":{"":{},"node_modules/body-parser":{"version":"1.20.5"}}}`
	repo, _ = removalSurface(t, manifest, "package-lock.json", npmLock)
	r, err = ResolveLockUpdate(repo, "demo", "ui", "npm", []string{"body-parser"})
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(r.Argv, " "); got != "npm update --ignore-scripts body-parser" || !r.Profile.ScriptsDisabled {
		t.Fatalf("npm argv = %q profile=%+v", got, r.Profile)
	}
}

// pnpm limits one `update` to direct dependencies as soon as any name is direct,
// so direct and transitive names must be separate commands of one resolution.
func TestResolveLockUpdateMovesDirectAndTransitiveNamesInSeparateCommands(t *testing.T) {
	manifest := `{"name":"demo-ui","dependencies":{"express":"^4.22.1"},"devDependencies":{"vite":"^5.4.6"}}`
	lock := "lockfileVersion: '9.0'\n\npackages:\n\n  body-parser@1.20.5:\n    resolution: {integrity: sha512-x}\n\n  express@4.22.1:\n    resolution: {integrity: sha512-y}\n\n  qs@6.14.2:\n    resolution: {integrity: sha512-z}\n\n  vite@5.4.21:\n    resolution: {integrity: sha512-v}\n"
	repo, _ := removalSurface(t, manifest, "pnpm-lock.yaml", lock)

	r, err := ResolveLockUpdate(repo, "demo", "ui", "npm", []string{"body-parser", "vite", "qs", "express"})
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(r.Argv, " "); got != "pnpm update vite express" {
		t.Fatalf("the direct names go first, got %q", got)
	}
	if len(r.ThenArgv) != 1 || strings.Join(r.ThenArgv[0], " ") != "pnpm update body-parser qs" {
		t.Fatalf("the transitive names follow in their own command, got %v", r.ThenArgv)
	}
	if got := r.Command(); got != "pnpm update vite express; then pnpm update body-parser qs" {
		t.Fatalf("the preview must show both commands, got %q", got)
	}
	if len(r.UpdatedPackages) != 4 {
		t.Fatalf("every name is still reported as updated: %v", r.UpdatedPackages)
	}

	r, err = ResolveLockUpdate(repo, "demo", "ui", "npm", []string{"body-parser", "qs"})
	if err != nil || len(r.ThenArgv) != 0 || strings.Join(r.Argv, " ") != "pnpm update body-parser qs" {
		t.Fatalf("names of one kind stay one command: %+v (%v)", r, err)
	}
}

func TestResolveLockUpdateRefusesWhatItCannotUpdateAndSaysWhy(t *testing.T) {
	manifest := `{"name":"demo-ui","dependencies":{"express":"^4.22.1"}}`
	lock := "lockfileVersion: '9.0'\n\npackages:\n\n  body-parser@1.20.5:\n    resolution: {integrity: sha512-x}\n"
	repo, _ := removalSurface(t, manifest, "pnpm-lock.yaml", lock)
	yarnRepo, _ := removalSurface(t, manifest, "yarn.lock", "")
	for _, test := range []struct {
		name                               string
		repo, surface, ecosystem, wantText string
		packages                           []string
	}{
		{"not locked", repo, "ui", "npm", "left-pad is not in the lockfile", []string{"body-parser", "left-pad"}},
		{"version attached", repo, "ui", "npm", "carries a version", []string{"body-parser@1.20.9"}},
		{"no packages", repo, "ui", "npm", "at least one package", nil},
		{"go ecosystem", repo, "ui", "go", "supports npm packages", []string{"body-parser"}},
		{"yarn surface", yarnRepo, "ui", "npm", "supports pnpm and npm surfaces", []string{"body-parser"}},
		{"missing surface", repo, "api", "npm", "surface directory not found", []string{"body-parser"}},
		{"template surface", repo, "template/vanilla", "npm", "does not apply", []string{"body-parser"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := ResolveLockUpdate(test.repo, "demo", test.surface, test.ecosystem, test.packages)
			if err == nil || !strings.Contains(err.Error(), test.wantText) {
				t.Fatalf("err = %v, want it to contain %q", err, test.wantText)
			}
		})
	}
}

// An update resolves in the staging copy with the package manager's own update
// verb, previews which locked versions leave and enter, and publishes the lock
// without touching the manifest range.
func TestStageUpdatePreviewsTheLockDeltaAndLeavesTheManifestAlone(t *testing.T) {
	fixture := newPnpmFixture(t, removeBaseManifest, testNextLock, "beta", "2.0.0")
	fixture.installedLock(t, testNextLock)
	fixture.resolution.Argv = []string{"pnpm", "update", "--workspace-root", "beta"}
	fixture.resolution.UpdatedPackages = []string{"beta"}
	fake := &fakePnpm{update: resolvesTo(removeBaseManifest, testBaseLock)}
	useFakePnpm(t, fake)
	useFakeSizer(t, &fakeSizer{})

	plan, err := PlanDiskAdmission(context.Background(), fixture.resolution)
	if err != nil || plan.Unsupported != "" {
		t.Fatalf("update admission = %+v, %v", plan, err)
	}
	defer plan.Release()
	removed, added, err := plan.JSStage.LockDelta()
	if err != nil || strings.Join(removed, ",") != "beta@2.0.0" || len(added) != 0 {
		t.Fatalf("lock delta = removed %v added %v (%v)", removed, added, err)
	}
	if got := strings.Join(fake.commands("update")[0].Args, " "); got != "update --lockfile-only --workspace-root beta" {
		t.Fatalf("resolver args = %q", got)
	}

	fixture.resolution.DiskJSStage = plan.JSStage
	if _, err := (ExecInstaller{}).Install(context.Background(), fixture.resolution); err != nil {
		t.Fatal(err)
	}
	if fixture.read(t, "package.json") != removeBaseManifest || fixture.read(t, "pnpm-lock.yaml") != testBaseLock {
		t.Fatal("the update must publish the lock and leave the manifest as it was")
	}
}
