package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

func (a *App) cmdValidate(args []string) error {
	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	entityType := fs.String("type", "", "Entity type: scenario or resource (default: auto-detect)")
	remaining, err := parseArgs(fs, args)
	if err != nil {
		return err
	}
	if len(remaining) == 0 {
		return fmt.Errorf("usage: validate <name> [--type scenario|resource] [--json]")
	}

	name := strings.TrimSpace(remaining[0])
	if name == "" {
		return fmt.Errorf("usage: validate <name> [--type scenario|resource] [--json]")
	}

	resolvedType := strings.ToLower(strings.TrimSpace(*entityType))
	if resolvedType == "" {
		resolvedType = detectEntityTypeFromRepo(name)
	}
	if resolvedType != "scenario" && resolvedType != "resource" {
		return fmt.Errorf("invalid --type %q (must be scenario or resource)", *entityType)
	}

	body, err := a.services.PRD.Validate(ValidateRequest{EntityType: resolvedType, EntityName: name})
	if err != nil {
		return err
	}
	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Printf("Validation requested for %s/%s\n", resolvedType, name)
	cliutil.PrintJSON(body)
	return nil
}

func detectEntityTypeFromRepo(name string) string {
	root := findRepoRoot()
	if root == "" {
		return "scenario"
	}
	if statDir(filepath.Join(root, "resources", name)) {
		return "resource"
	}
	if statDir(filepath.Join(root, "scenarios", name)) {
		return "scenario"
	}
	return "scenario"
}

func findRepoRoot() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	dir := cwd
	for i := 0; i < 25; i++ {
		if statDir(filepath.Join(dir, "scenarios")) && statDir(filepath.Join(dir, "resources")) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

func statDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
