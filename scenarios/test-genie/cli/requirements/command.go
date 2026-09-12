package requirements

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

// Run dispatches requirements subcommands.
func Run(args []string) error {
	if len(args) == 0 {
		return printUsage()
	}

	switch args[0] {
	case "sync":
		return runSync(args[1:])
	case "report", "validate", "manual-log", "lint-prd", "drift", "phase", "phase-inspect", "init":
		return fmt.Errorf("`test-genie requirements %s` moved to business-health (the business-contract provider).\n\nUse `vrooli scenario requirements %s <scenario>` (routed automatically) or the business-health CLI directly.", args[0], args[0])
	case "help", "-h", "--help":
		return printUsage()
	default:
		return fmt.Errorf("unknown subcommand: %s\n\nRun 'test-genie requirements help' for usage", args[0])
	}
}

func printUsage() error {
	fmt.Println(`Usage: test-genie requirements <command>

Commands:
  sync         Sync requirement statuses from local evidence (the run-coupled
               evidence writer; fires automatically inside suite execution)

The contract-side verbs (validate, report, lint-prd, drift, phase, init,
manual-log) live in business-health — use
'vrooli scenario requirements <verb> <scenario>' or the business-health CLI.

Run 'test-genie requirements sync -h' for options.`)
	return nil
}

func resolveDir(flagDir string) (string, error) {
	dir := flagDir
	if dir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to get current directory: %w", err)
		}
		dir = cwd
	}
	abs, err := filepathAbs(dir)
	if err != nil {
		return "", err
	}
	return abs, nil
}

// filepathAbs is extracted for testability.
var filepathAbs = func(path string) (string, error) {
	return filepath.Abs(path)
}

// parseCommonFlags parses scenario directory and optional scenario name flags.
func parseCommonFlags(fs *flag.FlagSet) (dir *string, scenario *string) {
	dir = fs.String("dir", "", "Path to scenario directory (defaults to current directory)")
	scenario = fs.String("scenario", "", "Scenario name (defaults to directory name)")
	return
}

// scenarioNameFromDir derives the scenario name from the directory if not provided.
func scenarioNameFromDir(dir, name string) string {
	if name != "" {
		return name
	}
	base := filepath.Base(dir)
	if base != "" && base != "." && base != "/" {
		return base
	}
	return ""
}

// ensureDir validates the scenario directory exists.
func ensureDir(dir string) error {
	info, err := os.Stat(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("scenario directory not found: %s", dir)
		}
		return fmt.Errorf("unable to stat scenario directory: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("scenario path is not a directory: %s", dir)
	}
	return nil
}
