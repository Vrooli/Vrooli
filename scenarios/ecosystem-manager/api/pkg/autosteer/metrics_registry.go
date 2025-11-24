package autosteer

// Central registry of metrics and operators that stop conditions may reference.

// allowedMetrics lists every metric name that ConditionEvaluator can resolve.
// Keep in sync with MetricsSnapshot fields and collector JSON tags.
var allowedMetrics = map[string]struct{}{
	// Loop counters
	"loops":       {},
	"phase_loops": {},
	"total_loops": {},

	// Universal health
	"build_status":                   {},
	"operational_targets_total":      {},
	"operational_targets_passing":    {},
	"operational_targets_percentage": {},

	// UX
	"accessibility_score":     {},
	"ui_test_coverage":        {},
	"responsive_breakpoints":  {},
	"user_flows_implemented":  {},
	"loading_states_count":    {},
	"error_handling_coverage": {},

	// Refactor / code quality
	"cyclomatic_complexity_avg": {},
	"duplication_percentage":    {},
	"standards_violations":      {},
	"tidiness_score":            {},
	"tech_debt_items":           {},

	// Testing
	"unit_test_coverage":        {},
	"integration_test_coverage": {},
	"edge_cases_covered":        {},
	"flaky_tests":               {},
	"test_quality_score":        {},

	// Performance
	"bundle_size_kb":       {},
	"initial_load_time_ms": {},
	"lcp_ms":               {},
	"fid_ms":               {},
	"cls_score":            {},

	// Security
	"vulnerability_count":       {},
	"input_validation_coverage": {},
	"auth_implementation_score": {},
	"security_scan_score":       {},
}

// AllowedMetrics returns a flat slice useful for error messages.
func AllowedMetrics() []string {
	out := make([]string, 0, len(allowedMetrics))
	for m := range allowedMetrics {
		out = append(out, m)
	}
	return out
}

// IsAllowedMetric reports whether a metric name is supported by the evaluator.
func IsAllowedMetric(metric string) bool {
	_, ok := allowedMetrics[metric]
	return ok
}

// IsValidConditionOperator ensures comparison operators are known.
func IsValidConditionOperator(op ConditionOperator) bool {
	switch op {
	case OpGreaterThan, OpLessThan, OpGreaterThanEquals, OpLessThanEquals, OpEquals, OpNotEquals:
		return true
	default:
		return false
	}
}

// IsValidLogicalOperator ensures logical operators are known.
func IsValidLogicalOperator(op LogicalOperator) bool {
	switch op {
	case LogicalAND, LogicalOR:
		return true
	default:
		return false
	}
}

// MetricUnavailableError is returned when a stop condition references a metric
// that is unsupported or not yet collected. The execution engine treats this
// as a non-fatal warning so Auto Steer can continue.
type MetricUnavailableError struct {
	Metric string
	Reason string
}

func (e *MetricUnavailableError) Error() string {
	if e == nil {
		return ""
	}
	if e.Reason != "" {
		return e.Reason
	}
	return "metric unavailable"
}
