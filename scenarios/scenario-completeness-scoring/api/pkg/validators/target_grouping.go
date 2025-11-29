package validators

import (
	"fmt"
	"math"
	"regexp"
	"strings"
)

// AnalyzeTargetGrouping analyzes operational target grouping patterns
func AnalyzeTargetGrouping(targets []OperationalTarget, requirements []Requirement) TargetGroupingAnalysis {
	// Build target-to-requirements mapping
	targetMap := make(map[string][]string) // OT-XXX -> [req_id, ...]

	pattern := regexp.MustCompile(`(?i)OT-[Pp][0-2]-\d{3}`)

	for _, req := range requirements {
		// Try prd_ref first
		match := pattern.FindString(req.PRDRef)
		if match == "" {
			// Try operational_target_id
			match = pattern.FindString(req.OperationalTargetID)
		}
		if match != "" {
			targetID := strings.ToUpper(match)
			targetMap[targetID] = append(targetMap[targetID], req.ID)
		}
	}

	// Count 1:1 mappings
	var oneToOneMappings []TargetMapping
	for targetID, reqIDs := range targetMap {
		if len(reqIDs) == 1 {
			oneToOneMappings = append(oneToOneMappings, TargetMapping{
				Target:      targetID,
				Requirement: reqIDs[0],
			})
		}
	}

	totalTargets := len(targetMap)
	oneToOneCount := len(oneToOneMappings)
	oneToOneRatio := 0.0
	if totalTargets > 0 {
		oneToOneRatio = float64(oneToOneCount) / float64(totalTargets)
	}

	// Dynamic threshold: min(20%, 5/total_targets)
	acceptableRatio := 0.2
	if totalTargets > 0 {
		dynamicRatio := 5.0 / float64(totalTargets)
		acceptableRatio = math.Min(0.2, dynamicRatio)
	}

	var violations []TargetGroupingViolation
	if oneToOneRatio > acceptableRatio {
		violations = append(violations, TargetGroupingViolation{
			Type:            "excessive_one_to_one_mappings",
			CurrentRatio:    oneToOneRatio,
			AcceptableRatio: acceptableRatio,
			OneToOneCount:   oneToOneCount,
			TotalTargets:    totalTargets,
			Message: fmt.Sprintf("%d%% of targets have 1:1 requirement mapping (max %d%% recommended)",
				int(oneToOneRatio*100), int(acceptableRatio*100)),
			Targets: oneToOneMappings,
		})
	}

	averageReqsPerTarget := 0.0
	if totalTargets > 0 {
		averageReqsPerTarget = float64(len(requirements)) / float64(totalTargets)
	}

	return TargetGroupingAnalysis{
		TotalTargets:                 totalTargets,
		OneToOneCount:                oneToOneCount,
		OneToOneRatio:                oneToOneRatio,
		AcceptableRatio:              acceptableRatio,
		AverageRequirementsPerTarget: averageReqsPerTarget,
		Violations:                   violations,
	}
}
