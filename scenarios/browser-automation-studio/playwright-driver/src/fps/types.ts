/**
 * FPS Controller Types
 *
 * Type definitions for the adaptive frame rate controller.
 * Uses a simple target-utilization feedback loop instead of
 * complex multi-mechanism approaches.
 *
 * ## Design Philosophy
 * The controller aims to keep capture time at a target percentage of
 * the frame interval. This single mechanism replaces:
 * - Streak-based adjustments
 * - P90 ceiling calculations
 * - EMA tracking (calculated but unused)
 * - Asymmetric thresholds
 *
 * ## Tuning
 * - targetUtilization: What % of frame time should capture use (0.6-0.8 recommended)
 * - smoothing: How fast to react (0.1 = slow/stable, 0.5 = fast/responsive)
 * - adjustmentInterval: How many frames between FPS changes
 */

/**
 * Configuration for the FPS controller.
 * All values have sensible defaults for typical browser automation.
 */
export interface FpsControllerConfig {
  /** Minimum FPS floor (default: 2) */
  minFps: number;

  /** Maximum FPS ceiling - typically the user's requested target (default: 30) */
  maxFps: number;

  /**
   * Target capture time as fraction of frame interval (default: 0.7)
   *
   * Example: At 10 FPS (100ms interval), target capture = 70ms
   * - If captures average 50ms → room to increase FPS
   * - If captures average 90ms → need to decrease FPS
   *
   * Lower values (0.5-0.6) = more headroom, more conservative
   * Higher values (0.7-0.8) = tighter timing, more aggressive
   */
  targetUtilization: number;

  /**
   * Smoothing factor for FPS adjustments (default: 0.25)
   *
   * Controls how quickly currentFps moves toward idealFps.
   * newFps = currentFps + smoothing * (idealFps - currentFps)
   *
   * Lower values (0.1-0.2) = smoother, slower adaptation
   * Higher values (0.3-0.5) = more responsive, may oscillate
   */
  smoothing: number;

  /**
   * Minimum frames between FPS adjustments (default: 3)
   *
   * Prevents oscillation by requiring multiple samples before changing.
   * Set to 1 for per-frame adjustment (not recommended).
   */
  adjustmentInterval: number;

  /**
   * Threshold multiplier for increasing FPS (default: 0.6)
   *
   * Only increase FPS when avgCapture < targetCapture * increaseThreshold
   * Lower = more aggressive increases, higher = more conservative
   */
  increaseThreshold: number;

  /**
   * Threshold multiplier for decreasing FPS (default: 1.15)
   *
   * Decrease FPS when avgCapture > targetCapture * decreaseThreshold
   * Lower = more aggressive decreases, higher = more tolerant
   */
  decreaseThreshold: number;
}

/**
 * Internal state of the FPS controller.
 * This is mutable and updated by the controller on each frame.
 */
export interface FpsControllerState {
  /** Current FPS being used (may be fractional internally, rounded for intervals) */
  currentFps: number;

  /** Ring buffer of recent capture times for averaging */
  recentCaptures: number[];

  /** Frame counter since last adjustment */
  framesSinceAdjustment: number;

  /** Total frames processed (for diagnostics) */
  totalFrames: number;

  /** Last adjustment direction for logging: 'up' | 'down' | 'none' */
  lastAdjustment: 'up' | 'down' | 'none';

  /** Timestamp of last FPS change (for rate limiting logs) */
  lastAdjustmentTime: number;
}

/**
 * Result of processing a frame through the controller.
 * Contains the updated state and whether an adjustment occurred.
 */
export interface FpsAdjustmentResult {
  /** Updated controller state */
  state: FpsControllerState;

  /** Whether FPS was adjusted this frame */
  adjusted: boolean;

  /** New FPS value (same as state.currentFps, for convenience) */
  newFps: number;

  /** Current frame interval in ms (1000 / newFps) */
  intervalMs: number;

  /** Diagnostic info for logging (only populated when adjusted=true) */
  diagnostics?: FpsAdjustmentDiagnostics;
}

/**
 * Diagnostic information about an FPS adjustment.
 * Useful for debugging and performance analysis.
 */
export interface FpsAdjustmentDiagnostics {
  /** Previous FPS before adjustment */
  previousFps: number;

  /** Average capture time over recent frames */
  avgCaptureMs: number;

  /** Target capture time based on current interval */
  targetCaptureMs: number;

  /** Ideal FPS calculated from capture time */
  idealFps: number;

  /** Reason for adjustment */
  reason: 'too_slow' | 'too_fast' | 'smoothing';
}

/**
 * Default configuration values.
 * These are tuned for typical browser automation workloads.
 */
export const DEFAULT_FPS_CONFIG: FpsControllerConfig = {
  minFps: 2,
  maxFps: 30,
  targetUtilization: 0.7,
  smoothing: 0.25,
  adjustmentInterval: 3,
  increaseThreshold: 0.6,
  decreaseThreshold: 1.15,
};

/**
 * Create initial controller state.
 *
 * @param initialFps - Starting FPS (typically from user config)
 */
export function createInitialState(initialFps: number): FpsControllerState {
  return {
    currentFps: initialFps,
    recentCaptures: [],
    framesSinceAdjustment: 0,
    totalFrames: 0,
    lastAdjustment: 'none',
    lastAdjustmentTime: 0,
  };
}
