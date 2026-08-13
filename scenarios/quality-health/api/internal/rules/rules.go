package rules

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"quality-health/internal/surfaces"

	"gopkg.in/yaml.v3"
)

const (
	RuleTSConfigStrict            = "TS_CONFIG_STRICT"
	RuleESLintSafetyRules         = "ESLINT_SAFETY_RULES"
	RuleTSDangerousPatterns       = "TS_DANGEROUS_PATTERNS"
	RuleESLintTypedConfig         = "ESLINT_TYPED_CONFIG"
	RuleNodeBuildTypecheck        = "NODE_BUILD_TYPECHECK"
	RuleUILazyChunkRecovery       = "UI_LAZY_CHUNK_RECOVERY"
	RuleTestingConfigStrict       = "TESTING_CONFIG_LINT_STRICT"
	RuleGoModPresent              = "GO_MOD_PRESENT_FOR_API_OR_CLI"
	RuleGoLintConfigPresent       = "GO_LINT_CONFIG_PRESENT"
	RuleGoLintRequiredLinters     = "GO_LINT_REQUIRED_LINTERS"
	RuleGoDangerousPatterns       = "GO_DANGEROUS_PATTERNS"
	RuleScenarioPrivilegeBoundary = "SCENARIO_PRIVILEGE_BOUNDARY"
	RuleMakefileQualityGates      = "MAKEFILE_QUALITY_GATES"
	RuleShellSyntaxLint           = "SHELL_SYNTAX_LINT"

	// RuleCoverageGap marks a discovered surface for which no quality contract
	// pack applies. It is an informational honesty signal, not a registry
	// contract, so it is intentionally absent from Registry().
	RuleCoverageGap = "QUALITY_COVERAGE_GAP"
)

const (
	FixClassAutofix       = "autofix"
	FixClassDetectionOnly = "detection_only"
)

type Applicability struct {
	Language    string
	Framework   string
	SurfaceKind string
	// Scenario marks a rule that asserts a scenario contract — a Makefile
	// target, a coverage/testing.json policy, a scenario-shaped shell layout.
	// A shared package or control-plane tree has none of those by
	// construction, so such a rule must not run against one.
	Scenario bool
}

// MatchesTargetKind reports whether a rule may run against a target of this
// kind. An empty kind means the caller could not resolve one, which must not
// narrow coverage: an unknown target keeps the pre-target-model behavior of
// running everything.
//
// This is the guard that stops `MAKEFILE_QUALITY_GATES` and
// `TESTING_CONFIG_LINT_STRICT` failing `package:api-core` — a package has no
// Makefile and no coverage/testing.json, so the finding was never about a
// real defect in the target.
func (a Applicability) MatchesTargetKind(kind string) bool {
	kind = strings.TrimSpace(kind)
	if !a.Scenario || kind == "" {
		return true
	}
	return kind == "scenario"
}

type Rule struct {
	ID           string
	Title        string
	Category     string
	Severity     string
	FixClass     string
	FixReason    string
	ContractID   string
	Applies      Applicability
	WhyItMatters string
	Remediation  string
	Evaluate     EvalFunc
}

type EvalFunc func(EvalContext) []Finding

type EvalContext struct {
	Inventory surfaces.Inventory
	Surface   surfaces.Surface
	Now       time.Time
}

type Finding struct {
	Surface          surfaces.Surface
	RuleID           string
	Category         string
	Severity         string
	FilePath         string
	Message          string
	Evidence         string
	Expected         string
	Observed         string
	FixClass         string
	AutofixAvailable bool
}

func Registry() []Rule {
	return []Rule{
		{ID: RuleTSConfigStrict, Title: "Strict TypeScript config", Category: "typescript", Severity: "error", FixClass: FixClassAutofix, ContractID: "typescript-static-quality", Applies: Applicability{Language: "typescript"}, Evaluate: evalTSConfig},
		{ID: RuleESLintSafetyRules, Title: "ESLint safety rules", Category: "typescript", Severity: "error", FixClass: FixClassAutofix, ContractID: "typescript-static-quality", Applies: Applicability{Language: "typescript"}, Evaluate: evalESLintSafety},
		{ID: RuleTSDangerousPatterns, Title: "TypeScript dangerous patterns", Category: "typescript", Severity: "warning", FixClass: FixClassDetectionOnly, FixReason: "Source-semantic suppressions require human intent and are not safe config rewrites.", ContractID: "typescript-static-quality", Applies: Applicability{Language: "typescript"}, Evaluate: evalDangerousPatterns},
		{ID: RuleESLintTypedConfig, Title: "Typed ESLint config", Category: "typescript", Severity: "error", FixClass: FixClassAutofix, ContractID: "typescript-static-quality", Applies: Applicability{Language: "typescript"}, Evaluate: evalESLintTyped},
		{ID: RuleNodeBuildTypecheck, Title: "Node build typecheck", Category: "typescript", Severity: "error", FixClass: FixClassAutofix, ContractID: "typescript-static-quality", Applies: Applicability{Language: "typescript"}, Evaluate: evalPackage},
		{ID: RuleUILazyChunkRecovery, Title: "Lazy-chunk deploy recovery", Category: "typescript", Severity: "error", FixClass: FixClassDetectionOnly, FixReason: "Installing the reload guard is an app-entry code change, not a safe config rewrite.", ContractID: "typescript-static-quality", Applies: Applicability{Language: "typescript"}, WhyItMatters: "Vite builds emit content-hashed chunks; a rebuild deletes the old ones, so any tab opened before the deploy crashes into its error boundary on the next lazy() navigation unless the app self-heals with a reload.", Remediation: "Call installChunkReloadGuard() from @vrooli/api-base at the app entry point (before React mounts), or handle Vite's vite:preloadError event directly.", Evaluate: evalLazyChunkRecovery},
		{ID: RuleTestingConfigStrict, Title: "Testing strict lint handlers", Category: "scenario", Severity: "error", FixClass: FixClassAutofix, ContractID: "scenario-quality-gates", Applies: Applicability{SurfaceKind: "scenario", Scenario: true}, Evaluate: evalTestingConfig},
		{ID: RuleGoModPresent, Title: "Go module present", Category: "go", Severity: "error", FixClass: FixClassAutofix, ContractID: "go-static-quality", Applies: Applicability{Language: "go"}, Evaluate: evalGoModPresent},
		{ID: RuleGoLintConfigPresent, Title: "Go lint config present", Category: "go", Severity: "error", FixClass: FixClassAutofix, ContractID: "go-static-quality", Applies: Applicability{Language: "go"}, Evaluate: evalGoLintConfigPresent},
		{ID: RuleGoLintRequiredLinters, Title: "Go lint required linters", Category: "go", Severity: "error", FixClass: FixClassAutofix, ContractID: "go-static-quality", Applies: Applicability{Language: "go"}, Evaluate: evalGoLintRequiredLinters},
		{ID: RuleGoDangerousPatterns, Title: "Go dangerous patterns", Category: "go", Severity: "warning", FixClass: FixClassDetectionOnly, FixReason: "Source suppressions require a written reason, not automated source edits.", ContractID: "go-static-quality", Applies: Applicability{Language: "go"}, Evaluate: evalGoDangerousPatterns},
		{ID: RuleScenarioPrivilegeBoundary, Title: "Runtime privilege boundary", Category: "go", Severity: "critical", FixClass: FixClassDetectionOnly, FixReason: "Elevation must be provisioned at setup, not spawned at runtime. Automated source edits cannot decide which grant covers a command.", ContractID: "go-static-quality", Applies: Applicability{Language: "go"}, WhyItMatters: "`vrooli setup` is the single consent and elevation boundary. A scenario that spawns sudo at runtime either prompts a process with no terminal and fails silently, or requires an operator to hold standing root for a long-lived service. `docs/architecture/PRIVILEGE_BROKER.md` rejects direct scenario sudo for both reasons.", Remediation: "Move the privileged command into a safeguard that provisions a grant during `sudo vrooli setup`, following `internal/safeguards/cloudflared-recovery-privileges/handler.go`, or add a typed action to the privilege broker registry in `internal/privilegebroker/policy.go`. Then call the granted argv, or return a typed needs-elevation result naming the operator command.", Evaluate: evalScenarioPrivilegeBoundary},
		{ID: RuleMakefileQualityGates, Title: "Makefile quality gates", Category: "scenario", Severity: "warning", FixClass: FixClassAutofix, ContractID: "scenario-quality-gates", Applies: Applicability{SurfaceKind: "scenario", Scenario: true}, Evaluate: evalMakefile},
		{ID: RuleShellSyntaxLint, Title: "Shell script syntax lint", Category: "shell", Severity: "warning", FixClass: FixClassDetectionOnly, FixReason: "Shell syntax fixes require human intent; quality-health detects, it does not rewrite scripts.", ContractID: "scenario-quality-gates", Applies: Applicability{SurfaceKind: "scenario", Scenario: true}, WhyItMatters: "A shell script that fails `bash -n` is broken before it runs; CLI scaffolding silently breaking is a common, undetected regression. quality-health owns shell *syntax* lint (unit-health owns bats *testing*).", Remediation: "Run `bash -n <script>` and `shellcheck <script>` locally and fix the reported syntax errors.", Evaluate: evalShellSyntax},
	}
}

func ByID(ruleID string) (Rule, bool) {
	for _, rule := range Registry() {
		if rule.ID == ruleID {
			return rule, true
		}
	}
	return Rule{}, false
}

func SurfaceRules(surface surfaces.Surface) []Rule {
	var out []Rule
	for _, rule := range Registry() {
		if rule.Applies.Scenario || !appliesToSurface(rule.Applies, surface) {
			continue
		}
		out = append(out, rule)
	}
	return out
}

func ScenarioRules() []Rule {
	var out []Rule
	for _, rule := range Registry() {
		if rule.Applies.Scenario {
			out = append(out, rule)
		}
	}
	return out
}

func ContractRules(contractID string) []Rule {
	var out []Rule
	for _, rule := range Registry() {
		if rule.ContractID == contractID {
			out = append(out, rule)
		}
	}
	return out
}

func Filter(ruleList []Rule, onlyRules []string) []Rule {
	if len(onlyRules) == 0 {
		return ruleList
	}
	var out []Rule
	for _, rule := range ruleList {
		if Wants(onlyRules, rule.ID) {
			out = append(out, rule)
		}
	}
	return out
}

func Wants(onlyRules []string, ruleID string) bool {
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

func NormalizeLanguage(language string) string {
	if language == "javascript" {
		return "typescript"
	}
	return language
}

func AppliesToFilter(rule Rule, language, framework, surfaceKind string) bool {
	if language != "" && rule.Applies.Language != "" && !strings.EqualFold(rule.Applies.Language, NormalizeLanguage(language)) {
		return false
	}
	if framework != "" && rule.Applies.Framework != "" && !strings.EqualFold(rule.Applies.Framework, framework) {
		return false
	}
	if surfaceKind != "" && rule.Applies.SurfaceKind != "" && !strings.Contains(rule.Applies.SurfaceKind, surfaceKind) {
		return false
	}
	return true
}

func appliesToSurface(applies Applicability, surface surfaces.Surface) bool {
	if applies.Language != "" && !strings.EqualFold(applies.Language, NormalizeLanguage(surface.Language)) {
		return false
	}
	if applies.Framework != "" && !strings.EqualFold(applies.Framework, surface.Framework) {
		return false
	}
	if applies.SurfaceKind != "" && !strings.Contains(applies.SurfaceKind, surface.Kind) {
		return false
	}
	return true
}

func ruleFinding(ctx EvalContext, rule Rule, path, message, evidence, expected, observed string) Finding {
	return Finding{
		Surface:  ctx.Surface,
		RuleID:   rule.ID,
		Category: rule.Category,
		Severity: rule.Severity,
		FilePath: path,
		Message:  message,
		Evidence: evidence,
		Expected: expected,
		Observed: observed,
		FixClass: rule.FixClass,
	}
}

func evalTSConfig(ctx EvalContext) []Finding {
	rule, _ := ByID(RuleTSConfigStrict)
	path := filepath.Join(ctx.Surface.RootPath, "tsconfig.json")
	content, err := os.ReadFile(path)
	if err != nil {
		// A JavaScript-only surface legitimately has no tsconfig; shared JS/TS
		// rules still run without a spurious TypeScript-config violation.
		if ctx.Surface.Language == "javascript" {
			return nil
		}
		return []Finding{ruleFinding(ctx, rule, path, "tsconfig.json not found", "TypeScript surface has no tsconfig.json.", "strict true, noUncheckedIndexedAccess true, and safety comments", "missing")}
	}
	raw := string(content)
	stripped := StripJSONCComments(raw)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(stripped), &parsed); err != nil {
		return []Finding{ruleFinding(ctx, rule, path, "tsconfig.json parse error", err.Error(), "valid JSONC", "parse error")}
	}
	compiler, _ := parsed["compilerOptions"].(map[string]any)
	var findings []Finding
	if compiler == nil {
		return []Finding{ruleFinding(ctx, rule, path, "Missing compilerOptions", "compilerOptions section is absent.", "compilerOptions with strict and noUncheckedIndexedAccess", "missing")}
	}
	if strict, ok := compiler["strict"].(bool); !ok || !strict {
		findings = append(findings, ruleFinding(ctx, rule, path, "strict mode not enabled", "compilerOptions.strict is not true.", "strict: true", fmt.Sprintf("%v", compiler["strict"])))
	}
	if noUnchecked, ok := compiler["noUncheckedIndexedAccess"].(bool); !ok || !noUnchecked {
		findings = append(findings, ruleFinding(ctx, rule, path, "noUncheckedIndexedAccess not enabled", "compilerOptions.noUncheckedIndexedAccess is not true.", "noUncheckedIndexedAccess: true", fmt.Sprintf("%v", compiler["noUncheckedIndexedAccess"])))
	}
	if !HasTSConfigProtectiveComments(raw) {
		findings = append(findings, ruleFinding(ctx, rule, path, "Missing protective comment block", "Strict settings exist without the guardrail comments that tell agents not to weaken them.", "all required safety comment phrases", "missing comment phrase(s)"))
	}
	return findings
}

func evalESLintSafety(ctx EvalContext) []Finding {
	rule, _ := ByID(RuleESLintSafetyRules)
	path, raw := findESLint(ctx.Surface.RootPath)
	if path == "" {
		return []Finding{ruleFinding(ctx, rule, filepath.Join(ctx.Surface.RootPath, "eslint.config.js"), "ESLint config not found", "Safety-critical lint rules cannot be verified.", "eslint config with safety rules and comments", "missing")}
	}
	var findings []Finding
	semantic := StripJSONCComments(raw)
	if !strings.Contains(raw, "SAFETY-CRITICAL RULES - DO NOT REMOVE, DISABLE, OR WEAKEN") {
		findings = append(findings, ruleFinding(ctx, rule, path, "Missing safety-critical header comment", "ESLint config lacks the guardrail header that tells agents not to disable safety rules.", "safety-critical header comment", "missing"))
	}
	if missing := missingCriticalComments(raw); len(missing) > 0 {
		findings = append(findings, ruleFinding(ctx, rule, path, "Missing per-rule CRITICAL comments", "Missing // CRITICAL comments for "+strings.Join(missing, ", ")+".", "per-rule // CRITICAL comments", "missing"))
	}
	if missing, weak := lintRuleProblems(semantic); len(missing) > 0 || len(weak) > 0 {
		observed := strings.Join(append(prefixAll("missing ", missing), prefixAll("weak ", weak)...), ", ")
		findings = append(findings, ruleFinding(ctx, rule, path, "ESLint safety rules incomplete", "Required safety rules are missing or weaker than the minimum level.", "required safety rules at warn/error minimums", observed))
	}
	return findings
}

func evalESLintTyped(ctx EvalContext) []Finding {
	rule, _ := ByID(RuleESLintTypedConfig)
	path, raw := findESLint(ctx.Surface.RootPath)
	if path == "" {
		return nil
	}
	semantic := StripJSONCComments(raw)
	var missing []string
	if !strings.Contains(semantic, "strictTypeChecked") {
		missing = append(missing, "strictTypeChecked")
	}
	if !strings.Contains(semantic, "parserOptions") || (!strings.Contains(semantic, "project") && !strings.Contains(semantic, "projectService")) {
		missing = append(missing, "parserOptions.project or projectService")
	}
	if strings.Contains(semantic, `"import/no-cycle"`) && (!strings.Contains(semantic, "import/resolver") || !strings.Contains(semantic, "typescript")) {
		missing = append(missing, "TypeScript import resolver")
	}
	if len(missing) == 0 {
		return nil
	}
	return []Finding{ruleFinding(ctx, rule, path, "ESLint typed configuration is incomplete", "Missing "+strings.Join(missing, ", ")+".", "strictTypeChecked, typed parser options, and TS import resolver", strings.Join(missing, ", "))}
}

func evalPackage(ctx EvalContext) []Finding {
	rule, _ := ByID(RuleNodeBuildTypecheck)
	path := filepath.Join(ctx.Surface.RootPath, "package.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var pkg struct {
		Scripts map[string]string `json:"scripts"`
	}
	if err := json.Unmarshal(raw, &pkg); err != nil {
		return []Finding{ruleFinding(ctx, rule, path, "package.json parse error", err.Error(), "valid package.json", "parse error")}
	}
	build := strings.TrimSpace(pkg.Scripts["build"])
	if build == "" {
		return []Finding{ruleFinding(ctx, rule, path, "Missing build script", "The package does not define a build script.", "build script that runs typecheck before bundling", "missing")}
	}
	if !strings.Contains(build, "tsc --noEmit") && !strings.Contains(build, "run type-check") && !strings.Contains(build, "type-check &&") {
		return []Finding{ruleFinding(ctx, rule, path, "Build script skips TypeScript type checking", "Build script is `"+build+"`.", "tsc --noEmit or type-check before bundling", build)}
	}
	return nil
}

// lazyCallPattern matches React's lazy()/React.lazy() call sites without
// matching identifiers that merely end in "lazy" (e.g. isLazy(...)).
var lazyCallPattern = regexp.MustCompile(`\blazy\s*\(`)

// evalLazyChunkRecovery flags Vite UIs that code-split with lazy() but have
// no stale-chunk recovery: after a rebuild the old hashed chunks are gone,
// so tabs opened before the deploy crash into their error boundary on the
// next lazy navigation unless the app reloads itself.
func evalLazyChunkRecovery(ctx EvalContext) []Finding {
	rule, _ := ByID(RuleUILazyChunkRecovery)
	viteConfig := firstExisting(
		filepath.Join(ctx.Surface.RootPath, "vite.config.ts"),
		filepath.Join(ctx.Surface.RootPath, "vite.config.js"),
		filepath.Join(ctx.Surface.RootPath, "vite.config.mts"),
		filepath.Join(ctx.Surface.RootPath, "vite.config.mjs"),
	)
	if viteConfig == "" {
		// Only Vite builds have hashed-chunk churn plus the vite:preloadError hook.
		return nil
	}
	var lazyFiles []string
	guarded := false
	_ = filepath.WalkDir(ctx.Surface.RootPath, func(path string, d os.DirEntry, err error) error {
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
		if strings.Contains(text, "installChunkReloadGuard") || strings.Contains(text, "vite:preloadError") {
			guarded = true
		}
		// Test files exercise lazy() without shipping chunks; skip them.
		base := filepath.Base(path)
		if strings.Contains(base, ".test.") || strings.Contains(base, ".spec.") {
			return nil
		}
		if lazyCallPattern.MatchString(text) {
			lazyFiles = append(lazyFiles, path)
		}
		return nil
	})
	if guarded || len(lazyFiles) == 0 {
		return nil
	}
	sort.Strings(lazyFiles)
	if len(lazyFiles) > 10 {
		lazyFiles = lazyFiles[:10]
	}
	return []Finding{ruleFinding(ctx, rule, lazyFiles[0], "Code-split UI has no stale-chunk recovery", "lazy() call sites: "+strings.Join(lazyFiles, ", ")+".", "installChunkReloadGuard() (from @vrooli/api-base) or a vite:preloadError handler", "no vite:preloadError handling found")}
}

func evalDangerousPatterns(ctx EvalContext) []Finding {
	rule, _ := ByID(RuleTSDangerousPatterns)
	type counts struct{ asAny, asType, ignore, nonNull, total int }
	total := counts{}
	perFile := map[string]counts{}
	_ = filepath.WalkDir(ctx.Surface.RootPath, func(path string, d os.DirEntry, err error) error {
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
			ignore:  countBareTSSuppressions(text),
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
	evidence := fmt.Sprintf("as any=%d, type assertions=%d, bare TS suppressions=%d, non-null assertions=%d, top files=%s", total.asAny, total.asType, total.ignore, total.nonNull, strings.Join(files, ", "))
	return []Finding{ruleFinding(ctx, rule, ctx.Surface.RootPath, "Dangerous TypeScript suppression patterns found", evidence, "zero dangerous suppression patterns", evidence)}
}

func evalGoModPresent(ctx EvalContext) []Finding {
	rule, _ := ByID(RuleGoModPresent)
	if hasGoFiles(ctx.Surface.RootPath) && !exists(filepath.Join(ctx.Surface.RootPath, "go.mod")) {
		return []Finding{ruleFinding(ctx, rule, filepath.Join(ctx.Surface.RootPath, "go.mod"), "Missing go.mod", "Go files exist without a module file.", "go.mod present", "missing")}
	}
	return nil
}

func evalGoLintConfigPresent(ctx EvalContext) []Finding {
	rule, _ := ByID(RuleGoLintConfigPresent)
	configPath := firstExisting(filepath.Join(ctx.Surface.RootPath, ".golangci.yml"), filepath.Join(ctx.Surface.RootPath, ".golangci.yaml"))
	if hasGoFiles(ctx.Surface.RootPath) && configPath == "" {
		return []Finding{ruleFinding(ctx, rule, filepath.Join(ctx.Surface.RootPath, ".golangci.yml"), "Missing golangci-lint config", "Go lint behavior is environment-dependent without checked-in config.", ".golangci.yml/.yaml present", "missing")}
	}
	return nil
}

func evalGoLintRequiredLinters(ctx EvalContext) []Finding {
	rule, _ := ByID(RuleGoLintRequiredLinters)
	configPath := firstExisting(filepath.Join(ctx.Surface.RootPath, ".golangci.yml"), filepath.Join(ctx.Surface.RootPath, ".golangci.yaml"))
	if configPath == "" {
		return nil
	}
	raw, _ := os.ReadFile(configPath)
	missing := missingGoLinters(raw)
	if len(missing) == 0 {
		return nil
	}
	return []Finding{ruleFinding(ctx, rule, configPath, "golangci-lint baseline incomplete", "Missing required linters: "+strings.Join(missing, ", ")+".", "baseline linters enabled", strings.Join(missing, ", "))}
}

func evalGoDangerousPatterns(ctx EvalContext) []Finding {
	rule, _ := ByID(RuleGoDangerousPatterns)
	type suppression struct {
		path string
		line string
	}
	var bare []suppression
	_ = filepath.WalkDir(ctx.Surface.RootPath, func(path string, d os.DirEntry, err error) error {
		if err != nil || d == nil {
			return nil
		}
		if d.IsDir() {
			switch d.Name() {
			case "vendor", "node_modules", ".git", "dist", "build":
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		for _, line := range strings.Split(string(raw), "\n") {
			if idx := strings.Index(line, "//nolint"); idx >= 0 && !hasSuppressionReason(line[idx:]) {
				bare = append(bare, suppression{path: path, line: strings.TrimSpace(line)})
			}
		}
		return nil
	})
	if len(bare) == 0 {
		return nil
	}
	files := map[string]bool{}
	for _, s := range bare {
		files[s.path] = true
	}
	top := make([]string, 0, len(files))
	for path := range files {
		top = append(top, path)
	}
	sort.Strings(top)
	if len(top) > 10 {
		top = top[:10]
	}
	evidence := fmt.Sprintf("bare //nolint directives=%d, top files=%s", len(bare), strings.Join(top, ", "))
	return []Finding{ruleFinding(ctx, rule, ctx.Surface.RootPath, "Bare Go lint suppressions found", evidence, "all //nolint directives include a written reason", evidence)}
}

func evalTestingConfig(ctx EvalContext) []Finding {
	rule, _ := ByID(RuleTestingConfigStrict)
	path := filepath.Join(ctx.Inventory.RootPath, ".vrooli", "testing.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return []Finding{ruleFinding(ctx, rule, path, "testing.json missing lint strictness policy", ".vrooli/testing.json is missing.", "strict node_package/go_module lint handlers", "missing")}
	}
	var cfg struct {
		Lint struct {
			Handlers struct {
				GoModule    strictHandler `json:"go_module"`
				NodePackage strictHandler `json:"node_package"`
			} `json:"handlers"`
		} `json:"lint"`
	}
	if err := json.Unmarshal([]byte(StripJSONCComments(string(raw))), &cfg); err != nil {
		return []Finding{ruleFinding(ctx, rule, path, "testing.json parse error", err.Error(), "valid strict lint config", "parse error")}
	}
	var findings []Finding
	if hasNode(ctx.Inventory) && !cfg.Lint.Handlers.NodePackage.StrictEnabled() {
		findings = append(findings, ruleFinding(ctx, rule, path, "Node lint strict mode not enabled", "node_package strict lint handler is not enabled.", "node_package enabled=true strict=true", "not strict"))
	}
	if hasGo(ctx.Inventory) && !cfg.Lint.Handlers.GoModule.StrictEnabled() {
		findings = append(findings, ruleFinding(ctx, rule, path, "Go lint strict mode not enabled", "go_module strict lint handler is not enabled.", "go_module enabled=true strict=true", "not strict"))
	}
	return findings
}

func evalMakefile(ctx EvalContext) []Finding {
	rule, _ := ByID(RuleMakefileQualityGates)
	path := filepath.Join(ctx.Inventory.RootPath, "Makefile")
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	targets := parseMakefileTargets(string(raw))
	var missing []string
	if hasNode(ctx.Inventory) {
		if !targetRuns(targets, "fmt-ui", "lint:fix") {
			missing = append(missing, "fmt-ui target that runs lint:fix")
		}
		if !targetRuns(targets, "lint-ui", "pnpm run lint", "pnpm run type-check") {
			missing = append(missing, "lint-ui target that runs lint and type-check")
		}
	}
	if hasGo(ctx.Inventory) {
		if !targetRuns(targets, "lint-go", "golangci-lint") {
			missing = append(missing, "lint-go target that runs golangci-lint")
		}
		if !targetRunsAny(targets, "fmt-go", "gofumpt", "gofmt") {
			missing = append(missing, "fmt-go target that runs gofumpt/gofmt")
		}
	}
	if len(missing) == 0 {
		return nil
	}
	return []Finding{ruleFinding(ctx, rule, path, "Makefile quality targets are incomplete", strings.Join(missing, ", "), "real fmt/lint gates for UI and Go surfaces", strings.Join(missing, ", "))}
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

func ESLintRules() []struct {
	Rule string
	Min  string
} {
	return []struct {
		Rule string
		Min  string
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
}

func lintRuleProblems(content string) (missing, weak []string) {
	for _, req := range ESLintRules() {
		escaped := regexp.QuoteMeta(req.Rule)
		re := regexp.MustCompile(`["']` + escaped + `["']\s*:\s*["'](error|warn|off)["']`)
		match := re.FindStringSubmatch(content)
		if len(match) == 0 {
			missing = append(missing, req.Rule)
			continue
		}
		if levelRank(match[1]) < levelRank(req.Min) {
			weak = append(weak, req.Rule+"="+match[1])
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

func missingGoLinters(raw []byte) []string {
	required := RequiredGoLinters()
	var cfg struct {
		Linters struct {
			Enable  []string `yaml:"enable"`
			Disable []string `yaml:"disable"`
		} `yaml:"linters"`
	}
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return required
	}
	enabled := map[string]bool{}
	for _, name := range cfg.Linters.Enable {
		enabled[strings.TrimSpace(name)] = true
	}
	for _, name := range cfg.Linters.Disable {
		delete(enabled, strings.TrimSpace(name))
	}
	var missing []string
	for _, name := range required {
		if !enabled[name] {
			missing = append(missing, name)
		}
	}
	return missing
}

func RequiredGoLinters() []string {
	return []string{"errcheck", "gofumpt", "govet", "ineffassign", "staticcheck", "typecheck", "unused"}
}

func ParseMakefileTargets(content string) map[string]string {
	return parseMakefileTargets(content)
}

func parseMakefileTargets(content string) map[string]string {
	targets := map[string]string{}
	var current string
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "\t") {
			if current != "" {
				targets[current] += "\n" + strings.TrimSpace(line)
			}
			continue
		}
		current = ""
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.Contains(trimmed, "=") {
			continue
		}
		if idx := strings.Index(trimmed, ":"); idx > 0 {
			for _, name := range strings.Fields(trimmed[:idx]) {
				targets[name] = ""
				current = name
			}
		}
	}
	return targets
}

func TargetRuns(targets map[string]string, target string, required ...string) bool {
	return targetRuns(targets, target, required...)
}

func targetRuns(targets map[string]string, target string, required ...string) bool {
	recipe, ok := targets[target]
	if !ok {
		return false
	}
	for _, item := range required {
		if !strings.Contains(recipe, item) {
			return false
		}
	}
	return true
}

func TargetRunsAny(targets map[string]string, target string, required ...string) bool {
	return targetRunsAny(targets, target, required...)
}

func targetRunsAny(targets map[string]string, target string, required ...string) bool {
	recipe, ok := targets[target]
	if !ok {
		return false
	}
	for _, item := range required {
		if strings.Contains(recipe, item) {
			return true
		}
	}
	return false
}

func HasSuppressionReason(comment string) bool {
	return hasSuppressionReason(comment)
}

func hasSuppressionReason(comment string) bool {
	idx := strings.Index(comment, "//")
	if idx < 0 {
		return false
	}
	reason := strings.TrimSpace(comment[idx+2:])
	if strings.HasPrefix(reason, "nolint") {
		if split := strings.SplitN(reason, "//", 2); len(split) == 2 {
			reason = strings.TrimSpace(split[1])
		} else if hash := strings.Index(reason, "#"); hash >= 0 {
			reason = strings.TrimSpace(reason[hash+1:])
		} else {
			return false
		}
	}
	reason = strings.Trim(strings.TrimSpace(reason), ".")
	return len(reason) >= 8
}

func countBareTSSuppressions(text string) int {
	count := 0
	for _, line := range strings.Split(text, "\n") {
		for _, marker := range []string{"@ts-ignore", "@ts-expect-error"} {
			idx := strings.Index(line, marker)
			if idx < 0 {
				continue
			}
			if !hasTSReason(line[idx:]) {
				count++
			}
			break
		}
	}
	return count
}

func hasTSReason(comment string) bool {
	for _, marker := range []string{"@ts-expect-error", "@ts-ignore"} {
		idx := strings.Index(comment, marker)
		if idx < 0 {
			continue
		}
		reason := strings.Trim(strings.TrimSpace(comment[idx+len(marker):]), ".")
		return len(reason) >= 8
	}
	return false
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

func FindESLint(root string) (string, string) {
	return findESLint(root)
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

func HasGoFiles(root string) bool {
	return hasGoFiles(root)
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

func Exists(path string) bool {
	return exists(path)
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func FirstExisting(paths ...string) string {
	return firstExisting(paths...)
}

func firstExisting(paths ...string) string {
	for _, path := range paths {
		if exists(path) {
			return path
		}
	}
	return ""
}

func HasNode(inv surfaces.Inventory) bool {
	return hasNode(inv)
}

func hasNode(inv surfaces.Inventory) bool {
	for _, s := range inv.Surfaces {
		if s.Language == "typescript" || s.Language == "javascript" || exists(filepath.Join(s.RootPath, "package.json")) {
			return true
		}
	}
	return false
}

func HasGo(inv surfaces.Inventory) bool {
	return hasGo(inv)
}

func hasGo(inv surfaces.Inventory) bool {
	for _, s := range inv.Surfaces {
		if s.Language == "go" || exists(filepath.Join(s.RootPath, "go.mod")) {
			return true
		}
	}
	return false
}

func HasTSConfigProtectiveComments(content string) bool {
	for _, phrase := range []string{
		"SAFETY-CRITICAL RULES",
		"DO NOT REMOVE OR WEAKEN",
		"DON'T: Use type assertions (as X)",
		"UI crashes are the #1 production issue",
	} {
		if !strings.Contains(content, phrase) {
			return false
		}
	}
	return true
}

func StripJSONCComments(input string) string {
	var result strings.Builder
	inString := false
	for i := 0; i < len(input); {
		ch := input[i]
		if inString {
			result.WriteByte(ch)
			if ch == '\\' && i+1 < len(input) {
				i++
				result.WriteByte(input[i])
			} else if ch == '"' {
				inString = false
			}
			i++
			continue
		}
		if ch == '"' {
			inString = true
			result.WriteByte(ch)
			i++
			continue
		}
		if ch == '/' && i+1 < len(input) {
			if input[i+1] == '/' {
				for i < len(input) && input[i] != '\n' {
					i++
				}
				continue
			}
			if input[i+1] == '*' {
				i += 2
				for i+1 < len(input) {
					if input[i] == '*' && input[i+1] == '/' {
						i += 2
						break
					}
					i++
				}
				continue
			}
		}
		result.WriteByte(ch)
		i++
	}
	return result.String()
}

func MarshalJSONIndent(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}

func prefixAll(prefix string, values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, prefix+value)
	}
	return out
}
