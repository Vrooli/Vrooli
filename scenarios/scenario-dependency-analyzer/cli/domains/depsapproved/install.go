package depsapproved

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"

	"scenario-dependency-analyzer/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	governancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_governance"
)

// RegisterInstall registers the governed install gateway as `deps install`, the
// sanctioned alternative to a raw `pnpm add`/`go get`/`pip install`. It is a
// separate top-level group from `deps-approved` (the governance-record verbs).
func RegisterInstall(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "deps",
		Description: "Governed third-party dependency install gateway",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "install", Description: "Install a governed dependency into a scenario surface (dry-run by default)", Run: func(args []string) error { return runInstall(core, args) }},
		},
	}
}

func runInstall(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("deps install")
	var scenario, surface, version string
	var apply, jsonOutput bool
	fs.StringVar(&scenario, "scenario", "", "Target scenario")
	fs.StringVar(&surface, "surface", "", "Target surface: ui, api, or cli")
	fs.StringVar(&version, "version", "", "Optional explicit version/range")
	fs.BoolVar(&apply, "apply", false, "Perform the install (default is a dry run)")
	fs.BoolVar(&jsonOutput, "json", false, "Output raw JSON")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	positionals := fs.Args()
	if len(positionals) != 1 {
		return fmt.Errorf("usage: %s deps install <ecosystem>/<package>[@version] --scenario <name> --surface <ui|api|cli> [--apply] [--json]", support.AppName)
	}
	ecosystem, rest, ok := strings.Cut(positionals[0], "/")
	if !ok || strings.TrimSpace(ecosystem) == "" || strings.TrimSpace(rest) == "" {
		return fmt.Errorf("dependency must be formatted as <ecosystem>/<package>[@version]")
	}
	packageName := rest
	if pkg, ver, hasVer := strings.Cut(rest, "@"); hasVer {
		packageName = pkg
		if strings.TrimSpace(version) == "" {
			version = ver
		}
	}
	if strings.TrimSpace(scenario) == "" || strings.TrimSpace(surface) == "" {
		return fmt.Errorf("--scenario and --surface are required")
	}

	resp, err := governanceClient(core).InstallDependency(context.Background(), connect.NewRequest(&governancev1.InstallDependencyRequest{
		Scenario:    scenario,
		Surface:     surface,
		Ecosystem:   ecosystem,
		PackageName: packageName,
		Version:     version,
		Apply:       apply,
	}))
	if err != nil {
		return cliapp.WrapAPIError("install dependency", err, nil)
	}
	if jsonOutput {
		return printProto(resp.Msg)
	}
	return printInstallResult(resp.Msg)
}

func printInstallResult(msg *governancev1.InstallDependencyResponse) error {
	summary := []string{
		msg.GetMessage(),
		fmt.Sprintf("Verdict: %s", msg.GetVerdict()),
		fmt.Sprintf("Command: %s", msg.GetCommand()),
	}
	if msg.GetManifestPath() != "" {
		summary = append(summary, fmt.Sprintf("Manifest: %s", msg.GetManifestPath()))
	}
	if notes := msg.GetSecurityNotes(); len(notes) > 0 {
		summary = append(summary, "Security: "+strings.Join(notes, "; "))
	}
	report := cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Dependency Install",
		Results:        msg.GetNextSteps(),
		RetrievalHints: []string{msg.GetGuidance()},
	}
	// A blocked install is a non-zero outcome the caller should notice.
	return support.PrintList(!msg.GetBlocked(), report, nil)
}
