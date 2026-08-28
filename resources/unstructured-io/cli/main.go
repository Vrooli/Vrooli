package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/vrooli/vrooli/resources/unstructured-io/cli/internal/unstructured"

	"github.com/vrooli/cli-core/cliapp"
)

const (
	appName    = "unstructured-io"
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
		Description:         "Unstructured.io resource CLI",
		SourceRootEnvVars:   env.SourceRootEnvVars,
		ControlPlaneEnvVars: env.ControlPlaneEnvVars,
		BuildFingerprint:    buildFingerprint,
		BuildTimestamp:      buildTimestamp,
		BuildSourceRoot:     buildSourceRoot,
	})
	if err != nil {
		return nil, err
	}
	commands := append(app.StandardLifecycleCommands(), cliapp.CommandGroup{Title: "Document processing", Commands: []cliapp.Command{{Name: "health", Description: "Verify the Unstructured API", Run: runHealth}, {Name: "formats", Description: "List supported document formats", Run: runFormats}, {Name: "process", Description: "Partition one supported document", Run: runProcess}}})
	app.SetCommands(commands)
	return app, nil
}

func unstructuredClient(f *flag.FlagSet) (*string, *time.Duration) {
	u := f.String("url", "http://127.0.0.1:11450", "Unstructured API URL")
	t := f.Duration("timeout", 60*time.Second, "request timeout")
	return u, t
}

func runHealth(args []string) error {
	f := flag.NewFlagSet("health", flag.ContinueOnError)
	u, t := unstructuredClient(f)
	if e := f.Parse(args); e != nil {
		return e
	}
	ctx, cancel := context.WithTimeout(context.Background(), *t)
	defer cancel()
	return (unstructured.Client{BaseURL: *u, HTTPClient: &http.Client{Timeout: *t}}).Health(ctx)
}

func runFormats(args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("formats accepts no arguments")
	}
	return printJSON(unstructured.SupportedFormats())
}

func runProcess(args []string) error {
	f := flag.NewFlagSet("process", flag.ContinueOnError)
	input := f.String("input", "", "document path")
	output := f.String("output", "", "JSON output path; defaults to stdout")
	u, t := unstructuredClient(f)
	if e := f.Parse(args); e != nil {
		return e
	}
	if *input == "" {
		return fmt.Errorf("--input is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), *t)
	defer cancel()
	v, e := (unstructured.Client{BaseURL: *u, HTTPClient: &http.Client{Timeout: *t}}).Process(ctx, *input)
	if e != nil {
		return e
	}
	b, e := json.MarshalIndent(v, "", "  ")
	if e != nil {
		return e
	}
	if *output == "" {
		fmt.Println(string(b))
		return nil
	}
	return os.WriteFile(*output, b, 0o600)
}

func printJSON(v any) error {
	b, e := json.MarshalIndent(v, "", "  ")
	if e == nil {
		fmt.Println(string(b))
	}
	return e
}
