package runtime

import "testing"

func TestExecutionContextSetAndGet(t *testing.T) {
	ctx := NewExecutionContext()
	if err := ctx.Set("token", "abc123"); err != nil {
		t.Fatalf("unexpected error setting variable: %v", err)
	}
	value, ok := ctx.Get("token")
	if !ok || value != "abc123" {
		t.Fatalf("expected token to be stored, got %v (ok=%v)", value, ok)
	}
}

func TestInterpolateInstructionReplacesVariables(t *testing.T) {
	ctx := NewExecutionContext()
	if err := ctx.Set("username", "Ava"); err != nil {
		t.Fatalf("failed to seed context: %v", err)
	}
	instr := Instruction{
		NodeID: "type-1",
		Type:   "type",
		Params: InstructionParam{Text: "Hello {{username}}"},
	}
	resolved, missing, err := InterpolateInstruction(instr, ctx)
	if err != nil {
		t.Fatalf("unexpected error interpolating instruction: %v", err)
	}
	if len(missing) != 0 {
		t.Fatalf("expected no missing variables, got %v", missing)
	}
	if resolved.Params.Text != "Hello Ava" {
		t.Fatalf("expected interpolation to replace placeholder, got %q", resolved.Params.Text)
	}
	if resolved.Context == nil || resolved.Context["username"] != "Ava" {
		t.Fatalf("expected context snapshot to include username")
	}
}

func TestInterpolateInstructionCollectsMissingVars(t *testing.T) {
	ctx := NewExecutionContext()
	instr := Instruction{
		NodeID: "type-2",
		Type:   "type",
		Params: InstructionParam{Text: "{{missing}}"},
	}
	_, missing, err := InterpolateInstruction(instr, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(missing) != 1 || missing[0] != "missing" {
		t.Fatalf("expected missing variable list, got %v", missing)
	}
}
