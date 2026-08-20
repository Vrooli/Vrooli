package validation

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// gosecScanner runs the gosec Go SAST tool per Go module. gosec's own
// severity (HIGH/MEDIUM/LOW) flows through NormalizeSeverity, so only HIGH
// gates R1 while MEDIUM/LOW stay advisory — the false-positive mitigation
// from the plan (gosec is FP-prone; don't let a medium-confidence medium
// finding fail a scenario).
//
// Suppressions: standalone gosec honors only its own `#nosec` tag, but this
// repo lints Go through golangci-lint, where `//nolint:gosec` is the reviewed
// suppression idiom. Issues whose flagged source line carries a covering
// nolint directive are dropped here, mirroring what golangci-lint's gosec
// integration would do — otherwise every already-reviewed suppression
// re-fires as a fresh finding.
type gosecScanner struct {
	cmd Commander
}

func newGosecScanner(cmd Commander) Scanner { return &gosecScanner{cmd: cmd} }

func (g *gosecScanner) Name() string             { return "gosec" }
func (g *gosecScanner) Binary() string           { return "gosec" }
func (g *gosecScanner) Applies(s Substrate) bool { return s.Go }
func (g *gosecScanner) EvidencePlan(ctx context.Context, scenarioDir string, sub Substrate, now time.Time) (ScannerEvidencePlan, error) {
	return scannerEvidencePlan(ctx, g.cmd, g.Name(), g.Binary(), scenarioDir, sub, now)
}

type gosecReport struct {
	Issues []gosecIssue `json:"Issues"`
}

type gosecIssue struct {
	Severity string `json:"severity"`
	RuleID   string `json:"rule_id"`
	Details  string `json:"details"`
	File     string `json:"file"`
	Line     string `json:"line"`
	// Code is gosec's numbered source snippet around the flagged line
	// ("463: …\n464: …\n465: …"); used to honor //nolint suppressions
	// without re-reading the file.
	Code string `json:"code"`
}

func (g *gosecScanner) Scan(ctx context.Context, scenarioDir string, sub Substrate) ([]Finding, error) {
	dirs := sub.GoModDirs
	if len(dirs) == 0 {
		dirs = []string{"."}
	}
	var findings []Finding
	var lastErr error
	parsedAny := false
	for _, rel := range dirs {
		modDir := filepath.Join(scenarioDir, rel)
		// -no-fail keeps gosec's exit code at 0 even with issues; ./... scans
		// the whole module. We deliberately omit -quiet: with -quiet gosec
		// emits *nothing* on a clean scan, which is indistinguishable from a
		// loader failure. Without it, stdout always carries the JSON report
		// ({"Issues":[]} when clean) and the progress chatter goes to stderr.
		stdout, stderr, _, err := g.cmd.Run(ctx, modDir, "gosec", "-fmt=json", "-no-fail", "./...")
		if err != nil {
			lastErr = fmt.Errorf("gosec failed to run in %s: %w", rel, err)
			continue
		}
		if len(stdout) == 0 {
			// gosec couldn't analyze (e.g. toolchain/loader incompatibility);
			// remember why so the Service can emit an INFO observation.
			lastErr = fmt.Errorf("gosec produced no output in %s: %s", rel, truncate(stderr, 200))
			continue
		}
		var report gosecReport
		if err := json.Unmarshal(stdout, &report); err != nil {
			lastErr = fmt.Errorf("parse gosec json in %s: %w", rel, err)
			continue
		}
		parsedAny = true
		for _, issue := range report.Issues {
			if nolintSuppressesGosec(flaggedSourceLine(issue.Code, issue.Line)) {
				continue
			}
			loc := relPath(scenarioDir, issue.File)
			if issue.Line != "" {
				loc = fmt.Sprintf("%s:%s", loc, strings.SplitN(issue.Line, "-", 2)[0])
			}
			findings = append(findings, Finding{
				RuleID:       "gosec." + nonEmpty(issue.RuleID, "unknown"),
				Severity:     gosecSeverity(issue.RuleID, issue.Severity),
				Title:        fmt.Sprintf("gosec %s", nonEmpty(issue.RuleID, "finding")),
				Description:  strings.TrimSpace(issue.Details),
				Remediation:  "Review the flagged code; apply the gosec rule's documented fix (e.g. validate inputs, avoid unsafe APIs, use crypto/rand). Suppress with a reviewed #nosec or //nolint:gosec comment only when proven safe.",
				FilePath:     loc,
				Scanner:      g.Name(),
				Class:        FindingSAST,
				Owner:        "security-health",
				FixClass:     FixManual,
				PolicyImpact: "source-review-required",
			})
		}
	}
	// If no module parsed cleanly, surface the failure so the Service records a
	// degraded INFO observation rather than silently reporting "clean".
	if !parsedAny && lastErr != nil {
		return nil, lastErr
	}
	return findings, nil
}

// gosecOverFiringRules caps specific gosec rules below their native severity.
// gosec is FP-prone (plan R2); the contract is to SCOPE an over-firing rule
// (downgrade it, here, with a regression test) rather than disable it globally.
// Every entry is one of gosec's newest taint/TOCTOU heuristics, which fire HIGH
// on idiomatic, operator-controlled code present in the react-vite template —
// so leaving them at ERROR would perma-gate every Go scenario's R1 on
// boilerplate. They stay visible as WARNINGs; the high-signal mature rules
// (G101 hardcoded creds, G401 weak crypto, G501 blocklisted imports, …) keep
// their native ERROR and still gate.
//
//	G115 — "integer overflow conversion": fires on every `int32(len(x))` /
//	        `int32(count)` cast even when the value is provably small.
//	G122 — "filesystem op in WalkDir callback (symlink TOCTOU)": fires on any
//	        os.ReadFile inside a filepath.WalkDir over a controlled tree.
//	G703 — "path traversal via taint": fires on os.MkdirAll/ReadFile whose path
//	        derives from operator config (env var, storage resolver), not from
//	        untrusted request input.
var gosecOverFiringRules = map[string]Severity{
	"G115": SeverityWarning,
	"G122": SeverityWarning,
	"G703": SeverityWarning,
}

// gosecSeverity normalizes gosec's native severity, then applies any per-rule
// cap from gosecOverFiringRules (never escalates, only caps).
func gosecSeverity(ruleID, native string) Severity {
	sev := NormalizeSeverity(native)
	if capped, ok := gosecOverFiringRules[strings.TrimSpace(ruleID)]; ok && capped > sev {
		return capped
	}
	return sev
}

// flaggedSourceLine extracts the flagged line's source text from gosec's
// numbered code snippet. line may be a range ("54-58"); the first line is the
// one the directive must sit on. Returns "" when the snippet is absent or the
// line isn't in it — the issue then keeps its native severity (fail closed).
func flaggedSourceLine(code, line string) string {
	target := strings.TrimSpace(strings.SplitN(line, "-", 2)[0])
	if target == "" || code == "" {
		return ""
	}
	for _, l := range strings.Split(code, "\n") {
		num, src, ok := strings.Cut(l, ":")
		if ok && strings.TrimSpace(num) == target {
			return src
		}
	}
	return ""
}

// nolintDirective matches golangci-lint suppression comments: bare `//nolint`
// or `//nolint:linter1,linter2`, optionally followed by an explanation.
var nolintDirective = regexp.MustCompile(`//\s*nolint\b(?::([a-zA-Z0-9_, \t-]+))?`)

// nolintSuppressesGosec reports whether src carries a nolint directive that
// covers gosec — either the bare form (all linters) or a linter list
// containing "gosec".
func nolintSuppressesGosec(src string) bool {
	m := nolintDirective.FindStringSubmatch(src)
	if m == nil {
		return false
	}
	if strings.TrimSpace(m[1]) == "" {
		return true // bare //nolint suppresses every linter
	}
	for _, name := range strings.Split(m[1], ",") {
		if strings.EqualFold(strings.TrimSpace(name), "gosec") {
			return true
		}
	}
	return false
}
