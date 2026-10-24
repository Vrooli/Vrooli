package validation

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
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

type argsCapturingCommander struct {
	stubCommander
	args []string
}

func (c *argsCapturingCommander) Run(ctx context.Context, dir, name string, args ...string) ([]byte, []byte, int, error) {
	c.args = append([]string(nil), args...)
	return c.stubCommander.Run(ctx, dir, name, args...)
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

// newScenarioTree builds a fake repo root with one scenario containing Go,
// JavaScript, and Python dependency surfaces in nested directories.
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
	for _, target := range sub.Targets {
		if target.Ecosystem == EcosystemPython && target.Coverage != CoverageSupported {
			t.Errorf("python target coverage = %q, want supported", target.Coverage)
		}
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
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "api", "go.mod"), "module example.com/demo\n")
	report := `{"results":[{"source":{"path":"go.mod"},"packages":[{"package":{"name":"foo","version":"1.0.0","ecosystem":"Go"},"vulnerabilities":[{"id":"GHSA-x","summary":"bad","database_specific":{"severity":"CRITICAL"}}],"groups":[{"ids":["GHSA-x"],"max_severity":"9.8"}]}]}]}`
	cmd := stubCommander{
		present: map[string]bool{"osv-scanner": true},
		out:     map[string]stubOut{"osv-scanner": {stdout: []byte(report)}},
	}
	sc := newOSVScanner(cmd)
	findings, err := sc.Scan(context.Background(), dir, Substrate{Go: true})
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

func TestOSVScanner_ScansOnlyFirstPartyLockfiles(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "api", "go.mod"), "module example.com/demo\n")
	writeFile(t, filepath.Join(dir, "ui", "pnpm-lock.yaml"), "lockfileVersion: '9.0'\n")
	writeFile(t, filepath.Join(dir, "ui", "node_modules", "fixture", "package-lock.json"), "{}\n")
	cmd := &argsCapturingCommander{stubCommander: stubCommander{
		present: map[string]bool{"osv-scanner": true},
		out:     map[string]stubOut{"osv-scanner": {stdout: []byte(`{"results":[]}`)}},
	}}
	if _, err := newOSVScanner(cmd).Scan(context.Background(), dir, Substrate{Go: true, PnpmUI: true}); err != nil {
		t.Fatal(err)
	}
	got := strings.Join(cmd.args, " ")
	if strings.Contains(got, " -r ") || strings.Contains(got, " .") {
		t.Fatalf("osv scanner must not recurse across the scenario: %v", cmd.args)
	}
	if strings.Contains(got, "node_modules") {
		t.Fatalf("osv scanner must not scan installed dependency fixtures: %v", cmd.args)
	}
	for _, want := range []string{"api/go.mod", "ui/pnpm-lock.yaml"} {
		if !strings.Contains(got, want) {
			t.Errorf("scanner arguments %v missing first-party lockfile %s", cmd.args, want)
		}
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

func TestPnpmAuditScanner_RSCOnlyAdvisoryRequiresRSCUsage(t *testing.T) {
	full := `{"advisories":{"rsc":{"id":1,"module_name":"react-router","severity":"high","title":"RSC-only CSRF","github_advisory_id":"GHSA-qwww-vcr4-c8h2"}}}`
	prod := full
	cases := []struct {
		name    string
		source  string
		wantSev Severity
	}{
		{
			name:    "browser router does not enable unstable RSC APIs",
			source:  `import { BrowserRouter } from "react-router-dom"; const label = "underscores"; export default BrowserRouter;`,
			wantSev: SeverityWarning,
		},
		{
			name:    "unstable RSC API remains blocking",
			source:  `import { RSCHydratedRouter } from "react-router-rsc"; export default RSCHydratedRouter;`,
			wantSev: SeverityError,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, filepath.Join(dir, "ui", "src", "App.tsx"), tc.source)
			sc := newPnpmAuditScanner(argsAwareCommander{full: []byte(full), prod: []byte(prod)})
			findings, err := sc.Scan(context.Background(), dir, Substrate{PnpmUI: true, PnpmLockDirs: []string{"ui"}})
			if err != nil {
				t.Fatal(err)
			}
			if len(findings) != 1 {
				t.Fatalf("want 1 finding, got %d", len(findings))
			}
			if findings[0].Severity != tc.wantSev {
				t.Errorf("severity = %v, want %v", findings[0].Severity, tc.wantSev)
			}
		})
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

func TestValidateScenarioAndValidateTargetReportsAreByteIdentical(t *testing.T) {
	repoRoot, scenarioDir := newScenarioTree(t, "demo")
	cmd := stubCommander{
		present: map[string]bool{"gitleaks": true},
		out:     map[string]stubOut{"gitleaks": {stdout: []byte(`[]`)}},
	}
	svc := New(Deps{RepoRoot: repoRoot, Commander: cmd})

	legacy, err := svc.ValidateScenario(context.Background(), "demo")
	if err != nil {
		t.Fatal(err)
	}
	target, err := svc.ValidateTarget(context.Background(), ValidationTargetScenario, scenarioDir)
	if err != nil {
		t.Fatal(err)
	}
	legacyJSON, err := json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	targetJSON, err := json.Marshal(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(legacyJSON) != string(targetJSON) {
		t.Fatalf("scenario and target reports differ:\nscenario: %s\ntarget:   %s", legacyJSON, targetJSON)
	}
}

func TestControlPlaneSubstrateKeepsRootAndPackageModules(t *testing.T) {
	sub := controlPlaneSubstrate(Substrate{
		Go:           true,
		PnpmUI:       true,
		GoModDirs:    []string{".", "packages/api-core", "scenarios/demo/api", "templates/scenarios/react-vite/api"},
		PnpmLockDirs: []string{"packages/ui/pnpm-lock.yaml", "scenarios/demo/ui"},
	})
	wantRoots := ".,packages/api-core"
	if got := strings.Join(sub.GoModDirs, ","); got != wantRoots {
		t.Fatalf("control-plane Go scan roots = %q, want %q", got, wantRoots)
	}
	if !sub.Go || sub.PnpmUI || len(sub.PnpmLockDirs) != 0 {
		t.Fatalf("unexpected control-plane substrate: %+v", sub)
	}
	if got := strings.Join(sub.GoPackagePatterns["."], ","); got != "./internal/...,./cmd/..." {
		t.Fatalf("root module package patterns = %q", got)
	}
	for _, name := range []string{"gosec", "govulncheck"} {
		if !controlPlaneScanner(name) {
			t.Fatalf("control-plane scanner %q unexpectedly disabled", name)
		}
	}
	for _, name := range []string{"gitleaks", "pnpm-audit", "osv-scanner"} {
		if controlPlaneScanner(name) {
			t.Fatalf("control-plane scanner %q unexpectedly enabled", name)
		}
	}
}

func TestControlPlaneErrorBudgetPassesAtBaselineAndFailsRegression(t *testing.T) {
	budget := ErrorBudget{Limit: 52, Baseline: 52, Ratchet: true, Declared: true}
	if !budget.allows(52) {
		t.Fatal("measured baseline must pass")
	}
	if budget.allows(53) {
		t.Fatal("one new blocking finding must fail")
	}
	if (ErrorBudget{Limit: 53, Baseline: 52, Ratchet: true, Declared: true}).allows(51) {
		t.Fatal("loosening a ratcheted budget must fail even below the baseline")
	}
	if !(ErrorBudget{Limit: 0, Baseline: 0, Ratchet: true, Declared: true}).allows(0) {
		t.Fatal("an explicit clean budget must pass a clean scan")
	}
}

func TestDedupeControlPlaneFindingsCollapsesOverlappingScanRoots(t *testing.T) {
	finding := Finding{Scanner: "gosec", RuleID: "gosec.G404", FilePath: "internal/credentialescrow/random.go:12", Description: "weak random number generator"}
	got := dedupeControlPlaneFindings([]Finding{finding, finding, {Scanner: "gosec", RuleID: "gosec.G401", FilePath: "internal/other.go:3"}})
	if len(got) != 2 {
		t.Fatalf("deduplicated findings = %d, want 2: %+v", len(got), got)
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

// [REQ:REQ-P0-020]
func TestService_SecurityHeadersMissingFailsScenario(t *testing.T) {
	repoRoot, dir := newScenarioTree(t, "demo")
	writeFile(t, filepath.Join(dir, "api", "internal", "server", "server.go"), `package server

import (
	"net/http"

	"github.com/gorilla/mux"
)

func New() http.Handler {
	r := mux.NewRouter()
	r.HandleFunc("/api/v1/demo", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	return r
}
`)
	cmd := stubCommander{
		present: map[string]bool{"gitleaks": true},
		out:     map[string]stubOut{"gitleaks": {stdout: []byte(`[]`)}},
	}
	svc := New(Deps{RepoRoot: repoRoot, Commander: cmd})
	rep, err := svc.ValidateScenario(context.Background(), "demo")
	if err != nil {
		t.Fatal(err)
	}
	if rep.Passed {
		t.Fatalf("missing security headers must fail the scenario: %+v", rep.Findings)
	}
	found := false
	for _, f := range rep.Findings {
		if f.RuleID == CodeSecurityHeadersMissing && f.Severity == SeverityError {
			found = true
			if f.Scanner != "security-headers" {
				t.Fatalf("scanner = %q, want security-headers", f.Scanner)
			}
		}
	}
	if !found {
		t.Fatalf("expected %s finding, got %+v", CodeSecurityHeadersMissing, rep.Findings)
	}
}

// [REQ:REQ-P0-020]
func TestSecurityHeadersCheckFlagsOnlyUnsafeCORS(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "api", "main.go"), `package main

import "net/http"

func securityHeaders(w http.ResponseWriter) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-XSS-Protection", "0")
	w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
}
`)
	findings, err := runSecurityHeaderChecks(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 || findings[0].RuleID != CodeSecurityHeadersCORS {
		t.Fatalf("expected only unsafe CORS finding, got %+v", findings)
	}
}

// [REQ:REQ-P0-021]
func TestSecurityHeadersFixPreviewAndApplyGeneratedServerShape(t *testing.T) {
	repoRoot, dir := newScenarioTree(t, "demo")
	serverPath := filepath.Join(dir, "api", "internal", "server", "server.go")
	writeFile(t, serverPath, `package server

import (
	"log"

	"demo/internal/middleware"

	"github.com/gorilla/mux"
)

func New(logger *log.Logger) *Server {
	s := &Server{router: mux.NewRouter()}
	s.router.Use(middleware.NewLoggingMiddleware(nil, logger))
	return s
}

`)
	svc := New(Deps{RepoRoot: repoRoot, Commander: stubCommander{}})
	scenario, candidates, messages, err := svc.PreviewFix(context.Background(), "demo", "", []string{CodeSecurityHeadersMissing})
	if err != nil {
		t.Fatal(err)
	}
	if scenario != "demo" || len(messages) != 0 {
		t.Fatalf("scenario/messages = %q/%v", scenario, messages)
	}
	if len(candidates) != 2 {
		t.Fatalf("expected middleware+server candidates, got %d: %+v", len(candidates), candidates)
	}
	if _, err := os.Stat(filepath.Join(dir, "api", "internal", "middleware", "securityheaders.go")); !os.IsNotExist(err) {
		t.Fatalf("preview should not write securityheaders.go, stat err=%v", err)
	}

	_, applied, _, err := svc.ApplyFix(context.Background(), "demo", "", []string{CodeSecurityHeadersMissing})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range applied {
		if !c.Applied {
			t.Fatalf("applied candidate not marked applied: %+v", c)
		}
	}
	serverAfter, err := os.ReadFile(serverPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(serverAfter), "NewSecurityHeadersMiddleware") {
		t.Fatalf("ApplyFix did not register security middleware:\n%s", serverAfter)
	}
}

func TestSecurityHeadersApplyRejectsChangedPreview(t *testing.T) {
	repoRoot, dir := newScenarioTree(t, "demo")
	serverPath := filepath.Join(dir, "api", "internal", "server", "server.go")
	writeFile(t, serverPath, `package server
import "demo/internal/middleware"
func New() { _ = middleware.NewSecurityHeadersMiddleware }
`)
	svc := New(Deps{RepoRoot: repoRoot, Commander: stubCommander{}})
	_, candidates, _, err := svc.PreviewFix(context.Background(), "demo", "", []string{CodeSecurityHeadersMissing})
	if err != nil || len(candidates) == 0 {
		t.Fatalf("preview = %v %+v", err, candidates)
	}
	if err := os.MkdirAll(filepath.Dir(candidates[0].FilePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(candidates[0].FilePath, []byte("changed"), 0o644); err != nil {
		t.Fatal(err)
	}
	expected := map[string]string{}
	for _, candidate := range candidates {
		expected[candidate.FilePath] = candidate.PreviewDigest
	}
	if _, _, _, err := svc.ApplyFixWithPreviewDigests(context.Background(), "demo", "", []string{CodeSecurityHeadersMissing}, expected); err == nil {
		t.Fatal("changed preview target was accepted")
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

// gitAwareCommander scripts gitleaks output plus the exact commit-eligible
// inventory returned by git ls-files.
type gitAwareCommander struct {
	gitleaks    []byte
	checkIgnore []byte
	lsFiles     []byte
}

func (c gitAwareCommander) LookPath(name string) (string, error) { return "/usr/bin/" + name, nil }

func (c gitAwareCommander) Run(_ context.Context, _ string, name string, args ...string) ([]byte, []byte, int, error) {
	switch {
	case name == "gitleaks":
		return c.gitleaks, nil, 1, nil
	case name == "git" && len(args) > 0 && args[0] == "check-ignore":
		return c.checkIgnore, nil, 0, nil
	case name == "git" && len(args) > 0 && args[0] == "ls-files":
		return c.lsFiles, nil, 0, nil
	}
	return nil, nil, 0, nil
}

func TestGitleaksScanner_SnapshotFindingsGate(t *testing.T) {
	// A scanner finding can only originate from the staged commit-eligible
	// inventory, so every normalized secret is gating.
	cmd := gitAwareCommander{
		gitleaks: []byte(`[
			{"Description":"Generic API Key","StartLine":5,"File":".env","RuleID":"generic-api-key"},
			{"Description":"Generic API Key","StartLine":21,"File":".build-fingerprint.json","RuleID":"generic-api-key"},
			{"Description":"AWS","StartLine":2,"File":"api/cfg.go","RuleID":"aws-access-token"}
		]`),
		checkIgnore: []byte(".env\n.build-fingerprint.json\n"),
		lsFiles:     []byte("api/cfg.go\x00"),
	}
	sc := newGitleaksScanner(cmd)
	findings, err := sc.Scan(context.Background(), "/tmp/x", Substrate{})
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 3 {
		t.Fatalf("want 3 findings (downgraded, not dropped), got %d", len(findings))
	}
	sev := map[string]Severity{}
	for _, f := range findings {
		sev[f.FilePath] = f.Severity
	}
	if sev[".env:5"] != SeverityError {
		t.Errorf("snapshot finding should be ERROR, got %v", sev[".env:5"])
	}
	if sev[".build-fingerprint.json:21"] != SeverityError {
		t.Errorf("snapshot finding should be ERROR, got %v", sev[".build-fingerprint.json:21"])
	}
	if sev["api/cfg.go:2"] != SeverityError {
		t.Errorf("tracked file must stay ERROR, got %v", sev["api/cfg.go:2"])
	}
}

func TestGitleaksScanner_TrackedDespiteIgnoreRuleStaysError(t *testing.T) {
	// A file force-added past .gitignore matches the ignore rules AND is
	// tracked: committed content must keep gating.
	cmd := gitAwareCommander{
		gitleaks:    []byte(`[{"Description":"AWS","StartLine":3,"File":".env","RuleID":"aws-access-token"}]`),
		checkIgnore: []byte(".env\n"),
		lsFiles:     []byte(".env\x00"),
	}
	sc := newGitleaksScanner(cmd)
	findings, err := sc.Scan(context.Background(), "/tmp/x", Substrate{})
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 || findings[0].Severity != SeverityError {
		t.Fatalf("tracked-but-ignore-matched secret must stay ERROR, got %+v", findings)
	}
}

func TestGitleaksScanner_GitUnavailableFailsOpen(t *testing.T) {
	// stubCommander has no "git" entry: empty output. All findings must keep
	// their native ERROR severity (fail open).
	cmd := stubCommander{
		present: map[string]bool{"gitleaks": true},
		out: map[string]stubOut{
			"gitleaks": {stdout: []byte(`[{"Description":"AWS","StartLine":2,"File":".env","RuleID":"aws-access-token"}]`)},
		},
	}
	sc := newGitleaksScanner(cmd)
	findings, err := sc.Scan(context.Background(), "/tmp/x", Substrate{})
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 || findings[0].Severity != SeverityError {
		t.Fatalf("without git the finding must stay ERROR, got %+v", findings)
	}
}

func TestGosecScanner_NolintGosecSuppressed(t *testing.T) {
	// Two issues: one whose flagged line carries //nolint:gosec (reviewed,
	// must be dropped — standalone gosec only honors #nosec), one bare.
	report := `{"Issues":[
		{"severity":"HIGH","rule_id":"G404","details":"weak rng","file":"/x/api/sel.go","line":"464",
		 "code":"463: func pick() {\n464: \trng := rand.New(rand.NewSource(1)) //nolint:gosec // deterministic exploration\n465: }"},
		{"severity":"HIGH","rule_id":"G404","details":"weak rng","file":"/x/api/other.go","line":"10",
		 "code":"9: func roll() {\n10: \tn := rand.Intn(6)\n11: }"}
	]}`
	cmd := stubCommander{
		present: map[string]bool{"gosec": true},
		out:     map[string]stubOut{"gosec": {stdout: []byte(report)}},
	}
	sc := newGosecScanner(cmd)
	findings, err := sc.Scan(context.Background(), "/x", Substrate{Go: true, GoModDirs: []string{"api"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 {
		t.Fatalf("want 1 finding (nolint:gosec suppressed), got %d: %+v", len(findings), findings)
	}
	if findings[0].FilePath != "api/other.go:10" {
		t.Errorf("surviving finding = %q, want api/other.go:10", findings[0].FilePath)
	}
}

func TestNolintSuppressesGosec(t *testing.T) {
	cases := map[string]bool{
		"x := rand.Intn(6) //nolint:gosec":           true,
		"x := rand.Intn(6) //nolint:gosec // reason": true,
		"x := rand.Intn(6) //nolint:errcheck,gosec":  true,
		"x := rand.Intn(6) //nolint":                 true,
		"x := rand.Intn(6) // nolint:gosec":          true,
		"x := rand.Intn(6) //nolint:errcheck":        false,
		"x := rand.Intn(6)":                          false,
		"x := rand.Intn(6) // no lint here":          false,
	}
	for src, want := range cases {
		if got := nolintSuppressesGosec(src); got != want {
			t.Errorf("nolintSuppressesGosec(%q) = %v, want %v", src, got, want)
		}
	}
}

func TestFlaggedSourceLine(t *testing.T) {
	code := "9: a\n10: \tflagged()\n11: c"
	if got := flaggedSourceLine(code, "10"); !containsSub(got, "flagged()") {
		t.Errorf("line 10 = %q, want flagged()", got)
	}
	if got := flaggedSourceLine(code, "10-12"); !containsSub(got, "flagged()") {
		t.Errorf("range 10-12 should resolve to first line, got %q", got)
	}
	if got := flaggedSourceLine(code, "99"); got != "" {
		t.Errorf("missing line should return empty, got %q", got)
	}
	if got := flaggedSourceLine("", "10"); got != "" {
		t.Errorf("empty snippet should return empty, got %q", got)
	}
}

func containsSub(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// TestCommitEligibleFiles_RealGit pins the actual source boundary against a
// real worktree: tracked, force-added, and ordinary untracked files are in;
// ignored local/runtime files are out.
func TestCommitEligibleFiles_RealGit(t *testing.T) {
	if _, err := NewExecCommander().LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		if _, _, code, err := NewExecCommander().Run(context.Background(), dir, "git", args...); err != nil || code != 0 {
			t.Fatalf("git %v: code=%d err=%v", args, code, err)
		}
	}
	run("init", "-q")
	writeFile(t, filepath.Join(dir, ".gitignore"), ".env\nforced.txt\n")
	writeFile(t, filepath.Join(dir, ".env"), "SECRET=x\n")
	writeFile(t, filepath.Join(dir, "tracked.go"), "package x\n")
	writeFile(t, filepath.Join(dir, "forced.txt"), "SECRET=y\n")
	writeFile(t, filepath.Join(dir, "untracked.go"), "package x\n")
	run("add", ".gitignore", "tracked.go")
	run("add", "-f", "forced.txt") // tracked despite matching .gitignore

	files, err := commitEligibleFiles(context.Background(), NewExecCommander(), dir)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, path := range files {
		rel, _ := filepath.Rel(dir, path)
		got[filepath.ToSlash(rel)] = true
	}
	if got[".env"] {
		t.Error("gitignored .env must not enter the scan inventory")
	}
	for _, want := range []string{".gitignore", "tracked.go", "forced.txt", "untracked.go"} {
		if !got[want] {
			t.Errorf("commit-eligible %s missing from inventory: %v", want, got)
		}
	}
}
