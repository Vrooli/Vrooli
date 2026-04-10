package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// TypeSafetyAnalyzer validates TypeScript/ESLint config for a scenario's UI directory
type TypeSafetyAnalyzer struct {
	scenarioPath string
}

// NewTypeSafetyAnalyzer creates an analyzer for the given scenario path
func NewTypeSafetyAnalyzer(scenarioPath string) *TypeSafetyAnalyzer {
	return &TypeSafetyAnalyzer{scenarioPath: scenarioPath}
}

// TypeSafetyConfigResult contains the results of a type-safety config scan
type TypeSafetyConfigResult struct {
	Scenario                     string                    `json:"scenario"`
	TSConfigPath                 string                    `json:"tsconfig_path,omitempty"`
	TSConfigFound                bool                      `json:"tsconfig_found"`
	TSConfigStrict               bool                      `json:"tsconfig_strict"`
	TSConfigNoUnchecked          bool                      `json:"tsconfig_no_unchecked_indexed_access"`
	TSConfigHasProtectiveComment bool                      `json:"tsconfig_has_protective_comment"`
	ESLintConfigPath             string                    `json:"eslint_config_path,omitempty"`
	ESLintConfigFound            bool                      `json:"eslint_config_found"`
	ESLintHasHeaderComment       bool                      `json:"eslint_has_header_comment"`
	ESLintHasPerRuleComments     bool                      `json:"eslint_has_per_rule_comments"`
	ESLintSafetyRules            []ESLintRuleStatus        `json:"eslint_safety_rules"`
	PatternSummary               *TypeSafetyPatternSummary `json:"pattern_summary,omitempty"`
	Violations                   []TypeSafetyViolation     `json:"violations"`
}

// ESLintRuleStatus tracks whether a required ESLint rule is configured
type ESLintRuleStatus struct {
	Rule     string `json:"rule"`
	MinLevel string `json:"min_level"`
	Found    bool   `json:"found"`
	Level    string `json:"level,omitempty"`
}

// TypeSafetyPatternSummary aggregates dangerous pattern counts
type TypeSafetyPatternSummary struct {
	TotalFiles            int                    `json:"total_files"`
	AsAnyCount            int                    `json:"as_any_count"`
	AsTypeAssertionCount  int                    `json:"as_type_assertion_count"`
	TsIgnoreCount         int                    `json:"ts_ignore_count"`
	NonNullAssertionCount int                    `json:"non_null_assertion_count"`
	TopFiles              []FilePatternBreakdown `json:"top_files,omitempty"`
}

// FilePatternBreakdown shows per-file dangerous pattern counts
type FilePatternBreakdown struct {
	FilePath              string `json:"file_path"`
	AsAnyCount            int    `json:"as_any_count,omitempty"`
	AsTypeAssertionCount  int    `json:"as_type_assertion_count,omitempty"`
	TsIgnoreCount         int    `json:"ts_ignore_count,omitempty"`
	NonNullAssertionCount int    `json:"non_null_assertion_count,omitempty"`
	Total                 int    `json:"total"`
}

// TypeSafetyViolation describes a single type-safety config violation
type TypeSafetyViolation struct {
	RuleID      string `json:"rule_id"`
	Severity    string `json:"severity"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Remediation string `json:"remediation,omitempty"`
	FilePath    string `json:"file_path,omitempty"`
}

// requiredESLintRules defines the rules that must be present in ESLint config
var requiredESLintRules = []ESLintRuleStatus{
	{Rule: "react-hooks/rules-of-hooks", MinLevel: "error"},
	{Rule: "@typescript-eslint/no-non-null-assertion", MinLevel: "error"},
	{Rule: "@typescript-eslint/no-explicit-any", MinLevel: "error"},
	{Rule: "@typescript-eslint/no-unsafe-member-access", MinLevel: "warn"},
	{Rule: "@typescript-eslint/no-unsafe-call", MinLevel: "warn"},
	{Rule: "@typescript-eslint/no-unsafe-argument", MinLevel: "warn"},
	{Rule: "@typescript-eslint/no-unsafe-assignment", MinLevel: "warn"},
	{Rule: "@typescript-eslint/no-unsafe-return", MinLevel: "warn"},
	{Rule: "import/no-cycle", MinLevel: "error"},
}

// tsconfig protective comment key phrases
var tsconfigProtectivePhrases = []string{
	"SAFETY-CRITICAL RULES",
	"DO NOT REMOVE OR WEAKEN",
	"DON'T: Use type assertions (as X)",
	"UI crashes are the #1 production issue",
}

// eslint protective comment phrases
var (
	eslintHeaderPhrase         = "SAFETY-CRITICAL RULES - DO NOT REMOVE, DISABLE, OR WEAKEN"
	eslintPerRuleCommentPrefix = "// CRITICAL:"
)

// tsconfigProtectiveCommentBlock is the exact comment block that must appear in tsconfig.json.
// Used by both the analyzer (to recommend) and the fixer (to inject).
const tsconfigProtectiveCommentBlock = `    // ╔══════════════════════════════════════════════════════════════════════════╗
    // ║  SAFETY-CRITICAL RULES - DO NOT REMOVE OR WEAKEN                         ║
    // ║                                                                          ║
    // ║  These rules prevent runtime crashes like:                               ║
    // ║  - "X is not a function"                                                 ║
    // ║  - "Cannot read property Y of undefined"                                 ║
    // ║  - "undefined is not iterable"                                           ║
    // ║                                                                          ║
    // ║  If you encounter type errors from these rules:                          ║
    // ║  ✅ DO: Use optional chaining (?.) or null checks (if (x) { ... })       ║
    // ║  ✅ DO: Use nullish coalescing (??) for defaults                         ║
    // ║  ✅ DO: Add proper type guards before accessing properties               ║
    // ║  ❌ DON'T: Use non-null assertion (!) - it hides bugs, use ?? instead    ║
    // ║  ❌ DON'T: Use type assertions (as X) to silence errors                  ║
    // ║  ❌ DON'T: Add @ts-ignore or @ts-expect-error comments                   ║
    // ║  ❌ DON'T: Remove or weaken these rules                                  ║
    // ║                                                                          ║
    // ║  These rules exist because UI crashes are the #1 production issue.       ║
    // ║  Removing them WILL cause crashes that are much harder to debug than     ║
    // ║  the type errors they produce at compile time.                           ║
    // ╚══════════════════════════════════════════════════════════════════════════╝
`

// eslintSafetyRulesSnippet is the exact rules block (with protective comments) that should
// appear in the ESLint config. Used as remediation guidance for missing-rules violations.
const eslintSafetyRulesSnippet = `    rules: {
      // ════════════════════════════════════════════════════════════════════════
      // SAFETY-CRITICAL RULES - DO NOT REMOVE, DISABLE, OR WEAKEN
      //
      // These rules prevent runtime crashes. If you encounter errors:
      // ✅ DO: Fix the code with optional chaining (?.), null checks, or proper types
      // ❌ DON'T: Disable the rule, use "as" casts, or use non-null assertion (!)
      //
      // Removing these rules WILL cause production crashes that are much harder
      // to debug than the lint errors they produce at development time.
      // ════════════════════════════════════════════════════════════════════════

      // CRITICAL: Catches React Error #310 (hook count changes between renders)
      // Detects early returns before hooks, conditional hook calls, etc.
      "react-hooks/rules-of-hooks": "error",

      // CRITICAL: Prevents non-null assertion (!) which bypasses TypeScript's null checks
      // Using ! hides bugs that will crash at runtime with "X is not a function"
      // Instead of arr[0]!, use: arr[0] ?? defaultValue or if (arr[0]) { ... }
      "@typescript-eslint/no-non-null-assertion": "error",

      // CRITICAL: Catches operations on 'any' typed values that will crash at runtime
      // These catch bugs like "v.trim is not a function" when v is not actually a string
      "@typescript-eslint/no-unsafe-member-access": "warn",
      "@typescript-eslint/no-unsafe-call": "warn",
      "@typescript-eslint/no-unsafe-argument": "warn",
      "@typescript-eslint/no-unsafe-assignment": "warn",
      "@typescript-eslint/no-unsafe-return": "warn",

      // Prevents explicit 'any' which disables all type checking for that value
      "@typescript-eslint/no-explicit-any": "error",

      // CRITICAL: Detects circular dependencies that cause "Cannot access X before initialization"
      // These runtime errors are extremely hard to debug in production (minified variable names).
      // Requires eslint-plugin-import and eslint-import-resolver-typescript
      "import/no-cycle": "error",
    }`

// Analyze runs all type-safety config checks
func (a *TypeSafetyAnalyzer) Analyze() *TypeSafetyConfigResult {
	result := &TypeSafetyConfigResult{
		Scenario: filepath.Base(a.scenarioPath),
	}

	a.checkTSConfig(result)
	a.checkESLintConfig(result)

	return result
}

// checkTSConfig validates tsconfig.json settings and protective comments
func (a *TypeSafetyAnalyzer) checkTSConfig(result *TypeSafetyConfigResult) {
	// Search for tsconfig.json in ui/ directory
	tsconfigPath := filepath.Join(a.scenarioPath, "ui", "tsconfig.json")
	raw, err := os.ReadFile(tsconfigPath)
	if err != nil {
		// No tsconfig found - check if ui/ directory exists at all
		if _, statErr := os.Stat(filepath.Join(a.scenarioPath, "ui")); os.IsNotExist(statErr) {
			// No UI directory - not applicable, no violation
			return
		}
		result.TSConfigFound = false
		result.Violations = append(result.Violations, TypeSafetyViolation{
			RuleID:      "TS_CONFIG_STRICT",
			Severity:    "high",
			Title:       "tsconfig.json not found",
			Description: "No tsconfig.json found in ui/ directory. TypeScript strict mode cannot be verified.",
			Remediation: "Create ui/tsconfig.json with the following compilerOptions and protective comment block:\n\n" +
				"```jsonc\n{\n  \"compilerOptions\": {\n" + tsconfigProtectiveCommentBlock +
				"    \"strict\": true,\n    \"noUncheckedIndexedAccess\": true\n  }\n}\n```",
			FilePath: tsconfigPath,
		})
		return
	}

	result.TSConfigFound = true
	result.TSConfigPath = tsconfigPath
	content := string(raw)

	// Strip JSONC comments for JSON parsing
	stripped := stripJSONCComments(content)

	var tsconfig map[string]interface{}
	if err := json.Unmarshal([]byte(stripped), &tsconfig); err != nil {
		result.Violations = append(result.Violations, TypeSafetyViolation{
			RuleID:      "TS_CONFIG_STRICT",
			Severity:    "high",
			Title:       "tsconfig.json parse error",
			Description: fmt.Sprintf("Failed to parse tsconfig.json: %v", err),
			Remediation: "Fix the JSON syntax error in tsconfig.json, then ensure it contains:\n\n" +
				"```jsonc\n\"compilerOptions\": {\n" + tsconfigProtectiveCommentBlock +
				"  \"strict\": true,\n  \"noUncheckedIndexedAccess\": true\n}\n```",
			FilePath: tsconfigPath,
		})
		return
	}

	// Check compilerOptions
	compilerOpts, _ := tsconfig["compilerOptions"].(map[string]interface{})
	if compilerOpts == nil {
		result.Violations = append(result.Violations, TypeSafetyViolation{
			RuleID:      "TS_CONFIG_STRICT",
			Severity:    "high",
			Title:       "Missing compilerOptions",
			Description: "tsconfig.json has no compilerOptions section.",
			Remediation: "Add the following compilerOptions block to tsconfig.json:\n\n" +
				"```jsonc\n\"compilerOptions\": {\n" + tsconfigProtectiveCommentBlock +
				"  \"strict\": true,\n  \"noUncheckedIndexedAccess\": true\n}\n```",
			FilePath: tsconfigPath,
		})
		return
	}

	// Check strict: true
	if strict, ok := compilerOpts["strict"].(bool); ok && strict {
		result.TSConfigStrict = true
	} else {
		result.Violations = append(result.Violations, TypeSafetyViolation{
			RuleID:      "TS_CONFIG_STRICT",
			Severity:    "high",
			Title:       "strict mode not enabled",
			Description: "tsconfig.json does not have \"strict\": true in compilerOptions. This allows null/undefined bugs to slip through at compile time.",
			Remediation: "Add \"strict\": true to compilerOptions in tsconfig.json. This can be auto-fixed:\n  scenario-auditor fix <scenario> --rules TS_CONFIG_STRICT",
			FilePath:    tsconfigPath,
		})
	}

	// Check noUncheckedIndexedAccess: true
	if noUnchecked, ok := compilerOpts["noUncheckedIndexedAccess"].(bool); ok && noUnchecked {
		result.TSConfigNoUnchecked = true
	} else {
		result.Violations = append(result.Violations, TypeSafetyViolation{
			RuleID:      "TS_CONFIG_STRICT",
			Severity:    "high",
			Title:       "noUncheckedIndexedAccess not enabled",
			Description: "tsconfig.json does not have \"noUncheckedIndexedAccess\": true. Without it, arr[0].trim() compiles but crashes at runtime if the array is empty.",
			Remediation: "Add \"noUncheckedIndexedAccess\": true to compilerOptions in tsconfig.json. This can be auto-fixed:\n  scenario-auditor fix <scenario> --rules TS_CONFIG_STRICT",
			FilePath:    tsconfigPath,
		})
	}

	// Check protective comment block (using raw content, not stripped)
	hasAllPhrases := true
	for _, phrase := range tsconfigProtectivePhrases {
		if !strings.Contains(content, phrase) {
			hasAllPhrases = false
			break
		}
	}
	result.TSConfigHasProtectiveComment = hasAllPhrases
	if !hasAllPhrases {
		result.Violations = append(result.Violations, TypeSafetyViolation{
			RuleID:      "TS_CONFIG_STRICT",
			Severity:    "high",
			Title:       "Missing protective comment block",
			Description: "tsconfig.json has settings but is missing the protective comment block. Without these comments, a future agent may disable strict mode to fix a type error — causing production crashes. The comment block explains the consequences and suggests safe alternatives (optional chaining, nullish coalescing, type guards).",
			Remediation: "Add this comment block inside compilerOptions, before the \"strict\" line in tsconfig.json:\n\n```jsonc\n" +
				tsconfigProtectiveCommentBlock + "```\n\nThis can be auto-fixed:\n  scenario-auditor fix <scenario> --rules TS_CONFIG_STRICT",
			FilePath: tsconfigPath,
		})
	}
}

// checkESLintConfig validates ESLint configuration
func (a *TypeSafetyAnalyzer) checkESLintConfig(result *TypeSafetyConfigResult) {
	uiDir := filepath.Join(a.scenarioPath, "ui")
	if _, err := os.Stat(uiDir); os.IsNotExist(err) {
		return // No UI directory - not applicable
	}

	// Search for ESLint config files
	configPatterns := []string{
		"eslint.config.js", "eslint.config.mjs", "eslint.config.cjs",
		".eslintrc.json", ".eslintrc.js", ".eslintrc.cjs",
	}

	var configPath string
	var content string
	for _, pattern := range configPatterns {
		path := filepath.Join(uiDir, pattern)
		raw, err := os.ReadFile(path)
		if err == nil {
			configPath = path
			content = string(raw)
			break
		}
	}

	if configPath == "" {
		result.ESLintConfigFound = false
		result.Violations = append(result.Violations, TypeSafetyViolation{
			RuleID:      "ESLINT_SAFETY_RULES",
			Severity:    "high",
			Title:       "ESLint config not found",
			Description: "No ESLint configuration found in ui/ directory. Safety-critical lint rules cannot be verified.",
			Remediation: "Create ui/eslint.config.js with the following safety-critical rules section:\n\n```js\n" +
				eslintSafetyRulesSnippet + "\n```\n\n" +
				"Required dev dependencies:\n  pnpm add -D eslint @eslint/js typescript-eslint eslint-plugin-react-hooks eslint-plugin-react-refresh eslint-plugin-import eslint-import-resolver-typescript",
		})
		return
	}

	result.ESLintConfigFound = true
	result.ESLintConfigPath = configPath

	// Check for header comment block
	result.ESLintHasHeaderComment = strings.Contains(content, eslintHeaderPhrase)
	if !result.ESLintHasHeaderComment {
		result.Violations = append(result.Violations, TypeSafetyViolation{
			RuleID:      "ESLINT_SAFETY_RULES",
			Severity:    "high",
			Title:       "Missing safety-critical header comment",
			Description: fmt.Sprintf("ESLint config is missing the safety-critical header comment block (\"%s\"). Without this comment, a future agent may disable safety rules without understanding the consequences.", eslintHeaderPhrase),
			Remediation: "Add this header comment block at the top of the rules section in your ESLint config:\n\n```js\n" +
				"      // ════════════════════════════════════════════════════════════════════════\n" +
				"      // SAFETY-CRITICAL RULES - DO NOT REMOVE, DISABLE, OR WEAKEN\n" +
				"      //\n" +
				"      // These rules prevent runtime crashes. If you encounter errors:\n" +
				"      // ✅ DO: Fix the code with optional chaining (?.), null checks, or proper types\n" +
				"      // ❌ DON'T: Disable the rule, use \"as\" casts, or use non-null assertion (!)\n" +
				"      //\n" +
				"      // Removing these rules WILL cause production crashes that are much harder\n" +
				"      // to debug than the lint errors they produce at development time.\n" +
				"      // ════════════════════════════════════════════════════════════════════════\n```",
			FilePath: configPath,
		})
	}

	// Check for per-rule CRITICAL comments
	criticalRules := []string{"rules-of-hooks", "no-non-null-assertion", "no-unsafe-", "no-cycle"}
	criticalCommentCount := 0
	for _, rule := range criticalRules {
		// Look for a "// CRITICAL:" comment near the rule name
		ruleIdx := strings.Index(content, rule)
		if ruleIdx < 0 {
			continue
		}
		// Look backwards from the rule for a "// CRITICAL:" comment (within 200 chars)
		start := ruleIdx - 200
		if start < 0 {
			start = 0
		}
		preceding := content[start:ruleIdx]
		if strings.Contains(preceding, eslintPerRuleCommentPrefix) {
			criticalCommentCount++
		}
	}
	result.ESLintHasPerRuleComments = criticalCommentCount >= len(criticalRules)
	if !result.ESLintHasPerRuleComments {
		result.Violations = append(result.Violations, TypeSafetyViolation{
			RuleID:      "ESLINT_SAFETY_RULES",
			Severity:    "medium",
			Title:       "Missing per-rule CRITICAL comments",
			Description: fmt.Sprintf("ESLint config is missing per-rule '// CRITICAL:' comments for safety rules. Found %d of %d expected. These comments explain what crash each rule prevents.", criticalCommentCount, len(criticalRules)),
			Remediation: "Add a '// CRITICAL:' comment above each safety rule explaining what crash it prevents. Example:\n\n```js\n" +
				"      // CRITICAL: Catches React Error #310 (hook count changes between renders)\n" +
				"      \"react-hooks/rules-of-hooks\": \"error\",\n\n" +
				"      // CRITICAL: Prevents non-null assertion (!) which bypasses TypeScript's null checks\n" +
				"      \"@typescript-eslint/no-non-null-assertion\": \"error\",\n\n" +
				"      // CRITICAL: Catches operations on 'any' typed values that will crash at runtime\n" +
				"      \"@typescript-eslint/no-unsafe-member-access\": \"warn\",\n\n" +
				"      // CRITICAL: Detects circular dependencies that cause \"Cannot access X before initialization\"\n" +
				"      \"import/no-cycle\": \"error\",\n```",
			FilePath: configPath,
		})
	}

	// Check for required rules
	result.ESLintSafetyRules = make([]ESLintRuleStatus, len(requiredESLintRules))
	var missingRules []string
	for i, req := range requiredESLintRules {
		status := ESLintRuleStatus{
			Rule:     req.Rule,
			MinLevel: req.MinLevel,
		}

		// Build regex to find rule and its level
		// Match patterns like: "rule-name": "error" or "rule-name": "warn"
		escaped := regexp.QuoteMeta(req.Rule)
		rulePattern := regexp.MustCompile(`["']` + escaped + `["']\s*:\s*["'](error|warn|off)["']`)
		match := rulePattern.FindStringSubmatch(content)

		if match != nil {
			status.Found = true
			status.Level = match[1]
		} else {
			missingRules = append(missingRules, req.Rule)
		}

		result.ESLintSafetyRules[i] = status
	}

	if len(missingRules) > 0 {
		// Build per-rule remediation showing the exact line to add for each missing rule
		var ruleLines strings.Builder
		for _, rule := range missingRules {
			for _, req := range requiredESLintRules {
				if req.Rule == rule {
					ruleLines.WriteString(fmt.Sprintf("      \"%s\": \"%s\",\n", req.Rule, req.MinLevel))
					break
				}
			}
		}
		result.Violations = append(result.Violations, TypeSafetyViolation{
			RuleID:      "ESLINT_SAFETY_RULES",
			Severity:    "high",
			Title:       "Missing ESLint safety rules",
			Description: fmt.Sprintf("ESLint config is missing %d required safety rules: %s", len(missingRules), strings.Join(missingRules, ", ")),
			Remediation: "Add the following rules to your ESLint config's rules section:\n\n```js\n" + ruleLines.String() + "```\n\n" +
				"For the complete rules section with protective comments, see:\n```js\n" + eslintSafetyRulesSnippet + "\n```",
			FilePath: configPath,
		})
	}
}

// FixTSConfig adds missing strict settings and protective comments to tsconfig.json
func (a *TypeSafetyAnalyzer) FixTSConfig() (*TypeSafetyConfigResult, error) {
	tsconfigPath := filepath.Join(a.scenarioPath, "ui", "tsconfig.json")
	raw, err := os.ReadFile(tsconfigPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read tsconfig.json: %w", err)
	}

	content := string(raw)
	modified := false

	// Strip comments for parsing
	stripped := stripJSONCComments(content)
	var tsconfig map[string]interface{}
	if err := json.Unmarshal([]byte(stripped), &tsconfig); err != nil {
		return nil, fmt.Errorf("cannot parse tsconfig.json: %w", err)
	}

	compilerOpts, _ := tsconfig["compilerOptions"].(map[string]interface{})
	if compilerOpts == nil {
		compilerOpts = make(map[string]interface{})
		tsconfig["compilerOptions"] = compilerOpts
	}

	// Add strict: true if missing
	if strict, ok := compilerOpts["strict"].(bool); !ok || !strict {
		compilerOpts["strict"] = true
		modified = true
	}

	// Add noUncheckedIndexedAccess: true if missing
	if noUnchecked, ok := compilerOpts["noUncheckedIndexedAccess"].(bool); !ok || !noUnchecked {
		compilerOpts["noUncheckedIndexedAccess"] = true
		modified = true
	}

	if modified {
		// Re-serialize the JSON (without comments - we'll add the protective block separately)
		out, err := json.MarshalIndent(tsconfig, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("cannot serialize tsconfig.json: %w", err)
		}
		content = string(out)
	}

	// Add protective comment block if missing
	hasAllPhrases := true
	for _, phrase := range tsconfigProtectivePhrases {
		if !strings.Contains(content, phrase) {
			hasAllPhrases = false
			break
		}
	}

	if !hasAllPhrases {
		content = injectTSConfigProtectiveComment(content)
		modified = true
	}

	if modified {
		if err := os.WriteFile(tsconfigPath, []byte(content), 0o644); err != nil {
			return nil, fmt.Errorf("cannot write tsconfig.json: %w", err)
		}
	}

	// Re-analyze to return current state
	return a.Analyze(), nil
}

// injectTSConfigProtectiveComment adds the protective comment block before "strict"
func injectTSConfigProtectiveComment(content string) string {
	// Insert before "strict" if found
	strictIdx := strings.Index(content, `"strict"`)
	if strictIdx >= 0 {
		// Find the beginning of the line
		lineStart := strings.LastIndex(content[:strictIdx], "\n") + 1
		return content[:lineStart] + tsconfigProtectiveCommentBlock + content[lineStart:]
	}
	// Fallback: insert after "compilerOptions": {
	coIdx := strings.Index(content, `"compilerOptions"`)
	if coIdx >= 0 {
		braceIdx := strings.Index(content[coIdx:], "{")
		if braceIdx >= 0 {
			insertAt := coIdx + braceIdx + 1
			return content[:insertAt] + "\n" + tsconfigProtectiveCommentBlock + content[insertAt:]
		}
	}
	return content
}

// stripJSONCComments removes // and /* */ comments from JSONC content for JSON parsing
func stripJSONCComments(input string) string {
	var result strings.Builder
	i := 0
	inString := false
	for i < len(input) {
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
				// Skip to end of line
				for i < len(input) && input[i] != '\n' {
					i++
				}
				continue
			}
			if input[i+1] == '*' {
				// Skip to */
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
