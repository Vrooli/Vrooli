package scenariocli

import (
	"context"
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/vrooli/vrooli/internal/cli/clipolicy"
	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/lifecycle"
)

// CommandDeployReport lists what a restart would deploy: the source changes
// since each running build, and the environment switches they add.
const CommandDeployReport CommandID = "deploy-report"

// DeployReporter is the lifecycle capability behind `scenario deploy-report`.
// It is optional on the phase runner so existing runner fakes stay valid.
type DeployReporter interface {
	DeployReportByName(ctx context.Context, name, customPath string) (lifecycle.DeployReport, error)
}

// DeployReportRequest is the parsed `vrooli scenario deploy-report` invocation.
type DeployReportRequest struct {
	Name string
	Path string
	JSON bool
}

// DeployReportResponse wraps the lifecycle report for rendering.
type DeployReportResponse struct {
	Report lifecycle.DeployReport
}

func deployReportSpec() commandtree.Spec[CommandID] {
	return commandtree.Spec[CommandID]{
		Name: string(CommandDeployReport), Group: "Read-only Commands", Summary: "List source changes and new env switches since the running build (pre-restart check)", Handler: CommandDeployReport, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot},
		Args: commandtree.ArgSchema{
			Positionals: []commandtree.PositionalArg{{Name: "scenario name", Required: true}},
			Options:     []commandtree.OptionArg{commandtree.JSONOption(), {Name: "--path", ValueName: "path"}},
		},
	}
}

func ParseDeployReportRequest(globalsJSON bool, args []string) (DeployReportRequest, error) {
	spec := commandSpec(CommandDeployReport)
	parsed, err := commandtree.ParseArgs("scenario deploy-report", commandHelpText(CommandDeployReport), spec.Args, args)
	if err != nil {
		return DeployReportRequest{}, err
	}
	if len(parsed.Positionals) == 0 {
		return DeployReportRequest{}, clipolicy.UsageErrorf("scenario deploy-report", "scenario deploy-report requires a scenario name")
	}
	return DeployReportRequest{
		Name: parsed.Positionals[0],
		Path: parsed.FlagValue("--path"),
		JSON: globalsJSON || parsed.HasFlag("--json"),
	}, nil
}

// restartDeployReportFlag asks restart to print the deploy report before it
// restarts. It is restart-only, so it is taken out before the shared start
// parser sees the arguments.
const restartDeployReportFlag = "--deploy-report"

func stripRestartDeployReportFlag(args []string) ([]string, bool) {
	out := make([]string, 0, len(args))
	found := false
	for _, arg := range args {
		if arg == restartDeployReportFlag {
			found = true
			continue
		}
		out = append(out, arg)
	}
	return out, found
}

// RenderDeployReportResponse prints the report. JSON emits the lifecycle
// structure; human output lists changed directories (scenario-owned first,
// then shared) with their files, then the switch verdict.
func RenderDeployReportResponse(w io.Writer, format cliout.Format, resp DeployReportResponse) error {
	return cliout.RenderJSONOr(w, format, func(w io.Writer) error { return cliout.WriteJSON(w, resp.Report) }, func(w io.Writer) error {
		return WriteDeployReport(w, resp.Report)
	})
}

// WriteDeployReport writes the human form of a deploy report.
func WriteDeployReport(w io.Writer, report lifecycle.DeployReport) error {
	var b strings.Builder
	fmt.Fprintf(&b, "Deploy report for %s: ", report.Scenario)
	if report.Changed {
		b.WriteString("a restart deploys the changes below\n")
	} else {
		// The baseline is the last recorded build, which a stopped or
		// mid-restart scenario may not be running; each artifact says so.
		b.WriteString("no changes since the recorded builds\n")
	}
	for _, art := range report.Artifacts {
		fmt.Fprintf(&b, "\n%s [%s] %s\n", art.Component, art.CheckType, art.Target)
		if art.Baseline != "recorded" {
			b.WriteString("  No recorded build manifest; changes are unknown until the next build.\n")
			continue
		}
		if art.BuiltAt != nil {
			fmt.Fprintf(&b, "  Built %s", cliout.FormatTimestamp(*art.BuiltAt))
			if art.Running != lifecycle.DeployRunningNotApplicable {
				fmt.Fprintf(&b, "; running: %s", deployRunningLabel(art.Running))
			}
			b.WriteString("\n")
		}
		for _, key := range art.KeyChanges {
			fmt.Fprintf(&b, "  Build key %s: %q -> %q\n", key.Name, key.Recorded, key.Current)
		}
		if len(art.Files) == 0 {
			b.WriteString("  No source changes since the build.\n")
		} else {
			fmt.Fprintf(&b, "  %d changed files in %d directories:\n", len(art.Files), len(art.Packages))
			for _, pkg := range art.Packages {
				shared := ""
				if pkg.Shared {
					shared = " [shared]"
				}
				fmt.Fprintf(&b, "    %s%s: %s\n", pkg.Dir, shared, deployPackageFiles(pkg.Dir, art.Files))
			}
		}
		if !art.SwitchScan {
			continue
		}
		switch {
		case !art.SwitchesKnown:
			b.WriteString("  Env switches: unknown for the current build; known after the next build.\n")
			if len(art.SwitchesInChangedFiles) > 0 {
				b.WriteString("    Switch names in changed files:\n")
				writeDeploySwitches(&b, art.SwitchesInChangedFiles)
			}
		case len(art.NewSwitches) == 0 && len(art.RemovedSwitches) == 0:
			b.WriteString("  Env switches: none added or removed.\n")
		default:
			if len(art.NewSwitches) > 0 {
				b.WriteString("  New env switches:\n")
				writeDeploySwitches(&b, art.NewSwitches)
			}
			if len(art.RemovedSwitches) > 0 {
				fmt.Fprintf(&b, "  Removed env switches: %s\n", strings.Join(art.RemovedSwitches, ", "))
			}
		}
	}
	_, err := io.WriteString(w, b.String())
	return err
}

func deployRunningLabel(state string) string {
	switch state {
	case lifecycle.DeployRunningCurrent:
		return "this build"
	case lifecycle.DeployRunningReplaced:
		return "an OLDER binary that was replaced on disk; changes between it and this build are not listed"
	case lifecycle.DeployRunningNone:
		return "no process"
	default:
		return "unknown on this host"
	}
}

func deployPackageFiles(dir string, files []lifecycle.DeployReportFile) string {
	var parts []string
	for _, file := range files {
		if path.Dir(file.Path) == dir {
			parts = append(parts, fmt.Sprintf("%s (%s)", path.Base(file.Path), file.Change))
		}
	}
	return strings.Join(parts, ", ")
}

func writeDeploySwitches(b *strings.Builder, switches []lifecycle.DeployReportSwitch) {
	for _, sw := range switches {
		fmt.Fprintf(b, "    %s  %s\n", sw.Name, strings.Join(sw.Files, ", "))
	}
}
