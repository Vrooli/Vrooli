package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/vrooli/cli-core/cliapp"

	"resource-twilio/cli/internal/health"
)

const (
	appName    = "twilio"
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
		Description:         "Twilio resource CLI",
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
		Commands: []cliapp.Command{{
			Name:        "provider-check",
			Description: "Verify Twilio credentials with a read-only provider request",
			Run:         runProviderCheck,
		}},
	})
	app.SetCommands(commands)
	return app, nil
}

func runProviderCheck(args []string) error {
	flags := flag.NewFlagSet("provider-check", flag.ContinueOnError)
	endpoint := flags.String("endpoint", "https://api.twilio.com/2010-04-01/Accounts.json", "Twilio Accounts endpoint")
	timeout := flags.Duration("timeout", 15*time.Second, "request timeout")
	if err := flags.Parse(args); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	status, err := health.Probe(ctx, nil, *endpoint, os.Getenv("TWILIO_ACCOUNT_SID"), os.Getenv("TWILIO_AUTH_TOKEN"))
	if err != nil {
		return err
	}
	fmt.Printf("Twilio provider credentials accepted (HTTP %d)\n", status)
	return nil
}
