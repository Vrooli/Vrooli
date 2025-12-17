/**
 * CDP Screencast Strategy
 *
 * Push-based frame streaming using Chrome's native Page.startScreencast API.
 * This is the preferred strategy for Chromium-based browsers.
 *
 * Key differences from polling:
 * - Chrome controls frame timing (compositor-native)
 * - Built-in change detection (no frames sent if nothing changed)
 * - Must acknowledge each frame via screencastFrameAck
 *
 * Expected performance: 30-60 FPS (vs 10-15 FPS with polling)
 *
 * @module frame-streaming/strategies/cdp-screencast
 */

import type { CDPSession, Page } from 'playwright';
import { logger, scopedLog, LogContext, metrics } from '../../utils';
import type {
  FrameStreamingStrategy,
  StreamingStrategyConfig,
  StreamingHandle,
  WebSocketProvider,
  FrameStatsReporter,
} from './interface';

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

/** ACK timeout - prevents hanging if CDP is slow */
const ACK_TIMEOUT_MS = 1000;

/** Max consecutive ACK failures before logging error */
const MAX_ACK_FAILURES = 5;

/** Frame logging interval (every N frames) */
const FRAME_LOG_INTERVAL = 60;

/**
 * CDP Screencast streaming strategy.
 *
 * Uses Chrome's native screencast API for efficient frame capture.
 * Frames are pushed from the compositor and must be acknowledged.
 */
export class CdpScreencastStrategy implements FrameStreamingStrategy {
  readonly name = 'cdp-screencast';

  async isSupported(page: Page): Promise<boolean> {
    try {
      // CDP is only available for Chromium-based browsers
      const browserType = page.context().browser()?.browserType().name();
      return browserType === 'chromium';
    } catch {
      return false;
    }
  }

  async start(
    page: Page,
    config: StreamingStrategyConfig,
    wsProvider: WebSocketProvider,
    statsReporter: FrameStatsReporter
  ): Promise<StreamingHandle> {
    const cdpSession = await page.context().newCDPSession(page);
    let isActive = true;
    let frameCount = 0;
    let ackFailures = 0;
    let currentQuality = config.quality;

    const { sessionId } = config;

    // Get viewport for screencast config
    const viewport = page.viewportSize() ?? { width: 1280, height: 720 };

    // Handle incoming frames
    cdpSession.on('Page.screencastFrame', async (event: ScreencastFrameEvent) => {
      if (!isActive) return;

      const frameStart = performance.now();

      try {
        // Check WebSocket before processing
        if (!wsProvider.isReady()) {
          statsReporter.onFrameSkipped('ws_not_ready');
          // Still need to ACK the frame
          await ackWithTimeout(cdpSession, event.sessionId, ACK_TIMEOUT_MS);
          return;
        }

        // Decode base64 to buffer (~1-2ms overhead, but CDP doesn't support binary)
        const decodeStart = performance.now();
        const buffer = Buffer.from(event.data, 'base64');
        const decodeMs = performance.now() - decodeStart;

        // Forward to WebSocket
        const ws = wsProvider.getWebSocket();
        if (ws && ws.readyState === 1) {
          const sendStart = performance.now();
          ws.send(buffer);
          const sendMs = performance.now() - sendStart;

          frameCount++;

          // Report stats
          statsReporter.onFrameSent({
            captureMs: decodeMs, // For screencast, "capture" is really decode
            wsSendMs: sendMs,
            frameBytes: buffer.length,
          });

          // Record metrics
          metrics.frameCaptureLatency.observe({ session_id: sessionId }, decodeMs);
          metrics.frameE2ELatency.observe({ session_id: sessionId }, decodeMs + sendMs);

          // Log periodically
          if (frameCount % FRAME_LOG_INTERVAL === 0) {
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

        // CRITICAL: Acknowledge frame or Chrome stops sending!
        await ackWithTimeout(cdpSession, event.sessionId, ACK_TIMEOUT_MS);
        ackFailures = 0;
      } catch (error) {
        ackFailures++;
        const message = error instanceof Error ? error.message : String(error);

        logger.warn(scopedLog(LogContext.RECORDING, 'screencast frame error'), {
          sessionId,
          error: message,
          frameNumber: event.sessionId,
          ackFailures,
        });

        if (ackFailures >= MAX_ACK_FAILURES) {
          logger.error(scopedLog(LogContext.RECORDING, 'screencast ACK failures exceeded threshold'), {
            sessionId,
            ackFailures,
            hint: 'CDP session may be unhealthy, consider restarting screencast',
          });
        }
      }
    });

    // Handle visibility changes
    cdpSession.on('Page.screencastVisibilityChanged', (event: ScreencastVisibilityEvent) => {
      logger.debug(scopedLog(LogContext.RECORDING, 'screencast visibility changed'), {
        sessionId,
        visible: event.visible,
      });
    });

    // Handle CDP session disconnect
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
        format: 'jpeg',
        quality: currentQuality,
        maxWidth: viewport.width,
        maxHeight: viewport.height,
        everyNthFrame: 1,
      },
    });

    await cdpSession.send('Page.startScreencast', {
      format: 'jpeg',
      quality: currentQuality,
      maxWidth: viewport.width,
      maxHeight: viewport.height,
      everyNthFrame: 1, // Every frame - Chrome handles change detection
    });

    return {
      getFrameCount: () => frameCount,
      isActive: () => isActive,

      updateQuality: (quality: number) => {
        currentQuality = quality;
        // Note: Quality changes require restarting screencast in CDP
        // For now we just track it for the next restart
        logger.debug(scopedLog(LogContext.RECORDING, 'screencast quality updated'), {
          sessionId,
          quality,
          note: 'Takes effect on next screencast restart',
        });
      },

      stop: async () => {
        if (!isActive) return;
        isActive = false;

        page.off('close', pageCloseHandler);

        try {
          await cdpSession.send('Page.stopScreencast');
        } catch (error) {
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
}

/**
 * Send screencastFrameAck with timeout protection.
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

/**
 * Create a CDP screencast strategy instance.
 */
export function createCdpScreencastStrategy(): CdpScreencastStrategy {
  return new CdpScreencastStrategy();
}
