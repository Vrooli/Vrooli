package executor

import (
	"testing"
)

func TestEvaluateExpressionBoolLiteral(t *testing.T) {
	if ok, valid := evaluateExpression("true", nil); !valid || !ok {
		t.Fatalf("expected true literal to be valid/true, got ok=%v valid=%v", ok, valid)
	}
	if ok, valid := evaluateExpression("false", nil); !valid || ok {
		t.Fatalf("expected false literal to be valid/false, got ok=%v valid=%v", ok, valid)
	}
}

func TestEvaluateExpressionVariableComparisons(t *testing.T) {
	state := newFlowState(map[string]any{
		"flag":  true,
		"count": 3,
		"name":  "Ada",
		"items": []any{"a", "b"},
	})

	if ok, valid := evaluateExpression("${flag} == true", state); !valid || !ok {
		t.Fatalf("expected flag comparison true, got ok=%v valid=%v", ok, valid)
	}
	if ok, valid := evaluateExpression("${count} > 2", state); !valid || !ok {
		t.Fatalf("expected count > 2 true, got ok=%v valid=%v", ok, valid)
	}
	if ok, valid := evaluateExpression("${count} < 2", state); !valid || ok {
		t.Fatalf("expected count < 2 false, got ok=%v valid=%v", ok, valid)
	}
	if ok, valid := evaluateExpression("${name} == \"Ada\"", state); !valid || !ok {
		t.Fatalf("expected name equality true, got ok=%v valid=%v", ok, valid)
	}
	if ok, valid := evaluateExpression("${items.1} == b", state); !valid || !ok {
		t.Fatalf("expected items.1 == b true, got ok=%v valid=%v", ok, valid)
	}
}

func TestEvaluateExpressionInvalid(t *testing.T) {
	if _, valid := evaluateExpression("", nil); valid {
		t.Fatalf("empty expression should be invalid")
	}
	if _, valid := evaluateExpression("a b", nil); valid {
		t.Fatalf("malformed expression should be invalid")
	}
}
