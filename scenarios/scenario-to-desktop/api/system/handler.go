package system

// Handler owns the collaborators used by the generated SystemService
// implementation. System administration is exposed exclusively through the
// typed Connect service; this type deliberately contains no HTTP transport.
type Handler struct {
	wineService *DefaultWineService
	builds      BuildStore
	templateDir string
}

// NewHandler assembles the system domain dependencies.
func NewHandler(wineService *DefaultWineService, builds BuildStore, templateDir string) *Handler {
	return &Handler{
		wineService: wineService,
		builds:      builds,
		templateDir: templateDir,
	}
}

func templateInfos() []TemplateInfo {
	return []TemplateInfo{
		{
			Name:        "Universal Desktop App",
			Description: "Universal wrapper that works for any scenario. Default choice - clean, professional desktop application with essential features.",
			Type:        "universal",
			Framework:   "electron",
			UseCases:    []string{"Any scenario needing desktop deployment", "Default choice", "Simple utilities", "Productivity apps", "Quick prototypes"},
			Features:    []string{"Native menus", "Auto-updater", "File operations", "Clean splash screen"},
			Complexity:  "low",
			Examples:    []string{"picker-wheel", "qr-code-generator", "palette-gen", "nutrition-tracker"},
		},
		{
			Name:        "Advanced Desktop App",
			Description: "Full-featured desktop application with advanced OS integration",
			Type:        "advanced",
			Framework:   "electron",
			UseCases:    []string{"Professional tools", "System administration", "Background services"},
			Features:    []string{"System tray", "Global shortcuts", "Rich notifications"},
			Complexity:  "medium",
			Examples:    []string{"system-monitor", "document-manager", "research-assistant"},
		},
		{
			Name:        "Multi-Window Desktop App",
			Description: "Advanced application supporting multiple windows and complex workflows",
			Type:        "multi_window",
			Framework:   "electron",
			UseCases:    []string{"IDE-like applications", "Dashboard applications", "Professional tools"},
			Features:    []string{"Window management", "Inter-window communication", "Floating panels"},
			Complexity:  "high",
			Examples:    []string{"agent-dashboard", "mind-maps", "brand-manager"},
		},
		{
			Name:        "Kiosk Mode App",
			Description: "Full-screen application for dedicated hardware and public displays",
			Type:        "kiosk",
			Framework:   "electron",
			UseCases:    []string{"Public displays", "Point-of-sale", "Industrial controls"},
			Features:    []string{"Full-screen lock", "Remote monitoring", "Auto-restart"},
			Complexity:  "high",
			Examples:    []string{"information-display", "booking-system", "retail-kiosk"},
		},
	}
}
