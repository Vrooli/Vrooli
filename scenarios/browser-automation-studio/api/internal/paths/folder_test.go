package paths

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestValidateAndNormalizeFolderPath(t *testing.T) {
	log := logrus.New()
	log.SetOutput(os.Stderr)

	// Get current working directory for test
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get cwd: %v", err)
	}

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "valid path within cwd",
			path:    filepath.Join(cwd, "test-folder"),
			wantErr: false,
		},
		{
			name:    "path outside cwd is allowed",
			path:    "/tmp/outside-project",
			wantErr: false,
		},
		{
			name:    "cwd itself is valid",
			path:    cwd,
			wantErr: false,
		},
		{
			name:    "empty path is invalid",
			path:    "   ",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValidateAndNormalizeFolderPath(tt.path, log)
			if tt.wantErr && err == nil {
				t.Errorf("ValidateAndNormalizeFolderPath() expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("ValidateAndNormalizeFolderPath() unexpected error: %v", err)
			}
			if !tt.wantErr && result == "" {
				t.Errorf("ValidateAndNormalizeFolderPath() returned empty path")
			}
		})
	}
}

func TestEnsureDirectoryExists(t *testing.T) {
	log := logrus.New()
	log.SetOutput(os.Stderr)

	// Create a temp directory for testing
	tmpDir := t.TempDir()

	testPath := filepath.Join(tmpDir, "new-dir", "nested")
	err := EnsureDirectoryExists(testPath, log)
	if err != nil {
		t.Errorf("EnsureDirectoryExists() failed: %v", err)
	}

	// Verify directory was created
	info, err := os.Stat(testPath)
	if err != nil {
		t.Errorf("Directory was not created: %v", err)
	}
	if !info.IsDir() {
		t.Errorf("Path is not a directory")
	}
}
