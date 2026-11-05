package workflowruntime

import "testing"

// TestExpressionEvaluatorCachesProgramsPerSource proves the compiled-program
// cache eliminates per-evaluation recompilation without changing results: the
// same source evaluated repeatedly caches one program, distinct sources cache
// distinct programs, and every evaluation returns the value the expression
// defines.
func TestExpressionEvaluatorCachesProgramsPerSource(t *testing.T) {
	evaluator, err := NewExpressionEvaluator()
	if err != nil {
		t.Fatal(err)
	}
	const condition = "iteration < 2"
	cases := []struct {
		iteration int64
		want      bool
	}{{iteration: 0, want: true}, {iteration: 1, want: true}, {iteration: 2, want: false}, {iteration: 5, want: false}}
	for _, c := range cases {
		got, evalErr := evaluator.Evaluate(condition, ExpressionContext{Iteration: c.iteration})
		if evalErr != nil {
			t.Fatalf("evaluate %q at iteration %d: %v", condition, c.iteration, evalErr)
		}
		if got != c.want {
			t.Fatalf("evaluate %q at iteration %d = %v, want %v", condition, c.iteration, got, c.want)
		}
	}
	if len(evaluator.cache) != 1 {
		t.Fatalf("repeated evaluation of one source cached %d programs, want 1", len(evaluator.cache))
	}
	if _, ok := evaluator.cache[condition]; !ok {
		t.Fatalf("cache is not keyed by condition source: %v", evaluator.cache)
	}

	// A distinct source caches a distinct program and still evaluates correctly.
	got, err := evaluator.Evaluate("status == 'running'", ExpressionContext{Status: "running"})
	if err != nil {
		t.Fatal(err)
	}
	if !got {
		t.Fatal("status condition evaluated false for a running status")
	}
	if len(evaluator.cache) != 2 {
		t.Fatalf("distinct sources cached %d programs, want 2", len(evaluator.cache))
	}
}
