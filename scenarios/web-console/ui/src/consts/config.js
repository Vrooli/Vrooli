// DOC: docs/reference/configuration.md#ui-constants
/**
 * Centralized configuration for the Web Console UI.
 *
 * These are the tunable levers that shape terminal appearance and behavior.
 * Values are designed to be safe defaults that work well for most operators.
 * See docs/reference/configuration.md for full documentation.
 */
/** Default theme applied to new panes. */
export const DEFAULT_THEME_ID = "slate-ocean";
/** Curated terminal color theme presets. */
export const TERMINAL_THEMES = {
    "slate-ocean": {
        id: "slate-ocean",
        label: "Slate Ocean",
        colors: { background: "#0f172a", foreground: "#e2e8f0", cursor: "#38bdf8", selectionBackground: "#334155" },
    },
    dracula: {
        id: "dracula",
        label: "Dracula",
        colors: { background: "#282a36", foreground: "#f8f8f2", cursor: "#ff79c6", selectionBackground: "#44475a" },
    },
    "solarized-dark": {
        id: "solarized-dark",
        label: "Solarized Dark",
        colors: { background: "#002b36", foreground: "#839496", cursor: "#b58900", selectionBackground: "#073642" },
    },
    monokai: {
        id: "monokai",
        label: "Monokai",
        colors: { background: "#272822", foreground: "#f8f8f2", cursor: "#f92672", selectionBackground: "#49483e" },
    },
    nord: {
        id: "nord",
        label: "Nord",
        colors: { background: "#2e3440", foreground: "#d8dee9", cursor: "#88c0d0", selectionBackground: "#434c5e" },
    },
    "github-dark": {
        id: "github-dark",
        label: "GitHub Dark",
        colors: { background: "#0d1117", foreground: "#c9d1d9", cursor: "#58a6ff", selectionBackground: "#161b22" },
    },
};
/** Font size in pixels for terminal text. Range: 8–24. */
export const TERMINAL_FONT_SIZE = 14;
/**
 * Font family stack for terminal text. Monospace fonts are required for
 * correct terminal rendering. The first available font is used.
 */
export const TERMINAL_FONT_FAMILY = "'JetBrains Mono', 'Fira Code', 'Cascadia Code', monospace";
// ---------------------------------------------------------------------------
// Terminal Defaults
// ---------------------------------------------------------------------------
// CROSS-LANGUAGE COUPLING: These defaults must match DefaultCols/DefaultRows
// in api/config.go. A mismatch causes resize flicker on session creation.
/** Default terminal column count when creating a new session. */
export const DEFAULT_COLS = 80;
/** Default terminal row count when creating a new session. */
export const DEFAULT_ROWS = 24;
// ---------------------------------------------------------------------------
// Pane Grid Layout
// ---------------------------------------------------------------------------
/**
 * Minimum pane width in CSS before wrapping to next row.
 * Lower values allow more panes side-by-side; higher values improve readability.
 */
export const PANE_MIN_WIDTH_PX = 500;
/**
 * Minimum pane height when more than 2 panes are displayed.
 * Prevents panes from becoming too small to be useful.
 */
export const PANE_MIN_HEIGHT_PX = 300;
// ---------------------------------------------------------------------------
// Connection & Retry
// ---------------------------------------------------------------------------
/** Number of times to retry the initial API health check before showing an error. */
export const HEALTH_RETRY_COUNT = 3;
/** Delay in milliseconds between health check retries. */
export const HEALTH_RETRY_DELAY_MS = 1000;
/** Duration in milliseconds before auto-dismissing session creation errors. */
export const ERROR_AUTO_DISMISS_MS = 8000;
// ---------------------------------------------------------------------------
// Grid Splitter & Resize
// ---------------------------------------------------------------------------
/** Width/height of the splitter handle between grid tracks. */
export const SPLITTER_SIZE_PX = 8;
/** Minimum column width in pixels before resize is clamped. */
export const MIN_COLUMN_PX = 240;
/** Minimum row height in pixels before resize is clamped. */
export const MIN_ROW_PX = 200;
// ---------------------------------------------------------------------------
// Touch Gestures
// ---------------------------------------------------------------------------
/** Pixels of movement before a touch is classified as drag vs tap. */
export const TOUCH_MOVE_THRESHOLD_PX = 8;
/** Milliseconds to hold before entering text-selection mode. */
export const TOUCH_LONG_PRESS_MS = 500;
/** Maximum touch duration (ms) that still counts as a tap. */
export const TOUCH_TAP_MAX_MS = 300;
/** Maximum gap (ms) between taps for a double-tap. */
export const TOUCH_DOUBLE_TAP_MS = 300;
/** Cumulative scroll distance (px) before long-press timer is cancelled.
 *  Small movements (hand tremor) keep the timer alive; once the finger
 *  travels this far it's clearly an intentional scroll. */
export const TOUCH_SCROLL_CANCEL_PX = 30;
/** Per-frame velocity multiplier for momentum scroll (0–1). */
export const TOUCH_SCROLL_DECEL = 0.95;
/** Minimum velocity (px/frame) below which momentum scroll stops. */
export const TOUCH_SCROLL_MIN_VELOCITY = 0.5;
// ---------------------------------------------------------------------------
// Terminal Header Colors
// ---------------------------------------------------------------------------
/**
 * Palette of preset colors for terminal pane headers. Perceptually-even hues
 * (≈equal lightness/chroma, evenly spaced hue) so no two swatches are
 * confusable on a phone — the old equal-lightness pastels had confusable
 * yellow/coral and magenta/pink pairs.
 */
export const HEADER_COLORS = [
    "#ff6b6b", // red
    "#ffa94d", // orange
    "#ffd43b", // yellow
    "#69db7c", // green
    "#38d9c0", // teal
    "#4dabf7", // blue
    "#9775fa", // violet
    "#f783ac", // pink
];
