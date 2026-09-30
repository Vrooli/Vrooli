package cliapp

import (
	"strings"
	"testing"

	"google.golang.org/protobuf/types/known/wrapperspb"
)

// panicManifest is a minimal manifest whose two commands bind proto methods and
// declare architecture primitives. Its handlers panic if invoked, so any test
// building evidence from it proves generation never executes a command.
const panicManifest = `{
  "name": "panic-scenario",
  "groups": [
    {
      "name": "notes",
      "commands": [
        {
          "name": "list",
          "binding": {"kind": "connect-rpc", "service": "NotesService", "method": "ListNotes"},
          "governance": {"effect": "read"},
          "architecture": {"primitive": "proto_list"}
        },
        {
          "name": "create",
          "binding": {"kind": "connect-rpc", "service": "NotesService", "method": "CreateNote"},
          "governance": {"effect": "write"},
          "architecture": {"primitive": "proto_mutation"}
        }
      ]
    }
  ]
}`

// panicBindings builds primitive handlers whose call/report funcs panic. If
// evidence generation ran them, the test would panic; it does not, because
// assembling the tree only wires closures.
func panicBindings() map[string]PrimitiveHandler {
	return map[string]PrimitiveHandler{
		"NotesService.ListNotes": ProtoList(
			func(OperationContext) (*wrapperspb.StringValue, error) {
				panic("list call must not run during evidence generation")
			},
			func(OperationContext, *wrapperspb.StringValue) ListReport {
				panic("list report must not run during evidence generation")
			},
		),
		"NotesService.CreateNote": ProtoMutation(
			func(OperationContext) (*wrapperspb.StringValue, error) {
				panic("create call must not run during evidence generation")
			},
			func(OperationContext, *wrapperspb.StringValue) MutationReport {
				panic("create report must not run during evidence generation")
			},
		),
	}
}

func TestBuildPrimitiveEvidence_DoesNotExecuteHandlers(t *testing.T) {
	group, err := LoadFromManifestPrimitives([]byte(panicManifest), "notes", panicBindings())
	if err != nil {
		t.Fatalf("assemble command tree: %v", err)
	}

	// If BuildPrimitiveEvidence executed any handler, the panics above would fire.
	artifact, err := BuildPrimitiveEvidence(EvidenceExportInput{
		Scenario:    "panic-scenario",
		ManifestRaw: []byte(panicManifest),
		Groups:      []SubcommandGroup{group},
	})
	if err != nil {
		t.Fatalf("BuildPrimitiveEvidence: %v", err)
	}

	if artifact.Schema != EvidenceSchemaID {
		t.Fatalf("artifact schema = %q, want %q", artifact.Schema, EvidenceSchemaID)
	}
	if artifact.Generator != EvidenceGeneratorVersion {
		t.Fatalf("artifact generator = %q, want %q", artifact.Generator, EvidenceGeneratorVersion)
	}
	if artifact.ManifestHash == "" {
		t.Fatalf("artifact should record a manifest hash")
	}
	if len(artifact.Commands) != 2 {
		t.Fatalf("artifact commands = %d, want 2", len(artifact.Commands))
	}

	// Commands are sorted by path: "notes create" then "notes list".
	create := artifact.Commands[0]
	if create.Path != "notes create" || create.ObservedPrimitive != string(PrimitiveProtoMutation) {
		t.Fatalf("create entry wrong: %+v", create)
	}
	if create.DeclaredPrimitive != string(PrimitiveProtoMutation) {
		t.Fatalf("create declared primitive = %q", create.DeclaredPrimitive)
	}
	if create.Binding != "NotesService.CreateNote" {
		t.Fatalf("create binding = %q", create.Binding)
	}
	list := artifact.Commands[1]
	if list.Path != "notes list" || list.ObservedPrimitive != string(PrimitiveProtoList) {
		t.Fatalf("list entry wrong: %+v", list)
	}

	observed := artifact.ObservedPrimitives()
	if observed["notes list"] != PrimitiveProtoList || observed["notes create"] != PrimitiveProtoMutation {
		t.Fatalf("observed primitives map wrong: %+v", observed)
	}
}

func TestBuildPrimitiveEvidence_FailsOnContradiction(t *testing.T) {
	// A top-level command whose declared primitive contradicts its observed
	// evidence must fail the export (never recorded as if it were evidence).
	cmd := Command{
		Name:              "list",
		Architecture:      CommandArchitecture{Primitive: PrimitiveProtoList},
		primitiveEvidence: PrimitiveProtoMutation,
	}
	_, err := BuildPrimitiveEvidence(EvidenceExportInput{
		Scenario: "x",
		TopLevel: []Command{cmd},
	})
	if err == nil || !strings.Contains(err.Error(), "proto_mutation") {
		t.Fatalf("expected contradiction error, got %v", err)
	}
}

func TestParsePrimitiveEvidence_RoundTripAndSchemaGuard(t *testing.T) {
	artifact := PrimitiveEvidenceArtifact{
		Schema:    EvidenceSchemaID,
		Scenario:  "demo",
		Generator: EvidenceGeneratorVersion,
		Commands: []CommandPrimitiveEvidence{
			{Path: "notes list", Command: "list", Group: "notes", ObservedPrimitive: string(PrimitiveProtoList)},
		},
	}
	body, err := MarshalPrimitiveEvidence(artifact)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if body[len(body)-1] != '\n' {
		t.Fatalf("marshaled artifact must end with a newline")
	}
	got, err := ParsePrimitiveEvidence(body)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got.Scenario != "demo" || len(got.Commands) != 1 {
		t.Fatalf("round-trip mismatch: %+v", got)
	}

	if _, err := ParsePrimitiveEvidence([]byte(`{"schema":"other/v9","commands":[]}`)); err == nil {
		t.Fatalf("expected schema-guard error for unknown schema")
	}
}
