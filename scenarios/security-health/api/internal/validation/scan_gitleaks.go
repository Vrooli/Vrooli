package validation

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

// gitleaksScanner detects committed secrets. It is language-agnostic, so it
// applies to every scenario. A leaked credential in tracked content is always
// normalized to ERROR — a live secret bound for git is exactly what the R1
// gate must catch — and the raw secret value is never propagated into a
// Finding (file:line only).
//
// Matches inside *gitignored* files are downgraded to INFO rather than
// dropped: a gitignored `.env` is the sanctioned home for local credentials
// (it can never be committed), and machine-generated ignored files like
// `.build-fingerprint.json` are full of hash strings that pattern-match as
// keys. Gating R1 on those would perma-fail every conventionally configured
// scenario, but the match stays visible as an observation.
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
	ignored := g.gitIgnoredSet(ctx, scenarioDir, raw)
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
		remediation := "Rotate the exposed credential immediately, purge it from git history, and move the secret into the vault resource (resource-vault content set …)."
		if ignored[r.File] {
			// Gitignored: cannot be committed, so it does not gate. Kept as an
			// INFO observation rather than dropped.
			severity = SeverityInfo
			description = fmt.Sprintf(
				"gitleaks rule %q matched a likely secret in a gitignored file. The file cannot be committed, so this does not gate; the matched value is redacted.",
				nonEmpty(r.RuleID, "generic"),
			)
			remediation = "Gitignored files are the sanctioned home for local credentials; no action required unless the value is unexpectedly a real shared secret, in which case move it into the vault resource (resource-vault content set …)."
		}
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

// gitIgnoredSet returns the subset of finding file paths that git ignores in
// scenarioDir, by batching them through `git check-ignore`. Paths come back
// exactly as passed in, so the result is keyed on the raw gitleaks File
// values. A path that is nonetheless *tracked* (force-added past .gitignore)
// is removed again — committed content must keep gating regardless of ignore
// rules. Fails open: when git is unavailable, the dir is not a work tree
// (exit 128), or nothing is ignored (exit 1), every finding keeps its native
// severity.
func (g *gitleaksScanner) gitIgnoredSet(ctx context.Context, scenarioDir string, raw []gitleaksFinding) map[string]bool {
	seen := make(map[string]bool, len(raw))
	paths := make([]string, 0, len(raw))
	for _, r := range raw {
		if r.File == "" || seen[r.File] {
			continue
		}
		seen[r.File] = true
		paths = append(paths, r.File)
	}
	if len(paths) == 0 {
		return nil
	}
	// check-ignore output is newline-separated (`-z` requires `--stdin`,
	// which the Commander seam doesn't support); ls-files supports `-z`
	// directly. gitPathSet tolerates both separators.
	ignored := g.gitPathSet(ctx, scenarioDir, append([]string{"check-ignore", "--"}, paths...))
	if len(ignored) == 0 {
		return nil
	}
	// check-ignore tests ignore rules only, not tracked status; a tracked file
	// that happens to match an ignore pattern is still committed content.
	for p := range g.gitPathSet(ctx, scenarioDir, append([]string{"ls-files", "-z", "--"}, paths...)) {
		delete(ignored, p)
	}
	return ignored
}

// gitPathSet runs git with args in dir and parses NUL- or newline-separated
// paths from stdout into a set. Paths git renders C-quoted (non-ASCII, with
// quotePath on) are unquoted only of their surrounding quotes — exotic enough
// names simply won't match a finding path and thus stay at native severity.
// Returns nil on any failure (fails open).
func (g *gitleaksScanner) gitPathSet(ctx context.Context, dir string, args []string) map[string]bool {
	stdout, _, _, err := g.cmd.Run(ctx, dir, "git", args...)
	if err != nil || len(stdout) == 0 {
		return nil
	}
	set := make(map[string]bool)
	for _, p := range strings.FieldsFunc(string(stdout), func(r rune) bool { return r == '\x00' || r == '\n' }) {
		p = strings.TrimSuffix(strings.TrimPrefix(p, `"`), `"`)
		if p != "" {
			set[p] = true
		}
	}
	return set
}
