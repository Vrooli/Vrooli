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
	return cliapp.SubcommandGroup{
		Name:        "deps",
		Description: "Governed third-party dependency install gateway",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "install", Description: "Install a governed dependency into a scenario surface (dry-run by default)", Run: func(args []string) error { return runInstall(core, args) }},
			{Name: "resync", Description: "Restore a diverged JavaScript lockfile through the governed gateway (dry-run by default)", Run: func(args []string) error { return runResync(core, args) }},
			{Name: "reconcile", Description: "Add missing local replaces for in-repo go.mod modules (dry-run by default)", Run: func(args []string) error { return runReconcile(core, args) }},
			{Name: "vendor", Description: "Synchronise a committed Go vendor tree through the dependency gateway", Run: func(args []string) error { return runVendor(core, args) }},
		},
	}
}

func runInstall(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("deps install")
	var scenario, surface, version string
	var apply, jsonOutput, exactVersion, development, refreshLock bool
	fs.StringVar(&scenario, "scenario", "", "Target scenario")
	fs.StringVar(&surface, "surface", "", "Target surface: ui, api, cli, agent, playwright-driver, resource, host-tool, tools/<package>, platforms/<package>, or template/<id>")
	fs.StringVar(&version, "version", "", "Optional explicit version/range")
	fs.BoolVar(&apply, "apply", false, "Perform the install (default is a dry run)")
	fs.BoolVar(&exactVersion, "exact", false, "Persist an exact package version (requires an explicit version)")
	fs.BoolVar(&development, "dev", false, "Save a JavaScript dependency as development-only")
	fs.BoolVar(&refreshLock, "refresh-lock", false, "Refresh only the closed host-tool source dependency inputs; does not install a runtime")
	fs.BoolVar(&jsonOutput, "json", false, "Output raw JSON")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	positionals := fs.Args()
	if len(positionals) != 1 {
		return fmt.Errorf("usage: %s deps install <ecosystem>/<package>[@version] --scenario <name> --surface <ui|api|cli|agent|playwright-driver|resource|host-tool|tools/<package>|platforms/<package>|template/<id>> [--exact] [--dev] [--refresh-lock] [--apply] [--json]", support.AppName)
	}
	ecosystem, packageName, version, err := parseInstallDependencySpec(positionals[0], version)
	if err != nil {
		return err
	}
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
		Apply:        apply,
		ExactVersion: exactVersion,
		Development:  development,
		RefreshLock:  refreshLock,
	}))
	if err != nil {
		return cliapp.WrapAPIError("install dependency", err, nil)
	}
	if jsonOutput {
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
