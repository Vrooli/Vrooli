package depsapproved

import (
	"context"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"

	"scenario-dependency-analyzer/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	governancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_governance"
)

// RegisterInstall registers the governed install gateway as `deps install`, the
// sanctioned alternative to a raw `pnpm add`/`go get`/`pip install`. It is a
// separate top-level group from `deps-approved` (the governance-record verbs).
func RegisterInstall(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	noAPI := false
	return cliapp.SubcommandGroup{
		Name:        "deps",
		Description: "Governed third-party dependency install gateway",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{
				Name:        "install",
				Description: "Install a governed dependency into a scenario surface (dry-run by default)",
				Usage:       "scenario-dependency-analyzer deps install <ecosystem>/<package>[@version] --scenario <name> --surface <surface> [options]",
				HelpText:    "This command previews by default; --apply requests the install. --exact requires an explicit version. --refresh-lock is limited to closed host-tool source inputs and does not install a runtime.",
				Args:        installArgumentSchema(),
				RunCtx:      func(ctx cliapp.RunContext) error { return runInstall(core, ctx) },
			},
			{Name: "remove", Description: "Remove one exact Go requirement from the root module (dry-run by default)", NeedsAPIOverride: &noAPI, Run: func(args []string) error { return runRemove(core, args) }},
			{Name: "resync", Description: "Restore a diverged JavaScript lockfile through the governed gateway (dry-run by default)", Run: func(args []string) error { return runResync(core, args) }},
			{Name: "reconcile", Description: "Add missing local replaces for in-repo go.mod modules (dry-run by default)", Run: func(args []string) error { return runReconcile(core, args) }},
			{Name: "vendor", Description: "Synchronise a committed Go vendor tree through the dependency gateway", Run: func(args []string) error { return runVendor(core, args) }},
		},
	}
}

func installArgumentSchema() cliapp.ArgSchema {
	return cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{
			Name:        "dependency",
			Required:    true,
			Description: "Dependency spec: <ecosystem>/<package>[@version]",
		}},
		Flags: []cliapp.Flag{
			{Name: "scenario", Description: "Required target scenario"},
			{Name: "surface", Description: "Required target surface: ui, api, cli, agent, playwright-driver, resource, host-tool, tools/<package>, platforms/<package>, template/<id>, or control-plane (vrooli Go only)"},
			{Name: "version", Description: "Optional explicit version/range; overrides an inline package version"},
			{Name: "apply", Bool: true, Description: "Perform the install; default is a dry-run preview"},
			{Name: "exact", Bool: true, Description: "Persist an exact package version; requires an explicit version"},
			{Name: "dev", Bool: true, Description: "Save a JavaScript dependency as development-only"},
			{Name: "refresh-lock", Bool: true, Description: "Refresh only closed host-tool source dependency inputs; does not install a runtime"},
		},
	}
}

func runInstall(core *cliapp.ScenarioApp, args cliapp.RunContext) error {
	ecosystem, packageName, version, err := parseInstallDependencySpec(args.Positional("dependency"), args.Flag("version"))
	if err != nil {
		return err
	}
	scenario := args.Flag("scenario")
	surface := args.Flag("surface")
	if strings.TrimSpace(scenario) == "" || strings.TrimSpace(surface) == "" {
		return fmt.Errorf("--scenario and --surface are required")
	}

	timeout := 45 * time.Second
	if surface == "host-tool" || surface == "resource" {
		// Closed host operations and resource Python locks have a three-minute owner budget. Leave a
		// bounded allowance for governance/resolution and response delivery.
		timeout = 3*time.Minute + 15*time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	resp, err := governanceClientWithTimeout(core, timeout).InstallDependency(ctx, connect.NewRequest(&governancev1.InstallDependencyRequest{
		Scenario:     scenario,
		Surface:      surface,
		Ecosystem:    ecosystem,
		PackageName:  packageName,
		Version:      version,
		Apply:        args.BoolFlag("apply"),
		ExactVersion: args.BoolFlag("exact"),
		Development:  args.BoolFlag("dev"),
		RefreshLock:  args.BoolFlag("refresh-lock"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("install dependency", err, nil)
	}
	if args.JSON() {
		if err := printProto(resp.Msg); err != nil {
			return err
		}
		return installOutcome(resp.Msg)
	}
	return printInstallResult(resp.Msg)
}

func parseInstallDependencySpec(spec, explicitVersion string) (ecosystem, packageName, version string, err error) {
	ecosystem, packageName, ok := strings.Cut(strings.TrimSpace(spec), "/")
	ecosystem = strings.TrimSpace(ecosystem)
	packageName = strings.TrimSpace(packageName)
	if !ok || ecosystem == "" || packageName == "" {
		return "", "", "", fmt.Errorf("dependency must be formatted as <ecosystem>/<package>[@version]")
	}

	inlineVersion := ""
	if versionIndex := strings.LastIndex(packageName, "@"); versionIndex > 0 {
		inlineVersion = strings.TrimSpace(packageName[versionIndex+1:])
		packageName = strings.TrimSpace(packageName[:versionIndex])
	}
	if packageName == "" {
		return "", "", "", fmt.Errorf("dependency must be formatted as <ecosystem>/<package>[@version]")
	}

	version = strings.TrimSpace(explicitVersion)
	if version == "" {
		version = inlineVersion
	}
	return ecosystem, packageName, version, nil
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
	// A refusal or failed applied install must be a non-zero outcome. A
	// non-blocked transport response does not establish installation success.
	if err := support.PrintList(false, report, nil); err != nil {
		return err
	}
	return installOutcome(msg)
}

func installOutcome(msg *governancev1.InstallDependencyResponse) error {
	if !msg.GetBlocked() && (msg.GetDryRun() || msg.GetInstalled() || msg.GetSourceUpdated()) {
		return nil
	}
	return fmt.Errorf("dependency installation did not succeed (verdict=%s, blocked=%t, installed=%t)", msg.GetVerdict(), msg.GetBlocked(), msg.GetInstalled())
}
