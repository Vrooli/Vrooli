package validationrunner

import (
	"context"
	"fmt"
	"strings"

	"github.com/vrooli/vrooli/scenarios/template-manager/api/internal/catalog"
	"github.com/vrooli/vrooli/scenarios/template-manager/api/internal/templatecontracts"
	"github.com/vrooli/vrooli/scenarios/template-manager/api/internal/templateengine"
)

// defaultTemplateValidationFindingKey is the stable fallback debt key for a
// validation issue that carries no failure-class path. It deliberately does not
// embed the message so repeated runs of the same failure reuse one debt entry.
const defaultTemplateValidationFindingKey = "template-validate.issue"

type EngineRunner struct {
	Engine *templateengine.Engine
}

func NewEngineRunner(root string) (EngineRunner, error) {
	engine, err := templateengine.New(root)
	if err != nil {
		return EngineRunner{}, err
	}
	return EngineRunner{Engine: engine}, nil
}

func (r EngineRunner) ValidateTemplate(ctx context.Context, req ValidateRequest) (ValidateResult, error) {
	engine, err := r.engine()
	if err != nil {
		return ValidateResult{}, err
	}
	mode := req.Mode
	if mode == "" {
		mode = catalog.ModeShallow
	}
	templateID := strings.TrimSpace(req.TemplateID)
	if templateID == "" {
		templateID = "react-vite"
	}
	report, err := engine.ValidateTemplate(ctx, templatecontracts.TemplateValidateRequest{
		Mode:          templatecontracts.TemplateValidationMode(mode),
		TemplateName:  templateID,
		TestPreset:    req.TestPreset,
		WarningPolicy: templatecontracts.TemplateValidationWarningPolicy(req.WarningPolicy),
		RetainTemp:    req.RetainTemp,
	})
	if err != nil {
		return ValidateResult{}, err
	}
	result := ValidateResult{
		Success:    len(report.Issues) == 0,
		Mode:       catalog.ValidationMode(report.Mode),
		TemplateID: templateID,
		Target:     fmt.Sprintf("templates/scenarios/%s", templateID),
	}
	if report.TemplateName != "" {
		result.TemplateID = report.TemplateName
		result.Target = fmt.Sprintf("templates/scenarios/%s", report.TemplateName)
	}
	if report.WarningSummary.Total > 0 {
		for _, phase := range report.WarningSummary.Phases {
			result.PhaseResults = append(result.PhaseResults, catalog.PhaseResult{Phase: phase.Name, Status: "warnings", FindingCount: int32(phase.Count)})
		}
	}
	if len(result.PhaseResults) == 0 {
		result.PhaseResults = append(result.PhaseResults, catalog.PhaseResult{Phase: "template-validate", Status: statusFromSuccess(result.Success), FindingCount: int32(len(report.Issues))})
	}
	for _, issue := range report.Issues {
		result.Findings = append(result.Findings, findingFromValidationIssue(result.TemplateID, issue))
	}
	return result, nil
}

// findingFromValidationIssue projects a validation issue onto a debt finding.
//
// Debt identity must survive across runs: issue.Path is the stable failure
// class (one of the testGenieDeepValidation* constants), while the message
// carries volatile prose (temp paths, phase counts) that would mint a fresh
// debt key every run if slugified into the key. The key is derived from the
// class alone and the full prose is kept in the summary. Everything in
// report.Issues is a hard failure that fails the run — warnings live in
// report.WarningSummary — so the finding is filed at error severity.
func findingFromValidationIssue(templateID string, issue templatecontracts.TemplateValidationIssue) catalog.ValidationFinding {
	key := strings.TrimSpace(issue.Path)
	if key == "" {
		key = defaultTemplateValidationFindingKey
	}
	return catalog.ValidationFinding{
		Key:      normalizeFindingKey(templateID, key),
		Severity: "error",
		Summary:  firstNonEmpty(issue.Message, key),
		Source:   firstNonEmpty(issue.Template, "template validate"),
	}
}

func (r EngineRunner) RecordFleetDrift(ctx context.Context) (DriftResult, error) {
	engine, err := r.engine()
	if err != nil {
		return DriftResult{}, err
	}
	report, err := engine.DriftReport(ctx, templatecontracts.TemplateDriftRequest{All: true, JSON: true})
	if err != nil {
		return DriftResult{}, err
	}
	result := DriftResult{Success: !report.AnyDrifted(), Scenarios: make([]DriftScenario, 0, len(report.Scenarios))}
	for _, scenario := range report.Scenarios {
		result.Scenarios = append(result.Scenarios, DriftScenario{
			Scenario:        scenario.Scenario,
			TemplateID:      scenario.TemplateID,
			Status:          string(scenario.Status),
			ManifestDrifted: scenario.ManifestDrifted,
			ContentDrifted:  scenario.ContentDrifted,
			Message:         scenario.Message,
		})
	}
	return result, nil
}

func (r EngineRunner) engine() (*templateengine.Engine, error) {
	if r.Engine != nil {
		return r.Engine, nil
	}
	return templateengine.New("")
}
