package packagecli

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/packagegov"
)

func RenderList(w io.Writer, format cliout.Format, resp ListResponse) error {
	if format == cliout.FormatJSON {
		return writePackageJSON(w, PackageListResponse(resp))
	}
	for _, item := range resp.Packages {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", item.Name, item.Manifest.Package.Kind, filepath.ToSlash(item.RootPath))
	}
	return nil
}

func RenderInfo(w io.Writer, format cliout.Format, resp packagegov.Package) error {
	if format == cliout.FormatJSON {
		return writePackageJSON(w, PackageInfoResponse(resp))
	}
	_, _ = fmt.Fprintf(w, "name: %s\n", resp.Name)
	_, _ = fmt.Fprintf(w, "kind: %s\n", resp.Manifest.Package.Kind)
	_, _ = fmt.Fprintf(w, "root: %s\n", filepath.ToSlash(resp.RootPath))
	_, _ = fmt.Fprintf(w, "display: %s\n", resp.Manifest.Package.DisplayName)
	_, _ = fmt.Fprintf(w, "adoptable: %t\n", resp.Manifest.Package.Adoption.ScenarioAdoptable)
	return nil
}

func RenderDependents(w io.Writer, format cliout.Format, resp DependentsResponse) error {
	if format == cliout.FormatJSON {
		return writePackageJSON(w, PackageDependentsResponse(resp))
	}
	for _, dep := range resp.Dependents {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", dep.ConsumerName, dep.ConsumerClass, dep.AdoptionMode, filepath.ToSlash(dep.DependencyFile))
	}
	if len(resp.Issues) > 0 {
		_, _ = fmt.Fprintln(w)
		for _, issue := range resp.Issues {
			_, _ = fmt.Fprintf(w, "%s: %s (%s)\n", issue.Severity, issue.Message, filepath.ToSlash(issue.Path))
		}
	}
	return nil
}

func RenderValidate(w io.Writer, format cliout.Format, resp ValidateResponse) error {
	if format == cliout.FormatJSON {
		return writePackageJSON(w, PackageValidateResponse(resp))
	}
	if len(resp.Report.Issues) == 0 {
		_, _ = fmt.Fprintln(w, "package governance validation passed")
		return nil
	}
	for _, issue := range resp.Report.Issues {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", issue.Severity, issue.Code, filepath.ToSlash(issue.Path), issue.Message)
	}
	return nil
}

func RenderRun(w io.Writer, format cliout.Format, resp RunResponse) error {
	if format == cliout.FormatJSON {
		return writePackageJSON(w, PackageRunResponse(resp))
	}
	_, _ = fmt.Fprintf(w, "%s %s completed\n", resp.Action, resp.PackageName)
	return nil
}

func RenderRefresh(w io.Writer, format cliout.Format, resp RefreshResponse) error {
	if format == cliout.FormatJSON {
		return writePackageJSON(w, PackageRefreshResponse(resp))
	}
	if len(resp.Items) == 0 {
		_, _ = fmt.Fprintf(w, "refreshed %s with no affected governed consumers\n", resp.PackageName)
		return nil
	}
	for _, item := range resp.Items {
		classText := string(item.Class)
		if len(item.Classes) > 1 {
			parts := make([]string, 0, len(item.Classes))
			for _, class := range item.Classes {
				parts = append(parts, string(class))
			}
			classText = strings.Join(parts, ",")
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", item.Consumer, classText, item.Action, item.Status)
	}
	return nil
}

func RenderAudit(w io.Writer, format cliout.Format, resp AuditResponse) error {
	if format == cliout.FormatJSON {
		return writePackageJSON(w, PackageAuditResponse(resp))
	}
	if len(resp.Report.Issues) == 0 {
		_, _ = fmt.Fprintln(w, "package governance audit passed")
		return nil
	}
	for _, issue := range resp.Report.Issues {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", issue.Severity, issue.Code, filepath.ToSlash(issue.Path), issue.Message)
	}
	return nil
}
