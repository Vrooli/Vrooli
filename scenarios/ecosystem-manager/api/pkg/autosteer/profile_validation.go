package autosteer

import (
	"fmt"
	"strings"
)

// ValidateProfile validates a profile's structure and data.
func ValidateProfile(profile *AutoSteerProfile) error {
	if profile == nil {
		return fmt.Errorf("profile is required")
	}
	if profile.Name == "" {
		return fmt.Errorf("profile name is required")
	}
	if len(profile.Phases) == 0 {
		return fmt.Errorf("profile must have at least one phase")
	}

	for i, phase := range profile.Phases {
		normalizedID := strings.TrimSpace(phase.SkillID)
		if normalizedID == "" {
			return fmt.Errorf("phase %d must have a skill_id", i)
		}
		if strings.TrimSpace(phase.SkillName) == "" {
			return fmt.Errorf("phase %d must have a skill_name", i)
		}
		profile.Phases[i].SkillID = normalizedID

		if phase.MaxIterations <= 0 {
			return fmt.Errorf("phase %d must have maxIterations > 0", i)
		}

		for j, condition := range phase.StopConditions {
			if err := ValidateCondition(condition); err != nil {
				return fmt.Errorf("phase %d, condition %d: %w", i, j, err)
			}
		}
	}

	return nil
}

// ValidateCondition validates a stop condition.
func ValidateCondition(condition StopCondition) error {
	switch condition.Type {
	case ConditionTypeSimple:
		if condition.Metric == "" {
			return fmt.Errorf("simple condition must have a metric")
		}
		if !IsAllowedMetric(condition.Metric) {
			return fmt.Errorf("unsupported metric '%s' (allowed: %v)", condition.Metric, AllowedMetrics())
		}
		if condition.CompareOperator == "" {
			return fmt.Errorf("simple condition must have a compare operator")
		}
		if !IsValidConditionOperator(condition.CompareOperator) {
			return fmt.Errorf("invalid compare operator '%s'", condition.CompareOperator)
		}
	case ConditionTypeCompound:
		if condition.Operator == "" {
			return fmt.Errorf("compound condition must have a logical operator")
		}
		if !IsValidLogicalOperator(condition.Operator) {
			return fmt.Errorf("invalid logical operator '%s'", condition.Operator)
		}
		if len(condition.Conditions) == 0 {
			return fmt.Errorf("compound condition must have sub-conditions")
		}
		for i, subCondition := range condition.Conditions {
			if err := ValidateCondition(subCondition); err != nil {
				return fmt.Errorf("sub-condition %d: %w", i, err)
			}
		}
	default:
		return fmt.Errorf("unknown condition type: %s", condition.Type)
	}

	return nil
}
