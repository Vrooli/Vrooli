package bundle

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	bundlemanifest "github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/manifest"
)

type packagerRuntimeResolver struct{ dir string }

func (r packagerRuntimeResolver) Resolve() (string, error) { return r.dir, nil }

type packagerRuntimeBuilder struct{}

func (packagerRuntimeBuilder) Build(_ string, outPath, _, _, _ string) error {
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(outPath, []byte("runtime"), 0o755)
}

type packagerCLIStager struct{ platforms []string }

func (s *packagerCLIStager) Stage(bundleRoot, platform string) error {
	s.platforms = append(s.platforms, platform)
	return os.MkdirAll(filepath.Join(bundleRoot, "bin"), 0o755)
}

type packagerSizeCalculator struct{}

func (packagerSizeCalculator) Calculate(string) (int64, []LargeFileInfo)        { return 42, nil }
func (packagerSizeCalculator) CheckWarning(int64, []LargeFileInfo) *SizeWarning { return nil }

type directoryCompiler struct{}

func (directoryCompiler) Compile(_ bundlemanifest.Service, _ string, root string) (string, error) {
	dir := filepath.Join(root, "compiled-driver")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	launcher := filepath.Join(dir, "run.sh")
	return dir, os.WriteFile(launcher, []byte("#!/bin/sh\n"), 0o755)
}

type peerOverlayCompiler struct{ called bool }

func (c *peerOverlayCompiler) Compile(_ bundlemanifest.Service, _ string, root string) (string, error) {
	c.called = true
	if _, err := os.Stat(filepath.Join(root, "peers", "helper", "api", "main.go")); err != nil {
		return "", err
	}
	path := filepath.Join(root, "compiled", "helper-api")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	return path, os.WriteFile(path, []byte("compiled"), 0o755)
}

func TestStageServiceBinariesUsesLauncherForDirectoryCompilerOutput(t *testing.T) {
	root := t.TempDir()
	bundleDir := t.TempDir()
	packager := NewPackager(WithServiceCompiler(directoryCompiler{}))
	service := bundlemanifest.Service{ID: "playwright-driver", Type: "api-binary", Build: &bundlemanifest.BuildConfig{Type: "npm"}}
	copied, err := packager.stageServiceBinaries(service, []string{"linux-amd64"}, bundleDir, root, root)
	if err != nil {
		t.Fatalf("stageServiceBinaries() error = %v", err)
	}
	if len(copied) != 1 {
		t.Fatalf("copied = %v, want one launcher", copied)
	}
	info, err := os.Stat(copied[0])
	if err != nil {
		t.Fatalf("staged launcher missing: %v", err)
	}
	if info.IsDir() || filepath.Base(copied[0]) != "run.sh" {
		t.Fatalf("staged path = %q, want regular run.sh", copied[0])
	}
}

func TestPackagerPackageStagesManifestServiceAndRuntime(t *testing.T) {
	app := t.TempDir()
	if err := os.MkdirAll(filepath.Join(app, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(app, "bin", "api"), []byte("api"), 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := bundlemanifest.Manifest{
		SchemaVersion: "v1", Target: "desktop",
		Services: []bundlemanifest.Service{{ID: "api", Type: "api", Binaries: map[string]bundlemanifest.Binary{
			"linux-amd64": {Path: "bin/api"},
		}}},
	}
	manifestPath := filepath.Join(app, "bundle.json")
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, data, 0o644); err != nil {
		t.Fatal(err)
	}
	stager := &packagerCLIStager{}
	result, err := NewPackager(
		WithRuntimeResolver(packagerRuntimeResolver{dir: app}),
		WithRuntimeBuilder(packagerRuntimeBuilder{}),
		WithCLIStager(stager),
		WithSizeCalculator(packagerSizeCalculator{}),
	).Package(app, manifestPath, "electron", []string{"linux-amd64"})
	if err != nil {
		t.Fatalf("package: %v", err)
	}
	if result.TotalSizeBytes != 42 || result.RuntimeBinaries["linux-amd64"] == "" || len(stager.platforms) != 1 {
		t.Fatalf("package result = %#v; staged=%v", result, stager.platforms)
	}
	for _, path := range []string{result.ManifestPath, filepath.Join(result.BundleDir, "bin", "api"), result.RuntimeBinaries["linux-amd64"]} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected staged path %s: %v", path, err)
		}
	}
}

func TestStageEmbeddedPeerSourceInputsCopiesDeclaredInputsWithoutMutatingSources(t *testing.T) {
	repo := t.TempDir()
	app := filepath.Join(repo, "scenarios", "desktop-app")
	peer := filepath.Join(repo, "scenarios", "helper")
	for path, contents := range map[string]string{
		filepath.Join(peer, "api"):                      "source",
		filepath.Join(peer, "ui", "dist", "index.html"): "ui-output",
		filepath.Join(peer, "config", "rules.json"):     "{}",
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	staging := filepath.Join(t.TempDir(), "inputs")
	manifest := &bundlemanifest.Manifest{
		Peers: []bundlemanifest.Peer{{Scenario: "helper", BundlePolicy: "embed"}},
		Services: []bundlemanifest.Service{
			{ID: "helper--api", Type: "api", Build: &bundlemanifest.BuildConfig{SourceDir: "peers/helper/api"}, Assets: []bundlemanifest.Asset{{Path: "peers/helper/config/rules.json"}}},
			{ID: "helper--ui", Type: "ui-bundle", DistRoot: "peers/helper/ui/dist", Binaries: map[string]bundlemanifest.Binary{"linux-amd64": {Path: "peers/helper/ui/dist/index.html"}}},
		},
	}
	if err := NewPackager().stageEmbeddedPeerSourceInputs(manifest, app, staging); err != nil {
		t.Fatalf("stageEmbeddedPeerSourceInputs: %v", err)
	}
	for _, path := range []string{"peers/helper/api", "peers/helper/ui/dist", "peers/helper/ui/dist/index.html", "peers/helper/config/rules.json"} {
		if _, err := os.Stat(filepath.Join(staging, filepath.FromSlash(path))); err != nil {
			t.Errorf("staged input %s missing: %v", path, err)
		}
	}
	for path, want := range map[string]string{
		filepath.Join(peer, "api"):                      "source",
		filepath.Join(peer, "ui", "dist", "index.html"): "ui-output",
		filepath.Join(peer, "config", "rules.json"):     "{}",
	} {
		got, err := os.ReadFile(path)
		if err != nil || string(got) != want {
			t.Errorf("source %s mutated: contents=%q err=%v", path, got, err)
		}
	}
}

func TestStageEmbeddedPeerSourceInputsRejectsMissingRequiredInputAndTraversal(t *testing.T) {
	app := filepath.Join(t.TempDir(), "scenarios", "desktop-app")
	manifest := &bundlemanifest.Manifest{Peers: []bundlemanifest.Peer{{Scenario: "helper", BundlePolicy: "embed"}}, Services: []bundlemanifest.Service{{ID: "helper--api", Build: &bundlemanifest.BuildConfig{SourceDir: "peers/helper/missing"}}}}
	if err := NewPackager().stageEmbeddedPeerSourceInputs(manifest, app, t.TempDir()); err == nil {
		t.Fatal("missing required source input accepted")
	}
	if _, ok := embeddedPeerSourceRelative("helper", "peers/helper/../../outside"); ok {
		t.Fatal("peer source traversal accepted")
	}
}

func TestStageEmbeddedPeerSourceInputsValidatesSymlinkContainment(t *testing.T) {
	tests := []struct {
		name       string
		linkTarget func(peerAPI, scenariosRoot, repoRoot string) string
		wantErr    bool
	}{
		{
			name: "contained relative symlink",
			linkTarget: func(peerAPI, _, _ string) string {
				return "main.go"
			},
		},
		{
			name: "relative sibling escape",
			linkTarget: func(_, _, _ string) string {
				return "../../other/secret.go"
			},
			wantErr: true,
		},
		{
			name: "absolute escape",
			linkTarget: func(_, _, repoRoot string) string {
				return filepath.Join(repoRoot, "outside", "secret.go")
			},
			wantErr: true,
		},
		{
			name: "dangling symlink",
			linkTarget: func(_, _, _ string) string {
				return "missing.go"
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repoRoot := t.TempDir()
			scenariosRoot := filepath.Join(repoRoot, "scenarios")
			app := filepath.Join(scenariosRoot, "desktop-app")
			peer := filepath.Join(scenariosRoot, "helper")
			peerAPI := filepath.Join(peer, "api")
			for path, contents := range map[string]string{
				filepath.Join(peerAPI, "main.go"):                  "package main",
				filepath.Join(scenariosRoot, "other", "secret.go"): "package other",
				filepath.Join(repoRoot, "outside", "secret.go"):    "package outside",
			} {
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.MkdirAll(app, 0o755); err != nil {
				t.Fatal(err)
			}
			linkPath := filepath.Join(peerAPI, "linked.go")
			if err := os.Symlink(tc.linkTarget(peerAPI, scenariosRoot, repoRoot), linkPath); err != nil {
				t.Fatal(err)
			}
			manifest := &bundlemanifest.Manifest{
				Peers: []bundlemanifest.Peer{{Scenario: "helper", BundlePolicy: "embed"}},
				Services: []bundlemanifest.Service{{
					ID:    "helper--api",
					Build: &bundlemanifest.BuildConfig{SourceDir: "peers/helper/api"},
				}},
			}
			staging := t.TempDir()
			err := NewPackager().stageEmbeddedPeerSourceInputs(manifest, app, staging)
			if tc.wantErr {
				if err == nil || !strings.Contains(err.Error(), "embedded peer helper") || !strings.Contains(err.Error(), "linked.go") {
					t.Fatalf("stageEmbeddedPeerSourceInputs error = %v, want peer/path symlink rejection", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("contained symlink rejected: %v", err)
			}
			stagedLink := filepath.Join(staging, "peers", "helper", "api", "linked.go")
			info, err := os.Lstat(stagedLink)
			if err != nil {
				t.Fatalf("staged contained symlink missing: %v", err)
			}
			if info.Mode()&os.ModeSymlink == 0 {
				t.Fatalf("contained symlink was not preserved: mode %v", info.Mode())
			}
		})
	}
}

func TestPackagerRejectsEmbeddedPeerWithoutServices(t *testing.T) {
	manifest := &bundlemanifest.Manifest{
		SchemaVersion: "v1",
		Target:        "desktop",
		Peers:         []bundlemanifest.Peer{{Scenario: "helper", BundlePolicy: "embed"}},
		Services:      []bundlemanifest.Service{{ID: "api", Type: "api", Build: &bundlemanifest.BuildConfig{Type: "go"}}},
	}
	if err := NewPackager().validateManifestForPlatforms(manifest, []string{"linux-amd64"}); err == nil || !strings.Contains(err.Error(), "required embedded peer helper has no bundled services") {
		t.Fatalf("missing embedded peer services error = %v", err)
	}
}

func TestPackageStagesEmbeddedPeerInputsAndRemovesTemporaryOverlay(t *testing.T) {
	repo := t.TempDir()
	scenariosRoot := filepath.Join(repo, "scenarios")
	app := filepath.Join(scenariosRoot, "desktop-app")
	peer := filepath.Join(scenariosRoot, "helper")
	for path, contents := range map[string]string{
		filepath.Join(peer, "api", "main.go"):           "package main",
		filepath.Join(peer, "ui", "dist", "index.html"): "<html></html>",
		filepath.Join(peer, "config", "rules.json"):     "{}",
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(app, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := bundlemanifest.Manifest{
		SchemaVersion: "v1", Target: "desktop",
		Peers: []bundlemanifest.Peer{{Scenario: "helper", BundlePolicy: "embed"}},
		Services: []bundlemanifest.Service{
			{ID: "helper--api", Type: "api-binary", Binaries: map[string]bundlemanifest.Binary{"linux-amd64": {Path: "peers/helper/bin/api"}}, Build: &bundlemanifest.BuildConfig{Type: "go", SourceDir: "peers/helper/api"}, Assets: []bundlemanifest.Asset{{Path: "peers/helper/config/rules.json"}}},
			{ID: "helper--ui", Type: "ui-bundle", Binaries: map[string]bundlemanifest.Binary{"linux-amd64": {Path: "peers/helper/ui/dist/index.html"}}, DistRoot: "peers/helper/ui/dist"},
		},
	}
	manifestPath := filepath.Join(app, "bundle.json")
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, data, 0o644); err != nil {
		t.Fatal(err)
	}
	compiler := &peerOverlayCompiler{}
	packager := NewPackager(
		WithRuntimeResolver(packagerRuntimeResolver{dir: app}),
		WithRuntimeBuilder(packagerRuntimeBuilder{}),
		WithCLIStager(&packagerCLIStager{}),
		WithSizeCalculator(packagerSizeCalculator{}),
		WithServiceCompiler(compiler),
	)
	result, err := packager.Package(app, manifestPath, "electron", []string{"linux-amd64"})
	if err != nil {
		t.Fatalf("Package: %v", err)
	}
	if !compiler.called {
		t.Fatal("peer source overlay was not used for compilation")
	}
	for _, path := range []string{
		filepath.Join(result.BundleDir, "peers", "helper", "bin", "api"),
		filepath.Join(result.BundleDir, "peers", "helper", "ui", "dist", "index.html"),
		filepath.Join(result.BundleDir, "peers", "helper", "config", "rules.json"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("packaged peer artifact %s missing: %v", path, err)
		}
	}
	overlays, err := filepath.Glob(filepath.Join(app, "platforms", "electron", ".peer-source-inputs-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(overlays) != 0 {
		t.Fatalf("temporary peer overlays remain after Package: %v", overlays)
	}
	if _, err := os.Stat(filepath.Join(peer, "api", "main.go")); err != nil {
		t.Fatalf("peer source was mutated or removed: %v", err)
	}
}

func TestPackageCompilesEmbeddedGoPeerWithLocalReplaceFromOverlay(t *testing.T) {
	repo := t.TempDir()
	scenariosRoot := filepath.Join(repo, "scenarios")
	app := filepath.Join(scenariosRoot, "desktop-app")
	peerAPI := filepath.Join(scenariosRoot, "helper", "api")
	shared := filepath.Join(repo, "packages", "helperlib")
	files := map[string]string{
		filepath.Join(peerAPI, "go.mod"): `module example.com/helper

go 1.26.0

require example.com/helperlib v0.0.0
require example.com/root v0.0.0

replace example.com/helperlib => ../../../packages/helperlib
replace example.com/root => ../../..
`,
		filepath.Join(peerAPI, "main.go"): `package main

import (
	"fmt"
	"example.com/helperlib"
	"example.com/root/packages/authority"
)

func main() { fmt.Print(helperlib.Value()+authority.Value()) }
`,
		filepath.Join(repo, "go.mod"):                                "module example.com/root\n\ngo 1.26.0\n",
		filepath.Join(repo, "packages", "authority", "authority.go"): "package authority\nfunc Value()string{return \"-root-module\"}\n",
		filepath.Join(shared, "go.mod"): `module example.com/helperlib

go 1.26.0
`,
		filepath.Join(shared, "helper.go"): `package helperlib

func Value() string { return "compiled-from-local-replace" }
`,
	}
	for path, contents := range files {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(app, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := bundlemanifest.Manifest{
		SchemaVersion: "v1",
		Target:        "desktop",
		Peers:         []bundlemanifest.Peer{{Scenario: "helper", BundlePolicy: "embed"}},
		Services: []bundlemanifest.Service{{
			ID:       "helper--api",
			Type:     "api-binary",
			Binaries: map[string]bundlemanifest.Binary{"linux-amd64": {Path: "peers/helper/bin/api"}},
			Build: &bundlemanifest.BuildConfig{
				Type:          "go",
				SourceDir:     "peers/helper/api",
				OutputPattern: "bin/{{platform}}/{{ext}}helper",
				Env:           map[string]string{"GOWORK": "off", "GOPROXY": "off", "GOSUMDB": "off"},
			},
		}},
	}
	manifestPath := filepath.Join(app, "bundle.json")
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, data, 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := NewPackager(
		WithRuntimeResolver(packagerRuntimeResolver{dir: app}),
		WithRuntimeBuilder(packagerRuntimeBuilder{}),
		WithCLIStager(&packagerCLIStager{}),
		WithSizeCalculator(packagerSizeCalculator{}),
	).Package(app, manifestPath, "electron", []string{"linux-amd64"})
	if err != nil {
		t.Fatalf("Package with default Go compiler: %v", err)
	}
	binary := filepath.Join(result.BundleDir, "peers", "helper", "bin", "api")
	if _, err := os.Stat(binary); err != nil {
		t.Fatalf("compiled peer binary missing: %v", err)
	}
	output, err := exec.Command(binary).CombinedOutput()
	if err != nil {
		t.Fatalf("run compiled peer binary: %v: %s", err, output)
	}
	if string(output) != "compiled-from-local-replace-root-module" {
		t.Fatalf("compiled peer output = %q, want local replacement dependency output", output)
	}
	if _, err := os.Stat(filepath.Join(peerAPI, "go.mod")); err != nil {
		t.Fatalf("source module was mutated or removed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(shared, "helper.go")); err != nil {
		t.Fatalf("local replacement source was mutated or removed: %v", err)
	}
}

func TestStageManifestCatalogCopiesOnlyDeclarativeManifests(t *testing.T) {
	repo := t.TempDir()
	app := filepath.Join(repo, "scenarios", "onboarding")
	for path, content := range map[string]string{
		"scenarios/onboarding/.vrooli/service.json":           `{"service":{"name":"onboarding"}}`,
		"scenarios/vrooli-onboarding/.vrooli/service.json":    `{"service":{"name":"vrooli-onboarding"}}`,
		"scenarios/vrooli-onboarding/profiles/local-use.json": `{"id":"local-use"}`,
		"scenarios/vrooli-onboarding/profiles/README.txt":     "must not ship",
		"scenarios/other/.vrooli/service.json":                `{"service":{"name":"other"}}`,
		"resources/openrouter/resource.json":                  `{"credentials":{"descriptors":[]}}`,
		"resources/openrouter/config/private.json":            `{"must_not_ship":true}`,
	} {
		full := filepath.Join(repo, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	bundleDir := filepath.Join(t.TempDir(), "bundle")
	copied, err := NewPackager().stageManifestCatalog(app, bundleDir)
	if err != nil {
		t.Fatalf("stage manifest catalog: %v", err)
	}
	if len(copied) != 5 {
		t.Fatalf("copied %d catalog files, want 5: %v", len(copied), copied)
	}
	for _, path := range []string{
		filepath.Join(bundleDir, "catalog", "scenarios", "onboarding", ".vrooli", "service.json"),
		filepath.Join(bundleDir, "catalog", "scenarios", "other", ".vrooli", "service.json"),
		filepath.Join(bundleDir, "catalog", "resources", "openrouter", "resource.json"),
		filepath.Join(bundleDir, "catalog", "scenarios", "vrooli-onboarding", "profiles", "local-use.json"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected catalog file %s: %v", path, err)
		}
	}
	if _, err := os.Stat(filepath.Join(bundleDir, "catalog", "scenarios", "vrooli-onboarding", "profiles", "README.txt")); !os.IsNotExist(err) {
		t.Fatalf("non-declarative onboarding profile file shipped: %v", err)
	}
	if _, err := os.Stat(filepath.Join(bundleDir, "catalog", "resources", "openrouter", "config", "private.json")); !os.IsNotExist(err) {
		t.Fatalf("non-manifest configuration shipped in catalog: %v", err)
	}
}

func TestStageManifestCatalogCopiesHostRequirementCatalogs(t *testing.T) {
	repo := t.TempDir()
	app := filepath.Join(repo, "scenarios", "onboarding")
	for path, content := range map[string]string{
		"scenarios/onboarding/.vrooli/service.json":         `{"service":{"name":"onboarding"}}`,
		"resources/postgres/resource.json":                  `{"name":"postgres"}`,
		"internal/tools/tmux/tool.json":                     `{"name":"tmux"}`,
		"internal/tools/tmux/handler.go":                    `package tmux`,
		"internal/safeguards/nat-protection/safeguard.json": `{"name":"nat-protection"}`,
		"internal/safeguards/nat-protection/handler.go":     `package natprotection`,
	} {
		full := filepath.Join(repo, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	bundleDir := filepath.Join(t.TempDir(), "bundle")
	copied, err := NewPackager().stageManifestCatalog(app, bundleDir)
	if err != nil {
		t.Fatalf("stage manifest catalog: %v", err)
	}
	if len(copied) != 4 {
		t.Fatalf("copied %d catalog files, want 4: %v", len(copied), copied)
	}
	for _, path := range []string{
		"catalog/scenarios/onboarding/.vrooli/service.json",
		"catalog/resources/postgres/resource.json",
		"catalog/internal/tools/tmux/tool.json",
		"catalog/internal/safeguards/nat-protection/safeguard.json",
	} {
		if _, err := os.Stat(filepath.Join(bundleDir, path)); err != nil {
			t.Fatalf("expected staged catalog path %s: %v", path, err)
		}
	}
	if _, err := os.Stat(filepath.Join(bundleDir, "catalog/internal/tools/tmux/handler.go")); !os.IsNotExist(err) {
		t.Fatalf("executable tool implementation shipped in catalog: %v", err)
	}
}

func TestStageManifestCatalogRestrictsToUnionRequirements(t *testing.T) {
	repo := t.TempDir()
	app := filepath.Join(repo, "scenarios", "onboarding")
	for path, content := range map[string]string{
		"scenarios/onboarding/.vrooli/service.json": `{"service":{"name":"onboarding"}}`,
		"scenarios/other/.vrooli/service.json":      `{"service":{"name":"other"}}`,
		"resources/needed/resource.json":            `{"name":"needed"}`,
		"resources/unused/resource.json":            `{"name":"unused"}`,
	} {
		full := filepath.Join(repo, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	bundleDir := filepath.Join(t.TempDir(), "bundle")
	copied, err := NewPackager().stageManifestCatalogForRequirements(app, bundleDir, []string{
		"catalog/scenarios/onboarding", "catalog/resources/needed",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(copied) != 2 {
		t.Fatalf("copied %d union files, want 2: %v", len(copied), copied)
	}
	for _, path := range []string{
		"catalog/scenarios/onboarding/.vrooli/service.json",
		"catalog/resources/needed/resource.json",
	} {
		if _, err := os.Stat(filepath.Join(bundleDir, path)); err != nil {
			t.Fatalf("required union path missing %s: %v", path, err)
		}
	}
	for _, path := range []string{
		"catalog/scenarios/other/.vrooli/service.json",
		"catalog/resources/unused/resource.json",
	} {
		if _, err := os.Stat(filepath.Join(bundleDir, path)); !os.IsNotExist(err) {
			t.Fatalf("unselected union path was staged %s: %v", path, err)
		}
	}
}

func TestResolvePackagePathsAndManifestValidationRejectUnsafeInputs(t *testing.T) {
	if _, err := resolvePackagePaths("", "manifest.json", nil); err == nil {
		t.Fatal("expected required path error")
	}
	app := t.TempDir()
	manifestPath := filepath.Join(app, "bundle.json")
	if err := os.WriteFile(manifestPath, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	paths, err := resolvePackagePaths(app, manifestPath, []string{filepath.Join(app, "out")})
	if err != nil || paths.outputRootAbs != filepath.Join(app, "out") {
		t.Fatalf("resolved paths = %#v, %v", paths, err)
	}
	packager := NewPackager()
	if err := packager.validateManifestForPlatforms(&bundlemanifest.Manifest{SchemaVersion: "v1", Target: "server"}, []string{"linux"}); err == nil {
		t.Fatal("expected non-desktop target error")
	}
	if _, err := resolveManifestPath(&defaultFileOperations{}, app, "../escape"); err == nil {
		t.Fatal("expected manifest traversal rejection")
	}
}

func TestPackagerStagesUIAndServiceAssetsFromScenarioRoot(t *testing.T) {
	app := t.TempDir()
	bundleDir := filepath.Join(t.TempDir(), "bundle")
	for path, content := range map[string]string{
		"ui/dist/index.html": "<html></html>",
		"ui/dist/app.js":     "console.log('ready')",
		"assets/config.json": `{"ready":true}`,
	} {
		full := filepath.Join(app, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	packager := NewPackager()
	ui := bundlemanifest.Service{ID: "ui", Type: "ui", Binaries: map[string]bundlemanifest.Binary{"linux": {Path: "ui/dist/index.html"}}, Assets: []bundlemanifest.Asset{{Path: "ui/dist/app.js"}}}
	copied, err := packager.stageUIService(ui, app, bundleDir, app)
	if err != nil || len(copied) != 2 {
		t.Fatalf("stage UI = %v, %v", copied, err)
	}
	service := bundlemanifest.Service{ID: "api", Type: "api", Assets: []bundlemanifest.Asset{{Path: "assets/config.json"}}}
	assets, err := packager.stageServiceAssets(service, bundleDir, app)
	if err != nil || len(assets) != 1 {
		t.Fatalf("stage assets = %v, %v", assets, err)
	}
	for _, path := range append(copied, assets...) {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("staged path missing %s: %v", path, err)
		}
	}
}

func TestPackagerStageServiceBinariesReportsMissingOrUnsafeInputs(t *testing.T) {
	packager := NewPackager()
	root := t.TempDir()
	_, err := packager.stageServiceBinaries(bundlemanifest.Service{ID: "api", Type: "api", Binaries: map[string]bundlemanifest.Binary{"linux": {Path: "bin/missing"}}}, []string{"linux"}, t.TempDir(), root, root)
	if err == nil {
		t.Fatal("expected missing binary error")
	}
	_, err = packager.stageServiceAssets(bundlemanifest.Service{ID: "api", Assets: []bundlemanifest.Asset{{Path: "../escape"}}}, t.TempDir(), root)
	if err == nil {
		t.Fatal("expected asset traversal error")
	}
}
