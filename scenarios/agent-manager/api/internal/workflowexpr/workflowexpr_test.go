package workflowexpr

import "testing"

func evalBool(t *testing.T, source string, journal []any) bool {
	t.Helper()
	env, err := NewEnv()
	if err != nil {
		t.Fatal(err)
	}
	ast, err := env.Compile(source)
	if err != nil {
		t.Fatalf("compile %q: %v", source, err)
	}
	program, err := env.Program(ast)
	if err != nil {
		t.Fatalf("program %q: %v", source, err)
	}
	out, _, err := program.Eval(map[string]any{"input": map[string]any{}, "journal": journal, "status": "", "iteration": 0, "edge_traversals": map[string]int{}, "budget": map[string]any{}})
	if err != nil {
		t.Fatalf("eval %q: %v", source, err)
	}
	b, ok := out.Value().(bool)
	if !ok {
		t.Fatalf("eval %q returned %T", source, out.Value())
	}
	return b
}

func TestHelpersCompileAndGuardEmptyJournal(t *testing.T) {
	// An empty journal must not panic or error: latest() yields an empty map so
	// has(...) guards read false, and count() is zero.
	if evalBool(t, "has(latest(journal).value)", nil) {
		t.Fatal("latest() over an empty journal should have no value")
	}
	if !evalBool(t, "count(journal, 'slice') == 0", nil) {
		t.Fatal("count() over an empty journal should be zero")
	}
}

func TestHelpersReadEnrichedProjection(t *testing.T) {
	journal := []any{
		map[string]any{"kind": "node_attempt", "nodeId": "slice", "ordinal": 0},
		map[string]any{"kind": "structured_result", "nodeId": "slice", "status": "success", "value": map[string]any{"outcome": "continue"}},
		map[string]any{"kind": "node_attempt", "nodeId": "slice", "ordinal": 1},
	}
	if !evalBool(t, "latest(journal).value.outcome == 'continue'", journal) {
		t.Fatal("latest() did not read the newest successful structured value")
	}
	if !evalBool(t, "count(journal, 'slice') == 2", journal) {
		t.Fatal("count() did not count the two slice attempts")
	}
	if evalBool(t, "count(journal, 'review') == 1", journal) {
		t.Fatal("count() counted a node with no attempts")
	}
}
