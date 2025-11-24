package autosteer

import (
	"strings"
	"testing"
)

func TestModeInstructions_FormatConditionProgress(t *testing.T) {
	instructions := NewModeInstructions(testPhasePromptsDir(t))
	evaluator := NewConditionEvaluator()

	conditions := []StopCondition{
		{
			Type:            ConditionTypeSimple,
			Metric:          "loops",
			CompareOperator: OpGreaterThanEquals,
			Value:           3,
		},
		{
			Type:            ConditionTypeSimple,
			Metric:          "operational_targets_percentage",
			CompareOperator: OpGreaterThanEquals,
			Value:           80,
		},
	}

	output := instructions.FormatConditionProgress(conditions, MetricsSnapshot{
		Loops:                        2,
		OperationalTargetsPercentage: 50,
	}, evaluator)

	if output == "" {
		t.Fatalf("expected formatted output for conditions")
	}
	if !strings.Contains(output, "loops") || !strings.Contains(output, "operational_targets_percentage") {
		t.Fatalf("formatted output missing metrics: %s", output)
	}
	if !strings.Contains(output, "✗") {
		t.Fatalf("expected unmet condition marker in output: %s", output)
	}
	if !strings.HasPrefix(strings.TrimSpace(output), "- ") {
		t.Fatalf("expected bullet formatting, got: %q", output)
	}
}
