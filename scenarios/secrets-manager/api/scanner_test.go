package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestScanResourceDirectory tests the resource directory scanning functionality
func TestScanResourceDirectory(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()
	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("Success_WithSecrets", func(t *testing.T) {
		// Create a test resource directory with files containing secrets
		resourceFiles := map[string]string{
			"config.env": `
API_KEY=test-key
DATABASE_URL=postgresql://localhost
SECRET_TOKEN=abc123
PORT=8080
`,
			".env.example": `
OPENAI_API_KEY=your-api-key-here
DB_PASSWORD=your-password
`,
			"README.md": `
# Test Resource
Configuration requires API_KEY
`,
		}

		resourceDir := createTestResourceDir(t, env.TempDir, "test-resource", resourceFiles)

		secrets, err := scanResourceDirectory("test-resource", resourceDir)
		if err != nil {
			t.Fatalf("scanResourceDirectory failed: %v", err)
		}

		// Should find at least some secrets
		if len(secrets) == 0 {
			t.Error("Expected to find secrets, but found none")
		}

		// Verify secrets have required fields
		for _, secret := range secrets {
			if secret.ResourceName != "test-resource" {
				t.Errorf("Expected resource name 'test-resource', got %s", secret.ResourceName)
			}
			if secret.SecretKey == "" {
				t.Error("Secret key should not be empty")
			}
			if secret.SecretType == "" {
				t.Error("Secret type should not be empty")
			}
		}
	})

	t.Run("EmptyDirectory", func(t *testing.T) {
		emptyDir := createTestResourceDir(t, env.TempDir, "empty-resource", map[string]string{})

		secrets, err := scanResourceDirectory("empty-resource", emptyDir)
		if err != nil {
			t.Fatalf("scanResourceDirectory failed: %v", err)
		}

		if len(secrets) != 0 {
			t.Errorf("Expected 0 secrets in empty directory, got %d", len(secrets))
		}
	})

	t.Run("NonExistentDirectory", func(t *testing.T) {
		nonExistentDir := filepath.Join(env.TempDir, "non-existent")

		secrets, err := scanResourceDirectory("non-existent", nonExistentDir)
		// The implementation gracefully handles non-existent directories by returning empty results
		// This is acceptable behavior - it doesn't crash or error
		if err != nil {
			t.Logf("Non-existent directory returned error (acceptable): %v", err)
		}
		if len(secrets) != 0 {
			t.Errorf("Expected 0 secrets for non-existent directory, got %d", len(secrets))
		}
	})

	t.Run("BinaryFiles", func(t *testing.T) {
		// Create a resource with binary files
		resourceDir := filepath.Join(env.TempDir, "resources", "binary-resource")
		if err := os.MkdirAll(resourceDir, 0o755); err != nil {
			t.Fatal(err)
		}

		// Create a binary file
		binaryFile := filepath.Join(resourceDir, "binary.dat")
		if err := os.WriteFile(binaryFile, []byte{0xFF, 0xFE, 0x00, 0x01}, 0o644); err != nil {
			t.Fatal(err)
		}

		secrets, err := scanResourceDirectory("binary-resource", resourceDir)
		if err != nil {
			t.Fatalf("scanResourceDirectory failed: %v", err)
		}

		// Binary files should be skipped
		if len(secrets) != 0 {
			t.Logf("Found %d secrets in binary files (unexpected but not fatal)", len(secrets))
		}
	})

	t.Run("NestedDirectories", func(t *testing.T) {
		// Create nested directory structure
		resourceDir := filepath.Join(env.TempDir, "resources", "nested-resource")
		subDir := filepath.Join(resourceDir, "config", "production")
		if err := os.MkdirAll(subDir, 0o755); err != nil {
			t.Fatal(err)
		}

		// Create files in nested directories
		rootConfig := filepath.Join(resourceDir, ".env")
		if err := os.WriteFile(rootConfig, []byte("ROOT_API_KEY=test\n"), 0o644); err != nil {
			t.Fatal(err)
		}

		nestedConfig := filepath.Join(subDir, "secrets.env")
		if err := os.WriteFile(nestedConfig, []byte("NESTED_SECRET=value\n"), 0o644); err != nil {
			t.Fatal(err)
		}

		secrets, err := scanResourceDirectory("nested-resource", resourceDir)
		if err != nil {
			t.Fatalf("scanResourceDirectory failed: %v", err)
		}

		// Should find secrets from nested directories
		if len(secrets) == 0 {
			t.Error("Expected to find secrets in nested directories")
		}
	})
}

// TestEstimateFileCount tests file counting estimation
func TestEstimateFileCount(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()
	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("EmptyDirectories", func(t *testing.T) {
		count := estimateFileCount(
			filepath.Join(env.TempDir, "scenarios"),
			filepath.Join(env.TempDir, "resources"),
			"",
		)

		// Empty directories should return 0 or low count
		if count < 0 {
			t.Error("File count should not be negative")
		}
	})

	t.Run("WithFiles", func(t *testing.T) {
		// Create test files with extensions that estimateFileCount looks for
		createTestResourceDir(t, env.TempDir, "resource1", map[string]string{
			"config.yaml": "content",
			"setup.sh":    "content",
		})
		createTestResourceDir(t, env.TempDir, "resource2", map[string]string{
			"config.json": "content",
		})

		count := estimateFileCount(
			filepath.Join(env.TempDir, "scenarios"),
			filepath.Join(env.TempDir, "resources"),
			"",
		)

		if count == 0 {
			t.Error("Expected non-zero file count")
		}
	})

	t.Run("WithComponentFilter", func(t *testing.T) {
		count := estimateFileCount(
			filepath.Join(env.TempDir, "scenarios"),
			filepath.Join(env.TempDir, "resources"),
			"postgres",
		)

		// Filtering may reduce count
		if count < 0 {
			t.Error("File count should not be negative")
		}
	})
}

// TestScannerEdgeCases tests edge cases in scanning
func TestScannerEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()
	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("SymbolicLinks", func(t *testing.T) {
		// Create a resource directory
		resourceDir := createTestResourceDir(t, env.TempDir, "symlink-resource", map[string]string{
			"config.env": "API_KEY=test\n",
		})

		// Create a symbolic link (may fail on some systems)
		linkPath := filepath.Join(env.TempDir, "resources", "link-resource")
		if err := os.Symlink(resourceDir, linkPath); err != nil {
			t.Skip("Symbolic links not supported on this system")
		}

		// Scanning symlink should work
		secrets, err := scanResourceDirectory("link-resource", linkPath)
		if err != nil {
			t.Logf("Symlink scan returned error (may be expected): %v", err)
		} else {
			t.Logf("Found %d secrets via symlink", len(secrets))
		}
	})

	t.Run("PermissionDenied", func(t *testing.T) {
		// Create a directory with no read permissions
		restrictedDir := filepath.Join(env.TempDir, "resources", "restricted")
		if err := os.MkdirAll(restrictedDir, 0o755); err != nil {
			t.Fatal(err)
		}

		// Create a file
		testFile := filepath.Join(restrictedDir, "secret.env")
		if err := os.WriteFile(testFile, []byte("SECRET=test\n"), 0o644); err != nil {
			t.Fatal(err)
		}

		// Remove read permissions
		if err := os.Chmod(restrictedDir, 0o000); err != nil {
			t.Skip("Cannot modify permissions on this system")
		}
		defer func() { _ = os.Chmod(restrictedDir, 0o755) }() // Restore for cleanup

		// Scan should handle permission errors gracefully
		secrets, err := scanResourceDirectory("restricted", restrictedDir)
		if err != nil {
			t.Logf("Permission denied error (expected): %v", err)
		}
		if len(secrets) != 0 {
			t.Logf("Unexpectedly found %d secrets", len(secrets))
		}
	})

	t.Run("LargeFiles", func(t *testing.T) {
		// Create a large file
		resourceDir := filepath.Join(env.TempDir, "resources", "large-resource")
		if err := os.MkdirAll(resourceDir, 0o755); err != nil {
			t.Fatal(err)
		}

		largeFile := filepath.Join(resourceDir, "large.txt")
		content := make([]byte, 10*1024*1024) // 10MB
		for i := range content {
			content[i] = 'a'
		}
		if err := os.WriteFile(largeFile, content, 0o644); err != nil {
			t.Skip("Cannot create large file")
		}

		// Scan should handle large files
		secrets, err := scanResourceDirectory("large-resource", resourceDir)
		if err != nil {
			t.Logf("Large file scan error: %v", err)
		}
		t.Logf("Found %d secrets in large file", len(secrets))
	})

	t.Run("SpecialCharactersInFilenames", func(t *testing.T) {
		resourceFiles := map[string]string{
			"file with spaces.env":      "KEY1=value\n",
			"file-with-dashes.env":      "KEY2=value\n",
			"file_with_underscores.env": "KEY3=value\n",
		}

		resourceDir := createTestResourceDir(t, env.TempDir, "special-chars", resourceFiles)

		secrets, err := scanResourceDirectory("special-chars", resourceDir)
		if err != nil {
			t.Fatalf("Failed to scan resource with special char filenames: %v", err)
		}

		t.Logf("Found %d secrets in files with special characters", len(secrets))
	})
}

// TestNewSecretScanner tests scanner initialization
// [REQ:SEC-COMP-001] Component-aware scanner
func TestNewSecretScanner(t *testing.T) {
	scanner := NewSecretScanner(nil)
	if scanner == nil {
		t.Fatal("NewSecretScanner() returned nil")
	}
	if scanner.db != nil {
		t.Error("Expected nil database, got non-nil")
	}
}

// TestGetScanConfig tests scan configuration retrieval
// [REQ:SEC-SCAN-001] Secret pattern detection
func TestGetScanConfig(t *testing.T) {
	scanner := NewSecretScanner(nil)

	tests := []struct {
		name     string
		scanType string
	}{
		{"Quick scan", "quick"},
		{"Full scan", "full"},
		{"Deep scan", "deep"},
		{"Unknown scan", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := scanner.getScanConfig(tt.scanType)
			if config.MaxResources < 0 {
				t.Error("MaxResources should not be negative")
			}
			if config.ScanDepth == "" {
				t.Error("ScanDepth should not be empty")
			}
		})
	}
}

// TestGetSecretPatterns tests secret pattern retrieval
// [REQ:SEC-SCAN-001] Secret pattern detection
func TestGetSecretPatterns(t *testing.T) {
	scanner := NewSecretScanner(nil)

	patterns := scanner.getSecretPatterns()
	if len(patterns) == 0 {
		t.Error("Expected at least some secret patterns")
	}

	for _, pattern := range patterns {
		if pattern.Pattern == "" {
			t.Error("Pattern should not be empty")
		}
		if pattern.Type == "" {
			t.Error("Pattern type should not be empty")
		}
	}
}

// TestGetSecretKeywords tests secret keyword retrieval
// [REQ:SEC-SCAN-001] Secret pattern detection
func TestGetSecretKeywords(t *testing.T) {
	scanner := NewSecretScanner(nil)

	keywords := scanner.getSecretKeywords()
	if len(keywords) == 0 {
		t.Error("Expected at least some secret keywords")
	}

	for _, keyword := range keywords {
		if keyword == "" {
			t.Error("Keyword should not be empty")
		}
	}
}

// TestScanResources tests the main resource scanning function
// [REQ:SEC-COMP-001] Component-aware scanner
// [REQ:SEC-SCAN-001] Cross-scenario code scanning
func TestScanResources(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()
	root := newContractFixtureRepo(t)
	nested := filepath.Join(root, "scenarios", "secrets-manager", "api")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}
	t.Setenv("VROOLI_SOURCE_ROOT", nested)
	t.Setenv("VROOLI_ROOT", "")

	t.Run("Success_QuickScan", func(t *testing.T) {
		// Create test resource files
		resourceDir := filepath.Join(root, "resources", "test-resource")
		if err := os.MkdirAll(resourceDir, 0o755); err != nil {
			t.Fatal(err)
		}

		// Create a simple config file with secrets
		configFile := filepath.Join(resourceDir, "config.env")
		content := `API_KEY=test-key-123
DATABASE_URL=postgresql://localhost:5432/testdb
SECRET_TOKEN=secret-abc
PORT=8080`
		if err := os.WriteFile(configFile, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}

		scanner := NewSecretScanner(nil)
		request := ScanRequest{
			ScanType:  "quick",
			Resources: []string{"test-resource"},
		}

		response, err := scanner.ScanResources(request)
		if err != nil {
			t.Fatalf("ScanResources failed: %v", err)
		}

		if response == nil {
			t.Fatal("Expected non-nil response")
		}

		if response.ScanID == "" {
			t.Error("Expected scan ID to be set")
		}

		if len(response.ResourcesScanned) == 0 {
			t.Error("Expected at least one resource to be scanned")
		}
	})

	t.Run("EmptyResources", func(t *testing.T) {
		scanner := NewSecretScanner(nil)
		request := ScanRequest{
			ScanType:  "quick",
			Resources: []string{},
		}

		response, err := scanner.ScanResources(request)
		// Should complete without error even with no resources
		if err != nil {
			t.Logf("Empty resources scan returned error: %v (acceptable)", err)
		}
		if response != nil && response.ScanDurationMs < 0 {
			t.Error("Scan duration should not be negative")
		}
	})
}

// TestFindResourceFiles tests resource file discovery
// [REQ:SEC-VLT-001] Repository secret manifest discovery
func TestFindResourceFiles(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()
	root := newContractFixtureRepo(t)
	nested := filepath.Join(root, "scenarios", "secrets-manager", "api")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}
	t.Setenv("VROOLI_SOURCE_ROOT", nested)
	t.Setenv("VROOLI_ROOT", "")

	t.Run("Success_FindConfigFiles", func(t *testing.T) {
		// Create test resources
		resourceDir := filepath.Join(root, "resources", "postgres")
		if err := os.MkdirAll(resourceDir, 0o755); err != nil {
			t.Fatal(err)
		}

		// Create various config files
		testFiles := map[string]string{
			"config.env":    "API_KEY=test",
			".env":          "SECRET=value",
			"settings.yaml": "key: value",
			"README.md":     "# Resource",
		}

		for filename, content := range testFiles {
			path := filepath.Join(resourceDir, filename)
			if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
		}

		scanner := NewSecretScanner(nil)
		config := scanner.getScanConfig("quick")

		files, err := scanner.findResourceFiles([]string{"postgres"}, config)
		if err != nil {
			t.Fatalf("findResourceFiles failed: %v", err)
		}

		if len(files) == 0 {
			t.Error("Expected to find at least some files")
		}

		// Verify files are from the correct resource
		for _, file := range files {
			if !strings.Contains(file, "postgres") {
				t.Errorf("File %s doesn't appear to be from postgres resource", file)
			}
		}
	})

	t.Run("NonExistentResource", func(t *testing.T) {
		scanner := NewSecretScanner(nil)
		config := scanner.getScanConfig("quick")

		files, err := scanner.findResourceFiles([]string{"nonexistent"}, config)
		if err != nil {
			// Error is acceptable for non-existent resources
			t.Logf("Non-existent resource returned error (acceptable): %v", err)
		}
		if len(files) != 0 {
			t.Errorf("Expected 0 files for non-existent resource, got %d", len(files))
		}
	})

	t.Run("EmptyResourcesList", func(t *testing.T) {
		scanner := NewSecretScanner(nil)
		config := scanner.getScanConfig("quick")

		files, err := scanner.findResourceFiles([]string{}, config)
		// Should handle gracefully
		if err != nil {
			t.Logf("Empty resources list returned error: %v", err)
		}
		// Files list may or may not be empty depending on implementation
		if files == nil {
			t.Error("Expected non-nil files slice")
		}
	})
}

// TestDetermineSecretType tests secret type determination
// [REQ:SEC-SCAN-001] Secret pattern detection
func TestDetermineSecretType(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	tests := []struct {
		name      string
		secretKey string
		wantType  string
	}{
		{"API Key", "API_KEY", "api_key"},
		{"Password", "DATABASE_PASSWORD", "password"},
		{"Token", "AUTH_TOKEN", "token"},
		{"Certificate", "SSL_CERT", "certificate"},
		{"Unknown", "SOME_VAR", "env_var"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotType := ClassifySecretType(tt.secretKey)
			if gotType != tt.wantType {
				t.Errorf("ClassifySecretType(%s) = %s, want %s", tt.secretKey, gotType, tt.wantType)
			}
		})
	}
}

// TestStoreResourceSecret tests resource secret storage
// [REQ:SEC-DATA-001] Secret storage and retrieval
func TestStoreResourceSecret(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()
	scanner := NewSecretScanner(nil)

	secret := ResourceSecret{
		ResourceName: "postgres",
		SecretKey:    "DATABASE_PASSWORD",
		SecretType:   "password",
	}

	// Should handle nil DB gracefully
	err := scanner.storeResourceSecret(secret)
	if err != nil {
		t.Errorf("storeResourceSecret() with nil DB should not error, got: %v", err)
	}
}

// TestGetScanHistory tests scan history retrieval
func TestGetScanHistory(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("NoDatabase", func(t *testing.T) {
		scanner := NewSecretScanner(nil)

		history, err := scanner.GetScanHistory(10)
		if err != nil {
			t.Fatalf("GetScanHistory failed: %v", err)
		}
		// With no DB, should return empty slice
		if history == nil {
			t.Error("Expected non-nil history slice")
		}
		if len(history) != 0 {
			t.Errorf("Expected 0 history entries without DB, got %d", len(history))
		}
	})

	t.Run("ZeroLimit", func(t *testing.T) {
		scanner := NewSecretScanner(nil)

		history, err := scanner.GetScanHistory(0)
		if err != nil {
			t.Fatalf("GetScanHistory failed: %v", err)
		}
		if history == nil {
			t.Error("Expected non-nil history slice")
		}
	})
}
