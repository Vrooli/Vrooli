package validation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	CodeSecurityHeadersMissing = "security-health.security-headers-missing"
	CodeSecurityHeadersCORS    = "security-health.insecure-cors"
	CodeSecurityHeadersLegacy  = "security-health.security-headers-legacy-xss"
)

var requiredSecurityHeaders = map[string]string{
	"X-Content-Type-Options":    "nosniff",
	"X-Frame-Options":           "DENY",
	"X-XSS-Protection":          "0",
	"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
}

// runSecurityHeaderChecks validates first-party Go API response-header policy.
// It looks for a centralized router/middleware implementation rather than
// requiring every handler to remember the same headers independently.
func runSecurityHeaderChecks(scenarioDir string) ([]Finding, error) {
	apiDir := filepath.Join(scenarioDir, "api")
	if info, err := os.Stat(apiDir); err != nil || !info.IsDir() {
		return nil, nil
	}

	files, err := firstPartyGoFiles(apiDir)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 || !hasHTTPServerSurface(files) {
		return nil, nil
	}

	var findings []Finding
	headerEvidence := collectHeaderEvidence(files)
	missing := missingRequiredHeaders(headerEvidence)
	if len(missing) > 0 {
		findings = append(findings, Finding{
			RuleID:         CodeSecurityHeadersMissing,
			Severity:       SeverityError,
			Title:          "API security headers are not centralized",
			Description:    fmt.Sprintf("No first-party API middleware or server setup stamps the baseline security headers: %s.", strings.Join(missing, ", ")),
			Remediation:    "Add a router-level middleware that sets X-Content-Type-Options=nosniff, X-Frame-Options=DENY, X-XSS-Protection=0, and Strict-Transport-Security=max-age=31536000; includeSubDomains before handlers run.",
			FilePath:       bestHeaderAnchor(scenarioDir, files),
			Scanner:        "security-headers",
			Class:          FindingSAST,
			Owner:          "security-health",
			FixClass:       FixDeterministic,
			FixPreviewable: true,
			PolicyImpact:   "api-boundary-hardening",
		})
	}

	findings = append(findings, insecureCORSFindings(scenarioDir, files)...)
	if value, ok := headerEvidence["X-XSS-Protection"]; ok && value != "" && value != "0" {
		findings = append(findings, Finding{
			RuleID:         CodeSecurityHeadersLegacy,
			Severity:       SeverityWarning,
			Title:          "Legacy XSS auditor header enabled",
			Description:    fmt.Sprintf("X-XSS-Protection is set to %q. Modern browser guidance is to disable the legacy auditor with 0.", value),
			Remediation:    "Set X-XSS-Protection to 0 in the central security headers middleware.",
			FilePath:       bestHeaderAnchor(scenarioDir, files),
			Scanner:        "security-headers",
			Class:          FindingSAST,
			Owner:          "security-health",
			FixClass:       FixDeterministic,
			FixPreviewable: true,
			PolicyImpact:   "api-boundary-hardening",
		})
	}

	return findings, nil
}

type goSourceFile struct {
	path    string
	relPath string
	content string
}

func firstPartyGoFiles(apiDir string) ([]goSourceFile, error) {
	var files []goSourceFile
	err := filepath.WalkDir(apiDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if _, skip := skipDirs[d.Name()]; skip {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".go") || strings.HasSuffix(d.Name(), "_test.go") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(filepath.Dir(apiDir), path)
		files = append(files, goSourceFile{path: path, relPath: filepath.ToSlash(rel), content: string(data)})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(files, func(i, j int) bool { return files[i].relPath < files[j].relPath })
	return files, nil
}

func hasHTTPServerSurface(files []goSourceFile) bool {
	for _, f := range files {
		if strings.Contains(f.content, "net/http") ||
			strings.Contains(f.content, "http.Handler") ||
			strings.Contains(f.content, "mux.NewRouter") ||
			strings.Contains(f.content, "connectx.RegisterServices") {
			return true
		}
	}
	return false
}

var headerSetRE = regexp.MustCompile(`(?s)\bSet\(\s*"([^"]+)"\s*,\s*"([^"]*)"`)

func collectHeaderEvidence(files []goSourceFile) map[string]string {
	evidence := map[string]string{}
	for _, f := range files {
		for _, match := range headerSetRE.FindAllStringSubmatch(f.content, -1) {
			if len(match) != 3 {
				continue
			}
			evidence[match[1]] = match[2]
		}
	}
	return evidence
}

func missingRequiredHeaders(evidence map[string]string) []string {
	var missing []string
	for header, want := range requiredSecurityHeaders {
		if got, ok := evidence[header]; !ok || strings.TrimSpace(got) == "" {
			missing = append(missing, header)
		} else if header == "X-XSS-Protection" && got != want {
			// Report legacy value separately; the header is at least considered.
			continue
		}
	}
	sort.Strings(missing)
	return missing
}

func bestHeaderAnchor(scenarioDir string, files []goSourceFile) string {
	preferred := []string{
		"api/internal/server/server.go",
		"api/server.go",
		"api/main.go",
	}
	for _, want := range preferred {
		for _, f := range files {
			if f.relPath == want {
				return want
			}
		}
	}
	if len(files) == 0 {
		return relPath(scenarioDir, scenarioDir)
	}
	return files[0].relPath
}

func insecureCORSFindings(scenarioDir string, files []goSourceFile) []Finding {
	var findings []Finding
	for _, f := range files {
		if !strings.Contains(f.content, "Access-Control-Allow-Origin") ||
			!strings.Contains(f.content, "Access-Control-Allow-Credentials") {
			continue
		}
		if !regexp.MustCompile(`Access-Control-Allow-Origin"\s*,\s*"\*"`).MatchString(f.content) {
			continue
		}
		if !regexp.MustCompile(`Access-Control-Allow-Credentials"\s*,\s*"true"`).MatchString(f.content) {
			continue
		}
		findings = append(findings, Finding{
			RuleID:       CodeSecurityHeadersCORS,
			Severity:     SeverityError,
			Title:        "Wildcard CORS allows credentials",
			Description:  "The API sets Access-Control-Allow-Origin to * while also allowing credentials, which browsers reject and which represents an unsafe origin policy.",
			Remediation:  "Replace wildcard credentialed CORS with an explicit allowlist and emit the matched origin only after validation, or disable credentials for wildcard responses.",
			FilePath:     relPath(scenarioDir, f.path),
			Scanner:      "security-headers",
			Class:        FindingSAST,
			Owner:        "security-health",
			FixClass:     FixManual,
			PolicyImpact: "cors-policy-review-required",
		})
	}
	return findings
}

type SecurityHeaderFixCandidate struct {
	RuleID        string
	FilePath      string
	Description   string
	Before        string
	After         string
	Applied       bool
	Owner         string
	FixClass      FixClass
	Idempotent    bool
	Rollback      string
	Validation    string
	PreviewDigest string
}

func (s *Service) PreviewFix(ctx context.Context, scenario, path string, ruleIDs []string) (string, []SecurityHeaderFixCandidate, []string, error) {
	return s.fixSecurityHeaders(ctx, scenario, path, ruleIDs, false, nil)
}

func (s *Service) ApplyFix(ctx context.Context, scenario, path string, ruleIDs []string) (string, []SecurityHeaderFixCandidate, []string, error) {
	_, preview, _, err := s.fixSecurityHeaders(ctx, scenario, path, ruleIDs, false, nil)
	if err != nil {
		return "", nil, nil, err
	}
	expected := make(map[string]string, len(preview))
	for _, candidate := range preview {
		expected[candidate.FilePath] = candidate.PreviewDigest
	}
	return s.fixSecurityHeaders(ctx, scenario, path, ruleIDs, true, expected)
}

// ApplyFixWithPreviewDigests applies only the exact files and bytes returned by
// PreviewFix. It is the bridge for API clients that persist a preview between
// user confirmation and apply; a changed target is rejected.
func (s *Service) ApplyFixWithPreviewDigests(ctx context.Context, scenario, path string, ruleIDs []string, expected map[string]string) (string, []SecurityHeaderFixCandidate, []string, error) {
	if len(expected) == 0 {
		return "", nil, nil, errors.New("preview digests are required")
	}
	return s.fixSecurityHeaders(ctx, scenario, path, ruleIDs, true, expected)
}

func (s *Service) fixSecurityHeaders(_ context.Context, scenario, path string, ruleIDs []string, apply bool, expected map[string]string) (string, []SecurityHeaderFixCandidate, []string, error) {
	scenario = strings.TrimSpace(scenario)
	root := strings.TrimSpace(path)
	if root == "" {
		if scenario == "" {
			return "", nil, nil, errors.New("scenario or path is required")
		}
		resolved, ok := resolveScenarioDir(s.repoRoot, scenario)
		if !ok {
			return "", nil, nil, fmt.Errorf("scenario %q not found under scenarios/", scenario)
		}
		root = resolved
	}
	if scenario == "" {
		var ok bool
		scenario, ok = scenarioNameForPath(s.repoRoot, root)
		if !ok {
			return "", nil, nil, errors.New("fix path must be inside a known scenario root")
		}
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return "", nil, nil, fmt.Errorf("resolve fix root: %w", err)
	}
	scenarioRoot, ok := resolveScenarioDir(s.repoRoot, scenario)
	if !ok {
		return "", nil, nil, fmt.Errorf("scenario %q not found under scenarios/", scenario)
	}
	scenarioRoot, _ = filepath.Abs(scenarioRoot)
	if !pathWithin(root, scenarioRoot) {
		return "", nil, nil, errors.New("fix path must remain inside the requested scenario")
	}

	allow := securityHeaderRuleAllowed(ruleIDs)
	var messages []string
	if !allow(CodeSecurityHeadersMissing) {
		return scenario, nil, []string{"requested rule id(s) have no deterministic Security Health fix"}, nil
	}
	candidates, err := securityHeaderFixCandidates(root)
	if err != nil {
		return "", nil, nil, err
	}
	if len(candidates) == 0 {
		return scenario, nil, []string{"no safe deterministic Security Health fix available for requested rule id(s)"}, nil
	}
	for i := range candidates {
		candidates[i].PreviewDigest = repairPreviewDigest(candidates[i])
	}
	if apply {
		for _, candidate := range candidates {
			if expected == nil || expected[candidate.FilePath] == "" || expected[candidate.FilePath] != candidate.PreviewDigest {
				return "", nil, nil, fmt.Errorf("preview digest mismatch for %s; re-run preview", candidate.FilePath)
			}
		}
		original := make(map[string][]byte, len(candidates))
		existed := make(map[string]bool, len(candidates))
		for i := range candidates {
			before, readErr := os.ReadFile(candidates[i].FilePath)
			if readErr == nil {
				original[candidates[i].FilePath] = before
				existed[candidates[i].FilePath] = true
			} else if !errors.Is(readErr, os.ErrNotExist) {
				return "", nil, nil, readErr
			}
		}
		for i := range candidates {
			if err := atomicWriteText(candidates[i].FilePath, candidates[i].After); err != nil {
				rollbackSecurityFixes(original, existed, candidates)
				return "", nil, nil, fmt.Errorf("apply deterministic fix and rollback: %w", err)
			}
			candidates[i].Applied = true
		}
		postFindings, validationErr := runSecurityHeaderChecks(scenarioRoot)
		if validationErr != nil || hasFinding(postFindings, CodeSecurityHeadersMissing) {
			rollbackSecurityFixes(original, existed, candidates)
			if validationErr != nil {
				return "", nil, nil, fmt.Errorf("post-repair validation failed and changes were rolled back: %w", validationErr)
			}
			return "", nil, nil, errors.New("post-repair validation still reports missing security headers; changes were rolled back")
		}
	}
	return scenario, candidates, messages, nil
}

func rollbackSecurityFixes(original map[string][]byte, existed map[string]bool, candidates []SecurityHeaderFixCandidate) {
	for _, candidate := range candidates {
		if existed[candidate.FilePath] {
			_ = atomicWriteText(candidate.FilePath, string(original[candidate.FilePath]))
			continue
		}
		_ = os.Remove(candidate.FilePath)
	}
}

func repairPreviewDigest(candidate SecurityHeaderFixCandidate) string {
	sum := sha256.Sum256([]byte(candidate.FilePath + "\x00" + candidate.Before + "\x00" + candidate.After + "\x00" + string(candidate.FixClass)))
	return hex.EncodeToString(sum[:])
}

func pathWithin(path, root string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func scenarioNameForPath(repoRoot, path string) (string, bool) {
	root, err := filepath.Abs(filepath.Join(repoRoot, "scenarios"))
	if err != nil {
		return "", false
	}
	path, err = filepath.Abs(path)
	if err != nil || !pathWithin(path, root) {
		return "", false
	}
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == "." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	parts := strings.Split(filepath.Clean(rel), string(filepath.Separator))
	if len(parts) == 0 || parts[0] == "." || parts[0] == ".." {
		return "", false
	}
	return parts[0], true
}

func hasFinding(findings []Finding, ruleID string) bool {
	for _, finding := range findings {
		if finding.RuleID == ruleID {
			return true
		}
	}
	return false
}

func atomicWriteText(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmpFile, err := os.CreateTemp(filepath.Dir(path), ".security-health-*.tmp")
	if err != nil {
		return err
	}
	tmp := tmpFile.Name()
	defer os.Remove(tmp)
	if err := tmpFile.Chmod(0o644); err != nil {
		_ = tmpFile.Close()
		return err
	}
	if _, err := tmpFile.WriteString(content); err != nil {
		_ = tmpFile.Close()
		return err
	}
	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func securityHeaderRuleAllowed(ruleIDs []string) func(string) bool {
	if len(ruleIDs) == 0 {
		return func(string) bool { return true }
	}
	allowed := map[string]bool{}
	for _, id := range ruleIDs {
		allowed[strings.TrimSpace(id)] = true
	}
	return func(rule string) bool { return allowed[rule] }
}

func securityHeaderFixCandidates(root string) ([]SecurityHeaderFixCandidate, error) {
	serverPath := filepath.Join(root, "api", "internal", "server", "server.go")
	beforeServer, err := os.ReadFile(serverPath)
	if err != nil {
		return nil, nil
	}
	serverAfter := string(beforeServer)
	if !strings.Contains(serverAfter, "internal/middleware") {
		return nil, nil
	}
	if !strings.Contains(serverAfter, "NewSecurityHeadersMiddleware") {
		marker := "s.router.Use("
		idx := strings.Index(serverAfter, marker)
		if idx == -1 {
			return nil, nil
		}
		serverAfter = serverAfter[:idx] + "s.router.Use(middleware.NewSecurityHeadersMiddleware())\n\t" + serverAfter[idx:]
	}

	middlewarePath := filepath.Join(root, "api", "internal", "middleware", "securityheaders.go")
	beforeMiddlewareBytes, _ := os.ReadFile(middlewarePath)
	beforeMiddleware := string(beforeMiddlewareBytes)
	afterMiddleware := canonicalSecurityHeadersMiddleware()

	var out []SecurityHeaderFixCandidate
	if beforeMiddleware != afterMiddleware {
		out = append(out, SecurityHeaderFixCandidate{
			RuleID:      CodeSecurityHeadersMissing,
			FilePath:    middlewarePath,
			Description: "Create or normalize the central API security headers middleware.",
			Before:      beforeMiddleware,
			After:       afterMiddleware,
			Owner:       "security-health",
			FixClass:    FixDeterministic,
			Idempotent:  true,
			Rollback:    "restore the preview before bytes",
			Validation:  "go test ./api/... && security-health validate scenario",
		})
	}
	if string(beforeServer) != serverAfter {
		out = append(out, SecurityHeaderFixCandidate{
			RuleID:      CodeSecurityHeadersMissing,
			FilePath:    serverPath,
			Description: "Register the security headers middleware before request handlers run.",
			Before:      string(beforeServer),
			After:       serverAfter,
			Owner:       "security-health",
			FixClass:    FixDeterministic,
			Idempotent:  true,
			Rollback:    "restore the preview before bytes",
			Validation:  "go test ./api/... && security-health validate scenario",
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].FilePath < out[j].FilePath })
	return out, nil
}

func canonicalSecurityHeadersMiddleware() string {
	return `package middleware

import "net/http"

// NewSecurityHeadersMiddleware stamps baseline browser security headers on
// every API response. These headers are intentionally centralized at the
// router boundary so REST handlers, Connect handlers, and health probes share
// the same default posture.
//
// CORS is deliberately not set here. Allowed origins, methods, and credential
// behavior are scenario-specific policy and must be handled by a dedicated CORS
// middleware when a scenario actually exposes cross-origin browser traffic.
func NewSecurityHeadersMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-Frame-Options", "DENY")
			h.Set("X-XSS-Protection", "0")
			h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			next.ServeHTTP(w, r)
		})
	}
}
`
}
