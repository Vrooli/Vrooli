package export

import "strings"

// Default playback and frame timing constants.
const (
	defaultFrameDurationMs         = 1600
	minFrameDurationMs             = 900
	minHoldDurationMs              = 600
	transitionFraction             = 0.22
	minTransitionDurationMs        = 180
	maxTransitionDurationMs        = 600
	fallbackViewportWidth          = 1920
	fallbackViewportHeight         = 1080
	defaultPlaybackFrameIntervalMs = 40
	defaultCursorSpeedProfile      = "easeInOut"
	defaultCursorPathStyle         = "bezier"
)

// DefaultExportTheme returns the default theme configuration for replay exports.
func DefaultExportTheme(workflowName, accent string) ExportTheme {
	title := workflowName
	if strings.TrimSpace(title) == "" {
		title = "Vrooli Ascension"
	}

	return ExportTheme{
		BackgroundGradient: []string{"#0f172a", "#020617", "#111827"},
		BackgroundPattern:  "orbits",
		AccentColor:        accent,
		SurfaceColor:       "rgba(15,23,42,0.72)",
		AmbientGlow:        "rgba(56,189,248,0.22)",
		BrowserChrome: ExportBrowserChrome{
			Visible:     true,
			Variant:     "dark",
			Title:       title,
			ShowAddress: true,
			AccentColor: accent,
		},
	}
}

// DefaultCursorSpec returns the default cursor configuration for replay exports.
func DefaultCursorSpec(accent string) ExportCursorSpec {
	return ExportCursorSpec{
		Style:       "halo",
		AccentColor: accent,
		Trail: ExportCursorTrail{
			Enabled: true,
			FadeMs:  650,
			Weight:  0.16,
			Opacity: 0.55,
		},
		ClickPulse: ExportClickPulse{
			Enabled:    true,
			Radius:     42,
			DurationMs: 420,
			Opacity:    0.65,
		},
		Scale:      1.0,
		InitialPos: "center",
		ClickAnim:  "pulse",
	}
}

// DefaultDecor returns the default decoration configuration for replay exports.
func DefaultDecor(cursor ExportCursorSpec) ExportDecor {
	return ExportDecor{
		ChromeTheme:          "aurora",
		BackgroundTheme:      "aurora",
		Background:           map[string]any{"type": "theme", "id": "aurora"},
		CursorTheme:          "white",
		CursorInitial:        cursor.InitialPos,
		CursorClickAnimation: cursor.ClickAnim,
		CursorScale:          cursor.Scale,
	}
}
