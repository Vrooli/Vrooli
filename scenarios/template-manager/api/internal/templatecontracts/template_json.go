package templatecontracts

import (
	"io"

	"github.com/vrooli/vrooli/internal/templatevalidation"
)

func writeScenarioTemplateListJSON(w io.Writer, templates []TemplateInfo) error {
	return marshalScenarioStatus(w, struct {
		Success   bool           `json:"success"`
		Templates []TemplateInfo `json:"templates"`
	}{Success: true, Templates: templates})
}

func writeScenarioTemplateDriftJSON(w io.Writer, report TemplateDriftReport) error {
	return marshalScenarioStatus(w, struct {
		Success bool                `json:"success"`
		Drift   TemplateDriftReport `json:"drift"`
	}{Success: true, Drift: report})
}

func writeScenarioTemplateValidateJSON(w io.Writer, report TemplateValidationReport) error {
	return marshalScenarioStatus(w, struct {
		Success bool                     `json:"success"`
		Report  TemplateValidationReport `json:"report"`
	}{Success: len(report.Issues) == 0, Report: report})
}

type templateCleanupRun struct {
	MarkerPath string                       `json:"marker_path"`
	Marker     templatevalidation.RunMarker `json:"marker"`
	Age        string                       `json:"age"`
}

func templateCleanupRuns(src []templatevalidation.Run) []templateCleanupRun {
	out := make([]templateCleanupRun, 0, len(src))
	for _, run := range src {
		out = append(out, templateCleanupRun{
			MarkerPath: run.MarkerPath,
			Marker:     run.Marker,
			Age:        run.Age,
		})
	}
	return out
}

func writeScenarioTemplateCleanupJSON(w io.Writer, result TemplateCleanupResult) error {
	return marshalScenarioStatus(w, struct {
		Success bool `json:"success"`
		Cleanup struct {
			DryRun             bool                            `json:"dry_run"`
			OlderThan          string                          `json:"older_than"`
			IncludeRetained    bool                            `json:"include_retained"`
			RunID              string                          `json:"run_id"`
			Eligible           []templateCleanupRun            `json:"eligible"`
			Skipped            []templatevalidation.SkippedRun `json:"skipped"`
			Failures           []templatevalidation.FailedRun  `json:"failures"`
			Removed            []templateCleanupRun            `json:"removed"`
			NeedsProtoGenerate bool                            `json:"needs_proto_generate"`
			ProtoGenerateRan   bool                            `json:"proto_generate_ran"`
			Message            string                          `json:"message"`
		} `json:"cleanup"`
	}{
		Success: len(result.Failures) == 0,
		Cleanup: struct {
			DryRun             bool                            `json:"dry_run"`
			OlderThan          string                          `json:"older_than"`
			IncludeRetained    bool                            `json:"include_retained"`
			RunID              string                          `json:"run_id"`
			Eligible           []templateCleanupRun            `json:"eligible"`
			Skipped            []templatevalidation.SkippedRun `json:"skipped"`
			Failures           []templatevalidation.FailedRun  `json:"failures"`
			Removed            []templateCleanupRun            `json:"removed"`
			NeedsProtoGenerate bool                            `json:"needs_proto_generate"`
			ProtoGenerateRan   bool                            `json:"proto_generate_ran"`
			Message            string                          `json:"message"`
		}{
			DryRun:             result.DryRun,
			OlderThan:          result.OlderThan.String(),
			IncludeRetained:    result.IncludeRetained,
			RunID:              result.RunID,
			Eligible:           templateCleanupRuns(result.Eligible),
			Skipped:            result.Skipped,
			Failures:           result.Failures,
			Removed:            templateCleanupRuns(result.Removed),
			NeedsProtoGenerate: result.NeedsProtoGenerate,
			ProtoGenerateRan:   result.ProtoGenerateRan,
			Message:            result.Message,
		},
	})
}
