package cliapp

import (
	"strings"
	"testing"

	"google.golang.org/protobuf/types/known/wrapperspb"
)

// evidenceManifest builds a one-command manifest whose single command binds
// NotesService.ListNotes, with an optional architecture fragment (leading-comma
// JSON, e.g. `, "architecture": {...}`).
func evidenceManifest(archFragment string) []byte {
	return []byte(`{
  "name": "demo",
  "groups": [
    {
      "name": "notes",
      "commands": [
        {
          "name": "list",
          "binding": { "kind": "connect-rpc", "service": "NotesService", "method": "ListNotes" },
          "governance": { "effect": "read", "run_eligible": true }` + archFragment + `
        }
      ]
    }
  ]
}`)
}

func listPrimitive() PrimitiveHandler {
	return ProtoList(
		func(ctx OperationContext) (*wrapperspb.StringValue, error) { return wrapperspb.String("x"), nil },
		func(ctx OperationContext, resp *wrapperspb.StringValue) ListReport {
			return ListReport{Results: []string{resp.GetValue()}}
		},
	)
}

func TestClassifyPrimitiveEvidence(t *testing.T) {
	tests := []struct {
		name     string
		declared PrimitiveClass
		observed PrimitiveClass
		want     EvidenceStatus
	}{
		{"neither", "", "", EvidenceNone},
		{"observed only", "", PrimitiveProtoList, EvidenceObservedOnly},
		{"declared only", PrimitiveProtoList, "", EvidenceDeclaredOnly},
		{"verified", PrimitiveProtoList, PrimitiveProtoList, EvidenceVerified},
		{"contradiction", PrimitiveProtoList, PrimitiveProtoMutation, EvidenceContradiction},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := ClassifyPrimitiveEvidence(tc.declared, tc.observed); got != tc.want {
				t.Fatalf("ClassifyPrimitiveEvidence(%q,%q) = %q, want %q", tc.declared, tc.observed, got, tc.want)
			}
		})
	}
}

// A primitive-built command carries observed evidence automatically, so the
// evidence is proven by construction and cannot be claimed by manifest text.
func TestLoadFromManifestPrimitives_AttachesEvidence(t *testing.T) {
	raw := evidenceManifest(`, "architecture": { "primitive": "proto_list" }`)
	bindings := map[string]PrimitiveHandler{"NotesService.ListNotes": listPrimitive()}
	group, err := LoadFromManifestPrimitives(raw, "notes", bindings)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(group.Subcommands) != 1 {
		t.Fatalf("expected 1 subcommand, got %d", len(group.Subcommands))
	}
	cmd := group.Subcommands[0]
	if cmd.PrimitiveEvidence() != PrimitiveProtoList {
		t.Fatalf("evidence = %q, want proto_list", cmd.PrimitiveEvidence())
	}
	if cmd.Architecture.Primitive != PrimitiveProtoList {
		t.Fatalf("declared primitive = %q, want proto_list", cmd.Architecture.Primitive)
	}
	if got := ClassifyPrimitiveEvidence(cmd.Architecture.Primitive, cmd.PrimitiveEvidence()); got != EvidenceVerified {
		t.Fatalf("expected verified, got %q", got)
	}
}

// A declaration alone (legacy bindings map, no primitive builder) yields no
// evidence: it is declaration-only debt, not verified maturity.
func TestLoadFromManifest_DeclarationWithoutEvidence(t *testing.T) {
	raw := evidenceManifest(`, "architecture": { "primitive": "proto_list" }`)
	bindings := map[string]func(RunContext) error{
		"NotesService.ListNotes": func(ctx RunContext) error { return nil },
	}
	group, err := LoadFromManifest(raw, "notes", bindings)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	cmd := group.Subcommands[0]
	if cmd.PrimitiveEvidence() != "" {
		t.Fatalf("legacy handler should carry no evidence, got %q", cmd.PrimitiveEvidence())
	}
	if got := ClassifyPrimitiveEvidence(cmd.Architecture.Primitive, cmd.PrimitiveEvidence()); got != EvidenceDeclaredOnly {
		t.Fatalf("expected declared_only, got %q", got)
	}
}

// A manifest that declares one primitive while the handler is built from another
// is a contradiction and must fail fast at load.
func TestLoadFromManifestPrimitives_ContradictionFails(t *testing.T) {
	raw := evidenceManifest(`, "architecture": { "primitive": "proto_mutation" }`)
	bindings := map[string]PrimitiveHandler{"NotesService.ListNotes": listPrimitive()} // proto_list evidence
	_, err := LoadFromManifestPrimitives(raw, "notes", bindings)
	if err == nil || !strings.Contains(err.Error(), "proto_list") {
		t.Fatalf("expected contradiction error mentioning proto_list, got %v", err)
	}
}

// Evidence with no declaration is allowed at load (observed-only): CLI Health
// classifies it, cli-core does not reject it.
func TestLoadFromManifestPrimitives_ObservedOnlyAllowed(t *testing.T) {
	raw := evidenceManifest(``)
	bindings := map[string]PrimitiveHandler{"NotesService.ListNotes": listPrimitive()}
	group, err := LoadFromManifestPrimitives(raw, "notes", bindings)
	if err != nil {
		t.Fatalf("observed-only should load, got %v", err)
	}
	cmd := group.Subcommands[0]
	if got := ClassifyPrimitiveEvidence(cmd.Architecture.Primitive, cmd.PrimitiveEvidence()); got != EvidenceObservedOnly {
		t.Fatalf("expected observed_only, got %q", got)
	}
}

func TestLoadFromManifestPrimitives_NilRunFails(t *testing.T) {
	raw := evidenceManifest(``)
	bindings := map[string]PrimitiveHandler{"NotesService.ListNotes": {primitive: PrimitiveProtoList}}
	if _, err := LoadFromManifestPrimitives(raw, "notes", bindings); err == nil || !strings.Contains(err.Error(), "nil Run") {
		t.Fatalf("expected nil Run error, got %v", err)
	}
}

func TestCommand_WithPrimitive(t *testing.T) {
	ran := false
	ph := PrimitiveHandler{primitive: PrimitiveProtoList, Run: func(ctx RunContext) error { ran = true; return nil }}
	cmd := Command{Name: "list"}.WithPrimitive(ph)
	if cmd.PrimitiveEvidence() != PrimitiveProtoList {
		t.Fatalf("WithPrimitive evidence = %q, want proto_list", cmd.PrimitiveEvidence())
	}
	if cmd.RunCtx == nil {
		t.Fatal("WithPrimitive did not set RunCtx")
	}
	_ = cmd.RunCtx(NewTestRunContext(TestRunContextOptions{Schema: ArgSchema{}}))
	if !ran {
		t.Fatal("WithPrimitive RunCtx did not invoke the handler")
	}
}
