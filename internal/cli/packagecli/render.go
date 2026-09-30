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
	return cliout.RenderJSONOr(w, format,
		func(w io.Writer) error { return cliout.WriteProtoJSON(w, PackageListResponse(resp)) },
		func(w io.Writer) error {
			for _, item := range resp.Packages {
				_, _ = fmt.Fprintln(w, strings.Join([]string{item.Name, string(item.Manifest.Package.Kind), filepath.ToSlash(item.RootPath)}, "\t"))
			}
			return nil
		})
}

func RenderInfo(w io.Writer, format cliout.Format, resp packagegov.Package) error {
	return cliout.RenderJSONOr(w, format,
		func(w io.Writer) error { return cliout.WriteProtoJSON(w, PackageInfoResponse(resp)) },
		func(w io.Writer) error {
			_, _ = fmt.Fprintf(w, "name: %s\n", resp.Name)
			_, _ = fmt.Fprintf(w, "kind: %s\n", resp.Manifest.Package.Kind)
			_, _ = fmt.Fprintf(w, "root: %s\n", filepath.ToSlash(resp.RootPath))
			_, _ = fmt.Fprintf(w, "display: %s\n", resp.Manifest.Package.DisplayName)
			_, _ = fmt.Fprintf(w, "adoptable: %t\n", resp.Manifest.Package.Adoption.ScenarioAdoptable)
			return nil
		})
}

func RenderDependents(w io.Writer, format cliout.Format, resp DependentsResponse) error {
	return cliout.RenderJSONOr(w, format,
		func(w io.Writer) error { return cliout.WriteProtoJSON(w, PackageDependentsResponse(resp)) },
		func(w io.Writer) error {
			for _, dep := range resp.Dependents {
				_, _ = fmt.Fprintln(w, strings.Join([]string{dep.ConsumerName, string(dep.ConsumerClass), string(dep.AdoptionMode), filepath.ToSlash(dep.DependencyFile)}, "\t"))
			}
			if len(resp.Issues) > 0 {
				_, _ = fmt.Fprintln(w)
				for _, issue := range resp.Issues {
					_, _ = fmt.Fprintf(w, "%s: %s (%s)\n", issue.Severity, issue.Message, filepath.ToSlash(issue.Path))
				}
			}
			return nil
		})
}

func RenderRun(w io.Writer, format cliout.Format, resp RunResponse) error {
	return cliout.RenderJSONOr(w, format,
		func(w io.Writer) error { return cliout.WriteProtoJSON(w, PackageRunResponse(resp)) },
		func(w io.Writer) error {
			_, _ = fmt.Fprintf(w, "%s %s completed\n", resp.Action, resp.PackageName)
			return nil
		})
}

func RenderRefresh(w io.Writer, format cliout.Format, resp RefreshResponse) error {
	return cliout.RenderJSONOr(w, format,
		func(w io.Writer) error { return cliout.WriteProtoJSON(w, PackageRefreshResponse(resp)) },
		func(w io.Writer) error {
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
				_, _ = fmt.Fprintln(w, strings.Join([]string{item.Consumer, classText, string(item.Action), item.Status}, "\t"))
			}
			return nil
		})
}
