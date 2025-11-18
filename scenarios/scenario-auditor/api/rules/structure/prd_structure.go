package structure

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode"

	rules "scenario-auditor/rules"
)

/*
Rule: Scenario PRD Structure
Description: Enforces the standard scenario PRD.md layout so capabilities, metrics, architecture, and lifecycle data stay machine-auditable
Reason: Consistent PRD structure unlocks automated auditing, discovery, and downstream tooling across the scenario catalog
Category: structure
Severity: high
Standard: prd-structure-v1
Targets: documentation
Enabled: false

<test-case id="valid-scenario-prd" should-fail="false" path="scenarios/demo/PRD.md">
  <description>Scenario PRD includes all required top-level sections</description>
  <input language="markdown"><![CDATA[
# Product Requirements Document (PRD)

## 🎯 Capability Definition
Content

## 📊 Success Metrics
Content

## 🏗️ Technical Architecture
Content

## 🖥️ CLI Interface Contract
Content

## 🔄 Integration Requirements
Content

## 🎨 Style and Branding Requirements
Content

## 💰 Value Proposition
Content

## 🧬 Evolution Path
Content

## 🔄 Scenario Lifecycle Integration
Content

## 🚨 Risk Mitigation
Content

## ✅ Validation Criteria
Content

## 📝 Implementation Notes
Content

## 🔗 References
Content
  ]]></input>
</test-case>

<test-case id="missing-section" should-fail="true" path="scenarios/demo/PRD.md">
  <description>Scenario PRD missing Success Metrics section</description>
  <input language="markdown"><![CDATA[
# Product Requirements Document (PRD)

## 🎯 Capability Definition
Content

## 🏗️ Technical Architecture
Content

## 🖥️ CLI Interface Contract
Content

## 🔄 Integration Requirements
Content

## 🎨 Style and Branding Requirements
Content

## 💰 Value Proposition
Content

## 🧬 Evolution Path
Content

## 🔄 Scenario Lifecycle Integration
Content

## 🚨 Risk Mitigation
Content

## ✅ Validation Criteria
Content

## 📝 Implementation Notes
Content

## 🔗 References
Content
  ]]></input>
  <expected-violations>1</expected-violations>
  <expected-message>Success Metrics</expected-message>
</test-case>

<test-case id="non-scenario-prd" should-fail="false" path="resources/example/PRD.md">
  <description>Resource PRDs are ignored by the scenario structure rule</description>
  <input language="markdown"><![CDATA[
# Product Requirements Document (PRD) - Resource

## 🎯 Infrastructure Definition
Content
  ]]></input>
</test-case>

<test-case id="valid-with-subsections" should-fail="false" path="scenarios/demo/PRD.md">
  <description>Scenario PRD contains required headings plus subsections</description>
  <input language="markdown"><![CDATA[
# Product Requirements Document (PRD)

## 🎯 Capability Definition
Content

### Sub Capability
More detail

## 📊 Success Metrics
Content

### Detailed Metrics
Notes

## 🏗️ Technical Architecture
Content

## 🖥️ CLI Interface Contract
Content

## 🔄 Integration Requirements
Content

## 🎨 Style and Branding Requirements
Content

## 💰 Value Proposition
Content

## 🧬 Evolution Path
Content

## 🔄 Scenario Lifecycle Integration
Content

## 🚨 Risk Mitigation
Content

## ✅ Validation Criteria
Content

## 📝 Implementation Notes
Content

## 🔗 References
Content
  ]]></input>
</test-case>

<test-case id="valid-with-extra-section" should-fail="false" path="scenarios/demo/PRD.md">
  <description>Scenario PRD includes extra sections beyond the required set</description>
  <input language="markdown"><![CDATA[
# Product Requirements Document (PRD)

## 🎯 Capability Definition
Content

## 📊 Success Metrics
Content

## 🏗️ Technical Architecture
Content

## 🖥️ CLI Interface Contract
Content

## 🔄 Integration Requirements
Content

## 🎨 Style and Branding Requirements
Content

## 💰 Value Proposition
Content

## 🧬 Evolution Path
Content

## 🔄 Scenario Lifecycle Integration
Content

## 🚨 Risk Mitigation
Content

## ✅ Validation Criteria
Content

## 📝 Implementation Notes
Content

## 🔗 References
Content

## 📎 Appendices
Extra material
  ]]></input>
</test-case>
*/

// CheckScenarioPRDStructure validates that scenario PRDs follow the required heading structure.
func CheckScenarioPRDStructure(content string, filePath string, _ string) ([]rules.Violation, error) {
	path := filepath.ToSlash(filePath)
	if !appliesToScenarioPRD(path) {
		return nil, nil
	}

	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return []rules.Violation{newPRDStructureViolation(path, 1, "PRD.md is empty; populate the document using the scenario PRD template")}, nil
	}

	lines := strings.Split(content, "\n")
	firstLine, firstIndex := firstNonEmptyLine(lines)
	if firstLine == "" {
		return []rules.Violation{newPRDStructureViolation(path, 1, "PRD.md must begin with '# Product Requirements Document' heading")}, nil
	}

	if !strings.HasPrefix(firstLine, "#") {
		return []rules.Violation{newPRDStructureViolation(path, firstIndex, "PRD.md must start with an H1 heading describing the document")}, nil
	}

	if headingLevel(firstLine) != 1 {
		return []rules.Violation{newPRDStructureViolation(path, firstIndex, "PRD.md first heading must be an H1 containing 'Product Requirements Document'")}, nil
	}

	normalizedFirst := normalizeHeading(firstLine)
	if !strings.Contains(normalizedFirst, "product requirements document") {
		return []rules.Violation{newPRDStructureViolation(path, firstIndex, "PRD.md H1 must mention 'Product Requirements Document' or 'PRD'")}, nil
	}

	headingLines := make(map[string]int)
	for i, raw := range lines {
		trim := strings.TrimSpace(raw)
		if !strings.HasPrefix(trim, "#") {
			continue
		}

		normalized := normalizeHeading(trim)
		if normalized == "" {
			continue
		}

		if _, exists := headingLines[normalized]; !exists {
			headingLines[normalized] = i + 1
		}
	}

	useNewStructure := strings.Contains(strings.ToLower(content), "🎯 overview") || strings.Contains(strings.ToLower(content), "🎯 operational targets")
	requiredHeadings := legacyScenarioPRDHeadings
	if useNewStructure {
		requiredHeadings = newScenarioPRDHeadings
	}

	var violations []rules.Violation
	for _, requirement := range requiredHeadings {
		if !headingExists(requirement.keys, headingLines) {
			message := fmt.Sprintf("PRD.md must include the section heading '%s'", requirement.display)
			violations = append(violations, newPRDStructureViolation(path, 1, message))
		}
	}
	return violations, nil
}

func appliesToScenarioPRD(path string) bool {
	if strings.EqualFold(filepath.Base(path), "PRD.md") {
		normalized := strings.ToLower(path)
		if strings.Contains(normalized, "/scenarios/") || strings.HasPrefix(normalized, "scenarios/") || strings.Contains(normalized, "./scenarios/") {
			return true
		}
		// Handle Windows-style paths
		return strings.Contains(normalized, "\\scenarios\\") || strings.HasPrefix(normalized, "scenarios\\")
	}
	return false
}

func firstNonEmptyLine(lines []string) (string, int) {
	for i, line := range lines {
		trim := strings.TrimSpace(line)
		if trim != "" {
			return trim, i + 1
		}
	}
	return "", 0
}

func headingLevel(line string) int {
	count := 0
	for _, r := range line {
		if r == '#' {
			count++
			continue
		}
		break
	}
	return count
}

func normalizeHeading(line string) string {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "#") {
		return ""
	}

	line = strings.TrimLeft(line, "#")
	line = strings.TrimSpace(line)
	line = strings.TrimLeftFunc(line, func(r rune) bool {
		return unicode.IsSpace(r) || (!unicode.IsLetter(r) && !unicode.IsDigit(r))
	})
	line = strings.TrimSpace(line)
	line = strings.ToLower(line)
	return line
}

func headingExists(keys []string, headingLines map[string]int) bool {
	for _, key := range keys {
		if _, ok := headingLines[key]; ok {
			return true
		}
	}
	return false
}

func newPRDStructureViolation(path string, line int, message string) rules.Violation {
	if line <= 0 {
		line = 1
	}
	return rules.Violation{
		Severity:       "high",
		Message:        message,
		FilePath:       filepath.ToSlash(path),
		LineNumber:     line,
		Category:       "structure",
		Standard:       "prd-structure-v1",
		Recommendation: "Update PRD.md to match scripts/scenarios/templates/react-vite/PRD.md",
	}
}

var newScenarioPRDHeadings = []struct {
	display string
	keys    []string
}{
	{display: "## 🎯 Overview", keys: []string{"🎯 overview", "overview"}},
	{display: "## 🎯 Operational Targets", keys: []string{"🎯 operational targets", "operational targets"}},
	{display: "### 🔴 P0 – Must ship for viability", keys: []string{"🔴 p0 – must ship for viability", "p0 – must ship for viability"}},
	{display: "### 🟠 P1 – Should have post-launch", keys: []string{"🟠 p1 – should have post-launch", "p1 – should have post-launch"}},
	{display: "### 🟢 P2 – Future / expansion", keys: []string{"🟢 p2 – future / expansion", "p2 – future / expansion"}},
	{display: "## 🧱 Tech Direction Snapshot", keys: []string{"🧱 tech direction snapshot", "tech direction snapshot"}},
	{display: "## 🤝 Dependencies & Launch Plan", keys: []string{"🤝 dependencies & launch plan", "dependencies & launch plan"}},
	{display: "## 🎨 UX & Branding", keys: []string{"🎨 ux & branding", "ux & branding"}},
}

var legacyScenarioPRDHeadings = []struct {
	display string
	keys    []string
}{
	{display: "## 🎯 Capability Definition", keys: []string{"capability definition"}},
	{display: "## 📊 Success Metrics", keys: []string{"success metrics"}},
	{display: "## 🏗️ Technical Architecture", keys: []string{"technical architecture"}},
	{display: "## 🖥️ CLI Interface Contract", keys: []string{"cli interface contract"}},
	{display: "## 🔄 Integration Requirements", keys: []string{"integration requirements"}},
	{display: "## 🎨 Style and Branding Requirements", keys: []string{"style and branding requirements"}},
	{display: "## 💰 Value Proposition", keys: []string{"value proposition"}},
	{display: "## 🧬 Evolution Path", keys: []string{"evolution path"}},
	{display: "## 🔄 Scenario Lifecycle Integration", keys: []string{"scenario lifecycle integration"}},
	{display: "## 🚨 Risk Mitigation", keys: []string{"risk mitigation"}},
	{display: "## ✅ Validation Criteria", keys: []string{"validation criteria"}},
	{display: "## 📝 Implementation Notes", keys: []string{"implementation notes"}},
	{display: "## 🔗 References", keys: []string{"references"}},
}
