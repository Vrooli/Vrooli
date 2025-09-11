package main

func initializeDefaultPresets(orchestrator *Orchestrator) {
	orchestrator.presets["full-maintenance"] = &Preset{
		ID:          "full-maintenance",
		Name:        "Full Maintenance",
		Description: "Activate all maintenance scenarios",
		Pattern:     "*",
		IsDefault:   true,
		States:      make(map[string]bool),
	}

	orchestrator.presets["security-only"] = &Preset{
		ID:          "security-only",
		Name:        "Security Only",
		Description: "Security-related maintenance only",
		Tags:        []string{"security"},
		IsDefault:   true,
		States:      make(map[string]bool),
	}

	orchestrator.presets["performance"] = &Preset{
		ID:          "performance",
		Name:        "Performance",
		Description: "Performance optimization scenarios",
		Tags:        []string{"performance", "optimization"},
		IsDefault:   true,
		States:      make(map[string]bool),
	}

	orchestrator.presets["off-hours"] = &Preset{
		ID:          "off-hours",
		Name:        "Off Hours",
		Description: "Heavy maintenance for quiet periods",
		Tags:        []string{"heavy", "backup", "cleanup"},
		IsDefault:   true,
		States:      make(map[string]bool),
	}

	orchestrator.presets["minimal"] = &Preset{
		ID:          "minimal",
		Name:        "Minimal",
		Description: "Essential maintenance only",
		Tags:        []string{"essential", "critical"},
		IsDefault:   true,
		States:      make(map[string]bool),
	}
}