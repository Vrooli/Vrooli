package autosteer

import "testing"

func TestValidateCondition(t *testing.T) {
	t.Run("invalid simple condition", func(t *testing.T) {
		condition := StopCondition{
			Type:            ConditionTypeSimple,
			Metric:          "unknown_metric",
			CompareOperator: OpEquals,
			Value:           1,
		}

		if err := ValidateCondition(condition); err == nil {
			t.Fatal("expected error for invalid metric")
		}
	})

	t.Run("valid compound condition", func(t *testing.T) {
		condition := StopCondition{
			Type:     ConditionTypeCompound,
			Operator: LogicalAND,
			Conditions: []StopCondition{
				{
					Type:            ConditionTypeSimple,
					Metric:          "loops",
					CompareOperator: OpGreaterThan,
					Value:           1,
				},
				{
					Type:            ConditionTypeSimple,
					Metric:          "build_status",
					CompareOperator: OpEquals,
					Value:           1,
				},
			},
		}

		if err := ValidateCondition(condition); err != nil {
			t.Fatalf("expected valid compound condition, got error: %v", err)
		}
	})
}
