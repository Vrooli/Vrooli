package protodispatch

import (
	"strings"
	"testing"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/vrooli/cli-core/cliapp"
	apiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
)

// Bool flags live in RunContext's bool-flag set, not the string-flag map, so the
// generic scalar fallback must consult BoolFlag. Before the fix --confirm and
// friends silently never reached their proto bool field. These tests pin that
// contract against the retention request's `confirm` field.

func boolFlagSchema() cliapp.ArgSchema {
	return cliapp.ArgSchema{Flags: []cliapp.Flag{
		{Name: "max-age-days"},
		{Name: "confirm", Bool: true},
	}}
}

func TestBoolFlag_Provided_SetsProtoField(t *testing.T) {
	rc := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Schema:    boolFlagSchema(),
		Flags:     map[string]string{"max-age-days": "3"},
		BoolFlags: map[string]bool{"confirm": true},
	})

	req := &apiv1.ExecutionArtifactRetentionRequest{}
	dyn := dynamicpb.NewMessage(req.ProtoReflect().Descriptor())
	if err := hydrateFromContext(rc, req.ProtoReflect().Descriptor(), dyn); err != nil {
		t.Fatalf("hydrate: %v", err)
	}
	out, err := protojson.Marshal(dyn)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := string(out)
	if !strings.Contains(got, `"confirm":true`) {
		t.Errorf("--confirm bool flag did not reach proto confirm field: %s", got)
	}
	if !strings.Contains(got, `"maxAgeDays":3`) {
		t.Errorf("scalar int flag dropped alongside bool: %s", got)
	}
}

func TestBoolFlag_NotProvided_LeavesProtoFieldFalse(t *testing.T) {
	rc := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Schema: boolFlagSchema(),
		Flags:  map[string]string{"max-age-days": "3"},
		// confirm declared but not provided
	})

	req := &apiv1.ExecutionArtifactRetentionRequest{}
	dyn := dynamicpb.NewMessage(req.ProtoReflect().Descriptor())
	if err := hydrateFromContext(rc, req.ProtoReflect().Descriptor(), dyn); err != nil {
		t.Fatalf("hydrate: %v", err)
	}
	out, err := protojson.Marshal(dyn)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	// proto3 omits false bools; confirm must not be serialized as true.
	if strings.Contains(string(out), `"confirm":true`) {
		t.Errorf("unprovided --confirm must leave proto confirm false: %s", string(out))
	}
}
