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
// GC (Garbage Collection) Types Tests [OT-P1-003]
// ============================================================================

func TestDefaultGCPolicy(t *testing.T) {
	policy := DefaultGCPolicy()

	// Verify default MaxAge is 24 hours
	if policy.MaxAge != 24*time.Hour {
		t.Errorf("DefaultGCPolicy().MaxAge = %v, want 24h", policy.MaxAge)
	}

	// Verify default IdleTimeout is 4 hours
	if policy.IdleTimeout != 4*time.Hour {
		t.Errorf("DefaultGCPolicy().IdleTimeout = %v, want 4h", policy.IdleTimeout)
	}

	// Verify IncludeTerminal is true
	if !policy.IncludeTerminal {
		t.Error("DefaultGCPolicy().IncludeTerminal = false, want true")
	}

	// Verify TerminalDelay is 1 hour
	if policy.TerminalDelay != 1*time.Hour {
		t.Errorf("DefaultGCPolicy().TerminalDelay = %v, want 1h", policy.TerminalDelay)
	}

	// Verify MaxTotalSizeBytes is not set (0)
	if policy.MaxTotalSizeBytes != 0 {
		t.Errorf("DefaultGCPolicy().MaxTotalSizeBytes = %d, want 0", policy.MaxTotalSizeBytes)
	}

	// Verify Statuses includes the expected statuses
	expectedStatuses := []Status{StatusStopped, StatusError, StatusApproved, StatusRejected}
	if len(policy.Statuses) != len(expectedStatuses) {
		t.Errorf("DefaultGCPolicy().Statuses length = %d, want %d", len(policy.Statuses), len(expectedStatuses))
	}

	// Check all expected statuses are present
	statusSet := make(map[Status]bool)
	for _, s := range policy.Statuses {
		statusSet[s] = true
	}
	for _, expected := range expectedStatuses {
		if !statusSet[expected] {
			t.Errorf("DefaultGCPolicy().Statuses missing %s", expected)
		}
	}
}

func TestGCPolicyStructure(t *testing.T) {
	policy := GCPolicy{
		MaxAge:            48 * time.Hour,
		IdleTimeout:       8 * time.Hour,
		MaxTotalSizeBytes: 10 * 1024 * 1024 * 1024, // 10GB
		IncludeTerminal:   true,
		TerminalDelay:     2 * time.Hour,
		Statuses:          []Status{StatusStopped, StatusError},
	}

	if policy.MaxAge != 48*time.Hour {
		t.Errorf("GCPolicy.MaxAge = %v, want 48h", policy.MaxAge)
	}
	if policy.IdleTimeout != 8*time.Hour {
		t.Errorf("GCPolicy.IdleTimeout = %v, want 8h", policy.IdleTimeout)
	}
	if policy.MaxTotalSizeBytes != 10*1024*1024*1024 {
		t.Errorf("GCPolicy.MaxTotalSizeBytes = %d, want 10GB", policy.MaxTotalSizeBytes)
	}
	if !policy.IncludeTerminal {
		t.Error("GCPolicy.IncludeTerminal = false, want true")
	}
	if policy.TerminalDelay != 2*time.Hour {
		t.Errorf("GCPolicy.TerminalDelay = %v, want 2h", policy.TerminalDelay)
	}
	if len(policy.Statuses) != 2 {
		t.Errorf("GCPolicy.Statuses length = %d, want 2", len(policy.Statuses))
	}
}

func TestGCRequestStructure(t *testing.T) {
	policy := DefaultGCPolicy()
	request := GCRequest{
		Policy: &policy,
		DryRun: true,
		Limit:  100,
		Actor:  "scheduler",
	}

	if request.Policy == nil {
		t.Error("GCRequest.Policy should not be nil")
	}
	if !request.DryRun {
		t.Error("GCRequest.DryRun = false, want true")
	}
	if request.Limit != 100 {
		t.Errorf("GCRequest.Limit = %d, want 100", request.Limit)
	}
	if request.Actor != "scheduler" {
		t.Errorf("GCRequest.Actor = %q, want 'scheduler'", request.Actor)
	}
}

func TestGCResultStructure(t *testing.T) {
	now := time.Now()
	result := GCResult{
		Collected: []*GCCollectedSandbox{
			{
				ID:        uuid.New(),
				ScopePath: "/project/test",
				Status:    StatusStopped,
				SizeBytes: 1024,
				CreatedAt: now.Add(-48 * time.Hour),
				Reason:    "exceeded max age",
			},
		},
		TotalCollected:      1,
		TotalBytesReclaimed: 1024,
		Errors:              []GCError{},
		DryRun:              false,
		StartedAt:           now.Add(-1 * time.Minute),
		CompletedAt:         now,
		Reasons:             map[string][]string{"test-id": {"age", "idle"}},
	}

	if result.TotalCollected != 1 {
		t.Errorf("GCResult.TotalCollected = %d, want 1", result.TotalCollected)
	}
	if result.TotalBytesReclaimed != 1024 {
		t.Errorf("GCResult.TotalBytesReclaimed = %d, want 1024", result.TotalBytesReclaimed)
	}
	if result.DryRun {
		t.Error("GCResult.DryRun = true, want false")
	}
	if len(result.Collected) != 1 {
		t.Errorf("GCResult.Collected length = %d, want 1", len(result.Collected))
	}
	if result.Collected[0].Reason != "exceeded max age" {
		t.Errorf("GCResult.Collected[0].Reason = %q, want 'exceeded max age'", result.Collected[0].Reason)
	}
}

func TestGCCollectedSandboxStructure(t *testing.T) {
	now := time.Now()
	id := uuid.New()
	collected := GCCollectedSandbox{
		ID:        id,
		ScopePath: "/project/old",
		Status:    StatusApproved,
		SizeBytes: 5000,
		CreatedAt: now.Add(-72 * time.Hour),
		Reason:    "terminal state cleanup",
	}

	if collected.ID != id {
		t.Error("GCCollectedSandbox.ID not set correctly")
	}
	if collected.ScopePath != "/project/old" {
		t.Errorf("GCCollectedSandbox.ScopePath = %q, want '/project/old'", collected.ScopePath)
	}
	if collected.Status != StatusApproved {
		t.Errorf("GCCollectedSandbox.Status = %s, want 'approved'", collected.Status)
	}
	if collected.SizeBytes != 5000 {
		t.Errorf("GCCollectedSandbox.SizeBytes = %d, want 5000", collected.SizeBytes)
	}
	if collected.Reason != "terminal state cleanup" {
		t.Errorf("GCCollectedSandbox.Reason = %q, want 'terminal state cleanup'", collected.Reason)
	}
}

func TestGCErrorStructure(t *testing.T) {
	id := uuid.New()
	gcErr := GCError{
		SandboxID: id,
		Error:     "failed to unmount: device busy",
	}

	if gcErr.SandboxID != id {
		t.Error("GCError.SandboxID not set correctly")
	}
	if gcErr.Error != "failed to unmount: device busy" {
		t.Errorf("GCError.Error = %q, want 'failed to unmount: device busy'", gcErr.Error)
	}
}

// ============================================================================
// Rebase Types Tests [OT-P2-003]
// ============================================================================

func TestRebaseStrategyConstants(t *testing.T) {
	if RebaseStrategyRegenerate != "regenerate" {
		t.Errorf("RebaseStrategyRegenerate = %q, want 'regenerate'", RebaseStrategyRegenerate)
	}
}

func TestRebaseRequestStructure(t *testing.T) {
	id := uuid.New()
	req := RebaseRequest{
		SandboxID: id,
		Strategy:  RebaseStrategyRegenerate,
		Actor:     "admin",
	}

	if req.SandboxID != id {
		t.Error("RebaseRequest.SandboxID not set correctly")
	}
	if req.Strategy != RebaseStrategyRegenerate {
		t.Errorf("RebaseRequest.Strategy = %q, want 'regenerate'", req.Strategy)
	}
	if req.Actor != "admin" {
		t.Errorf("RebaseRequest.Actor = %q, want 'admin'", req.Actor)
	}
}

func TestRebaseResultStructure(t *testing.T) {
	now := time.Now()
	result := RebaseResult{
		Success:          true,
		PreviousBaseHash: "abc123",
		NewBaseHash:      "def456",
		ConflictingFiles: []string{"file1.go", "file2.go"},
		RepoChangedFiles: []string{"file1.go", "file2.go", "file3.go"},
		Strategy:         RebaseStrategyRegenerate,
		ErrorMsg:         "",
		RebasedAt:        now,
	}

	if !result.Success {
		t.Error("RebaseResult.Success = false, want true")
	}
	if result.PreviousBaseHash != "abc123" {
		t.Errorf("RebaseResult.PreviousBaseHash = %q, want 'abc123'", result.PreviousBaseHash)
	}
	if result.NewBaseHash != "def456" {
		t.Errorf("RebaseResult.NewBaseHash = %q, want 'def456'", result.NewBaseHash)
	}
	if len(result.ConflictingFiles) != 2 {
		t.Errorf("RebaseResult.ConflictingFiles length = %d, want 2", len(result.ConflictingFiles))
	}
	if len(result.RepoChangedFiles) != 3 {
		t.Errorf("RebaseResult.RepoChangedFiles length = %d, want 3", len(result.RepoChangedFiles))
	}
	if result.Strategy != RebaseStrategyRegenerate {
		t.Errorf("RebaseResult.Strategy = %q, want 'regenerate'", result.Strategy)
	}
	if result.ErrorMsg != "" {
		t.Errorf("RebaseResult.ErrorMsg = %q, want ''", result.ErrorMsg)
	}
}

func TestRebaseResultFailure(t *testing.T) {
	result := RebaseResult{
		Success:  false,
		ErrorMsg: "merge conflict detected",
		Strategy: RebaseStrategyRegenerate,
	}

	if result.Success {
		t.Error("RebaseResult.Success = true, want false")
	}
	if result.ErrorMsg != "merge conflict detected" {
		t.Errorf("RebaseResult.ErrorMsg = %q, want 'merge conflict detected'", result.ErrorMsg)
	}
}

// ============================================================================
// Conflict Detection Types Tests [OT-P2-002]
// ============================================================================

func TestConflictInfoStructure(t *testing.T) {
	info := ConflictInfo{
		HasConflict:      true,
		BaseCommitHash:   "abc123",
		CurrentHash:      "xyz789",
		ConflictingFiles: []string{"main.go", "config.go"},
		RepoChangedFiles: []string{"main.go", "config.go", "readme.md"},
	}

	if !info.HasConflict {
		t.Error("ConflictInfo.HasConflict = false, want true")
	}
	if info.BaseCommitHash != "abc123" {
		t.Errorf("ConflictInfo.BaseCommitHash = %q, want 'abc123'", info.BaseCommitHash)
	}
	if info.CurrentHash != "xyz789" {
		t.Errorf("ConflictInfo.CurrentHash = %q, want 'xyz789'", info.CurrentHash)
	}
	if len(info.ConflictingFiles) != 2 {
		t.Errorf("ConflictInfo.ConflictingFiles length = %d, want 2", len(info.ConflictingFiles))
	}
	if len(info.RepoChangedFiles) != 3 {
		t.Errorf("ConflictInfo.RepoChangedFiles length = %d, want 3", len(info.RepoChangedFiles))
	}
}

func TestConflictCheckResponseStructure(t *testing.T) {
	now := time.Now()
	resp := ConflictCheckResponse{
		HasConflict:         true,
		BaseCommitHash:      "base123",
		CurrentHash:         "current456",
		RepoChangedFiles:    []string{"a.go", "b.go"},
		SandboxChangedFiles: []string{"a.go", "c.go"},
		ConflictingFiles:    []string{"a.go"},
		CheckedAt:           now,
	}

	if !resp.HasConflict {
		t.Error("ConflictCheckResponse.HasConflict = false, want true")
	}
	if resp.BaseCommitHash != "base123" {
		t.Errorf("ConflictCheckResponse.BaseCommitHash = %q, want 'base123'", resp.BaseCommitHash)
	}
	if resp.CurrentHash != "current456" {
		t.Errorf("ConflictCheckResponse.CurrentHash = %q, want 'current456'", resp.CurrentHash)
	}
	if len(resp.ConflictingFiles) != 1 || resp.ConflictingFiles[0] != "a.go" {
		t.Errorf("ConflictCheckResponse.ConflictingFiles = %v, want ['a.go']", resp.ConflictingFiles)
	}
}

// ============================================================================
// Path Validation Types Tests
// ============================================================================

func TestPathValidationResultStructure(t *testing.T) {
	result := PathValidationResult{
		Path:              "/project/src",
		ProjectRoot:       "/project",
		Valid:             true,
		Exists:            true,
		IsDirectory:       true,
		WithinProjectRoot: true,
		Error:             "",
	}

	if result.Path != "/project/src" {
		t.Errorf("PathValidationResult.Path = %q, want '/project/src'", result.Path)
	}
	if !result.Valid {
		t.Error("PathValidationResult.Valid = false, want true")
	}
	if !result.Exists {
		t.Error("PathValidationResult.Exists = false, want true")
	}
	if !result.IsDirectory {
		t.Error("PathValidationResult.IsDirectory = false, want true")
	}
	if !result.WithinProjectRoot {
		t.Error("PathValidationResult.WithinProjectRoot = false, want true")
	}
}

func TestPathValidationResultInvalid(t *testing.T) {
	result := PathValidationResult{
		Path:              "/etc/passwd",
		ProjectRoot:       "/project",
		Valid:             false,
		WithinProjectRoot: false,
		Error:             "path is outside project root",
	}

	if result.Valid {
		t.Error("PathValidationResult.Valid = true, want false")
	}
	if result.WithinProjectRoot {
		t.Error("PathValidationResult.WithinProjectRoot = true, want false")
	}
	if result.Error != "path is outside project root" {
		t.Errorf("PathValidationResult.Error = %q, want 'path is outside project root'", result.Error)
	}
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
