package export

import (
	"time"

	"github.com/google/uuid"
	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
)

// Schema version for replay export packages.
const replayExportSchemaVersion = "2025-11-07"

// ReplayMovieSpec encapsulates replay metadata ready for renderer/export tooling.
type ReplayMovieSpec struct {
	Version      string                  `json:"version"`
	GeneratedAt  time.Time               `json:"generated_at"`
	Execution    ExportExecutionMetadata `json:"execution"`
	Theme        ExportTheme             `json:"theme"`
	Cursor       ExportCursorSpec        `json:"cursor"`
	Decor        ExportDecor             `json:"decor"`
	Watermark    *ExportWatermark        `json:"watermark,omitempty"`
	IntroCard    *ExportIntroCard        `json:"intro_card,omitempty"`
	OutroCard    *ExportOutroCard        `json:"outro_card,omitempty"`
	Playback     ExportPlayback          `json:"playback"`
	Presentation ExportPresentation      `json:"presentation"`
	CursorMotion ExportCursorMotion      `json:"cursor_motion"`
	Frames       []ExportFrame           `json:"frames"`
	Assets       []ExportAsset           `json:"assets"`
	Summary      ExportSummary           `json:"summary"`
}

// ExportExecutionMetadata summarises execution context for the replay movie spec.
type ExportExecutionMetadata struct {
	ExecutionID   uuid.UUID  `json:"execution_id"`
	WorkflowID    uuid.UUID  `json:"workflow_id"`
	WorkflowName  string     `json:"workflow_name,omitempty"`
	Status        string     `json:"status"`
	StartedAt     time.Time  `json:"started_at"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
	Progress      int        `json:"progress"`
	TotalDuration int        `json:"total_duration_ms"`
}

// ExportTheme encodes chrome/backdrop styling hints for marketing renders.
type ExportTheme struct {
	BackgroundGradient []string            `json:"background_gradient"`
	BackgroundPattern  string              `json:"background_pattern"`
	AccentColor        string              `json:"accent_color"`
	SurfaceColor       string              `json:"surface_color"`
	AmbientGlow        string              `json:"ambient_glow"`
	BrowserChrome      ExportBrowserChrome `json:"browser_chrome"`
}

// ExportBrowserChrome defines the stylised browser frame settings.
type ExportBrowserChrome struct {
	Visible     bool   `json:"visible"`
	Variant     string `json:"variant"`
	Title       string `json:"title,omitempty"`
	ShowAddress bool   `json:"show_address"`
	AccentColor string `json:"accent_color"`
}

// ExportCursorSpec configures synthetic cursor overlays for the replay.
type ExportCursorSpec struct {
	Style       string            `json:"style"`
	AccentColor string            `json:"accent_color"`
	Trail       ExportCursorTrail `json:"trail"`
	ClickPulse  ExportClickPulse  `json:"click_pulse"`
	Scale       float64           `json:"scale,omitempty"`
	InitialPos  string            `json:"initial_position,omitempty"`
	ClickAnim   string            `json:"click_animation,omitempty"`
}

// ExportDecor captures the high-level presentation presets selected in the UI.
type ExportDecor struct {
	ChromeTheme          string         `json:"chrome_theme,omitempty"`
	BackgroundTheme      string         `json:"background_theme,omitempty"`
	Background           map[string]any `json:"background,omitempty"`
	CursorTheme          string         `json:"cursor_theme,omitempty"`
	CursorInitial        string         `json:"cursor_initial_position,omitempty"`
	CursorClickAnimation string         `json:"cursor_click_animation,omitempty"`
	CursorScale          float64        `json:"cursor_scale,omitempty"`
}

// ExportWatermark defines watermark settings for replay exports.
type ExportWatermark struct {
	Enabled  bool   `json:"enabled"`
	AssetID  string `json:"asset_id,omitempty"`
	Position string `json:"position,omitempty"`
	Size     int    `json:"size,omitempty"`
	Opacity  int    `json:"opacity,omitempty"`
	Margin   int    `json:"margin,omitempty"`
}

// ExportIntroCard defines intro card settings for replay exports.
type ExportIntroCard struct {
	Enabled           bool   `json:"enabled"`
	Title             string `json:"title,omitempty"`
	Subtitle          string `json:"subtitle,omitempty"`
	LogoAssetID       string `json:"logo_asset_id,omitempty"`
	BackgroundAssetID string `json:"background_asset_id,omitempty"`
	BackgroundColor   string `json:"background_color,omitempty"`
	TextColor         string `json:"text_color,omitempty"`
	DurationMs        int    `json:"duration_ms,omitempty"`
}

// ExportOutroCard defines outro card settings for replay exports.
type ExportOutroCard struct {
	Enabled           bool   `json:"enabled"`
	Title             string `json:"title,omitempty"`
	CtaText           string `json:"cta_text,omitempty"`
	CtaURL            string `json:"cta_url,omitempty"`
	LogoAssetID       string `json:"logo_asset_id,omitempty"`
	BackgroundAssetID string `json:"background_asset_id,omitempty"`
	BackgroundColor   string `json:"background_color,omitempty"`
	TextColor         string `json:"text_color,omitempty"`
	DurationMs        int    `json:"duration_ms,omitempty"`
}

// ExportPlayback captures canonical playback configuration for renderers.
type ExportPlayback struct {
	FPS             int `json:"fps"`
	DurationMs      int `json:"duration_ms"`
	FrameIntervalMs int `json:"frame_interval_ms"`
	TotalFrames     int `json:"total_frames"`
}

// ExportFrameRect describes the chrome bounding box for presentation framing.
type ExportFrameRect struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
	Radius int `json:"radius,omitempty"`
}

// ExportPresentation captures the canvas and viewport dimensions for replay.
type ExportPresentation struct {
	Canvas            ExportDimensions `json:"canvas"`
	Viewport          ExportDimensions `json:"viewport"`
	BrowserFrame      ExportFrameRect  `json:"browser_frame"`
	DeviceScaleFactor float64          `json:"device_scale_factor"`
}

// ExportCursorMotion summarises default cursor movement behaviours.
type ExportCursorMotion struct {
	SpeedProfile    string  `json:"speed_profile"`
	PathStyle       string  `json:"path_style"`
	InitialPosition string  `json:"initial_position"`
	ClickAnimation  string  `json:"click_animation"`
	CursorScale     float64 `json:"cursor_scale"`
}

// ExportCursorTrail describes cursor trail animation.
type ExportCursorTrail struct {
	Enabled bool    `json:"enabled"`
	FadeMs  int     `json:"fade_ms"`
	Weight  float64 `json:"weight"`
	Opacity float64 `json:"opacity"`
}

// ExportClickPulse configures click pulse animation parameters.
type ExportClickPulse struct {
	Enabled    bool    `json:"enabled"`
	Radius     float64 `json:"radius"`
	DurationMs int     `json:"duration_ms"`
	Opacity    float64 `json:"opacity"`
}

// ExportFrame captures per-step replay metadata plus animation hints.
type ExportFrame struct {
	Index                   int                              `json:"index"`
	StepIndex               int                              `json:"step_index"`
	NodeID                  string                           `json:"node_id"`
	StepType                string                           `json:"step_type"`
	Title                   string                           `json:"title"`
	Status                  string                           `json:"status"`
	StartOffsetMs           int                              `json:"start_offset_ms"`
	DurationMs              int                              `json:"duration_ms"`
	HoldMs                  int                              `json:"hold_ms"`
	Enter                   TransitionSpec                   `json:"enter"`
	Exit                    TransitionSpec                   `json:"exit"`
	ScreenshotAssetID       string                           `json:"screenshot_asset_id,omitempty"`
	Viewport                ExportDimensions                 `json:"viewport"`
	ZoomFactor              float64                          `json:"zoom_factor,omitempty"`
	HighlightRegions        []*autocontracts.HighlightRegion `json:"highlight_regions,omitempty"`
	MaskRegions             []*autocontracts.MaskRegion      `json:"mask_regions,omitempty"`
	FocusedElement          *autocontracts.ElementFocus      `json:"focused_element,omitempty"`
	ElementBoundingBox      *autocontracts.BoundingBox       `json:"element_bounding_box,omitempty"`
	NormalizedFocusBounds   *ExportNormalizedRect            `json:"normalized_focus_bounds,omitempty"`
	NormalizedElementBounds *ExportNormalizedRect            `json:"normalized_element_bounds,omitempty"`
	ClickPosition           *autocontracts.Point             `json:"click_position,omitempty"`
	NormalizedClickPosition *ExportNormalizedPoint           `json:"normalized_click_position,omitempty"`
	CursorTrail             []*autocontracts.Point           `json:"cursor_trail,omitempty"`
	NormalizedCursorTrail   []ExportNormalizedPoint          `json:"normalized_cursor_trail,omitempty"`
	ConsoleLogCount         int                              `json:"console_log_count,omitempty"`
	NetworkEventCount       int                              `json:"network_event_count,omitempty"`
	FinalURL                string                           `json:"final_url,omitempty"`
	Error                   string                           `json:"error,omitempty"`
	Assertion               *autocontracts.AssertionOutcome  `json:"assertion,omitempty"`
	Resilience              ExportResilience                 `json:"resilience"`
}

// ExportResilience captures retry/backoff metadata for a step.
type ExportResilience struct {
	Attempt           int                 `json:"attempt"`
	MaxAttempts       int                 `json:"max_attempts"`
	ConfiguredRetries int                 `json:"configured_retries"`
	DelayMs           int                 `json:"delay_ms"`
	BackoffFactor     float64             `json:"backoff_factor"`
	History           []RetryHistoryEntry `json:"history,omitempty"`
}

// TransitionSpec defines enter/exit animation parameters.
type TransitionSpec struct {
	Type       string `json:"type"`
	DurationMs int    `json:"duration_ms"`
	Easing     string `json:"easing"`
}

// ExportAsset lists external assets (screenshots, audio) required for the replay.
type ExportAsset struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Source    string `json:"source"`
	Thumbnail string `json:"thumbnail,omitempty"`
	Width     int    `json:"width,omitempty"`
	Height    int    `json:"height,omitempty"`
	SizeBytes *int64 `json:"size_bytes,omitempty"`
}

// ExportSummary aggregates package-level metrics.
type ExportSummary struct {
	FrameCount         int `json:"frame_count"`
	ScreenshotCount    int `json:"screenshot_count"`
	TotalDurationMs    int `json:"total_duration_ms"`
	MaxFrameDurationMs int `json:"max_frame_duration_ms"`
}

// ExportDimensions captures viewport dimensions for a frame.
type ExportDimensions struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// ExportNormalizedPoint stores a point normalised to 0..1 range.
type ExportNormalizedPoint struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// ExportNormalizedRect stores a rectangle normalised to 0..1 range.
type ExportNormalizedRect struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}
