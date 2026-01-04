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

import type { CDPSession, Page } from 'rebrowser-playwright';
import { logger, scopedLog, LogContext, metrics } from '../../utils';
import type {
  FrameStreamingStrategy,
  StreamingStrategyConfig,
  StreamingHandle,
  WebSocketProvider,
  FrameStatsReporter,
  PageProvider,
  CdpStreamingConfig,
} from './interface';
import { DEFAULT_CDP_CONFIG } from './interface';

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

/**
 * Resolve CDP config by merging provided values with defaults.
 *
 * DECISION: Merge strategy uses spread with defaults first.
 * This ensures all values are present while allowing partial overrides.
 */
function resolveCdpConfig(partial?: Partial<CdpStreamingConfig>): CdpStreamingConfig {
  return {
    ...DEFAULT_CDP_CONFIG,
    ...partial,
  };
}

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
    pageProvider: PageProvider,
    config: StreamingStrategyConfig,
    wsProvider: WebSocketProvider,
    statsReporter: FrameStatsReporter
  ): Promise<StreamingHandle> {
    // Resolve CDP config with defaults for any unspecified values
    const cdpConfig = resolveCdpConfig(config.cdp);

    // Get initial page - CDP screencast is bound to a specific page
    let currentPage = pageProvider();
    let cdpSession = await currentPage.context().newCDPSession(currentPage);
    let isActive = true;
    let frameCount = 0;
    let ackFailures = 0;
    let currentQuality = config.quality;
    let pageCheckInterval: ReturnType<typeof setInterval> | null = null;

    const { sessionId } = config;

    // Helper to restart screencast on a new page
    const restartScreencastOnPage = async (newPage: Page) => {
      // Stop old screencast
      try {
        await cdpSession.send('Page.stopScreencast');
        await cdpSession.detach();
      } catch {
        // Ignore errors - old session may already be invalid
      }

      // Start new screencast on new page
      currentPage = newPage;
      cdpSession = await newPage.context().newCDPSession(newPage);

      // Re-attach frame handler
      setupFrameHandler();

      const viewport = newPage.viewportSize() ?? { width: 1280, height: 720 };
      await cdpSession.send('Page.startScreencast', {
        format: 'jpeg',
        quality: currentQuality,
        maxWidth: viewport.width,
        maxHeight: viewport.height,
        everyNthFrame: 1,
      });

      logger.info(scopedLog(LogContext.RECORDING, 'screencast restarted for new page'), {
        sessionId,
        frameCount,
      });
    };

    // Check for page changes periodically
    // Trade-off: Lower interval = faster tab detection, more CPU overhead
    pageCheckInterval = setInterval(() => {
      if (!isActive) return;
      const newPage = pageProvider();
      if (newPage !== currentPage && !newPage.isClosed()) {
        void restartScreencastOnPage(newPage);
      }
    }, cdpConfig.pageCheckIntervalMs);

    // Setup frame handler (extracted so we can re-attach after page switch)
    const setupFrameHandler = () => {
      cdpSession.on('Page.screencastFrame', async (event: ScreencastFrameEvent) => {
        if (!isActive) return;

        const frameStart = performance.now();

        try {
          // Check WebSocket before processing
          if (!wsProvider.isReady()) {
            statsReporter.onFrameSkipped('ws_not_ready');
            // Still need to ACK the frame to prevent Chrome from stopping
            await ackWithTimeout(cdpSession, event.sessionId, cdpConfig.ackTimeoutMs);
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

            // Log periodically (if enabled)
            // DECISION: frameLogInterval=0 disables periodic logging
            if (cdpConfig.frameLogInterval > 0 && frameCount % cdpConfig.frameLogInterval === 0) {
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
          await ackWithTimeout(cdpSession, event.sessionId, cdpConfig.ackTimeoutMs);
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

          // DECISION: Log error when consecutive failures exceed threshold
          // This indicates a potentially unhealthy CDP session
          if (ackFailures >= cdpConfig.maxAckFailures) {
            logger.error(scopedLog(LogContext.RECORDING, 'screencast ACK failures exceeded threshold'), {
              sessionId,
              ackFailures,
              threshold: cdpConfig.maxAckFailures,
              hint: 'CDP session may be unhealthy, consider restarting screencast',
            });
          }
        }
      });
    };

    // Initial frame handler setup
    setupFrameHandler();

    // Get viewport for screencast config
    const viewport = currentPage.viewportSize() ?? { width: 1280, height: 720 };

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

        // Clear page change detection interval
        if (pageCheckInterval) {
          clearInterval(pageCheckInterval);
          pageCheckInterval = null;
        }

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
