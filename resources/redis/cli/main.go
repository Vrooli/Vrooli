package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"resource-redis/cli/internal/backup"

	"github.com/vrooli/cli-core/cliapp"
)

const (
	appName    = "redis"
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
		Description:         "Redis resource CLI",
		SourceRootEnvVars:   env.SourceRootEnvVars,
		ControlPlaneEnvVars: env.ControlPlaneEnvVars,
		BuildFingerprint:    buildFingerprint,
		BuildTimestamp:      buildTimestamp,
		BuildSourceRoot:     buildSourceRoot,
	})
	if err != nil {
		return nil, err
	}
	commands := append(app.StandardLifecycleCommands(), cliapp.CommandGroup{Title: "Backup", Commands: []cliapp.Command{
		{Name: "dump", Description: "Write a best-effort Redis prefix archive", Run: runDump},
		{Name: "restore", Description: "Restore a Redis prefix archive", Run: runRestore},
	}})
	app.SetCommands(commands)
	return app, nil
}

func redisClient(flags *flag.FlagSet) (*string, *time.Duration) {
	address := flags.String("address", "127.0.0.1:6380", "Redis address")
	timeout := flags.Duration("timeout", 15*time.Second, "operation timeout")
	return address, timeout
}

func runDump(args []string) error {
	flags := flag.NewFlagSet("dump", flag.ContinueOnError)
	prefix := flags.String("prefix", "", "key prefix to back up")
	output := flags.String("output", "", "archive file path")
	address, timeout := redisClient(flags)
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *output == "" {
		return fmt.Errorf("--output is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	archive, err := (backup.Client{Address: *address, Timeout: *timeout}).Dump(ctx, *prefix)
	if err != nil {
		return err
	}
	data, err := backup.Encode(archive)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(*output), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(*output, data, 0o600); err != nil {
		return err
	}
	fmt.Printf("Wrote %d Redis entries to %s\n", len(archive.Entries), *output)
	return nil
}

func runRestore(args []string) error {
	flags := flag.NewFlagSet("restore", flag.ContinueOnError)
	prefix := flags.String("prefix", "", "key prefix to restore")
	input := flags.String("input", "", "archive file path")
	address, timeout := redisClient(flags)
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *input == "" {
		return fmt.Errorf("--input is required")
	}
	data, err := os.ReadFile(*input)
	if err != nil {
		return err
	}
	archive, err := backup.Decode(data)
	if err != nil {
		return fmt.Errorf("read Redis archive: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	if err := (backup.Client{Address: *address, Timeout: *timeout}).Restore(ctx, archive, *prefix); err != nil {
		return err
	}
	fmt.Printf("Restored %d Redis entries from %s\n", len(archive.Entries), *input)
	return nil
}
