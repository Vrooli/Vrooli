package validation

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// stubCommander is a Commander whose LookPath and Run results are scripted per
// binary, so scanner tests never invoke the real tools.
type stubCommander struct {
	present map[string]bool
	out     map[string]stubOut
}

type stubOut struct {
	stdout, stderr []byte
	exit           int
	err            error
}

func (s stubCommander) LookPath(name string) (string, error) {
	if s.present[name] {
		return "/usr/bin/" + name, nil
	}
	return "", errors.New("not found")
}

func (s stubCommander) Run(_ context.Context, _ string, name string, _ ...string) ([]byte, []byte, int, error) {
	o := s.out[name]
	return o.stdout, o.stderr, o.exit, o.err
}

func TestNormalizeSeverity(t *testing.T) {
	cases := map[string]Severity{
		"critical": SeverityError,
		"HIGH":     SeverityError,
		"high":     SeverityError,
		"moderate": SeverityWarning,
		"Medium":   SeverityWarning,
		"low":      SeverityInfo,
		"info":     SeverityInfo,
		"":         SeverityInfo,
		"weird":    SeverityInfo,
	}
	for raw, want := range cases {
		if got := NormalizeSeverity(raw); got != want {
			t.Errorf("NormalizeSeverity(%q) = %v, want %v", raw, got, want)
		}
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// newScenarioTree builds a fake repo root with one scenario containing a Go
// module, a pnpm UI, and a Python requirements file (unsupported).
func newScenarioTree(t *testing.T, id string) (repoRoot, scenarioDir string) {
	t.Helper()
	repoRoot = t.TempDir()
	scenarioDir = filepath.Join(repoRoot, "scenarios", id)
	writeFile(t, filepath.Join(scenarioDir, "api", "go.mod"), "module x\ngo 1.24\n")
	writeFile(t, filepath.Join(scenarioDir, "ui", "pnpm-lock.yaml"), "lockfileVersion: '9.0'\n")
	writeFile(t, filepath.Join(scenarioDir, "scripts", "requirements.txt"), "requests==2.0\n")
	// A vendored tree that must be ignored.
	writeFile(t, filepath.Join(scenarioDir, "ui", "node_modules", "dep", "go.mod"), "module ignored\n")
	return repoRoot, scenarioDir
}

func TestDetectSubstrate(t *testing.T) {
	_, dir := newScenarioTree(t, "demo")
	sub, err := DetectSubstrate(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !sub.Go {
		t.Error("expected Go substrate")
	}
	if !sub.PnpmUI {
		t.Error("expected PnpmUI substrate")
	}
	if len(sub.GoModDirs) != 1 || sub.GoModDirs[0] != "api" {
		t.Errorf("GoModDirs = %v, want [api] (node_modules must be skipped)", sub.GoModDirs)
	}
	if len(sub.PnpmLockDirs) != 1 || sub.PnpmLockDirs[0] != "ui" {
		t.Errorf("PnpmLockDirs = %v, want [ui]", sub.PnpmLockDirs)
	}
	wantUnsupported := false
	for _, u := range sub.Unsupported {
		if u == "python" {
			wantUnsupported = true
		}
	}
	if !wantUnsupported {
		t.Errorf("expected python in Unsupported, got %v", sub.Unsupported)
	}
}

func TestGitleaksScanner_RedactsAndErrors(t *testing.T) {
	cmd := stubCommander{
		present: map[string]bool{"gitleaks": true},
		out: map[string]stubOut{
			"gitleaks": {stdout: []byte(`[{"Description":"AWS","StartLine":2,"File":"leak.go","RuleID":"aws-access-token","Secret":"AKIA...","Match":"AKIA..."}]`)},
		},
	}
	sc := newGitleaksScanner(cmd)
	findings, err := sc.Scan(context.Background(), "/tmp/x", Substrate{})
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 {
		t.Fatalf("want 1 finding, got %d", len(findings))
	}
	f := findings[0]
	if f.Severity != SeverityError {
		t.Errorf("secret should be ERROR, got %v", f.Severity)
	}
	if f.FilePath != "leak.go:2" {
		t.Errorf("FilePath = %q, want leak.go:2", f.FilePath)
	}
	if f.RuleID != "gitleaks.aws-access-token" {
		t.Errorf("RuleID = %q", f.RuleID)
	}
	// The raw secret must never appear in any rendered field.
	for _, field := range []string{f.Title, f.Description, f.Remediation, f.FilePath} {
		if containsAKIA(field) {
			t.Errorf("secret leaked into finding field: %q", field)
		}
	}
}

func containsAKIA(s string) bool {
	for i := 0; i+4 <= len(s); i++ {
		if s[i:i+4] == "AKIA" {
			return true
		}
	}
	return false
}

func TestPnpmAuditScanner_Normalizes(t *testing.T) {
	cmd := stubCommander{
		present: map[string]bool{"pnpm": true},
		out: map[string]stubOut{
			"pnpm": {stdout: []byte(`{"advisories":{"1":{"id":1,"module_name":"esbuild","severity":"moderate","title":"dev server bug","url":"https://x","vulnerable_versions":"<=0.24.2","recommendation":"Upgrade to 0.25.0","github_advisory_id":"GHSA-67mh"}}}`)},
		},
	}
	sc := newPnpmAuditScanner(cmd)
	findings, err := sc.Scan(context.Background(), "/tmp/x", Substrate{PnpmUI: true, PnpmLockDirs: []string{"ui"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 {
		t.Fatalf("want 1 finding, got %d", len(findings))
	}
	f := findings[0]
	if f.Severity != SeverityWarning {
		t.Errorf("moderate should be WARNING, got %v", f.Severity)
	}
	if f.RuleID != "pnpm-audit.GHSA-67mh" {
		t.Errorf("RuleID = %q", f.RuleID)
	}
	if f.FilePath != "ui/pnpm-lock.yaml" {
		t.Errorf("FilePath = %q", f.FilePath)
	}
}

func TestGosecSeverity_ScopesG115(t *testing.T) {
	// G115 fires HIGH natively but is the fleet's noisiest FP — capped to WARNING.
	if got := gosecSeverity("G115", "HIGH"); got != SeverityWarning {
		t.Errorf("G115 HIGH should cap to WARNING, got %v", got)
	}
	// A non-scoped HIGH rule still gates.
	if got := gosecSeverity("G101", "HIGH"); got != SeverityError {
		t.Errorf("G101 HIGH should stay ERROR, got %v", got)
	}
	// The cap never escalates a lower native severity.
	if got := gosecSeverity("G115", "LOW"); got != SeverityInfo {
		t.Errorf("G115 LOW should stay INFO (cap never escalates), got %v", got)
	}
}

func TestGovulncheckScanner_StdlibIsWarningThirdPartyIsError(t *testing.T) {
	// Two reached findings: one in stdlib (toolchain-only fix → WARNING), one
	// in a third-party module (dep bump → ERROR).
	stream := `{"osv":{"id":"GO-2025-0001","summary":"stdlib bug"}}
{"osv":{"id":"GO-2025-0002","summary":"third-party bug"}}
{"finding":{"osv":"GO-2025-0001","trace":[{"module":"stdlib","function":"Parse"}]}}
{"finding":{"osv":"GO-2025-0002","trace":[{"module":"golang.org/x/net","function":"Do"}]}}`
	cmd := stubCommander{
		present: map[string]bool{"govulncheck": true},
		out:     map[string]stubOut{"govulncheck": {stdout: []byte(stream)}},
	}
	sc := newGovulncheckScanner(cmd)
	findings, err := sc.Scan(context.Background(), "/tmp/x", Substrate{Go: true, GoModDirs: []string{"api"}})
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]Severity{}
	for _, f := range findings {
		got[f.RuleID] = f.Severity
	}
	if got["govulncheck.GO-2025-0001"] != SeverityWarning {
		t.Errorf("stdlib vuln should be WARNING, got %v", got["govulncheck.GO-2025-0001"])
	}
	if got["govulncheck.GO-2025-0002"] != SeverityError {
		t.Errorf("third-party reachable vuln should be ERROR, got %v", got["govulncheck.GO-2025-0002"])
	}
}

func TestOSVScanner_IsAdvisoryInGate(t *testing.T) {
	// osv-scanner findings stay INFO in the gate even for a high-CVSS CVE —
	// govulncheck/pnpm-audit are the gating authorities.
	report := `{"results":[{"source":{"path":"go.mod"},"packages":[{"package":{"name":"foo","version":"1.0.0","ecosystem":"Go"},"vulnerabilities":[{"id":"GHSA-x","summary":"bad","database_specific":{"severity":"CRITICAL"}}],"groups":[{"ids":["GHSA-x"],"max_severity":"9.8"}]}]}]}`
	cmd := stubCommander{
		present: map[string]bool{"osv-scanner": true},
		out:     map[string]stubOut{"osv-scanner": {stdout: []byte(report)}},
	}
	sc := newOSVScanner(cmd)
	findings, err := sc.Scan(context.Background(), "/tmp/x", Substrate{Go: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 {
		t.Fatalf("want 1 finding, got %d", len(findings))
	}
	if findings[0].Severity != SeverityInfo {
		t.Errorf("osv findings must be advisory (INFO) in the gate, got %v", findings[0].Severity)
	}
}

// argsAwareCommander differentiates the full vs `--prod` pnpm audit calls so a
// dev-only advisory can be exercised.
type argsAwareCommander struct {
	full, prod []byte
}

func (c argsAwareCommander) LookPath(string) (string, error) { return "/usr/bin/pnpm", nil }

func (c argsAwareCommander) Run(_ context.Context, _ string, _ string, args ...string) ([]byte, []byte, int, error) {
	for _, a := range args {
		if a == "--prod" {
			return c.prod, nil, 0, nil
		}
	}
	return c.full, nil, 0, nil
}

func TestPnpmAuditScanner_DevOnlyCriticalDowngraded(t *testing.T) {
	// vitest critical appears in the full audit but NOT in the --prod audit, so
	// it is a dev-only vuln → downgraded to WARNING (advisory), not gating.
	full := `{"advisories":{"100":{"id":100,"module_name":"vitest","severity":"critical","title":"x","github_advisory_id":"GHSA-vitest"},"200":{"id":200,"module_name":"shipped","severity":"critical","title":"y","github_advisory_id":"GHSA-shipped"}}}`
	prod := `{"advisories":{"200":{"id":200,"module_name":"shipped","severity":"critical","title":"y","github_advisory_id":"GHSA-shipped"}}}`
	sc := newPnpmAuditScanner(argsAwareCommander{full: []byte(full), prod: []byte(prod)})
	findings, err := sc.Scan(context.Background(), "/tmp/x", Substrate{PnpmUI: true, PnpmLockDirs: []string{"ui"}})
	if err != nil {
		t.Fatal(err)
	}
	sev := map[string]Severity{}
	for _, f := range findings {
		sev[f.RuleID] = f.Severity
	}
	if sev["pnpm-audit.GHSA-vitest"] != SeverityWarning {
		t.Errorf("dev-only critical should be WARNING, got %v", sev["pnpm-audit.GHSA-vitest"])
	}
	if sev["pnpm-audit.GHSA-shipped"] != SeverityError {
		t.Errorf("production critical must stay ERROR, got %v", sev["pnpm-audit.GHSA-shipped"])
	}
}

func TestService_AbsentScannerIsSkippedNotFailed(t *testing.T) {
	repoRoot, _ := newScenarioTree(t, "demo")
	cmd := stubCommander{
		// gitleaks present and clean; gosec/govulncheck/osv/pnpm absent.
		present: map[string]bool{"gitleaks": true},
		out:     map[string]stubOut{"gitleaks": {stdout: []byte(`[]`)}},
	}
	svc := New(Deps{RepoRoot: repoRoot, Commander: cmd})
	rep, err := svc.ValidateScenario(context.Background(), "demo")
	if err != nil {
		t.Fatal(err)
	}
	if !rep.Passed {
		t.Errorf("absent scanners must not fail the scenario; passed=%v findings=%+v", rep.Passed, rep.Findings)
	}
	// gosec, govulncheck (Go), pnpm-audit, osv-scanner all apply but are absent.
	if len(rep.SkippedScanners) == 0 {
		t.Errorf("expected skipped scanners, got none")
	}
	if rep.Summary.Errors != 0 {
		t.Errorf("no ERROR findings expected, got %d", rep.Summary.Errors)
	}
}

func TestService_SecretFailsScenario(t *testing.T) {
	repoRoot, _ := newScenarioTree(t, "demo")
	cmd := stubCommander{
		present: map[string]bool{"gitleaks": true},
		out: map[string]stubOut{
			"gitleaks": {stdout: []byte(`[{"Description":"AWS","StartLine":2,"File":"leak.go","RuleID":"aws-access-token"}]`)},
		},
	}
	svc := New(Deps{RepoRoot: repoRoot, Commander: cmd})
	rep, err := svc.ValidateScenario(context.Background(), "demo")
	if err != nil {
		t.Fatal(err)
	}
	if rep.Passed {
		t.Error("a secret finding must fail the scenario (gates R1)")
	}
	if rep.Summary.Errors != 1 {
		t.Errorf("want 1 error, got %d", rep.Summary.Errors)
	}
	// ERROR findings sort first.
	if rep.Findings[0].Severity != SeverityError {
		t.Errorf("ERROR findings must sort first, got %v", rep.Findings[0].Severity)
	}
}

func TestService_UnknownScenarioSkipsGracefully(t *testing.T) {
	// A non-existent target is a graceful skip (WARNING, passed=true), mirroring
	// the cli-health sibling — so the optional test-genie security phase pointed
	// at a synthetic scenario never crashes the suite.
	repoRoot := t.TempDir()
	svc := New(Deps{RepoRoot: repoRoot, Commander: stubCommander{}})
	rep, err := svc.ValidateScenario(context.Background(), "ghost")
	if err != nil {
		t.Fatalf("missing scenario should skip, not error; got %v", err)
	}
	if !rep.Passed {
		t.Errorf("missing scenario must not fail (passed=%v)", rep.Passed)
	}
	if len(rep.Findings) != 1 || rep.Findings[0].RuleID != "security-health.scenario-not-found" {
		t.Errorf("expected a scenario-not-found warning, got %+v", rep.Findings)
	}
}

func TestService_EmptyScenarioErrors(t *testing.T) {
	svc := New(Deps{RepoRoot: t.TempDir(), Commander: stubCommander{}})
	if _, err := svc.ValidateScenario(context.Background(), "  "); err == nil {
		t.Error("expected error for empty scenario name")
	}
}

func TestService_DegradedScannerIsInfo(t *testing.T) {
	repoRoot, _ := newScenarioTree(t, "demo")
	cmd := stubCommander{
		present: map[string]bool{"gitleaks": true, "gosec": true},
		out: map[string]stubOut{
			"gitleaks": {stdout: []byte(`[]`)},
			// gosec installed but emits nothing parseable (toolchain incompat).
			"gosec": {stdout: nil, stderr: []byte("internal error: package fmt without types"), exit: 1},
		},
	}
	svc := New(Deps{RepoRoot: repoRoot, Commander: cmd})
	rep, err := svc.ValidateScenario(context.Background(), "demo")
	if err != nil {
		t.Fatal(err)
	}
	if !rep.Passed {
		t.Error("a degraded scanner must not fail the scenario")
	}
	found := false
	for _, f := range rep.Findings {
		if f.RuleID == "security-health.scanner-degraded" && f.Scanner == "gosec" {
			found = true
		}
	}
	if !found {
		t.Error("expected a degraded INFO observation for gosec")
	}
}
