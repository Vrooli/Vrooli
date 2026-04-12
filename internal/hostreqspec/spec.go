package hostreqspec

import (
	"fmt"
	"runtime"
	"strings"
)

type Kind string

const (
	KindTool      Kind = "tool"
	KindSafeguard Kind = "safeguard"
)

type Declaration struct {
	Name         string   `json:"name"`
	Required     bool     `json:"required"`
	Reason       string   `json:"reason"`
	When         []string `json:"when,omitempty"`
	Environments []string `json:"environments,omitempty"`
	Platforms    []string `json:"platforms,omitempty"`
	Notes        string   `json:"notes,omitempty"`
	Manual       bool     `json:"manual,omitempty"`
}

func ValidateDeclarations(kind Kind, declarations []Declaration) error {
	seen := make(map[string]struct{}, len(declarations))
	for index, declaration := range declarations {
		name := strings.TrimSpace(declaration.Name)
		if name == "" {
			return fmt.Errorf("%s declarations[%d].name is required", kind, index)
		}
		if strings.TrimSpace(declaration.Reason) == "" {
			return fmt.Errorf("%s %q reason is required", kind, name)
		}
		if _, exists := seen[name]; exists {
			return fmt.Errorf("duplicate %s declaration %q", kind, name)
		}
		seen[name] = struct{}{}
		if err := validateList(kind, name, "when", declaration.When); err != nil {
			return err
		}
		if err := validateList(kind, name, "environments", declaration.Environments); err != nil {
			return err
		}
		if err := validateList(kind, name, "platforms", declaration.Platforms); err != nil {
			return err
		}
	}
	return nil
}

func validateList(kind Kind, name, field string, values []string) error {
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s %q contains an empty %s entry", kind, name, field)
		}
	}
	return nil
}

func CurrentPlatform() string {
	switch runtime.GOOS {
	case "darwin":
		return "macos"
	default:
		return runtime.GOOS
	}
}

func NormalizeEnvironment(environment string) string {
	switch strings.ToLower(strings.TrimSpace(environment)) {
	case "production":
		return "production"
	case "minimal":
		return "minimal"
	default:
		return "development"
	}
}
