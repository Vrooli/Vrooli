package validators

// AnalyzeTestRefUsage analyzes test ref usage across requirements
// Detects when test files validate multiple requirements (monolithic tests)
func AnalyzeTestRefUsage(requirements []Requirement) TestRefUsageAnalysis {
	refUsage := make(map[string][]string) // ref -> [req_id, req_id, ...]

	for _, req := range requirements {
		for _, v := range req.Validation {
			ref := v.Ref
			if ref == "" {
				ref = v.WorkflowID
			}
			if ref == "" {
				continue
			}

			refUsage[ref] = append(refUsage[ref], req.ID)
		}
	}

	// Detect violations (1 test file validates ≥4 requirements)
	var violations []MonolithicTestInfo
	for ref, reqIDs := range refUsage {
		if len(reqIDs) >= 4 { // Threshold: 4+ requirements per test file
			severity := "medium"
			if len(reqIDs) >= 6 {
				severity = "high"
			}
			violations = append(violations, MonolithicTestInfo{
				TestRef:  ref,
				SharedBy: reqIDs,
				Count:    len(reqIDs),
				Severity: severity,
			})
		}
	}

	// Sort violations by count descending
	for i := 0; i < len(violations)-1; i++ {
		for j := i + 1; j < len(violations); j++ {
			if violations[j].Count > violations[i].Count {
				violations[i], violations[j] = violations[j], violations[i]
			}
		}
	}

	totalRefs := len(refUsage)
	averageReqsPerRef := 0.0
	if totalRefs > 0 {
		averageReqsPerRef = float64(len(requirements)) / float64(totalRefs)
	}

	duplicateRatio := 0.0
	if len(requirements) > 0 {
		duplicateRatio = float64(len(violations)) / float64(len(requirements))
	}

	return TestRefUsageAnalysis{
		TotalRefs:         totalRefs,
		TotalRequirements: len(requirements),
		AverageReqsPerRef: averageReqsPerRef,
		Violations:        violations,
		DuplicateRatio:    duplicateRatio,
	}
}
