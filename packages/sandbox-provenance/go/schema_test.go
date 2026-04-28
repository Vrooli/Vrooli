package sandboxprovenance

import (
	"errors"
	"testing"
)

func TestSchemaVersionConstant(t *testing.T) {
	if SchemaVersion != "1.0.0" {
		t.Fatalf("SchemaVersion drifted to %q; coordinate a bump with gct-pending-ai-provenance-hardening before changing", SchemaVersion)
	}
}

func TestValidate_HappyPath(t *testing.T) {
	r := Record{
		SchemaVersion: SchemaVersion,
		RunID:         "run-1",
		RunOutcome:    RunOutcomeSuccess,
		State:         FileStateApplied,
		CostUSD:       0.42,
	}
	if err := r.Validate(); err != nil {
		t.Fatalf("Validate() = %v, want nil", err)
	}
}

func TestValidate_LegacyEmptyVersion(t *testing.T) {
	r := Record{RunID: "run-1"} // pre-rollout shape
	if err := r.Validate(); err != nil {
		t.Fatalf("Validate() = %v, want nil for legacy record", err)
	}
}

func TestValidate_UnknownSchemaVersionFailsLoud(t *testing.T) {
	r := Record{SchemaVersion: "2.0.0", RunID: "run-1"}
	err := r.Validate()
	if err == nil {
		t.Fatal("Validate() = nil, want ErrUnknownSchemaVersion")
	}
	if !errors.Is(err, ErrUnknownSchemaVersion) {
		t.Fatalf("Validate() error = %v, want ErrUnknownSchemaVersion", err)
	}
}

func TestValidate_RejectsMissingRunID(t *testing.T) {
	r := Record{SchemaVersion: SchemaVersion}
	if err := r.Validate(); err == nil {
		t.Fatal("Validate() = nil, want error for empty runId")
	}
}

func TestValidate_RejectsInvalidRunOutcome(t *testing.T) {
	r := Record{SchemaVersion: SchemaVersion, RunID: "run-1", RunOutcome: "weird"}
	if err := r.Validate(); err == nil {
		t.Fatal("Validate() = nil, want error for invalid runOutcome")
	}
}

func TestValidate_RejectsInvalidState(t *testing.T) {
	r := Record{SchemaVersion: SchemaVersion, RunID: "run-1", State: "frozen"}
	if err := r.Validate(); err == nil {
		t.Fatal("Validate() = nil, want error for invalid state")
	}
}

func TestValidate_RejectsNegativeCost(t *testing.T) {
	r := Record{SchemaVersion: SchemaVersion, RunID: "run-1", CostUSD: -1}
	if err := r.Validate(); err == nil {
		t.Fatal("Validate() = nil, want error for negative cost")
	}
}

func TestRunOutcome_IsValid(t *testing.T) {
	for _, o := range []RunOutcome{"", RunOutcomeSuccess, RunOutcomeFailure, RunOutcomeCancelled, RunOutcomeTimeout} {
		if !o.IsValid() {
			t.Errorf("expected %q to be valid", o)
		}
	}
	if RunOutcome("nope").IsValid() {
		t.Error("expected 'nope' to be invalid")
	}
}

func TestFileState_IsValid(t *testing.T) {
	for _, s := range []FileState{"", FileStateApplied, FileStatePendingReview, FileStateDenied} {
		if !s.IsValid() {
			t.Errorf("expected %q to be valid", s)
		}
	}
	if FileState("frozen").IsValid() {
		t.Error("expected 'frozen' to be invalid")
	}
}
