package ruleengine

// Category captures metadata for display in clients.
type Category struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// DefaultCategories returns the built-in category definitions.
func DefaultCategories() map[string]Category {
	return map[string]Category{
		"api": {
			ID:          "api",
			Name:        "API Standards",
			Description: "Rules for API design, implementation, and resource management",
		},
		"cli": {
			ID:          "cli",
			Name:        "CLI & Structure",
			Description: "Rules for command-line interfaces and code structure",
		},
		"config": {
			ID:          "config",
			Name:        "Configuration",
			Description: "Rules for configuration management and environment variables",
		},
		"test": {
			ID:          "test",
			Name:        "Testing",
			Description: "Rules for test coverage and quality",
		},
		"ui": {
			ID:          "ui",
			Name:        "User Interface",
			Description: "Rules for UI components and frontend code",
		},
	}
}
