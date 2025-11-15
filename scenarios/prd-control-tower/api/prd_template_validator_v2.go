package main

import (
	"fmt"
	"regexp"
	"strings"
)

// PRDTemplateSchemaV2 represents the comprehensive PRD template structure
type PRDTemplateSchemaV2 struct {
	Version  string
	Sections []PRDSectionV2
}

// PRDSectionV2 represents a section in the PRD with hierarchical subsections
type PRDSectionV2 struct {
	Title        string
	Level        int // 2 for ##, 3 for ###
	Required     bool
	Description  string
	Subsections  []PRDSectionV2
	Validations  []PRDContentValidation
	ExpectedKeys []string // For YAML blocks
}

// PRDContentValidation defines content-level validation rules
type PRDContentValidation struct {
	Type        string // "contains", "regex", "yaml_block", "checklist", "table"
	Pattern     string
	Description string
	Required    bool
}

// PRDValidationResultV2 extends the basic result with deeper insights
type PRDValidationResultV2 struct {
	CompliantSections    []string                `json:"compliant_sections"`
	MissingSections      []string                `json:"missing_sections"`
	Violations           []PRDTemplateViolation  `json:"violations"`
	ContentIssues        []PRDContentIssue       `json:"content_issues"`
	StructureScore       float64                 `json:"structure_score"`    // 0-100, section presence
	ContentScore         float64                 `json:"content_score"`      // 0-100, content quality
	OverallScore         float64                 `json:"overall_score"`      // Combined score
	IsFullyCompliant     bool                    `json:"is_fully_compliant"`
	MissingSubsections   map[string][]string     `json:"missing_subsections"` // parent -> missing children
}

// PRDContentIssue represents content-level problems
type PRDContentIssue struct {
	Section     string `json:"section"`
	IssueType   string `json:"issue_type"` // "empty_section", "missing_yaml", "invalid_checklist", "missing_table"
	Message     string `json:"message"`
	Severity    string `json:"severity"`
	LineNumber  int    `json:"line_number,omitempty"`
	Suggestion  string `json:"suggestion,omitempty"`
}

// GetPRDTemplateSchemaV2 returns the comprehensive template definition
func GetPRDTemplateSchemaV2() PRDTemplateSchemaV2 {
	return PRDTemplateSchemaV2{
		Version: "2.0",
		Sections: []PRDSectionV2{
			{
				Title:       "🎯 Capability Definition",
				Level:       2,
				Required:    true,
				Description: "Core capability and intelligence amplification",
				Subsections: []PRDSectionV2{
					{
						Title:       "Core Capability",
						Level:       3,
						Required:    true,
						Description: "What permanent capability this adds to Vrooli",
						Validations: []PRDContentValidation{
							{Type: "contains", Pattern: "What permanent capability", Description: "Must answer core capability question", Required: true},
						},
					},
					{
						Title:       "Intelligence Amplification",
						Level:       3,
						Required:    true,
						Description: "How this makes future agents smarter",
						Validations: []PRDContentValidation{
							{Type: "contains", Pattern: "How does this capability make future agents smarter", Description: "Must answer intelligence question", Required: true},
						},
					},
					{
						Title:       "Recursive Value",
						Level:       3,
						Required:    true,
						Description: "What new scenarios become possible",
						Validations: []PRDContentValidation{
							{Type: "contains", Pattern: "What new scenarios become possible", Description: "Must list future scenarios", Required: true},
						},
					},
				},
			},
			{
				Title:       "📊 Success Metrics",
				Level:       2,
				Required:    true,
				Description: "Functional requirements and quality gates",
				Subsections: []PRDSectionV2{
					{
						Title:       "Functional Requirements",
						Level:       3,
						Required:    true,
						Description: "Must Have (P0), Should Have (P1), Nice to Have (P2)",
						Validations: []PRDContentValidation{
							{Type: "contains", Pattern: "Must Have (P0)", Description: "Must have P0 section", Required: true},
							{Type: "checklist", Pattern: `- \[[ x]\]`, Description: "Must use checkbox format for requirements", Required: true},
						},
					},
					{
						Title:       "Performance Criteria",
						Level:       3,
						Required:    false,
						Description: "Measurable performance targets",
						Validations: []PRDContentValidation{
							{Type: "table", Pattern: `\|.*Metric.*\|.*Target.*\|`, Description: "Should have performance metrics table", Required: false},
						},
					},
					{
						Title:       "Quality Gates",
						Level:       3,
						Required:    true,
						Description: "Validation criteria for completion",
						Validations: []PRDContentValidation{
							{Type: "checklist", Pattern: `- \[[ x]\]`, Description: "Must use checkbox format", Required: true},
						},
					},
				},
			},
			{
				Title:       "🏗️ Technical Architecture",
				Level:       2,
				Required:    true,
				Description: "Resource dependencies and data models",
				Subsections: []PRDSectionV2{
					{
						Title:       "Resource Dependencies",
						Level:       3,
						Required:    true,
						Description: "Required and optional resources",
						Validations: []PRDContentValidation{
							{Type: "yaml_block", Pattern: "```yaml", Description: "Should use YAML format", Required: false},
						},
					},
					{
						Title:       "Resource Integration Standards",
						Level:       3,
						Required:    false,
						Description: "Integration priority order",
					},
					{
						Title:       "Data Models",
						Level:       3,
						Required:    false,
						Description: "Core data structures",
					},
					{
						Title:       "API Contract",
						Level:       3,
						Required:    false,
						Description: "API endpoints and schemas",
					},
					{
						Title:       "Event Interface",
						Level:       3,
						Required:    false,
						Description: "Published and consumed events",
					},
				},
			},
			{
				Title:       "🖥️ CLI Interface Contract",
				Level:       2,
				Required:    false,
				Description: "CLI commands and API parity",
				Subsections: []PRDSectionV2{
					{Title: "Command Structure", Level: 3, Required: false},
					{Title: "CLI-API Parity Requirements", Level: 3, Required: false},
					{Title: "Implementation Standards", Level: 3, Required: false},
				},
			},
			{
				Title:       "🔄 Integration Requirements",
				Level:       2,
				Required:    false,
				Description: "Upstream and downstream dependencies",
				Subsections: []PRDSectionV2{
					{Title: "Upstream Dependencies", Level: 3, Required: false},
					{Title: "Downstream Enablement", Level: 3, Required: false},
					{Title: "Cross-Scenario Interactions", Level: 3, Required: false},
				},
			},
			{
				Title:       "🎨 Style and Branding Requirements",
				Level:       2,
				Required:    false,
				Description: "UI/UX guidelines",
				Subsections: []PRDSectionV2{
					{Title: "UI/UX Style Guidelines", Level: 3, Required: false},
					{Title: "Target Audience Alignment", Level: 3, Required: false},
					{Title: "Brand Consistency Rules", Level: 3, Required: false},
				},
			},
			{
				Title:       "💰 Value Proposition",
				Level:       2,
				Required:    false,
				Description: "Revenue impact and business justification",
				Subsections: []PRDSectionV2{
					{Title: "Business Value", Level: 3, Required: false},
					{Title: "Technical Value", Level: 3, Required: false},
				},
			},
			{
				Title:       "🧬 Evolution Path",
				Level:       2,
				Required:    false,
				Description: "Future phases and roadmap",
			},
			{
				Title:       "🔄 Scenario Lifecycle Integration",
				Level:       2,
				Required:    false,
				Description: "Setup, develop, test, stop steps",
			},
			{
				Title:       "🚨 Risk Mitigation",
				Level:       2,
				Required:    false,
				Description: "Technical and operational risks",
				Subsections: []PRDSectionV2{
					{Title: "Technical Risks", Level: 3, Required: false},
					{Title: "Operational Risks", Level: 3, Required: false},
				},
			},
			{
				Title:       "✅ Validation Criteria",
				Level:       2,
				Required:    false,
				Description: "Capability validation checklist",
			},
			{
				Title:       "📝 Implementation Notes",
				Level:       2,
				Required:    false,
				Description: "Design decisions and known limitations",
			},
			{
				Title:       "🔗 References",
				Level:       2,
				Required:    false,
				Description: "Documentation and external resources",
			},
		},
	}
}

// ValidatePRDTemplateV2 performs comprehensive template validation
func ValidatePRDTemplateV2(content string) PRDValidationResultV2 {
	schema := GetPRDTemplateSchemaV2()
	_ = schema // Suppress unused warning during development

	result := PRDValidationResultV2{
		CompliantSections:  []string{},
		MissingSections:    []string{},
		Violations:         []PRDTemplateViolation{},
		ContentIssues:      []PRDContentIssue{},
		MissingSubsections: make(map[string][]string),
	}

	// Extract sections and content
	foundSections := extractSectionsWithContent(content)

	// Track scoring
	var totalRequired, foundRequired int
	var totalSubsections, foundSubsections int

	// Validate each top-level section
	for _, section := range schema.Sections {
		sectionKey := fmt.Sprintf("%d:%s", section.Level, normalizeTitle(section.Title))
		sectionFound := false

		for foundKey := range foundSections {
			if foundKey == sectionKey {
				sectionFound = true
				break
			}
		}

		if section.Required {
			totalRequired++
			if sectionFound {
				foundRequired++
				result.CompliantSections = append(result.CompliantSections, section.Title)

				// Validate subsections
				if len(section.Subsections) > 0 {
					validateSubsections(section, foundSections, &result, &totalSubsections, &foundSubsections)
				}

				// Validate content
				if sectionContent, ok := foundSections[sectionKey]; ok {
					validateSectionContent(section, sectionContent, &result)
				}
			} else {
				result.MissingSections = append(result.MissingSections, section.Title)
				result.Violations = append(result.Violations, PRDTemplateViolation{
					Section:    section.Title,
					Level:      section.Level,
					Message:    fmt.Sprintf("Required section '%s' is missing", section.Title),
					Severity:   "error",
					Suggestion: fmt.Sprintf("Add section: %s %s\n\n%s", strings.Repeat("#", section.Level), section.Title, section.Description),
				})
			}
		} else {
			// Optional section - just validate if present
			if sectionFound {
				result.CompliantSections = append(result.CompliantSections, section.Title)
				if len(section.Subsections) > 0 {
					validateSubsections(section, foundSections, &result, &totalSubsections, &foundSubsections)
				}
			}
		}
	}

	// Calculate scores
	if totalRequired > 0 {
		result.StructureScore = float64(foundRequired) / float64(totalRequired) * 100
	} else {
		result.StructureScore = 100
	}

	totalContentChecks := float64(len(result.ContentIssues))
	contentPassed := 0.0
	for _, issue := range result.ContentIssues {
		if issue.Severity != "error" {
			contentPassed++
		}
	}
	if totalContentChecks > 0 {
		result.ContentScore = (contentPassed / totalContentChecks) * 100
	} else {
		result.ContentScore = 100
	}

	result.OverallScore = (result.StructureScore*0.6 + result.ContentScore*0.4)
	result.IsFullyCompliant = result.StructureScore == 100 && len(result.Violations) == 0

	return result
}

// extractSectionsWithContent extracts sections and their content
func extractSectionsWithContent(content string) map[string]string {
	sections := make(map[string]string)
	lines := strings.Split(content, "\n")

	sectionPattern := regexp.MustCompile(`^(#{2,3})\s+(.+)$`)

	var currentSection string
	var currentContent strings.Builder

	for _, line := range lines {
		matches := sectionPattern.FindStringSubmatch(line)
		if len(matches) == 3 {
			// Save previous section
			if currentSection != "" {
				sections[currentSection] = currentContent.String()
			}

			// Start new section
			level := len(matches[1])
			title := strings.TrimSpace(matches[2])
			normalizedTitle := normalizeTitle(title)
			currentSection = fmt.Sprintf("%d:%s", level, normalizedTitle)
			currentContent.Reset()
		} else if currentSection != "" {
			currentContent.WriteString(line)
			currentContent.WriteString("\n")
		}
	}

	// Save last section
	if currentSection != "" {
		sections[currentSection] = currentContent.String()
	}

	return sections
}

// validateSubsections checks for presence of required subsections
func validateSubsections(parent PRDSectionV2, foundSections map[string]string, result *PRDValidationResultV2, totalSub, foundSub *int) {
	var missing []string

	for _, subsection := range parent.Subsections {
		subsectionKey := fmt.Sprintf("%d:%s", subsection.Level, normalizeTitle(subsection.Title))

		if subsection.Required {
			*totalSub++
		}

		found := false
		for foundKey := range foundSections {
			if foundKey == subsectionKey {
				found = true
				break
			}
		}

		if subsection.Required && !found {
			missing = append(missing, subsection.Title)
			result.Violations = append(result.Violations, PRDTemplateViolation{
				Section:    fmt.Sprintf("%s > %s", parent.Title, subsection.Title),
				Level:      subsection.Level,
				Message:    fmt.Sprintf("Required subsection '%s' is missing under '%s'", subsection.Title, parent.Title),
				Severity:   "error",
				Suggestion: fmt.Sprintf("Add subsection: %s %s", strings.Repeat("#", subsection.Level), subsection.Title),
			})
		} else if found {
			if subsection.Required {
				*foundSub++
			}
			result.CompliantSections = append(result.CompliantSections, fmt.Sprintf("%s > %s", parent.Title, subsection.Title))
		}
	}

	if len(missing) > 0 {
		result.MissingSubsections[parent.Title] = missing
	}
}

// validateSectionContent validates the content of a section
func validateSectionContent(section PRDSectionV2, content string, result *PRDValidationResultV2) {
	// Check if section is empty (only whitespace)
	trimmedContent := strings.TrimSpace(content)
	if trimmedContent == "" || len(trimmedContent) < 20 {
		result.ContentIssues = append(result.ContentIssues, PRDContentIssue{
			Section:    section.Title,
			IssueType:  "empty_section",
			Message:    fmt.Sprintf("Section '%s' appears to be empty or has minimal content", section.Title),
			Severity:   "warning",
			Suggestion: "Add meaningful content to this section",
		})
		return
	}

	// Run validation rules
	for _, validation := range section.Validations {
		switch validation.Type {
		case "contains":
			if !strings.Contains(content, validation.Pattern) {
				severity := "warning"
				if validation.Required {
					severity = "error"
				}
				result.ContentIssues = append(result.ContentIssues, PRDContentIssue{
					Section:    section.Title,
					IssueType:  "missing_content",
					Message:    validation.Description,
					Severity:   severity,
					Suggestion: fmt.Sprintf("Include: %s", validation.Pattern),
				})
			}

		case "checklist":
			checklistPattern := regexp.MustCompile(validation.Pattern)
			if !checklistPattern.MatchString(content) {
				severity := "warning"
				if validation.Required {
					severity = "error"
				}
				result.ContentIssues = append(result.ContentIssues, PRDContentIssue{
					Section:    section.Title,
					IssueType:  "invalid_checklist",
					Message:    validation.Description,
					Severity:   severity,
					Suggestion: "Use checkbox format: - [ ] item or - [x] completed item",
				})
			}

		case "yaml_block":
			if !strings.Contains(content, validation.Pattern) {
				result.ContentIssues = append(result.ContentIssues, PRDContentIssue{
					Section:    section.Title,
					IssueType:  "missing_yaml",
					Message:    validation.Description,
					Severity:   "info",
					Suggestion: "Consider using YAML code blocks for structured data",
				})
			}

		case "table":
			tablePattern := regexp.MustCompile(validation.Pattern)
			if !tablePattern.MatchString(content) {
				result.ContentIssues = append(result.ContentIssues, PRDContentIssue{
					Section:    section.Title,
					IssueType:  "missing_table",
					Message:    validation.Description,
					Severity:   "info",
					Suggestion: "Consider using markdown tables for structured data",
				})
			}
		}
	}
}
