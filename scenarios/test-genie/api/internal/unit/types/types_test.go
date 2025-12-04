package types

import (
	"context"
	"errors"
	"io"
	"testing"
)

func TestNewSectionObservation(t *testing.T) {
	obs := NewSectionObservation("🧪", "Testing")
	if obs.Type != ObservationSection {
		t.Errorf("Type = %v, want %v", obs.Type, ObservationSection)
	}
	if obs.Icon != "🧪" {
		t.Errorf("Icon = %q, want %q", obs.Icon, "🧪")
	}
	if obs.Message != "Testing" {
		t.Errorf("Message = %q, want %q", obs.Message, "Testing")
	}
}

func TestNewSuccessObservation(t *testing.T) {
	obs := NewSuccessObservation("tests passed")
	if obs.Type != ObservationSuccess {
		t.Errorf("Type = %v, want %v", obs.Type, ObservationSuccess)
	}
	if obs.Message != "tests passed" {
		t.Errorf("Message = %q, want %q", obs.Message, "tests passed")
	}
}

func TestNewWarningObservation(t *testing.T) {
	obs := NewWarningObservation("some warning")
	if obs.Type != ObservationWarning {
		t.Errorf("Type = %v, want %v", obs.Type, ObservationWarning)
	}
	if obs.Message != "some warning" {
		t.Errorf("Message = %q, want %q", obs.Message, "some warning")
	}
}

func TestNewErrorObservation(t *testing.T) {
	obs := NewErrorObservation("something failed")
	if obs.Type != ObservationError {
		t.Errorf("Type = %v, want %v", obs.Type, ObservationError)
	}
	if obs.Message != "something failed" {
		t.Errorf("Message = %q, want %q", obs.Message, "something failed")
	}
}

func TestNewInfoObservation(t *testing.T) {
	obs := NewInfoObservation("FYI")
	if obs.Type != ObservationInfo {
		t.Errorf("Type = %v, want %v", obs.Type, ObservationInfo)
	}
	if obs.Message != "FYI" {
		t.Errorf("Message = %q, want %q", obs.Message, "FYI")
	}
}

func TestNewSkipObservation(t *testing.T) {
	obs := NewSkipObservation("not applicable")
	if obs.Type != ObservationSkip {
		t.Errorf("Type = %v, want %v", obs.Type, ObservationSkip)
	}
	if obs.Message != "not applicable" {
		t.Errorf("Message = %q, want %q", obs.Message, "not applicable")
	}
}

func TestOK(t *testing.T) {
	r := OK()
	if !r.Success {
		t.Error("OK() should return Success = true")
	}
	if r.FailureClass != FailureClassNone {
		t.Errorf("FailureClass = %q, want empty", r.FailureClass)
	}
}

func TestOKWithCoverage(t *testing.T) {
	r := OKWithCoverage("85.5")
	if !r.Success {
		t.Error("OKWithCoverage() should return Success = true")
	}
	if r.Coverage != "85.5" {
		t.Errorf("Coverage = %q, want %q", r.Coverage, "85.5")
	}
}

func TestSkip(t *testing.T) {
	r := Skip("no tests found")
	if !r.Success {
		t.Error("Skip() should return Success = true")
	}
	if !r.Skipped {
		t.Error("Skip() should set Skipped = true")
	}
	if r.SkipReason != "no tests found" {
		t.Errorf("SkipReason = %q, want %q", r.SkipReason, "no tests found")
	}
}

func TestFail(t *testing.T) {
	err := errors.New("test error")
	r := Fail(err, FailureClassTestFailure, "fix the tests")
	if r.Success {
		t.Error("Fail() should return Success = false")
	}
	if r.Error != err {
		t.Errorf("Error = %v, want %v", r.Error, err)
	}
	if r.FailureClass != FailureClassTestFailure {
		t.Errorf("FailureClass = %q, want %q", r.FailureClass, FailureClassTestFailure)
	}
	if r.Remediation != "fix the tests" {
		t.Errorf("Remediation = %q, want %q", r.Remediation, "fix the tests")
	}
}

func TestFailMisconfiguration(t *testing.T) {
	r := FailMisconfiguration(nil, "fix config")
	if r.FailureClass != FailureClassMisconfiguration {
		t.Errorf("FailureClass = %q, want %q", r.FailureClass, FailureClassMisconfiguration)
	}
}

func TestFailMissingDependency(t *testing.T) {
	r := FailMissingDependency(nil, "install go")
	if r.FailureClass != FailureClassMissingDependency {
		t.Errorf("FailureClass = %q, want %q", r.FailureClass, FailureClassMissingDependency)
	}
}

func TestFailTestFailure(t *testing.T) {
	r := FailTestFailure(nil, "fix tests")
	if r.FailureClass != FailureClassTestFailure {
		t.Errorf("FailureClass = %q, want %q", r.FailureClass, FailureClassTestFailure)
	}
}

func TestFailSystem(t *testing.T) {
	r := FailSystem(nil, "check system")
	if r.FailureClass != FailureClassSystem {
		t.Errorf("FailureClass = %q, want %q", r.FailureClass, FailureClassSystem)
	}
}

func TestResult_WithObservations(t *testing.T) {
	r := OK()
	obs1 := NewSuccessObservation("first")
	obs2 := NewInfoObservation("second")

	r = r.WithObservations(obs1, obs2)

	if len(r.Observations) != 2 {
		t.Errorf("len(Observations) = %d, want 2", len(r.Observations))
	}
	if r.Observations[0].Message != "first" {
		t.Errorf("Observations[0].Message = %q, want %q", r.Observations[0].Message, "first")
	}
	if r.Observations[1].Message != "second" {
		t.Errorf("Observations[1].Message = %q, want %q", r.Observations[1].Message, "second")
	}
}

func TestNewDefaultExecutor(t *testing.T) {
	e := NewDefaultExecutor()
	if e == nil {
		t.Error("NewDefaultExecutor() returned nil")
	}
}

func TestDefaultExecutor_Run_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	e := NewDefaultExecutor()
	err := e.Run(ctx, "", nil, "echo", "hello")
	if err == nil {
		t.Error("Run() should fail when context is cancelled")
	}
}

func TestDefaultExecutor_Run_Success(t *testing.T) {
	e := NewDefaultExecutor()
	err := e.Run(context.Background(), "", io.Discard, "echo", "hello")
	if err != nil {
		t.Errorf("Run() error = %v, want nil", err)
	}
}

func TestDefaultExecutor_Run_WithDir(t *testing.T) {
	e := NewDefaultExecutor()
	err := e.Run(context.Background(), "/tmp", io.Discard, "pwd")
	if err != nil {
		t.Errorf("Run() error = %v, want nil", err)
	}
}

func TestDefaultExecutor_Run_NilLogWriter(t *testing.T) {
	e := NewDefaultExecutor()
	err := e.Run(context.Background(), "", nil, "echo", "hello")
	if err != nil {
		t.Errorf("Run() error = %v, want nil", err)
	}
}

func TestDefaultExecutor_Capture_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	e := NewDefaultExecutor()
	_, err := e.Capture(ctx, "", nil, "echo", "hello")
	if err == nil {
		t.Error("Capture() should fail when context is cancelled")
	}
}

func TestDefaultExecutor_Capture_Success(t *testing.T) {
	e := NewDefaultExecutor()
	output, err := e.Capture(context.Background(), "", nil, "echo", "hello")
	if err != nil {
		t.Errorf("Capture() error = %v, want nil", err)
	}
	if output != "hello\n" {
		t.Errorf("Capture() output = %q, want %q", output, "hello\n")
	}
}

func TestDefaultExecutor_Capture_WithDir(t *testing.T) {
	e := NewDefaultExecutor()
	output, err := e.Capture(context.Background(), "/tmp", nil, "pwd")
	if err != nil {
		t.Errorf("Capture() error = %v, want nil", err)
	}
	if output != "/tmp\n" {
		t.Errorf("Capture() output = %q, want %q", output, "/tmp\n")
	}
}

func TestDefaultExecutor_Capture_WithLogWriter(t *testing.T) {
	e := NewDefaultExecutor()
	output, err := e.Capture(context.Background(), "", io.Discard, "echo", "hello")
	if err != nil {
		t.Errorf("Capture() error = %v, want nil", err)
	}
	if output != "hello\n" {
		t.Errorf("Capture() output = %q, want %q", output, "hello\n")
	}
}

func TestDefaultExecutor_LookPath_Exists(t *testing.T) {
	e := NewDefaultExecutor()
	path, err := e.LookPath("echo")
	if err != nil {
		t.Errorf("LookPath() error = %v, want nil", err)
	}
	if path == "" {
		t.Error("LookPath() returned empty path")
	}
}

func TestDefaultExecutor_LookPath_NotExists(t *testing.T) {
	e := NewDefaultExecutor()
	_, err := e.LookPath("nonexistent-command-xyz")
	if err == nil {
		t.Error("LookPath() should fail for nonexistent command")
	}
}

func TestEnsureCommand_Success(t *testing.T) {
	e := NewDefaultExecutor()
	err := EnsureCommand(e, "echo")
	if err != nil {
		t.Errorf("EnsureCommand() error = %v, want nil", err)
	}
}

func TestEnsureCommand_Failure(t *testing.T) {
	e := NewDefaultExecutor()
	err := EnsureCommand(e, "nonexistent-command-xyz")
	if err == nil {
		t.Error("EnsureCommand() should fail for nonexistent command")
	}
}

func TestFailureClassConstants(t *testing.T) {
	// Verify the constants have expected values
	if FailureClassNone != "" {
		t.Errorf("FailureClassNone = %q, want empty", FailureClassNone)
	}
	if FailureClassMisconfiguration != "misconfiguration" {
		t.Errorf("FailureClassMisconfiguration = %q, want %q", FailureClassMisconfiguration, "misconfiguration")
	}
	if FailureClassMissingDependency != "missing_dependency" {
		t.Errorf("FailureClassMissingDependency = %q, want %q", FailureClassMissingDependency, "missing_dependency")
	}
	if FailureClassTestFailure != "test_failure" {
		t.Errorf("FailureClassTestFailure = %q, want %q", FailureClassTestFailure, "test_failure")
	}
	if FailureClassSystem != "system" {
		t.Errorf("FailureClassSystem = %q, want %q", FailureClassSystem, "system")
	}
}

func TestObservationTypeConstants(t *testing.T) {
	// Verify observation types are distinct
	types := []ObservationType{
		ObservationSection,
		ObservationSuccess,
		ObservationWarning,
		ObservationError,
		ObservationInfo,
		ObservationSkip,
	}
	seen := make(map[ObservationType]bool)
	for _, typ := range types {
		if seen[typ] {
			t.Errorf("Duplicate ObservationType value: %v", typ)
		}
		seen[typ] = true
	}
}
