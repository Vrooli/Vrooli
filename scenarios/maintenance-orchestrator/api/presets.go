package main

func initializeDefaultPresets(orchestrator *Orchestrator) {
	orchestrator.presets["full"] = &Preset{
		ID:          "full",
		Name:        "Full",
		Description: "Activate all maintenance scenarios",
		Pattern:     "*",
		Tags:        []string{"maintenance"},
		IsDefault:   true,
		IsActive:    false,
		States:      make(map[string]bool),
	}

	orchestrator.presets["emergency"] = &Preset{
		ID:          "emergency",
		Name:        "Emergency",
		Description: "Emergency response scenarios",
		Tags:        []string{"maintenance", "emergency-response"},
		IsDefault:   true,
		IsActive:    false,
		States:      make(map[string]bool),
	}

	orchestrator.presets["security"] = &Preset{
		ID:          "security",
		Name:        "Security",
		Description: "Security-related maintenance scenarios",
		Tags:        []string{"maintenance", "security"},
		IsDefault:   true,
		IsActive:    false,
		States:      make(map[string]bool),
	}

	orchestrator.presets["self-improvement"] = &Preset{
		ID:          "self-improvement",
		Name:        "Self-Improvement",
		Description: "Self-improvement and learning scenarios",
		Tags:        []string{"self-improvement"},
		IsDefault:   true,
		IsActive:    false,
		States:      make(map[string]bool),
	}

	orchestrator.presets["performance"] = &Preset{
		ID:          "performance",
		Name:        "Performance",
		Description: "Performance optimization scenarios",
		Tags:        []string{"maintenance", "performance"},
		IsDefault:   true,
		IsActive:    false,
		States:      make(map[string]bool),
	}

	orchestrator.presets["quality"] = &Preset{
		ID:          "quality",
		Name:        "Quality",
		Description: "Quality assurance and testing scenarios",
		Tags:        []string{"maintenance", "quality"},
		IsDefault:   true,
		IsActive:    false,
		States:      make(map[string]bool),
	}

	orchestrator.presets["analytics"] = &Preset{
		ID:          "analytics",
		Name:        "Analytics",
		Description: "Analytics and monitoring scenarios",
		Tags:        []string{"maintenance", "analytics"},
		IsDefault:   true,
		IsActive:    false,
		States:      make(map[string]bool),
	}
}
