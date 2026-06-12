package planscli

import (
	"fmt"
	"io"
	"path/filepath"

	planapp "github.com/vrooli/vrooli/internal/app/plans"
	"github.com/vrooli/vrooli/internal/cliout"
)

func RenderAdd(w io.Writer, format cliout.Format, resp planapp.AddOutput) error {
	if format == cliout.FormatJSON {
		return writePlansJSON(w, PlansAddResponse(resp))
	}
	printPlanSummary(w, resp.Plan)
	_, _ = fmt.Fprintln(w, "next: vrooli plans show "+resp.Plan.Slug)
	return nil
}

func RenderList(w io.Writer, format cliout.Format, resp planapp.ListOutput) error {
	if format == cliout.FormatJSON {
		return writePlansJSON(w, PlansListResponse(resp))
	}
	if len(resp.Plans) == 0 {
		_, _ = fmt.Fprintln(w, "no plans found")
		return nil
	}
	for _, plan := range resp.Plans {
		archived := ""
		if plan.Archived {
			archived = "\tarchived"
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s%s\n", plan.ID, plan.Title, filepath.ToSlash(plan.Path), archived)
	}
	return nil
}

func RenderShow(w io.Writer, format cliout.Format, resp planapp.ShowOutput) error {
	if format == cliout.FormatJSON {
		return writePlansJSON(w, PlansShowResponse(resp))
	}
	printPlanSummary(w, resp.Plan)
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprint(w, resp.Content)
	return nil
}

func RenderPath(w io.Writer, format cliout.Format, resp planapp.PathOutput) error {
	if format == cliout.FormatJSON {
		return writePlansJSON(w, PlansPathResponse(resp))
	}
	_, _ = fmt.Fprintln(w, filepath.ToSlash(resp.Path))
	return nil
}

func RenderArchive(w io.Writer, format cliout.Format, resp planapp.ArchiveOutput) error {
	if format == cliout.FormatJSON {
		return writePlansJSON(w, PlansArchiveResponse(resp))
	}
	_, _ = fmt.Fprintf(w, "archived %s\n", resp.Plan.ID)
	return nil
}

func RenderImport(w io.Writer, format cliout.Format, resp planapp.ImportOutput) error {
	if format == cliout.FormatJSON {
		return writePlansJSON(w, PlansImportResponse(resp))
	}
	printPlanSummary(w, resp.Plan)
	if resp.Deleted {
		_, _ = fmt.Fprintln(w, "source: deleted")
	}
	return nil
}

func RenderExport(w io.Writer, format cliout.Format, resp planapp.ExportOutput) error {
	if format == cliout.FormatJSON {
		return writePlansJSON(w, PlansExportResponse(resp))
	}
	_, _ = fmt.Fprintf(w, "exported %s to %s\n", resp.ID, filepath.ToSlash(resp.Path))
	return nil
}

func printPlanSummary(w io.Writer, plan planapp.PlanRecord) {
	_, _ = fmt.Fprintf(w, "id: %s\n", plan.ID)
	_, _ = fmt.Fprintf(w, "title: %s\n", plan.Title)
	_, _ = fmt.Fprintf(w, "path: %s\n", filepath.ToSlash(plan.Path))
}
