package bundle

import (
	"encoding/json"
	"os"
	"path/filepath"
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

func TestStageManifestCatalogCopiesOnlyDeclarativeManifests(t *testing.T) {
	repo := t.TempDir()
	app := filepath.Join(repo, "scenarios", "onboarding")
	for path, content := range map[string]string{
		"scenarios/onboarding/.vrooli/service.json": `{"service":{"name":"onboarding"}}`,
		"scenarios/other/.vrooli/service.json":      `{"service":{"name":"other"}}`,
		"resources/openrouter/resource.json":        `{"credentials":{"descriptors":[]}}`,
		"resources/openrouter/config/private.json":  `{"must_not_ship":true}`,
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
	if len(copied) != 3 {
		t.Fatalf("copied %d catalog files, want 3: %v", len(copied), copied)
	}
	for _, path := range []string{
		filepath.Join(bundleDir, "catalog", "scenarios", "onboarding", ".vrooli", "service.json"),
		filepath.Join(bundleDir, "catalog", "scenarios", "other", ".vrooli", "service.json"),
		filepath.Join(bundleDir, "catalog", "resources", "openrouter", "resource.json"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected catalog file %s: %v", path, err)
		}
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
