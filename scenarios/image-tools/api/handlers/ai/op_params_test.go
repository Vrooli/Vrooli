package ai

import (
	"strings"
	"testing"

	"google.golang.org/protobuf/types/known/structpb"

	aiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/ai"
)

func opParamsStruct(t *testing.T, values map[string]any) *structpb.Struct {
	t.Helper()
	s, err := structpb.NewStruct(values)
	if err != nil {
		t.Fatalf("structpb.NewStruct: %v", err)
	}
	return s
}

// TestOpParamsValidateAndThread proves the manifest-declared parameter contract is
// enforced at the submit edge and threaded to the backend. object_removal declares
// grow_px / halo_threshold, which the builtin mask-fill provider reads — until this
// path existed those knobs were unreachable from every surface.
func TestOpParamsValidateAndThread(t *testing.T) {
	params := &aiv1.AIParams{
		OpParams: opParamsStruct(t, map[string]any{"grow_px": float64(12), "halo_threshold": float64(40)}),
	}
	if err := validateOpParams("object_removal", params); err != nil {
		t.Fatalf("valid op_params rejected: %v", err)
	}
	m := paramsMap(params)
	if got := m["grow_px"]; got != "12" {
		t.Errorf("threaded grow_px = %q, want \"12\"", got)
	}
	if got := m["halo_threshold"]; got != "40" {
		t.Errorf("threaded halo_threshold = %q, want \"40\"", got)
	}
}

func TestOpParamsRejectsUnknownKey(t *testing.T) {
	params := &aiv1.AIParams{OpParams: opParamsStruct(t, map[string]any{"nope": float64(1)})}
	err := validateOpParams("object_removal", params)
	if err == nil || !strings.Contains(err.Error(), "unknown parameter") {
		t.Fatalf("expected unknown-parameter error, got %v", err)
	}
}

func TestOpParamsRejectsNonInteger(t *testing.T) {
	params := &aiv1.AIParams{OpParams: opParamsStruct(t, map[string]any{"grow_px": 1.5})}
	if err := validateOpParams("object_removal", params); err == nil {
		t.Fatal("expected integer constraint violation")
	}
}

func TestOpParamsRejectsOutOfRange(t *testing.T) {
	params := &aiv1.AIParams{OpParams: opParamsStruct(t, map[string]any{"grow_px": float64(65)})}
	if err := validateOpParams("object_removal", params); err == nil {
		t.Fatal("expected maximum constraint violation")
	}
}

// TestOpParamsRejectedWhenNoSchema proves an operation must opt in explicitly: an
// operation with no params_schema rejects any op_params instead of silently
// ignoring them.
func TestOpParamsRejectedWhenNoSchema(t *testing.T) {
	params := &aiv1.AIParams{OpParams: opParamsStruct(t, map[string]any{"grow_px": float64(4)})}
	err := validateOpParams("upscale", params)
	if err == nil || !strings.Contains(err.Error(), "does not accept op_params") {
		t.Fatalf("expected no-schema rejection, got %v", err)
	}
}

func TestOpParamsAbsentIsAlwaysValid(t *testing.T) {
	for _, op := range []string{"object_removal", "upscale", "inpaint"} {
		if err := validateOpParams(op, &aiv1.AIParams{}); err != nil {
			t.Errorf("%s with no op_params: %v", op, err)
		}
	}
}

// TestTypedFieldsWinOverOpParams guards precedence: a shared typed field is
// authoritative, so op_params cannot shadow it even if a schema declared it.
func TestTypedFieldsWinOverOpParams(t *testing.T) {
	params := &aiv1.AIParams{
		Prompt:   "typed prompt",
		OpParams: opParamsStruct(t, map[string]any{"grow_px": float64(3)}),
	}
	m := paramsMap(params)
	if m["prompt"] != "typed prompt" {
		t.Errorf("prompt = %q, want the typed value", m["prompt"])
	}
	if m["grow_px"] != "3" {
		t.Errorf("grow_px = %q, want \"3\"", m["grow_px"])
	}
}
