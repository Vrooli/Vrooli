/**
 * FPS Controller
 *
 * A simple, single-mechanism adaptive frame rate controller.
 *
 * ## How It Works
 *
 * The controller uses a target-utilization feedback loop:
 *
 * 1. Calculate target capture time = frameInterval * targetUtilization
 *    Example: At 10 FPS (100ms interval) with 0.7 utilization → target = 70ms
 *
 * 2. Compare actual average capture time to target:
 *    - avgCapture < target * 0.6 → captures are fast, can increase FPS
 *    - avgCapture > target * 1.15 → captures are slow, must decrease FPS
 *    - Otherwise → in the sweet spot, no change
 *
 * 3. Apply smoothing to prevent oscillation:
 *    newFps = currentFps + smoothing * (idealFps - currentFps)
 *
 * ## Why This Design
 *
 * Previous implementation had 5 interacting mechanisms (streaks, EMA, P90,
 * timeouts, ring buffers) that were hard to reason about and tune.
 *
 * This design has ONE mechanism with clear semantics:
 * "Keep capture time at X% of frame budget"
 *
 * Benefits:
 * - Easy to understand and debug
 * - Single source of truth for FPS decisions
 * - Testable in isolation (pure functions)
 * - Predictable behavior under all conditions
 */

import {
  FpsControllerConfig,
  FpsControllerState,
  FpsAdjustmentResult,
  FpsAdjustmentDiagnostics,
  DEFAULT_FPS_CONFIG,
  createInitialState,
} from './types';

/** Maximum capture times to keep in the ring buffer */
const MAX_CAPTURE_BUFFER_SIZE = 10;

/**
 * Process a frame capture and potentially adjust FPS.
 *
 * This is a pure function - it takes state and returns new state.
 * No side effects, no external dependencies.
 *
 * @param state - Current controller state
 * @param captureTimeMs - How long the screenshot capture took (ms)
 * @param config - Controller configuration
 * @returns Updated state and adjustment info
 *
 * @example
 * ```typescript
 * let state = createInitialState(10);
 * const config = { ...DEFAULT_FPS_CONFIG, maxFps: 20 };
 *
 * // Process each frame
 * const result = processFrame(state, 45, config);
 * state = result.state;
 * const intervalMs = result.intervalMs; // Use for next frame timing
 * ```
 */
export function processFrame(
  state: FpsControllerState,
  captureTimeMs: number,
  config: FpsControllerConfig = DEFAULT_FPS_CONFIG
): FpsAdjustmentResult {
  // Clone state to avoid mutation
  const newState: FpsControllerState = {
    ...state,
    recentCaptures: [...state.recentCaptures],
  };

  // Add capture time to ring buffer
  newState.recentCaptures.push(captureTimeMs);
  if (newState.recentCaptures.length > MAX_CAPTURE_BUFFER_SIZE) {
    newState.recentCaptures.shift();
  }

  newState.totalFrames++;
  newState.framesSinceAdjustment++;

  // Check if we should adjust FPS
  const shouldAdjust =
    newState.framesSinceAdjustment >= config.adjustmentInterval &&
    newState.recentCaptures.length >= Math.min(config.adjustmentInterval, 3);

  if (!shouldAdjust) {
    return {
      state: newState,
      adjusted: false,
      newFps: Math.round(newState.currentFps),
      intervalMs: Math.floor(1000 / newState.currentFps),
    };
  }

  // Calculate average capture time
  const avgCaptureMs = average(newState.recentCaptures);

  // Calculate target capture time based on current FPS
  const currentIntervalMs = 1000 / newState.currentFps;
  const targetCaptureMs = currentIntervalMs * config.targetUtilization;

  // Determine ideal FPS based on actual capture performance
  // If captures take X ms, ideal interval = X / targetUtilization
  // Ideal FPS = 1000 / idealInterval
  const idealIntervalMs = avgCaptureMs / config.targetUtilization;
  const idealFps = clamp(1000 / idealIntervalMs, config.minFps, config.maxFps);

  // Determine if adjustment is needed
  let newFps = newState.currentFps;
  let adjusted = false;
  let reason: FpsAdjustmentDiagnostics['reason'] | undefined;

  if (avgCaptureMs > targetCaptureMs * config.decreaseThreshold) {
    // Captures are too slow - must decrease FPS
    // Apply smoothing toward ideal (which will be lower)
    newFps = newState.currentFps + config.smoothing * (idealFps - newState.currentFps);
    newFps = Math.max(config.minFps, Math.floor(newFps)); // Floor when decreasing
    adjusted = newFps !== newState.currentFps;
    reason = 'too_slow';
  } else if (avgCaptureMs < targetCaptureMs * config.increaseThreshold) {
    // Captures are fast - can increase FPS
    // Only increase if we're below max
    if (newState.currentFps < config.maxFps) {
      newFps = newState.currentFps + config.smoothing * (idealFps - newState.currentFps);
      newFps = Math.min(config.maxFps, Math.ceil(newFps)); // Ceil when increasing
      adjusted = newFps !== newState.currentFps;
      reason = 'too_fast';
    }
  }

  // Update state
  if (adjusted) {
    newState.currentFps = newFps;
    newState.framesSinceAdjustment = 0;
    newState.lastAdjustment = newFps > state.currentFps ? 'up' : 'down';
    newState.lastAdjustmentTime = Date.now();
  }

  const result: FpsAdjustmentResult = {
    state: newState,
    adjusted,
    newFps: Math.round(newState.currentFps),
    intervalMs: Math.floor(1000 / newState.currentFps),
  };

  if (adjusted && reason) {
    result.diagnostics = {
      previousFps: Math.round(state.currentFps),
      avgCaptureMs: Math.round(avgCaptureMs),
      targetCaptureMs: Math.round(targetCaptureMs),
      idealFps: Math.round(idealFps),
      reason,
    };
  }

  return result;
}

/**
 * Handle a capture timeout (screenshot took too long).
 *
 * Timeouts are treated as a severe signal - we immediately reduce FPS
 * without waiting for the adjustment interval.
 *
 * @param state - Current controller state
 * @param timeoutMs - The timeout threshold that was exceeded
 * @param config - Controller configuration
 * @returns Updated state with reduced FPS
 */
export function handleTimeout(
  state: FpsControllerState,
  timeoutMs: number,
  config: FpsControllerConfig = DEFAULT_FPS_CONFIG
): FpsAdjustmentResult {
  // Clone state
  const newState: FpsControllerState = {
    ...state,
    recentCaptures: [...state.recentCaptures],
  };

  // Add timeout as a capture time (it's at least this slow)
  newState.recentCaptures.push(timeoutMs);
  if (newState.recentCaptures.length > MAX_CAPTURE_BUFFER_SIZE) {
    newState.recentCaptures.shift();
  }

  newState.totalFrames++;

  // Immediate FPS reduction on timeout - drop by 2 FPS or 25%, whichever is larger
  const reduction = Math.max(2, Math.floor(newState.currentFps * 0.25));
  const newFps = Math.max(config.minFps, newState.currentFps - reduction);
  const adjusted = newFps !== newState.currentFps;

  if (adjusted) {
    newState.currentFps = newFps;
    newState.framesSinceAdjustment = 0;
    newState.lastAdjustment = 'down';
    newState.lastAdjustmentTime = Date.now();
  }

  const result: FpsAdjustmentResult = {
    state: newState,
    adjusted,
    newFps: Math.round(newState.currentFps),
    intervalMs: Math.floor(1000 / newState.currentFps),
  };

  if (adjusted) {
    result.diagnostics = {
      previousFps: Math.round(state.currentFps),
      avgCaptureMs: timeoutMs,
      targetCaptureMs: Math.round((1000 / state.currentFps) * config.targetUtilization),
      idealFps: Math.round(newFps),
      reason: 'too_slow',
    };
  }

  return result;
}

/**
 * Get current frame interval in milliseconds.
 * Convenience function for frame timing.
 */
export function getIntervalMs(state: FpsControllerState): number {
  return Math.floor(1000 / state.currentFps);
}

/**
 * Get current effective FPS (rounded).
 */
export function getCurrentFps(state: FpsControllerState): number {
  return Math.round(state.currentFps);
}

/**
 * Create a new controller with initial state and config.
 * Factory function for cleaner API.
 */
export function createFpsController(
  initialFps: number,
  config: Partial<FpsControllerConfig> = {}
): { state: FpsControllerState; config: FpsControllerConfig } {
  const mergedConfig: FpsControllerConfig = {
    ...DEFAULT_FPS_CONFIG,
    ...config,
    maxFps: config.maxFps ?? Math.max(DEFAULT_FPS_CONFIG.maxFps, initialFps),
  };

  // Clamp initial FPS to valid range
  const clampedInitialFps = clamp(initialFps, mergedConfig.minFps, mergedConfig.maxFps);

  return {
    state: createInitialState(clampedInitialFps),
    config: mergedConfig,
  };
}

// --- Helper functions ---

function average(arr: number[]): number {
  if (arr.length === 0) return 0;
  return arr.reduce((sum, val) => sum + val, 0) / arr.length;
}

function clamp(value: number, min: number, max: number): number {
  return Math.max(min, Math.min(max, value));
}

// Re-export types for convenience
export {
  FpsControllerConfig,
  FpsControllerState,
  FpsAdjustmentResult,
  FpsAdjustmentDiagnostics,
  DEFAULT_FPS_CONFIG,
  createInitialState,
};
