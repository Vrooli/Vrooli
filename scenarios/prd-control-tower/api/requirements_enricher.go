package main

// enrichRequirementsWithTestsAndValidation adds test file references and PRD validation to requirements
func enrichRequirementsWithTestsAndValidation(entityType, entityName string, groups []RequirementGroup) []RequirementGroup {
	// First, flatten all requirements to get a list
	allReqs := flattenRequirements(groups)

	// Get PRD validation issues
	prdIssues := validatePRDReferences(entityType, entityName, allReqs)
	issueMap := make(map[string]*PRDValidationIssue)
	for i := range prdIssues {
		issueMap[prdIssues[i].RequirementID] = &prdIssues[i]
	}

	// Scan for test file references
	testRefs := getAllTestReferencesForEntity(entityType, entityName, allReqs)

	// Now enrich each group recursively
	return enrichGroupsRecursive(groups, issueMap, testRefs)
}

// enrichGroupsRecursive enriches requirement groups recursively with test and validation data
func enrichGroupsRecursive(groups []RequirementGroup, issueMap map[string]*PRDValidationIssue, testRefs map[string][]TestFileReference) []RequirementGroup {
	var enriched []RequirementGroup

	for _, group := range groups {
		enrichedGroup := RequirementGroup{
			ID:           group.ID,
			Name:         group.Name,
			Description:  group.Description,
			FilePath:     group.FilePath,
			Requirements: make([]RequirementRecord, len(group.Requirements)),
			Children:     enrichGroupsRecursive(group.Children, issueMap, testRefs),
		}

		// Enrich each requirement in this group
		for i, req := range group.Requirements {
			enrichedReq := req

			// Add PRD validation issue if exists
			if issue, ok := issueMap[req.ID]; ok {
				enrichedReq.PRDRefIssue = issue
			}

			// Add test file references if exist
			if refs, ok := testRefs[req.ID]; ok {
				enrichedReq.TestFiles = refs
			}

			enrichedGroup.Requirements[i] = enrichedReq
		}

		enriched = append(enriched, enrichedGroup)
	}

	return enriched
}
