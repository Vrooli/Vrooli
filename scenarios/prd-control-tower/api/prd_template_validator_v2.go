package main

import (
	"fmt"
	"regexp"
	"sort"
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
	ExpectedKeys []string // reserved for future structured validation
}

// PRDContentValidation defines content-level validation rules
type PRDContentValidation struct {
	Type        string // "contains", "regex", "checklist"
	Pattern     string
	Description string
	Required    bool
}

// PRDValidationResultV2 extends the basic result with deeper insights
type PRDValidationResultV2 struct {
	CompliantSections  []string               `json:"compliant_sections"`
	MissingSections    []string               `json:"missing_sections"`
	Violations         []PRDTemplateViolation `json:"violations"`
	ContentIssues      []PRDContentIssue      `json:"content_issues"`
	StructureScore     float64                `json:"structure_score"`
	ContentScore       float64                `json:"content_score"`
	OverallScore       float64                `json:"overall_score"`
	IsFullyCompliant   bool                   `json:"is_fully_compliant"`
	MissingSubsections map[string][]string    `json:"missing_subsections"`
	RequiredSections   int                    `json:"required_sections"`
	CompletedSections  int                    `json:"completed_sections"`
	UnexpectedSections []string               `json:"unexpected_sections"`
}

// PRDContentIssue represents content-level problems
type PRDContentIssue struct {
	Section    string `json:"section"`
	IssueType  string `json:"issue_type"`
	Message    string `json:"message"`
	Severity   string `json:"severity"`
	LineNumber int    `json:"line_number,omitempty"`
	Suggestion string `json:"suggestion,omitempty"`
}

// GetPRDTemplateSchemaV2 returns the comprehensive template definition
func GetPRDTemplateSchemaV2() PRDTemplateSchemaV2 {
	return PRDTemplateSchemaV2{
		Version: "2.0",
		Sections: []PRDSectionV2{
			{
				Title:       "🎯 Overview",
				Level:       2,
				Required:    true,
				Description: "Purpose, users, deployment surfaces",
				Validations: []PRDContentValidation{
					{Type: "contains", Pattern: "Purpose", Description: "Outline the permanent capability", Required: true},
				},
			},
			{
				Title:       "🎯 Operational Targets",
				Level:       2,
				Required:    true,
				Description: "Outcome checklists grouped into tiers",
				Subsections: []PRDSectionV2{
					{
						Title:       "🔴 P0 – Must ship for viability",
						Level:       3,
						Required:    true,
						Description: "Non-negotiable launch targets",
						Validations: []PRDContentValidation{
							{Type: "checklist", Pattern: `- \[[ x]\]`, Description: "Use checklist format for each target", Required: true},
						},
					},
					{
						Title:       "🟠 P1 – Should have post-launch",
						Level:       3,
						Required:    true,
						Description: "Important enhancements",
						Validations: []PRDContentValidation{
							{Type: "checklist", Pattern: `- \[[ x]\]`, Description: "Use checklist format", Required: false},
						},
					},
					{
						Title:       "🟢 P2 – Future / expansion",
						Level:       3,
						Required:    true,
						Description: "Aspirational follow-ups",
						Validations: []PRDContentValidation{
							{Type: "checklist", Pattern: `- \[[ x]\]`, Description: "Use checklist format", Required: false},
						},
					},
				},
			},
			{
				Title:       "🧱 Tech Direction Snapshot",
				Level:       2,
				Required:    true,
				Description: "Preferred stacks, data, integrations, non-goals",
				Validations: []PRDContentValidation{
					{Type: "contains", Pattern: "Preferred", Description: "List preferred stacks or approaches", Required: true},
				},
			},
			{
				Title:       "🤝 Dependencies & Launch Plan",
				Level:       2,
				Required:    true,
				Description: "Resources, scenario dependencies, risks, sequencing",
			},
			{
				Title:       "🎨 UX & Branding",
				Level:       2,
				Required:    true,
				Description: "Look/feel, accessibility, voice",
				Validations: []PRDContentValidation{
					{Type: "contains", Pattern: "Accessibility", Description: "State accessibility expectations", Required: false},
				},
			},
			{
				Title:       "📎 Appendix",
				Level:       2,
				Required:    false,
				Description: "Optional references",
			},
		},
	}
}

// ValidatePRDTemplateV2 performs comprehensive template validation
func ValidatePRDTemplateV2(content string) PRDValidationResultV2 {
	schema := GetPRDTemplateSchemaV2()
	result := PRDValidationResultV2{
		CompliantSections:  []string{},
		MissingSections:    []string{},
		Violations:         []PRDTemplateViolation{},
		ContentIssues:      []PRDContentIssue{},
		MissingSubsections: make(map[string][]string),
		UnexpectedSections: []string{},
	}

	foundSections := extractSectionsWithContent(content)
	var totalRequired, foundRequired int
	var totalSubsections, foundSubsections int
	validSectionKeys := buildValidSectionMap(schema)

	for _, section := range schema.Sections {
		sectionKey := fmt.Sprintf("%d:%s", section.Level, normalizeTitle(section.Title))
		sectionContent, sectionFound := foundSections[sectionKey]

		requiredSubsections := countRequiredSubsections(section)
		totalSubsections += requiredSubsections

		if section.Required {
			totalRequired++
			if sectionFound {
				foundRequired++
				result.CompliantSections = append(result.CompliantSections, section.Title)
				if len(section.Subsections) > 0 {
					validateSubsections(section, foundSections, &result, &foundSubsections)
				}
				validateSectionContent(section, sectionContent.Content, &result)
			} else {
				result.MissingSections = append(result.MissingSections, section.Title)
				result.Violations = append(result.Violations, PRDTemplateViolation{
					Section:    section.Title,
					Level:      section.Level,
					Message:    fmt.Sprintf("Required section '%s' is missing", section.Title),
					Severity:   "error",
					Suggestion: fmt.Sprintf("Add section: %s %s", strings.Repeat("#", section.Level), section.Title),
				})
				if requiredSubsections > 0 {
					result.MissingSubsections[section.Title] = collectRequiredSubsectionTitles(section)
				}
			}
		} else if sectionFound {
			result.CompliantSections = append(result.CompliantSections, section.Title)
			if len(section.Subsections) > 0 {
				validateSubsections(section, foundSections, &result, &foundSubsections)
			}
			validateSectionContent(section, sectionContent.Content, &result)
		} else if requiredSubsections > 0 {
			// Optional parent section missing but with required subsections
			result.MissingSubsections[section.Title] = collectRequiredSubsectionTitles(section)
		}
	}

	result.RequiredSections = totalRequired + totalSubsections
	result.CompletedSections = foundRequired + foundSubsections

	if result.RequiredSections > 0 {
		result.StructureScore = float64(result.CompletedSections) / float64(result.RequiredSections) * 100
	} else {
		result.StructureScore = 100
	}

	if issues := len(result.ContentIssues); issues == 0 {
		result.ContentScore = 100
	} else {
		passed := 0
		for _, issue := range result.ContentIssues {
			if issue.Severity != "error" {
				passed++
			}
		}
		result.ContentScore = float64(passed) / float64(issues) * 100
	}

	result.OverallScore = (result.StructureScore*0.6 + result.ContentScore*0.4)
	result.IsFullyCompliant = result.CompletedSections == result.RequiredSections && len(result.Violations) == 0
	result.UnexpectedSections = collectUnexpectedSections(foundSections, validSectionKeys)
	return result
}

type extractedSection struct {
	Title   string
	Level   int
	Content string
}

func extractSectionsWithContent(content string) map[string]extractedSection {
	sections := make(map[string]extractedSection)
	lines := strings.Split(content, "\n")
	sectionPattern := regexp.MustCompile(`^(#{2,3})\s+(.+)$`)

	var currentKey string
	var currentTitle string
	var currentLevel int
	var currentContent strings.Builder

	storeCurrent := func() {
		if currentKey == "" {
			return
		}
		sections[currentKey] = extractedSection{
			Title:   currentTitle,
			Level:   currentLevel,
			Content: strings.TrimRight(currentContent.String(), "\n"),
		}
		currentContent.Reset()
	}

	for _, line := range lines {
		matches := sectionPattern.FindStringSubmatch(line)
		if len(matches) == 3 {
			storeCurrent()
			currentLevel = len(matches[1])
			currentTitle = strings.TrimSpace(matches[2])
			currentKey = fmt.Sprintf("%d:%s", currentLevel, normalizeTitle(currentTitle))
			continue
		}
		if currentKey != "" {
			currentContent.WriteString(line)
			currentContent.WriteString("\n")
		}
	}

	storeCurrent()
	return sections
}

func validateSubsections(parent PRDSectionV2, foundSections map[string]extractedSection, result *PRDValidationResultV2, foundSub *int) {
	var missing []string
	for _, subsection := range parent.Subsections {
		subsectionKey := fmt.Sprintf("%d:%s", subsection.Level, normalizeTitle(subsection.Title))
		sectionContent, found := foundSections[subsectionKey]

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
			validateSectionContent(subsection, sectionContent.Content, result)
		}
	}

	if len(missing) > 0 {
		result.MissingSubsections[parent.Title] = missing
	}
}

func validateSectionContent(section PRDSectionV2, content string, result *PRDValidationResultV2) {
	trimmedContent := strings.TrimSpace(content)
	if trimmedContent == "" {
		// Parent sections that primarily exist to hold structured subsections (e.g. 🎯 Operational Targets)
		// may legitimately have no direct body content because the parser attributes text to the ### subsections.
		// Treat these as non-empty when subsections are present/validated elsewhere.
		if len(section.Subsections) > 0 {
			return
		}
		result.ContentIssues = append(result.ContentIssues, PRDContentIssue{
			Section:    section.Title,
			IssueType:  "empty_section",
			Message:    fmt.Sprintf("Section '%s' appears empty", section.Title),
			Severity:   "warning",
			Suggestion: "Add descriptive content",
		})
		return
	}

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
					Suggestion: "Use checklist format: - [ ] item",
				})
			}
		case "regex":
			regex := regexp.MustCompile(validation.Pattern)
			if !regex.MatchString(content) {
				severity := "warning"
				if validation.Required {
					severity = "error"
				}
				result.ContentIssues = append(result.ContentIssues, PRDContentIssue{
					Section:   section.Title,
					IssueType: "missing_content",
					Message:   validation.Description,
					Severity:  severity,
				})
			}
		}
	}
}

func countRequiredSubsections(section PRDSectionV2) int {
	count := 0
	for _, subsection := range section.Subsections {
		if subsection.Required {
			count++
		}
	}
	return count
}

func collectRequiredSubsectionTitles(section PRDSectionV2) []string {
	var titles []string
	for _, subsection := range section.Subsections {
		if subsection.Required {
			titles = append(titles, subsection.Title)
		}
	}
	return titles
}

func buildValidSectionMap(schema PRDTemplateSchemaV2) map[string]struct{} {
	valid := make(map[string]struct{})
	for _, section := range schema.Sections {
		key := fmt.Sprintf("%d:%s", section.Level, normalizeTitle(section.Title))
		valid[key] = struct{}{}
		for _, subsection := range section.Subsections {
			subKey := fmt.Sprintf("%d:%s", subsection.Level, normalizeTitle(subsection.Title))
			valid[subKey] = struct{}{}
		}
	}
	return valid
}

func collectUnexpectedSections(found map[string]extractedSection, valid map[string]struct{}) []string {
	var unexpected []string
	for key, section := range found {
		if _, ok := valid[key]; ok {
			continue
		}
		unexpected = append(unexpected, section.Title)
	}
	sort.Strings(unexpected)
	return unexpected
}
