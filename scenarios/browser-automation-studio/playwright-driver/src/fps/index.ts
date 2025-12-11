/**
 * FPS Controller Module
 *
 * Adaptive frame rate control for browser screenshot streaming.
 *
 * @example
 * ```typescript
 * import {
 *   createFpsController,
 *   processFrame,
 *   handleTimeout,
 *   getIntervalMs,
 * } from './fps';
 *
 * // Initialize
 * const { state, config } = createFpsController(15, { maxFps: 30 });
 * let fpsState = state;
 *
 * // In your frame loop:
 * const captureStart = performance.now();
 * const buffer = await page.screenshot(...);
 * const captureTime = performance.now() - captureStart;
 *
 * if (buffer === null) {
 *   // Timeout occurred
 *   const result = handleTimeout(fpsState, 200, config);
 *   fpsState = result.state;
 * } else {
 *   // Normal frame
 *   const result = processFrame(fpsState, captureTime, config);
 *   fpsState = result.state;
 *
 *   if (result.adjusted && result.diagnostics) {
 *     console.log(`FPS: ${result.diagnostics.previousFps} → ${result.newFps}`);
 *   }
 * }
 *
 * // Use for next frame timing
 * const sleepMs = getIntervalMs(fpsState) - elapsed;
 * ```
 */

export {
  // Core functions
  processFrame,
  handleTimeout,
  createFpsController,
  getIntervalMs,
  getCurrentFps,
  // Types
  FpsControllerConfig,
  FpsControllerState,
  FpsAdjustmentResult,
  FpsAdjustmentDiagnostics,
  // Constants
  DEFAULT_FPS_CONFIG,
  // State factory
  createInitialState,
} from './controller';
