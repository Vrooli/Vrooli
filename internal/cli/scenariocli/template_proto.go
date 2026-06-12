package scenariocli

import (
	"io"

	"github.com/vrooli/vrooli/internal/templatevalidation"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

// -----------------------------------------------------------------------------
// `scenario template list`
// -----------------------------------------------------------------------------

func scenarioTemplateVar(v TemplateVar) *cliv1.ScenarioTemplateVar {
	return &cliv1.ScenarioTemplateVar{
		Flag:        v.Flag,
		Description: v.Description,
		Default:     v.Default,
	}
}

func scenarioTemplateVarMap(src map[string]TemplateVar) map[string]*cliv1.ScenarioTemplateVar {
	if len(src) == 0 {
		return map[string]*cliv1.ScenarioTemplateVar{}
	}
	out := make(map[string]*cliv1.ScenarioTemplateVar, len(src))
	for k, v := range src {
		out[k] = scenarioTemplateVar(v)
	}
	return out
}

func scenarioTemplateHook(h TemplateHook) *cliv1.ScenarioTemplateHook {
	return &cliv1.ScenarioTemplateHook{
		Description: h.Description,
		Cmd:         h.Cmd,
		Cwd:         h.Cwd,
	}
}

func scenarioTemplateHooks(src []TemplateHook) []*cliv1.ScenarioTemplateHook {
	out := make([]*cliv1.ScenarioTemplateHook, 0, len(src))
	for _, h := range src {
		out = append(out, scenarioTemplateHook(h))
	}
	return out
}

func scenarioTemplateOrientation(o *TemplateOrientation) *cliv1.ScenarioTemplateOrientation {
	if o == nil {
		return nil
	}
	msg := &cliv1.ScenarioTemplateOrientation{
		Version:       o.Version,
		CopyTo:        o.CopyTo,
		StartDocument: o.StartDocument,
		Finalize: &cliv1.ScenarioTemplateOrientationFinalize{
			Cleanup: o.Finalize.Cleanup,
			Message: o.Finalize.Message,
		},
	}
	for _, step := range o.Steps {
		stepMsg := &cliv1.ScenarioTemplateOrientationStep{
			Id:          step.ID,
			Title:       step.Title,
			Description: step.Description,
			Docs:        step.Docs,
			Required:    step.Required != nil && *step.Required,
		}
		for _, check := range step.Checks {
			stepMsg.Checks = append(stepMsg.Checks, &cliv1.ScenarioTemplateOrientationCheck{
				Kind:     check.Kind,
				Path:     check.Path,
				Pattern:  check.Pattern,
				Query:    check.Query,
				Text:     check.Text,
				Run:      check.Run,
				Timeout:  check.Timeout,
				Optional: check.Optional,
			})
		}
		msg.Steps = append(msg.Steps, stepMsg)
	}
	return msg
}

func scenarioTemplateManifest(m TemplateManifest) *cliv1.ScenarioTemplateManifest {
	msg := &cliv1.ScenarioTemplateManifest{
		Name:          m.Name,
		Version:       m.Version,
		DisplayName:   m.DisplayName,
		Description:   m.Description,
		Stack:         m.Stack,
		StartDocument: m.StartDocument,
		Design: &cliv1.ScenarioTemplateDesign{
			Adapter:  m.Design.Adapter,
			Default:  m.Design.Default,
			Required: m.Design.Required,
		},
		Orientation:  scenarioTemplateOrientation(m.Orientation),
		RequiredVars: scenarioTemplateVarMap(m.RequiredVars),
		OptionalVars: scenarioTemplateVarMap(m.OptionalVars),
		Docs:         copyStringMap(m.Docs),
		CopyExcludes: m.CopyExcludes,
		PostHooks:    scenarioTemplateHooks(m.PostHooks),
	}
	for _, r := range m.Relocations {
		msg.Relocations = append(msg.Relocations, &cliv1.ScenarioTemplateRelocation{
			Description: r.Description,
			From:        r.From,
			To:          r.To,
			Post:        scenarioTemplateHooks(r.Post),
		})
	}
	return msg
}

// copyStringMap maps a map[string]string onto a proto map<string,string>.
func copyStringMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

// ScenarioTemplateListResponse maps discovered templates onto the wire contract
// (cliout.WriteSuccessJSON under the "templates" key).
func ScenarioTemplateListResponse(templates []TemplateInfo) *cliv1.ScenarioTemplateListResponse {
	resp := &cliv1.ScenarioTemplateListResponse{Success: true}
	for _, item := range templates {
		resp.Templates = append(resp.Templates, &cliv1.ScenarioTemplateInfo{
			Name:     item.Name,
			Path:     item.Path,
			Manifest: scenarioTemplateManifest(item.Manifest),
			Missing:  item.Missing,
		})
	}
	return resp
}

func writeScenarioTemplateListJSON(w io.Writer, templates []TemplateInfo) error {
	return marshalScenarioStatus(w, ScenarioTemplateListResponse(templates))
}

// -----------------------------------------------------------------------------
// `scenario template drift`
// -----------------------------------------------------------------------------

// ScenarioTemplateDriftResponse maps a TemplateDriftReport onto the wire
// contract (cliout.WriteSuccessJSON under the "drift" key).
func ScenarioTemplateDriftResponse(report TemplateDriftReport) *cliv1.ScenarioTemplateDriftResponse {
	drift := &cliv1.ScenarioTemplateDriftReport{}
	for _, s := range report.Scenarios {
		scenarioMsg := &cliv1.ScenarioTemplateDriftScenario{
			Scenario:            s.Scenario,
			TemplateId:          s.TemplateID,
			RecordedVersion:     s.RecordedVersion,
			CurrentVersion:      s.CurrentVersion,
			RecordedManifestSha: s.RecordedManifest,
			CurrentManifestSha:  s.CurrentManifest,
			RecordedContentSha:  s.RecordedContent,
			CurrentContentSha:   s.CurrentContent,
			ManifestDrifted:     s.ManifestDrifted,
			ContentDrifted:      s.ContentDrifted,
			Status:              string(s.Status),
			Message:             s.Message,
		}
		for _, fd := range s.FileDiffs {
			scenarioMsg.FileDiffs = append(scenarioMsg.FileDiffs, &cliv1.ScenarioTemplateDriftFileDiff{
				Path:   fd.Path,
				Reason: fd.Reason,
			})
		}
		drift.Scenarios = append(drift.Scenarios, scenarioMsg)
	}
	return &cliv1.ScenarioTemplateDriftResponse{Success: true, Drift: drift}
}

func writeScenarioTemplateDriftJSON(w io.Writer, report TemplateDriftReport) error {
	return marshalScenarioStatus(w, ScenarioTemplateDriftResponse(report))
}

// -----------------------------------------------------------------------------
// `scenario template validate`
// -----------------------------------------------------------------------------

func scenarioTemplateWarningSummary(s TemplateValidationWarningSummary) *cliv1.ScenarioTemplateValidationWarningSummary {
	msg := &cliv1.ScenarioTemplateValidationWarningSummary{
		Total: int32(s.Total),
	}
	for _, phase := range s.Phases {
		phaseMsg := &cliv1.ScenarioTemplateValidationPhaseWarningSummary{
			Name:  phase.Name,
			Count: int32(phase.Count),
		}
		for _, warn := range phase.Warnings {
			phaseMsg.Warnings = append(phaseMsg.Warnings, &cliv1.ScenarioTemplateValidationWarning{
				Message:      warn.Message,
				Source:       warn.Source,
				LogPath:      warn.LogPath,
				ArtifactPath: warn.ArtifactPath,
			})
		}
		msg.Phases = append(msg.Phases, phaseMsg)
	}
	return msg
}

// ScenarioTemplateValidateResponse maps a TemplateValidationReport onto the wire
// contract (cliout.WriteFieldsWithSuccess; success mirrors len(issues)==0).
func ScenarioTemplateValidateResponse(report TemplateValidationReport) *cliv1.ScenarioTemplateValidateResponse {
	reportMsg := &cliv1.ScenarioTemplateValidationReport{
		Mode:           string(report.Mode),
		TemplateName:   report.TemplateName,
		TestPreset:     report.TestPreset,
		WarningPolicy:  string(report.WarningPolicy),
		WarningSummary: scenarioTemplateWarningSummary(report.WarningSummary),
		Count:          int32(report.Count),
	}
	for _, run := range report.DeepRuns {
		reportMsg.DeepRuns = append(reportMsg.DeepRuns, &cliv1.ScenarioTemplateValidationDeepRun{
			Template:            run.Template,
			RunId:               run.RunID,
			ScenarioId:          run.ScenarioID,
			ScenarioPath:        run.ScenarioPath,
			TempRoot:            run.TempRoot,
			TestPreset:          run.TestPreset,
			WarningSummary:      scenarioTemplateWarningSummary(run.WarningSummary),
			RetainedTemp:        run.RetainedTemp,
			CleanupStatus:       run.CleanupStatus,
			RelocationArtifacts: run.RelocationArtifacts,
			CleanupCommand:      run.CleanupCommand,
		})
	}
	for _, issue := range report.Issues {
		reportMsg.Issues = append(reportMsg.Issues, &cliv1.ScenarioTemplateValidationIssue{
			Template: issue.Template,
			Path:     issue.Path,
			Message:  issue.Message,
		})
	}
	return &cliv1.ScenarioTemplateValidateResponse{
		Success: len(report.Issues) == 0,
		Report:  reportMsg,
	}
}

func writeScenarioTemplateValidateJSON(w io.Writer, report TemplateValidationReport) error {
	return marshalScenarioStatus(w, ScenarioTemplateValidateResponse(report))
}

// -----------------------------------------------------------------------------
// `scenario template cleanup`
// -----------------------------------------------------------------------------

func scenarioTemplateRun(r templatevalidation.Run) *cliv1.ScenarioTemplateCleanupRun {
	return &cliv1.ScenarioTemplateCleanupRun{
		MarkerPath: r.MarkerPath,
		Marker: &cliv1.ScenarioTemplateCleanupRunMarker{
			Version:             r.Marker.Version,
			RunId:               r.Marker.RunID,
			RepoRoot:            r.Marker.RepoRoot,
			Template:            r.Marker.Template,
			ScenarioId:          r.Marker.ScenarioID,
			ScenarioPath:        r.Marker.ScenarioPath,
			TempRoot:            r.Marker.TempRoot,
			CreatedAt:           formatTime(r.Marker.CreatedAt),
			Retained:            r.Marker.Retained,
			CreatorPid:          int32(r.Marker.CreatorPID),
			Completed:           r.Marker.Completed,
			CleanupStatus:       r.Marker.CleanupStatus,
			RelocationArtifacts: r.Marker.RelocationArtifacts,
		},
		Age: r.Age,
	}
}

func scenarioTemplateRunPtr(r *templatevalidation.Run) *cliv1.ScenarioTemplateCleanupRun {
	if r == nil {
		return nil
	}
	return scenarioTemplateRun(*r)
}

func scenarioTemplateRuns(src []templatevalidation.Run) []*cliv1.ScenarioTemplateCleanupRun {
	out := make([]*cliv1.ScenarioTemplateCleanupRun, 0, len(src))
	for _, r := range src {
		out = append(out, scenarioTemplateRun(r))
	}
	return out
}

// ScenarioTemplateCleanupResponse maps a TemplateCleanupResult onto the wire
// contract (cliout.WriteFieldsWithSuccess; success mirrors len(failures)==0).
func ScenarioTemplateCleanupResponse(result TemplateCleanupResult) *cliv1.ScenarioTemplateCleanupResponse {
	cleanup := &cliv1.ScenarioTemplateCleanupResult{
		DryRun:             result.DryRun,
		OlderThan:          int64(result.OlderThan),
		IncludeRetained:    result.IncludeRetained,
		RunId:              result.RunID,
		Eligible:           scenarioTemplateRuns(result.Eligible),
		Removed:            scenarioTemplateRuns(result.Removed),
		NeedsProtoGenerate: result.NeedsProtoGenerate,
		ProtoGenerateRan:   result.ProtoGenerateRan,
		Message:            result.Message,
	}
	for _, s := range result.Skipped {
		cleanup.Skipped = append(cleanup.Skipped, &cliv1.ScenarioTemplateCleanupSkippedRun{
			Run:    scenarioTemplateRunPtr(s.Run),
			Path:   s.Path,
			Reason: s.Reason,
		})
	}
	for _, f := range result.Failures {
		cleanup.Failures = append(cleanup.Failures, &cliv1.ScenarioTemplateCleanupFailedRun{
			Run:   scenarioTemplateRunPtr(f.Run),
			Path:  f.Path,
			Error: f.Error,
		})
	}
	return &cliv1.ScenarioTemplateCleanupResponse{
		Success: len(result.Failures) == 0,
		Cleanup: cleanup,
	}
}

func writeScenarioTemplateCleanupJSON(w io.Writer, result TemplateCleanupResult) error {
	return marshalScenarioStatus(w, ScenarioTemplateCleanupResponse(result))
}
