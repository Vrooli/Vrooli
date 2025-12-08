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
	case "report":
		return runReport(args[1:])
	case "validate":
		return runValidate(args[1:])
	case "sync":
		return runSync(args[1:])
	case "manual-log":
		return runManualLog(args[1:])
	case "lint-prd":
		return runLintPRD(args[1:])
	case "drift":
		return runDrift(args[1:])
	case "phase", "phase-inspect":
		return runPhaseInspect(args[1:])
	case "init":
		return runInit(args[1:])
	case "help", "-h", "--help":
		return printUsage()
	default:
		return fmt.Errorf("unknown subcommand: %s\n\nRun 'test-genie requirements help' for usage", args[0])
	}
}

func printUsage() error {
	fmt.Println(`Usage: test-genie requirements <command>

Commands:
  report       Generate requirement coverage report (json|markdown|trace|summary)
  validate     Validate requirement files (schema + semantics)
  sync         Sync requirement statuses from local evidence
  manual-log   Record a manual validation entry
  lint-prd     Check PRD ↔ requirements mapping
  drift        Detect drift between snapshots, evidence, and requirements
  phase        Inspect validations for a single phase
  init         Scaffold a requirements/ registry

Run 'test-genie requirements <command> -h' for command-specific options.`)
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
