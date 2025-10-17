//go:build ruletests
// +build ruletests

package structure

import "testing"

func TestScenarioPRDStructureDocCases(t *testing.T) {
	runDocTests(t, "prd_structure.go", "scenarios/demo/PRD.md", CheckScenarioPRDStructure)
}

func TestScenarioPRDStructureDetectsMissingHeading(t *testing.T) {
	const content = `# Product Requirements Document (PRD)

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
Content`

	violations, err := CheckScenarioPRDStructure(content, "scenarios/demo/PRD.md", "demo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(violations) == 0 {
		t.Fatalf("expected missing Success Metrics section to be reported")
	}
}

func TestScenarioPRDStructureAllowsSubsections(t *testing.T) {
	const content = `# Product Requirements Document (PRD)

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
Content`

	violations, err := CheckScenarioPRDStructure(content, "scenarios/demo/PRD.md", "demo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(violations) != 0 {
		t.Fatalf("expected no violations, got %+v", violations)
	}
}

func TestScenarioPRDStructureAllowsExtraSections(t *testing.T) {
	const content = `# Product Requirements Document (PRD)

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
Extra material`

	violations, err := CheckScenarioPRDStructure(content, "scenarios/demo/PRD.md", "demo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(violations) != 0 {
		t.Fatalf("expected no violations, got %+v", violations)
	}
}
