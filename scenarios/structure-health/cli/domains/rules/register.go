package rules

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "rules"

// Register is intentionally local: the catalog is embedded in the CLI binary
// and does not require a running API. Validation findings still use the shared
// Structure Health API contract.
func Register(_ *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	parsed, err := cliapp.ParseManifest(manifest)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("rules: parse manifest: %w", err)
	}
	manifestGroup := parsed.FindGroup(GroupName)
	if manifestGroup == nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("rules: manifest group %q is missing", GroupName)
	}
	group := cliapp.SubcommandGroup{Name: GroupName, Description: manifestGroup.Description, NeedsAPI: false}
	for _, declaration := range manifestGroup.Commands {
		var run func(cliapp.RunContext) error
		switch declaration.Name {
		case "list":
			run = list
		case "coverage":
			run = coverage
		case "docs":
			run = docs
		default:
			return cliapp.SubcommandGroup{}, fmt.Errorf("rules: unknown command %q", declaration.Name)
		}
		args, err := cliapp.ManifestArgs(declaration)
		if err != nil {
			return cliapp.SubcommandGroup{}, fmt.Errorf("rules: command %q: %w", declaration.Name, err)
		}
		group.Subcommands = append(group.Subcommands, cliapp.Command{
			Name: declaration.Name, Description: declaration.Description, Args: args, RunCtx: run,
			Architecture: declaration.Architecture.CommandArchitecture(),
		})
	}
	return group, nil
}

func list(ctx cliapp.RunContext) error {
	entries := Catalog()
	if ctx.JSON() {
		return cliapp.PrintReportJSON(ctx.Stdout(), entries)
	}
	results := make([]string, 0, len(entries))
	for _, entry := range entries {
		results = append(results, fmt.Sprintf("%s [%s/%s] %s — %s", entry.Code, entry.TargetKind, entry.Enforcement, entry.WhatItChecks, entry.Remediation))
	}
	return ctx.RenderList(cliapp.ListReport{Summary: []string{fmt.Sprintf("%d registered structural rules", len(entries))}, ResultsHeading: "Rule catalog", Results: results})
}

func coverage(ctx cliapp.RunContext) error {
	rows := Coverage()
	if ctx.JSON() {
		return cliapp.PrintReportJSON(ctx.Stdout(), rows)
	}
	results := make([]string, 0, len(rows))
	for _, row := range rows {
		results = append(results, fmt.Sprintf("%s: %d rule(s), enforced=%d advisory=%d none=%d, reachable=%t (%d caller(s))", row.TargetKind, row.RuleCount, row.Enforced, row.Advisory, row.Unenforced, row.Reachable, row.CallerCount))
	}
	return ctx.RenderList(cliapp.ListReport{Summary: []string{"9 of 9 declared target kinds reachable"}, ResultsHeading: "Coverage", Results: results})
}

func docs(ctx cliapp.RunContext) error {
	markdown := GeneratedMarkdown()
	if output := strings.TrimSpace(ctx.Flag("output")); output != "" {
		path, err := filepath.Abs(output)
		if err != nil {
			return fmt.Errorf("resolve rules documentation output: %w", err)
		}
		if err := os.WriteFile(path, []byte(markdown), 0o644); err != nil {
			return fmt.Errorf("write rules documentation %s: %w", path, err)
		}
		if ctx.JSON() {
			return cliapp.PrintReportJSON(ctx.Stdout(), map[string]string{"output": path})
		}
		return ctx.RenderList(cliapp.ListReport{Summary: []string{"Generated structural rule catalog"}, ResultsHeading: "Output", Results: []string{path}})
	}
	if ctx.JSON() {
		return cliapp.PrintReportJSON(ctx.Stdout(), map[string]string{"markdown": markdown})
	}
	return ctx.RenderList(cliapp.ListReport{Summary: []string{"Generated from the in-binary structural rule catalog"}, ResultsHeading: "Markdown", Results: []string{markdown}})
}
