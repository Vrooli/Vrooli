package audit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"quality-health/internal/autofix"
	"quality-health/internal/commands"
	"quality-health/internal/contracts"
	"quality-health/internal/surfaces"
)

type Service struct {
	Discoverer surfaces.Discoverer
	Locator    surfaces.Locator
	Executor   commands.Executor
	Now        func() time.Time
}

type Request struct {
	Scenario                string
	Path                    string
	RuleIDs                 []string
	Surfaces                []string
	IncludeCommandExecution bool
	IncludeAutofixPreview   bool
	UseCache                bool
}

type Response struct {
	RunID             string
	Status            string
	Summary           string
	Inventory         surfaces.Inventory
	Contracts         []ContractEvaluation
	Findings          []Finding
	CommandResults    []commands.Result
	Maturity          Maturity
	NextSteps         []string
	AutofixCandidates []autofix.Candidate
}

type ContractEvaluation struct {
	ContractID string
	SurfaceID  string
	Status     string
	RuleIDs    []string
}

type Finding struct {
	ID               string
	Scenario         string
	TargetKind       string
	SurfaceID        string
	SurfaceKind      string
	Language         string
	Framework        string
	RuleID           string
	Category         string
	Severity         string
	FilePath         string
	Symbol           string
	Message          string
	Evidence         string
	Expected         string
	Observed         string
	WhyItMatters     string
	Remediation      string
	AutofixAvailable bool
	AutofixCommand   string
	SourceCommand    string
	CreatedAt        string
}

type Maturity struct {
	Rung      int
	Label     string
	Rationale string
}

func New(disc surfaces.Discoverer) *Service {
	return &Service{Discoverer: disc}
}

func (s *Service) Audit(ctx context.Context, req Request) (Response, error) {
	now := s.now()
	disc := s.Discoverer
	if disc == nil {
		disc = surfaces.CodeFactsClient{Locator: s.Locator}
	}
	inv, err := disc.Discover(ctx, req.Scenario, req.Path, req.UseCache)
	if err != nil {
		return Response{}, err
	}
	filtered := filterSurfaces(inv.Surfaces, req.Surfaces)
	inv.Surfaces = filtered
	res := Response{
		RunID:     "qh-" + now.UTC().Format("20060102-150405"),
		Inventory: inv,
	}
	for _, surface := range filtered {
		before := len(res.Findings)
		res.Findings = append(res.Findings, s.evaluateSurface(inv, surface, req.RuleIDs, now)...)
		res.Contracts = append(res.Contracts, ContractEvaluation{
			ContractID: contractIDForSurface(surface),
			SurfaceID:  surface.ID,
			Status:     statusFromFindings(res.Findings[before:]),
			RuleIDs:    rulesForSurface(surface),
		})
	}
	beforeScenario := len(res.Findings)
	res.Findings = append(res.Findings, s.evaluateScenario(inv, req.RuleIDs, now)...)
	res.Contracts = append(res.Contracts, ContractEvaluation{
		ContractID: "scenario-quality-gates",
		SurfaceID:  "scenario",
		Status:     statusFromFindings(res.Findings[beforeScenario:]),
		RuleIDs:    []string{contracts.RuleTestingConfigStrict, contracts.RuleMakefileQualityGates},
	})
	if req.IncludeCommandExecution {
		res.CommandResults = commands.RunAll(ctx, s.Executor, inv)
	}
	if req.IncludeAutofixPreview {
		candidates, err := autofix.Preview(inv.RootPath, req.RuleIDs)
		if err == nil {
			res.AutofixCandidates = candidates
		}
	}
	sortFindings(res.Findings)
	res.Maturity = maturity(inv, res.Findings, res.CommandResults)
	res.Status = auditStatus(inv, res.Findings)
	res.Summary = summary(res)
	res.NextSteps = nextSteps(res)
	return res, nil
}

func (s *Service) PreviewFix(ctx context.Context, scenario, path string, ruleIDs []string) (surfaces.Inventory, []autofix.Candidate, error) {
	inv, err := s.inventory(ctx, scenario, path)
	if err != nil {
		return surfaces.Inventory{}, nil, err
	}
	candidates, err := autofix.Preview(inv.RootPath, ruleIDs)
	return inv, candidates, err
}

func (s *Service) ApplyFix(ctx context.Context, scenario, path string, ruleIDs []string) (surfaces.Inventory, []autofix.Candidate, error) {
	inv, err := s.inventory(ctx, scenario, path)
	if err != nil {
		return surfaces.Inventory{}, nil, err
	}
	candidates, err := autofix.Apply(inv.RootPath, ruleIDs)
	return inv, candidates, err
}

func (s *Service) inventory(ctx context.Context, scenario, path string) (surfaces.Inventory, error) {
	disc := s.Discoverer
	if disc == nil {
		disc = surfaces.CodeFactsClient{Locator: s.Locator}
	}
	return disc.Discover(ctx, scenario, path, false)
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *Service) evaluateSurface(inv surfaces.Inventory, surface surfaces.Surface, onlyRules []string, now time.Time) []Finding {
	var out []Finding
	if surface.Kind == "ui" && (surface.Language == "typescript" || surface.Language == "javascript") {
		out = append(out, evalTSConfig(inv, surface, onlyRules, now)...)
		out = append(out, evalESLint(inv, surface, onlyRules, now)...)
		out = append(out, evalPackage(inv, surface, onlyRules, now)...)
		out = append(out, evalDangerousPatterns(inv, surface, onlyRules, now)...)
	}
	if surface.Language == "go" || surface.Kind == "api" || surface.Kind == "cli" {
		out = append(out, evalGo(inv, surface, onlyRules, now)...)
	}
	return out
}

func (s *Service) evaluateScenario(inv surfaces.Inventory, onlyRules []string, now time.Time) []Finding {
	var out []Finding
	out = append(out, evalTestingConfig(inv, onlyRules, now)...)
	out = append(out, evalMakefile(inv, onlyRules, now)...)
	return out
}

func evalTSConfig(inv surfaces.Inventory, surface surfaces.Surface, onlyRules []string, now time.Time) []Finding {
	if !wants(onlyRules, contracts.RuleTSConfigStrict) {
		return nil
	}
	path := filepath.Join(surface.RootPath, "tsconfig.json")
	content, err := os.ReadFile(path)
	if err != nil {
		return []Finding{newFinding(inv, surface, contracts.RuleTSConfigStrict, "typescript", "error", path, "tsconfig.json not found", "TypeScript UI surface has no tsconfig.json.", "strict true, noUncheckedIndexedAccess true, and safety comments", "missing", true, now)}
	}
	raw := string(content)
	stripped := autofix.StripJSONCComments(raw)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(stripped), &parsed); err != nil {
		return []Finding{newFinding(inv, surface, contracts.RuleTSConfigStrict, "typescript", "error", path, "tsconfig.json parse error", err.Error(), "valid JSONC", "parse error", true, now)}
	}
	compiler, _ := parsed["compilerOptions"].(map[string]any)
	var findings []Finding
	if compiler == nil {
		findings = append(findings, newFinding(inv, surface, contracts.RuleTSConfigStrict, "typescript", "error", path, "Missing compilerOptions", "compilerOptions section is absent.", "compilerOptions with strict and noUncheckedIndexedAccess", "missing", true, now))
		return findings
	}
	if strict, ok := compiler["strict"].(bool); !ok || !strict {
		findings = append(findings, newFinding(inv, surface, contracts.RuleTSConfigStrict, "typescript", "error", path, "strict mode not enabled", "compilerOptions.strict is not true.", "strict: true", fmt.Sprintf("%v", compiler["strict"]), true, now))
	}
	if noUnchecked, ok := compiler["noUncheckedIndexedAccess"].(bool); !ok || !noUnchecked {
		findings = append(findings, newFinding(inv, surface, contracts.RuleTSConfigStrict, "typescript", "error", path, "noUncheckedIndexedAccess not enabled", "compilerOptions.noUncheckedIndexedAccess is not true.", "noUncheckedIndexedAccess: true", fmt.Sprintf("%v", compiler["noUncheckedIndexedAccess"]), true, now))
	}
	if !autofix.HasTSConfigProtectiveComments(raw) {
		findings = append(findings, newFinding(inv, surface, contracts.RuleTSConfigStrict, "typescript", "error", path, "Missing protective comment block", "Strict settings exist without the guardrail comments that tell agents not to weaken them.", "all required safety comment phrases", "missing comment phrase(s)", true, now))
	}
	return findings
}

func evalESLint(inv surfaces.Inventory, surface surfaces.Surface, onlyRules []string, now time.Time) []Finding {
	var findings []Finding
	path, raw := findESLint(surface.RootPath)
	if path == "" {
		if wants(onlyRules, contracts.RuleESLintSafetyRules) {
			findings = append(findings, newFinding(inv, surface, contracts.RuleESLintSafetyRules, "typescript", "error", filepath.Join(surface.RootPath, "eslint.config.js"), "ESLint config not found", "Safety-critical lint rules cannot be verified.", "eslint config with safety rules and comments", "missing", false, now))
		}
		return findings
	}
	if wants(onlyRules, contracts.RuleESLintSafetyRules) {
		if !strings.Contains(raw, "SAFETY-CRITICAL RULES - DO NOT REMOVE, DISABLE, OR WEAKEN") {
			findings = append(findings, newFinding(inv, surface, contracts.RuleESLintSafetyRules, "typescript", "error", path, "Missing safety-critical header comment", "ESLint config lacks the guardrail header that tells agents not to disable safety rules.", "safety-critical header comment", "missing", false, now))
		}
		if missing := missingCriticalComments(raw); len(missing) > 0 {
			findings = append(findings, newFinding(inv, surface, contracts.RuleESLintSafetyRules, "typescript", "error", path, "Missing per-rule CRITICAL comments", "Missing // CRITICAL comments for "+strings.Join(missing, ", ")+".", "per-rule // CRITICAL comments", "missing", false, now))
		}
		if missing, weak := lintRuleProblems(raw); len(missing) > 0 || len(weak) > 0 {
			observed := strings.Join(append(prefixAll("missing ", missing), prefixAll("weak ", weak)...), ", ")
			findings = append(findings, newFinding(inv, surface, contracts.RuleESLintSafetyRules, "typescript", "error", path, "ESLint safety rules incomplete", "Required safety rules are missing or weaker than the minimum level.", "required safety rules at warn/error minimums", observed, false, now))
		}
	}
	if wants(onlyRules, contracts.RuleESLintTypedConfig) {
		var missing []string
		if !strings.Contains(raw, "strictTypeChecked") {
			missing = append(missing, "strictTypeChecked")
		}
		if !strings.Contains(raw, "parserOptions") || (!strings.Contains(raw, "project") && !strings.Contains(raw, "projectService")) {
			missing = append(missing, "parserOptions.project or projectService")
		}
		if strings.Contains(raw, `"import/no-cycle"`) && (!strings.Contains(raw, "import/resolver") || !strings.Contains(raw, "typescript")) {
			missing = append(missing, "TypeScript import resolver")
		}
		if len(missing) > 0 {
			findings = append(findings, newFinding(inv, surface, contracts.RuleESLintTypedConfig, "typescript", "error", path, "ESLint typed configuration is incomplete", "Missing "+strings.Join(missing, ", ")+".", "strictTypeChecked, typed parser options, and TS import resolver", strings.Join(missing, ", "), false, now))
		}
	}
	return findings
}

func evalPackage(inv surfaces.Inventory, surface surfaces.Surface, onlyRules []string, now time.Time) []Finding {
	if !wants(onlyRules, contracts.RuleNodeBuildTypecheck) {
		return nil
	}
	path := filepath.Join(surface.RootPath, "package.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var pkg struct {
		Scripts map[string]string `json:"scripts"`
	}
	if err := json.Unmarshal(raw, &pkg); err != nil {
		return []Finding{newFinding(inv, surface, contracts.RuleNodeBuildTypecheck, "typescript", "error", path, "package.json parse error", err.Error(), "valid package.json", "parse error", false, now)}
	}
	build := strings.TrimSpace(pkg.Scripts["build"])
	if build == "" {
		return []Finding{newFinding(inv, surface, contracts.RuleNodeBuildTypecheck, "typescript", "error", path, "Missing build script", "The UI package does not define a build script.", "build script that runs typecheck before bundling", "missing", false, now)}
	}
	if !strings.Contains(build, "tsc --noEmit") && !strings.Contains(build, "run type-check") && !strings.Contains(build, "type-check &&") {
		return []Finding{newFinding(inv, surface, contracts.RuleNodeBuildTypecheck, "typescript", "error", path, "Build script skips TypeScript type checking", "Build script is `"+build+"`.", "tsc --noEmit or type-check before bundling", build, false, now)}
	}
	return nil
}

func evalDangerousPatterns(inv surfaces.Inventory, surface surfaces.Surface, onlyRules []string, now time.Time) []Finding {
	if !wants(onlyRules, contracts.RuleTSDangerousPatterns) {
		return nil
	}
	type counts struct{ asAny, asType, ignore, nonNull, total int }
	total := counts{}
	perFile := map[string]counts{}
	_ = filepath.WalkDir(surface.RootPath, func(path string, d os.DirEntry, err error) error {
		if err != nil || d == nil {
			return nil
		}
		if d.IsDir() {
			switch d.Name() {
			case "node_modules", "dist", "build", "coverage", ".vite", ".git":
				return filepath.SkipDir
			}
			return nil
		}
		if ext := filepath.Ext(path); ext != ".ts" && ext != ".tsx" && ext != ".js" && ext != ".jsx" {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		text := string(raw)
		c := counts{
			asAny:   strings.Count(text, "as any"),
			asType:  countTypeAssertions(text),
			ignore:  strings.Count(text, "@ts-ignore"),
			nonNull: countNonNullAssertions(text),
		}
		c.total = c.asAny + c.asType + c.ignore + c.nonNull
		if c.total > 0 {
			perFile[path] = c
			total.asAny += c.asAny
			total.asType += c.asType
			total.ignore += c.ignore
			total.nonNull += c.nonNull
			total.total += c.total
		}
		return nil
	})
	if total.total == 0 {
		return nil
	}
	files := make([]string, 0, len(perFile))
	for path := range perFile {
		files = append(files, path)
	}
	sort.Strings(files)
	if len(files) > 10 {
		files = files[:10]
	}
	evidence := fmt.Sprintf("as any=%d, type assertions=%d, @ts-ignore=%d, non-null assertions=%d, top files=%s", total.asAny, total.asType, total.ignore, total.nonNull, strings.Join(files, ", "))
	return []Finding{newFinding(inv, surface, contracts.RuleTSDangerousPatterns, "typescript", "warning", surface.RootPath, "Dangerous TypeScript suppression patterns found", evidence, "zero dangerous suppression patterns", evidence, false, now)}
}

func evalGo(inv surfaces.Inventory, surface surfaces.Surface, onlyRules []string, now time.Time) []Finding {
	var findings []Finding
	if wants(onlyRules, contracts.RuleGoModPresent) && hasGoFiles(surface.RootPath) && !exists(filepath.Join(surface.RootPath, "go.mod")) {
		findings = append(findings, newFinding(inv, surface, contracts.RuleGoModPresent, "go", "error", filepath.Join(surface.RootPath, "go.mod"), "Missing go.mod", "Go files exist without a module file.", "go.mod present", "missing", false, now))
	}
	configPath := firstExisting(filepath.Join(surface.RootPath, ".golangci.yml"), filepath.Join(surface.RootPath, ".golangci.yaml"))
	if wants(onlyRules, contracts.RuleGoLintConfigPresent) && hasGoFiles(surface.RootPath) && configPath == "" {
		findings = append(findings, newFinding(inv, surface, contracts.RuleGoLintConfigPresent, "go", "error", filepath.Join(surface.RootPath, ".golangci.yml"), "Missing golangci-lint config", "Go lint behavior is environment-dependent without checked-in config.", ".golangci.yml/.yaml present", "missing", false, now))
	}
	if wants(onlyRules, contracts.RuleGoLintRequiredLinters) && configPath != "" {
		raw, _ := os.ReadFile(configPath)
		var missing []string
		for _, linter := range []string{"errcheck", "gofumpt", "govet", "ineffassign", "staticcheck", "typecheck", "unused"} {
			if !strings.Contains(string(raw), linter) {
				missing = append(missing, linter)
			}
		}
		if len(missing) > 0 {
			findings = append(findings, newFinding(inv, surface, contracts.RuleGoLintRequiredLinters, "go", "error", configPath, "golangci-lint baseline incomplete", "Missing required linters: "+strings.Join(missing, ", ")+".", "baseline linters enabled", strings.Join(missing, ", "), false, now))
		}
	}
	return findings
}

func evalTestingConfig(inv surfaces.Inventory, onlyRules []string, now time.Time) []Finding {
	if !wants(onlyRules, contracts.RuleTestingConfigStrict) {
		return nil
	}
	path := filepath.Join(inv.RootPath, ".vrooli", "testing.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return []Finding{scenarioFinding(inv, contracts.RuleTestingConfigStrict, "error", path, "testing.json missing lint strictness policy", ".vrooli/testing.json is missing.", "strict node_package/go_module lint handlers", "missing", now)}
	}
	var cfg struct {
		Lint struct {
			Handlers struct {
				GoModule    strictHandler `json:"go_module"`
				NodePackage strictHandler `json:"node_package"`
			} `json:"handlers"`
		} `json:"lint"`
	}
	if err := json.Unmarshal([]byte(autofix.StripJSONCComments(string(raw))), &cfg); err != nil {
		return []Finding{scenarioFinding(inv, contracts.RuleTestingConfigStrict, "error", path, "testing.json parse error", err.Error(), "valid strict lint config", "parse error", now)}
	}
	var findings []Finding
	if hasNode(inv) && !cfg.Lint.Handlers.NodePackage.StrictEnabled() {
		findings = append(findings, scenarioFinding(inv, contracts.RuleTestingConfigStrict, "error", path, "Node lint strict mode not enabled", "node_package strict lint handler is not enabled.", "node_package enabled=true strict=true", "not strict", now))
	}
	if hasGo(inv) && !cfg.Lint.Handlers.GoModule.StrictEnabled() {
		findings = append(findings, scenarioFinding(inv, contracts.RuleTestingConfigStrict, "error", path, "Go lint strict mode not enabled", "go_module strict lint handler is not enabled.", "go_module enabled=true strict=true", "not strict", now))
	}
	return findings
}

func evalMakefile(inv surfaces.Inventory, onlyRules []string, now time.Time) []Finding {
	if !wants(onlyRules, contracts.RuleMakefileQualityGates) {
		return nil
	}
	path := filepath.Join(inv.RootPath, "Makefile")
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	text := string(raw)
	var missing []string
	if hasNode(inv) {
		if !strings.Contains(text, "fmt-ui:") || !strings.Contains(text, "lint:fix") {
			missing = append(missing, "fmt-ui target that runs lint:fix")
		}
		if !strings.Contains(text, "lint-ui:") || !strings.Contains(text, "pnpm run lint") || !strings.Contains(text, "pnpm run type-check") {
			missing = append(missing, "lint-ui target that runs lint and type-check")
		}
	}
	if hasGo(inv) {
		if !strings.Contains(text, "lint-go:") || !strings.Contains(text, "golangci-lint") {
			missing = append(missing, "lint-go target that runs golangci-lint")
		}
		if !strings.Contains(text, "fmt-go:") || (!strings.Contains(text, "gofumpt") && !strings.Contains(text, "gofmt")) {
			missing = append(missing, "fmt-go target that runs gofumpt/gofmt")
		}
	}
	if len(missing) == 0 {
		return nil
	}
	return []Finding{scenarioFinding(inv, contracts.RuleMakefileQualityGates, "warning", path, "Makefile quality targets are incomplete", strings.Join(missing, ", "), "real fmt/lint gates for UI and Go surfaces", strings.Join(missing, ", "), now)}
}

type strictHandler struct {
	Enabled *bool `json:"enabled"`
	Strict  *bool `json:"strict"`
}

func (h strictHandler) StrictEnabled() bool {
	if h.Enabled != nil && !*h.Enabled {
		return false
	}
	return h.Strict != nil && *h.Strict
}

var eslintRules = []struct {
	rule string
	min  string
}{
	{"react-hooks/rules-of-hooks", "error"},
	{"@typescript-eslint/no-non-null-assertion", "error"},
	{"@typescript-eslint/no-explicit-any", "error"},
	{"@typescript-eslint/no-unsafe-member-access", "warn"},
	{"@typescript-eslint/no-unsafe-call", "warn"},
	{"@typescript-eslint/no-unsafe-argument", "warn"},
	{"@typescript-eslint/no-unsafe-assignment", "warn"},
	{"@typescript-eslint/no-unsafe-return", "warn"},
	{"import/no-cycle", "error"},
}

func lintRuleProblems(content string) (missing, weak []string) {
	for _, req := range eslintRules {
		escaped := regexp.QuoteMeta(req.rule)
		re := regexp.MustCompile(`["']` + escaped + `["']\s*:\s*["'](error|warn|off)["']`)
		match := re.FindStringSubmatch(content)
		if len(match) == 0 {
			missing = append(missing, req.rule)
			continue
		}
		if levelRank(match[1]) < levelRank(req.min) {
			weak = append(weak, req.rule+"="+match[1])
		}
	}
	return missing, weak
}

func missingCriticalComments(content string) []string {
	var missing []string
	for _, rule := range []string{"rules-of-hooks", "no-non-null-assertion", "no-unsafe-", "no-cycle"} {
		idx := strings.Index(content, rule)
		if idx < 0 {
			continue
		}
		start := idx - 200
		if start < 0 {
			start = 0
		}
		if !strings.Contains(content[start:idx], "// CRITICAL:") {
			missing = append(missing, rule)
		}
	}
	return missing
}

func levelRank(level string) int {
	switch level {
	case "off":
		return 0
	case "warn":
		return 1
	case "error":
		return 2
	default:
		return -1
	}
}

func newFinding(inv surfaces.Inventory, surface surfaces.Surface, ruleID, category, severity, path, message, evidence, expected, observed string, autofixAvailable bool, now time.Time) Finding {
	c, _ := contracts.ByRule(ruleID)
	f := Finding{
		Scenario:         inv.Scenario,
		TargetKind:       inv.TargetKind,
		SurfaceID:        surface.ID,
		SurfaceKind:      surface.Kind,
		Language:         surface.Language,
		Framework:        surface.Framework,
		RuleID:           ruleID,
		Category:         category,
		Severity:         severity,
		FilePath:         path,
		Message:          message,
		Evidence:         evidence,
		Expected:         expected,
		Observed:         observed,
		WhyItMatters:     c.WhyItMatters,
		Remediation:      c.Remediation,
		AutofixAvailable: autofixAvailable,
		CreatedAt:        now.UTC().Format(time.RFC3339),
	}
	if autofixAvailable {
		f.AutofixCommand = fmt.Sprintf("quality-health fix-config %s --rule %s --dry-run", inv.Scenario, ruleID)
	}
	f.ID = stableID(f)
	return f
}

func scenarioFinding(inv surfaces.Inventory, ruleID, severity, path, message, evidence, expected, observed string, now time.Time) Finding {
	return newFinding(inv, surfaces.Surface{ID: "scenario", Kind: "scenario"}, ruleID, "scenario", severity, path, message, evidence, expected, observed, false, now)
}

func stableID(f Finding) string {
	h := sha256.Sum256([]byte(strings.Join([]string{f.Scenario, f.SurfaceID, f.RuleID, f.FilePath, f.Symbol, f.Expected, f.Observed}, "\x00")))
	return hex.EncodeToString(h[:])[:16]
}

func wants(onlyRules []string, ruleID string) bool {
	if len(onlyRules) == 0 {
		return true
	}
	for _, id := range onlyRules {
		if strings.EqualFold(strings.TrimSpace(id), ruleID) {
			return true
		}
	}
	return false
}

func filterSurfaces(in []surfaces.Surface, ids []string) []surfaces.Surface {
	if len(ids) == 0 {
		return in
	}
	want := map[string]bool{}
	for _, id := range ids {
		want[strings.TrimSpace(id)] = true
	}
	var out []surfaces.Surface
	for _, s := range in {
		if want[s.ID] || want[s.Kind] {
			out = append(out, s)
		}
	}
	return out
}

func findESLint(root string) (string, string) {
	for _, name := range []string{"eslint.config.js", "eslint.config.mjs", "eslint.config.cjs", ".eslintrc.json", ".eslintrc.js", ".eslintrc.cjs"} {
		path := filepath.Join(root, name)
		raw, err := os.ReadFile(path)
		if err == nil {
			return path, string(raw)
		}
	}
	return "", ""
}

func countTypeAssertions(text string) int {
	re := regexp.MustCompile(`\bas\s+[A-ZA-Za-z_{\[]`)
	return len(re.FindAllString(text, -1))
}

func countNonNullAssertions(text string) int {
	re := regexp.MustCompile(`[A-Za-z0-9_\]\)]!([.;,\)\]\}])`)
	return len(re.FindAllString(text, -1))
}

func hasGoFiles(root string) bool {
	found := false
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d == nil || found {
			return nil
		}
		if d.IsDir() {
			switch d.Name() {
			case "vendor", "node_modules", ".git":
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) == ".go" {
			found = true
			return filepath.SkipAll
		}
		return nil
	})
	return found
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func firstExisting(paths ...string) string {
	for _, path := range paths {
		if exists(path) {
			return path
		}
	}
	return ""
}

func hasNode(inv surfaces.Inventory) bool {
	for _, s := range inv.Surfaces {
		if s.Language == "typescript" || s.Language == "javascript" || exists(filepath.Join(s.RootPath, "package.json")) {
			return true
		}
	}
	return false
}

func hasGo(inv surfaces.Inventory) bool {
	for _, s := range inv.Surfaces {
		if s.Language == "go" || exists(filepath.Join(s.RootPath, "go.mod")) {
			return true
		}
	}
	return false
}

func contractIDForSurface(surface surfaces.Surface) string {
	if surface.Kind == "ui" {
		return "typescript-react-vite-ui"
	}
	if surface.Kind == "api" || surface.Kind == "cli" {
		return "go-api-cli-quality"
	}
	return "scenario-quality-gates"
}

func rulesForSurface(surface surfaces.Surface) []string {
	if surface.Kind == "ui" {
		return []string{contracts.RuleTSConfigStrict, contracts.RuleESLintSafetyRules, contracts.RuleTSDangerousPatterns, contracts.RuleESLintTypedConfig, contracts.RuleNodeBuildTypecheck}
	}
	if surface.Kind == "api" || surface.Kind == "cli" {
		return []string{contracts.RuleGoModPresent, contracts.RuleGoLintConfigPresent, contracts.RuleGoLintRequiredLinters}
	}
	return nil
}

func statusFromFindings(findings []Finding) string {
	for _, f := range findings {
		if f.Severity == "error" {
			return "failed"
		}
	}
	if len(findings) > 0 {
		return "warning"
	}
	return "passed"
}

func sortFindings(findings []Finding) {
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Severity != findings[j].Severity {
			return findings[i].Severity < findings[j].Severity
		}
		if findings[i].RuleID != findings[j].RuleID {
			return findings[i].RuleID < findings[j].RuleID
		}
		return findings[i].FilePath < findings[j].FilePath
	})
}

func auditStatus(inv surfaces.Inventory, findings []Finding) string {
	if inv.DegradedReason != "" {
		return "degraded"
	}
	for _, f := range findings {
		if f.Severity == "error" {
			return "failed"
		}
	}
	return "passed"
}

func maturity(inv surfaces.Inventory, findings []Finding, results []commands.Result) Maturity {
	if inv.DegradedReason != "" || len(inv.Surfaces) == 0 {
		return Maturity{Rung: 0, Label: "L0", Rationale: "No reliable Code Facts-backed quality audit."}
	}
	if hasError(findings) {
		return Maturity{Rung: 2, Label: "L2", Rationale: "Surfaces discovered, but strict quality contracts are not yet satisfied."}
	}
	if len(results) == 0 {
		return Maturity{Rung: 3, Label: "L3", Rationale: "Strict quality contracts are satisfied; command execution was not requested."}
	}
	for _, r := range results {
		if r.Status != "passed" {
			return Maturity{Rung: 3, Label: "L3", Rationale: "Contracts are satisfied but one or more lint/type commands failed."}
		}
	}
	return Maturity{Rung: 4, Label: "L4", Rationale: "Contracts and lint/type commands passed."}
}

func hasError(findings []Finding) bool {
	for _, f := range findings {
		if f.Severity == "error" {
			return true
		}
	}
	return false
}

func summary(res Response) string {
	errors, warnings, infos := countFindings(res.Findings)
	return fmt.Sprintf("%s: %d error(s), %d warning(s), %d info(s) across %d surface(s)", res.Inventory.Scenario, errors, warnings, infos, len(res.Inventory.Surfaces))
}

func nextSteps(res Response) []string {
	if len(res.AutofixCandidates) > 0 {
		return []string{fmt.Sprintf("Run `quality-health fix-config run %s --dry-run` to inspect safe config repairs.", res.Inventory.Scenario)}
	}
	if len(res.Findings) > 0 {
		return []string{fmt.Sprintf("Run `quality-health explain finding %s --scenario %s --rule %s` for remediation detail.", res.Findings[0].ID, res.Inventory.Scenario, res.Findings[0].RuleID)}
	}
	return []string{"No Quality Health remediation is required."}
}

func countFindings(findings []Finding) (errors, warnings, infos int) {
	for _, f := range findings {
		switch f.Severity {
		case "error":
			errors++
		case "warning":
			warnings++
		default:
			infos++
		}
	}
	return errors, warnings, infos
}

func prefixAll(prefix string, values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, prefix+value)
	}
	return out
}
