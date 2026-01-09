package export

// ChromeThemePreset defines the visual configuration for the browser chrome overlay.
type ChromeThemePreset struct {
	Visible     bool
	Variant     string
	ShowAddress bool
	AccentColor string
	AmbientGlow string
}

// BackgroundThemePreset defines the visual configuration for the export background.
type BackgroundThemePreset struct {
	Gradient    []string
	Pattern     string
	Surface     string
	AmbientGlow string
}

// CursorThemePreset defines the visual configuration for cursor rendering.
type CursorThemePreset struct {
	Style        string
	AccentColor  string
	TrailEnabled bool
	TrailOpacity float64
	TrailFadeMs  int
	TrailWeight  float64
	ClickEnabled bool
	ClickOpacity float64
}

const (
	// DefaultAccentColor is the default accent color for themes and cursors.
	DefaultAccentColor = "#38BDF8"

	// CursorScaleMin is the minimum allowed cursor scale.
	CursorScaleMin = 0.5
	// CursorScaleMax is the maximum allowed cursor scale.
	CursorScaleMax = 3.0
)

// ChromeThemePresets provides predefined chrome theme configurations.
var ChromeThemePresets = map[string]ChromeThemePreset{
	"aurora": {
		Visible:     true,
		Variant:     "aurora",
		ShowAddress: true,
		AccentColor: "#38BDF8",
		AmbientGlow: "rgba(56,189,248,0.28)",
	},
	"chromium": {
		Visible:     true,
		Variant:     "chromium",
		ShowAddress: true,
		AccentColor: "#0EA5E9",
		AmbientGlow: "rgba(59,130,246,0.22)",
	},
	"midnight": {
		Visible:     true,
		Variant:     "midnight",
		ShowAddress: true,
		AccentColor: "#A855F7",
		AmbientGlow: "rgba(168,85,247,0.26)",
	},
	"minimal": {
		Visible:     false,
		Variant:     "minimal",
		ShowAddress: false,
		AccentColor: "#94A3B8",
		AmbientGlow: "rgba(148,163,184,0.18)",
	},
}

// BackgroundThemePresets provides predefined background theme configurations.
var BackgroundThemePresets = map[string]BackgroundThemePreset{
	"aurora": {
		Gradient:    []string{"#0F172A", "#1D4ED8", "#38BDF8"},
		Pattern:     "aurora",
		Surface:     "rgba(15,23,42,0.78)",
		AmbientGlow: "rgba(56,189,248,0.24)",
	},
	"sunset": {
		Gradient:    []string{"#F472B6", "#FBBF24", "#43112D"},
		Pattern:     "sunset",
		Surface:     "rgba(43,12,30,0.85)",
		AmbientGlow: "rgba(244,114,182,0.24)",
	},
	"ocean": {
		Gradient:    []string{"#0EA5E9", "#1D4ED8", "#020617"},
		Pattern:     "ocean",
		Surface:     "rgba(8,31,60,0.82)",
		AmbientGlow: "rgba(14,116,144,0.26)",
	},
	"nebula": {
		Gradient:    []string{"#A855F7", "#6366F1", "#111827"},
		Pattern:     "nebula",
		Surface:     "rgba(49,16,80,0.82)",
		AmbientGlow: "rgba(147,51,234,0.26)",
	},
	"grid": {
		Gradient:    []string{"#0F172A", "#0B1120"},
		Pattern:     "grid",
		Surface:     "rgba(8,29,57,0.9)",
		AmbientGlow: "rgba(14,165,233,0.18)",
	},
	"charcoal": {
		Gradient:    []string{"#0F172A", "#020617"},
		Pattern:     "charcoal",
		Surface:     "rgba(15,23,42,0.92)",
		AmbientGlow: "rgba(148,163,184,0.18)",
	},
	"steel": {
		Gradient:    []string{"#1F2937", "#0B1120"},
		Pattern:     "steel",
		Surface:     "rgba(30,41,59,0.9)",
		AmbientGlow: "rgba(148,163,184,0.22)",
	},
	"emerald": {
		Gradient:    []string{"#10B981", "#064E3B", "#022C22"},
		Pattern:     "emerald",
		Surface:     "rgba(2,44,34,0.88)",
		AmbientGlow: "rgba(16,185,129,0.22)",
	},
	"none": {
		Gradient:    []string{"#0F172A"},
		Pattern:     "none",
		Surface:     "rgba(15,23,42,0.85)",
		AmbientGlow: "rgba(59,130,246,0.18)",
	},
}

// CursorThemePresets provides predefined cursor theme configurations.
var CursorThemePresets = map[string]CursorThemePreset{
	"white": {
		Style:        "halo",
		AccentColor:  "#F8FAFC",
		TrailEnabled: true,
		TrailOpacity: 0.5,
		TrailFadeMs:  700,
		TrailWeight:  0.16,
		ClickEnabled: true,
		ClickOpacity: 0.6,
	},
	"dark": {
		Style:        "halo",
		AccentColor:  "#0F172A",
		TrailEnabled: true,
		TrailOpacity: 0.5,
		TrailFadeMs:  700,
		TrailWeight:  0.18,
		ClickEnabled: true,
		ClickOpacity: 0.6,
	},
	"aura": {
		Style:        "halo",
		AccentColor:  "#38BDF8",
		TrailEnabled: true,
		TrailOpacity: 0.6,
		TrailFadeMs:  800,
		TrailWeight:  0.22,
		ClickEnabled: true,
		ClickOpacity: 0.7,
	},
	"arrowLight": {
		Style:        "arrow-light",
		AccentColor:  "#FFFFFF",
		TrailEnabled: false,
		ClickEnabled: true,
		ClickOpacity: 0.55,
	},
	"arrowDark": {
		Style:        "arrow-dark",
		AccentColor:  "#1E293B",
		TrailEnabled: false,
		ClickEnabled: true,
		ClickOpacity: 0.55,
	},
	"arrowNeon": {
		Style:        "arrow-neon",
		AccentColor:  "#38BDF8",
		TrailEnabled: true,
		TrailOpacity: 0.4,
		TrailFadeMs:  600,
		TrailWeight:  0.14,
		ClickEnabled: true,
		ClickOpacity: 0.6,
	},
	"handNeutral": {
		Style:        "hand-neutral",
		AccentColor:  "#F1F5F9",
		TrailEnabled: false,
		ClickEnabled: true,
		ClickOpacity: 0.55,
	},
	"handAura": {
		Style:        "hand-aura",
		AccentColor:  "#38BDF8",
		TrailEnabled: true,
		TrailOpacity: 0.45,
		TrailFadeMs:  720,
		TrailWeight:  0.18,
		ClickEnabled: true,
		ClickOpacity: 0.65,
	},
}

// ClampCursorScale constrains a cursor scale value to the valid range [0.5, 3.0].
func ClampCursorScale(value float64) float64 {
	if value < CursorScaleMin {
		return CursorScaleMin
	}
	if value > CursorScaleMax {
		return CursorScaleMax
	}
	return value
}
