// DOC: docs/reference/configuration.md
// Tunable levers for the Stream of Consciousness Analyzer UI.
// Each constant is a deliberate, named control surface—easy to find, safe to adjust.

// --- Canvas (spatial view) ---

/** Minimum zoom level (0.25 = 25%). Lower values let users see more at once. */
export const CANVAS_ZOOM_MIN = 0.25;
/** Maximum zoom level (4 = 400%). Higher values let users zoom in further. */
export const CANVAS_ZOOM_MAX = 4;
/** Zoom-in multiplier per scroll tick. Values closer to 1 = slower zoom. */
export const CANVAS_ZOOM_IN_FACTOR = 1.1;
/** Zoom-out multiplier per scroll tick. Values closer to 1 = slower zoom. */
export const CANVAS_ZOOM_OUT_FACTOR = 0.9;

// --- Initial placement ---

/** Width of the random placement area for new information items (px). */
export const INFO_PLACEMENT_WIDTH = 600;
/** Height of the random placement area for new information items (px). */
export const INFO_PLACEMENT_HEIGHT = 400;
/** Width of the random placement area for new thought nodes (px). */
export const THOUGHT_PLACEMENT_WIDTH = 500;
/** Height of the random placement area for new thought nodes (px). */
export const THOUGHT_PLACEMENT_HEIGHT = 300;

// --- Graph view ---

/** SVG edge stroke color between connected thoughts. */
export const EDGE_STROKE_COLOR = "rgba(148,163,184,0.3)";
/** SVG edge stroke width in pixels. */
export const EDGE_STROKE_WIDTH = 2;
/** Minimum height of the thought graph container (px). */
export const GRAPH_MIN_HEIGHT = 400;
/**
 * Sentinel value for the link-mode state machine in GraphView.
 *
 * Link-mode has three states (stored in the `linkSource` variable):
 *   - null              → link-mode OFF (normal interaction)
 *   - LINK_MODE_WAITING → link-mode ON, waiting for user to click a source thought
 *   - <thought-id>      → source selected, next click creates the edge to target
 *
 * This sentinel is intentionally an invalid UUID so it cannot collide with
 * a real thought ID.
 */
export const LINK_MODE_WAITING = "__waiting__";

// --- Text capture ---

/** Default number of visible rows in the capture textarea. */
export const TEXT_CAPTURE_ROWS = 2;

// --- Polling & health ---

/** Interval (ms) for polling LLM provider status. */
export const PROVIDER_POLL_MS = 30_000;
/** Interval (ms) for polling API health when connection is healthy. */
export const HEALTH_POLL_MS = 15_000;
/** Interval (ms) for polling API health when connection is degraded (faster retry). */
export const HEALTH_DEGRADED_POLL_MS = 5_000;
