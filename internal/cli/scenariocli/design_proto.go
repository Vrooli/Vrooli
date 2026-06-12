package scenariocli

import (
	"io"

	"github.com/vrooli/vrooli/internal/resources"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

// -----------------------------------------------------------------------------
// `scenario design list` / `show`
// -----------------------------------------------------------------------------

func scenarioDesignKitInfo(kit DesignKitInfo) *cliv1.ScenarioDesignKitInfo {
	manifest := &cliv1.ScenarioDesignKitManifest{
		Id:          kit.Manifest.ID,
		Name:        kit.Manifest.Name,
		Version:     kit.Manifest.Version,
		Default:     kit.Manifest.Default,
		Description: kit.Manifest.Description,
		Tags:        kit.Manifest.Tags,
		Adapters:    map[string]*cliv1.ScenarioDesignKitAdapter{},
	}
	for id, adapter := range kit.Manifest.Adapters {
		manifest.Adapters[id] = &cliv1.ScenarioDesignKitAdapter{
			Path:     adapter.Path,
			Supports: adapter.Supports,
		}
	}
	return &cliv1.ScenarioDesignKitInfo{
		Id:       kit.ID,
		Path:     kit.Path,
		Manifest: manifest,
		Missing:  kit.Missing,
	}
}

// ScenarioDesignListResponse maps discovered design kits onto the wire contract
// (cliout.WriteSuccessJSON under the "designKits" key).
func ScenarioDesignListResponse(kits []DesignKitInfo) *cliv1.ScenarioDesignListResponse {
	resp := &cliv1.ScenarioDesignListResponse{Success: true}
	for _, kit := range kits {
		resp.DesignKits = append(resp.DesignKits, scenarioDesignKitInfo(kit))
	}
	return resp
}

func writeScenarioDesignListJSON(w io.Writer, kits []DesignKitInfo) error {
	return marshalScenarioStatus(w, ScenarioDesignListResponse(kits))
}

// ScenarioDesignShowResponse maps a single design kit onto the wire contract
// (cliout.WriteSuccessJSON under the "designKit" key).
func ScenarioDesignShowResponse(info DesignKitInfo) *cliv1.ScenarioDesignShowResponse {
	return &cliv1.ScenarioDesignShowResponse{
		Success:   true,
		DesignKit: scenarioDesignKitInfo(info),
	}
}

func writeScenarioDesignShowJSON(w io.Writer, info DesignKitInfo) error {
	return marshalScenarioStatus(w, ScenarioDesignShowResponse(info))
}

// -----------------------------------------------------------------------------
// `scenario design validate`
// -----------------------------------------------------------------------------

// ScenarioDesignValidateResponse maps a DesignValidationReport onto the wire
// contract (cliout.WriteSuccessJSON under the "designValidation" key).
func ScenarioDesignValidateResponse(report DesignValidationReport) *cliv1.ScenarioDesignValidateResponse {
	validation := &cliv1.ScenarioDesignValidationReport{
		Count: int32(report.Count),
	}
	for _, issue := range report.Issues {
		validation.Issues = append(validation.Issues, &cliv1.ScenarioDesignValidationIssue{
			Kit:     issue.Kit,
			Adapter: issue.Adapter,
			Path:    issue.Path,
			Message: issue.Message,
		})
	}
	return &cliv1.ScenarioDesignValidateResponse{
		Success:          true,
		DesignValidation: validation,
	}
}

func writeScenarioDesignValidateJSON(w io.Writer, report DesignValidationReport) error {
	return marshalScenarioStatus(w, ScenarioDesignValidateResponse(report))
}

// -----------------------------------------------------------------------------
// `scenario orient`
// -----------------------------------------------------------------------------

func scenarioOrientationStep(step OrientationStepReport) *cliv1.ScenarioOrientationStep {
	msg := &cliv1.ScenarioOrientationStep{
		Id:          step.ID,
		Title:       step.Title,
		Description: step.Description,
		Docs:        step.Docs,
		Required:    step.Required,
		Complete:    step.Complete,
	}
	for _, check := range step.Checks {
		msg.Checks = append(msg.Checks, &cliv1.ScenarioOrientationCheck{
			Kind:     check.Kind,
			Label:    check.Label,
			Passed:   check.Passed,
			Skipped:  check.Skipped,
			Message:  check.Message,
			Optional: check.Optional,
		})
	}
	return msg
}

// ScenarioOrientationResponse maps an OrientationReport onto the wire contract
// (cliout.WriteFieldsWithSuccess; success is always true).
func ScenarioOrientationResponse(report OrientationReport) *cliv1.ScenarioOrientationResponse {
	orientation := &cliv1.ScenarioOrientationReport{
		Scenario:        report.Scenario,
		ScenarioPath:    report.ScenarioPath,
		OrientationPath: report.OrientationPath,
		Finalized:       report.Finalized,
		Template: &cliv1.ScenarioGenerationTemplateRef{
			Id:      report.Template.ID,
			Version: report.Template.Version,
		},
		Design: &cliv1.ScenarioGenerationDesignRef{
			Id:      report.Design.ID,
			Version: report.Design.Version,
			Adapter: report.Design.Adapter,
		},
		StartDocument:    report.StartDocument,
		Completed:        int32(report.Completed),
		Required:         int32(report.Required),
		Message:          report.Message,
		FinalizeRequired: report.FinalizeRequired,
	}
	for _, step := range report.Steps {
		orientation.Steps = append(orientation.Steps, scenarioOrientationStep(step))
	}
	if report.NextStep != nil {
		orientation.NextStep = scenarioOrientationStep(*report.NextStep)
	}
	return &cliv1.ScenarioOrientationResponse{
		Success:     true,
		Orientation: orientation,
	}
}

func writeScenarioOrientationJSON(w io.Writer, report OrientationReport) error {
	return marshalScenarioStatus(w, ScenarioOrientationResponse(report))
}

// -----------------------------------------------------------------------------
// `scenario validate-env`
// -----------------------------------------------------------------------------

// ScenarioEnvValidationResponse maps a ScenarioEnvValidationReport onto the wire
// contract (cliout.WriteFieldsWithSuccess; success mirrors report.Passed).
func ScenarioEnvValidationResponse(report resources.ScenarioEnvValidationReport) *cliv1.ScenarioEnvValidationResponse {
	reportMsg := &cliv1.ScenarioEnvValidationReport{
		Scenario: report.Scenario,
		Values:   copyStringMap(report.Values),
		Passed:   report.Passed,
	}
	for _, issue := range report.Issues {
		reportMsg.Issues = append(reportMsg.Issues, &cliv1.ScenarioValidationIssue{
			Severity: issue.Severity,
			Message:  issue.Message,
		})
	}
	for _, rr := range report.ResourceReports {
		reportMsg.ResourceReports = append(reportMsg.ResourceReports, &cliv1.ScenarioResourceReport{
			Name:         rr.Name,
			ManifestPath: rr.Manifest,
			Values:       copyStringMap(rr.Values),
			Warnings:     rr.Warnings,
		})
	}
	return &cliv1.ScenarioEnvValidationResponse{
		Success: report.Passed,
		Report:  reportMsg,
	}
}

func writeScenarioEnvValidationJSON(w io.Writer, report resources.ScenarioEnvValidationReport) error {
	return marshalScenarioStatus(w, ScenarioEnvValidationResponse(report))
}
