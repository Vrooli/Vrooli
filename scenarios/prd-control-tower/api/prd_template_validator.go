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
	Section    string `json:"section"`
	Level      int    `json:"level"`
	Message    string `json:"message"`
	Severity   string `json:"severity"` // "error" or "warning"
	Suggestion string `json:"suggestion,omitempty"`
	LineNumber int    `json:"line_number,omitempty"`
}

// PRDTemplateValidationResult contains the results of template validation
type PRDTemplateValidationResult struct {
	CompliantSections []string               `json:"compliant_sections"`
	MissingSections   []string               `json:"missing_sections"`
	Violations        []PRDTemplateViolation `json:"violations"`
	CompliancePercent float64                `json:"compliance_percent"`
	IsCompliant       bool                   `json:"is_compliant"`
}

// GetPRDTemplateDefinition returns the modern PRD template structure (v2)
func GetPRDTemplateDefinition() []PRDTemplateSection {
	return []PRDTemplateSection{
		{Title: "🎯 Overview", Level: 2, Required: true, Description: "Purpose, users, deployment surfaces"},
		{Title: "🎯 Operational Targets", Level: 2, Required: true, Description: "Outcome checklists"},
		{Title: "🧱 Tech Direction Snapshot", Level: 2, Required: true, Description: "High-level architecture intent"},
		{Title: "🤝 Dependencies & Launch Plan", Level: 2, Required: true, Description: "Resources, dependencies, risks"},
		{Title: "🎨 UX & Branding", Level: 2, Required: true, Description: "Desired look, feel, accessibility"},
		{Title: "📎 Appendix", Level: 2, Required: false, Description: "Optional reference section"},
		{Title: "🔴 P0 – Must ship for viability", Level: 3, Required: true, Description: "Checklist of P0 outcomes"},
		{Title: "🟠 P1 – Should have post-launch", Level: 3, Required: true, Description: "Checklist of P1 outcomes"},
		{Title: "🟢 P2 – Future / expansion", Level: 3, Required: true, Description: "Checklist of P2 outcomes"},
	}
}

// getLegacyPRDTemplateDefinition returns the previous verbose structure to keep older PRDs readable
func getLegacyPRDTemplateDefinition() []PRDTemplateSection {
	return []PRDTemplateSection{
		{Title: "🎯 Capability Definition", Level: 2, Required: true, Description: "Core capability narrative"},
		{Title: "📊 Success Metrics", Level: 2, Required: true, Description: "Functional requirements"},
		{Title: "🏗️ Technical Architecture", Level: 2, Required: true, Description: "Resource dependencies"},
		{Title: "🖥️ CLI Interface Contract", Level: 2, Required: false, Description: "CLI/API mapping"},
		{Title: "🔄 Integration Requirements", Level: 2, Required: false, Description: "Upstream/downstream"},
		{Title: "🎨 Style and Branding Requirements", Level: 2, Required: false, Description: "UI/UX"},
		{Title: "💰 Value Proposition", Level: 2, Required: false, Description: "Business justification"},
		{Title: "🧬 Evolution Path", Level: 2, Required: false, Description: "Future roadmap"},
		{Title: "🔄 Scenario Lifecycle Integration", Level: 2, Required: false, Description: "Lifecycle steps"},
		{Title: "🚨 Risk Mitigation", Level: 2, Required: false, Description: "Risks"},
		{Title: "✅ Validation Criteria", Level: 2, Required: false, Description: "Quality gates"},
		{Title: "Core Capability", Level: 3, Required: true, Description: "Permanent capability"},
		{Title: "Intelligence Amplification", Level: 3, Required: true, Description: "Compound learning"},
		{Title: "Recursive Value", Level: 3, Required: true, Description: "Future scenarios"},
		{Title: "Functional Requirements", Level: 3, Required: true, Description: "P0/P1/P2 checklists"},
		{Title: "Performance Criteria", Level: 3, Required: false, Description: "Performance targets"},
		{Title: "Quality Gates", Level: 3, Required: true, Description: "Validation"},
		{Title: "Resource Dependencies", Level: 3, Required: true, Description: "Required + optional resources"},
	}
}

// ValidatePRDTemplate checks if a PRD markdown content matches the standard template
func ValidatePRDTemplate(content string) PRDTemplateValidationResult {
	modern := validateAgainstDefinition(content, GetPRDTemplateDefinition())
	if modern.IsCompliant || strings.Contains(content, "## 🎯 Overview") || strings.Contains(content, "## 🎯 Operational Targets") {
		return modern
	}

	legacy := validateAgainstDefinition(content, getLegacyPRDTemplateDefinition())
	if legacy.IsCompliant || legacy.CompliancePercent > modern.CompliancePercent {
		return legacy
	}
	return modern
}

func validateAgainstDefinition(content string, template []PRDTemplateSection) PRDTemplateValidationResult {
	lines := strings.Split(content, "\n")
	result := PRDTemplateValidationResult{
		CompliantSections: []string{},
		MissingSections:   []string{},
		Violations:        []PRDTemplateViolation{},
	}

	foundSections := make(map[string]bool)
	sectionPattern := regexp.MustCompile(`^(#{2,3})\s+(.+)$`)

	for _, line := range lines {
		matches := sectionPattern.FindStringSubmatch(line)
		if len(matches) == 3 {
			level := len(matches[1])
			title := strings.TrimSpace(matches[2])
			normalizedTitle := normalizeTitle(title)
			foundSections[fmt.Sprintf("%d:%s", level, normalizedTitle)] = true
		}
	}

	totalRequired := 0
	foundRequired := 0

	for _, section := range template {
		sectionKey := fmt.Sprintf("%d:%s", section.Level, normalizeTitle(section.Title))
		if section.Level == 3 {
			// only check subsection presence if parent exists
			parent := ""
			switch normalizeTitle(section.Title) {
			case "🔴 p0 – must ship for viability", "p0 – must ship for viability", "p0":
				parent = normalizeTitle("🎯 Operational Targets")
			case "🟠 p1 – should have post-launch", "p1 – should have post-launch", "p1":
				parent = normalizeTitle("🎯 Operational Targets")
			case "🟢 p2 – future / expansion", "p2 – future / expansion", "p2":
				parent = normalizeTitle("🎯 Operational Targets")
			}
			if parent != "" {
				parentKey := fmt.Sprintf("%d:%s", 2, parent)
				if !foundSections[parentKey] {
					continue
				}
			}
		}

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
		} else if foundSections[sectionKey] {
			result.CompliantSections = append(result.CompliantSections, section.Title)
		}
	}

	if totalRequired > 0 {
		result.CompliancePercent = float64(foundRequired) / float64(totalRequired) * 100
	}
	result.IsCompliant = result.CompliancePercent == 100
	return result
}

// normalizeTitle removes emojis and normalizes whitespace for comparison
func normalizeTitle(title string) string {
	emojiPattern := regexp.MustCompile(`[\x{1F600}-\x{1F64F}\x{1F300}-\x{1F5FF}\x{1F680}-\x{1F6FF}\x{2600}-\x{26FF}\x{2700}-\x{27BF}\x{FE00}-\x{FE0F}\x{1F900}-\x{1F9FF}\x{1F1E0}-\x{1F1FF}]`)
	normalized := emojiPattern.ReplaceAllString(title, "")
	normalized = strings.TrimSpace(normalized)
	normalized = regexp.MustCompile(`\s+`).ReplaceAllString(normalized, " ")
	return strings.ToLower(normalized)
}

// GetTemplateSectionSuggestions returns auto-completion suggestions for missing sections
func GetTemplateSectionSuggestions(content string) []string {
	template := GetPRDTemplateDefinition()
	validation := ValidatePRDTemplate(content)

	suggestions := []string{}

	for _, section := range template {
		if !section.Required {
			continue
		}
		found := false
		for _, missing := range validation.MissingSections {
			if normalizeTitle(missing) == normalizeTitle(section.Title) {
				found = true
				break
			}
		}

		if found {
			sectionHeader := strings.Repeat("#", section.Level) + " " + section.Title
			sectionContent := getSectionTemplate(section.Title)
			suggestions = append(suggestions, fmt.Sprintf("%s\n%s", sectionHeader, sectionContent))
		}
	}

	return suggestions
}

// getSectionTemplate returns boilerplate content for common sections
func getSectionTemplate(sectionTitle string) string {
	normalized := normalizeTitle(sectionTitle)

	switch normalized {
	case normalizeTitle("🎯 Overview"):
		return `- **Purpose**: [Permanent capability]
- **Primary users / verticals**: [...]
- **Deployment surfaces**: [CLI, API, UI]
- **Value promise**: [Plain-language outcome]`

	case normalizeTitle("🎯 Operational Targets"):
		return `### 🔴 P0 – Must ship for viability
- [ ] OT-P0-001 | Title | One-line description

### 🟠 P1 – Should have post-launch
- [ ] OT-P1-001 | Title | One-line description

### 🟢 P2 – Future / expansion
- [ ] OT-P2-001 | Title | One-line description`

	case normalizeTitle("🧱 Tech Direction Snapshot"):
		return `- Preferred stacks: [React, Go, etc.]
- Data + storage: [postgres, redis]
- Integration strategy: [shared workflows > resource CLI > direct API]
- Non-goals: [Out-of-scope statements]`

	case normalizeTitle("🤝 Dependencies & Launch Plan"):
		return `- Required resources: [...]
- Scenario dependencies: [...]
- Operational risks: [...]
- Launch sequencing: [...]`

	case normalizeTitle("🎨 UX & Branding"):
		return `- Look & feel: [...]
- Accessibility: [...]
- Voice & tone: [...]
- Branding hooks: [...]`

	case normalizeTitle("🔴 P0 – Must ship for viability"):
		return `- [ ] OT-P0-001 | Title | Description`
	case normalizeTitle("🟠 P1 – Should have post-launch"):
		return `- [ ] OT-P1-001 | Title | Description`
	case normalizeTitle("🟢 P2 – Future / expansion"):
		return `- [ ] OT-P2-001 | Title | Description`
	}

	// Legacy fallbacks
	switch normalized {
	case normalizeTitle("Core Capability"):
		return `**What permanent capability does this scenario add to Vrooli?**`
	case normalizeTitle("Intelligence Amplification"):
		return `**How does this capability make future agents smarter?**`
	case normalizeTitle("Recursive Value"):
		return `**What new scenarios become possible after this exists?**`
	case normalizeTitle("Functional Requirements"):
		return `- **Must Have (P0)**\n  - [ ] Requirement`
	case normalizeTitle("Quality Gates"):
		return `- [ ] All P0 requirements implemented`
	case normalizeTitle("Resource Dependencies"):
		return `**Required:**\n- resource_name: postgres`
	}

	return "[Content for " + sectionTitle + "]"
}
