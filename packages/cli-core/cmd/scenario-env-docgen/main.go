package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
)

const (
	startMarker = "<!-- scenario-cli-environment:start -->"
	endMarker   = "<!-- scenario-cli-environment:end -->"
)

var readmePath = func() string {
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		return filepath.Join("packages", "cli-core", "README.md")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(source), "..", "..", "README.md"))
}()

func markdownVariables(values []string) string {
	for i, value := range values {
		values[i] = "`" + strings.ReplaceAll(value, "EXAMPLE_SCENARIO", "<PREFIX>") + "`"
	}
	return strings.Join(values, ", ")
}

func generatedTable() string {
	env := cliapp.StandardScenarioEnv("example-scenario", cliapp.ScenarioEnvOptions{})
	rows := [][2]string{
		{"API base URL", markdownVariables(env.APIEnvVars)},
		{"API port", markdownVariables(env.APIPortEnvVars)},
		{"CLI config directory", markdownVariables(env.ConfigDirEnvVars)},
		{"CLI source root", markdownVariables(env.SourceRootEnvVars)},
		{"API token", markdownVariables(env.TokenEnvVars)},
		{"HTTP timeout", markdownVariables(env.HTTPTimeoutEnvVars)},
	}
	lines := []string{
		"| Purpose | Standard variables (first matching) |",
		"|---|---|",
	}
	for _, row := range rows {
		lines = append(lines, "| "+row[0]+" | "+row[1]+" |")
	}
	return strings.Join(lines, "\n")
}

func replaceGeneratedBlock(document string) (string, error) {
	if strings.Count(document, startMarker) != 1 || strings.Count(document, endMarker) != 1 {
		return "", fmt.Errorf("README must contain one pair of scenario CLI environment markers")
	}
	before, rest, _ := strings.Cut(document, startMarker)
	_, after, _ := strings.Cut(rest, endMarker)
	return before + startMarker + "\n" + generatedTable() + "\n" + endMarker + after, nil
}

func update(write bool) error {
	document, err := os.ReadFile(readmePath)
	if err != nil {
		return err
	}
	updated, err := replaceGeneratedBlock(string(document))
	if err != nil {
		return err
	}
	if !write {
		if string(document) != updated {
			return fmt.Errorf("%s has a stale scenario environment table; run with --write", readmePath)
		}
		fmt.Println("scenario environment table matches cliapp.StandardScenarioEnv")
		return nil
	}
	if err := os.WriteFile(readmePath, []byte(updated), 0o644); err != nil {
		return err
	}
	fmt.Println("updated scenario environment table from cliapp.StandardScenarioEnv")
	return nil
}

func main() {
	write := flag.Bool("write", false, "refresh the generated table")
	check := flag.Bool("check", false, "fail when the generated table is stale")
	flag.Parse()
	if *write == *check {
		fmt.Fprintln(os.Stderr, "choose exactly one of --write or --check")
		os.Exit(2)
	}
	if err := update(*write); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
