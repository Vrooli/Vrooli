package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliapp"

	"github.com/vrooli/vrooli/resources/searxng/cli/internal/config"
	resourceenv "github.com/vrooli/vrooli/resources/searxng/cli/internal/env"
	"github.com/vrooli/vrooli/resources/searxng/cli/internal/health"
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
		Title: "Configuration",
		Commands: []cliapp.Command{
			{Name: "config-show", Description: "Show the owned SearXNG configuration (secrets redacted)", Run: func(args []string) error { return runConfigShow(args, os.Stdout) }},
			{Name: "config-validate", Description: "Validate the current SearXNG configuration", Run: func(args []string) error { return runConfigValidate(args, os.Stdout) }},
			{Name: "config-apply", Description: "Create or safely migrate the SearXNG configuration", Run: func(args []string) error { return runConfigApply(args, os.Stdout) }},
		},
	}, cliapp.CommandGroup{
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

func configFlags(name string, args []string, stdout io.Writer) (*flag.FlagSet, *string, error) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(stdout)
	dir := fs.String("config-dir", "", "Configuration directory (default: $RESOURCE_CONFIG_DIR)")
	if err := fs.Parse(args); err != nil {
		return nil, nil, err
	}
	return fs, dir, nil
}

func runConfigShow(args []string, stdout io.Writer) error {
	_, dirFlag, err := configFlags("config-show", args, stdout)
	if err != nil {
		return err
	}
	dir, err := config.ConfigDir(*dirFlag)
	if err != nil {
		return err
	}
	document, _, err := config.Load(dir)
	if err != nil {
		return err
	}
	return json.NewEncoder(stdout).Encode(config.RedactedSummary(document))
}

func runConfigValidate(args []string, stdout io.Writer) error {
	_, dirFlag, err := configFlags("config-validate", args, stdout)
	if err != nil {
		return err
	}
	dir, err := config.ConfigDir(*dirFlag)
	if err != nil {
		return err
	}
	document, _, err := config.Load(dir)
	if err != nil {
		return err
	}
	if err := config.Validate(document); err != nil {
		return err
	}
	_, err = fmt.Fprintln(stdout, "SearXNG configuration is valid.")
	return err
}

func runConfigApply(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("config-apply", flag.ContinueOnError)
	fs.SetOutput(stdout)
	dirFlag := fs.String("config-dir", "", "Configuration directory (default: $RESOURCE_CONFIG_DIR)")
	baseURL := fs.String("base-url", resourceenv.ResolveBaseURL(""), "SearXNG public base URL")
	instanceName := fs.String("instance-name", "Vrooli SearXNG", "SearXNG instance name")
	secretFile := fs.String("secret-file", "", "Read a session secret from this file when no existing secret is present")
	if err := fs.Parse(args); err != nil {
		return err
	}
	dir, err := config.ConfigDir(*dirFlag)
	if err != nil {
		return err
	}
	secret := ""
	if strings.TrimSpace(*secretFile) != "" {
		data, err := os.ReadFile(*secretFile)
		if err != nil {
			return fmt.Errorf("read secret file: %w", err)
		}
		secret = strings.TrimSpace(string(data))
		if secret == "" {
			return fmt.Errorf("secret file is empty")
		}
	}
	report, err := config.Apply(dir, *baseURL, *instanceName, secret)
	if err != nil {
		return err
	}
	return json.NewEncoder(stdout).Encode(report)
}

// runEngineHealth surfaces the signal the liveness healthcheck cannot see:
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
