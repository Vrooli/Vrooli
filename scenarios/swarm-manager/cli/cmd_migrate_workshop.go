package main

import (
	"flag"
	"path/filepath"

	"github.com/vrooli/cli-core/cliutil"
	"swarm-manager/cli/cmd"
)

func (a *App) cmdMigrateWorkshop(args []string) error {
	fs := flag.NewFlagSet("migrate-workshop", flag.ContinueOnError)
	root := fs.String("root", "", "Override root directory (default: scenarios/swarm-manager)")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	rootPath := *root
	if rootPath == "" {
		// Try to resolve from the build-time source root, which points at
		// the scenario directory. Fall back to a relative path that works
		// when run from the repo root.
		sr := cliutil.ResolveSourceRoot(buildSourceRoot, "SWARM_MANAGER_SOURCE_ROOT", "VROOLI_CLI_SOURCE_ROOT")
		if sr != "" {
			rootPath = sr
		} else {
			rootPath = filepath.Join("scenarios", "swarm-manager")
		}
	}

	return cmd.RunMigrateWorkshop(cmd.MigrateWorkshopOptions{
		Root:   rootPath,
		DryRun: a.globalDry,
	})
}
