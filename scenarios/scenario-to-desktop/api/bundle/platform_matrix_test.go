package bundle

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	bundlemanifest "github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/manifest"
)

func TestPlanPlatformMatrixEnumeratesSupportedAndUnsupportedCases(t *testing.T) {
	root := t.TempDir()
	for path, contents := range map[string]string{
		"bin/linux-api":      "linux-api",
		"bin/darwin-arm-api": "darwin-arm-api",
	} {
		fullPath := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte(contents), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	manifest := &bundlemanifest.Manifest{
		SchemaVersion: "v1", Target: "desktop",
		Services: []bundlemanifest.Service{
			{ID: "api", Type: "api", Binaries: map[string]bundlemanifest.Binary{
				"linux-x64":     {Path: "bin/linux-api"},
				"macos-aarch64": {Path: "bin/darwin-arm-api"},
			}},
			{ID: "worker", Type: "api", Build: &bundlemanifest.BuildConfig{Type: "go"}},
			{ID: "ui", Type: "ui-bundle"},
		},
	}

	packager := NewPackager()
	matrix, selected, err := packager.planPlatformMatrix(manifest, []string{"linux-amd64", "darwin-arm64"}, root)
	if err != nil {
		t.Fatalf("planPlatformMatrix: %v", err)
	}
	if strings.Join(selected, ",") != "darwin-arm64,linux-amd64" {
		t.Fatalf("selected targets = %v", selected)
	}
	if matrix.SchemaVersion != "v1" || len(matrix.Targets) != 6 {
		t.Fatalf("matrix header/target count = %q/%d, want v1/6", matrix.SchemaVersion, len(matrix.Targets))
	}

	byKey := make(map[string]PlatformMatrixTarget, len(matrix.Targets))
	for _, target := range matrix.Targets {
		byKey[target.Key] = target
	}
	for _, key := range desktopPlatformKeys {
		if _, exists := byKey[key]; !exists {
			t.Errorf("finite matrix omitted target %s", key)
		}
	}
	if got := byKey["linux-amd64"].Services[0].Disposition; got != "prebuilt-available" {
		t.Errorf("Linux API disposition = %q, want prebuilt-available", got)
	}
	if got := byKey["darwin-arm64"].Services[0].Disposition; got != "prebuilt-available" {
		t.Errorf("macOS ARM API disposition = %q, want prebuilt-available", got)
	}
	if got := byKey["windows-amd64"].Services[0].Disposition; got != "unsupported" {
		t.Errorf("Windows API disposition = %q, want unsupported", got)
	}
	if got := byKey["windows-amd64"].Services[0].Reason; !strings.Contains(got, "no matching binary") {
		t.Errorf("Windows API reason = %q", got)
	}
	if got := byKey["linux-arm64"].Services[1].Disposition; got != "build-configured" {
		t.Errorf("Linux ARM worker disposition = %q, want build-configured", got)
	}
	if got := byKey["linux-arm64"].Services[2].Disposition; got != "shared-ui" {
		t.Errorf("Linux ARM UI disposition = %q, want shared-ui", got)
	}
	if byKey["linux-amd64"].Services[0].Bundled || byKey["windows-amd64"].Selected {
		t.Fatal("planning should not claim artifacts bundled or select unrequested rows")
	}
}

func TestResolveBinaryForPlatformNeverFallsAcrossArchitectures(t *testing.T) {
	resolver := &defaultPlatformResolver{}
	service := bundlemanifest.Service{Binaries: map[string]bundlemanifest.Binary{
		"linux-arm64": {Path: "bin/linux-arm64"},
		"linux-x64":   {Path: "bin/linux-x64"},
	}}
	for _, target := range []struct{ platform, want string }{
		{"linux-amd64", "bin/linux-x64"},
		{"linux-arm64", "bin/linux-arm64"},
	} {
		binary, ok := resolver.ResolveBinaryForPlatform(service, target.platform)
		if !ok || binary.Path != target.want {
			t.Errorf("ResolveBinaryForPlatform(%s) = %q, %v; want %q", target.platform, binary.Path, ok, target.want)
		}
	}
	if _, ok := resolver.ResolveBinaryForPlatform(service, "windows-amd64"); ok {
		t.Fatal("resolver returned a Linux artifact for a Windows target")
	}
}

func TestPackageRejectsUnsupportedPlatformBeforeChangingExistingBundle(t *testing.T) {
	app := t.TempDir()
	manifest := bundlemanifest.Manifest{
		SchemaVersion: "v1", Target: "desktop",
		Services: []bundlemanifest.Service{{
			ID: "api", Type: "api", Build: &bundlemanifest.BuildConfig{Type: "go"},
			Binaries: map[string]bundlemanifest.Binary{"linux-amd64": {Path: "api"}},
		}},
	}
	manifestPath := filepath.Join(app, "bundle.json")
	payload, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, payload, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(app, "api"), []byte("saved"), 0o755); err != nil {
		t.Fatal(err)
	}

	outputRoot := t.TempDir()
	bundleDir := filepath.Join(outputRoot, "platforms", "electron", "bundle")
	if err := os.MkdirAll(bundleDir, 0o755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(bundleDir, "prior-artifact.txt")
	if err := os.WriteFile(marker, []byte("preserve-me"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = NewPackager().Package(app, manifestPath, "electron", []string{"freebsd-amd64"}, outputRoot)
	if err == nil || !strings.Contains(err.Error(), "unsupported desktop target") {
		t.Fatalf("unsupported platform error = %v", err)
	}
	got, readErr := os.ReadFile(marker)
	if readErr != nil || string(got) != "preserve-me" {
		t.Fatalf("prior bundle artifact changed: contents=%q err=%v", got, readErr)
	}
}

func TestPackageEmitsMatrixForBuiltTargetsAndKeepsOtherPlatformCases(t *testing.T) {
	app := t.TempDir()
	apiPath := filepath.Join(app, "bin", "linux-api")
	if err := os.MkdirAll(filepath.Dir(apiPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(apiPath, []byte("linux-api"), 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := bundlemanifest.Manifest{
		SchemaVersion: "v1", Target: "desktop",
		Services: []bundlemanifest.Service{
			{ID: "api", Type: "api", Binaries: map[string]bundlemanifest.Binary{"linux-amd64": {Path: "bin/linux-api"}}, Build: &bundlemanifest.BuildConfig{Type: "go"}},
			{ID: "worker", Type: "api", Build: &bundlemanifest.BuildConfig{Type: "go"}},
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

	result, err := NewPackager(
		WithRuntimeResolver(packagerRuntimeResolver{dir: app}),
		WithRuntimeBuilder(packagerRuntimeBuilder{}),
		WithCLIStager(&packagerCLIStager{}),
		WithServiceCompiler(matrixCompiler{}),
		WithSizeCalculator(packagerSizeCalculator{}),
	).Package(app, manifestPath, "electron", []string{"linux-x64", "win-amd64"})
	if err != nil {
		t.Fatalf("Package: %v", err)
	}

	matrixPath := filepath.Join(result.BundleDir, "platform-matrix.json")
	if !containsString(result.CopiedArtifacts, matrixPath) {
		t.Fatalf("copied artifacts omit emitted matrix %s: %v", matrixPath, result.CopiedArtifacts)
	}
	payload, err := os.ReadFile(matrixPath)
	if err != nil {
		t.Fatal(err)
	}
	var matrix PlatformMatrix
	if err := json.Unmarshal(payload, &matrix); err != nil {
		t.Fatalf("decode emitted matrix: %v", err)
	}
	if len(matrix.Targets) != 6 {
		t.Fatalf("emitted target count = %d, want 6", len(matrix.Targets))
	}
	byKey := make(map[string]PlatformMatrixTarget, len(matrix.Targets))
	for _, target := range matrix.Targets {
		byKey[target.Key] = target
	}
	for _, key := range []string{"linux-amd64", "windows-amd64"} {
		row := byKey[key]
		if !row.Selected || row.Runtime != "built" || row.RuntimeArtifact == "" || len(row.Services) != 2 || !row.Services[0].Bundled || !row.Services[1].Bundled {
			t.Errorf("selected target %s was not reported as fully bundled: %+v", key, row)
		}
		for _, service := range row.Services {
			if service.ArtifactPath == "" {
				t.Errorf("selected service %s/%s omitted its staged artifact path: %+v", key, service.ID, service)
				continue
			}
			if _, err := os.Stat(filepath.Join(result.BundleDir, filepath.FromSlash(service.ArtifactPath))); err != nil {
				t.Errorf("matrix artifact %s/%s is missing: %v", key, service.ID, err)
			}
		}
	}
	if byKey["darwin-arm64"].Selected || byKey["darwin-arm64"].Services[0].Bundled {
		t.Errorf("unrequested macOS ARM case claimed inclusion: %+v", byKey["darwin-arm64"])
	}
}

type matrixCompiler struct{}

func (matrixCompiler) Compile(service bundlemanifest.Service, platform, root string) (string, error) {
	name := strings.ReplaceAll(service.ID+"-"+platform, "/", "-")
	path := filepath.Join(root, ".compiled", name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, []byte(name), 0o755); err != nil {
		return "", err
	}
	return path, nil
}
