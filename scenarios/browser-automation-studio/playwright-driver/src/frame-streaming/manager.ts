/**
 * Frame Streaming Manager
 *
 * Manages frame streaming for recording sessions. Supports two modes:
 * - CDP Screencast: Push-based streaming from Chrome (preferred, 30-60 FPS)
 * - Polling: Pull-based screenshot capture (fallback, adaptive FPS)
 *
 * Uses WebSocket for efficient binary frame delivery to the API.
 *
 * @module frame-streaming/manager
 */

import type { Page, CDPSession } from 'playwright';
import WebSocket from 'ws';
import { logger, metrics, scopedLog, LogContext } from '../utils';
import { loadConfig } from '../config';
import { PerfCollector } from '../performance';
import {
  createFpsController,
  processFrame as processFpsFrame,
  handleTimeout as handleFpsTimeout,
  getIntervalMs,
  getCurrentFps,
} from '../fps';
import {
  startScreencastStreaming,
  type ScreencastState,
} from '../routes/record-mode/screencast-streaming';
import type {
  FrameStreamState,
  FrameStreamOptions,
  FrameStreamUpdateOptions,
  FrameStreamSettings,
  FrameWebSocket,
  SessionProvider,
} from './types';
import {
  MAX_FRAME_FAILURES,
  WS_RECONNECT_DELAY_MS,
  FPS_LOG_INTERVAL,
  SCREENSHOT_TIMEOUT_MS,
} from './types';

// =============================================================================
// Module State
// =============================================================================

/** Per-session frame streaming state */
const frameStreamStates = new Map<string, FrameStreamState>();

/**
 * CDP session cache for frame capture.
 * Reusing the CDP session avoids the overhead of creating a new one per frame.
 * WeakMap ensures sessions are cleaned up when pages are garbage collected.
 */
const cdpSessionCache = new WeakMap<Page, CDPSession>();

// Type cast for WebSocket constructor
const WS: new (url: string) => FrameWebSocket = WebSocket as unknown as new (url: string) => FrameWebSocket;

// =============================================================================
// Public API
// =============================================================================

/**
 * Start frame streaming for a recording session.
 *
 * Uses CDP screencast by default for push-based frame delivery (30-60 FPS).
 * Falls back to polling-based screenshot capture if screencast fails.
 *
 * WebSocket is used for efficient binary frame delivery to the API.
 *
 * @param sessionId - Session ID
 * @param sessionProvider - Provider to get session/page data
 * @param options - Streaming options
 */
export function startFrameStreaming(
  sessionId: string,
  sessionProvider: SessionProvider,
  options: FrameStreamOptions
): void {
  // Stop any existing stream for this session (fire and forget - don't block)
  void stopFrameStreaming(sessionId);

  const quality = options.quality ?? 65;
  const fps = Math.min(Math.max(options.fps ?? 30, 1), 60); // Clamp 1-60 FPS
  const scale = options.scale ?? 'css';
  const wsUrl = buildWebSocketUrl(options.callbackUrl, sessionId);

  // Load config for feature flags and performance settings
  const config = loadConfig();

  // Always create performance collector (it's cheap - just a ring buffer).
  const perfCollector = PerfCollector.fromConfig(sessionId, config, fps);

  // Initialize FPS controller (only used for polling fallback mode)
  const { state: fpsState, config: fpsConfig } = createFpsController(fps, {
    minFps: 2,
    maxFps: Math.min(60, fps * 2),
    targetUtilization: 0.7,
    smoothing: 0.25,
    adjustmentInterval: 3,
  });

  const state: FrameStreamState = {
    isStreaming: true,
    abortController: new AbortController(),
    lastFrameBuffer: null,
    quality,
    targetFps: fps,
    frameCount: 0,
    ws: null,
    wsUrl,
    consecutiveFailures: 0,
    wsReady: false,
    scale,
    perfCollector,
    includePerfHeaders: config.performance.enabled && config.performance.includeTimingHeaders,
    fpsState,
    fpsConfig,
    useScreencast: config.frameStreaming.useScreencast, // Feature flag from config
  };

  frameStreamStates.set(sessionId, state);

  // Connect WebSocket first (needed for both modes)
  connectFrameWebSocket(sessionId, state);

  // Start streaming based on mode
  if (state.useScreencast) {
    void startScreencastMode(sessionId, sessionProvider, state, config);
  } else {
    logger.info(scopedLog(LogContext.RECORDING, 'frame streaming started (polling mode)'), {
      sessionId,
      targetFps: fps,
      quality,
      scale,
      wsUrl,
    });
    void runFrameStreamLoop(sessionId, sessionProvider, state);
  }
}

/**
 * Stop frame streaming for a session.
 * Handles cleanup for both screencast and polling modes.
 *
 * @param sessionId - Session ID to stop streaming for
 */
export async function stopFrameStreaming(sessionId: string): Promise<void> {
  const state = frameStreamStates.get(sessionId);
  if (!state) return;

  state.isStreaming = false;
  state.abortController.abort();

  // Stop CDP screencast if active
  if (state.screencastHandle) {
    try {
      await state.screencastHandle.stop();
    } catch (error) {
      logger.debug(scopedLog(LogContext.RECORDING, 'screencast stop error (may already be stopped)'), {
        sessionId,
        error: error instanceof Error ? error.message : String(error),
      });
    }
    state.screencastHandle = undefined;
  }

  // Close WebSocket connection
  if (state.ws) {
    try {
      state.ws.close();
    } catch {
      // Ignore close errors
    }
    state.ws = null;
  }

  frameStreamStates.delete(sessionId);

  logger.info(scopedLog(LogContext.RECORDING, 'frame streaming stopped'), {
    sessionId,
    totalFrames: state.frameCount,
    mode: state.useScreencast ? 'screencast' : 'polling',
  });
}

/**
 * Update frame streaming settings for an active session.
 * Only updates quality and fps - scale changes require session restart.
 *
 * @param sessionId - Session ID
 * @param options - Settings to update
 * @returns true if settings were updated, false if no active stream
 */
export function updateFrameStreamSettings(
  sessionId: string,
  options: FrameStreamUpdateOptions
): boolean {
  const state = frameStreamStates.get(sessionId);
  if (!state || !state.isStreaming) {
    return false;
  }

  let changed = false;

  if (options.quality !== undefined) {
    const newQuality = Math.min(Math.max(options.quality, 1), 100);
    if (newQuality !== state.quality) {
      state.quality = newQuality;
      changed = true;
    }
  }

  if (options.fps !== undefined) {
    const newFps = Math.min(Math.max(options.fps, 1), 60);
    if (newFps !== state.targetFps) {
      state.targetFps = newFps;
      // Update FPS config with new max (allow up to 2x target, capped at 60)
      state.fpsConfig = {
        ...state.fpsConfig,
        maxFps: Math.min(60, newFps * 2),
      };
      changed = true;
    }
  }

  if (options.perfMode !== undefined && options.perfMode !== state.includePerfHeaders) {
    state.includePerfHeaders = options.perfMode;
    changed = true;
  }

  if (changed) {
    logger.info(scopedLog(LogContext.RECORDING, 'frame stream settings updated'), {
      sessionId,
      quality: state.quality,
      targetFps: state.targetFps,
      currentFps: getCurrentFps(state.fpsState),
      scale: state.scale,
      perfMode: state.includePerfHeaders,
    });
  }

  return changed;
}

/**
 * Get current frame streaming settings for a session.
 *
 * @param sessionId - Session ID
 * @returns Current settings or null if no active stream
 */
export function getFrameStreamSettings(sessionId: string): FrameStreamSettings | null {
  const state = frameStreamStates.get(sessionId);
  if (!state) {
    return null;
  }

  return {
    quality: state.quality,
    fps: state.targetFps,
    scale: state.scale,
    currentFps: getCurrentFps(state.fpsState),
    isStreaming: state.isStreaming,
    perfMode: state.includePerfHeaders,
  };
}

// =============================================================================
// Internal Functions - WebSocket
// =============================================================================

/**
 * Build WebSocket URL from HTTP callback URL.
 * Converts http://host:port/api/v1/recordings/live/{sessionId}/frame
 * to ws://host:port/ws/recording/{sessionId}/frames
 */
function buildWebSocketUrl(callbackUrl: string, sessionId: string): string {
  try {
    const url = new URL(callbackUrl);
    const protocol = url.protocol === 'https:' ? 'wss:' : 'ws:';
    return `${protocol}//${url.host}/ws/recording/${sessionId}/frames`;
  } catch {
    // Fallback: assume localhost API
    return `ws://127.0.0.1:8080/ws/recording/${sessionId}/frames`;
  }
}

/**
 * Connect WebSocket for frame streaming.
 */
function connectFrameWebSocket(sessionId: string, state: FrameStreamState): void {
  if (!state.isStreaming) return;

  try {
    const ws = new WS(state.wsUrl);
    state.ws = ws;

    ws.on('open', () => {
      state.wsReady = true;
      state.consecutiveFailures = 0;
      logger.info(scopedLog(LogContext.RECORDING, 'frame WebSocket connected'), { sessionId });
    });

    ws.on('close', () => {
      state.wsReady = false;
      logger.debug(scopedLog(LogContext.RECORDING, 'frame WebSocket closed'), { sessionId });

      // Reconnect if still streaming
      if (state.isStreaming) {
        setTimeout(() => connectFrameWebSocket(sessionId, state), WS_RECONNECT_DELAY_MS);
      }
    });

    ws.on('error', (err: Error) => {
      state.wsReady = false;
      logger.warn(scopedLog(LogContext.RECORDING, 'frame WebSocket error'), {
        sessionId,
        error: err.message,
      });
    });
  } catch (err) {
    logger.warn(scopedLog(LogContext.RECORDING, 'failed to create frame WebSocket'), {
      sessionId,
      error: err instanceof Error ? err.message : String(err),
    });

    // Retry connection
    if (state.isStreaming) {
      setTimeout(() => connectFrameWebSocket(sessionId, state), WS_RECONNECT_DELAY_MS);
    }
  }
}

// =============================================================================
// Internal Functions - Screencast Mode
// =============================================================================

/**
 * Start CDP screencast streaming mode.
 * Falls back to polling mode if screencast fails (and fallback is enabled).
 */
async function startScreencastMode(
  sessionId: string,
  sessionProvider: SessionProvider,
  state: FrameStreamState,
  config: ReturnType<typeof loadConfig>
): Promise<void> {
  try {
    const session = sessionProvider.getSession(sessionId);
    const viewport = session.page.viewportSize() ?? { width: 1280, height: 720 };

    // Wait briefly for WebSocket to connect
    await new Promise((resolve) => setTimeout(resolve, 100));

    // Create a state proxy that keeps screencast in sync with our state
    const screencastState: ScreencastState = {
      get ws() {
        return state.ws;
      },
      get wsReady() {
        return state.wsReady;
      },
      get isStreaming() {
        return state.isStreaming;
      },
      frameCount: state.frameCount,
    };

    // Define a setter to update frame count in our state
    Object.defineProperty(screencastState, 'frameCount', {
      get: () => state.frameCount,
      set: (value: number) => {
        state.frameCount = value;
      },
    });

    state.screencastHandle = await startScreencastStreaming(
      session.page,
      sessionId,
      screencastState,
      {
        format: 'jpeg',
        quality: state.quality,
        maxWidth: viewport.width,
        maxHeight: viewport.height,
        everyNthFrame: 1, // Every frame - Chrome handles change detection
      }
    );

    logger.info(scopedLog(LogContext.RECORDING, 'frame streaming started (CDP screencast)'), {
      sessionId,
      quality: state.quality,
      viewport,
      wsUrl: state.wsUrl,
      mode: 'screencast',
    });
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);

    // Screencast failed - check if we should fall back to polling
    if (config.frameStreaming.fallbackToPolling) {
      logger.warn(scopedLog(LogContext.RECORDING, 'CDP screencast failed, falling back to polling'), {
        sessionId,
        error: message,
        fallback: true,
      });

      state.useScreencast = false;
      void runFrameStreamLoop(sessionId, sessionProvider, state);
    } else {
      logger.error(scopedLog(LogContext.RECORDING, 'CDP screencast failed (no fallback)'), {
        sessionId,
        error: message,
        fallback: false,
        hint: 'Set FRAME_STREAMING_FALLBACK=true to enable polling fallback',
      });

      // Clean up state since we're not streaming
      state.isStreaming = false;
      frameStreamStates.delete(sessionId);
    }
  }
}

// =============================================================================
// Internal Functions - Polling Mode
// =============================================================================

/**
 * The main frame streaming loop.
 * Captures frames and sends via WebSocket as binary data.
 *
 * Uses target-utilization based adaptive FPS via the fps controller:
 * - Measures actual screenshot capture time (the bottleneck)
 * - Adjusts FPS to keep capture at ~70% of frame budget
 * - Simple, single-mechanism feedback loop
 */
async function runFrameStreamLoop(
  sessionId: string,
  sessionProvider: SessionProvider,
  state: FrameStreamState
): Promise<void> {
  while (state.isStreaming && !state.abortController.signal.aborted) {
    const loopStart = performance.now();

    // Get current interval from FPS controller
    const currentIntervalMs = getIntervalMs(state.fpsState);

    try {
      // Check if session still exists
      let session;
      try {
        session = sessionProvider.getSession(sessionId);
      } catch {
        // Session closed, stop streaming
        void stopFrameStreaming(sessionId);
        return;
      }

      // Skip if WebSocket not connected (readyState 1 = OPEN)
      if (!state.wsReady || !state.ws || state.ws.readyState !== 1) {
        await sleepUntilNextFrame(loopStart, currentIntervalMs, state.abortController.signal);
        continue;
      }

      // Skip if too many consecutive failures (circuit breaker)
      if (state.consecutiveFailures >= MAX_FRAME_FAILURES) {
        // Wait longer before retrying
        await sleep(currentIntervalMs * 5, state.abortController.signal);
        state.consecutiveFailures = 0; // Reset and retry
        continue;
      }

      // === CAPTURE FRAME WITH TIMING ===
      const captureStart = performance.now();
      const buffer = await captureFrameBuffer(session.page, state.quality, state.scale);
      const captureTime = performance.now() - captureStart;

      if (!buffer) {
        // Timeout hit - use FPS controller's timeout handler
        const timeoutResult = handleFpsTimeout(state.fpsState, SCREENSHOT_TIMEOUT_MS, state.fpsConfig);
        state.fpsState = timeoutResult.state;

        // Record timeout in perf metrics
        metrics.frameSkipCount.inc({ session_id: sessionId, reason: 'timeout' });

        if (timeoutResult.adjusted && timeoutResult.diagnostics) {
          logger.debug(scopedLog(LogContext.RECORDING, 'FPS reduced (capture timeout)'), {
            sessionId,
            previousFps: timeoutResult.diagnostics.previousFps,
            currentFps: timeoutResult.newFps,
          });
        }

        await sleepUntilNextFrame(loopStart, currentIntervalMs, state.abortController.signal);
        continue;
      }

      // === UPDATE FPS CONTROLLER WITH CAPTURE TIME ===
      const fpsResult = processFpsFrame(state.fpsState, captureTime, state.fpsConfig);
      state.fpsState = fpsResult.state;
      state.frameCount++;

      // Log FPS changes periodically
      if (fpsResult.adjusted && fpsResult.diagnostics && state.frameCount % FPS_LOG_INTERVAL === 0) {
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

      // === COMPARE WITH TIMING ===
      const compareStart = performance.now();
      const isUnchanged = isFrameUnchanged(buffer, state);
      const compareTime = performance.now() - compareStart;

      // Skip if frame content unchanged (fast buffer comparison)
      if (isUnchanged) {
        // Record skipped frame in perf collector
        state.perfCollector.recordSkipped(captureTime, compareTime);
        metrics.frameSkipCount.inc({ session_id: sessionId, reason: 'unchanged' });
        await sleepUntilNextFrame(loopStart, currentIntervalMs, state.abortController.signal);
        continue;
      }

      // Update last frame state
      state.lastFrameBuffer = buffer;

      // === SEND WITH TIMING ===
      const wsSendStart = performance.now();

      // Send frame with or without perf header
      if (state.includePerfHeaders) {
        // Build and prepend binary header with timing data
        const header = state.perfCollector.buildFrameHeader(
          captureTime,
          compareTime,
          0, // wsSendMs will be measured after send
          buffer.length
        );
        state.ws.send(Buffer.concat([header, buffer]));
      } else {
        // Send raw binary frame (no JSON, no base64!)
        state.ws.send(buffer);
      }

      const wsSendTime = performance.now() - wsSendStart;
      state.consecutiveFailures = 0;

      // === RECORD PERFORMANCE DATA ===
      state.perfCollector.recordFrame({
        captureMs: captureTime,
        compareMs: compareTime,
        wsSendMs: wsSendTime,
        frameBytes: buffer.length,
        skipped: false,
      });

      // Record to Prometheus metrics
      metrics.frameCaptureLatency.observe({ session_id: sessionId }, captureTime);
      metrics.frameE2ELatency.observe({ session_id: sessionId }, captureTime + compareTime + wsSendTime);

      // Log summary periodically
      if (state.perfCollector.shouldLogSummary()) {
        const stats = state.perfCollector.getAggregatedStats();
        logger.info(scopedLog(LogContext.RECORDING, 'frame perf summary'), {
          session_id: sessionId,
          frame_count: stats.frame_count,
          skipped_count: stats.skipped_count,
          capture_p50_ms: stats.capture_p50_ms,
          capture_p90_ms: stats.capture_p90_ms,
          e2e_p50_ms: stats.e2e_p50_ms,
          e2e_p90_ms: stats.e2e_p90_ms,
          actual_fps: getCurrentFps(state.fpsState),
          target_fps: state.targetFps,
          bottleneck: stats.primary_bottleneck,
        });
      }

    } catch (err) {
      if (state.abortController.signal.aborted) {
        return; // Normal shutdown
      }

      state.consecutiveFailures++;
      const message = err instanceof Error ? err.message : String(err);

      logger.warn(scopedLog(LogContext.RECORDING, 'frame streaming error'), {
        sessionId,
        error: message,
        consecutiveFailures: state.consecutiveFailures,
      });
    }

    await sleepUntilNextFrame(loopStart, getIntervalMs(state.fpsState), state.abortController.signal);
  }
}

// =============================================================================
// Internal Functions - Frame Capture
// =============================================================================

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
 *
 * Only captures the visible viewport (not full page) to reduce bandwidth and improve performance.
 * No base64 encoding overhead on our side - CDP returns base64 but we decode it once.
 *
 * Performance notes:
 * - optimizeForSpeed uses faster zlib settings (q1/RLE) for ~2x speedup
 * - CDP clip uses device pixels, so we scale viewport dimensions accordingly
 * - Using 'css' scale (1x) produces smaller images, 'device' uses devicePixelRatio
 *
 * @param page - Playwright page to capture
 * @param quality - JPEG quality 1-100
 * @param scale - 'css' captures at 1x logical pixels (smaller, faster),
 *                'device' captures at devicePixelRatio (sharper on HiDPI, but 4x larger on 2x displays)
 */
async function captureFrameBuffer(
  page: Page,
  quality: number,
  scale: 'css' | 'device' = 'css'
): Promise<Buffer | null> {
  try {
    // Get viewport dimensions for clipping - only capture what's visible
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

    // Get or create CDP session (cached for performance)
    const cdp = await getCDPSession(page);

    // Use CDP Page.captureScreenshot directly with optimizeForSpeed
    // This provides ~2x speedup on encoding by using faster compression settings
    //
    // Key settings:
    // - captureBeyondViewport: false - only capture visible viewport (not full page)
    // - fromSurface: true - capture from compositor surface (default, faster)
    // - optimizeForSpeed: true - use faster encoding (zlib q1/RLE)
    //
    // Note: We don't use 'clip' because it captures a fixed page region, not the
    // current viewport. With captureBeyondViewport: false, CDP captures exactly
    // what's visible in the viewport, which is what we want for live streaming.
    const result = await Promise.race([
      cdp.send('Page.captureScreenshot', {
        format: 'jpeg',
        quality,
        optimizeForSpeed: true, // Key optimization: faster encoding
        captureBeyondViewport: false, // Only capture visible viewport
        fromSurface: true, // Capture from compositor (faster)
      }),
      // Timeout handling
      new Promise<null>((resolve) => setTimeout(() => resolve(null), SCREENSHOT_TIMEOUT_MS)),
    ]);

    if (!result) {
      return null; // Timeout
    }

    // CDP returns base64, decode to Buffer
    return Buffer.from(result.data, 'base64');
  } catch {
    // CDP error or other failure - skip this frame
    return null;
  }
}

/**
 * Check if frame is unchanged from last frame.
 * Uses fast buffer length check before expensive byte comparison.
 */
function isFrameUnchanged(buffer: Buffer, state: FrameStreamState): boolean {
  if (!state.lastFrameBuffer) {
    return false;
  }

  // Fast path: different length means different content
  if (buffer.length !== state.lastFrameBuffer.length) {
    return false;
  }

  // Same length: do byte comparison (still faster than MD5 for most cases)
  return buffer.equals(state.lastFrameBuffer);
}

// =============================================================================
// Internal Functions - Timing Utilities
// =============================================================================

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
