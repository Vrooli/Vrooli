package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/vrooli/cli-core/cliutil"

	runtimeinfo "github.com/vrooli/resources/sqlite"
	"github.com/vrooli/resources/sqlite/internal/cli"
	"github.com/vrooli/resources/sqlite/internal/config"
)

const (
	appName    = "resource-sqlite"
	appVersion = "0.2.0"
)

var (
	buildFingerprint = "unknown"
	buildTimestamp   = "unknown"
	buildSourceRoot  = ""
)

func main() {
	cfg := config.Load()

	exePath, _ := os.Executable()
	exeDir := filepath.Dir(exePath)
	manifestPath := filepath.Join(exeDir, "resource.json")

	if len(runtimeinfo.ResourceJSON) == 0 {
		fmt.Fprintln(os.Stderr, "embedded resource.json missing")
		os.Exit(1)
	}

	stale := cliutil.NewStaleChecker(appName, buildFingerprint, buildTimestamp, buildSourceRoot,
		"SQLITE_CLI_SOURCE_ROOT", "VROOLI_CLI_SOURCE_ROOT")

	legacy := cli.New(cfg, manifestPath, runtimeinfo.ResourceJSON, buildSourceRoot)
	app := newApp(cfg, legacy, stale)

	if err := app.Run(os.Args[1:]); err != nil {
		if exit, ok := err.(exitError); ok {
			os.Exit(int(exit))
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
