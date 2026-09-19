package validation

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"
)

// gitleaksScanner detects committed secrets. It is language-agnostic, so it
// applies to every scenario. A leaked credential in tracked content is always
// normalized to ERROR — a live secret bound for git is exactly what the R1
// gate must catch — and the raw secret value is never propagated into a
// Finding (file:line only).
//
// The scanner runs over an exact snapshot of tracked plus non-ignored untracked
// files. Ignored local secrets, dependencies, build output, and runtime state
// cannot enter a commit and are outside this gate; scanning them created both
// false-positive noise and cache self-invalidation.
type gitleaksScanner struct {
	cmd Commander
}

func newGitleaksScanner(cmd Commander) Scanner { return &gitleaksScanner{cmd: cmd} }

func (g *gitleaksScanner) Name() string   { return "gitleaks" }
func (g *gitleaksScanner) Binary() string { return "gitleaks" }

// Applies to every scenario: secrets can hide in any substrate.
func (g *gitleaksScanner) Applies(Substrate) bool { return true }

func (g *gitleaksScanner) EvidencePlan(ctx context.Context, scenarioDir string, sub Substrate, now time.Time) (ScannerEvidencePlan, error) {
	return scannerEvidencePlan(ctx, g.cmd, g.Name(), g.Binary(), scenarioDir, sub, now)
}

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
	sourceDir, cleanup, err := prepareCommitEligibleSnapshot(ctx, g.cmd, scenarioDir)
	if err != nil {
		return nil, fmt.Errorf("prepare gitleaks source inventory: %w", err)
	}
	defer cleanup()
	// --report-path /dev/stdout streams the JSON report to stdout (which the
	// Commander captures); --no-git scans the working tree rather than history;
	// --no-banner keeps stderr clean. gitleaks exits 1 when leaks are found, so
	// we ignore exitCode and parse stdout directly.
	stdout, stderr, _, _, err := runCommand(ctx, g.cmd, sourceDir,
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
		severity := SeverityError // a live credential bound for git gates R1
		description := fmt.Sprintf(
			"gitleaks rule %q matched a likely secret. The matched value is redacted; review the file directly.",
			nonEmpty(r.RuleID, "generic"),
		)
		remediation := "Rotate the exposed credential immediately, purge it from git history, and provision it through the canonical credential authority; use Vault only when the finding explicitly targets a Vault capability."
		findings = append(findings, Finding{
			RuleID:       "gitleaks." + nonEmpty(r.RuleID, "generic"),
			Severity:     severity,
			Title:        title,
			Description:  description,
			Remediation:  remediation,
			FilePath:     fmt.Sprintf("%s:%d", filepath.ToSlash(r.File), r.StartLine),
			Scanner:      g.Name(),
			Class:        FindingSecret,
			Owner:        "vault-owner",
			FixClass:     FixManual,
			PolicyImpact: "credential-rotation-required",
		})
	}
	return findings, nil
}
