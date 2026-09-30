package contractcli

import (
	"fmt"
	"io"
	"strings"

	contractapp "github.com/vrooli/vrooli/internal/app/contract"
	"github.com/vrooli/vrooli/internal/cliout"
)

func RenderValidate(w io.Writer, format cliout.Format, output contractapp.ValidationOutput) error {
	return cliout.RenderJSONOr(w, format, func(w io.Writer) error {
		return cliout.WriteProtoJSON(w, ContractValidationOutputMessage(output))
	}, func(w io.Writer) error {
		status := "passed"
		if !output.Success {
			status = "failed"
		}
		if _, err := fmt.Fprintf(w, "Repo contract validation %s\n", status); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "Root: %s\n", output.Root); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "Schema: %s\n", RenderCheckLine(output.Schema.Passed, output.Schema.Message)); err != nil {
			return err
		}
		for _, check := range output.Report.Checks {
			if _, err := fmt.Fprintf(w, "%s: %s\n", check.Name, RenderCheckLine(check.Passed, check.Message)); err != nil {
				return err
			}
		}
		return nil
	})
}

func RenderShow(w io.Writer, format cliout.Format, output contractapp.ShowOutput) error {
	return cliout.RenderJSONOr(w, format, func(w io.Writer) error {
		return cliout.WriteProtoJSON(w, contractShowOutputMessage(output))
	}, func(w io.Writer) error {
		if _, err := fmt.Fprintln(w, "Repo contract"); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "Root: %s\n", output.Root); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "Contract: %s\n", output.ContractPath); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "Version: %s\n", output.Version); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "Platform mode: %s\n", output.Platform.Mode); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "Markers: dirs=%s files=%s\n", strings.Join(output.Markers.RequiredDirs, ","), strings.Join(output.Markers.RequiredFiles, ",")); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "Layout: scenarios=%s resources=%s packages=%s cmd=%s internal=%s docs=%s\n",
			output.Layout.ScenarioDir, output.Layout.ResourceDir, output.Layout.PackageDir, output.Layout.CommandDir, output.Layout.InternalDir, output.Layout.DocsDir); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "Scenario service path: %s\n", output.Scenario.WellKnownPaths["service"]); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "Resource manifest path: %s\n", output.Resource.Manifest); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "Glob policy: syntax=%s root_relative=%t case_sensitive=%t allow_absolute=%t path_format=%s\n",
			output.Globs.Syntax, output.Globs.RootRelative, output.Globs.CaseSensitive, output.Globs.AllowAbsolute, output.Globs.PathFormat); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "Sandbox scopes: %s prefix=%s\n", strings.Join(output.Sandbox.FullRepoScopes, ","), output.Sandbox.ScenarioScopePrefix); err != nil {
			return err
		}
		profileNames := contractapp.SortedProfileNames(output.Profiles)
		if _, err := fmt.Fprintf(w, "Profiles: %s\n", strings.Join(profileNames, ",")); err != nil {
			return err
		}
		return nil
	})
}

func RenderResolveScenario(w io.Writer, format cliout.Format, output contractapp.ResolveScenarioOutput) error {
	return cliout.RenderJSONOr(w, format, func(w io.Writer) error {
		return cliout.WriteProtoJSON(w, contractResolveScenarioOutputMessage(output))
	}, func(w io.Writer) error { _, err := fmt.Fprintln(w, output.Path); return err })
}

func RenderMatchGlob(w io.Writer, format cliout.Format, output contractapp.MatchGlobOutput) error {
	return cliout.RenderJSONOr(w, format, func(w io.Writer) error {
		return cliout.WriteProtoJSON(w, contractMatchGlobOutputMessage(output))
	}, func(w io.Writer) error {
		if output.Matched {
			_, err := fmt.Fprintln(w, "matched")
			return err
		}
		_, err := fmt.Fprintln(w, "not matched")
		return err
	})
}

func RenderCheckLine(passed bool, message string) string {
	if strings.TrimSpace(message) == "" {
		message = "ok"
	}
	if passed {
		return "PASS (" + message + ")"
	}
	return "FAIL (" + message + ")"
}
