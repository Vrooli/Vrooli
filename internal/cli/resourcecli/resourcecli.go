package resourcecli

import (
	"fmt"
	"io"
	"strings"

	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/control"
	"github.com/vrooli/vrooli/internal/resources"
)

func WriteList(w io.Writer, format cliout.Format, items []resources.Resource) error {
	if format == cliout.FormatJSON {
		return cliout.WriteJSON(w, map[string]any{
			"success":   true,
			"resources": items,
		})
	}
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		decision := ""
		if item.ControlMode == "legacy-adapter" {
			decision = item.LegacyAdapter.FinalDisposition
		}
		rows = append(rows, []string{
			item.Name,
			cliout.BoolLabel(item.Enabled),
			item.ControlMode,
			item.Driver,
			item.PortabilityTier,
			decision,
			cliout.BoolLabel(item.Registered),
		})
	}
	return cliout.RenderTable(w, []string{"Name", "Enabled", "Control", "Driver", "Portability", "Decision", "Registered"}, rows)
}

func WriteStatuses(w io.Writer, format cliout.Format, items []resources.Status) error {
	if format == cliout.FormatJSON {
		return cliout.WriteJSON(w, map[string]any{
			"success":   true,
			"resources": items,
		})
	}
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
	return cliout.RenderTable(w, []string{"Name", "Enabled", "Control", "Running", "Health", "Status"}, rows)
}

func WriteStatus(w io.Writer, format cliout.Format, item resources.Status) error {
	if format == cliout.FormatJSON {
		return cliout.WriteJSON(w, map[string]any{
			"success":   true,
			"name":      item.Resource.Name,
			"installed": item.Installed,
			"running":   item.Running,
			"healthy":   item.Healthy,
			"status":    item.Message,
			"resource":  item,
		})
	}
	rows := [][]string{
		{"Name", item.Resource.Name},
		{"Enabled", cliout.BoolLabel(item.Resource.Enabled)},
		{"Control", item.Resource.ControlMode},
		{"Driver", item.Resource.Driver},
		{"Portability", item.Resource.PortabilityTier},
		{"Installed", cliout.BoolLabel(item.Installed)},
		{"Running", cliout.BoolLabel(item.Running)},
	}
	if item.Healthy != nil {
		rows = append(rows, []string{"Healthy", cliout.BoolLabel(*item.Healthy)})
	}
	if item.StatusCode != "" {
		rows = append(rows, []string{"Status Code", item.StatusCode})
	}
	rows = append(rows, []string{"Status", item.Message})
	if item.Resource.ControlMode == "legacy-adapter" {
		rows = append(rows,
			[]string{"Adapter Owner", item.Resource.LegacyAdapter.Owner},
			[]string{"Decision Deadline", item.Resource.LegacyAdapter.DecisionDeadline},
			[]string{"Final Disposition", item.Resource.LegacyAdapter.FinalDisposition},
			[]string{"Legacy CLI", item.Resource.LegacyAdapter.LegacyCLIPath},
		)
		if item.Resource.LegacyAdapter.Notes != "" {
			rows = append(rows, []string{"Adapter Notes", item.Resource.LegacyAdapter.Notes})
		}
	}
	if item.ProbeError != "" {
		rows = append(rows, []string{"Probe Error", item.ProbeError})
	}
	return cliout.RenderTable(w, []string{"Field", "Value"}, rows)
}

func WriteInfo(w io.Writer, format cliout.Format, item resources.Status) error {
	if format == cliout.FormatJSON {
		return cliout.WriteJSON(w, map[string]any{
			"success":  true,
			"resource": item,
		})
	}
	return WriteStatus(w, format, item)
}

func WriteControlReport(w io.Writer, format cliout.Format, reportKey, verb string, report any, items, failed []control.ResultItem) error {
	if format == cliout.FormatJSON {
		return cliout.WriteJSON(w, map[string]any{
			"success": true,
			reportKey: report,
		})
	}
	for _, item := range items {
		_, _ = fmt.Fprintf(w, "%s %s\n", verb, item.Name)
	}
	for _, item := range failed {
		_, _ = fmt.Fprintf(w, "Failed %s: %s\n", item.Name, item.Error)
	}
	return nil
}

func WriteDeprecationReport(w io.Writer, format cliout.Format, report resources.DeprecationReport) error {
	if format == cliout.FormatJSON {
		return cliout.WriteJSON(w, map[string]any{
			"success": true,
			"report":  report,
		})
	}
	_, _ = fmt.Fprintf(w, "Deprecated %s\n", report.Resource.Name)
	if report.ArchiveDir != "" {
		_, _ = fmt.Fprintf(w, "Archive: %s\n", report.ArchiveDir)
	}
	return nil
}

func WriteRestoreReport(w io.Writer, format cliout.Format, report resources.RestoreReport) error {
	if format == cliout.FormatJSON {
		return cliout.WriteJSON(w, map[string]any{
			"success": true,
			"report":  report,
		})
	}
	_, _ = fmt.Fprintf(w, "Restored %s to %s\n", report.Resource.Name, report.RestoredPath)
	return nil
}

func WriteBlueprintArchiveReport(w io.Writer, format cliout.Format, report resources.BlueprintArchiveReport) error {
	if format == cliout.FormatJSON {
		return cliout.WriteJSON(w, map[string]any{
			"success": true,
			"report":  report,
		})
	}
	_, _ = fmt.Fprintf(w, "Archived %s to blueprint-only state\n", report.Resource.Name)
	if report.ArchiveDir != "" {
		_, _ = fmt.Fprintf(w, "Archive: %s\n", report.ArchiveDir)
	}
	return nil
}

func WriteBlueprintRestoreReport(w io.Writer, format cliout.Format, report resources.BlueprintRestoreReport) error {
	if format == cliout.FormatJSON {
		return cliout.WriteJSON(w, map[string]any{
			"success": true,
			"report":  report,
		})
	}
	_, _ = fmt.Fprintf(w, "Restored blueprint-archived %s to %s\n", report.Resource.Name, report.RestoredPath)
	return nil
}

func WriteArchiveGCReport(w io.Writer, format cliout.Format, report resources.ArchiveGCReport, kind string) error {
	if format == cliout.FormatJSON {
		return cliout.WriteJSON(w, map[string]any{
			"success": true,
			"report":  report,
		})
	}
	_, _ = fmt.Fprintf(w, "Purged %d %s archives\n", len(report.Removed), kind)
	return nil
}

func WriteDeprecatedList(w io.Writer, format cliout.Format, items []resources.DeprecatedResource) error {
	if format == cliout.FormatJSON {
		return cliout.WriteJSON(w, map[string]any{
			"success":   true,
			"resources": items,
		})
	}
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		state := "deprecated"
		if strings.TrimSpace(item.PurgedAt) != "" {
			state = "purged"
		}
		rows = append(rows, []string{item.Name, state, item.DeprecatedAt, item.PurgeAfter, item.Replacement})
	}
	return cliout.RenderTable(w, []string{"Name", "State", "Deprecated", "Purge After", "Replacement"}, rows)
}

func WriteBlueprintArchivedList(w io.Writer, format cliout.Format, items []resources.BlueprintArchivedResource) error {
	if format == cliout.FormatJSON {
		return cliout.WriteJSON(w, map[string]any{
			"success":   true,
			"resources": items,
		})
	}
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		state := "blueprint-archived"
		if strings.TrimSpace(item.PurgedAt) != "" {
			state = "purged"
		}
		rows = append(rows, []string{item.Name, state, item.ArchivedAt, item.PurgeAfter, item.BlueprintName})
	}
	return cliout.RenderTable(w, []string{"Name", "State", "Archived", "Purge After", "Blueprint"}, rows)
}

func WriteBlueprintList(w io.Writer, format cliout.Format, items []resources.Blueprint) error {
	if format == cliout.FormatJSON {
		return cliout.WriteJSON(w, map[string]any{
			"success":    true,
			"blueprints": items,
		})
	}
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{item.Name, item.Category, item.Status, item.SuggestedTemplate, item.LastReviewed})
	}
	return cliout.RenderTable(w, []string{"Name", "Category", "Status", "Template", "Reviewed"}, rows)
}

func WriteBlueprintInfo(w io.Writer, format cliout.Format, item resources.Blueprint) error {
	if format == cliout.FormatJSON {
		return cliout.WriteJSON(w, map[string]any{
			"success":   true,
			"blueprint": item,
		})
	}
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
}

func WriteBlueprintSearch(w io.Writer, format cliout.Format, query string, items []resources.Blueprint) error {
	if format == cliout.FormatJSON {
		return cliout.WriteJSON(w, map[string]any{
			"success":    true,
			"query":      query,
			"blueprints": items,
		})
	}
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{item.Name, item.Category, item.Status, item.Summary})
	}
	return cliout.RenderTable(w, []string{"Name", "Category", "Status", "Summary"}, rows)
}

func WriteBlueprintValidationReport(w io.Writer, format cliout.Format, report resources.BlueprintValidationReport) error {
	if format == cliout.FormatJSON {
		return cliout.WriteJSON(w, map[string]any{
			"success": true,
			"report":  report,
		})
	}
	_, _ = fmt.Fprintf(w, "Validated %d resource blueprints\n", report.Count)
	return nil
}
