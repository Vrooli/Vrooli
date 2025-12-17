package types

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// types.go Tests
// ============================================================================

func TestOwnerTypeConstants(t *testing.T) {
	if OwnerTypeAgent != "agent" {
		t.Errorf("OwnerTypeAgent = %q, want 'agent'", OwnerTypeAgent)
	}
	if OwnerTypeUser != "user" {
		t.Errorf("OwnerTypeUser = %q, want 'user'", OwnerTypeUser)
	}
	if OwnerTypeTask != "task" {
		t.Errorf("OwnerTypeTask = %q, want 'task'", OwnerTypeTask)
	}
	if OwnerTypeSystem != "system" {
		t.Errorf("OwnerTypeSystem = %q, want 'system'", OwnerTypeSystem)
	}
}

func TestChangeTypeConstants(t *testing.T) {
	if ChangeTypeAdded != "added" {
		t.Errorf("ChangeTypeAdded = %q, want 'added'", ChangeTypeAdded)
	}
	if ChangeTypeModified != "modified" {
		t.Errorf("ChangeTypeModified = %q, want 'modified'", ChangeTypeModified)
	}
	if ChangeTypeDeleted != "deleted" {
		t.Errorf("ChangeTypeDeleted = %q, want 'deleted'", ChangeTypeDeleted)
	}
}

func TestApprovalStatusConstants(t *testing.T) {
	if ApprovalPending != "pending" {
		t.Errorf("ApprovalPending = %q, want 'pending'", ApprovalPending)
	}
	if ApprovalApproved != "approved" {
		t.Errorf("ApprovalApproved = %q, want 'approved'", ApprovalApproved)
	}
	if ApprovalRejected != "rejected" {
		t.Errorf("ApprovalRejected = %q, want 'rejected'", ApprovalRejected)
	}
}

func TestConflictTypeConstants(t *testing.T) {
	if ConflictTypeExact != "exact" {
		t.Errorf("ConflictTypeExact = %q, want 'exact'", ConflictTypeExact)
	}
	if ConflictTypeNewContainsExisting != "new_contains_existing" {
		t.Errorf("ConflictTypeNewContainsExisting = %q, want 'new_contains_existing'", ConflictTypeNewContainsExisting)
	}
	if ConflictTypeExistingContainsNew != "existing_contains_new" {
		t.Errorf("ConflictTypeExistingContainsNew = %q, want 'existing_contains_new'", ConflictTypeExistingContainsNew)
	}
}

func TestSandbox_WorkspacePath(t *testing.T) {
	t.Run("returns merged dir when active", func(t *testing.T) {
		s := &Sandbox{
			Status:    StatusActive,
			MergedDir: "/var/lib/sandbox/merged",
		}
		if got := s.WorkspacePath(); got != "/var/lib/sandbox/merged" {
			t.Errorf("WorkspacePath() = %q, want '/var/lib/sandbox/merged'", got)
		}
	})

	t.Run("returns empty when not active", func(t *testing.T) {
		for _, status := range []Status{StatusCreating, StatusStopped, StatusApproved, StatusRejected, StatusDeleted, StatusError} {
			s := &Sandbox{
				Status:    status,
				MergedDir: "/var/lib/sandbox/merged",
			}
			if got := s.WorkspacePath(); got != "" {
				t.Errorf("WorkspacePath() with status %s = %q, want ''", status, got)
			}
		}
	})

	t.Run("returns empty when active but no merged dir", func(t *testing.T) {
		s := &Sandbox{
			Status:    StatusActive,
			MergedDir: "",
		}
		if got := s.WorkspacePath(); got != "" {
			t.Errorf("WorkspacePath() with empty MergedDir = %q, want ''", got)
		}
	})
}

func TestCheckPathOverlap(t *testing.T) {
	tests := []struct {
		name     string
		existing string
		proposed string
		want     ConflictType
	}{
		{
			name:     "exact match",
			existing: "/project/src",
			proposed: "/project/src",
			want:     ConflictTypeExact,
		},
		{
			name:     "exact match with trailing slash normalization",
			existing: "/project/src/",
			proposed: "/project/src",
			want:     ConflictTypeExact,
		},
		{
			name:     "existing contains new",
			existing: "/project",
			proposed: "/project/src",
			want:     ConflictTypeExistingContainsNew,
		},
		{
			name:     "existing contains new - deeper nesting",
			existing: "/project",
			proposed: "/project/src/components",
			want:     ConflictTypeExistingContainsNew,
		},
		{
			name:     "new contains existing",
			existing: "/project/src",
			proposed: "/project",
			want:     ConflictTypeNewContainsExisting,
		},
		{
			name:     "new contains existing - deeper nesting",
			existing: "/project/src/components",
			proposed: "/project",
			want:     ConflictTypeNewContainsExisting,
		},
		{
			name:     "no overlap - siblings",
			existing: "/project/src",
			proposed: "/project/tests",
			want:     "",
		},
		{
			name:     "no overlap - different roots",
			existing: "/home/user/project",
			proposed: "/var/lib/project",
			want:     "",
		},
		{
			name:     "no overlap - prefix but not path segment",
			existing: "/project/src",
			proposed: "/project/srcgen",
			want:     "",
		},
		{
			name:     "no overlap - similar prefix",
			existing: "/project/app",
			proposed: "/project/application",
			want:     "",
		},
		{
			name:     "root path edge case",
			existing: "/",
			proposed: "/anything",
			want:     "", // Root path "/" becomes "/" after Clean, but "/anything" doesn't start with "//"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CheckPathOverlap(tt.existing, tt.proposed)
			if got != tt.want {
				t.Errorf("CheckPathOverlap(%q, %q) = %q, want %q",
					tt.existing, tt.proposed, got, tt.want)
			}
		})
	}
}

func TestSandboxStructure(t *testing.T) {
	// Test that Sandbox can be properly initialized with all fields
	now := time.Now()
	id := uuid.New()

	s := Sandbox{
		ID:             id,
		ScopePath:      "/project/src",
		ProjectRoot:    "/project",
		Owner:          "user-1",
		OwnerType:      OwnerTypeUser,
		Status:         StatusActive,
		ErrorMsg:       "",
		CreatedAt:      now,
		LastUsedAt:     now,
		Driver:         "overlayfs",
		DriverVersion:  "1.0",
		LowerDir:       "/lower",
		UpperDir:       "/upper",
		WorkDir:        "/work",
		MergedDir:      "/merged",
		SizeBytes:      1024,
		FileCount:      10,
		Tags:           []string{"test"},
		Metadata:       map[string]interface{}{"key": "value"},
		IdempotencyKey: "idem-123",
		Version:        1,
	}

	if s.ID != id {
		t.Error("ID not set correctly")
	}
	if s.Owner != "user-1" {
		t.Error("Owner not set correctly")
	}
}

// ============================================================================
// status.go Tests
// ============================================================================

func TestStatusConstants(t *testing.T) {
	if StatusCreating != "creating" {
		t.Errorf("StatusCreating = %q, want 'creating'", StatusCreating)
	}
	if StatusActive != "active" {
		t.Errorf("StatusActive = %q, want 'active'", StatusActive)
	}
	if StatusStopped != "stopped" {
		t.Errorf("StatusStopped = %q, want 'stopped'", StatusStopped)
	}
	if StatusApproved != "approved" {
		t.Errorf("StatusApproved = %q, want 'approved'", StatusApproved)
	}
	if StatusRejected != "rejected" {
		t.Errorf("StatusRejected = %q, want 'rejected'", StatusRejected)
	}
	if StatusDeleted != "deleted" {
		t.Errorf("StatusDeleted = %q, want 'deleted'", StatusDeleted)
	}
	if StatusError != "error" {
		t.Errorf("StatusError = %q, want 'error'", StatusError)
	}
}

func TestStatus_IsActive(t *testing.T) {
	tests := []struct {
		status Status
		want   bool
	}{
		{StatusCreating, true},
		{StatusActive, true},
		{StatusStopped, false},
		{StatusApproved, false},
		{StatusRejected, false},
		{StatusDeleted, false},
		{StatusError, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := tt.status.IsActive(); got != tt.want {
				t.Errorf("%s.IsActive() = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

func TestStatus_IsTerminal(t *testing.T) {
	tests := []struct {
		status Status
		want   bool
	}{
		{StatusCreating, false},
		{StatusActive, false},
		{StatusStopped, false},
		{StatusApproved, true},
		{StatusRejected, true},
		{StatusDeleted, true},
		{StatusError, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := tt.status.IsTerminal(); got != tt.want {
				t.Errorf("%s.IsTerminal() = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

func TestStatus_IsMounted(t *testing.T) {
	tests := []struct {
		status Status
		want   bool
	}{
		{StatusCreating, false},
		{StatusActive, true},
		{StatusStopped, false},
		{StatusApproved, false},
		{StatusRejected, false},
		{StatusDeleted, false},
		{StatusError, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := tt.status.IsMounted(); got != tt.want {
				t.Errorf("%s.IsMounted() = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

func TestStatus_RequiresCleanup(t *testing.T) {
	tests := []struct {
		status Status
		want   bool
	}{
		{StatusCreating, true},
		{StatusActive, true},
		{StatusStopped, true},
		{StatusApproved, false},
		{StatusRejected, false},
		{StatusDeleted, false},
		{StatusError, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := tt.status.RequiresCleanup(); got != tt.want {
				t.Errorf("%s.RequiresCleanup() = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

func TestCanStop(t *testing.T) {
	tests := []struct {
		status  Status
		wantErr bool
	}{
		{StatusActive, false},
		{StatusCreating, true},
		{StatusStopped, true},
		{StatusApproved, true},
		{StatusRejected, true},
		{StatusDeleted, true},
		{StatusError, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			err := CanStop(tt.status)
			if (err != nil) != tt.wantErr {
				t.Errorf("CanStop(%s) error = %v, wantErr %v", tt.status, err, tt.wantErr)
			}
			if err != nil {
				var transErr *InvalidTransitionError
				if !errors.As(err, &transErr) {
					t.Errorf("expected InvalidTransitionError, got %T", err)
				}
			}
		})
	}
}

func TestCanApprove(t *testing.T) {
	tests := []struct {
		status  Status
		wantErr bool
	}{
		{StatusActive, false},
		{StatusStopped, false},
		{StatusCreating, true},
		{StatusApproved, true},
		{StatusRejected, true},
		{StatusDeleted, true},
		{StatusError, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			err := CanApprove(tt.status)
			if (err != nil) != tt.wantErr {
				t.Errorf("CanApprove(%s) error = %v, wantErr %v", tt.status, err, tt.wantErr)
			}
		})
	}
}

func TestCanReject(t *testing.T) {
	tests := []struct {
		status  Status
		wantErr bool
	}{
		{StatusActive, false},
		{StatusStopped, false},
		{StatusCreating, true},
		{StatusApproved, true},
		{StatusRejected, true},
		{StatusDeleted, true},
		{StatusError, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			err := CanReject(tt.status)
			if (err != nil) != tt.wantErr {
				t.Errorf("CanReject(%s) error = %v, wantErr %v", tt.status, err, tt.wantErr)
			}
		})
	}
}

func TestCanDelete(t *testing.T) {
	tests := []struct {
		status  Status
		wantErr bool
	}{
		{StatusCreating, false},
		{StatusActive, false},
		{StatusStopped, false},
		{StatusApproved, false},
		{StatusRejected, false},
		{StatusError, false},
		{StatusDeleted, true}, // Already deleted
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			err := CanDelete(tt.status)
			if (err != nil) != tt.wantErr {
				t.Errorf("CanDelete(%s) error = %v, wantErr %v", tt.status, err, tt.wantErr)
			}
		})
	}
}

func TestCanGenerateDiff(t *testing.T) {
	tests := []struct {
		status  Status
		wantErr bool
	}{
		{StatusActive, false},
		{StatusStopped, false},
		{StatusApproved, false}, // Allow historical view
		{StatusRejected, false}, // Allow historical view
		{StatusCreating, true},
		{StatusDeleted, true},
		{StatusError, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			err := CanGenerateDiff(tt.status)
			if (err != nil) != tt.wantErr {
				t.Errorf("CanGenerateDiff(%s) error = %v, wantErr %v", tt.status, err, tt.wantErr)
			}
		})
	}
}

func TestCanGetWorkspacePath(t *testing.T) {
	tests := []struct {
		status  Status
		wantErr bool
	}{
		{StatusActive, false},
		{StatusCreating, true},
		{StatusStopped, true},
		{StatusApproved, true},
		{StatusRejected, true},
		{StatusDeleted, true},
		{StatusError, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			err := CanGetWorkspacePath(tt.status)
			if (err != nil) != tt.wantErr {
				t.Errorf("CanGetWorkspacePath(%s) error = %v, wantErr %v", tt.status, err, tt.wantErr)
			}
		})
	}
}

func TestCanTransitionTo(t *testing.T) {
	tests := []struct {
		from Status
		to   Status
		want bool
	}{
		// From Creating
		{StatusCreating, StatusActive, true},
		{StatusCreating, StatusError, true},
		{StatusCreating, StatusDeleted, true},
		{StatusCreating, StatusStopped, false},
		{StatusCreating, StatusApproved, false},

		// From Active
		{StatusActive, StatusStopped, true},
		{StatusActive, StatusApproved, true},
		{StatusActive, StatusRejected, true},
		{StatusActive, StatusError, true},
		{StatusActive, StatusDeleted, true},
		{StatusActive, StatusCreating, false},

		// From Stopped
		{StatusStopped, StatusActive, true},
		{StatusStopped, StatusApproved, true},
		{StatusStopped, StatusRejected, true},
		{StatusStopped, StatusDeleted, true},
		{StatusStopped, StatusCreating, false},
		{StatusStopped, StatusError, false},

		// From Approved (terminal)
		{StatusApproved, StatusDeleted, true},
		{StatusApproved, StatusActive, false},
		{StatusApproved, StatusRejected, false},

		// From Rejected (terminal)
		{StatusRejected, StatusDeleted, true},
		{StatusRejected, StatusActive, false},
		{StatusRejected, StatusApproved, false},

		// From Error
		{StatusError, StatusDeleted, true},
		{StatusError, StatusActive, false},
		{StatusError, StatusStopped, false},

		// From Deleted (no transitions)
		{StatusDeleted, StatusActive, false},
		{StatusDeleted, StatusDeleted, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.from)+"_to_"+string(tt.to), func(t *testing.T) {
			if got := CanTransitionTo(tt.from, tt.to); got != tt.want {
				t.Errorf("CanTransitionTo(%s, %s) = %v, want %v", tt.from, tt.to, got, tt.want)
			}
		})
	}
}

func TestValidTransitionsCompleteness(t *testing.T) {
	// Ensure all statuses have entries in ValidTransitions
	allStatuses := []Status{StatusCreating, StatusActive, StatusStopped, StatusApproved, StatusRejected, StatusDeleted, StatusError}

	for _, status := range allStatuses {
		if _, exists := ValidTransitions[status]; !exists {
			t.Errorf("ValidTransitions missing entry for %s", status)
		}
	}
}

func TestInvalidTransitionError(t *testing.T) {
	t.Run("with attempted status", func(t *testing.T) {
		err := &InvalidTransitionError{
			Current:   StatusStopped,
			Attempted: StatusCreating,
			Reason:    "cannot go back to creating",
		}
		msg := err.Error()
		if msg == "" {
			t.Error("expected non-empty error message")
		}
		if !containsAll(msg, "stopped", "creating", "cannot go back") {
			t.Errorf("error message missing expected parts: %s", msg)
		}
	})

	t.Run("without attempted status", func(t *testing.T) {
		err := &InvalidTransitionError{
			Current: StatusCreating,
			Reason:  "not ready yet",
		}
		msg := err.Error()
		if msg == "" {
			t.Error("expected non-empty error message")
		}
		if !containsAll(msg, "creating", "not ready yet") {
			t.Errorf("error message missing expected parts: %s", msg)
		}
	})
}

// ============================================================================
// errors.go Tests
// ============================================================================

func TestNotFoundError(t *testing.T) {
	t.Run("with resource", func(t *testing.T) {
		err := &NotFoundError{Resource: "sandbox", ID: "sb-123"}
		if err.HTTPStatus() != http.StatusNotFound {
			t.Errorf("HTTPStatus() = %d, want %d", err.HTTPStatus(), http.StatusNotFound)
		}
		if err.IsRetryable() {
			t.Error("IsRetryable() should be false")
		}
		msg := err.Error()
		if !containsAll(msg, "sandbox", "sb-123", "not found") {
			t.Errorf("Error() missing expected parts: %s", msg)
		}
		hint := err.Hint()
		if hint == "" {
			t.Error("Hint() should not be empty")
		}
	})

	t.Run("without resource", func(t *testing.T) {
		err := &NotFoundError{ID: "unknown-123"}
		msg := err.Error()
		if !containsAll(msg, "resource not found", "unknown-123") {
			t.Errorf("Error() missing expected parts: %s", msg)
		}
	})

	t.Run("NewNotFoundError helper", func(t *testing.T) {
		err := NewNotFoundError("sb-456")
		if err.Resource != "sandbox" {
			t.Errorf("Resource = %q, want 'sandbox'", err.Resource)
		}
		if err.ID != "sb-456" {
			t.Errorf("ID = %q, want 'sb-456'", err.ID)
		}
	})
}

func TestScopeConflictError(t *testing.T) {
	t.Run("single conflict", func(t *testing.T) {
		err := &ScopeConflictError{
			Conflicts: []PathConflict{
				{ExistingID: "sb-1", ExistingScope: "/project", NewScope: "/project/src"},
			},
		}
		if err.HTTPStatus() != http.StatusConflict {
			t.Errorf("HTTPStatus() = %d, want %d", err.HTTPStatus(), http.StatusConflict)
		}
		if err.IsRetryable() {
			t.Error("IsRetryable() should be false")
		}
		msg := err.Error()
		if !containsAll(msg, "scope conflict", "sb-1", "/project") {
			t.Errorf("Error() missing expected parts: %s", msg)
		}
	})

	t.Run("multiple conflicts", func(t *testing.T) {
		err := &ScopeConflictError{
			Conflicts: []PathConflict{
				{ExistingID: "sb-1"},
				{ExistingID: "sb-2"},
			},
		}
		msg := err.Error()
		if !containsAll(msg, "2 existing sandboxes") {
			t.Errorf("Error() missing expected parts: %s", msg)
		}
	})

	t.Run("hint", func(t *testing.T) {
		err := &ScopeConflictError{}
		hint := err.Hint()
		if hint == "" {
			t.Error("Hint() should not be empty")
		}
	})
}

func TestValidationError(t *testing.T) {
	t.Run("with field", func(t *testing.T) {
		err := &ValidationError{Field: "scopePath", Message: "cannot be empty"}
		if err.HTTPStatus() != http.StatusBadRequest {
			t.Errorf("HTTPStatus() = %d, want %d", err.HTTPStatus(), http.StatusBadRequest)
		}
		if err.IsRetryable() {
			t.Error("IsRetryable() should be false")
		}
		msg := err.Error()
		if !containsAll(msg, "scopePath", "cannot be empty") {
			t.Errorf("Error() missing expected parts: %s", msg)
		}
	})

	t.Run("without field", func(t *testing.T) {
		err := &ValidationError{Message: "invalid request format"}
		msg := err.Error()
		if !containsAll(msg, "validation error", "invalid request format") {
			t.Errorf("Error() missing expected parts: %s", msg)
		}
	})

	t.Run("GetHint with custom hint", func(t *testing.T) {
		err := &ValidationError{Hint: "Use absolute paths"}
		if got := err.GetHint(); got != "Use absolute paths" {
			t.Errorf("GetHint() = %q, want 'Use absolute paths'", got)
		}
	})

	t.Run("GetHint default", func(t *testing.T) {
		err := &ValidationError{}
		hint := err.GetHint()
		if hint == "" {
			t.Error("GetHint() should return default hint")
		}
	})

	t.Run("NewValidationError helper", func(t *testing.T) {
		err := NewValidationError("field1", "message1")
		if err.Field != "field1" || err.Message != "message1" {
			t.Error("NewValidationError did not set fields correctly")
		}
	})

	t.Run("NewValidationErrorWithHint helper", func(t *testing.T) {
		err := NewValidationErrorWithHint("field1", "message1", "hint1")
		if err.Field != "field1" || err.Message != "message1" || err.Hint != "hint1" {
			t.Error("NewValidationErrorWithHint did not set fields correctly")
		}
	})
}

func TestStateError(t *testing.T) {
	wrapped := &InvalidTransitionError{Current: StatusApproved, Attempted: StatusActive, Reason: "terminal"}
	err := &StateError{
		Message:       "custom message",
		Wrapped:       wrapped,
		CurrentStatus: StatusApproved,
		Operation:     "resume",
	}

	if err.HTTPStatus() != http.StatusConflict {
		t.Errorf("HTTPStatus() = %d, want %d", err.HTTPStatus(), http.StatusConflict)
	}
	if err.IsRetryable() {
		t.Error("IsRetryable() should be false")
	}
	if err.Error() != "custom message" {
		t.Errorf("Error() = %q, want 'custom message'", err.Error())
	}
	if err.Unwrap() != wrapped {
		t.Error("Unwrap() did not return wrapped error")
	}

	t.Run("Hint for different statuses", func(t *testing.T) {
		for _, status := range []Status{StatusDeleted, StatusApproved, StatusRejected, StatusError, StatusActive} {
			err := &StateError{CurrentStatus: status}
			hint := err.Hint()
			if hint == "" {
				t.Errorf("Hint() for status %s should not be empty", status)
			}
		}
	})

	t.Run("NewStateError helper", func(t *testing.T) {
		transErr := &InvalidTransitionError{Current: StatusStopped, Attempted: StatusCreating, Reason: "test"}
		stateErr := NewStateError(transErr)
		if stateErr.CurrentStatus != StatusStopped {
			t.Errorf("CurrentStatus = %s, want 'stopped'", stateErr.CurrentStatus)
		}
		if stateErr.Wrapped != transErr {
			t.Error("Wrapped error not set correctly")
		}
	})
}

func TestDriverError(t *testing.T) {
	originalErr := errors.New("permission denied")
	err := &DriverError{
		Operation: "mount",
		Message:   "permission denied",
		Wrapped:   originalErr,
	}

	if err.HTTPStatus() != http.StatusInternalServerError {
		t.Errorf("HTTPStatus() = %d, want %d", err.HTTPStatus(), http.StatusInternalServerError)
	}
	if !err.IsRetryable() {
		t.Error("IsRetryable() should be true for driver errors")
	}
	msg := err.Error()
	if !containsAll(msg, "driver", "mount", "permission denied") {
		t.Errorf("Error() missing expected parts: %s", msg)
	}
	if err.Unwrap() != originalErr {
		t.Error("Unwrap() did not return wrapped error")
	}

	t.Run("without operation", func(t *testing.T) {
		err := &DriverError{Message: "generic error"}
		msg := err.Error()
		if !containsAll(msg, "driver error", "generic error") {
			t.Errorf("Error() missing expected parts: %s", msg)
		}
	})

	t.Run("Hint for different operations", func(t *testing.T) {
		for _, op := range []string{"mount", "unmount", "cleanup", "unknown"} {
			err := &DriverError{Operation: op}
			hint := err.Hint()
			if hint == "" {
				t.Errorf("Hint() for operation %q should not be empty", op)
			}
		}
	})

	t.Run("NewDriverError helper", func(t *testing.T) {
		origErr := errors.New("test error")
		err := NewDriverError("test_op", origErr)
		if err.Operation != "test_op" {
			t.Errorf("Operation = %q, want 'test_op'", err.Operation)
		}
		if err.Message != "test error" {
			t.Errorf("Message = %q, want 'test error'", err.Message)
		}
	})
}

func TestConcurrentModificationError(t *testing.T) {
	err := &ConcurrentModificationError{
		Resource:        "sandbox",
		ID:              "sb-123",
		ExpectedVersion: 5,
		ActualVersion:   7,
	}

	if err.HTTPStatus() != http.StatusConflict {
		t.Errorf("HTTPStatus() = %d, want %d", err.HTTPStatus(), http.StatusConflict)
	}
	if !err.IsRetryable() {
		t.Error("IsRetryable() should be true for concurrent modification")
	}
	msg := err.Error()
	if !containsAll(msg, "sb-123", "5", "7") {
		t.Errorf("Error() missing expected parts: %s", msg)
	}
	hint := err.Hint()
	if hint == "" {
		t.Error("Hint() should not be empty")
	}

	t.Run("NewConcurrentModificationError helper", func(t *testing.T) {
		err := NewConcurrentModificationError("sb-456", 10, 12)
		if err.Resource != "sandbox" {
			t.Errorf("Resource = %q, want 'sandbox'", err.Resource)
		}
		if err.ID != "sb-456" || err.ExpectedVersion != 10 || err.ActualVersion != 12 {
			t.Error("Fields not set correctly")
		}
	})
}

func TestAlreadyExistsError(t *testing.T) {
	err := &AlreadyExistsError{
		Resource:       "sandbox",
		IdempotencyKey: "idem-abc",
		ExistingID:     "sb-existing",
	}

	if err.HTTPStatus() != http.StatusConflict {
		t.Errorf("HTTPStatus() = %d, want %d", err.HTTPStatus(), http.StatusConflict)
	}
	if err.IsRetryable() {
		t.Error("IsRetryable() should be false for idempotency conflicts")
	}
	msg := err.Error()
	if !containsAll(msg, "sandbox", "idem-abc", "sb-existing") {
		t.Errorf("Error() missing expected parts: %s", msg)
	}
	hint := err.Hint()
	if !containsAll(hint, "sb-existing") {
		t.Errorf("Hint() missing expected parts: %s", hint)
	}

	t.Run("NewAlreadyExistsError helper", func(t *testing.T) {
		err := NewAlreadyExistsError("key-123", "sb-789")
		if err.Resource != "sandbox" {
			t.Errorf("Resource = %q, want 'sandbox'", err.Resource)
		}
		if err.IdempotencyKey != "key-123" || err.ExistingID != "sb-789" {
			t.Error("Fields not set correctly")
		}
	})
}

func TestIdempotentSuccessResult(t *testing.T) {
	result := IdempotentSuccessResult{
		WasNoOp:       true,
		Message:       "Already processed",
		PriorResultID: "result-123",
	}

	if !result.WasNoOp {
		t.Error("WasNoOp should be true")
	}
	if result.Message != "Already processed" {
		t.Errorf("Message = %q, want 'Already processed'", result.Message)
	}
	if result.PriorResultID != "result-123" {
		t.Errorf("PriorResultID = %q, want 'result-123'", result.PriorResultID)
	}
}

func TestDomainErrorInterface(t *testing.T) {
	// Ensure all domain errors implement the DomainError interface
	var _ DomainError = (*NotFoundError)(nil)
	var _ DomainError = (*ScopeConflictError)(nil)
	var _ DomainError = (*ValidationError)(nil)
	var _ DomainError = (*StateError)(nil)
	var _ DomainError = (*DriverError)(nil)
	var _ DomainError = (*ConcurrentModificationError)(nil)
	var _ DomainError = (*AlreadyExistsError)(nil)
}

// ============================================================================
// Helper functions
// ============================================================================

func containsAll(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if !contains(s, sub) {
			return false
		}
	}
	return true
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
