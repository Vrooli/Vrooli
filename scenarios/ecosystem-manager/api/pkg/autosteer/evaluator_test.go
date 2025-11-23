package autosteer

import (
	"testing"
	"time"
)

func TestConditionEvaluator_EvaluateSimple(t *testing.T) {
	evaluator := NewConditionEvaluator()

	metrics := MetricsSnapshot{
		Timestamp:                    time.Now(),
		Loops:                        5,
		BuildStatus:                  1,
		OperationalTargetsTotal:      10,
		OperationalTargetsPassing:    7,
		OperationalTargetsPercentage: 70.0,
		UX: &UXMetrics{
			AccessibilityScore: 85.0,
			UITestCoverage:     60.0,
		},
		Test: &TestMetrics{
			UnitTestCoverage: 75.0,
			FlakyTests:       2,
		},
	}

	tests := []struct {
		name      string
		condition StopCondition
		want      bool
		wantErr   bool
	}{
		{
			name: "loops greater than - true",
			condition: StopCondition{
				Type:            ConditionTypeSimple,
				Metric:          "loops",
				CompareOperator: OpGreaterThan,
				Value:           3,
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "loops greater than - false",
			condition: StopCondition{
				Type:            ConditionTypeSimple,
				Metric:          "loops",
				CompareOperator: OpGreaterThan,
				Value:           10,
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "operational_targets_percentage >= 70",
			condition: StopCondition{
				Type:            ConditionTypeSimple,
				Metric:          "operational_targets_percentage",
				CompareOperator: OpGreaterThanEquals,
				Value:           70,
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "build_status equals 1 (passing)",
			condition: StopCondition{
				Type:            ConditionTypeSimple,
				Metric:          "build_status",
				CompareOperator: OpEquals,
				Value:           1,
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "accessibility_score > 90 - false",
			condition: StopCondition{
				Type:            ConditionTypeSimple,
				Metric:          "accessibility_score",
				CompareOperator: OpGreaterThan,
				Value:           90,
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "accessibility_score > 80 - true",
			condition: StopCondition{
				Type:            ConditionTypeSimple,
				Metric:          "accessibility_score",
				CompareOperator: OpGreaterThan,
				Value:           80,
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "flaky_tests == 0 - false",
			condition: StopCondition{
				Type:            ConditionTypeSimple,
				Metric:          "flaky_tests",
				CompareOperator: OpEquals,
				Value:           0,
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "unit_test_coverage >= 75 - true",
			condition: StopCondition{
				Type:            ConditionTypeSimple,
				Metric:          "unit_test_coverage",
				CompareOperator: OpGreaterThanEquals,
				Value:           75,
			},
			want:    true,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := evaluator.Evaluate(tt.condition, metrics)
			if (err != nil) != tt.wantErr {
				t.Errorf("Evaluate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Evaluate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConditionEvaluator_EvaluateCompound(t *testing.T) {
	evaluator := NewConditionEvaluator()

	metrics := MetricsSnapshot{
		Timestamp:                    time.Now(),
		Loops:                        5,
		OperationalTargetsPercentage: 80.0,
		UX: &UXMetrics{
			AccessibilityScore: 92.0,
			UITestCoverage:     85.0,
		},
	}

	tests := []struct {
		name      string
		condition StopCondition
		want      bool
		wantErr   bool
	}{
		{
			name: "AND condition - all true",
			condition: StopCondition{
				Type:     ConditionTypeCompound,
				Operator: LogicalAND,
				Conditions: []StopCondition{
					{
						Type:            ConditionTypeSimple,
						Metric:          "loops",
						CompareOperator: OpGreaterThan,
						Value:           3,
					},
					{
						Type:            ConditionTypeSimple,
						Metric:          "accessibility_score",
						CompareOperator: OpGreaterThan,
						Value:           90,
					},
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "AND condition - one false",
			condition: StopCondition{
				Type:     ConditionTypeCompound,
				Operator: LogicalAND,
				Conditions: []StopCondition{
					{
						Type:            ConditionTypeSimple,
						Metric:          "loops",
						CompareOperator: OpGreaterThan,
						Value:           10,
					},
					{
						Type:            ConditionTypeSimple,
						Metric:          "accessibility_score",
						CompareOperator: OpGreaterThan,
						Value:           90,
					},
				},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "OR condition - one true",
			condition: StopCondition{
				Type:     ConditionTypeCompound,
				Operator: LogicalOR,
				Conditions: []StopCondition{
					{
						Type:            ConditionTypeSimple,
						Metric:          "loops",
						CompareOperator: OpGreaterThan,
						Value:           10,
					},
					{
						Type:            ConditionTypeSimple,
						Metric:          "accessibility_score",
						CompareOperator: OpGreaterThan,
						Value:           90,
					},
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "OR condition - all false",
			condition: StopCondition{
				Type:     ConditionTypeCompound,
				Operator: LogicalOR,
				Conditions: []StopCondition{
					{
						Type:            ConditionTypeSimple,
						Metric:          "loops",
						CompareOperator: OpGreaterThan,
						Value:           10,
					},
					{
						Type:            ConditionTypeSimple,
						Metric:          "accessibility_score",
						CompareOperator: OpGreaterThan,
						Value:           95,
					},
				},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "nested compound - (loops > 10) OR (loops > 3 AND accessibility > 90)",
			condition: StopCondition{
				Type:     ConditionTypeCompound,
				Operator: LogicalOR,
				Conditions: []StopCondition{
					{
						Type:            ConditionTypeSimple,
						Metric:          "loops",
						CompareOperator: OpGreaterThan,
						Value:           10,
					},
					{
						Type:     ConditionTypeCompound,
						Operator: LogicalAND,
						Conditions: []StopCondition{
							{
								Type:            ConditionTypeSimple,
								Metric:          "loops",
								CompareOperator: OpGreaterThan,
								Value:           3,
							},
							{
								Type:            ConditionTypeSimple,
								Metric:          "accessibility_score",
								CompareOperator: OpGreaterThan,
								Value:           90,
							},
						},
					},
				},
			},
			want:    true,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := evaluator.Evaluate(tt.condition, metrics)
			if (err != nil) != tt.wantErr {
				t.Errorf("Evaluate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Evaluate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConditionEvaluator_FormatCondition(t *testing.T) {
	evaluator := NewConditionEvaluator()

	metrics := MetricsSnapshot{
		Timestamp:                    time.Now(),
		Loops:                        5,
		OperationalTargetsPercentage: 80.0,
		UX: &UXMetrics{
			AccessibilityScore: 85.0,
		},
	}

	condition := StopCondition{
		Type:            ConditionTypeSimple,
		Metric:          "accessibility_score",
		CompareOperator: OpGreaterThan,
		Value:           90,
	}

	formatted := evaluator.FormatCondition(condition, metrics)
	if formatted == "" {
		t.Error("FormatCondition() returned empty string")
	}

	// Should contain the metric name, operator, and values
	t.Logf("Formatted condition: %s", formatted)
}

func TestConditionEvaluator_Operators(t *testing.T) {
	evaluator := NewConditionEvaluator()

	tests := []struct {
		name     string
		actual   float64
		operator ConditionOperator
		expected float64
		want     bool
	}{
		{"greater than - true", 10, OpGreaterThan, 5, true},
		{"greater than - false", 5, OpGreaterThan, 10, false},
		{"less than - true", 5, OpLessThan, 10, true},
		{"less than - false", 10, OpLessThan, 5, false},
		{"greater than equals - true (greater)", 10, OpGreaterThanEquals, 5, true},
		{"greater than equals - true (equal)", 10, OpGreaterThanEquals, 10, true},
		{"greater than equals - false", 5, OpGreaterThanEquals, 10, false},
		{"less than equals - true (less)", 5, OpLessThanEquals, 10, true},
		{"less than equals - true (equal)", 10, OpLessThanEquals, 10, true},
		{"less than equals - false", 10, OpLessThanEquals, 5, false},
		{"equals - true", 10, OpEquals, 10, true},
		{"equals - false", 10, OpEquals, 5, false},
		{"not equals - true", 10, OpNotEquals, 5, true},
		{"not equals - false", 10, OpNotEquals, 10, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := evaluator.compareValues(tt.actual, tt.operator, tt.expected)
			if err != nil {
				t.Errorf("compareValues() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("compareValues(%v, %s, %v) = %v, want %v", tt.actual, tt.operator, tt.expected, got, tt.want)
			}
		})
	}
}
