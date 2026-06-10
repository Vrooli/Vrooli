package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/vrooli/cli-core/cliapp"

	resourceenv "resource-searxng/cli/internal/env"
	"resource-searxng/cli/internal/health"
)

const (
	appName    = "searxng"
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
		Description:         "SearXNG resource CLI",
		SourceRootEnvVars:   env.SourceRootEnvVars,
		ControlPlaneEnvVars: env.ControlPlaneEnvVars,
		BuildFingerprint:    buildFingerprint,
		BuildTimestamp:      buildTimestamp,
		BuildSourceRoot:     buildSourceRoot,
	})
	if err != nil {
		return nil, err
	}
	commands := append(app.StandardLifecycleCommands(), cliapp.CommandGroup{
		Title: "Diagnostics",
		Commands: []cliapp.Command{
			{
				Name:        "engine-health",
				Description: "Probe engine-level health (canary query + /stats/errors)",
				Run: func(args []string) error {
					return runEngineHealth(args, os.Stdout)
				},
			},
		},
	})
	app.SetCommands(commands)
	return app, nil
}

// runEngineHealth surfaces the signal the container healthcheck cannot see:
// /healthz stays 200 while every engine is suspended. Exit is non-zero only
// when zero engines respond (critical), so try_start orchestration flows are
// not broken by a merely degraded instance.
func runEngineHealth(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("engine-health", flag.ContinueOnError)
	fs.SetOutput(stdout)
	jsonOut := fs.Bool("json", false, "Emit the report as JSON")
	query := fs.String("query", health.DefaultCanaryQuery, "Canary search query")
	baseURL := fs.String("base-url", "", "Override SearXNG base URL (default: $SEARXNG_URL or http://localhost:8280)")
	timeout := fs.Duration("timeout", 30*time.Second, "Probe timeout")
	if err := fs.Parse(args); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	report, err := health.Probe(ctx, nil, resourceenv.ResolveBaseURL(*baseURL), *query)
	if err != nil {
		return fmt.Errorf("engine health probe failed: %w", err)
	}

	if *jsonOut {
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(report); err != nil {
			return err
		}
	} else {
		fmt.Fprintf(stdout, "Engine health: %s\n", report.Status)
		fmt.Fprintf(stdout, "Responsive engines (%d): %v\n", len(report.ResponsiveEngines), report.ResponsiveEngines)
		for _, issue := range report.UnresponsiveEngines {
			fmt.Fprintf(stdout, "  ⚠ %s: %s\n", issue.Engine, issue.Reason)
		}
		for engine, engineErrors := range report.ErrorStats {
			fmt.Fprintf(stdout, "  recorded errors %s: %v\n", engine, engineErrors)
		}
	}

	if report.Status == health.StatusCritical {
		return fmt.Errorf("no engines responded to the canary query")
	}
	return nil
}
