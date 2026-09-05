package deployments

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"deployment-manager/build"
	"deployment-manager/bundles"
)

func TestResolveBuildAndInstallerPlatforms(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{"default builds", nil, []string{"linux-x64", "linux-arm64", "darwin-x64", "darwin-arm64", "win-x64"}},
		{"aliases", []string{"linux", "mac", "windows"}, []string{"linux-x64", "linux-arm64", "darwin-x64", "darwin-arm64", "win-x64"}},
		{"deduplicates", []string{"linux-x64", "linux", "linux-x64"}, []string{"linux-x64", "linux-arm64"}},
		{"unknown falls back", []string{"not-a-platform"}, []string{"linux-x64", "linux-arm64", "darwin-x64", "darwin-arm64", "win-x64"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveBuildPlatforms(tc.input)
			if strings.Join(got, ",") != strings.Join(tc.want, ",") {
				t.Fatalf("resolveBuildPlatforms(%v) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
	for _, tc := range []struct {
		input []string
		want  []string
	}{
		{nil, []string{"win", "mac", "linux"}},
		{[]string{"windows", "darwin-arm64", "linux-x64"}, []string{"win", "mac", "linux"}},
		{[]string{"unknown"}, []string{"win", "mac", "linux"}},
		{[]string{"mac", "mac"}, []string{"mac"}},
	} {
		got := resolveInstallerTargets(tc.input)
		if strings.Join(got, ",") != strings.Join(tc.want, ",") {
			t.Errorf("resolveInstallerTargets(%v) = %v, want %v", tc.input, got, tc.want)
		}
	}
}

func TestAssetMetadataAndUIExpansion(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "ui", "dist", "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "ui", "dist", "index.html"), []byte("index"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "ui", "dist", "assets", "app.js"), []byte("app"), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := &bundles.Manifest{Services: []bundles.ServiceEntry{
		{ID: "demo-api", Type: "api-binary", Assets: []bundles.Asset{{Path: "missing.txt"}}},
		{ID: "demo-ui", Type: "ui-bundle"},
	}}
	if err := populateAssetMetadata(manifest, root); err == nil || !strings.Contains(err.Error(), "missing.txt") {
		t.Fatalf("missing asset error = %v", err)
	}
	api := manifest.Services[0]
	if api.Env["CORS_ALLOWED_ORIGINS"] != "*" || api.Env["UI_PORT"] != "${ui.ui}" {
		t.Fatalf("API environment = %#v", api.Env)
	}
	ui := manifest.Services[1]
	if len(ui.Assets) != 2 {
		t.Fatalf("expanded UI assets = %#v", ui.Assets)
	}
	for _, asset := range ui.Assets {
		if asset.SHA256 == "" || asset.SizeBytes == 0 || !strings.HasPrefix(asset.Path, "ui/dist/") {
			t.Errorf("incomplete asset metadata: %#v", asset)
		}
	}
	if got, err := hashFileSHA256(filepath.Join(root, "ui", "dist", "index.html")); err != nil || got == "" {
		t.Fatalf("hashFileSHA256() = %q, %v", got, err)
	}
	if _, err := hashFileSHA256(filepath.Join(root, "missing")); err == nil {
		t.Fatal("hashFileSHA256(missing) returned nil error")
	}
	if err := populateAssetMetadata(nil, root); err == nil {
		t.Fatal("nil manifest returned nil error")
	}
}

func TestBundleBinaryCopyAndInstallerDiscovery(t *testing.T) {
	root := t.TempDir()
	manifestDir := filepath.Join(root, "manifest")
	bundleDir := filepath.Join(root, "bundle")
	if err := os.MkdirAll(manifestDir, 0o755); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(manifestDir, "api")
	if err := os.WriteFile(source, []byte("binary"), 0o744); err != nil {
		t.Fatal(err)
	}
	manifest := &bundles.Manifest{Services: []bundles.ServiceEntry{{ID: "api", Binaries: map[string]bundles.ServiceBinary{
		"linux-x64": {Path: "api"}, "win-x64": {Path: "missing.exe"},
	}}}}
	missing, err := copyBuiltBinariesToBundle(manifest, manifestDir, bundleDir, []string{"linux-x64", "win-x64"})
	if err != nil || len(missing) != 1 || missing[0] != "api:win-x64" {
		t.Fatalf("copy result = %v, %v", missing, err)
	}
	data, err := os.ReadFile(filepath.Join(bundleDir, "bin", "linux-x64", "api"))
	if err != nil || string(data) != "binary" {
		t.Fatalf("copied binary = %q, %v", data, err)
	}
	if _, err := copyBuiltBinariesToBundle(nil, manifestDir, bundleDir, nil); err == nil {
		t.Fatal("nil manifest returned nil error")
	}

	dist := filepath.Join(root, "dist")
	if err := os.MkdirAll(dist, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"setup.exe", "package.msi", "release.dmg", "release.pkg", "release.zip", "app.AppImage", "app.deb", "app.tar.gz"} {
		if err := os.WriteFile(filepath.Join(dist, name), []byte(name), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, platform := range []string{"win", "mac", "linux"} {
		if got, err := findInstallerArtifact(dist, platform); err != nil || got == "" {
			t.Errorf("findInstallerArtifact(%s) = %q, %v", platform, got, err)
		}
	}
	if _, err := findInstallerArtifact(dist, "unknown"); err == nil {
		t.Fatal("unknown platform returned nil error")
	}
	if _, err := findInstallerArtifact(t.TempDir(), "linux"); err == nil {
		t.Fatal("missing artifact returned nil error")
	}
	if err := copyFilePreserveMode(source, filepath.Join(root, "copy")); err != nil {
		t.Fatal(err)
	}
	if err := copyFilePreserveMode(filepath.Join(root, "missing"), filepath.Join(root, "copy2")); err == nil {
		t.Fatal("missing source returned nil error")
	}
}

func TestCLIServicePruningAndCommandLogging(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "cli"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "cli", "main.go"), []byte("package main"), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := &bundles.Manifest{Services: []bundles.ServiceEntry{
		{ID: "api", Type: "api-binary"},
		{ID: "demo-cli", Build: &bundles.BuildConfig{Type: "go", SourceDir: "cli"}},
		{ID: "legacy-cli", Build: &bundles.BuildConfig{Type: "npm", SourceDir: "cli"}},
	}}
	pruned, err := pruneNonCrossPlatformCLIs(manifest, root)
	if err != nil || len(pruned) != 1 || pruned[0] != "legacy-cli" || len(manifest.Services) != 2 {
		t.Fatalf("pruned = %v, err=%v services=%d", pruned, err, len(manifest.Services))
	}
	if !isCLIService(bundles.ServiceEntry{ID: "worker-cli"}) || !isCLIService(bundles.ServiceEntry{Build: &bundles.BuildConfig{SourceDir: "cli"}}) || isCLIService(bundles.ServiceEntry{ID: "api"}) {
		t.Fatal("isCLIService classification incorrect")
	}
	if !isCrossPlatformCLIBuild(manifest.Services[1], root) || isCrossPlatformCLIBuild(bundles.ServiceEntry{}, root) {
		t.Fatal("isCrossPlatformCLIBuild classification incorrect")
	}
	logs := 0
	if err := runCommandLogged(context.Background(), "sh", []string{"-c", "printf ok"}, root, func(string, map[string]interface{}) { logs++ }); err != nil || logs != 1 {
		t.Fatalf("successful command = %v logs=%d", err, logs)
	}
	if err := runCommandLogged(context.Background(), "sh", []string{"-c", "exit 2"}, root, func(string, map[string]interface{}) { logs++ }); err == nil {
		t.Fatal("failed command returned nil error")
	}
}

func TestUpdateManifestBinaryPathsAndCrossPlatformBuildDetection(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "rust-cli"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "rust-cli", "main.rs"), []byte("fn main() {}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "rust-cli", "Cargo.toml"), []byte("[package]"), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := &bundles.Manifest{Services: []bundles.ServiceEntry{
		{ID: "api", Build: &bundles.BuildConfig{Type: "go"}, Binaries: map[string]bundles.ServiceBinary{
			"linux-x64": {Path: "old-linux", Args: []string{"--serve"}},
			"win-x64":   {Path: "old-win"},
		}},
		{ID: "embedded"},
	}}
	updateManifestBinaryPaths(manifest, []build.BuildResult{
		{Platform: "linux-x64", OutputPath: filepath.Join(root, "bin", "api"), Success: true},
		{Platform: "darwin-arm64", OutputPath: filepath.Join(root, "bin", "api-arm"), Success: true},
		{Platform: "win-x64", OutputPath: filepath.Join(root, "bin", "api.exe"), Success: false},
	}, root, filepath.Join(root, "manifest"))
	if manifest.Services[0].Build != nil || manifest.Services[0].Binaries["linux-x64"].Path == "old-linux" || manifest.Services[0].Binaries["darwin-arm64"].Path == "" {
		t.Fatalf("binary paths were not updated: %+v", manifest.Services[0])
	}
	if !isCrossPlatformCLIBuild(bundles.ServiceEntry{Build: &bundles.BuildConfig{Type: "rust", SourceDir: "rust-cli"}}, root) {
		t.Fatal("rust CLI with Cargo.toml should be cross-platform")
	}
	emptyRoot := t.TempDir()
	if isCrossPlatformCLIBuild(bundles.ServiceEntry{Build: &bundles.BuildConfig{Type: "rust", SourceDir: "rust-cli"}}, emptyRoot) {
		t.Fatal("missing rust source should not be cross-platform")
	}
	if isCrossPlatformCLIBuild(bundles.ServiceEntry{Build: &bundles.BuildConfig{Type: "python", SourceDir: "rust-cli"}}, root) {
		t.Fatal("unknown CLI build type should not be cross-platform")
	}
}
