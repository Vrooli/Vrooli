package autosteer

import (
	"fmt"
	"reflect"
	"strings"
)

// ConditionEvaluator evaluates stop conditions against metrics
type ConditionEvaluator struct{}

// NewConditionEvaluator creates a new condition evaluator
func NewConditionEvaluator() *ConditionEvaluator {
	return &ConditionEvaluator{}
}

// Evaluate evaluates a stop condition against current metrics
func (e *ConditionEvaluator) Evaluate(condition StopCondition, metrics MetricsSnapshot) (bool, error) {
	switch condition.Type {
	case ConditionTypeSimple:
		return e.evaluateSimple(condition, metrics)
	case ConditionTypeCompound:
		return e.evaluateCompound(condition, metrics)
	default:
		return false, fmt.Errorf("unknown condition type: %s", condition.Type)
	}
}

// evaluateSimple evaluates a simple condition
func (e *ConditionEvaluator) evaluateSimple(condition StopCondition, metrics MetricsSnapshot) (bool, error) {
	// Get metric value from snapshot
	value, err := e.GetMetricValue(condition.Metric, metrics)
	if err != nil {
		return false, err
	}

	// Compare using operator
	return e.compareValues(value, condition.CompareOperator, condition.Value)
}

// evaluateCompound evaluates a compound condition (recursive)
func (e *ConditionEvaluator) evaluateCompound(condition StopCondition, metrics MetricsSnapshot) (bool, error) {
	if len(condition.Conditions) == 0 {
		return false, fmt.Errorf("compound condition has no sub-conditions")
	}

	results := make([]bool, len(condition.Conditions))
	for i, subCondition := range condition.Conditions {
		result, err := e.Evaluate(subCondition, metrics)
		if err != nil {
			return false, fmt.Errorf("error evaluating sub-condition %d: %w", i, err)
		}
		results[i] = result
	}

	// Apply logical operator
	switch condition.Operator {
	case LogicalAND:
		for _, result := range results {
			if !result {
				return false, nil
			}
		}
		return true, nil

	case LogicalOR:
		for _, result := range results {
			if result {
				return true, nil
			}
		}
		return false, nil

	default:
		return false, fmt.Errorf("unknown logical operator: %s", condition.Operator)
	}
}

// GetMetricValue extracts a metric value from the metrics snapshot
func (e *ConditionEvaluator) GetMetricValue(metricName string, metrics MetricsSnapshot) (float64, error) {
	// Try direct fields first
	switch metricName {
	case "loops":
		return float64(metrics.Loops), nil
	case "build_status":
		return float64(metrics.BuildStatus), nil
	case "operational_targets_total":
		return float64(metrics.OperationalTargetsTotal), nil
	case "operational_targets_passing":
		return float64(metrics.OperationalTargetsPassing), nil
	case "operational_targets_percentage":
		return metrics.OperationalTargetsPercentage, nil
	}

	// Check UX metrics
	if metrics.UX != nil {
		if value, ok := e.extractFromStruct(metricName, *metrics.UX); ok {
			return value, nil
		}
	}

	// Check Refactor metrics
	if metrics.Refactor != nil {
		if value, ok := e.extractFromStruct(metricName, *metrics.Refactor); ok {
			return value, nil
		}
	}

	// Check Test metrics
	if metrics.Test != nil {
		if value, ok := e.extractFromStruct(metricName, *metrics.Test); ok {
			return value, nil
		}
	}

	// Check Performance metrics
	if metrics.Performance != nil {
		if value, ok := e.extractFromStruct(metricName, *metrics.Performance); ok {
			return value, nil
		}
	}

	// Check Security metrics
	if metrics.Security != nil {
		if value, ok := e.extractFromStruct(metricName, *metrics.Security); ok {
			return value, nil
		}
	}

	return 0, fmt.Errorf("metric not found: %s", metricName)
}

// extractFromStruct attempts to extract a metric from a struct using reflection
func (e *ConditionEvaluator) extractFromStruct(metricName string, structData interface{}) (float64, bool) {
	val := reflect.ValueOf(structData)
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// Get JSON tag name
		jsonTag := fieldType.Tag.Get("json")
		if jsonTag == "" {
			continue
		}

		// Extract tag name (remove options like ",omitempty")
		tagName := jsonTag
		if commaIdx := strings.Index(jsonTag, ","); commaIdx != -1 {
			tagName = jsonTag[:commaIdx]
		}

		if tagName == metricName {
			// Convert field to float64
			switch field.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				return float64(field.Int()), true
			case reflect.Float32, reflect.Float64:
				return field.Float(), true
			default:
				return 0, false
			}
		}
	}

	return 0, false
}

// compareValues compares two values using the specified operator
func (e *ConditionEvaluator) compareValues(actual float64, op ConditionOperator, expected float64) (bool, error) {
	switch op {
	case OpGreaterThan:
		return actual > expected, nil
	case OpLessThan:
		return actual < expected, nil
	case OpGreaterThanEquals:
		return actual >= expected, nil
	case OpLessThanEquals:
		return actual <= expected, nil
	case OpEquals:
		return actual == expected, nil
	case OpNotEquals:
		return actual != expected, nil
	default:
		return false, fmt.Errorf("unknown comparison operator: %s", op)
	}
}

// FormatCondition formats a condition as a human-readable string with current values
func (e *ConditionEvaluator) FormatCondition(condition StopCondition, metrics MetricsSnapshot) string {
	if condition.Type == ConditionTypeSimple {
		value, err := e.GetMetricValue(condition.Metric, metrics)
		if err != nil {
			return fmt.Sprintf("%s %s %.2f (error: %v)", condition.Metric, condition.CompareOperator, condition.Value, err)
		}
		result, _ := e.compareValues(value, condition.CompareOperator, condition.Value)
		status := "✗"
		if result {
			status = "✓"
		}
		return fmt.Sprintf("%s %s %s %.2f (current: %.2f)", status, condition.Metric, condition.CompareOperator, condition.Value, value)
	}

	// Compound condition
	parts := make([]string, len(condition.Conditions))
	for i, subCondition := range condition.Conditions {
		parts[i] = e.FormatCondition(subCondition, metrics)
	}

	operator := " AND "
	if condition.Operator == LogicalOR {
		operator = " OR "
	}

	result := "("
	for i, part := range parts {
		if i > 0 {
			result += operator
		}
		result += part
	}
	result += ")"

	return result
}
