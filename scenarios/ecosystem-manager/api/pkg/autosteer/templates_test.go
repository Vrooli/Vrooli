package autosteer

import (
	"fmt"
	"testing"
)

func TestTemplates_AllTemplatesValid(t *testing.T) {
	templates := GetBuiltInTemplates()

	if len(templates) == 0 {
		t.Fatal("Expected at least one built-in template")
	}

	// Track template names to ensure uniqueness
	templateNames := make(map[string]bool)

	for _, template := range templates {
		t.Run(template.Name, func(t *testing.T) {
			// Validate basic fields
			if template.Name == "" {
				t.Error("Template name is empty")
			}
			if templateNames[template.Name] {
				t.Errorf("Duplicate template name: %s", template.Name)
			}
			templateNames[template.Name] = true

			if template.Description == "" {
				t.Error("Template description is empty")
			}

			// Validate phases
			if len(template.Phases) == 0 {
				t.Error("Template has no phases")
			}

			for i, phase := range template.Phases {
				validatePhase(t, phase, i, template.Name)
			}

			// Validate quality gates
			for i, gate := range template.QualityGates {
				validateQualityGate(t, gate, i, template.Name)
			}
		})
	}
}

func validatePhase(t *testing.T, phase SteerPhase, index int, templateName string) {
	t.Helper()

	prefix := func() string {
		return "Template '" + templateName + "' phase " + string(rune('0'+index))
	}

	// Validate mode
	if !phase.Mode.IsValid() {
		t.Errorf("%s: invalid mode %s", prefix(), phase.Mode)
	}

	// Validate max iterations
	if phase.MaxIterations <= 0 {
		t.Errorf("%s: max iterations must be > 0, got %d", prefix(), phase.MaxIterations)
	}
	if phase.MaxIterations > 100 {
		t.Errorf("%s: max iterations seems too high (%d), may be a mistake", prefix(), phase.MaxIterations)
	}

	// Validate stop conditions
	if len(phase.StopConditions) == 0 {
		t.Errorf("%s: no stop conditions defined", prefix())
	}

	for j, condition := range phase.StopConditions {
		if err := validateStopCondition(condition); err != nil {
			t.Errorf("%s condition %d: %v", prefix(), j, err)
		}
	}
}

func validateQualityGate(t *testing.T, gate QualityGate, index int, templateName string) {
	t.Helper()

	prefix := func() string {
		return "Template '" + templateName + "' gate " + string(rune('0'+index))
	}

	// Validate name
	if gate.Name == "" {
		t.Errorf("%s: name is empty", prefix())
	}

	// Validate condition
	if err := validateStopCondition(gate.Condition); err != nil {
		t.Errorf("%s: %v", prefix(), err)
	}

	// Validate failure action
	validActions := map[QualityGateAction]bool{
		ActionHalt:      true,
		ActionSkipPhase: true,
		ActionWarn:      true,
	}
	if !validActions[gate.FailureAction] {
		t.Errorf("%s: invalid failure action %s", prefix(), gate.FailureAction)
	}

	// Validate message
	if gate.Message == "" {
		t.Errorf("%s: message is empty", prefix())
	}
}

func validateStopCondition(condition StopCondition) error {
	if condition.Type == ConditionTypeSimple {
		if condition.Metric == "" {
			return newValidationError("simple condition missing metric")
		}
		if condition.CompareOperator == "" {
			return newValidationError("simple condition missing operator")
		}
		if !isValidOperator(condition.CompareOperator) {
			return newValidationError("invalid operator: %s", condition.CompareOperator)
		}
	} else if condition.Type == ConditionTypeCompound {
		if condition.Operator != LogicalAND && condition.Operator != LogicalOR {
			return newValidationError("compound condition has invalid logical operator: %s", condition.Operator)
		}
		if len(condition.Conditions) == 0 {
			return newValidationError("compound condition has no sub-conditions")
		}
		// Recursively validate sub-conditions
		for i, subCondition := range condition.Conditions {
			if err := validateStopCondition(subCondition); err != nil {
				return newValidationError("compound condition sub-condition %d: %v", i, err)
			}
		}
	} else {
		return newValidationError("invalid condition type: %s", condition.Type)
	}
	return nil
}

func isValidOperator(op ConditionOperator) bool {
	validOps := []ConditionOperator{
		OpGreaterThan,
		OpLessThan,
		OpGreaterThanEquals,
		OpLessThanEquals,
		OpEquals,
		OpNotEquals,
	}
	for _, valid := range validOps {
		if op == valid {
			return true
		}
	}
	return false
}

func newValidationError(format string, args ...interface{}) error {
	return &validationError{format: format, args: args}
}

type validationError struct {
	format string
	args   []interface{}
}

func (e *validationError) Error() string {
	if len(e.args) > 0 {
		return fmt.Sprintf(e.format, e.args...)
	}
	return e.format
}

func TestTemplates_BalancedTemplate(t *testing.T) {
	template := getBalancedTemplate()

	if template.Name != "Balanced" {
		t.Errorf("Expected name 'Balanced', got %s", template.Name)
	}

	// Should have phases covering multiple modes
	if len(template.Phases) < 3 {
		t.Errorf("Expected at least 3 phases for balanced approach, got %d", len(template.Phases))
	}

	// Verify it has quality gates
	if len(template.QualityGates) == 0 {
		t.Log("Warning: Balanced template has no quality gates")
	}

	// Should include progress phase
	hasProgress := false
	for _, phase := range template.Phases {
		if phase.Mode == ModeProgress {
			hasProgress = true
			break
		}
	}
	if !hasProgress {
		t.Error("Expected Balanced template to include Progress phase")
	}
}

func TestTemplates_RapidMVPTemplate(t *testing.T) {
	template := getRapidMVPTemplate()

	if template.Name != "Rapid MVP" {
		t.Errorf("Expected name 'Rapid MVP', got %s", template.Name)
	}

	// Should prioritize speed - fewer phases
	if len(template.Phases) > 4 {
		t.Logf("Warning: Rapid MVP has %d phases, seems high for rapid development", len(template.Phases))
	}

	// Should start with progress phase
	if len(template.Phases) > 0 && template.Phases[0].Mode != ModeProgress {
		t.Error("Expected Rapid MVP to start with Progress phase")
	}

	// Max iterations should be relatively low for rapid iteration
	for i, phase := range template.Phases {
		if phase.MaxIterations > 20 {
			t.Logf("Warning: Phase %d has high max iterations (%d) for rapid approach", i, phase.MaxIterations)
		}
	}
}

func TestTemplates_ProductionReadyTemplate(t *testing.T) {
	template := getProductionReadyTemplate()

	if template.Name != "Production Ready" {
		t.Errorf("Expected name 'Production Ready', got %s", template.Name)
	}

	// Should have comprehensive coverage
	if len(template.Phases) < 5 {
		t.Errorf("Expected at least 5 phases for production readiness, got %d", len(template.Phases))
	}

	// Should include security and testing phases
	hasSecurity := false
	hasTesting := false
	for _, phase := range template.Phases {
		if phase.Mode == ModeSecurity {
			hasSecurity = true
		}
		if phase.Mode == ModeTest {
			hasTesting = true
		}
	}

	if !hasSecurity {
		t.Error("Expected Production Ready template to include Security phase")
	}
	if !hasTesting {
		t.Error("Expected Production Ready template to include Test phase")
	}

	// Should have quality gates
	if len(template.QualityGates) == 0 {
		t.Error("Expected Production Ready template to have quality gates")
	}
}

func TestTemplates_RefactorTestFocusTemplate(t *testing.T) {
	template := getRefactorTestFocusTemplate()

	if template.Name != "Refactor & Test Focus" {
		t.Errorf("Expected name 'Refactor & Test Focus', got %s", template.Name)
	}

	// Should prioritize testing and refactoring
	hasTest := false
	hasRefactor := false
	for _, phase := range template.Phases {
		if phase.Mode == ModeTest {
			hasTest = true
		}
		if phase.Mode == ModeRefactor {
			hasRefactor = true
		}
	}

	if !hasTest {
		t.Error("Expected Refactor & Test Focus template to include Test phase")
	}
	if !hasRefactor {
		t.Error("Expected Refactor & Test Focus template to include Refactor phase")
	}
}

func TestTemplates_UXExcellenceTemplate(t *testing.T) {
	template := getUXExcellenceTemplate()

	if template.Name != "UX Excellence" {
		t.Errorf("Expected name 'UX Excellence', got %s", template.Name)
	}

	// Should include UX phase
	hasUX := false
	uxPhaseIndex := -1
	for i, phase := range template.Phases {
		if phase.Mode == ModeUX {
			hasUX = true
			uxPhaseIndex = i
			break
		}
	}

	if !hasUX {
		t.Error("Expected UX Excellence template to include UX phase")
	}

	// UX phase should have significant iteration budget
	if hasUX && uxPhaseIndex >= 0 {
		uxPhase := template.Phases[uxPhaseIndex]
		if uxPhase.MaxIterations < 10 {
			t.Logf("Warning: UX phase has only %d max iterations, may be too few for excellence", uxPhase.MaxIterations)
		}
	}

	// Should likely include explore phase for creativity
	hasExplore := false
	for _, phase := range template.Phases {
		if phase.Mode == ModeExplore {
			hasExplore = true
			break
		}
	}
	if !hasExplore {
		t.Log("Note: UX Excellence template doesn't include Explore phase")
	}
}

func TestTemplates_ModeCoverage(t *testing.T) {
	templates := GetBuiltInTemplates()

	// Track which modes are used across all templates
	modeUsage := make(map[SteerMode]int)
	for _, template := range templates {
		for _, phase := range template.Phases {
			modeUsage[phase.Mode]++
		}
	}

	t.Logf("Mode usage across %d templates:", len(templates))
	for mode, count := range modeUsage {
		t.Logf("  %s: %d phases", mode, count)
	}

	// Ensure common modes are represented
	importantModes := []SteerMode{ModeProgress, ModeUX, ModeTest, ModeRefactor}
	for _, mode := range importantModes {
		if modeUsage[mode] == 0 {
			t.Errorf("Important mode %s is not used in any template", mode)
		}
	}
}

func TestTemplates_TagsPresent(t *testing.T) {
	templates := GetBuiltInTemplates()

	for _, template := range templates {
		if len(template.Tags) == 0 {
			t.Logf("Warning: Template %s has no tags", template.Name)
		}
	}
}

func TestTemplates_UniqueIDs(t *testing.T) {
	templates := GetBuiltInTemplates()

	// Collect all phase IDs across all templates
	phaseIDs := make(map[string]string) // ID -> template name

	for _, template := range templates {
		for _, phase := range template.Phases {
			if existingTemplate, exists := phaseIDs[phase.ID]; exists {
				t.Errorf("Phase ID %s is duplicated between templates %s and %s",
					phase.ID, existingTemplate, template.Name)
			}
			phaseIDs[phase.ID] = template.Name
		}
	}
}

func TestTemplates_ReasonableStopConditions(t *testing.T) {
	templates := GetBuiltInTemplates()

	for _, template := range templates {
		for phaseIdx, phase := range template.Phases {
			for condIdx, condition := range phase.StopConditions {
				checkReasonableCondition(t, condition, template.Name, phaseIdx, condIdx)
			}
		}
	}
}

func checkReasonableCondition(t *testing.T, condition StopCondition, templateName string, phaseIdx, condIdx int) {
	t.Helper()

	if condition.Type == ConditionTypeSimple {
		// Check for unreasonable values
		metric := condition.Metric
		value := condition.Value

		switch metric {
		case "operational_targets_percentage", "accessibility_score", "ui_test_coverage",
			"unit_test_coverage", "integration_test_coverage", "tidiness_score":
			// Percentages should be 0-100
			if value < 0 || value > 100 {
				t.Errorf("Template %s phase %d condition %d: %s value %.2f is outside 0-100 range",
					templateName, phaseIdx, condIdx, metric, value)
			}
		case "loops":
			// Loops should be reasonable
			if value < 0 {
				t.Errorf("Template %s phase %d condition %d: loops value %.0f is negative",
					templateName, phaseIdx, condIdx, value)
			}
			if value > 1000 {
				t.Logf("Warning: Template %s phase %d condition %d: loops value %.0f seems very high",
					templateName, phaseIdx, condIdx, value)
			}
		}
	} else if condition.Type == ConditionTypeCompound {
		// Recursively check sub-conditions
		for i, subCondition := range condition.Conditions {
			checkReasonableCondition(t, subCondition, templateName, phaseIdx, i)
		}
	}
}
