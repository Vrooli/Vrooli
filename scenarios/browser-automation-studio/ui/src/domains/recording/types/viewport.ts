/**
 * Viewport Types - Single source of truth for all viewport-related types
 *
 * This module consolidates all viewport type definitions to avoid duplication
 * and ensure consistent naming across the recording domain.
 *
 * Naming conventions:
 * - ContainerBounds: The measured CSS dimensions of a container element
 * - BrowserViewport: The logical viewport size requested from/used by Playwright
 * - ActualViewport: The viewport reported by Playwright (may differ due to profile)
 * - DisplayDimensions: The rendered size of content (may include letterboxing)
 *
 * @module recording/types/viewport
 */

// =============================================================================
// Base Types
// =============================================================================

/**
 * Basic width/height dimensions.
 * Used for container bounds measurement and browser viewport requests.
 */
export interface ViewportDimensions {
  width: number;
  height: number;
}

/**
 * Alias for ViewportDimensions for semantic clarity in container measurement contexts.
 */
export type ContainerBounds = ViewportDimensions;

/**
 * Alias for ViewportDimensions for semantic clarity in browser viewport contexts.
 */
export type BrowserViewport = ViewportDimensions;

// =============================================================================
// Source Attribution
// =============================================================================

/**
 * Describes what determined the actual viewport dimensions.
 * This attribution helps users understand why dimensions may differ from requested.
 */
export type ViewportSource =
  | 'requested'           // Used the UI-requested dimensions
  | 'fingerprint'         // Browser profile fingerprint override
  | 'fingerprint_partial' // Fingerprint set one dimension, requested used for other
  | 'default';            // Fallback defaults used

/**
 * Actual viewport with source attribution.
 * Includes the dimensions and explanation of what determined them.
 *
 * This is returned by the backend when a session is created or viewport is updated,
 * because the actual dimensions may differ from requested due to session profile settings.
 */
export interface ActualViewport extends ViewportDimensions {
  /** What determined these dimensions */
  source: ViewportSource;
  /** Human-readable explanation of why these dimensions were used */
  reason: string;
}

/**
 * Partial actual viewport where source/reason are optional.
 * Used when receiving data from APIs that may not include attribution.
 */
export interface ActualViewportOptional extends ViewportDimensions {
  source?: ViewportSource;
  reason?: string;
}

// =============================================================================
// Sync State Types
// =============================================================================

/**
 * State of viewport synchronization with the backend.
 */
export interface ViewportSyncState {
  /** Current computed viewport dimensions */
  viewport: ViewportDimensions | null;
  /** Whether a resize operation is in progress (rapid changes detected) */
  isResizing: boolean;
  /** Whether viewport sync to backend is pending */
  isSyncing: boolean;
  /** Last successful sync timestamp */
  lastSyncTime: number | null;
  /** Error from last sync attempt */
  syncError: string | null;
}

// =============================================================================
// Utility Functions
// =============================================================================

/**
 * Check if two viewport dimensions are equal within a tolerance.
 * @param a First viewport (can be null)
 * @param b Second viewport (can be null)
 * @param tolerance Maximum allowed difference in pixels (default: 1)
 */
export function viewportsEqual(
  a: ViewportDimensions | null,
  b: ViewportDimensions | null,
  tolerance = 1
): boolean {
  if (a === null && b === null) return true;
  if (a === null || b === null) return false;
  return (
    Math.abs(a.width - b.width) <= tolerance &&
    Math.abs(a.height - b.height) <= tolerance
  );
}

/**
 * Get aspect ratio of a viewport (width / height).
 */
export function getAspectRatio(viewport: ViewportDimensions): number {
  return viewport.height > 0 ? viewport.width / viewport.height : 1;
}

/**
 * Clamp viewport dimensions within min/max bounds.
 */
export function clampViewport(
  viewport: ViewportDimensions,
  minDimension = 320,
  maxDimension = 3840
): ViewportDimensions {
  return {
    width: Math.max(minDimension, Math.min(maxDimension, Math.round(viewport.width))),
    height: Math.max(minDimension, Math.min(maxDimension, Math.round(viewport.height))),
  };
}

/**
 * Calculate viewport that fits within bounds while preserving aspect ratio.
 * Used for "contain" style fitting where the viewport is scaled down to fit.
 */
export function fitViewportToBounds(
  viewport: ViewportDimensions,
  bounds: ViewportDimensions
): ViewportDimensions {
  const viewportRatio = getAspectRatio(viewport);
  const boundsRatio = getAspectRatio(bounds);

  if (viewportRatio > boundsRatio) {
    // Viewport is wider - constrain by width
    return {
      width: bounds.width,
      height: Math.round(bounds.width / viewportRatio),
    };
  } else {
    // Viewport is taller - constrain by height
    return {
      width: Math.round(bounds.height * viewportRatio),
      height: bounds.height,
    };
  }
}

/**
 * Check if viewport has a significant mismatch from another (beyond tolerance).
 */
export function hasDimensionMismatch(
  requested: ViewportDimensions | null,
  actual: ViewportDimensions | null,
  tolerance = 5
): boolean {
  if (!requested || !actual) return false;
  const widthDiff = Math.abs(requested.width - actual.width);
  const heightDiff = Math.abs(requested.height - actual.height);
  return widthDiff > tolerance || heightDiff > tolerance;
}
