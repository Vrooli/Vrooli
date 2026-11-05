package planning

import planningv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tech-tree-designer/v1/planning"

func errorFinding(code, location, message, suggestion string) PlanFinding {
	return PlanFinding{
		Severity:   planningv1.PlanFindingSeverity_PLAN_FINDING_SEVERITY_ERROR,
		Code:       code,
		Location:   location,
		Message:    message,
		Suggestion: suggestion,
	}
}

func warningFinding(code, location, message, suggestion string) PlanFinding {
	return PlanFinding{
		Severity:   planningv1.PlanFindingSeverity_PLAN_FINDING_SEVERITY_WARNING,
		Code:       code,
		Location:   location,
		Message:    message,
		Suggestion: suggestion,
	}
}
