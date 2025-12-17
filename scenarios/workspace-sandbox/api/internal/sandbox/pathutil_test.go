package sandbox

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"workspace-sandbox/internal/types"
)

// [REQ:P0-002] Stable Sandbox Identifier - UUID format validation
func TestUUIDGeneration(t *testing.T) {
	id := uuid.New()
	if id == uuid.Nil {
		t.Error("generated UUID should not be nil")
	}

	// Validate UUID format
	if len(id.String()) != 36 {
		t.Errorf("UUID should be 36 characters, got %d", len(id.String()))
	}
}

// [REQ:P0-002] Stable Sandbox Identifier - Path normalization
func TestPathNormalization(t *testing.T) {
	// Create a temp directory for testing
	tmpDir, err := os.MkdirTemp("", "sandbox-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create subdirectory
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	validator, err := NewPathValidator(tmpDir)
	if err != nil {
		t.Fatalf("failed to create validator: %v", err)
	}

	tests := []struct {
		name      string
		input     string
		wantPath  string
		wantError bool
	}{
		{
			name:     "empty path returns project root",
			input:    "",
			wantPath: tmpDir,
		},
		{
			name:     "relative path normalized",
			input:    "subdir",
			wantPath: subDir,
		},
		{
			name:     "absolute path within project",
			input:    subDir,
			wantPath: subDir,
		},
		{
			name:      "path outside project",
			input:     "/etc/passwd",
			wantError: true,
		},
		{
			name:     "path with dots normalized",
			input:    "subdir/../subdir",
			wantPath: subDir,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validator.NormalizePath(tt.input)
			if tt.wantError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if got != tt.wantPath {
				t.Errorf("got %q, want %q", got, tt.wantPath)
			}
		})
	}
}

// [REQ:P0-005] Scope Path Validation - Scope overlap detection
// Tests use the semantic: CheckPathOverlap(existingScope, newScope)
func TestCheckPathOverlap(t *testing.T) {
	tests := []struct {
		name          string
		existingScope string
		newScope      string
		wantType      types.ConflictType
	}{
		{
			name:          "exact match",
			existingScope: "/project/src",
			newScope:      "/project/src",
			wantType:      types.ConflictTypeExact,
		},
		{
			name:          "existing contains new (existing is parent)",
			existingScope: "/project",
			newScope:      "/project/src/main.go",
			wantType:      types.ConflictTypeExistingContainsNew,
		},
		{
			name:          "new contains existing (new is parent)",
			existingScope: "/project/src/main.go",
			newScope:      "/project",
			wantType:      types.ConflictTypeNewContainsExisting,
		},
		{
			name:          "no overlap - siblings",
			existingScope: "/project/src",
			newScope:      "/project/tests",
			wantType:      "",
		},
		{
			name:          "no overlap - different roots",
			existingScope: "/project1/src",
			newScope:      "/project2/src",
			wantType:      "",
		},
		{
			name:          "partial name match but not ancestor",
			existingScope: "/project/src",
			newScope:      "/project/src-backup",
			wantType:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := types.CheckPathOverlap(tt.existingScope, tt.newScope)
			if got != tt.wantType {
				t.Errorf("types.CheckPathOverlap(%q, %q) = %q, want %q",
					tt.existingScope, tt.newScope, got, tt.wantType)
			}
		})
	}
}

// [REQ:P0-005] Scope Path Validation - Conflict detection
func TestFindConflicts(t *testing.T) {
	existing := []*types.Sandbox{
		{
			ID:        uuid.New(),
			ScopePath: "/project/src",
			Status:    types.StatusActive,
		},
		{
			ID:        uuid.New(),
			ScopePath: "/project/tests",
			Status:    types.StatusActive,
		},
		{
			ID:        uuid.New(),
			ScopePath: "/project/old",
			Status:    types.StatusStopped, // Should be ignored
		},
	}

	tests := []struct {
		name          string
		newScope      string
		wantConflicts int
	}{
		{
			name:          "no conflict",
			newScope:      "/project/docs",
			wantConflicts: 0,
		},
		{
			name:          "exact match conflict",
			newScope:      "/project/src",
			wantConflicts: 1,
		},
		{
			name:          "ancestor conflict",
			newScope:      "/project/src/main",
			wantConflicts: 1,
		},
		{
			name:          "parent conflict",
			newScope:      "/project",
			wantConflicts: 2, // Conflicts with both src and tests
		},
		{
			name:          "stopped sandbox ignored",
			newScope:      "/project/old",
			wantConflicts: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conflicts := FindConflicts(tt.newScope, existing)
			if len(conflicts) != tt.wantConflicts {
				t.Errorf("FindConflicts(%q) returned %d conflicts, want %d",
					tt.newScope, len(conflicts), tt.wantConflicts)
			}
		})
	}
}

// [REQ:P0-001] Fast Sandbox Creation - Status transitions
func TestStatusTransitions(t *testing.T) {
	tests := []struct {
		status     types.Status
		isActive   bool
		isTerminal bool
	}{
		{types.StatusCreating, true, false},
		{types.StatusActive, true, false},
		{types.StatusStopped, false, false},
		{types.StatusApproved, false, true},
		{types.StatusRejected, false, true},
		{types.StatusDeleted, false, true},
		{types.StatusError, false, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := tt.status.IsActive(); got != tt.isActive {
				t.Errorf("%s.IsActive() = %v, want %v", tt.status, got, tt.isActive)
			}
			if got := tt.status.IsTerminal(); got != tt.isTerminal {
				t.Errorf("%s.IsTerminal() = %v, want %v", tt.status, got, tt.isTerminal)
			}
		})
	}
}

// [REQ:P0-005] Scope Path Validation - Performance test
func BenchmarkCheckPathOverlap(b *testing.B) {
	path1 := "/project/src/components/users/profile/settings"
	path2 := "/project/src/components/users/profile/avatar"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		types.CheckPathOverlap(path1, path2)
	}
}

// [REQ:P0-002] Stable Sandbox Identifier - Validate path within project
func TestIsWithinProject(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sandbox-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	validator, err := NewPathValidator(tmpDir)
	if err != nil {
		t.Fatalf("failed to create validator: %v", err)
	}

	tests := []struct {
		name   string
		path   string
		within bool
	}{
		{
			name:   "exact root",
			path:   tmpDir,
			within: true,
		},
		{
			name:   "subdirectory",
			path:   filepath.Join(tmpDir, "subdir"),
			within: true,
		},
		{
			name:   "nested subdirectory",
			path:   filepath.Join(tmpDir, "a", "b", "c"),
			within: true,
		},
		{
			name:   "outside project",
			path:   "/etc",
			within: false,
		},
		{
			name:   "similar prefix but outside",
			path:   tmpDir + "-other",
			within: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validator.IsWithinProject(tt.path); got != tt.within {
				t.Errorf("IsWithinProject(%q) = %v, want %v", tt.path, got, tt.within)
			}
		})
	}
}
