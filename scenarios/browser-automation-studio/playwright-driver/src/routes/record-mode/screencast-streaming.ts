/**
 * CDP Screencast Streaming
 *
 * Push-based frame streaming using Chrome's native Page.startScreencast API.
 * This replaces the polling-based screenshot approach for better performance.
 *
 * Key differences from polling:
 * - Chrome controls frame timing (compositor-native)
 * - Built-in change detection (no frames sent if nothing changed)
 * - Must acknowledge each frame via screencastFrameAck
 *
 * Expected performance improvement: 10-15 FPS -> 30-60 FPS
 */

import type { CDPSession, Page } from 'playwright';
import { logger, scopedLog, LogContext, metrics } from '../../utils';

/** Configuration for CDP screencast */
export interface ScreencastConfig {
  /** Image format */
  format: 'jpeg' | 'png';
  /** Compression quality 0-100 (jpeg only) */
  quality: number;
  /** Maximum width - frames scaled down if larger */
  maxWidth: number;
  /** Maximum height - frames scaled down if larger */
  maxHeight: number;
  /** Send every n-th frame (1 = all frames) */
  everyNthFrame: number;
}

/** Frame event from CDP */
interface ScreencastFrameEvent {
  /** Base64-encoded image data */
  data: string;
  /** Frame metadata */
  metadata: {
    offsetTop: number;
    pageScaleFactor: number;
    deviceWidth: number;
    deviceHeight: number;
    scrollOffsetX: number;
    scrollOffsetY: number;
    timestamp?: number;
  };
  /** Frame number - MUST be used in ACK */
  sessionId: number;
}

/** Visibility change event from CDP */
interface ScreencastVisibilityEvent {
  visible: boolean;
}

/** WebSocket interface (matches ws module) */
interface FrameWebSocket {
  readyState: number;
  send(data: Buffer): void;
}

/** Result of starting screencast */
export interface ScreencastHandle {
  /** CDP session (for advanced usage) */
  cdpSession: CDPSession;
  /** Stop the screencast and cleanup */
  stop: () => Promise<void>;
  /** Get current frame count */
  getFrameCount: () => number;
  /** Whether screencast is currently active */
  isActive: () => boolean;
}

/** State passed to screencast for WebSocket access */
export interface ScreencastState {
  ws: FrameWebSocket | null;
  wsReady: boolean;
  frameCount: number;
  isStreaming: boolean;
}

/** ACK timeout - prevents hanging if CDP is slow */
const ACK_TIMEOUT_MS = 1000;

/** Max consecutive ACK failures before logging error */
const MAX_ACK_FAILURES = 5;

/**
 * Start CDP screencast streaming.
 *
 * Frames are pushed from Chrome's compositor and forwarded to the WebSocket.
 * Each frame MUST be acknowledged or Chrome stops sending.
 *
 * @param page - Playwright page to capture
 * @param sessionId - Session ID for logging
 * @param state - Shared state object for WebSocket access
 * @param config - Screencast configuration
 * @returns Handle to stop screencast and get stats
 */
export async function startScreencastStreaming(
  page: Page,
  sessionId: string,
  state: ScreencastState,
  config: ScreencastConfig
): Promise<ScreencastHandle> {
  const cdpSession = await page.context().newCDPSession(page);
  let isActive = true;
  let frameCount = 0;
  let ackFailures = 0;

  // Handle incoming frames
  cdpSession.on('Page.screencastFrame', async (event: ScreencastFrameEvent) => {
    if (!isActive) return;

    const frameStart = performance.now();

    try {
      // 1. Decode base64 to buffer
      // Note: This adds ~1-2ms overhead, but Chrome doesn't support binary screencast
      const decodeStart = performance.now();
      const buffer = Buffer.from(event.data, 'base64');
      const decodeMs = performance.now() - decodeStart;

      // 2. Forward to WebSocket if connected
      if (state.ws && state.ws.readyState === 1 && state.wsReady && state.isStreaming) {
        const sendStart = performance.now();
        state.ws.send(buffer);
        const sendMs = performance.now() - sendStart;

        frameCount++;
        state.frameCount = frameCount;

        // Record metrics
        metrics.frameCaptureLatency.observe({ session_id: sessionId }, decodeMs);
        metrics.frameE2ELatency.observe({ session_id: sessionId }, decodeMs + sendMs);

        // Log periodically for debugging (every 60 frames ~ 1-2 seconds at 30-60 FPS)
        if (frameCount % 60 === 0) {
          logger.debug(scopedLog(LogContext.RECORDING, 'screencast stats'), {
            sessionId,
            frameCount,
            frameBytes: buffer.length,
            decodeMs: decodeMs.toFixed(2),
            sendMs: sendMs.toFixed(2),
            totalMs: (performance.now() - frameStart).toFixed(2),
          });
        }
      }

      // 3. CRITICAL: Acknowledge frame or Chrome stops sending!
      // Wrap in timeout to prevent hanging if CDP is slow
      await ackWithTimeout(cdpSession, event.sessionId, ACK_TIMEOUT_MS);
      ackFailures = 0; // Reset on success
    } catch (error) {
      ackFailures++;
      const message = error instanceof Error ? error.message : String(error);

      // Log but don't throw - we want to keep receiving frames
      logger.warn(scopedLog(LogContext.RECORDING, 'screencast frame error'), {
        sessionId,
        error: message,
        frameNumber: event.sessionId,
        ackFailures,
      });

      // If too many ACK failures, something is wrong
      if (ackFailures >= MAX_ACK_FAILURES) {
        logger.error(scopedLog(LogContext.RECORDING, 'screencast ACK failures exceeded threshold'), {
          sessionId,
          ackFailures,
          hint: 'CDP session may be unhealthy, consider restarting screencast',
        });
      }
    }
  });

  // Handle visibility changes (Chrome notifies us when tab becomes hidden/visible)
  cdpSession.on('Page.screencastVisibilityChanged', (event: ScreencastVisibilityEvent) => {
    logger.debug(scopedLog(LogContext.RECORDING, 'screencast visibility changed'), {
      sessionId,
      visible: event.visible,
    });
  });

  // Handle CDP session disconnect (event is emitted but not typed by Playwright)
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  (cdpSession as any).on('disconnected', () => {
    if (isActive) {
      logger.warn(scopedLog(LogContext.RECORDING, 'CDP session disconnected during screencast'), {
        sessionId,
        frameCount,
      });
      isActive = false;
    }
  });

  // Handle page close
  const pageCloseHandler = () => {
    if (isActive) {
      logger.debug(scopedLog(LogContext.RECORDING, 'page closed during screencast'), {
        sessionId,
        frameCount,
      });
      isActive = false;
    }
  };
  page.on('close', pageCloseHandler);

  // Start screencast
  logger.info(scopedLog(LogContext.RECORDING, 'starting CDP screencast'), {
    sessionId,
    config: {
      format: config.format,
      quality: config.quality,
      maxWidth: config.maxWidth,
      maxHeight: config.maxHeight,
      everyNthFrame: config.everyNthFrame,
    },
  });

  await cdpSession.send('Page.startScreencast', config);

  return {
    cdpSession,
    getFrameCount: () => frameCount,
    isActive: () => isActive,
    stop: async () => {
      if (!isActive) return;
      isActive = false;

      // Remove page close listener
      page.off('close', pageCloseHandler);

      try {
        await cdpSession.send('Page.stopScreencast');
      } catch (error) {
        // May fail if page already closed
        logger.debug(scopedLog(LogContext.RECORDING, 'stopScreencast failed (page may be closed)'), {
          sessionId,
          error: error instanceof Error ? error.message : String(error),
        });
      }

      try {
        await cdpSession.detach();
      } catch {
        // Session may already be detached
      }

      logger.info(scopedLog(LogContext.RECORDING, 'CDP screencast stopped'), {
        sessionId,
        totalFrames: frameCount,
      });
    },
  };
}

/**
 * Send screencastFrameAck with timeout protection.
 *
 * Without ACK, Chrome stops sending frames. But if CDP is slow,
 * we don't want to hang forever waiting for ACK to complete.
 */
async function ackWithTimeout(
  cdpSession: CDPSession,
  frameSessionId: number,
  timeoutMs: number
): Promise<void> {
  const ackPromise = cdpSession.send('Page.screencastFrameAck', {
    sessionId: frameSessionId,
  });

  const timeoutPromise = new Promise<never>((_, reject) => {
    setTimeout(() => reject(new Error(`ACK timeout after ${timeoutMs}ms`)), timeoutMs);
  });

  await Promise.race([ackPromise, timeoutPromise]);
}
