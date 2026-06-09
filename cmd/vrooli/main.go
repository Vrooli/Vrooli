package main

import (
	"fmt"
	"io"
	"os"

	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/cli/vroolicli"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/floorengagement"
	"github.com/vrooli/vrooli/internal/lifecycle"
	"github.com/vrooli/vrooli/internal/scenarioexec"
	projectsetup "github.com/vrooli/vrooli/internal/setup"
	"github.com/vrooli/vrooli/internal/shell"
)

const (
	vrooliVersion = "2.0.0"
	cliVersion    = "1.0.0"
)

var (
	resolveSourceRootFn = buildinfo.ResolveSourceRoot
	checkStalenessFn    = buildinfo.CheckStaleness
	rebuildAndReexecFn  = buildinfo.RebuildAndReexec
	lookPathFn          = shell.LookPath
	newLoggerFn         = createCommandLogger
)

type globalOptions = rootcli.GlobalOptions

func main() {
	installEngagementResolver(os.Stderr)
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// installEngagementResolver wires the Baseline Modes engagement resolver into
// the lifecycle once, before any Runner is constructed. With it installed, a
// live restart during an open shadow engagement resolves its build/run CWD to
// the frozen restore-point copy rather than the working tree the agent is
// editing (see internal/lifecycle effectiveSourceDir). It is intentionally not
// called from run(), so tests exercising run() stay hermetic. A resolver-
// construction failure is non-fatal (it only happens if the cache root cannot be
// resolved, which would already break far more), but it is surfaced loudly: the
// floor would be unenforced.
func installEngagementResolver(stderr io.Writer) {
	resolver, err := floorengagement.New()
	if err != nil {
		fmt.Fprintf(stderr, "warning: baseline-modes engagement resolver unavailable; live isolation not enforced: %v\n", err)
		return
	}
	lifecycle.SetDefaultEngagementResolver(resolver)
}

func run(args []string, stdout, stderr io.Writer) int {
	return configuredRunner().Run(args, stdout, stderr)
}

func configuredRunner() *rootcli.Runner[*vroolicli.CommandContext] {
	return configuredApp().Runner()
}

func configuredApp() *vroolicli.App {
	return vroolicli.New(vroolicli.Config{
		VersionInfo: vroolicli.VersionInfo{
			CLIVersion:      cliVersion,
			PlatformVersion: vrooliVersion,
		},
		ResolveSourceRootFn: resolveSourceRootFn,
		HomeDirFn:           config.HomeDir,
		CheckStalenessFn:    checkStalenessFn,
		RebuildAndReexecFn:  rebuildAndReexecFn,
		LookPathFn:          lookPathFn,
		NewLoggerFn:         newLoggerFn,
		DebugLogFn:          debugLog,
		RunProjectBuildFn:   projectsetup.RunBuild,
		RunProjectSetupFn:   projectsetup.RunSetupWithOptions,
		RunProjectDevelopFn: projectsetup.RunDevelopWithOptions,
		RunScenarioSubprocess: func(spec scenarioexec.SubprocessSpec) error {
			return scenarioexec.RunSubprocess(spec)
		},
		ScenarioExecutableFn: os.Executable,
	})
}
