package security

import (
	"fmt"
	"strings"
)

// SanitizeAllowedTools filters and validates the allowed tools list.
func SanitizeAllowedTools(tools []string) ([]string, error) {
	validator := NewBashCommandValidator()

	if err := validator.Validate(tools); err != nil {
		return nil, err
	}

	sanitized := make([]string, 0, len(tools))
	for _, tool := range tools {
		tool = strings.TrimSpace(tool)
		if tool == "" {
			continue
		}
		sanitized = append(sanitized, tool)
	}

	for _, tool := range sanitized {
		if tool == "*" {
			return nil, fmt.Errorf("wildcard tool permission (*) is not allowed for spawned agents")
		}
	}

	return sanitized, nil
}

// ValidateAllowedTools validates a list of allowed tools without sanitizing.
func ValidateAllowedTools(tools []string) error {
	_, err := SanitizeAllowedTools(tools)
	return err
}

// DefaultSafeTools returns a safe default set of allowed tools for agents.
func DefaultSafeTools() []string {
	return []string{
		"read",
		"edit",
		"write",
		"glob",
		"grep",
	}
}

// SafeBashPatterns returns commonly allowed bash patterns for test operations.
func SafeBashPatterns() []string {
	return []string{
		"bash(pnpm test|pnpm run test|npm test|npm run test)",
		"bash(go test|go test ./...)",
		"bash(vitest|vitest run)",
		"bash(jest|jest --)",
		"bash(bats|bats *)",
		"bash(make test)",
	}
}
