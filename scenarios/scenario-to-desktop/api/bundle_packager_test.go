package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	bundlemanifest "scenario-to-desktop-runtime/manifest"
)

func TestCopyPathPreservesExecutablesAndDirectories(t *testing.T) {
	root := t.TempDir()

	srcDir := filepath.Join(root, "src", "resources", "playwright")
	nested := filepath.Join(srcDir, "chromium")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir src: %v", err)
	}

	// Executable asset inside the directory tree.
	chromeSrc := filepath.Join(nested, "chrome")
	if err := os.WriteFile(chromeSrc, []byte("fake-chrome"), 0o755); err != nil {
		t.Fatalf("write source chrome: %v", err)
	}

	dstDir := filepath.Join(root, "bundle", "resources", "playwright")
	if err := copyPath(srcDir, dstDir); err != nil {
		t.Fatalf("copyPath: %v", err)
	}

	chromeDst := filepath.Join(dstDir, "chromium", "chrome")
	info, err := os.Stat(chromeDst)
	if err != nil {
		t.Fatalf("stat copied chrome: %v", err)
	}
	if info.Mode()&0o111 == 0 {
		t.Fatalf("expected executable bit preserved, got mode %v", info.Mode())
	}

	// Direct file copy still works and preserves mode.
	fileDst := filepath.Join(root, "bundle", "bin", "shim")
	if err := copyPath(chromeSrc, fileDst); err != nil {
		t.Fatalf("copyPath file: %v", err)
	}
	fileInfo, err := os.Stat(fileDst)
	if err != nil {
		t.Fatalf("stat copied file: %v", err)
	}
	if fileInfo.Mode()&0o111 == 0 {
		t.Fatalf("expected executable bit preserved on file copy, got mode %v", fileInfo.Mode())
	}
}

func TestEnsureBundleExtraResourcesAddsEntry(t *testing.T) {
	appDir := t.TempDir()
	pkgPath := filepath.Join(appDir, "package.json")

	original := map[string]any{
		"build": map[string]any{
			"extraResources": []any{
				map[string]any{"from": "app", "to": "app"},
			},
		},
	}
	data, _ := json.Marshal(original)
	if err := os.WriteFile(pkgPath, data, 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}

	if err := ensureBundleExtraResources(appDir); err != nil {
		t.Fatalf("ensureBundleExtraResources: %v", err)
	}

	updatedRaw, err := os.ReadFile(pkgPath)
	if err != nil {
		t.Fatalf("read updated package.json: %v", err)
	}

	var updated map[string]any
	if err := json.Unmarshal(updatedRaw, &updated); err != nil {
		t.Fatalf("parse updated package.json: %v", err)
	}

	build, _ := updated["build"].(map[string]any)
	extras, _ := build["extraResources"].([]any)

	foundBundle := false
	for _, entry := range extras {
		m, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		if m["from"] == "bundle" || m["to"] == "bundle" {
			foundBundle = true
			break
		}
	}
	if !foundBundle {
		t.Fatalf("bundle extraResources entry not added: %+v", extras)
	}
}

func TestBundlePackagerBuildsRuntimeViaInjectedBuilder(t *testing.T) {
	manifestRoot := t.TempDir()
	appDir := filepath.Join(manifestRoot, "app")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatalf("mkdir app dir: %v", err)
	}

	// package.json needed for ensureBundleExtraResources
	if err := os.WriteFile(filepath.Join(appDir, "package.json"), []byte(`{"name":"demo","build":{}}`), 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}

	binaryRel := filepath.Join("bin", "api")
	binaryAbs := filepath.Join(manifestRoot, binaryRel)
	if err := os.MkdirAll(filepath.Dir(binaryAbs), 0o755); err != nil {
		t.Fatalf("mkdir binary dir: %v", err)
	}
	if err := os.WriteFile(binaryAbs, []byte("api"), 0o755); err != nil {
		t.Fatalf("write binary: %v", err)
	}

	manifest := bundlemanifest.Manifest{
		SchemaVersion: "0.1",
		Target:        "desktop",
		App:           bundlemanifest.App{Name: "demo", Version: "1.0.0"},
		IPC:           bundlemanifest.IPC{Host: "127.0.0.1", Port: 4000, AuthTokenRel: "runtime/token"},
		Telemetry:     bundlemanifest.Telemetry{File: "telemetry.jsonl"},
		Services: []bundlemanifest.Service{
			{
				ID:   "api",
				Type: "api-binary",
				Binaries: map[string]bundlemanifest.Binary{
					"linux-amd64": {Path: binaryRel},
				},
				Health:    bundlemanifest.HealthCheck{Type: "command"},
				Readiness: bundlemanifest.ReadinessCheck{Type: "health_success"},
			},
		},
	}

	manifestPath := filepath.Join(manifestRoot, "bundle.json")
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	if err := os.WriteFile(manifestPath, data, 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	builderCalls := []string{}
	builder := func(srcDir, outPath, goos, goarch, target string) error {
		builderCalls = append(builderCalls, fmt.Sprintf("%s/%s", target, goos))
		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			return err
		}
		return os.WriteFile(outPath, []byte(target), 0o755)
	}

	runtimeDir := filepath.Join(manifestRoot, "runtime")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime dir: %v", err)
	}

	packager := &bundlePackager{
		runtimeResolver: func() (string, error) { return runtimeDir, nil },
		runtimeBuilder:  builder,
	}

	result, err := packager.packageBundle(appDir, manifestPath, []string{"linux-amd64"})
	if err != nil {
		t.Fatalf("packageBundle: %v", err)
	}

	if len(builderCalls) == 0 {
		t.Fatalf("expected runtime builder to be called")
	}
	if result == nil || result.BundleDir == "" {
		t.Fatalf("expected bundle result to include BundleDir")
	}

	runtimePath := filepath.Join(result.BundleDir, "runtime", "linux-amd64", "runtime")
	if _, err := os.Stat(runtimePath); err != nil {
		t.Fatalf("runtime binary not staged: %v", err)
	}
}

func TestValidateManifestForPlatforms(t *testing.T) {
	t.Run("errors on missing schema or target", func(t *testing.T) {
		err := validateManifestForPlatforms(&bundlemanifest.Manifest{}, []string{"linux-amd64"})
		if err == nil || err.Error() != "schema_version is required" {
			t.Fatalf("expected schema_version error, got %v", err)
		}

		err = validateManifestForPlatforms(&bundlemanifest.Manifest{SchemaVersion: "0.1", Target: "mobile"}, []string{"linux-amd64"})
		if err == nil || err.Error() != `unsupported target "mobile" (expected desktop)` {
			t.Fatalf("expected desktop target error, got %v", err)
		}
	})

	t.Run("errors when binaries missing for requested platform", func(t *testing.T) {
		m := &bundlemanifest.Manifest{
			SchemaVersion: "0.1",
			Target:        "desktop",
			Services: []bundlemanifest.Service{
				{
					ID:       "api",
					Type:     "api-binary",
					Binaries: map[string]bundlemanifest.Binary{"linux-amd64": {Path: "bin/api"}},
				},
			},
		}
		err := validateManifestForPlatforms(m, []string{"linux-amd64", "darwin-arm64"})
		if err == nil || !strings.Contains(err.Error(), "missing binary for darwin-arm64") {
			t.Fatalf("expected missing binary error, got %v", err)
		}
	})
}

func TestResolveBinaryForPlatformHandlesAliases(t *testing.T) {
	svc := bundlemanifest.Service{
		ID: "api",
		Binaries: map[string]bundlemanifest.Binary{
			"windows-amd64": {Path: "bin/api.exe"},
			"mac-arm64":     {Path: "bin/api-mac"},
		},
	}

	tests := []struct {
		platform string
		want     string
	}{
		{platform: "win-amd64", want: "bin/api.exe"},
		{platform: "windows-amd64", want: "bin/api.exe"},
		{platform: "darwin-arm64", want: "bin/api-mac"},
		{platform: "mac-arm64", want: "bin/api-mac"},
	}

	for _, tt := range tests {
		t.Run(tt.platform, func(t *testing.T) {
			bin, ok := resolveBinaryForPlatform(svc, tt.platform)
			if !ok {
				t.Fatalf("resolveBinaryForPlatform(%s) returned !ok", tt.platform)
			}
			if bin.Path != tt.want {
				t.Fatalf("resolveBinaryForPlatform(%s) = %s, want %s", tt.platform, bin.Path, tt.want)
			}
		})
	}
}

func TestParsePlatformKeyRejectsInvalidValues(t *testing.T) {
	if _, _, err := parsePlatformKey("solaris-amd64"); err == nil {
		t.Fatal("expected unsupported OS error")
	}

	if _, _, err := parsePlatformKey("linux-ppc"); err == nil {
		t.Fatal("expected unsupported arch error")
	}

	if _, _, err := parsePlatformKey("linux"); err == nil {
		t.Fatal("expected malformed key error")
	}
}

func TestResolveManifestAndBundlePathsPreventEscapes(t *testing.T) {
	root := t.TempDir()

	manifestSafe, err := resolveManifestPath(root, "bin/api")
	if err != nil {
		t.Fatalf("resolveManifestPath safe path: %v", err)
	}
	if !strings.HasPrefix(manifestSafe, root) {
		t.Fatalf("resolved manifest path %s should stay within %s", manifestSafe, root)
	}

	if _, err := resolveManifestPath(root, "../etc/passwd"); err == nil {
		t.Fatal("expected resolveManifestPath to reject path escape")
	}

	bundleSafe, err := resolveBundlePath(root, "assets/model.bin")
	if err != nil {
		t.Fatalf("resolveBundlePath safe path: %v", err)
	}
	if !strings.HasPrefix(bundleSafe, root) {
		t.Fatalf("resolved bundle path %s should stay within %s", bundleSafe, root)
	}

	if _, err := resolveBundlePath(root, "../../tmp/outside"); err == nil {
		t.Fatal("expected resolveBundlePath to reject path escape")
	}
}

func TestCollectPlatformsDedupesAndSorts(t *testing.T) {
	m := bundlemanifest.Manifest{
		Services: []bundlemanifest.Service{
			{
				ID: "api",
				Binaries: map[string]bundlemanifest.Binary{
					"linux-amd64":   {Path: "bin/api"},
					"windows-amd64": {Path: "bin/api.exe"},
				},
			},
			{
				ID: "worker",
				Binaries: map[string]bundlemanifest.Binary{
					"linux-amd64": {Path: "bin/worker"},
					"mac-arm64":   {Path: "bin/worker"},
				},
			},
		},
	}

	got := collectPlatforms(m)
	want := []string{"linux-amd64", "mac-arm64", "windows-amd64"}
	if len(got) != len(want) {
		t.Fatalf("collectPlatforms length = %d, want %d", len(got), len(want))
	}
	for i, platform := range want {
		if got[i] != platform {
			t.Fatalf("collectPlatforms[%d] = %s, want %s", i, got[i], platform)
		}
	}
}

func TestHumanReadableSize(t *testing.T) {
	tests := []struct {
		bytes int64
		want  string
	}{
		{0, "0 B"},
		{500, "500 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
		{536870912, "512.0 MB"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d bytes", tt.bytes), func(t *testing.T) {
			got := humanReadableSize(tt.bytes)
			if got != tt.want {
				t.Errorf("humanReadableSize(%d) = %s, want %s", tt.bytes, got, tt.want)
			}
		})
	}
}

func TestCalculateBundleSize(t *testing.T) {
	root := t.TempDir()

	// Create files of various sizes
	smallFile := filepath.Join(root, "small.txt")
	if err := os.WriteFile(smallFile, make([]byte, 1000), 0o644); err != nil {
		t.Fatalf("write small file: %v", err)
	}

	// Create a file over 10MB threshold
	largeFile := filepath.Join(root, "large.bin")
	if err := os.WriteFile(largeFile, make([]byte, 11*1024*1024), 0o644); err != nil {
		t.Fatalf("write large file: %v", err)
	}

	totalSize, largeFiles := calculateBundleSize(root)

	// Check total size is at least the sum of our files
	expectedMin := int64(1000 + 11*1024*1024)
	if totalSize < expectedMin {
		t.Errorf("totalSize = %d, want at least %d", totalSize, expectedMin)
	}

	// Check large file was detected
	if len(largeFiles) != 1 {
		t.Errorf("expected 1 large file, got %d", len(largeFiles))
	}
	if len(largeFiles) > 0 && largeFiles[0].Path != "large.bin" {
		t.Errorf("expected large file path 'large.bin', got %s", largeFiles[0].Path)
	}
}

func TestCheckBundleSizeWarning(t *testing.T) {
	tests := []struct {
		name      string
		size      int64
		wantNil   bool
		wantLevel string
	}{
		{"small bundle", 100 * 1024 * 1024, true, ""},                       // 100MB - no warning
		{"warning threshold", 600 * 1024 * 1024, false, "warning"},          // 600MB - warning
		{"critical threshold", 1200 * 1024 * 1024, false, "critical"},       // 1.2GB - critical
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warning := checkBundleSizeWarning(tt.size, nil)

			if tt.wantNil {
				if warning != nil {
					t.Errorf("expected nil warning, got %+v", warning)
				}
				return
			}

			if warning == nil {
				t.Fatalf("expected warning, got nil")
			}
			if warning.Level != tt.wantLevel {
				t.Errorf("warning.Level = %s, want %s", warning.Level, tt.wantLevel)
			}
		})
	}
}
