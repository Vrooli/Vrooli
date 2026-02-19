// DOC: docs/reference/configuration.md#ui-constants
/**
 * Centralized configuration for the Web Console UI.
 *
 * These are the tunable levers that shape terminal appearance and behavior.
 * Values are designed to be safe defaults that work well for most operators.
 * See docs/reference/configuration.md for full documentation.
 */

// ---------------------------------------------------------------------------
// Terminal Appearance
// ---------------------------------------------------------------------------

/** Terminal color theme applied to xterm.js instances. */
export const TERMINAL_THEME = {
  background: "#0f172a",
  foreground: "#e2e8f0",
  cursor: "#38bdf8",
  selectionBackground: "#334155",
} as const;

/** Font size in pixels for terminal text. Range: 8–24. */
export const TERMINAL_FONT_SIZE = 14;

/**
 * Font family stack for terminal text. Monospace fonts are required for
 * correct terminal rendering. The first available font is used.
 */
export const TERMINAL_FONT_FAMILY =
  "'JetBrains Mono', 'Fira Code', 'Cascadia Code', monospace";

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
