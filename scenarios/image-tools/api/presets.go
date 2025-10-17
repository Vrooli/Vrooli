package main

// PresetProfile defines a reusable image processing configuration
type PresetProfile struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Operations  []PresetOperation      `json:"operations"`
	TargetUse   string                 `json:"target_use"`
}

type PresetOperation struct {
	Type    string                 `json:"type"`
	Options map[string]interface{} `json:"options"`
}

// Built-in preset profiles
var BuiltinPresets = map[string]PresetProfile{
	"web-optimized": {
		Name:        "web-optimized",
		Description: "Optimized for web delivery - balanced quality and size",
		TargetUse:   "websites, blogs, web applications",
		Operations: []PresetOperation{
			{
				Type: "resize",
				Options: map[string]interface{}{
					"max_width":       1920,
					"max_height":      1080,
					"maintain_aspect": true,
					"algorithm":       "lanczos",
				},
			},
			{
				Type: "compress",
				Options: map[string]interface{}{
					"quality": 85,
					"format":  "webp",
				},
			},
			{
				Type: "metadata",
				Options: map[string]interface{}{
					"strip": true,
				},
			},
		},
	},
	"email-safe": {
		Name:        "email-safe",
		Description: "Small file size for email attachments",
		TargetUse:   "email attachments, mobile messaging",
		Operations: []PresetOperation{
			{
				Type: "resize",
				Options: map[string]interface{}{
					"max_width":       800,
					"max_height":      600,
					"maintain_aspect": true,
					"algorithm":       "lanczos",
				},
			},
			{
				Type: "compress",
				Options: map[string]interface{}{
					"quality": 75,
				},
			},
			{
				Type: "metadata",
				Options: map[string]interface{}{
					"strip": true,
				},
			},
		},
	},
	"aggressive": {
		Name:        "aggressive",
		Description: "Maximum compression for minimal file size",
		TargetUse:   "thumbnails, previews, low-bandwidth scenarios",
		Operations: []PresetOperation{
			{
				Type: "resize",
				Options: map[string]interface{}{
					"max_width":       1024,
					"max_height":      768,
					"maintain_aspect": true,
					"algorithm":       "bilinear",
				},
			},
			{
				Type: "compress",
				Options: map[string]interface{}{
					"quality": 60,
				},
			},
			{
				Type: "metadata",
				Options: map[string]interface{}{
					"strip": true,
				},
			},
		},
	},
	"high-quality": {
		Name:        "high-quality",
		Description: "Minimal compression for print and professional use",
		TargetUse:   "print materials, professional photography, archival",
		Operations: []PresetOperation{
			{
				Type: "compress",
				Options: map[string]interface{}{
					"quality": 95,
				},
			},
		},
	},
	"social-media": {
		Name:        "social-media",
		Description: "Optimized for social media platforms",
		TargetUse:   "Instagram, Facebook, Twitter posts",
		Operations: []PresetOperation{
			{
				Type: "resize",
				Options: map[string]interface{}{
					"max_width":       1200,
					"max_height":      1200,
					"maintain_aspect": true,
					"algorithm":       "lanczos",
				},
			},
			{
				Type: "compress",
				Options: map[string]interface{}{
					"quality": 82,
					"format":  "jpeg",
				},
			},
			{
				Type: "metadata",
				Options: map[string]interface{}{
					"strip": true,
				},
			},
		},
	},
}

// GetPreset retrieves a preset by name
func GetPreset(name string) (PresetProfile, bool) {
	preset, ok := BuiltinPresets[name]
	return preset, ok
}

// ListPresets returns all available presets
func ListPresets() []PresetProfile {
	presets := make([]PresetProfile, 0, len(BuiltinPresets))
	for _, preset := range BuiltinPresets {
		presets = append(presets, preset)
	}
	return presets
}
