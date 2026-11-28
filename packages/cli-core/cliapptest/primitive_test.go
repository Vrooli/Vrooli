package cliapptest

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestRunPrimitiveModes_DrivesBothOutputModes(t *testing.T) {
	// A primitive handler whose call func needs no core, so the helper can be
	// exercised without a fake API server.
	handler := cliapp.ProtoList(
		func(ctx cliapp.OperationContext) (*wrapperspb.StringValue, error) {
			return wrapperspb.String("value-" + ctx.Positional("name")), nil
		},
		func(ctx cliapp.OperationContext, resp *wrapperspb.StringValue) cliapp.ListReport {
			return cliapp.ListReport{Summary: []string{"one"}, Results: []string{resp.GetValue()}}
		},
	)
	schema := cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "name"}}}

	if handler.Primitive() != cliapp.PrimitiveProtoList {
		t.Fatalf("ProtoList should carry proto_list evidence, got %q", handler.Primitive())
	}
	modes := RunPrimitiveHandlerModes(t, handler, schema, []string{"alpha"}, nil)
	if modes.HumanErr != nil || modes.JSONErr != nil {
		t.Fatalf("handler errored: human=%v json=%v", modes.HumanErr, modes.JSONErr)
	}
	if !strings.Contains(modes.Human, "Summary:") || !strings.Contains(modes.Human, "value-alpha") {
		t.Fatalf("human mode wrong: %q", modes.Human)
	}
	if strings.Contains(modes.JSON, "Summary:") {
		t.Fatalf("json mode leaked human report: %q", modes.JSON)
	}
	var decoded any
	if err := json.Unmarshal([]byte(modes.JSON), &decoded); err != nil {
		t.Fatalf("json mode not valid JSON: %v (%q)", err, modes.JSON)
	}
}

// TestPrimitiveEvidenceIsNonForgeable is the runtime witness for plan decision
// D3: observed primitive evidence can only be produced by a cli-core builder.
// cliapptest is an EXTERNAL package (not cliapp), so it stands in for scenario
// code. The commented literals below do not compile — the evidence-bearing fields
// are unexported — which is the actual guarantee; here we assert the positive
// side, that a builder stamps evidence a zero handler lacks.
func TestPrimitiveEvidenceIsNonForgeable(t *testing.T) {
	// A zero handler carries no evidence: an empty struct cannot claim a primitive.
	var zero cliapp.PrimitiveHandler
	if zero.Primitive() != "" {
		t.Fatalf("zero PrimitiveHandler must carry no evidence, got %q", zero.Primitive())
	}

	// The ONLY way to obtain a handler that claims a primitive is a cli-core
	// builder. Scenario code (this package) cannot write
	//   cliapp.PrimitiveHandler{primitive: cliapp.PrimitiveProtoList}   // unexported field: compile error
	//   cmd.primitiveEvidence = cliapp.PrimitiveProtoList               // unexported field: compile error
	built := cliapp.ProtoMutation(
		func(ctx cliapp.OperationContext) (*wrapperspb.StringValue, error) { return wrapperspb.String("x"), nil },
		func(ctx cliapp.OperationContext, resp *wrapperspb.StringValue) cliapp.MutationReport {
			return cliapp.MutationReport{}
		},
	)
	cmd := cliapp.Command{Name: "edit"}.WithPrimitive(built)
	if cmd.PrimitiveEvidence() != cliapp.PrimitiveProtoMutation {
		t.Fatalf("WithPrimitive should stamp proto_mutation evidence, got %q", cmd.PrimitiveEvidence())
	}
}
