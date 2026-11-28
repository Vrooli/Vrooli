package skills

import (
	"regexp"
	"sort"
	"strings"
)

// Variable represents a detected template variable in skill content.
type Variable struct {
	Name        string `json:"name"`        // e.g., "TARGET"
	Placeholder string `json:"placeholder"` // e.g., "{{TARGET}}"
	Occurrences int    `json:"occurrences"` // Number of times it appears in content
}

// variableRegex matches {{VARIABLE_NAME}} patterns.
// Requires uppercase start, allows uppercase letters, numbers, and underscores.
var variableRegex = regexp.MustCompile(`\{\{([A-Z][A-Z0-9_]*)\}\}`)

// ExtractVariables parses content and returns all unique {{VAR}} variables.
// Variables are returned sorted alphabetically by name.
// Returns nil if no variables are found.
func ExtractVariables(content string) []Variable {
	matches := variableRegex.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return nil
	}

	// Count occurrences of each variable
	counts := make(map[string]int)
	for _, match := range matches {
		if len(match) >= 2 {
			counts[match[1]]++
		}
	}

	// Build result slice
	vars := make([]Variable, 0, len(counts))
	for name, count := range counts {
		vars = append(vars, Variable{
			Name:        name,
			Placeholder: "{{" + name + "}}",
			Occurrences: count,
		})
	}

	// Sort alphabetically by name for consistent output
	sort.Slice(vars, func(i, j int) bool {
		return vars[i].Name < vars[j].Name
	})

	return vars
}

// SubstituteVariables replaces {{VAR}} placeholders with provided values.
// Variables not in the values map are left unchanged in the content.
// Returns the content with substitutions applied.
func SubstituteVariables(content string, values map[string]string) string {
	if len(values) == 0 {
		return content
	}

	result := content
	for name, value := range values {
		placeholder := "{{" + name + "}}"
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result
}
