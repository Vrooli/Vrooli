package inference

import (
	"encoding/json"
	"regexp"
	"strings"
)

var (
	// Reasoning models emit their chain of thought before the answer. The
	// local catalog leads with qwen3.5, whose renderer declares a thinking
	// capability, so this is the common case rather than an edge case.
	thinkBlockPattern = regexp.MustCompile(`(?is)<think>.*?</think>`)
	codeFencePattern  = regexp.MustCompile("(?s)```[a-zA-Z0-9_-]*[ \\t]*\\r?\\n?(.*?)```")
)

// ExtractJSONValue recovers the single JSON value a provider was asked to
// return from output that may also carry reasoning blocks, Markdown fences, or
// surrounding prose.
//
// Being permissive here does not weaken the contract. The extracted text is
// still validated against the request schema, and ValidateJSON remains the only
// authority on whether a response succeeds. Refusing to strip a fence would
// only convert a recoverable formatting habit into a hard failure.
func ExtractJSONValue(raw string) string {
	candidate := strings.TrimSpace(thinkBlockPattern.ReplaceAllString(raw, ""))
	if isJSONValue(candidate) {
		return candidate
	}
	if match := codeFencePattern.FindStringSubmatch(candidate); len(match) == 2 {
		fenced := strings.TrimSpace(match[1])
		if isJSONValue(fenced) {
			return fenced
		}
		candidate = fenced
	}
	if balanced := firstBalancedJSON(candidate); balanced != "" {
		return balanced
	}
	return candidate
}

func isJSONValue(text string) bool {
	trimmed := strings.TrimSpace(text)
	return trimmed != "" && json.Valid([]byte(trimmed))
}

// firstBalancedJSON returns the first structurally complete JSON object or
// array in text, or "" when none is present.
func firstBalancedJSON(text string) string {
	for index, char := range text {
		if char != '{' && char != '[' {
			continue
		}
		if value := balancedFrom(text[index:]); value != "" {
			return value
		}
	}
	return ""
}

func balancedFrom(text string) string {
	openers := make([]rune, 0, 8)
	inString := false
	escaped := false
	for index, char := range text {
		if escaped {
			escaped = false
			continue
		}
		switch {
		case inString && char == '\\':
			escaped = true
		case char == '"':
			inString = !inString
		case inString:
			// Structural characters inside a string literal carry no meaning.
		case char == '{' || char == '[':
			openers = append(openers, char)
		case char == '}' || char == ']':
			if len(openers) == 0 {
				return ""
			}
			opener := openers[len(openers)-1]
			if (char == '}' && opener != '{') || (char == ']' && opener != '[') {
				return ""
			}
			openers = openers[:len(openers)-1]
			if len(openers) == 0 {
				value := text[:index+1]
				if isJSONValue(value) {
					return value
				}
				return ""
			}
		}
	}
	return ""
}
