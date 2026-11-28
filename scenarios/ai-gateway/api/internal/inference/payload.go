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
// still validated against the request schema, and schemagate.ValidateJSON remains the only
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
	scanner := balancedJSONScanner{openers: make([]rune, 0, 8)}
	for index, char := range text {
		complete, invalid := scanner.consume(char)
		if invalid {
			return ""
		}
		if complete {
			value := text[:index+1]
			if isJSONValue(value) {
				return value
			}
			return ""
		}
	}
	return ""
}

type balancedJSONScanner struct {
	openers  []rune
	inString bool
	escaped  bool
}

func (scanner *balancedJSONScanner) consume(char rune) (complete, invalid bool) {
	if scanner.escaped {
		scanner.escaped = false
		return false, false
	}
	if scanner.inString {
		scanner.consumeStringCharacter(char)
		return false, false
	}
	return scanner.consumeStructuralCharacter(char)
}

func (scanner *balancedJSONScanner) consumeStringCharacter(char rune) {
	switch char {
	case '\\':
		scanner.escaped = true
	case '"':
		scanner.inString = false
	}
}

func (scanner *balancedJSONScanner) consumeStructuralCharacter(char rune) (complete, invalid bool) {
	switch char {
	case '"':
		scanner.inString = true
	case '{', '[':
		scanner.openers = append(scanner.openers, char)
	case '}', ']':
		return scanner.consumeClosingCharacter(char)
	}
	return false, false
}

func (scanner *balancedJSONScanner) consumeClosingCharacter(char rune) (complete, invalid bool) {
	if len(scanner.openers) == 0 {
		return false, true
	}
	opener := scanner.openers[len(scanner.openers)-1]
	if (char == '}' && opener != '{') || (char == ']' && opener != '[') {
		return false, true
	}
	scanner.openers = scanner.openers[:len(scanner.openers)-1]
	return len(scanner.openers) == 0, false
}
