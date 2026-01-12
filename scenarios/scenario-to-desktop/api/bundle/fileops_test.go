package bundle

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyPathPreservesExecutablesAndDirectories(t *testing.T) {
	fileOps := &defaultFileOperations{}
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
	if err := fileOps.CopyPath(srcDir, dstDir); err != nil {
		t.Fatalf("CopyPath: %v", err)
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
	if err := fileOps.CopyPath(chromeSrc, fileDst); err != nil {
		t.Fatalf("CopyPath file: %v", err)
	}
	fileInfo, err := os.Stat(fileDst)
	if err != nil {
		t.Fatalf("stat copied file: %v", err)
	}
	if fileInfo.Mode()&0o111 == 0 {
		t.Fatalf("expected executable bit preserved on file copy, got mode %v", fileInfo.Mode())
	}
}

func TestCopyFileSameSourceAndDest(t *testing.T) {
	fileOps := &defaultFileOperations{}
	root := t.TempDir()

	// Create a file
	filePath := filepath.Join(root, "test.txt")
	if err := os.WriteFile(filePath, []byte("test content"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	// Copy to same location should not error
	if err := fileOps.CopyFile(filePath, filePath); err != nil {
		t.Fatalf("CopyFile same src/dst should succeed: %v", err)
	}

	// Verify content is preserved
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if string(content) != "test content" {
		t.Fatalf("expected content preserved, got %q", string(content))
	}
}

func TestNormalizeBundlePath(t *testing.T) {
	fileOps := &defaultFileOperations{}

	tests := []struct {
		input    string
		expected string
	}{
		{"../foo/bar", "foo/bar"},
		{"../../foo", "foo"},
		{"..", ""},
		{"foo/bar", "foo/bar"},
		{"../../../deeply/nested", "deeply/nested"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := fileOps.NormalizeBundlePath(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeBundlePath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestWithinBase(t *testing.T) {
	fileOps := &defaultFileOperations{}

	tests := []struct {
		base     string
		target   string
		expected bool
	}{
		{"/home/user", "/home/user/file.txt", true},
		{"/home/user", "/home/user/sub/file.txt", true},
		{"/home/user", "/home/other/file.txt", false},
		{"/home/user", "/home/../etc/passwd", false},
	}

	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			result := fileOps.WithinBase(tt.base, tt.target)
			if result != tt.expected {
				t.Errorf("WithinBase(%q, %q) = %v, want %v", tt.base, tt.target, result, tt.expected)
			}
		})
	}
}

// Size calculator tests

func TestSizeCalculator_Calculate(t *testing.T) {
	calc := &defaultSizeCalculator{}
	tmpDir := t.TempDir()

	// Create test files
	smallFile := filepath.Join(tmpDir, "small.txt")
	if err := os.WriteFile(smallFile, []byte("small"), 0o644); err != nil {
		t.Fatalf("write small file: %v", err)
	}

	// Create a large file (>10MB threshold)
	largeDir := filepath.Join(tmpDir, "large")
	if err := os.MkdirAll(largeDir, 0o755); err != nil {
		t.Fatalf("mkdir large: %v", err)
	}
	largeFile := filepath.Join(largeDir, "big.bin")
	largeData := make([]byte, 11*1024*1024) // 11MB
	if err := os.WriteFile(largeFile, largeData, 0o644); err != nil {
		t.Fatalf("write large file: %v", err)
	}

	totalSize, largeFiles := calc.Calculate(tmpDir)

	if totalSize < 11*1024*1024 {
		t.Errorf("expected total size >= 11MB, got %d", totalSize)
	}

	if len(largeFiles) != 1 {
		t.Errorf("expected 1 large file, got %d", len(largeFiles))
	}

	if len(largeFiles) > 0 && largeFiles[0].Path != "large/big.bin" {
		t.Errorf("expected large file path 'large/big.bin', got %q", largeFiles[0].Path)
	}
}

func TestSizeCalculator_CheckWarning(t *testing.T) {
	calc := &defaultSizeCalculator{}

	t.Run("no warning below threshold", func(t *testing.T) {
		warning := calc.CheckWarning(100*1024*1024, nil) // 100MB
		if warning != nil {
			t.Errorf("expected no warning for 100MB bundle")
		}
	})

	t.Run("warning at 500MB", func(t *testing.T) {
		warning := calc.CheckWarning(550*1024*1024, nil) // 550MB
		if warning == nil {
			t.Fatalf("expected warning for 550MB bundle")
		}
		if warning.Level != "warning" {
			t.Errorf("expected level 'warning', got %q", warning.Level)
		}
	})

	t.Run("critical at 1GB", func(t *testing.T) {
		warning := calc.CheckWarning(1100*1024*1024, nil) // 1.1GB
		if warning == nil {
			t.Fatalf("expected warning for 1.1GB bundle")
		}
		if warning.Level != "critical" {
			t.Errorf("expected level 'critical', got %q", warning.Level)
		}
	})
}

func TestHumanReadableSize(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{500, "500 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1024 * 1024, "1.0 MB"},
		{1024 * 1024 * 1024, "1.0 GB"},
		{1024 * 1024 * 1024 * 1024, "1.0 TB"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := HumanReadableSize(tt.bytes)
			if result != tt.expected {
				t.Errorf("HumanReadableSize(%d) = %q, want %q", tt.bytes, result, tt.expected)
			}
		})
	}
}

// Platform resolver tests

func TestPlatformResolver_ParseKey(t *testing.T) {
	resolver := &defaultPlatformResolver{}

	tests := []struct {
		name    string
		key     string
		goos    string
		goarch  string
		wantErr bool
	}{
		{"linux-x64", "linux-x64", "linux", "amd64", false},
		{"linux-arm64", "linux-arm64", "linux", "arm64", false},
		{"darwin-x64", "darwin-x64", "darwin", "amd64", false},
		{"darwin-arm64", "darwin-arm64", "darwin", "arm64", false},
		{"win-x64", "win-x64", "windows", "amd64", false},
		{"windows-amd64", "windows-amd64", "windows", "amd64", false},
		{"mac-arm64", "mac-arm64", "darwin", "arm64", false},
		{"invalid", "invalid", "", "", true},
		{"bad-arch", "linux-x86", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			goos, goarch, err := resolver.ParseKey(tt.key)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for key %q", tt.key)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseKey(%q) error: %v", tt.key, err)
			}
			if goos != tt.goos {
				t.Errorf("expected goos %q, got %q", tt.goos, goos)
			}
			if goarch != tt.goarch {
				t.Errorf("expected goarch %q, got %q", tt.goarch, goarch)
			}
		})
	}
}

func TestPlatformResolver_NormalizeRuntime(t *testing.T) {
	resolver := &defaultPlatformResolver{}

	tests := []struct {
		input    string
		expected string
	}{
		{"linux", "linux-x64"},
		{"mac", "darwin-x64"},
		{"darwin", "darwin-x64"},
		{"win", "win-x64"},
		{"windows", "win-x64"},
		{"linux-arm64", "linux-arm64"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := resolver.NormalizeRuntime(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeRuntime(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestPlatformResolver_BinaryNames(t *testing.T) {
	resolver := &defaultPlatformResolver{}

	t.Run("runtime binary linux", func(t *testing.T) {
		name := resolver.RuntimeBinaryName("linux")
		if name != "runtime" {
			t.Errorf("expected 'runtime', got %q", name)
		}
	})

	t.Run("runtime binary windows", func(t *testing.T) {
		name := resolver.RuntimeBinaryName("windows")
		if name != "runtime.exe" {
			t.Errorf("expected 'runtime.exe', got %q", name)
		}
	})

	t.Run("runtimectl binary linux", func(t *testing.T) {
		name := resolver.RuntimeCtlBinaryName("linux")
		if name != "runtimectl" {
			t.Errorf("expected 'runtimectl', got %q", name)
		}
	})

	t.Run("runtimectl binary windows", func(t *testing.T) {
		name := resolver.RuntimeCtlBinaryName("windows")
		if name != "runtimectl.exe" {
			t.Errorf("expected 'runtimectl.exe', got %q", name)
		}
	})
}

func TestAliasPlatformKey(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"windows-x64", "win-x64"},
		{"win-x64", "windows-x64"},
		{"darwin-arm64", "mac-arm64"},
		{"mac-arm64", "darwin-arm64"},
		{"linux-x64", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := aliasPlatformKey(tt.input)
			if result != tt.expected {
				t.Errorf("aliasPlatformKey(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExpandShorthandPlatform(t *testing.T) {
	t.Run("win expands", func(t *testing.T) {
		keys := expandShorthandPlatform("win")
		if len(keys) == 0 {
			t.Errorf("expected keys for 'win'")
		}
		found := false
		for _, k := range keys {
			if k == "win-x64" || k == "windows-x64" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected win-x64 or windows-x64 in keys")
		}
	})

	t.Run("linux expands", func(t *testing.T) {
		keys := expandShorthandPlatform("linux")
		if len(keys) == 0 {
			t.Errorf("expected keys for 'linux'")
		}
	})
}

// Packager tests

func TestNewPackager(t *testing.T) {
	packager := NewPackager()
	if packager == nil {
		t.Fatalf("expected packager to be created")
	}
}

func TestNewPackagerWithOptions(t *testing.T) {
	customResolver := &defaultRuntimeResolver{}
	customBuilder := &defaultRuntimeBuilder{}
	customCalculator := &defaultSizeCalculator{}
	customPlatform := &defaultPlatformResolver{}
	customFileOps := &defaultFileOperations{}

	packager := NewPackager(
		WithRuntimeResolver(customResolver),
		WithRuntimeBuilder(customBuilder),
		WithSizeCalculator(customCalculator),
		WithPlatformResolver(customPlatform),
		WithFileOperations(customFileOps),
	)

	if packager == nil {
		t.Fatalf("expected packager to be created")
	}
	if packager.runtimeResolver != customResolver {
		t.Errorf("expected custom runtime resolver")
	}
	if packager.sizeCalculator != customCalculator {
		t.Errorf("expected custom size calculator")
	}
}

func TestPackager_Package_Validation(t *testing.T) {
	packager := NewPackager()

	t.Run("empty app path", func(t *testing.T) {
		_, err := packager.Package("", "/manifest.json", nil)
		if err == nil {
			t.Errorf("expected error for empty app path")
		}
	})

	t.Run("empty manifest path", func(t *testing.T) {
		_, err := packager.Package("/app", "", nil)
		if err == nil {
			t.Errorf("expected error for empty manifest path")
		}
	})

	t.Run("nonexistent app path", func(t *testing.T) {
		_, err := packager.Package("/nonexistent/app", "/manifest.json", nil)
		if err == nil {
			t.Errorf("expected error for nonexistent app path")
		}
	})
}

// Types tests

func TestPackageResult_Fields(t *testing.T) {
	result := &PackageResult{
		BundleDir:       "/path/to/bundle",
		ManifestPath:    "/path/to/manifest.json",
		RuntimeBinaries: map[string]string{"linux": "/path/to/runtime"},
		CopiedArtifacts: []string{"/path/to/artifact"},
		TotalSizeBytes:  1024,
		TotalSizeHuman:  "1.0 KB",
		SizeWarning:     nil,
	}

	if result.BundleDir != "/path/to/bundle" {
		t.Errorf("expected BundleDir '/path/to/bundle'")
	}
	if len(result.RuntimeBinaries) != 1 {
		t.Errorf("expected 1 runtime binary")
	}
}

func TestPackageRequest_Fields(t *testing.T) {
	req := &PackageRequest{
		AppPath:            "/path/to/app",
		BundleManifestPath: "/path/to/manifest.json",
		Platforms:          []string{"linux", "mac"},
		Store:              "personal",
		Enterprise:         true,
	}

	if req.AppPath != "/path/to/app" {
		t.Errorf("expected AppPath '/path/to/app'")
	}
	if !req.Enterprise {
		t.Errorf("expected Enterprise to be true")
	}
}

func TestSizeWarning_Fields(t *testing.T) {
	warning := &SizeWarning{
		Level:      "warning",
		Message:    "Bundle too large",
		TotalBytes: 600 * 1024 * 1024,
		TotalHuman: "600 MB",
		LargeFiles: []LargeFileInfo{
			{Path: "large.bin", SizeBytes: 100 * 1024 * 1024, SizeHuman: "100 MB"},
		},
	}

	if warning.Level != "warning" {
		t.Errorf("expected Level 'warning'")
	}
	if len(warning.LargeFiles) != 1 {
		t.Errorf("expected 1 large file")
	}
}
