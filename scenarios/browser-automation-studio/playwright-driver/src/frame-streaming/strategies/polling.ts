/**
 * Polling Strategy
 *
 * Pull-based frame streaming using periodic screenshot capture.
 * This is the fallback strategy when CDP screencast is not available.
 *
 * Features:
 * - Adaptive FPS based on capture time (target 70% utilization)
 * - Frame deduplication via byte comparison
 * - Works with all browser types
 *
 * Expected performance: 10-15 FPS (limited by screenshot capture time)
 *
 * @module frame-streaming/strategies/polling
 */

import type { Page, CDPSession } from 'playwright';
import { logger, scopedLog, LogContext, metrics } from '../../utils';
import {
  createFpsController,
  processFrame as processFpsFrame,
  handleTimeout as handleFpsTimeout,
  getIntervalMs,
  getCurrentFps,
  type FpsControllerState,
  type FpsControllerConfig,
} from '../../fps';
import { SCREENSHOT_TIMEOUT_MS, MAX_FRAME_FAILURES } from '../types';
import type {
  FrameStreamingStrategy,
  StreamingStrategyConfig,
  StreamingHandle,
  WebSocketProvider,
  FrameStatsReporter,
} from './interface';

/** FPS logging interval (every N frames) */
const FPS_LOG_INTERVAL = 30;

/**
 * CDP session cache for frame capture.
 * Reusing the CDP session avoids the overhead of creating a new one per frame.
 * WeakMap ensures sessions are cleaned up when pages are garbage collected.
 */
const cdpSessionCache = new WeakMap<Page, CDPSession>();

/**
 * Polling-based frame streaming strategy.
 *
 * Captures screenshots at regular intervals with adaptive FPS control.
 * Falls back to this when CDP screencast is not available.
 */
export class PollingStrategy implements FrameStreamingStrategy {
  readonly name = 'polling';

  async isSupported(_page: Page): Promise<boolean> {
    // Polling works with any browser
    return true;
  }

  async start(
    page: Page,
    config: StreamingStrategyConfig,
    wsProvider: WebSocketProvider,
    statsReporter: FrameStatsReporter
  ): Promise<StreamingHandle> {
    const { sessionId, quality, targetFps, scale } = config;

    // Initialize FPS controller
    const { state: initialFpsState, config: fpsConfig } = createFpsController(targetFps, {
      minFps: 2,
      maxFps: Math.min(60, targetFps * 2),
      targetUtilization: 0.7,
      smoothing: 0.25,
      adjustmentInterval: 3,
    });

    // Mutable state
    let fpsState: FpsControllerState = initialFpsState;
    let currentFpsConfig: FpsControllerConfig = fpsConfig;
    let isActive = true;
    let frameCount = 0;
    let consecutiveFailures = 0;
    let lastFrameBuffer: Buffer | null = null;
    let currentQuality = quality;
    let currentTargetFps = targetFps;
    const abortController = new AbortController();

    // Start the capture loop
    const captureLoop = async () => {
      while (isActive && !abortController.signal.aborted) {
        const loopStart = performance.now();
        const currentIntervalMs = getIntervalMs(fpsState);

        try {
          // Skip if WebSocket not ready
          if (!wsProvider.isReady()) {
            statsReporter.onFrameSkipped('ws_not_ready');
            await sleepUntilNextFrame(loopStart, currentIntervalMs, abortController.signal);
            continue;
          }

          // Skip if too many consecutive failures (circuit breaker)
          if (consecutiveFailures >= MAX_FRAME_FAILURES) {
            await sleep(currentIntervalMs * 5, abortController.signal);
            consecutiveFailures = 0;
            continue;
          }

          // Capture frame with timing
          const captureStart = performance.now();
          const buffer = await captureFrameBuffer(page, currentQuality, scale);
          const captureTime = performance.now() - captureStart;

          if (!buffer) {
            // Timeout hit - adjust FPS
            const timeoutResult = handleFpsTimeout(fpsState, SCREENSHOT_TIMEOUT_MS, currentFpsConfig);
            fpsState = timeoutResult.state;

            statsReporter.onFrameSkipped('timeout');
            metrics.frameSkipCount.inc({ session_id: sessionId, reason: 'timeout' });

            if (timeoutResult.adjusted && timeoutResult.diagnostics) {
              logger.debug(scopedLog(LogContext.RECORDING, 'FPS reduced (capture timeout)'), {
                sessionId,
                previousFps: timeoutResult.diagnostics.previousFps,
                currentFps: timeoutResult.newFps,
              });
            }

            await sleepUntilNextFrame(loopStart, currentIntervalMs, abortController.signal);
            continue;
          }

          // Update FPS controller with capture time
          const fpsResult = processFpsFrame(fpsState, captureTime, currentFpsConfig);
          fpsState = fpsResult.state;

          // Log FPS changes periodically
          if (fpsResult.adjusted && fpsResult.diagnostics && frameCount % FPS_LOG_INTERVAL === 0) {
            const direction = fpsResult.diagnostics.reason === 'too_fast' ? 'increased' : 'reduced';
            logger.debug(scopedLog(LogContext.RECORDING, `FPS ${direction}`), {
              sessionId,
              previousFps: fpsResult.diagnostics.previousFps,
              currentFps: fpsResult.newFps,
              avgCaptureMs: fpsResult.diagnostics.avgCaptureMs,
              targetCaptureMs: fpsResult.diagnostics.targetCaptureMs,
              reason: fpsResult.diagnostics.reason,
            });
          }

          // Compare with last frame
          const compareStart = performance.now();
          const isUnchanged = isFrameUnchanged(buffer, lastFrameBuffer);
          const compareTime = performance.now() - compareStart;

          if (isUnchanged) {
            statsReporter.onFrameSkipped('unchanged');
            metrics.frameSkipCount.inc({ session_id: sessionId, reason: 'unchanged' });
            await sleepUntilNextFrame(loopStart, currentIntervalMs, abortController.signal);
            continue;
          }

          // Update last frame
          lastFrameBuffer = buffer;

          // Send frame
          const ws = wsProvider.getWebSocket();
          if (ws && ws.readyState === 1) {
            const wsSendStart = performance.now();
            ws.send(buffer);
            const wsSendTime = performance.now() - wsSendStart;

            frameCount++;
            consecutiveFailures = 0;

            // Report stats
            statsReporter.onFrameSent({
              captureMs: captureTime,
              compareMs: compareTime,
              wsSendMs: wsSendTime,
              frameBytes: buffer.length,
            });

            // Record metrics
            metrics.frameCaptureLatency.observe({ session_id: sessionId }, captureTime);
            metrics.frameE2ELatency.observe({ session_id: sessionId }, captureTime + compareTime + wsSendTime);
          }
        } catch (err) {
          if (abortController.signal.aborted) {
            return; // Normal shutdown
          }

          consecutiveFailures++;
          const message = err instanceof Error ? err.message : String(err);

          logger.warn(scopedLog(LogContext.RECORDING, 'polling frame error'), {
            sessionId,
            error: message,
            consecutiveFailures,
          });
        }

        await sleepUntilNextFrame(loopStart, getIntervalMs(fpsState), abortController.signal);
      }
    };

    // Start the loop (don't await - runs in background)
    const loopPromise = captureLoop().catch((err) => {
      if (!abortController.signal.aborted) {
        logger.error(scopedLog(LogContext.RECORDING, 'polling loop crashed'), {
          sessionId,
          error: err instanceof Error ? err.message : String(err),
        });
      }
    });

    logger.info(scopedLog(LogContext.RECORDING, 'frame streaming started (polling mode)'), {
      sessionId,
      targetFps,
      quality,
      scale,
    });

    return {
      getFrameCount: () => frameCount,
      isActive: () => isActive,

      updateQuality: (quality: number) => {
        currentQuality = Math.min(Math.max(quality, 1), 100);
        logger.debug(scopedLog(LogContext.RECORDING, 'polling quality updated'), {
          sessionId,
          quality: currentQuality,
        });
      },

      updateTargetFps: (fps: number) => {
        const newFps = Math.min(Math.max(fps, 1), 60);
        if (newFps !== currentTargetFps) {
          currentTargetFps = newFps;
          currentFpsConfig = {
            ...currentFpsConfig,
            maxFps: Math.min(60, newFps * 2),
          };
          logger.debug(scopedLog(LogContext.RECORDING, 'polling target FPS updated'), {
            sessionId,
            targetFps: currentTargetFps,
            currentFps: getCurrentFps(fpsState),
          });
        }
      },

      stop: async () => {
        if (!isActive) return;
        isActive = false;
        abortController.abort();

        // Wait for loop to finish
        await loopPromise;

        logger.info(scopedLog(LogContext.RECORDING, 'polling streaming stopped'), {
          sessionId,
          totalFrames: frameCount,
        });
      },
    };
  }
}

/**
 * Get or create a CDP session for a page.
 * CDP sessions are cached to avoid per-frame creation overhead.
 */
async function getCDPSession(page: Page): Promise<CDPSession> {
  let session = cdpSessionCache.get(page);
  if (!session) {
    session = await page.context().newCDPSession(page);
    cdpSessionCache.set(page, session);
  }
  return session;
}

/**
 * Capture a frame as raw JPEG buffer using CDP directly.
 * Uses Page.captureScreenshot with optimizeForSpeed for faster encoding.
 */
async function captureFrameBuffer(
  page: Page,
  quality: number,
  scale: 'css' | 'device'
): Promise<Buffer | null> {
  try {
    const viewport = page.viewportSize();
    if (!viewport) {
      // Fallback to Playwright's screenshot if viewport not available
      return await page.screenshot({
        type: 'jpeg',
        quality,
        timeout: SCREENSHOT_TIMEOUT_MS,
        scale,
      });
    }

    const cdp = await getCDPSession(page);

    const result = await Promise.race([
      cdp.send('Page.captureScreenshot', {
        format: 'jpeg',
        quality,
        optimizeForSpeed: true,
        captureBeyondViewport: false,
        fromSurface: true,
      }),
      new Promise<null>((resolve) => setTimeout(() => resolve(null), SCREENSHOT_TIMEOUT_MS)),
    ]);

    if (!result) {
      return null; // Timeout
    }

    return Buffer.from(result.data, 'base64');
  } catch {
    return null;
  }
}

/**
 * Check if frame is unchanged from last frame.
 */
function isFrameUnchanged(buffer: Buffer, lastBuffer: Buffer | null): boolean {
  if (!lastBuffer) return false;
  if (buffer.length !== lastBuffer.length) return false;
  return buffer.equals(lastBuffer);
}

/**
 * Sleep until the next frame capture time.
 */
async function sleepUntilNextFrame(
  loopStart: number,
  intervalMs: number,
  signal: AbortSignal
): Promise<void> {
  const elapsed = performance.now() - loopStart;
  const sleepTime = Math.max(0, intervalMs - elapsed);
  if (sleepTime > 0) {
    await sleep(sleepTime, signal);
  }
}

/**
 * Sleep with abort signal support.
 */
function sleep(ms: number, signal?: AbortSignal): Promise<void> {
  return new Promise((resolve, reject) => {
    const timeoutId = setTimeout(resolve, ms);
    if (signal) {
      signal.addEventListener('abort', () => {
        clearTimeout(timeoutId);
        reject(new Error('Aborted'));
      }, { once: true });
    }
  });
}

/**
 * Create a polling strategy instance.
 */
export function createPollingStrategy(): PollingStrategy {
  return new PollingStrategy();
}
