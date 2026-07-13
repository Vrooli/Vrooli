package domain

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestValidateInteractiveRunMode(t *testing.T) {
	cases := []struct {
		name    string
		exec    ExecutionMode
		runMode RunMode
		wantErr bool
	}{
		{"interactive+sandboxed rejected", ExecutionModeInteractive, RunModeSandboxed, true},
		{"interactive+in_place allowed", ExecutionModeInteractive, RunModeInPlace, false},
		{"codec_pipe+sandboxed allowed", ExecutionModeCodecPipe, RunModeSandboxed, false},
		{"codec_pipe+in_place allowed", ExecutionModeCodecPipe, RunModeInPlace, false},
		{"empty(default codec)+sandboxed allowed", "", RunModeSandboxed, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateInteractiveRunMode(tc.exec, tc.runMode)
			if tc.wantErr != (err != nil) {
				t.Fatalf("ValidateInteractiveRunMode(%q,%q): wantErr=%v got %v", tc.exec, tc.runMode, tc.wantErr, err)
			}
			if err != nil {
				ve, ok := err.(*ValidationError)
				if !ok {
					t.Fatalf("expected *ValidationError, got %T", err)
				}
				if ve.Field != "executionMode" {
					t.Errorf("field: got %q want executionMode", ve.Field)
				}
				// The error must be actionable: say WHY (protected/sandboxed) and
				// WHAT to do (in-place / sandbox off).
				hint := strings.ToLower(ve.Hint)
				if !strings.Contains(hint, "in-place") && !strings.Contains(hint, "in_place") {
					t.Errorf("hint should point to in-place mode, got %q", ve.Hint)
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
	if err := protected.Validate(); err == nil {
		t.Fatal("expected Run.Validate to reject interactive + sandboxed")
	}

	inPlace := base()
	inPlace.RunMode = RunModeInPlace
	inPlace.ExecutionMode = ExecutionModeInteractive
	if err := inPlace.Validate(); err != nil {
		t.Fatalf("interactive + in_place should validate: %v", err)
	}

	codec := base() // codec_pipe + sandboxed
	if err := codec.Validate(); err != nil {
		t.Fatalf("codec-pipe + sandboxed should validate: %v", err)
	}
}
