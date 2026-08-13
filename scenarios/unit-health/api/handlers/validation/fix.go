package validation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"connectrpc.com/connect"

	"unit-health/internal/discovery"

	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
)

const canonicalTestingSchemaRel = "scenarios/test-genie/schemas/testing.schema.json"

// PreviewFix and ApplyFix implement Unit Health's deterministic low-risk fixes.
// They intentionally cover only config/projection edits that can be described as
// complete file before/after candidates. Dependency installation and behavioral
// test generation stay out of scope.
func (h *SharedHandler) PreviewFix(ctx context.Context, req *connect.Request[scenariovalidationv1.FixRequest]) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	return h.fix(ctx, req, false)
}

func (h *SharedHandler) ApplyFix(ctx context.Context, req *connect.Request[scenariovalidationv1.FixRequest]) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	return h.fix(ctx, req, true)
}

func (h *SharedHandler) fix(ctx context.Context, req *connect.Request[scenariovalidationv1.FixRequest], apply bool) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("fix request is required"))
	}
	scenario, root, err := h.resolveFixTarget(ctx, req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	candidates, messages := collectFixCandidates(root, req.Msg.GetRuleIds())
	if apply {
		for _, c := range candidates {
			if err := applyCandidate(c); err != nil {
				return nil, connect.NewError(connect.CodeFailedPrecondition, err)
			}
			c.Applied = true
		}
	}
	if len(candidates) == 0 {
		messages = append(messages, "no deterministic Unit Health fixes available")
	}
	return connect.NewResponse(&scenariovalidationv1.FixResponse{
		Scenario:   scenario,
		Applied:    apply,
		Candidates: candidates,
		Messages:   messages,
	}), nil
}

func (h *SharedHandler) resolveFixTarget(ctx context.Context, req *scenariovalidationv1.FixRequest) (string, string, error) {
	if h == nil || h.handler == nil {
		return "", "", errors.New("unit validation handler not wired")
	}
	locator := discovery.Locator(discovery.DefaultLocator{})
	if h.handler.svc != nil && h.handler.svc.Locator != nil {
		locator = h.handler.svc.Locator
	}
	scenario, _, root, err := locator.Locate(ctx, req.GetScenario(), req.GetPath())
	if err != nil {
		return "", "", err
	}
	return scenario, root, nil
}

func collectFixCandidates(root string, ruleIDs []string) ([]*scenariovalidationv1.FixCandidate, []string) {
	allow := allowedRules(ruleIDs)
	var candidates []*scenariovalidationv1.FixCandidate
	var messages []string
	if allow(codeUnitPolicyInvalid) {
		candidates = append(candidates, testingSchemaCandidate(root)...)
	}
	if allow(codeUnitProjectionDrift) {
		candidates = append(candidates, packageJSONCandidates(root)...)
		candidates = append(candidates, viteConfigCandidates(root)...)
		candidates = append(candidates, testUtilsCandidates(root)...)
		candidates = append(candidates, eslintImportBanCandidates(root)...)
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].FilePath != candidates[j].FilePath {
			return candidates[i].FilePath < candidates[j].FilePath
		}
		return candidates[i].RuleId < candidates[j].RuleId
	})
	if len(ruleIDs) > 0 && len(candidates) == 0 {
		messages = append(messages, "requested rule id(s) have no deterministic Unit Health fix")
	}
	return candidates, messages
}

func allowedRules(ruleIDs []string) func(string) bool {
	if len(ruleIDs) == 0 {
		return func(string) bool { return true }
	}
	allowed := map[string]bool{}
	for _, id := range ruleIDs {
		allowed[strings.TrimSpace(id)] = true
	}
	return func(rule string) bool { return allowed[rule] }
}

func testingSchemaCandidate(root string) []*scenariovalidationv1.FixCandidate {
	path := filepath.Join(root, ".vrooli", "testing.json")
	before, ok := readFixFile(path)
	if !ok || !strings.Contains(before, "scripts/scenarios/testing/schemas/testing.schema.json") {
		return nil
	}
	repoRoot, err := findRepoRootFrom(root)
	if err != nil {
		return nil
	}
	target := filepath.Join(repoRoot, canonicalTestingSchemaRel)
	rel, err := filepath.Rel(filepath.Dir(path), target)
	if err != nil {
		return nil
	}
	after := strings.ReplaceAll(before, "../../../../scripts/scenarios/testing/schemas/testing.schema.json", filepath.ToSlash(rel))
	return []*scenariovalidationv1.FixCandidate{candidate(codeUnitPolicyInvalid, path, "Normalize testing.json schema reference to the active Test Genie schema.", before, after)}
}

func packageJSONCandidates(root string) []*scenariovalidationv1.FixCandidate {
	path := filepath.Join(root, "ui", "package.json")
	before, ok := readFixFile(path)
	if !ok {
		return nil
	}
	var doc map[string]any
	if err := json.Unmarshal([]byte(before), &doc); err != nil {
		return nil
	}
	scripts, _ := doc["scripts"].(map[string]any)
	if scripts == nil {
		scripts = map[string]any{}
		doc["scripts"] = scripts
	}
	changed := false
	if scriptValue(scripts["test"]) == "" || !strings.Contains(scriptValue(scripts["test"]), "vitest") {
		scripts["test"] = "vitest run"
		changed = true
	}
	if scriptValue(scripts["test:coverage"]) == "" || !strings.Contains(scriptValue(scripts["test:coverage"]), "vitest") || !strings.Contains(scriptValue(scripts["test:coverage"]), "coverage") {
		scripts["test:coverage"] = "vitest run --coverage"
		changed = true
	}
	if !changed {
		return nil
	}
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil
	}
	after := string(raw) + "\n"
	return []*scenariovalidationv1.FixCandidate{candidate(codeUnitProjectionDrift, path, "Restore canonical Vitest test and coverage scripts.", before, after)}
}

func scriptValue(v any) string {
	s, _ := v.(string)
	return s
}

func viteConfigCandidates(root string) []*scenariovalidationv1.FixCandidate {
	path := filepath.Join(root, "ui", "vite.config.ts")
	before, ok := readFixFile(path)
	if !ok {
		path = filepath.Join(root, "ui", "vite.config.js")
		before, ok = readFixFile(path)
		if !ok {
			return nil
		}
	}
	after := before
	expect := unitVitestFixProjection(root)
	after = ensureVitestEnvironment(after, expect.environment)
	after = ensureSetupFiles(after, expect.setupFiles)
	after = ensureCoverageProvider(after, expect.coverageProvider)
	after = ensureCoverageReporters(after, expect.coverageReporters)
	after = ensureCoverageStringArray(after, "include", expect.coverageInclude)
	after = ensureCoverageStringArray(after, "exclude", expect.coverageExclude)
	after = ensureCoverageBoolean(after, "reportOnFailure", expect.reportOnFailure)
	after = raiseCoverageThresholds(after, expect.coverageFloor)
	if after == before {
		return nil
	}
	return []*scenariovalidationv1.FixCandidate{candidate(codeUnitProjectionDrift, path, "Restore policy-declared Vitest environment, setupFiles, coverage provider/reporters, and minimum thresholds where the test block already exists.", before, after)}
}

type vitestFixProjection struct {
	environment       string
	setupFiles        []string
	coverageProvider  string
	coverageReporters []string
	coverageInclude   []string
	coverageExclude   []string
	reportOnFailure   bool
	coverageFloor     float64
}

func unitVitestFixProjection(root string) vitestFixProjection {
	expect := vitestFixProjection{
		environment:       "jsdom",
		setupFiles:        []string{"./src/test-setup.ts"},
		coverageProvider:  "v8",
		coverageReporters: []string{"text", "json-summary", "json"},
		coverageInclude:   []string{"src/**/*.{ts,tsx}"},
		coverageExclude: []string{
			"src/**/*.test.{ts,tsx}",
			"src/**/*.spec.{ts,tsx}",
			"src/**/*.d.ts",
			"src/main.tsx",
			"src/test-setup.ts",
			"src/test-utils/**",
			"src/consts/strings.generated.ts",
			"src/i18n/locales/**",
			"src/**/generated/**",
		},
		reportOnFailure: true,
		coverageFloor:   85,
	}
	raw, err := os.ReadFile(filepath.Join(root, ".vrooli", "testing.json"))
	if err != nil {
		return expect
	}
	var doc struct {
		Unit struct {
			PolicyProfile struct {
				RequiredRoles []struct {
					Role        string `json:"role"`
					PolicyClass string `json:"policy_class"`
				} `json:"required_roles"`
				PolicyClasses map[string]struct {
					Coverage struct {
						MinimumPercent float64  `json:"minimum_percent"`
						Provider       string   `json:"provider"`
						Reporters      []string `json:"reporters"`
					} `json:"coverage"`
					Projection struct {
						Vitest struct {
							Environment string   `json:"environment"`
							SetupFiles  []string `json:"setup_files"`
						} `json:"vitest"`
					} `json:"projection"`
				} `json:"policy_classes"`
			} `json:"policy_profile"`
		} `json:"unit"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return expect
	}
	for _, role := range doc.Unit.PolicyProfile.RequiredRoles {
		if role.Role != "ui" {
			continue
		}
		class, ok := doc.Unit.PolicyProfile.PolicyClasses[role.PolicyClass]
		if !ok {
			return expect
		}
		if class.Projection.Vitest.Environment != "" {
			expect.environment = class.Projection.Vitest.Environment
		}
		if len(class.Projection.Vitest.SetupFiles) > 0 {
			expect.setupFiles = class.Projection.Vitest.SetupFiles
		}
		if class.Coverage.Provider != "" {
			expect.coverageProvider = class.Coverage.Provider
		}
		if len(class.Coverage.Reporters) > 0 {
			expect.coverageReporters = class.Coverage.Reporters
		}
		if class.Coverage.MinimumPercent > 0 {
			expect.coverageFloor = class.Coverage.MinimumPercent
		}
		return expect
	}
	return expect
}

func ensureVitestEnvironment(src, environment string) string {
	if environment == "" || !strings.Contains(src, "test:") {
		return src
	}
	re := regexp.MustCompile(`(environment\s*:\s*)['"][^'"]+['"]`)
	if re.MatchString(src) {
		return re.ReplaceAllString(src, "${1}'"+environment+"'")
	}
	return insertAfterObjectOpen(src, "test", "environment: '"+environment+"',")
}

func ensureSetupFiles(src string, setupFiles []string) string {
	if len(setupFiles) == 0 || !strings.Contains(src, "test:") {
		return src
	}
	value := formatStringArray(setupFiles)
	re := regexp.MustCompile(`(?s)(setupFiles\s*:\s*)(\[[^\]]*\]|['"][^'"]+['"])`)
	if re.MatchString(src) {
		return re.ReplaceAllString(src, "${1}"+value)
	}
	lines := strings.SplitAfter(src, "\n")
	for i, line := range lines {
		if !strings.Contains(line, "environment") {
			continue
		}
		indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
		lines = append(lines[:i+1], append([]string{indent + "setupFiles: " + value + ",\n"}, lines[i+1:]...)...)
		return strings.Join(lines, "")
	}
	return insertAfterObjectOpen(src, "test", "setupFiles: "+value+",")
}

func ensureCoverageProvider(src, provider string) string {
	if provider == "" || !strings.Contains(src, "coverage:") {
		return src
	}
	re := regexp.MustCompile(`(provider\s*:\s*)['"][^'"]+['"]`)
	if re.MatchString(src) {
		return re.ReplaceAllString(src, "${1}'"+provider+"'")
	}
	return insertAfterObjectOpen(src, "coverage", "provider: '"+provider+"',")
}

func ensureCoverageReporters(src string, reporters []string) string {
	if len(reporters) == 0 || !strings.Contains(src, "coverage:") {
		return src
	}
	return ensureCoverageStringArray(src, "reporter", reporters)
}

func ensureCoverageStringArray(src, property string, values []string) string {
	if len(values) == 0 || !strings.Contains(src, "coverage:") {
		return src
	}
	value := formatStringArray(values)
	re := regexp.MustCompile(`(?s)(` + regexp.QuoteMeta(property) + `\s*:\s*)(\[[^\]]*\]|['"][^'"]+['"])`)
	if re.MatchString(src) {
		return re.ReplaceAllString(src, "${1}"+value)
	}
	return insertAfterObjectOpen(src, "coverage", property+": "+value+",")
}

func ensureCoverageBoolean(src, property string, value bool) string {
	if !strings.Contains(src, "coverage:") {
		return src
	}
	rendered := strconv.FormatBool(value)
	re := regexp.MustCompile(`(` + regexp.QuoteMeta(property) + `\s*:\s*)(true|false)`)
	if re.MatchString(src) {
		return re.ReplaceAllString(src, "${1}"+rendered)
	}
	return insertAfterObjectOpen(src, "coverage", property+": "+rendered+",")
}

func insertAfterObjectOpen(src, objectName, propertyLine string) string {
	re := regexp.MustCompile(`(?m)^([ \t]*)` + regexp.QuoteMeta(objectName) + `\s*:\s*\{\s*$`)
	loc := re.FindStringSubmatchIndex(src)
	if loc == nil {
		return src
	}
	indent := src[loc[2]:loc[3]] + "  "
	insertAt := loc[1]
	return src[:insertAt] + "\n" + indent + propertyLine + src[insertAt:]
}

func formatStringArray(values []string) string {
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, "'"+strings.ReplaceAll(value, "'", `\'`)+"'")
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}

func raiseCoverageThresholds(src string, floor float64) string {
	for _, key := range []string{"lines", "functions", "branches", "statements"} {
		re := regexp.MustCompile(`(` + key + `\s*:\s*)([0-9]+(?:\.[0-9]+)?)`)
		src = re.ReplaceAllStringFunc(src, func(match string) string {
			parts := re.FindStringSubmatch(match)
			if len(parts) != 3 {
				return match
			}
			v, err := strconv.ParseFloat(parts[2], 64)
			if err != nil || v >= floor {
				return match
			}
			return parts[1] + formatCoverageFloor(floor)
		})
	}
	return src
}

func testUtilsCandidates(root string) []*scenariovalidationv1.FixCandidate {
	var candidates []*scenariovalidationv1.FixCandidate
	// A package-owned helper is the canonical projection. Do not offer the
	// legacy fix that recreates a scenario-local implementation after a
	// migration has deliberately removed it.
	if workspaceUsesSharedRenderHelper(filepath.Join(root, "ui")) {
		return nil
	}
	renderPath := filepath.Join(root, "ui", "src", "test-utils", "renderWithProviders.tsx")
	before, exists := readFixFile(renderPath)
	if !exists {
		candidates = append(candidates, candidate(codeUnitProjectionDrift, renderPath, "Create the canonical provider-aware render helper.", "", canonicalRenderWithProviders()))
	} else if !strings.Contains(before, "renderWithProviders") {
		candidates = append(candidates, candidate(codeUnitProjectionDrift, renderPath, "Restore the canonical renderWithProviders export.", before, canonicalRenderWithProviders()))
	}

	indexPath := filepath.Join(root, "ui", "src", "test-utils", "index.ts")
	indexBefore, indexExists := readFixFile(indexPath)
	indexAfter := indexBefore
	if !strings.Contains(indexAfter, "renderWithProviders") {
		if strings.TrimSpace(indexAfter) != "" && !strings.HasSuffix(indexAfter, "\n") {
			indexAfter += "\n"
		}
		indexAfter += "export { renderWithProviders } from \"./renderWithProviders\";\n"
		indexAfter += "export type { ProviderRenderOptions, ProviderRenderResult } from \"./renderWithProviders\";\n"
	}
	if !indexExists || indexAfter != indexBefore {
		candidates = append(candidates, candidate(codeUnitProjectionDrift, indexPath, "Re-export the canonical render helper from src/test-utils.", indexBefore, indexAfter))
	}
	return candidates
}

func workspaceUsesSharedRenderHelper(root string) bool {
	found := false
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || found || info == nil || info.IsDir() {
			return nil
		}
		switch filepath.Ext(path) {
		case ".ts", ".tsx", ".js", ".jsx", ".mts", ".cts":
			if strings.Contains(readFixFileContents(path), "@vrooli/api-base/testing") && strings.Contains(readFixFileContents(path), "renderWithProviders") {
				found = true
			}
		}
		return nil
	})
	return found
}

func readFixFileContents(path string) string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(raw)
}

func canonicalRenderWithProviders() string {
	return `import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { type RenderOptions, type RenderResult, render } from "@testing-library/react";
import type { ReactElement, ReactNode } from "react";
import { I18nextProvider } from "react-i18next";
import { MemoryRouter } from "react-router-dom";

import { i18n } from "../i18n";
import { ThemeProvider, type ThemeChoice } from "../theme/ThemeProvider";

export interface ProviderRenderOptions extends Omit<RenderOptions, "wrapper"> {
  queryClient?: QueryClient;
  routerEntries?: string[];
  initialTheme?: ThemeChoice;
  withoutRouter?: boolean;
}

export interface ProviderRenderResult extends RenderResult {
  queryClient: QueryClient;
}

const buildClient = (): QueryClient =>
  new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });

export function renderWithProviders(
  ui: ReactElement,
  options: ProviderRenderOptions = {},
): ProviderRenderResult {
  const {
    queryClient = buildClient(),
    routerEntries = ["/"],
    initialTheme = "light",
    withoutRouter = false,
    ...rest
  } = options;

  const Wrapper = ({ children }: { children: ReactNode }) => {
    const themed = (
      <ThemeProvider initialChoice={initialTheme}>{children}</ThemeProvider>
    );
    const routed = withoutRouter ? themed : (
      <MemoryRouter initialEntries={routerEntries}>{themed}</MemoryRouter>
    );
    return (
      <QueryClientProvider client={queryClient}>
        <I18nextProvider i18n={i18n}>{routed}</I18nextProvider>
      </QueryClientProvider>
    );
  };

  return { ...render(ui, { wrapper: Wrapper, ...rest }), queryClient };
}
`
}

func eslintImportBanCandidates(root string) []*scenariovalidationv1.FixCandidate {
	path := filepath.Join(root, "ui", "eslint.config.js")
	before, exists := readFixFile(path)
	if !exists {
		return []*scenariovalidationv1.FixCandidate{candidate(codeUnitProjectionDrift, path, "Create the production import ban for test-only UI helpers.", "", canonicalESLintImportBanConfig())}
	}
	if hasCanonicalESLintImportBan(before) {
		return nil
	}
	after := insertNoRestrictedImportsRule(before)
	if after == before {
		return nil
	}
	return []*scenariovalidationv1.FixCandidate{candidate(codeUnitProjectionDrift, path, "Restore the production import ban for test-only UI helpers.", before, after)}
}

func canonicalESLintImportBanConfig() string {
	return `export default [{
  rules: {
    "no-restricted-imports": ["error", {
      patterns: [{
        group: [
          "**/test-utils",
          "**/test-utils/*",
          "@/test-utils",
          "@/test-utils/*",
          "**/features/*/mocks",
          "**/features/*/mocks/*",
          "@/features/*/mocks",
          "@/features/*/mocks/*",
        ],
      }],
    }],
  },
}];
`
}

func insertNoRestrictedImportsRule(src string) string {
	re := regexp.MustCompile(`(?m)^([ \t]*)rules\s*:\s*\{\s*$`)
	loc := re.FindStringSubmatchIndex(src)
	if loc == nil {
		return src
	}
	indent := src[loc[2]:loc[3]] + "  "
	insertAt := loc[1]
	return src[:insertAt] + "\n" + indent + `"no-restricted-imports": ["error", { patterns: [{ group: ["**/test-utils", "**/test-utils/*", "@/test-utils", "@/test-utils/*", "**/features/*/mocks", "**/features/*/mocks/*", "@/features/*/mocks", "@/features/*/mocks/*"] }] }],` + src[insertAt:]
}

func hasCanonicalESLintImportBan(src string) bool {
	clean := stripFixJSComments(src)
	return containsQuotedFixLiteral(clean, "no-restricted-imports") &&
		containsQuotedFixLiteral(clean, "**/test-utils") &&
		containsQuotedFixLiteral(clean, "@/test-utils/*") &&
		containsQuotedFixLiteral(clean, "**/features/*/mocks") &&
		containsQuotedFixLiteral(clean, "@/features/*/mocks/*")
}

func containsQuotedFixLiteral(src, value string) bool {
	return strings.Contains(src, `"`+value+`"`) ||
		strings.Contains(src, `'`+value+`'`) ||
		strings.Contains(src, "`"+value+"`")
}

func stripFixJSComments(src string) string {
	var b strings.Builder
	inLineComment := false
	inBlockComment := false
	var quote byte
	escaped := false
	for i := 0; i < len(src); i++ {
		c := src[i]
		if inLineComment {
			if c == '\n' {
				inLineComment = false
				b.WriteByte(c)
			}
			continue
		}
		if inBlockComment {
			if c == '*' && i+1 < len(src) && src[i+1] == '/' {
				inBlockComment = false
				i++
			}
			continue
		}
		if quote != 0 {
			b.WriteByte(c)
			if escaped {
				escaped = false
				continue
			}
			if c == '\\' {
				escaped = true
				continue
			}
			if c == quote {
				quote = 0
			}
			continue
		}
		if c == '/' && i+1 < len(src) {
			switch src[i+1] {
			case '/':
				inLineComment = true
				i++
				continue
			case '*':
				inBlockComment = true
				i++
				continue
			}
		}
		if c == '\'' || c == '"' || c == '`' {
			quote = c
		}
		b.WriteByte(c)
	}
	return b.String()
}

func formatCoverageFloor(floor float64) string {
	if floor == float64(int64(floor)) {
		return strconv.FormatInt(int64(floor), 10)
	}
	return strconv.FormatFloat(floor, 'f', -1, 64)
}

func applyCandidate(c *scenariovalidationv1.FixCandidate) error {
	current, err := os.ReadFile(c.GetFilePath())
	if err != nil && !(os.IsNotExist(err) && c.GetBefore() == "") {
		return fmt.Errorf("read %s before applying fix: %w", c.GetFilePath(), err)
	}
	if string(current) != c.GetBefore() {
		return fmt.Errorf("refusing to apply fix for %s: file changed since preview", c.GetFilePath())
	}
	if err := os.MkdirAll(filepath.Dir(c.GetFilePath()), 0o755); err != nil {
		return fmt.Errorf("create parent for %s: %w", c.GetFilePath(), err)
	}
	if err := os.WriteFile(c.GetFilePath(), []byte(c.GetAfter()), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", c.GetFilePath(), err)
	}
	return nil
}

func candidate(rule, path, desc, before, after string) *scenariovalidationv1.FixCandidate {
	return &scenariovalidationv1.FixCandidate{
		RuleId:      rule,
		FilePath:    path,
		Description: desc,
		Before:      before,
		After:       after,
	}
}

func readFixFile(path string) (string, bool) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	return string(raw), true
}

func findRepoRootFrom(start string) (string, error) {
	dir := filepath.Clean(start)
	for {
		if _, err := os.Stat(filepath.Join(dir, "scenarios", "test-genie", "schemas", "testing.schema.json")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("repo root with %s not found from %s", canonicalTestingSchemaRel, start)
		}
		dir = parent
	}
}

const (
	codeUnitPolicyInvalid   = "UNIT_POLICY_PROFILE_INVALID"
	codeUnitProjectionDrift = "UNIT_POLICY_PROJECTION_DRIFT"
)
