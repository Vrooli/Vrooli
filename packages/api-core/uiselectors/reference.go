package uiselectors

import (
	"strconv"
	"strings"
)

func ResolveReference(selectorRef string, manifest map[string]interface{}) string {
	// Check if this is a @selector/ reference
	if !strings.HasPrefix(selectorRef, "@selector/") {
		return "" // Not a reference, leave as-is
	}

	// Extract the path (e.g., "dashboard.newProjectButton" from "@selector/dashboard.newProjectButton")
	path := strings.TrimPrefix(selectorRef, "@selector/")

	// Strip /*dup-N*/ suffix if present (used to make selectors unique in workflows)
	// Example: "dialogs.project.root /*dup-1*/" -> "dialogs.project.root"
	if idx := strings.Index(path, " /*dup-"); idx != -1 {
		path = path[:idx]
	}

	// Split a dynamic selector invocation from an optional CSS suffix. Dynamic
	// registry entries are deliberately parameterized (for example
	// projects.cardById(id="${@params/projectId}")), so treating the whole call
	// as a manifest key makes valid reusable BAS subflows uncompilable.
	basePath := ""
	for _, namespace := range []string{"selectors", "dynamicSelectors"} {
		entries, _ := manifest[namespace].(map[string]interface{})
		for key := range entries {
			if path == key || strings.HasPrefix(path, key+"(") || strings.HasPrefix(path, key+":") || strings.HasPrefix(path, key+"[") {
				if len(key) > len(basePath) {
					basePath = key
				}
			}
		}
	}
	if basePath == "" {
		return ""
	}
	suffix := strings.TrimPrefix(path, basePath)
	var arguments map[string]string
	if strings.HasPrefix(suffix, "(") {
		closeIdx := invocationEnd(suffix)
		if closeIdx < 1 {
			return ""
		}
		parsed, ok := parseSelectorArguments(suffix[1:closeIdx])
		if !ok {
			return ""
		}
		arguments = parsed
		suffix = suffix[closeIdx+1:]
	}
	if manifest == nil {
		return ""
	}
	dynamic, _ := manifest["dynamicSelectors"].(map[string]interface{})
	if _, ok := dynamic[basePath]; ok {
		for _, value := range arguments {
			if strings.Contains(value, "${") {
				return deferReference(manifest, basePath, arguments, suffix)
			}
		}
	}
	value, err := resolveArgs(manifest, basePath, arguments)
	if err != nil {
		return ""
	}
	return value + suffix
}

// parseSelectorArguments accepts the intentionally small named-argument
// grammar used by selector registry references: name=value pairs separated by
// commas, with values optionally quoted. Quoted values may contain commas and
// preserve workflow placeholders such as ${@params/projectId} verbatim for the
// normal execution-parameter interpolation phase.
func parseSelectorArguments(input string) (map[string]string, bool) {
	args := make(map[string]string)
	for len(strings.TrimSpace(input)) > 0 {
		input = strings.TrimSpace(input)
		eq := strings.IndexByte(input, '=')
		if eq <= 0 {
			return nil, false
		}
		name := strings.TrimSpace(input[:eq])
		if name == "" {
			return nil, false
		}
		input = strings.TrimSpace(input[eq+1:])
		value := ""
		if len(input) > 0 && (input[0] == '\'' || input[0] == '"') {
			quote := input[0]
			end := 1
			for end < len(input) && input[end] != quote {
				if input[end] == '\\' {
					end++
				}
				end++
			}
			if end >= len(input) {
				return nil, false
			}
			quoted := input[:end+1]
			if quote == '"' {
				unquoted, err := strconv.Unquote(quoted)
				if err != nil {
					return nil, false
				}
				value = unquoted
			} else {
				value = quoted[1 : len(quoted)-1]
			}
			input = strings.TrimSpace(input[end+1:])
		} else {
			end := strings.IndexByte(input, ',')
			if end == -1 {
				value, input = strings.TrimSpace(input), ""
			} else {
				value, input = strings.TrimSpace(input[:end]), input[end:]
			}
		}
		if value == "" || args[name] != "" {
			return nil, false
		}
		args[name] = value
		if input == "" {
			break
		}
		if input[0] != ',' {
			return nil, false
		}
		input = input[1:]
	}
	return args, true
}

// invocationEnd excludes pseudo-class parentheses after the argument list.
func invocationEnd(text string) int {
	var quote byte
	depth := 0
	for i := 0; i < len(text); i++ {
		ch := text[i]
		if quote != 0 {
			if ch == '\\' {
				i++
				continue
			}
			if ch == quote {
				quote = 0
			}
			continue
		}
		if ch == '\'' || ch == '"' {
			quote = ch
			continue
		}
		if ch == '(' {
			depth++
		}
		if ch == ')' {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}
