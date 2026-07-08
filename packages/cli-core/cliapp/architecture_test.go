package cliapp

import (
	"strings"
	"testing"
)

func TestCommandArchitecture_ZeroIsValid(t *testing.T) {
	var a CommandArchitecture
	if !a.IsZero() {
		t.Fatalf("zero CommandArchitecture should be IsZero")
	}
	if err := a.Validate(); err != nil {
		t.Fatalf("zero CommandArchitecture should validate (metadata is opt-in): %v", err)
	}
}

func TestCommandArchitecture_ValidatesVocabulary(t *testing.T) {
	tests := []struct {
		name    string
		arch    CommandArchitecture
		wantErr string // substring; "" means expect success
	}{
		{"proto_list ok", CommandArchitecture{Primitive: PrimitiveProtoList}, ""},
		{"proto_mutation ok", CommandArchitecture{Primitive: PrimitiveProtoMutation}, ""},
		{"operational ok", CommandArchitecture{Primitive: PrimitiveOperational}, ""},
		{"action ok", CommandArchitecture{Primitive: PrimitiveAction}, ""},
		{"unknown primitive", CommandArchitecture{Primitive: PrimitiveClass("frobnicate")}, "unknown primitive class"},
		{"special-case primitive rejected as declaration", CommandArchitecture{Primitive: PrimitiveDurableRun}, "must be declared as an exception"},
		{"upload primitive rejected as declaration", CommandArchitecture{Primitive: PrimitiveUpload}, "must be declared as an exception"},
		{"exception needs reason", CommandArchitecture{Exception: ExceptionDurableRun}, "requires a reason"},
		{"exception with reason ok", CommandArchitecture{Exception: ExceptionDurableRun, ExceptionReason: "server-owned run lifecycle"}, ""},
		{"unknown exception", CommandArchitecture{Exception: ExceptionClass("nope"), ExceptionReason: "x"}, "unknown exception class"},
		{"reason without exception", CommandArchitecture{ExceptionReason: "orphan reason"}, "reason set without an exception"},
		{"normal primitive plus exception", CommandArchitecture{Primitive: PrimitiveProtoList, Exception: ExceptionUpload, ExceptionReason: "r"}, "cannot also declare exception"},
		{"special-case primitive plus exception still rejected as declaration", CommandArchitecture{Primitive: PrimitiveUpload, Exception: ExceptionUpload, ExceptionReason: "multipart"}, "must be declared as an exception"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.arch.Validate()
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("expected valid, got error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestPrimitiveClass_Classification(t *testing.T) {
	if !PrimitiveProtoList.RequiresProtoBinding() {
		t.Errorf("proto_list should require a proto binding")
	}
	if PrimitiveAction.RequiresProtoBinding() {
		t.Errorf("action should not require a proto binding")
	}
	if !PrimitiveDurableRun.IsSpecialCase() {
		t.Errorf("durable_run should be a special case")
	}
	if PrimitiveProtoMutation.IsSpecialCase() {
		t.Errorf("proto_mutation should not be a special case")
	}
	if got := PrimitiveUpload.SatisfiesException(); got != ExceptionUpload {
		t.Errorf("upload primitive should satisfy the upload exception, got %q", got)
	}
	if got := PrimitiveProtoList.SatisfiesException(); got != "" {
		t.Errorf("normal primitive should satisfy no exception, got %q", got)
	}
}

func TestValidClasses_AreSortedAndComplete(t *testing.T) {
	prims := ValidPrimitiveClasses()
	if len(prims) != len(primitiveClasses) {
		t.Fatalf("ValidPrimitiveClasses returned %d, want %d", len(prims), len(primitiveClasses))
	}
	for i := 1; i < len(prims); i++ {
		if prims[i-1] >= prims[i] {
			t.Fatalf("ValidPrimitiveClasses not sorted at %d: %q >= %q", i, prims[i-1], prims[i])
		}
	}
	excs := ValidExceptionClasses()
	if len(excs) != len(exceptionClasses) {
		t.Fatalf("ValidExceptionClasses returned %d, want %d", len(excs), len(exceptionClasses))
	}
	// Every special-case primitive must map to a real exception class.
	for p, e := range specialCasePrimitives {
		if !p.Valid() {
			t.Errorf("special-case primitive %q is not in the vocabulary", p)
		}
		if !e.Valid() {
			t.Errorf("special-case primitive %q maps to unknown exception %q", p, e)
		}
	}
}
