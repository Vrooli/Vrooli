package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PRDValidationIssue represents a problem with a requirement's prd_ref field
type PRDValidationIssue struct {
	RequirementID string `json:"requirement_id"`
	PRDRef        string `json:"prd_ref"`
	IssueType     string `json:"issue_type"` // "missing_section", "ambiguous_match", "no_checkbox"
	Message       string `json:"message"`
	Suggestions   []string `json:"suggestions,omitempty"`
}

// validatePRDReferences checks if requirement prd_ref fields match actual PRD content
func validatePRDReferences(entityType, entityName string, requirements []RequirementRecord) []PRDValidationIssue {
	var issues []PRDValidationIssue

	// Load PRD content
	vrooliRoot, err := getVrooliRoot()
	if err != nil {
		return issues
	}

	prdPath := filepath.Join(vrooliRoot, entityType+"s", entityName, "PRD.md")
	content, err := os.ReadFile(prdPath)
	if err != nil {
		return issues
	}

	prdText := string(content)
	prdLower := strings.ToLower(prdText)

	// Extract all sections and checkboxes from PRD
	sections := extractPRDSections(prdText)
	checkboxes := extractPRDCheckboxes(prdText)

	for _, req := range requirements {
		if req.PRDRef == "" {
			continue
		}

		// Validate that prd_ref can be found in PRD
		issue := validateSinglePRDRef(req, prdLower, sections, checkboxes)
		if issue != nil {
			issues = append(issues, *issue)
		}
	}

	return issues
}

// validateSinglePRDRef validates one requirement's prd_ref against PRD content
func validateSinglePRDRef(req RequirementRecord, prdLower string, sections []string, checkboxes []string) *PRDValidationIssue {
	ref := req.PRDRef

	// Try to find the exact path in PRD
	// prd_ref format: "Section > Subsection > Item"
	parts := strings.Split(ref, ">")
	if len(parts) == 0 {
		return &PRDValidationIssue{
			RequirementID: req.ID,
			PRDRef:        ref,
			IssueType:     "invalid_format",
			Message:       "prd_ref is empty or invalid format (expected: 'Section > Subsection > Item')",
		}
	}

	// Check if the full path exists in PRD
	lastPart := strings.TrimSpace(parts[len(parts)-1])
	lastPartLower := strings.ToLower(lastPart)

	// Strategy 1: Look for exact section heading match
	foundInSections := false
	for _, section := range sections {
		if strings.Contains(strings.ToLower(section), lastPartLower) {
			foundInSections = true
			break
		}
	}

	// Strategy 2: Look for checkbox item match
	foundInCheckboxes := false
	matchingCheckboxes := []string{}
	for _, checkbox := range checkboxes {
		checkboxLower := strings.ToLower(checkbox)
		if strings.Contains(checkboxLower, lastPartLower) || strings.Contains(lastPartLower, checkboxLower) {
			foundInCheckboxes = true
			matchingCheckboxes = append(matchingCheckboxes, checkbox)
		}
	}

	// Strategy 3: Look for any mention in PRD text
	foundInText := strings.Contains(prdLower, lastPartLower)

	// Determine if this is a problem
	if !foundInSections && !foundInCheckboxes && !foundInText {
		suggestions := findSimilarSections(lastPart, sections, checkboxes)
		return &PRDValidationIssue{
			RequirementID: req.ID,
			PRDRef:        ref,
			IssueType:     "missing_section",
			Message:       fmt.Sprintf("Could not find '%s' in PRD sections or checkboxes", lastPart),
			Suggestions:   suggestions,
		}
	}

	// If found only in text but not in sections/checkboxes, it's ambiguous
	if foundInText && !foundInSections && !foundInCheckboxes {
		return &PRDValidationIssue{
			RequirementID: req.ID,
			PRDRef:        ref,
			IssueType:     "ambiguous_match",
			Message:       fmt.Sprintf("'%s' found in PRD text but not as a formal section or checkbox - consider using exact heading/checkbox text", lastPart),
		}
	}

	// If found in sections but the requirement expects a checkbox (P0/P1/P2 criticality), warn
	if foundInSections && !foundInCheckboxes && (req.Criticality == "P0" || req.Criticality == "P1" || req.Criticality == "P2") {
		return &PRDValidationIssue{
			RequirementID: req.ID,
			PRDRef:        ref,
			IssueType:     "no_checkbox",
			Message:       fmt.Sprintf("'%s' is a section heading but requirement has criticality '%s' - expected a checkbox item in Functional Requirements", lastPart, req.Criticality),
		}
	}

	return nil // Valid reference
}

// extractPRDSections extracts all markdown headings from PRD
func extractPRDSections(content string) []string {
	var sections []string
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			// Remove the # symbols and emoji
			heading := strings.TrimSpace(strings.TrimLeft(trimmed, "#"))
			// Remove common emoji patterns (all emojis used in PRD template)
			heading = strings.TrimSpace(strings.Split(heading, "🎯")[0])
			heading = strings.TrimSpace(strings.Split(heading, "📊")[0])
			heading = strings.TrimSpace(strings.Split(heading, "🏗️")[0])
			heading = strings.TrimSpace(strings.Split(heading, "🖥️")[0])
			heading = strings.TrimSpace(strings.Split(heading, "🔄")[0])
			heading = strings.TrimSpace(strings.Split(heading, "🎨")[0])
			heading = strings.TrimSpace(strings.Split(heading, "💰")[0])
			heading = strings.TrimSpace(strings.Split(heading, "🧬")[0])
			heading = strings.TrimSpace(strings.Split(heading, "🚨")[0])
			heading = strings.TrimSpace(strings.Split(heading, "✅")[0])
			heading = strings.TrimSpace(strings.Split(heading, "📝")[0])
			heading = strings.TrimSpace(strings.Split(heading, "🔗")[0])

			if heading != "" {
				sections = append(sections, heading)
			}
		}
	}

	return sections
}

// extractPRDCheckboxes extracts all checkbox items from PRD
func extractPRDCheckboxes(content string) []string {
	var checkboxes []string
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if checkboxPattern.MatchString(trimmed) {
			// Remove checkbox prefix: - [ ] or - [x]
			text := strings.TrimSpace(trimmed[5:])
			if text != "" {
				checkboxes = append(checkboxes, text)
			}
		}
	}

	return checkboxes
}

// findSimilarSections finds sections/checkboxes that are similar to the target
func findSimilarSections(target string, sections, checkboxes []string) []string {
	var suggestions []string
	targetLower := strings.ToLower(target)
	targetWords := strings.Fields(targetLower)

	// Score each section/checkbox by word overlap
	type scoredItem struct {
		text  string
		score int
	}
	var scored []scoredItem

	for _, section := range sections {
		score := 0
		sectionLower := strings.ToLower(section)
		for _, word := range targetWords {
			if len(word) > 3 && strings.Contains(sectionLower, word) {
				score++
			}
		}
		if score > 0 {
			scored = append(scored, scoredItem{text: section, score: score})
		}
	}

	for _, checkbox := range checkboxes {
		score := 0
		checkboxLower := strings.ToLower(checkbox)
		for _, word := range targetWords {
			if len(word) > 3 && strings.Contains(checkboxLower, word) {
				score++
			}
		}
		if score > 0 {
			scored = append(scored, scoredItem{text: checkbox, score: score})
		}
	}

	// Sort by score descending
	for i := 0; i < len(scored); i++ {
		for j := i + 1; j < len(scored); j++ {
			if scored[j].score > scored[i].score {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}

	// Take top 3
	for i := 0; i < len(scored) && i < 3; i++ {
		suggestions = append(suggestions, scored[i].text)
	}

	return suggestions
}
