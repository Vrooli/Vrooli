package main

import (
	"io"
	"os"

	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/cli/vroolicli"
	"github.com/vrooli/vrooli/internal/config"
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
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
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
