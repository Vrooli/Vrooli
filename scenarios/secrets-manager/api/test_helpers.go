package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/google/uuid"
)

// setupTestLogger initializes the logger for testing
func setupTestLogger() func() {
	// Initialize the package-level logger if not already initialized
	if logger == nil {
		logger = NewLogger("test")
	}
	// Save original logger output and redirect to discard during tests
	originalOutput := log.Writer()
	log.SetOutput(io.Discard)
	return func() {
		log.SetOutput(originalOutput)
	}
}

// TestEnvironment manages isolated test environment
type TestEnvironment struct {
	TempDir    string
	OriginalWD string
	Cleanup    func()
}

// setupTestDirectory creates an isolated test environment with proper cleanup
func setupTestDirectory(t *testing.T) *TestEnvironment {
	tempDir, err := os.MkdirTemp("", "secrets-manager-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// Create necessary test subdirectories
	testDirs := []string{
		filepath.Join(tempDir, "data"),
		filepath.Join(tempDir, "resources"),
		filepath.Join(tempDir, "scenarios"),
	}

	for _, dir := range testDirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			_ = os.RemoveAll(tempDir)
			t.Fatalf("Failed to create test directory %s: %v", dir, err)
		}
	}

	return &TestEnvironment{
		TempDir:    tempDir,
		OriginalWD: originalWD,
		Cleanup: func() {
			_ = os.Chdir(originalWD)
			_ = os.RemoveAll(tempDir)
		},
	}
}

// createTestResourceSecret creates a test resource secret
func createTestResourceSecret(resourceName, secretKey string) ResourceSecret {
	description := fmt.Sprintf("Test secret for %s", secretKey)
	return ResourceSecret{
		ID:           uuid.New().String(),
		ResourceName: resourceName,
		SecretKey:    secretKey,
		SecretType:   "api_key",
		Required:     true,
		Description:  &description,
	}
}

// createTestSecretValidation creates a test secret validation
func createTestSecretValidation(resourceSecretID string, status string) SecretValidation {
	return SecretValidation{
		ID:                uuid.New().String(),
		ResourceSecretID:  resourceSecretID,
		ValidationStatus:  status,
		ValidationMethod:  "pattern_match",
		ErrorMessage:      nil,
		ValidationDetails: nil,
	}
}

// createTestScanResult creates a test scan result
func createTestScanResult(scanType string, resourcesScanned []string, secretsFound int) SecretScan {
	return SecretScan{
		ID:                uuid.New().String(),
		ScanType:          scanType,
		ResourcesScanned:  resourcesScanned,
		SecretsDiscovered: secretsFound,
		ScanDurationMs:    100,
		ScanStatus:        "completed",
		ErrorMessage:      nil,
	}
}

// createTestFile creates a test file with content
func createTestFile(t *testing.T, dir, filename, content string) string {
	filePath := filepath.Join(dir, filename)
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to create test file %s: %v", filePath, err)
	}
	return filePath
}

// createTestResourceDir creates a test resource directory structure
func createTestResourceDir(t *testing.T, baseDir, resourceName string, files map[string]string) string {
	resourceDir := filepath.Join(baseDir, "resources", resourceName)
	if err := os.MkdirAll(resourceDir, 0o755); err != nil {
		t.Fatalf("Failed to create resource directory %s: %v", resourceDir, err)
	}

	for filename, content := range files {
		createTestFile(t, resourceDir, filename, content)
	}

	return resourceDir
}

// validateSecretType checks if a secret type is valid
func validateSecretType(secretType string) bool {
	validTypes := map[string]bool{
		"api_key":     true,
		"endpoint":    true,
		"password":    true,
		"token":       true,
		"certificate": true,
		"quota":       true,
		"config":      true,
	}
	return validTypes[secretType]
}

// validateValidationStatus checks if a validation status is valid
func validateValidationStatus(status string) bool {
	validStatuses := map[string]bool{
		"valid":   true,
		"invalid": true,
		"missing": true,
		"pending": true,
		"error":   true,
	}
	return validStatuses[status]
}

func liveRepoRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", ".."))
}

func newContractFixtureRepo(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	repoRoot := liveRepoRoot(t)

	contractData, err := os.ReadFile(filepath.Join(repoRoot, ".vrooli", "repo-contract.json"))
	if err != nil {
		t.Fatalf("read repo contract: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".vrooli"), 0o755); err != nil {
		t.Fatalf("mkdir .vrooli: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".vrooli", "repo-contract.json"), contractData, 0o644); err != nil {
		t.Fatalf("write repo contract: %v", err)
	}

	for _, dir := range []string{"templates", "scenarios", "resources", "packages", "cmd", "internal"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/secrets-manager-test\n\ngo 1.21\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	return root
}
