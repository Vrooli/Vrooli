package domain

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestValidateInteractiveRunMode(t *testing.T) {
	cases := []struct {
		name        string
		exec        ExecutionMode
		sandboxMode SandboxMode
		wantErr     bool
	}{
		{"interactive+protected is policy-resolved", ExecutionModeInteractive, SandboxModeProtected, false},
		{"interactive+tracking allowed", ExecutionModeInteractive, SandboxModeTracking, false},
		{"interactive+off allowed", ExecutionModeInteractive, SandboxModeOff, false},
		{"codec_pipe+protected allowed", ExecutionModeCodecPipe, SandboxModeProtected, false},
		{"empty(default codec)+protected allowed", "", SandboxModeProtected, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateInteractiveRunMode(tc.exec, tc.sandboxMode)
			if tc.wantErr != (err != nil) {
				t.Fatalf("ValidateInteractiveRunMode(%q,%q): wantErr=%v got %v", tc.exec, tc.sandboxMode, tc.wantErr, err)
			}
			if err != nil {
				ve, ok := err.(*ValidationError)
				if !ok {
					t.Fatalf("expected *ValidationError, got %T", err)
				}
				if ve.Field != "executionMode" {
					t.Errorf("field: got %q want executionMode", ve.Field)
				}
				// Any future domain validation error remains actionable.
				hint := strings.ToLower(ve.Hint)
				if !strings.Contains(hint, "tracking") {
					t.Errorf("hint should point to tracking mode, got %q", ve.Hint)
				}
			}
		})
	}
}

// TestRunValidate_InteractiveGate confirms the gate is wired into the canonical
// Run.Validate() domain validation layer, not only the standalone helper.
func TestRunValidate_InteractiveGate(t *testing.T) {
	base := func() *Run {
		id := uuid.New()
		return &Run{
			TaskID:         uuid.New(),
			AgentProfileID: &id,
			RunMode:        RunModeSandboxed,
		}
	}

	protected := base()
	protected.ExecutionMode = ExecutionModeInteractive
	protected.SandboxConfig = &SandboxConfig{Mode: SandboxModeProtected}
	if err := protected.Validate(); err != nil {
		t.Fatalf("interactive + sandboxed should be policy-resolvable: %v", err)
	}

	tracking := base()
	tracking.ExecutionMode = ExecutionModeInteractive
	tracking.SandboxConfig = &SandboxConfig{Mode: SandboxModeTracking}
	if err := tracking.Validate(); err != nil {
		t.Fatalf("interactive + tracking should validate: %v", err)
	}

	codec := base() // codec_pipe + sandboxed
	if err := codec.Validate(); err != nil {
		t.Fatalf("codec-pipe + sandboxed should validate: %v", err)
	}
}
