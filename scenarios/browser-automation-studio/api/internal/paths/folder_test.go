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

func TestScenarioDataRootCallersPreserveEnvironmentOverrides(t *testing.T) {
	capturesOverride := filepath.Join(t.TempDir(), "capture-root")
	recordingsOverride := filepath.Join(t.TempDir(), "recording-root")
	profilesOverride := filepath.Join(t.TempDir(), "profile-root")
	t.Setenv("BAS_CAPTURES_ROOT", capturesOverride)
	t.Setenv("BAS_RECORDINGS_ROOT", recordingsOverride)
	t.Setenv("BAS_SESSION_PROFILES_ROOT", profilesOverride)

	for _, tc := range []struct {
		name string
		got  func() string
		want string
	}{
		{name: "captures", got: func() string { return ResolveCapturesRoot(nil) }, want: capturesOverride},
		{name: "recordings", got: func() string { return ResolveRecordingsRoot(nil) }, want: recordingsOverride},
		{name: "session profiles", got: func() string { return ResolveSessionProfilesRoot(nil) }, want: profilesOverride},
	} {
		t.Run(tc.name, func(t *testing.T) {
			want, err := filepath.Abs(tc.want)
			if err != nil {
				t.Fatal(err)
			}
			if got := tc.got(); got != want {
				t.Fatalf("root = %q, want %q", got, want)
			}
		})
	}
}

func TestScenarioDataRootCallersShareStorageAndRepositoryFallback(t *testing.T) {
	t.Setenv("BAS_CAPTURES_ROOT", "")
	t.Setenv("BAS_RECORDINGS_ROOT", "")
	t.Setenv("BAS_SESSION_PROFILES_ROOT", "")

	for _, tc := range []struct {
		name    string
		subdir  string
		purpose string
		got     func() string
	}{
		{name: "captures", subdir: "captures", purpose: "captures", got: func() string { return ResolveCapturesRoot(nil) }},
		{name: "recordings", subdir: "recordings", purpose: "recordings", got: func() string { return ResolveRecordingsRoot(nil) }},
		{name: "session profiles", subdir: "session-profiles", purpose: "session profiles", got: func() string { return ResolveSessionProfilesRoot(nil) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			want := resolveScenarioStoragePath(tc.subdir)
			if want == "" {
				want = filepath.Join(resolveScenarioDir(nil, tc.purpose), "data", tc.subdir)
			}
			if got := tc.got(); filepath.Clean(got) != filepath.Clean(want) {
				t.Fatalf("root = %q, want shared resolver path %q", got, want)
			}
		})
	}
}
