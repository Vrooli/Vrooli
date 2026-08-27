package resourcecli

import (
	"fmt"
	"io"
	"strings"

	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/control"
	"github.com/vrooli/vrooli/internal/discovery"
	"github.com/vrooli/vrooli/internal/resources"
)

func WriteList(w io.Writer, format cliout.Format, items []resources.Resource, failures []discovery.Failure) error {
	return cliout.RenderJSONOr(w, format, func(w io.Writer) error { return cliout.WriteProtoJSON(w, ResourceListResponse(items, failures)) }, func(w io.Writer) error {
		rows := make([][]string, 0, len(items))
		for _, item := range items {
			rows = append(rows, []string{
				item.Name,
				cliout.BoolLabel(item.Enabled),
				item.ControlMode,
				item.Driver,
				cliout.BoolLabel(item.Registered),
			})
		}
		if err := cliout.RenderTable(w, []string{"Name", "Enabled", "Control", "Driver", "Registered"}, rows); err != nil {
			return err
		}
		if len(failures) > 0 {
			_, _ = fmt.Fprintf(w, "\nSkipped %d resources with discovery errors:\n", len(failures))
			for _, failure := range failures {
				_, _ = fmt.Fprintf(w, "  %s: %s\n", failure.Name, failure.Error)
			}
		}
		return nil
	})
}

func WriteStatuses(w io.Writer, format cliout.Format, items []resources.Status, failures []discovery.Failure) error {
	return cliout.RenderJSONOr(w, format, func(w io.Writer) error { return cliout.WriteProtoJSON(w, ResourceStatusesResponse(items, failures)) }, func(w io.Writer) error {
		rows := make([][]string, 0, len(items))
		for _, item := range items {
			healthy := "n/a"
			if item.Healthy != nil {
				if *item.Healthy {
					healthy = "healthy"
				} else {
					healthy = "unhealthy"
				}
			}
			rows = append(rows, []string{
				item.Resource.Name,
				cliout.BoolLabel(item.Resource.Enabled),
				item.Resource.ControlMode,
				cliout.BoolLabel(item.Running),
				healthy,
				item.Message,
			})
		}
		if err := cliout.RenderTable(w, []string{"Name", "Enabled", "Control", "Running", "Health", "Status"}, rows); err != nil {
			return err
		}
		if len(failures) > 0 {
			_, _ = fmt.Fprintf(w, "\nSkipped %d resources with discovery errors:\n", len(failures))
			for _, failure := range failures {
				_, _ = fmt.Fprintf(w, "  %s: %s\n", failure.Name, failure.Error)
			}
		}
		return nil
	})
}

func WriteStatus(w io.Writer, format cliout.Format, item resources.Status) error {
	return cliout.RenderJSONOr(w, format, func(w io.Writer) error { return cliout.WriteProtoJSON(w, ResourceStatusResponse(item)) }, func(w io.Writer) error {
		rows := [][]string{
			{"Name", item.Resource.Name},
			{"Enabled", cliout.BoolLabel(item.Resource.Enabled)},
			{"Control", item.Resource.ControlMode},
			{"Driver", item.Resource.Driver},
			{"Installed", cliout.BoolLabel(item.Installed)},
			{"Running", cliout.BoolLabel(item.Running)},
		}
		if item.Healthy != nil {
			rows = append(rows, []string{"Healthy", cliout.BoolLabel(*item.Healthy)})
		}
		if item.Serving != nil {
			rows = append(rows, []string{"Serving", cliout.BoolLabel(*item.Serving)})
		}
		if item.DeclaredMode != "" {
			rows = append(rows, []string{"Declared mode", item.DeclaredMode})
			observed := item.ObservedMode
			if observed == "" {
				observed = "unknown"
			}
			rows = append(rows, []string{"Observed mode", observed})
			rows = append(rows, []string{"Mode drift", cliout.BoolLabel(item.ModeDrift)})
			if item.ModeReason != "" {
				rows = append(rows, []string{"Mode reason", item.ModeReason})
			}
		}
		if item.StatusCode != "" {
			rows = append(rows, []string{"Status Code", item.StatusCode})
		}
		rows = append(rows, []string{"Status", item.Message})
		if item.ProbeError != "" {
			rows = append(rows, []string{"Probe Error", item.ProbeError})
		}
		return cliout.RenderTable(w, []string{"Field", "Value"}, rows)
	})
}

func WriteInfo(w io.Writer, format cliout.Format, item resources.Status) error {
	return cliout.RenderJSONOr(w, format, func(w io.Writer) error { return cliout.WriteProtoJSON(w, ResourceInfoResponse(item)) }, func(w io.Writer) error { return WriteStatus(w, format, item) })
}

func WriteControlReport(w io.Writer, format cliout.Format, reportKey, verb string, report any, items, failed []control.ResultItem) error {
	return cliout.RenderJSONOr(w, format, func(w io.Writer) error {
		switch typed := report.(type) {
		case *control.StartReport:
			return cliout.WriteProtoJSON(w, ResourceStartAllResponse(*typed))
		case control.StartReport:
			return cliout.WriteProtoJSON(w, ResourceStartAllResponse(typed))
		case *control.StopReport:
			return cliout.WriteProtoJSON(w, ResourceStopAllResponse(*typed))
		case control.StopReport:
			return cliout.WriteProtoJSON(w, ResourceStopAllResponse(typed))
		default:
			return cliout.WriteSuccessJSON(w, reportKey, report)
		}
	}, func(w io.Writer) error {
		for _, item := range items {
			_, _ = fmt.Fprintf(w, "%s %s\n", verb, item.Name)
		}
		for _, item := range failed {
			_, _ = fmt.Fprintf(w, "Failed %s: %s\n", item.Name, item.Error)
		}
		return nil
	})
}

func WriteDeprecationReport(w io.Writer, format cliout.Format, report resources.DeprecationReport) error {
	return cliout.RenderJSONOr(w, format, func(w io.Writer) error { return cliout.WriteProtoJSON(w, ResourceDeprecationResponse(report)) }, func(w io.Writer) error {
		_, _ = fmt.Fprintf(w, "Deprecated %s\n", report.Resource.Name)
		if report.ArchiveDir != "" {
			_, _ = fmt.Fprintf(w, "Archive: %s\n", report.ArchiveDir)
		}
		return nil
	})
}

func WriteRestoreReport(w io.Writer, format cliout.Format, report resources.RestoreReport) error {
	return cliout.RenderJSONOr(w, format, func(w io.Writer) error { return cliout.WriteProtoJSON(w, ResourceRestoreResponse(report)) }, func(w io.Writer) error {
		_, _ = fmt.Fprintf(w, "Restored %s to %s\n", report.Resource.Name, report.RestoredPath)
		return nil
	})
}

func WriteBlueprintArchiveReport(w io.Writer, format cliout.Format, report resources.BlueprintArchiveReport) error {
	return cliout.RenderJSONOr(w, format, func(w io.Writer) error { return cliout.WriteProtoJSON(w, ResourceBlueprintArchiveResponse(report)) }, func(w io.Writer) error {
		_, _ = fmt.Fprintf(w, "Archived %s to blueprint-only state\n", report.Resource.Name)
		if report.ArchiveDir != "" {
			_, _ = fmt.Fprintf(w, "Archive: %s\n", report.ArchiveDir)
		}
		return nil
	})
}

func WriteBlueprintRestoreReport(w io.Writer, format cliout.Format, report resources.BlueprintRestoreReport) error {
	return cliout.RenderJSONOr(w, format, func(w io.Writer) error { return cliout.WriteProtoJSON(w, ResourceBlueprintRestoreResponse(report)) }, func(w io.Writer) error {
		_, _ = fmt.Fprintf(w, "Restored blueprint-archived %s to %s\n", report.Resource.Name, report.RestoredPath)
		return nil
	})
}

func WriteArchiveGCReport(w io.Writer, format cliout.Format, report resources.ArchiveGCReport, kind string) error {
	return cliout.RenderJSONOr(w, format, func(w io.Writer) error { return cliout.WriteProtoJSON(w, ResourceArchiveGCResponse(report)) }, func(w io.Writer) error {
		_, _ = fmt.Fprintf(w, "Purged %d %s archives\n", len(report.Removed), kind)
		return nil
	})
}

func WriteDeprecatedList(w io.Writer, format cliout.Format, items []resources.DeprecatedResource) error {
	return cliout.RenderJSONOr(w, format, func(w io.Writer) error { return cliout.WriteProtoJSON(w, ResourceListDeprecatedResponse(items)) }, func(w io.Writer) error {
		rows := make([][]string, 0, len(items))
		for _, item := range items {
			state := "deprecated"
			if strings.TrimSpace(item.PurgedAt) != "" {
				state = "purged"
			}
			rows = append(rows, []string{item.Name, state, item.DeprecatedAt, item.PurgeAfter, item.Replacement})
		}
		return cliout.RenderTable(w, []string{"Name", "State", "Deprecated", "Purge After", "Replacement"}, rows)
	})
}

func WriteBlueprintArchivedList(w io.Writer, format cliout.Format, items []resources.BlueprintArchivedResource) error {
	return cliout.RenderJSONOr(w, format, func(w io.Writer) error { return cliout.WriteProtoJSON(w, ResourceListBlueprintArchivedResponse(items)) }, func(w io.Writer) error {
		rows := make([][]string, 0, len(items))
		for _, item := range items {
			state := "blueprint-archived"
			if strings.TrimSpace(item.PurgedAt) != "" {
				state = "purged"
			}
			rows = append(rows, []string{item.Name, state, item.ArchivedAt, item.PurgeAfter, item.BlueprintName})
		}
		return cliout.RenderTable(w, []string{"Name", "State", "Archived", "Purge After", "Blueprint"}, rows)
	})
}

func WriteBlueprintList(w io.Writer, format cliout.Format, items []resources.Blueprint) error {
	return cliout.RenderJSONOr(w, format, func(w io.Writer) error { return cliout.WriteProtoJSON(w, ResourceBlueprintListResponse(items)) }, func(w io.Writer) error {
		rows := make([][]string, 0, len(items))
		for _, item := range items {
			rows = append(rows, []string{item.Name, item.Category, item.Status, item.SuggestedTemplate, item.LastReviewed})
		}
		return cliout.RenderTable(w, []string{"Name", "Category", "Status", "Template", "Reviewed"}, rows)
	})
}

func WriteBlueprintInfo(w io.Writer, format cliout.Format, item resources.Blueprint) error {
	return cliout.RenderJSONOr(w, format, func(w io.Writer) error { return cliout.WriteProtoJSON(w, ResourceBlueprintInfoResponse(item)) }, func(w io.Writer) error {
		rows := [][]string{
			{"Name", item.Name},
			{"Display Name", item.DisplayName},
			{"Category", item.Category},
			{"Status", item.Status},
			{"Integration Kind", item.IntegrationKind},
			{"Template", item.SuggestedTemplate},
			{"Reviewed", item.LastReviewed},
			{"Summary", item.Summary},
			{"Why It Matters", item.WhyItMatters},
		}
		return cliout.RenderTable(w, []string{"Field", "Value"}, rows)
	})
}

func WriteBlueprintSearch(w io.Writer, format cliout.Format, query string, items []resources.Blueprint) error {
	return cliout.RenderJSONOr(w, format, func(w io.Writer) error {
		return cliout.WriteProtoJSON(w, ResourceBlueprintSearchResponse(query, items))
	}, func(w io.Writer) error {
		rows := make([][]string, 0, len(items))
		for _, item := range items {
			rows = append(rows, []string{item.Name, item.Category, item.Status, item.Summary})
		}
		return cliout.RenderTable(w, []string{"Name", "Category", "Status", "Summary"}, rows)
	})
}

func WriteBlueprintValidationReport(w io.Writer, format cliout.Format, report resources.BlueprintValidationReport) error {
	return cliout.RenderJSONOr(w, format, func(w io.Writer) error { return cliout.WriteProtoJSON(w, ResourceBlueprintValidationResponse(report)) }, func(w io.Writer) error {
		_, _ = fmt.Fprintf(w, "Validated %d resource blueprints\n", report.Count)
		return nil
	})
}

func WriteSchemaValidationReport(w io.Writer, format cliout.Format, report resources.ResourceSchemaValidationReport) error {
	return cliout.RenderJSONOr(w, format, func(w io.Writer) error { return cliout.WriteProtoJSON(w, ResourceSchemaValidationResponse(report)) }, func(w io.Writer) error {
		status := "passed"
		if !report.Passed {
			status = "failed"
		}
		if _, err := fmt.Fprintf(w, "Resource schema validation %s\n", status); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "Resources: %d\n", report.ResourceCount); err != nil {
			return err
		}
		for _, issue := range report.ArtifactIssues {
			if _, err := fmt.Fprintf(w, "- artifact: %s\n  [error] %s\n", issue.Path, issue.Message); err != nil {
				return err
			}
		}
		for _, item := range report.MissingReferences {
			if _, err := fmt.Fprintf(w, "- scenario: %s\n  [error] missing resource %s in %s\n", item.Scenario, item.Resource, item.ManifestPath); err != nil {
				return err
			}
		}
		return nil
	})
}

func WriteSchemaSyncReport(w io.Writer, format cliout.Format, report resources.ResourceSchemaSyncReport) error {
	return cliout.RenderJSONOr(w, format, func(w io.Writer) error { return cliout.WriteProtoJSON(w, ResourceSchemaSyncResponse(report)) }, func(w io.Writer) error {
		status := "completed"
		if !report.Passed {
			status = "completed with outstanding issues"
		}
		if _, err := fmt.Fprintf(w, "Resource schema sync %s\n", status); err != nil {
			return err
		}
		for _, path := range report.WrittenPaths {
			if _, err := fmt.Fprintf(w, "- wrote %s\n", path); err != nil {
				return err
			}
		}
		for _, item := range report.MissingReferences {
			if _, err := fmt.Fprintf(w, "- scenario: %s\n  [error] missing resource %s in %s\n", item.Scenario, item.Resource, item.ManifestPath); err != nil {
				return err
			}
		}
		return nil
	})
}
