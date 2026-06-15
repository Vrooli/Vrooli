package autofix

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"quality-health/internal/contracts"
)

const TSConfigProtectiveCommentBlock = `    // SAFETY-CRITICAL RULES - DO NOT REMOVE OR WEAKEN
    // These rules prevent runtime crashes like "X is not a function".
    // DO: Use optional chaining (?.), null checks, nullish coalescing (??), and type guards.
    // DON'T: Use type assertions (as X), non-null assertions (!), @ts-ignore, or weaken these rules.
    // These rules exist because UI crashes are the #1 production issue.
`

type Candidate struct {
	RuleID      string
	FilePath    string
	Description string
	Before      string
	After       string
	Applied     bool
}

func Preview(root string, ruleIDs []string) ([]Candidate, error) {
	if !wantsRule(ruleIDs, contracts.RuleTSConfigStrict) {
		return nil, nil
	}
	path := filepath.Join(root, "ui", "tsconfig.json")
	before, after, changed, err := fixedTSConfig(path)
	if err != nil || !changed {
		return nil, err
	}
	return []Candidate{{
		RuleID:      contracts.RuleTSConfigStrict,
		FilePath:    path,
		Description: "Enable strict TypeScript settings and restore the safety-critical guardrail comment block.",
		Before:      before,
		After:       after,
	}}, nil
}

func Apply(root string, ruleIDs []string) ([]Candidate, error) {
	candidates, err := Preview(root, ruleIDs)
	if err != nil {
		return nil, err
	}
	for i := range candidates {
		if err := os.WriteFile(candidates[i].FilePath, []byte(candidates[i].After), 0o644); err != nil {
			return candidates, err
		}
		candidates[i].Applied = true
	}
	return candidates, nil
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
		out, err := json.MarshalIndent(parsed, "", "  ")
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

func wantsRule(ruleIDs []string, ruleID string) bool {
	if len(ruleIDs) == 0 {
		return true
	}
	for _, id := range ruleIDs {
		if strings.EqualFold(strings.TrimSpace(id), ruleID) {
			return true
		}
	}
	return false
}
