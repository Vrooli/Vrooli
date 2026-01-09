// Package registry provides CLI commands for managing playbook registries.
package registry

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	registrypkg "test-genie/cli/internal/registry"
)

// Run executes the registry command with the given arguments.
func Run(args []string) error {
	if len(args) == 0 {
		return printUsage()
	}

	switch args[0] {
	case "build":
		return runBuild(args[1:])
	case "help", "-h", "--help":
		return printUsage()
	default:
		return fmt.Errorf("unknown subcommand: %s\n\nRun 'test-genie registry help' for usage", args[0])
	}
}

func printUsage() error {
	fmt.Println(`Usage: test-genie registry <command>

Commands:
  build    Build playbook registry for a scenario

Run 'test-genie registry <command> -h' for more information on a command.`)
	return nil
}

func runBuild(args []string) error {
	fs := flag.NewFlagSet("registry build", flag.ContinueOnError)
	scenarioDir := fs.String("scenario", "", "Path to scenario directory (defaults to current directory)")
	quiet := fs.Bool("q", false, "Quiet mode - only output errors")

	if err := fs.Parse(args); err != nil {
		return err
	}

	// Determine scenario directory
	dir := *scenarioDir
	if dir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		dir = cwd
	}

	// Make path absolute
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	// Build registry
	builder := registrypkg.NewBuilder(absDir)
	result, err := builder.Build()
	if err != nil {
		return fmt.Errorf("failed to build registry: %w", err)
	}

	if !*quiet {
		relPath, _ := filepath.Rel(absDir, result.RegistryPath)
		if relPath == "" {
			relPath = result.RegistryPath
		}
		fmt.Printf("[playbook-registry] Updated %s (%d workflows)\n", relPath, result.PlaybookCount)
	}

	return nil
}
