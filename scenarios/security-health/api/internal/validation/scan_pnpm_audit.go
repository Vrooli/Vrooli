package validation

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// pnpmAuditScanner runs `pnpm audit` against each pnpm-lock.yaml and normalizes
// the classic npm-audit advisory format. npm severities (critical/high ->
// ERROR, moderate -> WARNING, low/info -> INFO) flow through NormalizeSeverity.
type pnpmAuditScanner struct {
	cmd Commander
}

func newPnpmAuditScanner(cmd Commander) Scanner { return &pnpmAuditScanner{cmd: cmd} }

func (p *pnpmAuditScanner) Name() string             { return "pnpm-audit" }
func (p *pnpmAuditScanner) Binary() string           { return "pnpm" }
func (p *pnpmAuditScanner) Applies(s Substrate) bool { return s.PnpmUI }
func (p *pnpmAuditScanner) EvidencePlan(ctx context.Context, scenarioDir string, sub Substrate, now time.Time) (ScannerEvidencePlan, error) {
	return scannerEvidencePlan(ctx, p.cmd, p.Name(), p.Binary(), scenarioDir, sub, now)
}

type pnpmAuditReport struct {
	Advisories map[string]pnpmAdvisory `json:"advisories"`
}

type pnpmAdvisory struct {
	ID                 int      `json:"id"`
	ModuleName         string   `json:"module_name"`
	Severity           string   `json:"severity"`
	Title              string   `json:"title"`
	URL                string   `json:"url"`
	VulnerableVersions string   `json:"vulnerable_versions"`
	Recommendation     string   `json:"recommendation"`
	CVES               []string `json:"cves"`
	GithubAdvisoryID   string   `json:"github_advisory_id"`
}

const reactRouterRSCAdvisoryID = "GHSA-qwww-vcr4-c8h2"

func (p *pnpmAuditScanner) Scan(ctx context.Context, scenarioDir string, sub Substrate) ([]Finding, error) {
	dirs := sub.PnpmLockDirs
	if len(dirs) == 0 {
		dirs = []string{"ui"}
	}
	var findings []Finding
	var lastErr error
	parsedAny := false
	for _, rel := range dirs {
		lockDir := filepath.Join(scenarioDir, rel)
		// --ignore-workspace audits this project's own lockfile rather than
		// walking up to the monorepo root. pnpm audit exits non-zero when
		// advisories exist, so we ignore exitCode and parse stdout.
		stdout, stderr, _, err := p.cmd.Run(ctx, lockDir, "pnpm", "audit", "--json", "--ignore-workspace")
		if err != nil {
			lastErr = fmt.Errorf("pnpm audit failed to run in %s: %w", rel, err)
			continue
		}
		if len(stdout) == 0 {
			lastErr = fmt.Errorf("pnpm audit produced no output in %s: %s", rel, truncate(stderr, 200))
			continue
		}
		var report pnpmAuditReport
		if err := json.Unmarshal(stdout, &report); err != nil {
			lastErr = fmt.Errorf("parse pnpm audit json in %s: %w", rel, err)
			continue
		}
		parsedAny = true
		// Production-only audit: the set of advisories that affect shipped
		// (non-dev) dependencies. A vuln in a dev-only tool (vitest, eslint,
		// the build toolchain) is NOT in the deployed attack surface, so it is
		// advisory (capped to WARNING) — only production-dependency vulns gate
		// R1. Best-effort: if the prod audit can't run, fall back to treating
		// everything at native severity (conservative).
		prodIDs := p.prodAdvisoryIDs(ctx, lockDir)
		// Deterministic order: advisories is a map keyed by id string.
		ids := make([]string, 0, len(report.Advisories))
		for id := range report.Advisories {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		lockRel := filepath.ToSlash(filepath.Join(rel, "pnpm-lock.yaml"))
		for _, id := range ids {
			a := report.Advisories[id]
			ruleID := a.GithubAdvisoryID
			if ruleID == "" {
				ruleID = fmt.Sprintf("npm-%d", a.ID)
			}
			rec := nonEmpty(a.Recommendation, "Upgrade "+a.ModuleName+" to a patched version.")
			sev := NormalizeSeverity(a.Severity)
			devOnly := prodIDs != nil && !prodIDs[id]
			if devOnly && sev == SeverityError {
				sev = SeverityWarning
				rec = "Dev-only dependency (not in the shipped artifact) — advisory. " + rec
			}
			if ruleID == reactRouterRSCAdvisoryID && !usesReactRouterRSC(filepath.Join(scenarioDir, rel)) && sev == SeverityError {
				sev = SeverityWarning
				rec = "This advisory affects only applications using React Router's unstable RSC APIs; no first-party RSC usage was found. " + rec
			}
			findings = append(findings, Finding{
				RuleID:       "pnpm-audit." + ruleID,
				Severity:     sev,
				Title:        fmt.Sprintf("%s: %s", a.ModuleName, nonEmpty(a.Title, "known vulnerability")),
				Description:  fmt.Sprintf("%s (vulnerable: %s). Advisory: %s", nonEmpty(a.Title, "known vulnerability"), nonEmpty(a.VulnerableVersions, "see advisory"), a.URL),
				Remediation:  rec,
				FilePath:     lockRel,
				Scanner:      p.Name(),
				Class:        FindingVulnerability,
				Owner:        "scenario-dependency-analyzer",
				FixClass:     FixAssisted,
				PolicyImpact: "dependency-upgrade-governed",
			})
		}
	}
	if !parsedAny && lastErr != nil {
		return nil, lastErr
	}
	return findings, nil
}

// usesReactRouterRSC conservatively identifies first-party usage of React
// Router's unstable RSC APIs. The GHSA-qwww-vcr4-c8h2 advisory explicitly
// limits exposure to that mode, so ordinary BrowserRouter applications must
// not receive a deployment-blocking finding solely from a transitive package.
// A source or package declaration mentioning RSC retains the native severity.
func usesReactRouterRSC(uiDir string) bool {
	used := false
	_ = filepath.WalkDir(uiDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || used {
			return nil
		}
		if entry.IsDir() {
			switch entry.Name() {
			case "node_modules", "dist", "build", "coverage", ".git":
				return filepath.SkipDir
			}
			return nil
		}
		extension := strings.ToLower(filepath.Ext(path))
		if extension != ".js" && extension != ".jsx" && extension != ".ts" && extension != ".tsx" && extension != ".mjs" && extension != ".cjs" && filepath.Base(path) != "package.json" {
			return nil
		}
		name := strings.ToLower(filepath.Base(path))
		if strings.Contains(name, ".test.") || strings.Contains(name, ".spec.") {
			return nil
		}
		contents, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		used = hasReactRouterRSCUsage(string(contents))
		return nil
	})
	return used
}

func hasReactRouterRSCUsage(contents string) bool {
	contents = strings.ToLower(contents)
	for _, marker := range []string{
		"react-router-rsc",
		"@react-router/rsc",
		"unstable_rsc",
		"rschydratedrouter",
		"rscstaticrouter",
		"rscrouter",
		"creatersc",
		"createcallserver",
		"serverrouter",
	} {
		if strings.Contains(contents, marker) {
			return true
		}
	}
	return false
}

// prodAdvisoryIDs runs `pnpm audit --prod` and returns the set of advisory id
// keys that affect production (shipped) dependencies. Returns nil when the
// production audit can't be run or parsed — callers treat nil as "don't know",
// keeping every finding at its native severity (conservative).
func (p *pnpmAuditScanner) prodAdvisoryIDs(ctx context.Context, lockDir string) map[string]bool {
	stdout, _, _, err := p.cmd.Run(ctx, lockDir, "pnpm", "audit", "--prod", "--json", "--ignore-workspace")
	if err != nil || len(stdout) == 0 {
		return nil
	}
	var report pnpmAuditReport
	if err := json.Unmarshal(stdout, &report); err != nil {
		return nil
	}
	ids := make(map[string]bool, len(report.Advisories))
	for id := range report.Advisories {
		ids[id] = true
	}
	return ids
}
