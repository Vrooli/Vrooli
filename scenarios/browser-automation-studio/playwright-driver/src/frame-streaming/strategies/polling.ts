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

import type { Page } from 'rebrowser-playwright';
import { encodeFrame, sameFrameSource, type FrameSource } from '../frame';
import { logger, scopedLog, LogContext, metrics } from '../../utils';
import {
  createFpsController,
  processFrame as processFpsFrame,
  handleTimeout as handleFpsTimeout,
  getIntervalMs,
  type FpsControllerState,
  type FpsControllerConfig,
} from '../../fps';
import { SCREENSHOT_TIMEOUT_MS, MAX_FRAME_FAILURES, MAX_QUEUED_FRAME_BYTES } from '../types';
import type {
  FrameStreamingStrategy,
  StreamingStrategyConfig,
  StreamingHandle,
  WebSocketProvider,
  FrameStatsReporter,
  PageProvider,
} from './interface';

/** FPS logging interval (every N frames) */
const FPS_LOG_INTERVAL = 30;

/**
 * Polling-based frame streaming strategy.
 *
 * Captures screenshots at regular intervals with adaptive FPS control.
 * Falls back to this when CDP screencast is not available.
 */
export class PollingStrategy implements FrameStreamingStrategy {
  readonly name = 'polling';

  isSupported(_page: Page): Promise<boolean> {
    // Polling works with any browser
    return Promise.resolve(true);
  }

  start(
    pageProvider: PageProvider,
    config: StreamingStrategyConfig,
    wsProvider: WebSocketProvider,
    statsReporter: FrameStatsReporter
  ): Promise<StreamingHandle> {
    const { sessionId, quality, targetFps, scale } = config;

    // Initialize FPS controller
    const { state: initialFpsState, config: fpsConfig } = createFpsController(targetFps, {
      minFps: Math.min(2, targetFps),
      maxFps: targetFps,
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
    let lastDelivered: {
      page: Page;
      source: FrameSource;
      socket: ReturnType<WebSocketProvider['getWebSocket']>;
      buffer: Buffer;
    } | undefined;
    let currentQuality = quality;
    let captureRevision = 0;
    const abortController = new AbortController();

    // Start the capture loop
    const captureLoop = async (): Promise<void> => {
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

          // Get current page (may have changed due to tab switch)
          const page = pageProvider();
          const ws = wsProvider.getWebSocket();
          const source = config.sourceForPage(page);
          if (!source) {
            await sleepUntilNextFrame(loopStart, currentIntervalMs, abortController.signal);
            continue;
          }

          // Capture frame with timing
          const capturedAt = Date.now();
          const revision = captureRevision;
          const captureStart = performance.now();
          const buffer = await captureFrameBuffer(page, currentQuality, scale);
          const captureTime = performance.now() - captureStart;

          if (!isActive) return;
          if (revision !== captureRevision || pageProvider() !== page || !sameFrameSource(config.sourceForPage(page),source) || wsProvider.getWebSocket() !== ws || !wsProvider.isReady()) {
            await sleepUntilNextFrame(loopStart, currentIntervalMs, abortController.signal);
            continue;
          }

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
          const isUnchanged = lastDelivered?.page === page && lastDelivered.socket === ws
            && sameFrameSource(lastDelivered.source,source)
            && buffer.equals(lastDelivered.buffer);
          const compareTime = performance.now() - compareStart;

          if (isUnchanged) {
            statsReporter.onFrameSkipped('unchanged');
            metrics.frameSkipCount.inc({ session_id: sessionId, reason: 'unchanged' });
            await sleepUntilNextFrame(loopStart, currentIntervalMs, abortController.signal);
            continue;
          }

          // Only successful delivery advances this viewer's deduplication state.
          if (ws && ws.readyState === 1) {
            const wsSendStart = performance.now();

            const frameToSend = encodeFrame(source,buffer,capturedAt,config.includePerfHeaders ? {
                frame_id: `${sessionId}-${frameCount + 1}`,
                capture_ms: captureTime,
                compare_ms: compareTime,
                ws_send_ms: 0, // Will be updated by API
                frame_bytes: buffer.length,
            } : undefined);

            if ((ws.bufferedAmount ?? 0) + frameToSend.length > MAX_QUEUED_FRAME_BYTES) {
              statsReporter.onFrameSkipped('ws_backpressure');
              await sleepUntilNextFrame(loopStart, getIntervalMs(fpsState), abortController.signal);
              continue;
            }

            ws.send(frameToSend);
            lastDelivered = { page, source, socket: ws, buffer };

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

    const handle: StreamingHandle = {
      getFrameCount: () => frameCount,
      isActive: () => isActive,

      updateQuality: (quality: number) => {
        currentQuality = Math.min(Math.max(quality, 1), 100);
        captureRevision++;
      },

      updatePerfMode: (enabled: boolean) => {
        config.includePerfHeaders = enabled;
        lastDelivered = undefined;
      },

      updateTargetFps: (fps: number) => {
        const newFps = Math.min(Math.max(fps, 1), 60);
        if (newFps !== currentFpsConfig.maxFps) {
          currentFpsConfig = {
            ...currentFpsConfig,
            minFps: Math.min(2, newFps),
            maxFps: newFps,
          };
          fpsState = { ...fpsState, currentFps: Math.min(fpsState.currentFps, newFps) };
        }
      },

      stop: async () => {
        if (!isActive) return loopPromise;
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
    return Promise.resolve(handle);
  }
}

/** Capture through the SDK owner, which bounds and disposes screenshot work. */
async function captureFrameBuffer(
  page: Page,
  quality: number,
  scale: 'css' | 'device'
): Promise<Buffer | null> {
  try {
    return await page.screenshot({
      type: 'jpeg',
      quality,
      scale,
      caret: 'initial',
      timeout: SCREENSHOT_TIMEOUT_MS,
    });
  } catch {
    return null;
  }
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
function sleep(ms: number, signal: AbortSignal): Promise<void> {
  return new Promise((resolve) => {
    if (signal.aborted) { resolve(); return; }
    const finish = () => {
      clearTimeout(timer);
      signal.removeEventListener('abort', finish);
      resolve();
    };
    const timer = setTimeout(finish, ms);
    signal.addEventListener('abort', finish, { once: true });
  });
}

/**
 * Create a polling strategy instance.
 */
export function createPollingStrategy(): PollingStrategy {
  return new PollingStrategy();
}
