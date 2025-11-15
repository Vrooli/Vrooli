package main

import (
	"fmt"
	"regexp"
	"strings"
)

// PRDTemplateSection represents a required section in the PRD template
type PRDTemplateSection struct {
	Title       string
	Level       int // 2 for ##, 3 for ###
	Required    bool
	Description string
}

// PRDTemplateViolation represents a missing or malformed section
type PRDTemplateViolation struct {
	Section     string `json:"section"`
	Level       int    `json:"level"`
	Message     string `json:"message"`
	Severity    string `json:"severity"` // "error" or "warning"
	Suggestion  string `json:"suggestion,omitempty"`
	LineNumber  int    `json:"line_number,omitempty"`
}

// PRDTemplateValidationResult contains the results of template validation
type PRDTemplateValidationResult struct {
	CompliantSections []string                `json:"compliant_sections"`
	MissingSections   []string                `json:"missing_sections"`
	Violations        []PRDTemplateViolation  `json:"violations"`
	CompliancePercent float64                 `json:"compliance_percent"`
	IsCompliant       bool                    `json:"is_compliant"`
}

// GetPRDTemplateDefinition returns the standard PRD template structure
func GetPRDTemplateDefinition() []PRDTemplateSection {
	return []PRDTemplateSection{
		// Level 2 sections (##)
		{Title: "🎯 Capability Definition", Level: 2, Required: true, Description: "Core capability and intelligence amplification"},
		{Title: "📊 Success Metrics", Level: 2, Required: true, Description: "Functional requirements and quality gates"},
		{Title: "🏗️ Technical Architecture", Level: 2, Required: true, Description: "Resource dependencies and data models"},
		{Title: "🖥️ CLI Interface Contract", Level: 2, Required: false, Description: "CLI commands and API parity"},
		{Title: "🔄 Integration Requirements", Level: 2, Required: false, Description: "Upstream and downstream dependencies"},
		{Title: "🎨 Style and Branding Requirements", Level: 2, Required: false, Description: "UI/UX guidelines"},
		{Title: "💰 Value Proposition", Level: 2, Required: false, Description: "Revenue impact and business justification"},
		{Title: "🧬 Evolution Path", Level: 2, Required: false, Description: "Future phases and roadmap"},
		{Title: "🔄 Scenario Lifecycle Integration", Level: 2, Required: false, Description: "Setup, develop, test, stop steps"},
		{Title: "🚨 Risk Mitigation", Level: 2, Required: false, Description: "Technical and operational risks"},
		{Title: "✅ Validation Criteria", Level: 2, Required: false, Description: "Capability validation checklist"},

		// Level 3 sections (###) - Critical subsections
		{Title: "Core Capability", Level: 3, Required: true, Description: "What permanent capability this adds to Vrooli"},
		{Title: "Intelligence Amplification", Level: 3, Required: true, Description: "How this makes future agents smarter"},
		{Title: "Recursive Value", Level: 3, Required: true, Description: "What new scenarios become possible"},
		{Title: "Functional Requirements", Level: 3, Required: true, Description: "Must Have (P0), Should Have (P1), Nice to Have (P2)"},
		{Title: "Performance Criteria", Level: 3, Required: false, Description: "Measurable performance targets"},
		{Title: "Quality Gates", Level: 3, Required: true, Description: "Validation criteria for completion"},
		{Title: "Resource Dependencies", Level: 3, Required: true, Description: "Required and optional resources"},
	}
}

// ValidatePRDTemplate checks if a PRD markdown content matches the standard template
func ValidatePRDTemplate(content string) PRDTemplateValidationResult {
	template := GetPRDTemplateDefinition()
	lines := strings.Split(content, "\n")

	result := PRDTemplateValidationResult{
		CompliantSections: []string{},
		MissingSections:   []string{},
		Violations:        []PRDTemplateViolation{},
	}

	// Extract all sections from content
	foundSections := make(map[string]bool)
	sectionPattern := regexp.MustCompile(`^(#{2,3})\s+(.+)$`)

	for lineNum, line := range lines {
		matches := sectionPattern.FindStringSubmatch(line)
		if len(matches) == 3 {
			level := len(matches[1])
			title := strings.TrimSpace(matches[2])
			// Normalize title by removing emoji variants and extra whitespace
			normalizedTitle := normalizeTitle(title)
			foundSections[fmt.Sprintf("%d:%s", level, normalizedTitle)] = true

			// Store line number for potential violations
			_ = lineNum // Will use this for line number tracking
		}
	}

	// Check each required section
	totalRequired := 0
	foundRequired := 0

	for _, section := range template {
		sectionKey := fmt.Sprintf("%d:%s", section.Level, normalizeTitle(section.Title))

		if section.Required {
			totalRequired++
			if foundSections[sectionKey] {
				foundRequired++
				result.CompliantSections = append(result.CompliantSections, section.Title)
			} else {
				result.MissingSections = append(result.MissingSections, section.Title)
				result.Violations = append(result.Violations, PRDTemplateViolation{
					Section:    section.Title,
					Level:      section.Level,
					Message:    fmt.Sprintf("Required section '%s' is missing", section.Title),
					Severity:   "error",
					Suggestion: fmt.Sprintf("Add section: %s %s", strings.Repeat("#", section.Level), section.Title),
				})
			}
		} else {
			// Optional section - just check if it exists
			if foundSections[sectionKey] {
				result.CompliantSections = append(result.CompliantSections, section.Title)
			}
		}
	}

	// Calculate compliance percentage
	if totalRequired > 0 {
		result.CompliancePercent = float64(foundRequired) / float64(totalRequired) * 100
	}
	result.IsCompliant = result.CompliancePercent == 100

	return result
}

// normalizeTitle removes emojis and normalizes whitespace for comparison
func normalizeTitle(title string) string {
	// Remove emoji (basic approach - removes most common emoji)
	emojiPattern := regexp.MustCompile(`[\x{1F600}-\x{1F64F}\x{1F300}-\x{1F5FF}\x{1F680}-\x{1F6FF}\x{2600}-\x{26FF}\x{2700}-\x{27BF}\x{FE00}-\x{FE0F}\x{1F900}-\x{1F9FF}\x{1F1E0}-\x{1F1FF}]`)
	normalized := emojiPattern.ReplaceAllString(title, "")

	// Normalize whitespace
	normalized = strings.TrimSpace(normalized)
	normalized = regexp.MustCompile(`\s+`).ReplaceAllString(normalized, " ")

	return normalized
}

// GetTemplateSectionSuggestions returns auto-completion suggestions for missing sections
func GetTemplateSectionSuggestions(content string) []string {
	template := GetPRDTemplateDefinition()
	validation := ValidatePRDTemplate(content)

	suggestions := []string{}

	for _, section := range template {
		if section.Required {
			// Check if this section is in missing sections
			found := false
			for _, missing := range validation.MissingSections {
				if normalizeTitle(missing) == normalizeTitle(section.Title) {
					found = true
					break
				}
			}

			if found {
				// Generate template content for this section
				sectionHeader := strings.Repeat("#", section.Level) + " " + section.Title
				sectionContent := getSectionTemplate(section.Title)
				suggestions = append(suggestions, fmt.Sprintf("%s\n%s", sectionHeader, sectionContent))
			}
		}
	}

	return suggestions
}

// getSectionTemplate returns boilerplate content for common sections
func getSectionTemplate(sectionTitle string) string {
	normalized := normalizeTitle(sectionTitle)

	switch normalized {
	case "Core Capability":
		return `**What permanent capability does this scenario add to Vrooli?**
[Define the fundamental capability this scenario provides that will persist forever in the system.]`

	case "Intelligence Amplification":
		return `**How does this capability make future agents smarter?**
[Describe how this capability compounds with existing capabilities to enable more complex problem-solving.]`

	case "Recursive Value":
		return `**What new scenarios become possible after this exists?**
- [Example future scenario 1]
- [Example future scenario 2]
- [Example future scenario 3]`

	case "Functional Requirements":
		return `- **Must Have (P0)**
  - [ ] [Core requirement that defines minimum viable capability]

- **Should Have (P1)**
  - [ ] [Enhancement that significantly improves capability]

- **Nice to Have (P2)**
  - [ ] [Future enhancement]`

	case "Quality Gates":
		return `- [ ] All P0 requirements implemented and tested
- [ ] Integration tests pass with all required resources
- [ ] Performance targets met under load
- [ ] Documentation complete (README, API docs, CLI help)`

	case "Resource Dependencies":
		return `**Required:**
- resource_name: [e.g., postgres]
  purpose: [Why this resource is essential]

**Optional:**
- resource_name: [e.g., redis]
  purpose: [Enhancement this enables]`

	default:
		return "[Content for " + sectionTitle + "]"
	}
}
