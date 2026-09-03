package autofix

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"quality-health/internal/contracts"
	"quality-health/internal/rules"

	autofixcore "github.com/vrooli/maturity-go/autofix"
	"gopkg.in/yaml.v3"
)

const TSConfigProtectiveCommentBlock = `    // SAFETY-CRITICAL RULES - DO NOT REMOVE OR WEAKEN
    // These rules prevent runtime crashes like "X is not a function".
    // DO: Use optional chaining (?.), null checks, nullish coalescing (??), and type guards.
    // DON'T: Use type assertions (as X), non-null assertions (!), @ts-ignore, or weaken these rules.
    // These rules exist because UI crashes are the #1 production issue.
`

// Candidate aliases the shared auto-fix candidate so quality-health consumers
// keep a single source of truth with the maturity-go/autofix orchestrator.
type Candidate = autofixcore.Candidate

// registry is the quality-health fixer set bound to the shared orchestrator.
var registry = autofixcore.NewRegistry(
	autofixcore.Fixer{RuleID: contracts.RuleTSConfigStrict, Preview: previewTSConfigStrict, CanFix: canFixTSConfigStrict},
	autofixcore.Fixer{RuleID: contracts.RuleESLintSafetyRules, Preview: previewESLintSafetyRules, CanFix: canFixESLint},
	autofixcore.Fixer{RuleID: contracts.RuleESLintTypedConfig, Preview: previewESLintTypedConfig, CanFix: canFixESLint},
	autofixcore.Fixer{RuleID: contracts.RuleTestingConfigStrict, Preview: previewTestingConfigStrict, CanFix: canFixTestingConfigStrict},
	autofixcore.Fixer{RuleID: contracts.RuleGoModPresent, Preview: previewGoModPresent, CanFix: canFixGoModPresent},
	autofixcore.Fixer{RuleID: contracts.RuleGoLintConfigPresent, Preview: previewGoLintConfigPresent, CanFix: canFixGoLintConfigPresent},
	autofixcore.Fixer{RuleID: contracts.RuleGoLintRequiredLinters, Preview: previewGoLintRequiredLinters, CanFix: canFixGoLintRequiredLinters},
	autofixcore.Fixer{RuleID: contracts.RuleMakefileQualityGates, Preview: previewMakefileQualityGates, CanFix: canFixMakefileQualityGates},
)

// Preview returns the candidate edits for the requested rules without writing.
func Preview(root string, ruleIDs []string) ([]Candidate, error) {
	return registry.Preview(root, ruleIDs)
}

// Apply previews then writes the candidate edits for the requested rules.
func Apply(root string, ruleIDs []string) ([]Candidate, error) {
	return registry.Apply(root, ruleIDs)
}

// CanFix reports whether the rule can currently remediate the given finding.
func CanFix(root, ruleID, findingPath string) bool {
	return registry.CanFix(root, ruleID, findingPath)
}

func previewTSConfigStrict(root string) ([]Candidate, error) {
	var out []Candidate
	for _, path := range candidateFiles(root, "tsconfig.json", []string{"ui"}) {
		before, after, changed, err := fixedTSConfig(path)
		if err != nil || !changed {
			continue
		}
		out = append(out, Candidate{
			RuleID:      contracts.RuleTSConfigStrict,
			FilePath:    path,
			Description: "Enable strict TypeScript settings and restore the safety-critical guardrail comment block.",
			Before:      before,
			After:       after,
		})
	}
	return out, nil
}

func canFixTSConfigStrict(root, findingPath string) bool {
	path := findingPathOrDefault(root, findingPath, filepath.Join("ui", "tsconfig.json"))
	_, _, changed, err := fixedTSConfig(path)
	return err == nil && changed
}

func fixedTSConfig(path string) (string, string, bool, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", "", false, fmt.Errorf("read tsconfig: %w", err)
	}
	before := string(raw)
	stripped := StripJSONCComments(before)
	var parsed map[string]any
	if err := json.Unmarshal([]byte(stripped), &parsed); err != nil {
		return before, before, false, fmt.Errorf("parse tsconfig: %w", err)
	}
	compiler, _ := parsed["compilerOptions"].(map[string]any)
	if compiler == nil {
		compiler = map[string]any{}
		parsed["compilerOptions"] = compiler
	}
	changed := false
	if strict, ok := compiler["strict"].(bool); !ok || !strict {
		compiler["strict"] = true
		changed = true
	}
	if noUnchecked, ok := compiler["noUncheckedIndexedAccess"].(bool); !ok || !noUnchecked {
		compiler["noUncheckedIndexedAccess"] = true
		changed = true
	}
	after := before
	if changed {
		out, err := marshalJSONIndent(parsed)
		if err != nil {
			return before, before, false, err
		}
		after = string(out) + "\n"
	}
	if !HasTSConfigProtectiveComments(after) {
		after = injectTSConfigComment(after)
		changed = true
	}
	return before, after, changed, nil
}

func previewESLintSafetyRules(root string) ([]Candidate, error) {
	return previewESLint(root, contracts.RuleESLintSafetyRules, "Add the safety-critical ESLint rule baseline and guardrail comments.")
}

func previewESLintTypedConfig(root string) ([]Candidate, error) {
	return previewESLint(root, contracts.RuleESLintTypedConfig, "Add typed ESLint configuration with strict type-checked parsing.")
}

func previewESLint(root, ruleID, description string) ([]Candidate, error) {
	var out []Candidate
	for _, path := range eslintConfigPaths(root) {
		before, after, changed, err := fixedESLint(path)
		if err != nil || !changed {
			continue
		}
		out = append(out, Candidate{RuleID: ruleID, FilePath: path, Description: description, Before: before, After: after})
	}
	return out, nil
}

func canFixESLint(root, findingPath string) bool {
	path := findingPathOrDefault(root, findingPath, filepath.Join("ui", "eslint.config.js"))
	_, _, changed, err := fixedESLint(path)
	return err == nil && changed
}

func fixedESLint(path string) (string, string, bool, error) {
	beforeBytes, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return "", "", false, err
	}
	before := string(beforeBytes)
	after := strings.TrimRight(before, "\n")
	if strings.TrimSpace(after) == "" {
		after = baselineESLintConfig()
		return before, after, true, nil
	}
	additions := []string{}
	for _, phrase := range []string{
		"SAFETY-CRITICAL RULES - DO NOT REMOVE, DISABLE, OR WEAKEN",
		"strictTypeChecked",
		"parserOptions",
		"projectService",
		"react-hooks/rules-of-hooks",
		"@typescript-eslint/no-non-null-assertion",
		"@typescript-eslint/no-explicit-any",
		"@typescript-eslint/no-unsafe-member-access",
		"@typescript-eslint/no-unsafe-call",
		"@typescript-eslint/no-unsafe-argument",
		"@typescript-eslint/no-unsafe-assignment",
		"@typescript-eslint/no-unsafe-return",
		"import/no-cycle",
		"import/resolver",
	} {
		if !strings.Contains(after, phrase) {
			additions = append(additions, phrase)
		}
	}
	if len(additions) == 0 {
		return before, before, false, nil
	}
	after += "\n\n" + baselineESLintConfig()
	return before, after, after != before, nil
}

func baselineESLintConfig() string {
	return `// SAFETY-CRITICAL RULES - DO NOT REMOVE, DISABLE, OR WEAKEN
// This block is managed by Quality Health as a minimum static-quality baseline.
export default [
  {
    languageOptions: {
      parserOptions: {
        projectService: true,
      },
    },
    extends: ["strictTypeChecked"],
    settings: {
      "import/resolver": {
        typescript: true,
      },
    },
    rules: {
      // CRITICAL: hooks must be enforced to prevent invalid React execution.
      "react-hooks/rules-of-hooks": "error",
      // CRITICAL: non-null assertions bypass runtime safety.
      "@typescript-eslint/no-non-null-assertion": "error",
      // CRITICAL: explicit any hides type contract breaks.
      "@typescript-eslint/no-explicit-any": "error",
      // CRITICAL: unsafe member/call/argument/assignment/return checks catch runtime crashes.
      "@typescript-eslint/no-unsafe-member-access": "warn",
      "@typescript-eslint/no-unsafe-call": "warn",
      "@typescript-eslint/no-unsafe-argument": "warn",
      "@typescript-eslint/no-unsafe-assignment": "warn",
      "@typescript-eslint/no-unsafe-return": "warn",
      // CRITICAL: cycles destabilize module initialization.
      "import/no-cycle": "error",
    },
  },
];
`
}

func previewTestingConfigStrict(root string) ([]Candidate, error) {
	path := filepath.Join(root, ".vrooli", "testing.json")
	before, after, changed, err := fixedTestingConfig(path, hasNodeSurface(root), hasGoSurface(root))
	if err != nil || !changed {
		return nil, err
	}
	return []Candidate{{
		RuleID:      contracts.RuleTestingConfigStrict,
		FilePath:    path,
		Description: "Enable strict Test Genie lint handlers for discovered Node and Go surfaces.",
		Before:      before,
		After:       after,
	}}, nil
}

func canFixTestingConfigStrict(root, findingPath string) bool {
	path := findingPathOrDefault(root, findingPath, filepath.Join(".vrooli", "testing.json"))
	_, _, changed, err := fixedTestingConfig(path, hasNodeSurface(root), hasGoSurface(root))
	return err == nil && changed
}

func fixedTestingConfig(path string, node, goMod bool) (string, string, bool, error) {
	raw, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return "", "", false, err
	}
	before := string(raw)
	cfg := map[string]any{}
	if strings.TrimSpace(before) != "" {
		if err := json.Unmarshal([]byte(StripJSONCComments(before)), &cfg); err != nil {
			return before, before, false, err
		}
	}
	lint := ensureMap(cfg, "lint")
	handlers := ensureMap(lint, "handlers")
	if node {
		handlers["node_package"] = map[string]any{"enabled": true, "strict": true}
	}
	if goMod {
		handlers["go_module"] = map[string]any{"enabled": true, "strict": true}
	}
	out, err := marshalJSONIndent(cfg)
	if err != nil {
		return before, before, false, err
	}
	after := string(out) + "\n"
	return before, after, after != before, nil
}

func previewGoModPresent(root string) ([]Candidate, error) {
	var out []Candidate
	for _, dir := range goSurfaceDirs(root) {
		path := filepath.Join(dir, "go.mod")
		before, after, changed, err := fixedGoMod(path, filepath.Base(dir))
		if err != nil || !changed {
			continue
		}
		out = append(out, Candidate{
			RuleID:      contracts.RuleGoModPresent,
			FilePath:    path,
			Description: "Create a minimal Go module file for the Go surface.",
			Before:      before,
			After:       after,
		})
	}
	return out, nil
}

func canFixGoModPresent(root, findingPath string) bool {
	path := findingPathOrDefault(root, findingPath, filepath.Join("api", "go.mod"))
	_, after, changed, err := fixedGoMod(path, filepath.Base(filepath.Dir(path)))
	return err == nil && changed && strings.TrimSpace(after) != ""
}

func fixedGoMod(path, moduleName string) (string, string, bool, error) {
	raw, err := os.ReadFile(path)
	if err == nil {
		return string(raw), string(raw), false, nil
	}
	if !os.IsNotExist(err) {
		return "", "", false, err
	}
	if moduleName == "" || moduleName == "." || moduleName == string(filepath.Separator) {
		moduleName = "scenario"
	}
	return "", fmt.Sprintf("module %s\n\ngo 1.25\n", sanitizeModuleName(moduleName)), true, nil
}

func previewGoLintConfigPresent(root string) ([]Candidate, error) {
	var out []Candidate
	for _, dir := range goSurfaceDirs(root) {
		path := filepath.Join(dir, ".golangci.yml")
		before, after, changed, err := fixedGoLintConfig(path)
		if err != nil || !changed {
			continue
		}
		out = append(out, Candidate{
			RuleID:      contracts.RuleGoLintConfigPresent,
			FilePath:    path,
			Description: "Create a baseline golangci-lint configuration.",
			Before:      before,
			After:       after,
		})
	}
	return out, nil
}

func canFixGoLintConfigPresent(root, findingPath string) bool {
	path := findingPathOrDefault(root, findingPath, filepath.Join("api", ".golangci.yml"))
	_, _, changed, err := fixedGoLintConfig(path)
	return err == nil && changed
}

func fixedGoLintConfig(path string) (string, string, bool, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return "", "", false, err
		}
		return "", baselineGoLintConfig(), true, nil
	}
	before := string(raw)
	var cfg goLintConfig
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return before, before, false, err
	}
	changed := ensureGoLinters(&cfg)
	if !changed {
		return before, before, false, nil
	}
	afterBytes, err := yaml.Marshal(&cfg)
	if err != nil {
		return before, before, false, err
	}
	return before, string(afterBytes), true, nil
}

func previewGoLintRequiredLinters(root string) ([]Candidate, error) {
	var out []Candidate
	for _, path := range existingGoLintConfigs(root) {
		before, after, changed, err := fixedGoLintConfig(path)
		if err != nil || !changed {
			continue
		}
		out = append(out, Candidate{
			RuleID:      contracts.RuleGoLintRequiredLinters,
			FilePath:    path,
			Description: "Enable the required golangci-lint baseline linters.",
			Before:      before,
			After:       after,
		})
	}
	return out, nil
}

func canFixGoLintRequiredLinters(root, findingPath string) bool {
	path := findingPathOrDefault(root, findingPath, filepath.Join("api", ".golangci.yml"))
	_, _, changed, err := fixedGoLintConfig(path)
	return err == nil && changed
}

type goLintConfig struct {
	Linters struct {
		Enable  []string `yaml:"enable,omitempty"`
		Disable []string `yaml:"disable,omitempty"`
	} `yaml:"linters,omitempty"`
}

func ensureGoLinters(cfg *goLintConfig) bool {
	required := requiredGoLinters()
	enabled := map[string]bool{}
	for _, name := range cfg.Linters.Enable {
		enabled[strings.TrimSpace(name)] = true
	}
	disabled := map[string]bool{}
	for _, name := range cfg.Linters.Disable {
		disabled[strings.TrimSpace(name)] = true
	}
	changed := false
	for _, name := range required {
		if !enabled[name] {
			cfg.Linters.Enable = append(cfg.Linters.Enable, name)
			changed = true
		}
		if disabled[name] {
			delete(disabled, name)
			changed = true
		}
	}
	var disable []string
	for name := range disabled {
		if name != "" {
			disable = append(disable, name)
		}
	}
	sort.Strings(disable)
	cfg.Linters.Disable = disable
	sort.Strings(cfg.Linters.Enable)
	return changed
}

func baselineGoLintConfig() string {
	return "linters:\n  enable:\n    - " + strings.Join(requiredGoLinters(), "\n    - ") + "\n"
}

func requiredGoLinters() []string {
	return rules.RequiredGoLinters()
}

func previewMakefileQualityGates(root string) ([]Candidate, error) {
	path := filepath.Join(root, "Makefile")
	before, after, changed, err := fixedMakefile(path, hasNodeSurface(root), hasGoSurface(root))
	if err != nil || !changed {
		return nil, err
	}
	return []Candidate{{
		RuleID:      contracts.RuleMakefileQualityGates,
		FilePath:    path,
		Description: "Add missing scenario-level format and lint quality targets.",
		Before:      before,
		After:       after,
	}}, nil
}

func canFixMakefileQualityGates(root, findingPath string) bool {
	path := findingPathOrDefault(root, findingPath, "Makefile")
	_, _, changed, err := fixedMakefile(path, hasNodeSurface(root), hasGoSurface(root))
	return err == nil && changed
}

func fixedMakefile(path string, node, goMod bool) (string, string, bool, error) {
	raw, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return "", "", false, err
	}
	before := string(raw)
	targets := parseMakefileTargets(before)
	additions := []string{}
	if node {
		if !targetRuns(targets, "fmt-ui", "lint:fix") {
			additions = append(additions, "fmt-ui:\n\tpnpm run lint:fix\n")
		}
		if !targetRuns(targets, "lint-ui", "pnpm run lint", "pnpm run type-check") {
			additions = append(additions, "lint-ui:\n\tpnpm run lint\n\tpnpm run type-check\n")
		}
	}
	if goMod {
		if !targetRuns(targets, "lint-go", "golangci-lint") {
			additions = append(additions, "lint-go:\n\tgolangci-lint run ./...\n")
		}
		if !targetRunsAny(targets, "fmt-go", "gofumpt", "gofmt") {
			additions = append(additions, "fmt-go:\n\tgofumpt -w .\n")
		}
	}
	if len(additions) == 0 {
		return before, before, false, nil
	}
	after := strings.TrimRight(before, "\n")
	if after != "" {
		after += "\n\n"
	}
	after += strings.Join(additions, "\n")
	return before, after, true, nil
}

func HasTSConfigProtectiveComments(content string) bool {
	return rules.HasTSConfigProtectiveComments(content)
}

func StripJSONCComments(input string) string {
	return rules.StripJSONCComments(input)
}

func marshalJSONIndent(v any) ([]byte, error) {
	return rules.MarshalJSONIndent(v)
}

func injectTSConfigComment(content string) string {
	if idx := strings.Index(content, `"strict"`); idx >= 0 {
		lineStart := strings.LastIndex(content[:idx], "\n") + 1
		return content[:lineStart] + TSConfigProtectiveCommentBlock + content[lineStart:]
	}
	if idx := strings.Index(content, `"compilerOptions"`); idx >= 0 {
		if brace := strings.Index(content[idx:], "{"); brace >= 0 {
			insertAt := idx + brace + 1
			return content[:insertAt] + "\n" + TSConfigProtectiveCommentBlock + content[insertAt:]
		}
	}
	return content
}

func candidateFiles(root, name string, conventionalDirs []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, dir := range conventionalDirs {
		path := filepath.Join(root, dir, name)
		if seen[path] {
			continue
		}
		seen[path] = true
		if _, err := os.Stat(filepath.Dir(path)); err == nil {
			out = append(out, path)
		}
	}
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d == nil {
			return nil
		}
		if d.IsDir() {
			switch d.Name() {
			case "node_modules", "vendor", ".git", "dist", "build", "coverage", ".vite":
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Base(path) == name && !seen[path] {
			seen[path] = true
			out = append(out, path)
		}
		return nil
	})
	sort.Strings(out)
	return out
}

func findingPathOrDefault(root, findingPath, fallback string) string {
	if strings.TrimSpace(findingPath) == "" {
		return filepath.Join(root, fallback)
	}
	return findingPath
}

func ensureMap(parent map[string]any, key string) map[string]any {
	child, _ := parent[key].(map[string]any)
	if child == nil {
		child = map[string]any{}
		parent[key] = child
	}
	return child
}

func hasNodeSurface(root string) bool {
	return len(nodePackageDirs(root)) > 0
}

func hasGoSurface(root string) bool {
	return len(goSurfaceDirs(root)) > 0
}

func goSurfaceDirs(root string) []string {
	seen := map[string]bool{}
	var dirs []string
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
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
		if filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		dir := filepath.Dir(path)
		if !seen[dir] {
			seen[dir] = true
			dirs = append(dirs, dir)
		}
		return nil
	})
	sort.Strings(dirs)
	return dirs
}

func existingGoLintConfigs(root string) []string {
	seen := map[string]bool{}
	var out []string
	for _, dir := range goSurfaceDirs(root) {
		for _, name := range []string{".golangci.yml", ".golangci.yaml"} {
			path := filepath.Join(dir, name)
			if _, err := os.Stat(path); err == nil && !seen[path] {
				seen[path] = true
				out = append(out, path)
			}
		}
	}
	sort.Strings(out)
	return out
}

func eslintConfigPaths(root string) []string {
	seen := map[string]bool{}
	var out []string
	for _, name := range []string{"eslint.config.js", "eslint.config.mjs", "eslint.config.cjs", ".eslintrc.json", ".eslintrc.js", ".eslintrc.cjs"} {
		for _, path := range candidateFiles(root, name, nil) {
			if !seen[path] {
				seen[path] = true
				out = append(out, path)
			}
		}
	}
	for _, dir := range nodePackageDirs(root) {
		path := filepath.Join(dir, "eslint.config.js")
		if !seen[path] {
			seen[path] = true
			out = append(out, path)
		}
	}
	sort.Strings(out)
	return out
}

func nodePackageDirs(root string) []string {
	seen := map[string]bool{}
	var dirs []string
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d == nil {
			return nil
		}
		if d.IsDir() {
			switch d.Name() {
			case "node_modules", "vendor", ".git", "dist", "build", "coverage", ".vite":
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Base(path) == "package.json" {
			dir := filepath.Dir(path)
			if !seen[dir] {
				seen[dir] = true
				dirs = append(dirs, dir)
			}
		}
		return nil
	})
	sort.Strings(dirs)
	return dirs
}

func sanitizeModuleName(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		return "scenario"
	}
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_' || r == '.' || r == '/':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	return strings.Trim(b.String(), "-._/")
}

func parseMakefileTargets(content string) map[string]string {
	return rules.ParseMakefileTargets(content)
}

func targetRuns(targets map[string]string, target string, required ...string) bool {
	return rules.TargetRuns(targets, target, required...)
}

func targetRunsAny(targets map[string]string, target string, required ...string) bool {
	return rules.TargetRunsAny(targets, target, required...)
}
