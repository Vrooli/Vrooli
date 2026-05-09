package hygienecli

import (
	"fmt"
	"io"

	hygieneapp "github.com/vrooli/vrooli/internal/app/hygiene"
	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/cliout"
)

type Request struct {
	FixSafe bool
	Plans   bool
	FailOn  hygieneapp.Severity
}

func CommandSpec() commandtree.Spec[string] {
	return commandtree.Spec[string]{
		Name:    "hygiene",
		Summary: "Run repository hygiene checks",
		Group:   "Maintenance Commands",
		Args: commandtree.ArgSchema{
			Options: []commandtree.OptionArg{
				{Name: "--fix-safe", Description: "Apply safe, non-destructive fixes"},
				{Name: "--plans", Description: "Include safe plan candidate imports when fixing"},
				{Name: "--fail-on", ValueName: "severity", Description: "Exit non-zero on warning or error"},
				commandtree.JSONOption(),
			},
		},
		Handler: "hygiene",
	}
}

func ParseRequest(args []string) (Request, error) {
	spec := CommandSpec()
	parsed, err := commandtree.ParseArgs("hygiene", commandtree.SpecHelpText("", "vrooli hygiene", spec), spec.Args, args)
	if err != nil {
		return Request{}, err
	}
	failOn := hygieneapp.Severity(parsed.FlagValue("--fail-on"))
	switch failOn {
	case "", hygieneapp.SeverityError, hygieneapp.SeverityWarning:
	default:
		return Request{}, fmt.Errorf("--fail-on must be warning or error")
	}
	return Request{
		FixSafe: parsed.HasFlag("--fix-safe"),
		Plans:   parsed.HasFlag("--plans"),
		FailOn:  failOn,
	}, nil
}

func Render(w io.Writer, format cliout.Format, report hygieneapp.Report) error {
	if format == cliout.FormatJSON {
		return cliout.WriteJSON(w, report)
	}
	status := "passed"
	if !report.Success {
		status = "failed"
	}
	_, _ = fmt.Fprintf(w, "Status: hygiene %s\n", status)
	_, _ = fmt.Fprintf(w, "Root: %s\n", report.Root)
	_, _ = fmt.Fprintf(w, "Blocking issues: %d\n", report.BlockingFailures)
	_, _ = fmt.Fprintf(w, "Warnings: %d\n", report.Warnings)
	if len(report.Checks) > 0 {
		_, _ = fmt.Fprintln(w, "\nChecks:")
		for _, check := range report.Checks {
			state := "ok"
			if !check.Passed {
				state = "failed"
			}
			_, _ = fmt.Fprintf(w, "- %s: %s (%s)\n", check.Name, state, check.Message)
		}
	}
	if len(report.PlanCandidates) > 0 {
		_, _ = fmt.Fprintln(w, "\nPlan candidates:")
		for _, candidate := range report.PlanCandidates {
			status := candidate.Status
			if status == "" {
				status = "unknown"
			}
			_, _ = fmt.Fprintf(w, "- %s [%s]\n", candidate.Path, status)
		}
	}
	if len(report.FixesApplied) > 0 {
		_, _ = fmt.Fprintln(w, "\nFixes applied:")
		for _, fix := range report.FixesApplied {
			_, _ = fmt.Fprintf(w, "- imported %s -> %s\n", fix.Source, fix.Plan.Path)
		}
	}
	if len(report.Actions) > 0 {
		_, _ = fmt.Fprintln(w, "\nNext steps:")
		for _, action := range report.Actions {
			if action.Command != "" {
				_, _ = fmt.Fprintf(w, "- %s\n", action.Command)
			} else {
				_, _ = fmt.Fprintf(w, "- %s\n", action.Message)
			}
		}
	}
	return nil
}
