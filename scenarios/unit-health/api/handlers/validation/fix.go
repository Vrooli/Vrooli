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

	"unit-health/internal/adapters/reactvitest"
	"unit-health/internal/discovery"
	internalvalidation "unit-health/internal/validation"

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
	migrate, ratchet := allow(codeUnitProjectionDrift), allow(codeCoverageCommitmentBehind)
	gate := planCoverageGateFix(root, migrate, ratchet)
	messages = append(messages, gate.messages...)
	if gate.testing != nil {
		candidates = append(candidates, gate.testing)
	}
	if migrate {
		candidates = append(candidates, packageJSONCandidates(root)...)
		candidates = append(candidates, viteConfigCandidates(root, gate.removeNativeThresholds)...)
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

// coverageGateFix is the coverage-gate half of the fix: a testing.json
// candidate that writes coverage.committed, and whether the native Vitest
// thresholds are owned by that commitment and may be deleted.
type coverageGateFix struct {
	testing                *scenariovalidationv1.FixCandidate
	removeNativeThresholds bool
	messages               []string
}

// planCoverageGateFix moves a native Vitest coverage gate into
// unit.policy_profile coverage.committed at the measured baseline (migrate)
// and raises existing commitments to the coverage they have earned (ratchet).
// Native thresholds are removed only once a commitment owns the gate, so the
// fix never leaves a workspace ungated.
func planCoverageGateFix(root string, migrate, ratchet bool) coverageGateFix {
	var fix coverageGateFix
	if !migrate && !ratchet {
		return fix
	}
	uiRoot := filepath.Join(root, "ui")
	manifest, _ := reactvitest.LoadManifest(uiRoot)
	config, _ := readFixFile(reactvitest.EffectiveConfigPath(uiRoot, manifest.Scripts["test:coverage"]))
	projection := reactvitest.Parse(config, "")
	req := internalvalidation.CoverageCommitmentRequest{Ratchet: ratchet}
	if migrate && projection.HasThresholds {
		native := map[string]float64{}
		for _, metric := range []string{"lines", "statements", "functions", "branches"} {
			// A non-literal threshold value still declares a gate on every
			// metric; the measured baseline decides the floor.
			native[metric] = projection.Thresholds[metric]
		}
		req.NativeGates = map[string]map[string]float64{"ui": native}
	}
	plan, err := internalvalidation.PlanCoverageCommitments(root, req)
	if err != nil {
		fix.messages = append(fix.messages, "coverage commitment: "+err.Error())
		return fix
	}
	fix.messages = append(fix.messages, plan.Messages...)
	fix.removeNativeThresholds = migrate && projection.HasThresholds && plan.GateOwned["ui"]
	if plan.After == plan.Before {
		return fix
	}
	rule := codeCoverageCommitmentBehind
	descriptions := make([]string, 0, len(plan.Changes))
	for _, change := range plan.Changes {
		verb := "raise"
		if len(change.From) == 0 {
			verb = "adopt"
			if _, native := req.NativeGates[change.Workspace]; native {
				rule = codeUnitProjectionDrift
			}
		}
		descriptions = append(descriptions, fmt.Sprintf("%s %s (%s) coverage.committed %s from measured %s in %s",
			verb, change.PolicyClass, change.Workspace, internalvalidation.FormatCommitment(change.To), internalvalidation.FormatCommitment(change.Measured), change.Artifact))
	}
	fix.testing = candidate(rule, plan.TestingPath, "Commit coverage floors in unit.policy_profile: "+strings.Join(descriptions, "; ")+".", plan.Before, plan.After)
	return fix
}

// viteConfigCandidates repairs only the Vitest projection fields that drift
// from policy, in the config Vitest actually loads.
func viteConfigCandidates(root string, removeNativeThresholds bool) []*scenariovalidationv1.FixCandidate {
	uiRoot := filepath.Join(root, "ui")
	manifest, _ := reactvitest.LoadManifest(uiRoot)
	path := reactvitest.EffectiveConfigPath(uiRoot, manifest.Scripts["test:coverage"])
	before, ok := readFixFile(path)
	if !ok {
		return nil
	}
	projection := reactvitest.Parse(before, "")
	expect := unitVitestFixProjection(root)
	after := before
	if projection.Environment != expect.environment {
		after, _ = reactvitest.SetTestProperty(after, "environment", "'"+expect.environment+"'")
	}
	if !reactvitest.ContainsAllSetupFiles(projection.SetupFiles, expect.setupFiles) {
		after, _ = reactvitest.EnsureTestStrings(after, "setupFiles", expect.setupFiles)
	}
	if projection.CoverageProvider != expect.coverageProvider {
		after, _ = reactvitest.SetCoverageProperty(after, "provider", "'"+expect.coverageProvider+"'")
	}
	if !reactvitest.ContainsAllStrings(projection.CoverageReporters, expect.coverageReporters) {
		after, _ = reactvitest.EnsureCoverageStrings(after, "reporter", expect.coverageReporters)
	}
	if !reactvitest.ContainsAllStrings(projection.CoverageInclude, expect.coverageInclude) {
		after, _ = reactvitest.EnsureCoverageStrings(after, "include", expect.coverageInclude)
	}
	if !reactvitest.ContainsAllStrings(projection.CoverageExclude, expect.coverageExclude) {
		after, _ = reactvitest.EnsureCoverageStrings(after, "exclude", expect.coverageExclude)
	}
	if !projection.HasReportOnFailure || projection.ReportOnFailure != expect.reportOnFailure {
		after, _ = reactvitest.SetCoverageProperty(after, "reportOnFailure", strconv.FormatBool(expect.reportOnFailure))
	}
	if removeNativeThresholds {
		after, _ = reactvitest.RemoveCoverageThresholds(after)
	}
	if after == before {
		return nil
	}
	return []*scenariovalidationv1.FixCandidate{candidate(codeUnitProjectionDrift, path, "Restore drifted Vitest projection fields from policy and remove native coverage thresholds once coverage.committed owns the gate.", before, after)}
}

type vitestFixProjection struct {
	environment       string
	setupFiles        []string
	coverageProvider  string
	coverageReporters []string
	coverageInclude   []string
	coverageExclude   []string
	reportOnFailure   bool
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
						Provider  string   `json:"provider"`
						Reporters []string `json:"reporters"`
					} `json:"coverage"`
					Projection struct {
						Settings map[string]json.RawMessage `json:"settings"`
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
		settings := reactvitest.PolicyFromSettings(class.Projection.Settings)
		if settings.Environment != "" {
			expect.environment = settings.Environment
		}
		if len(settings.SetupFiles) > 0 {
			expect.setupFiles = settings.SetupFiles
		}
		if class.Coverage.Provider != "" {
			expect.coverageProvider = class.Coverage.Provider
		}
		if len(class.Coverage.Reporters) > 0 {
			expect.coverageReporters = class.Coverage.Reporters
		}
		return expect
	}
	return expect
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
	// The fixer acts on the same projection verdict Unit Health reports.
	if reactvitest.Parse("", before).HasImportBanRule {
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
	codeUnitPolicyInvalid        = "UNIT_POLICY_PROFILE_INVALID"
	codeUnitProjectionDrift      = "UNIT_POLICY_PROJECTION_DRIFT"
	codeCoverageCommitmentBehind = "COVERAGE_COMMITMENT_BEHIND"
)
