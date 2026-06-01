package validation

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
)

// gitleaksScanner detects committed secrets. It is language-agnostic, so it
// applies to every scenario. A leaked credential is always normalized to
// ERROR — a live secret in the tree is exactly what the R1 gate must catch —
// and the raw secret value is never propagated into a Finding (file:line only).
type gitleaksScanner struct {
	cmd Commander
}

func newGitleaksScanner(cmd Commander) Scanner { return &gitleaksScanner{cmd: cmd} }

func (g *gitleaksScanner) Name() string   { return "gitleaks" }
func (g *gitleaksScanner) Binary() string { return "gitleaks" }

// Applies to every scenario: secrets can hide in any substrate.
func (g *gitleaksScanner) Applies(Substrate) bool { return true }

// gitleaksFinding mirrors the fields of a single object in gitleaks' JSON
// report array. We deliberately omit Secret/Match from the struct we keep so
// the value cannot accidentally leak into a Finding.
type gitleaksFinding struct {
	Description string `json:"Description"`
	StartLine   int    `json:"StartLine"`
	File        string `json:"File"`
	RuleID      string `json:"RuleID"`
}

func (g *gitleaksScanner) Scan(ctx context.Context, scenarioDir string, _ Substrate) ([]Finding, error) {
	// --report-path /dev/stdout streams the JSON report to stdout (which the
	// Commander captures); --no-git scans the working tree rather than history;
	// --no-banner keeps stderr clean. gitleaks exits 1 when leaks are found, so
	// we ignore exitCode and parse stdout directly.
	stdout, stderr, _, err := g.cmd.Run(ctx, scenarioDir,
		"gitleaks", "detect",
		"--source", ".",
		"--no-git",
		"--report-format", "json",
		"--report-path", "/dev/stdout",
		"--no-banner",
	)
	if err != nil {
		return nil, fmt.Errorf("gitleaks failed to run: %w", err)
	}
	if len(stdout) == 0 {
		// No report emitted; treat as clean only when stderr is also quiet.
		if len(stderr) > 0 {
			return nil, fmt.Errorf("gitleaks produced no report: %s", truncate(stderr, 200))
		}
		return nil, nil
	}
	var raw []gitleaksFinding
	if err := json.Unmarshal(stdout, &raw); err != nil {
		return nil, fmt.Errorf("parse gitleaks json: %w", err)
	}
	findings := make([]Finding, 0, len(raw))
	for _, r := range raw {
		title := r.Description
		if title == "" {
			title = "Potential secret detected"
		}
		findings = append(findings, Finding{
			RuleID:   "gitleaks." + nonEmpty(r.RuleID, "generic"),
			Severity: SeverityError, // a live credential gates R1
			Title:    title,
			Description: fmt.Sprintf(
				"gitleaks rule %q matched a likely secret. The matched value is redacted; review the file directly.",
				nonEmpty(r.RuleID, "generic"),
			),
			Remediation: "Rotate the exposed credential immediately, purge it from git history, and move the secret into the vault resource (resource-vault content set …).",
			FilePath:    fmt.Sprintf("%s:%d", filepath.ToSlash(r.File), r.StartLine),
			Scanner:     g.Name(),
		})
	}
	return findings, nil
}
