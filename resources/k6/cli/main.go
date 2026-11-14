package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/vrooli/packages/resource-agent-install"
)

const (
	appName    = "k6"
	appVersion = "0.1.0"
)

var (
	buildFingerprint = "unknown"
	buildTimestamp   = "unknown"
	buildSourceRoot  = ""
)

func main() {
	app, err := newApp()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if err := app.CLI.Run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func newApp() (*cliapp.ResourceApp, error) {
	env := cliapp.StandardResourceEnv(appName, cliapp.ResourceEnvOptions{})
	app, err := cliapp.NewResourceApp(cliapp.ResourceOptions{
		Name:                appName,
		Version:             appVersion,
		Description:         "K6 resource CLI",
		SourceRootEnvVars:   env.SourceRootEnvVars,
		ControlPlaneEnvVars: env.ControlPlaneEnvVars,
		BuildFingerprint:    buildFingerprint,
		BuildTimestamp:      buildTimestamp,
		BuildSourceRoot:     buildSourceRoot,
	})
	if err != nil {
		return nil, err
	}
	install := agentinstall.DirectInstallCommand(agentinstall.Spec{
		Binary:  "k6",
		BinDir:  filepath.Join(os.Getenv("HOME"), ".local", "bin"),
		Version: "0.49.0",
		URLTemplates: map[string]string{
			"linux":   "https://github.com/grafana/k6/releases/download/v${version}/k6-v${version}-linux-${arch}.tar.gz",
			"darwin":  "https://github.com/grafana/k6/releases/download/v${version}/k6-v${version}-macos-${arch}.tar.gz",
			"windows": "https://github.com/grafana/k6/releases/download/v${version}/k6-v${version}-windows-${arch}.zip",
		},
		ArchiveEntry: "k6",
	})
	app.SetCommands(append(app.StandardLifecycleCommands(), cliapp.CommandGroup{Title: "Installation", Commands: []cliapp.Command{install}}))
	return app, nil
}
