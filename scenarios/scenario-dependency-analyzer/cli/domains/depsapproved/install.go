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
		Subcommands: documentCommands("deps", []cliapp.Command{
			{
				Name:              "install",
				Description:       "Install a governed dependency into a scenario surface (dry-run by default)",
				DryRunAlternative: "omit --apply; deps commands preview by default",
				Usage:             "scenario-dependency-analyzer deps install <ecosystem>/<package>[@version] (--scenario <name> --surface <surface> | --workspace <absolute-path>) [options]",
				HelpText:          "This command previews by default; --apply requests the install. --workspace targets an existing external pnpm JavaScript workspace with packageManager declared. --exact requires an explicit version. --refresh-lock is limited to closed host-tool source inputs and does not install a runtime. --lock-only (JavaScript, with --apply) publishes the manifest and lockfile without linking node_modules, admitted as a small metadata disk write. Example: scenario-dependency-analyzer deps install npm/three@0.170.0 --scenario notification-hub --surface ui --apply",
				Args:              installArgumentSchema(),
				RunCtx:            func(ctx cliapp.RunContext) error { return runInstall(core, ctx) },
			},
			{Name: "declare-runtime", DryRunAlternative: "omit --apply; deps commands preview by default", Description: "Declare one optional runtime-only scenario dependency with startup_policy=ignore (dry-run by default)", NeedsAPIOverride: &noAPI, Run: func(args []string) error { return runDeclareRuntime(core, args) }},
			{
				Name:              "remove",
				Description:       "Remove JavaScript dependencies from a scenario surface through the governed gateway, or one Go requirement from the root module (dry-run by default)",
				DryRunAlternative: "omit --apply; deps commands preview by default",
				Usage:             "scenario-dependency-analyzer deps remove <ecosystem>/<package>... --scenario <name> --surface <surface> [--apply]",
				HelpText:          "Previews by default and shows what leaves the lockfile; --apply publishes the manifest and lockfile and relinks node_modules from the frozen lock, restoring the previous inputs if relinking fails. The Go root-module form (--scenario vrooli --surface control-plane --ecosystem go --module <path>) edits only go.mod and needs no running API. Example: scenario-dependency-analyzer deps remove npm/three --scenario notification-hub --surface ui --apply",
				Args:              removeArgumentSchema(),
				NeedsAPIOverride:  &noAPI,
				RunCtx:            func(ctx cliapp.RunContext) error { return runRemove(core, ctx) },
			},
			{
				Name:              "update",
				Description:       "Update locked JavaScript packages (direct or transitive) to the newest version their declared ranges allow, for example to take a security fix an in-range release carries (dry-run by default)",
				DryRunAlternative: "omit --apply; deps commands preview by default",
				Usage:             "scenario-dependency-analyzer deps update (<ecosystem>/<package>... | --advisory <GHSA|CVE>) --scenario <name> --surface <surface> [--apply]",
				HelpText:          "Previews by default and lists each locked version that leaves and enters the lockfile; a lockfile that would not change is reported as current. --apply publishes the lockfile and relinks node_modules from the frozen lock, restoring the previous inputs if relinking fails. Only the lockfile changes: no manifest range is edited and no dependency is added, so no governance approval is involved. --advisory selects the packages Security Health reports as exposed to that advisory on the surface (\"all\" selects every exposed package), so a caller needs no package names; an advisory with no exposure there is reported as not_affected. Every response names, as \"exposed: name@version to GHSA... (fixed in ...)\" notes, the selected packages whose locked version still sits inside an advisory after the change, so the caller learns what the declared ranges could not fix. --lock-only publishes the lockfile alone: no download, no package-store write, node_modules untouched, admitted as a small metadata disk write; the next governed setup links it. Example: scenario-dependency-analyzer deps update npm/body-parser npm/proxy-addr --scenario notification-hub --surface ui --apply. Lockfile only: scenario-dependency-analyzer deps update npm/body-parser --scenario notification-hub --surface ui --apply --lock-only. By advisory: scenario-dependency-analyzer deps update --advisory GHSA-27r3-jxmh-p828 --scenario notification-hub --surface ui",
				Args:              updateArgumentSchema(),
				RunCtx:            func(ctx cliapp.RunContext) error { return runUpdate(core, ctx) },
			},
			{Name: "resync", DryRunAlternative: "omit --apply; deps commands preview by default", Description: "Restore a diverged JavaScript lockfile through the governed gateway (dry-run by default)", Run: func(args []string) error { return runResync(core, args) }},
			{Name: "reconcile", DryRunAlternative: "omit --apply; deps commands preview by default", Description: "Converge Go go.mod/go.sum on their import closure through the dependency gate (dry-run by default)", Run: func(args []string) error { return runReconcile(core, args) }},
			{Name: "vendor", DryRunAlternative: "omit --apply; deps commands preview by default", Description: "Synchronise a committed Go vendor tree through the dependency gateway", Run: func(args []string) error { return runVendor(core, args) }},
		}),
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
			{Name: "workspace", Description: "Existing absolute external JavaScript workspace; replaces --scenario and --surface"},
			{Name: "version", Description: "Optional explicit version/range; overrides an inline package version. Also accepts a shared package file:../../../packages/<name> (pnpm) or a plain https tarball URL ending in .tgz"},
			{Name: "apply", Bool: true, Description: "Perform the install; default is a dry-run preview"},
			{Name: "exact", Bool: true, Description: "Persist an exact package version; requires an explicit version (a file: path or tarball URL is exact by construction)"},
			{Name: "dev", Bool: true, Description: "Save a JavaScript dependency as development-only"},
			{Name: "refresh-lock", Bool: true, Description: "Refresh only closed host-tool source dependency inputs; does not install a runtime"},
			{Name: "override-selector", Description: "With --version override:<v>, the exact pnpm override to set, such as brace-expansion@1 or minimatch@3>brace-expansion (its last segment must name the package); default is a package-wide override"},
			{Name: "lock-only", Bool: true, Description: "With --apply on a JavaScript surface, publish package.json and the lockfile without linking node_modules: nothing is downloaded and no package store is written, so the apply is a small metadata disk write and the next governed setup links it"},
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
	workspace := strings.TrimSpace(args.Flag("workspace"))
	if workspace != "" {
		if strings.TrimSpace(scenario) != "" || strings.TrimSpace(surface) != "" {
			return fmt.Errorf("--workspace cannot be combined with --scenario or --surface")
		}
	} else if strings.TrimSpace(scenario) == "" || strings.TrimSpace(surface) == "" {
		return fmt.Errorf("--scenario and --surface, or --workspace, are required")
	}

	// Resolving a dependency closure, sizing it for disk admission and linking it
	// can each take a while on a large surface, and the server finishes the work
	// either way. A short client timeout only turns a slow success into a
	// confusing failure, so every install waits for the owner's three-minute
	// budget plus a bounded allowance for response delivery.
	timeout := 3*time.Minute + 15*time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	resp, err := governanceClientWithTimeout(core, timeout).InstallDependency(ctx, connect.NewRequest(&governancev1.InstallDependencyRequest{
		Scenario:         scenario,
		Surface:          surface,
		Ecosystem:        ecosystem,
		PackageName:      packageName,
		Version:          version,
		Apply:            args.BoolFlag("apply"),
		ExactVersion:     args.BoolFlag("exact"),
		Development:      args.BoolFlag("dev"),
		RefreshLock:      args.BoolFlag("refresh-lock"),
		LockOnly:         args.BoolFlag("lock-only"),
		OverrideSelector: args.Flag("override-selector"),
		WorkspacePath:    workspace,
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
