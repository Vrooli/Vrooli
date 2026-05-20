package shareddriftcli

import (
	"fmt"
	"io"
	"strconv"

	shareddrift "github.com/vrooli/vrooli/internal/app/shareddrift"
	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/cliout"
)

type Request struct {
	Fix         bool
	OnlyTouched bool
	Build       bool
	Concurrency int
}

func CommandSpec() commandtree.Spec[string] {
	return commandtree.Spec[string]{
		Name:    "check-shared-drift",
		Summary: "Check dependent scenarios for stale shared-package state",
		Group:   "Maintenance Commands",
		Args: commandtree.ArgSchema{
			Options: []commandtree.OptionArg{
				{Name: "--fix", Description: "Apply go mod tidy to stale scenarios (still exits non-zero so files can be staged)"},
				{Name: "--only-touched", Description: "Limit to scenarios depending on staged or unstaged shared-package changes"},
				{Name: "--build", Description: "Also run go build ./... in each scenario after the module check"},
				{Name: "--concurrency", ValueName: "N", Description: "Maximum parallel scenario checks (default 8)"},
				commandtree.JSONOption(),
			},
		},
		Handler: "check-shared-drift",
	}
}

func ParseRequest(args []string) (Request, error) {
	spec := CommandSpec()
	parsed, err := commandtree.ParseArgs("check-shared-drift", commandtree.SpecHelpText("", "vrooli check-shared-drift", spec), spec.Args, args)
	if err != nil {
		return Request{}, err
	}
	req := Request{
		Fix:         parsed.HasFlag("--fix"),
		OnlyTouched: parsed.HasFlag("--only-touched"),
		Build:       parsed.HasFlag("--build"),
	}
	if raw := parsed.FlagValue("--concurrency"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n <= 0 {
			return Request{}, fmt.Errorf("--concurrency must be a positive integer")
		}
		req.Concurrency = n
	}
	return req, nil
}

func Render(w io.Writer, format cliout.Format, report shareddrift.Report) error {
	if format == cliout.FormatJSON {
		return cliout.WriteJSON(w, report)
	}
	status := "clean"
	if !report.Clean {
		status = "drift detected"
	}
	_, _ = fmt.Fprintf(w, "Status: shared-package drift %s\n", status)
	_, _ = fmt.Fprintf(w, "Root: %s\n", report.Root)
	if report.OnlyTouchedUsed {
		if len(report.TouchedPackages) == 0 {
			_, _ = fmt.Fprintln(w, "Touched shared paths: (none — no scenarios checked)")
		} else {
			_, _ = fmt.Fprintf(w, "Touched paths: %d\n", len(report.TouchedPackages))
		}
	}
	_, _ = fmt.Fprintf(w, "Scenarios checked: %d\n", len(report.Scenarios))
	_, _ = fmt.Fprintf(w, "Elapsed: %dms\n", report.ElapsedMs)

	var stale, errored []shareddrift.ScenarioReport
	for _, sc := range report.Scenarios {
		switch sc.Status {
		case shareddrift.StatusStaleModules, shareddrift.StatusStaleBuild:
			stale = append(stale, sc)
		case shareddrift.StatusError:
			errored = append(errored, sc)
		}
	}

	if len(stale) > 0 {
		_, _ = fmt.Fprintln(w, "\nStale scenarios:")
		for i, sc := range stale {
			_, _ = fmt.Fprintf(w, "%d. %s [%s]\n", i+1, sc.Path, sc.Status)
			if len(sc.DiffPaths) > 0 {
				_, _ = fmt.Fprintf(w, "   Changed: %v\n", sc.DiffPaths)
			}
			if sc.BuildError != "" {
				_, _ = fmt.Fprintf(w, "   Build error: %s\n", firstLine(sc.BuildError))
			}
		}
	}

	if len(errored) > 0 {
		_, _ = fmt.Fprintln(w, "\nScenarios with check errors:")
		for i, sc := range errored {
			_, _ = fmt.Fprintf(w, "%d. %s — %s\n", i+1, sc.Path, sc.Error)
		}
	}

	if !report.Clean {
		_, _ = fmt.Fprintln(w, "\nNext steps:")
		if report.FixApplied {
			_, _ = fmt.Fprintln(w, "- Stage the tidied go.mod/go.sum files and recommit.")
			_, _ = fmt.Fprintln(w, "  git add scenarios/*/api/go.mod scenarios/*/api/go.sum")
		} else {
			_, _ = fmt.Fprintln(w, "- Apply fixes: vrooli check-shared-drift --fix --only-touched")
		}
	}
	return nil
}

func firstLine(s string) string {
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			return s[:i]
		}
	}
	return s
}
