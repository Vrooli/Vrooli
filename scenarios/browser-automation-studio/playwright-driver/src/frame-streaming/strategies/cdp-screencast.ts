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
import type { DirectFrameServer } from '../websocket';

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
    let currentViewport = currentPage.viewportSize() ?? { width: 1280, height: 720 };
    let viewportUpdatePending = false;
    let pageCheckInterval: ReturnType<typeof setInterval> | null = null;

    const { sessionId } = config;

    // Minimum change threshold to trigger restart (prevents rapid restarts during resize)
    const VIEWPORT_CHANGE_THRESHOLD = 20; // pixels

    // Buffer for frames received before WebSocket is ready
    // This prevents flickering/white screen on initial connection
    let pendingFrame: { data: string; sessionId: number } | null = null;

    // Helper to restart screencast on a new page or with new viewport
    const restartScreencast = async (newPage: Page, newViewport?: { width: number; height: number }) => {
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

      // Use provided viewport or get from page
      const viewport = newViewport ?? newPage.viewportSize() ?? { width: 1280, height: 720 };
      currentViewport = viewport;

      await cdpSession.send('Page.startScreencast', {
        format: 'jpeg',
        quality: currentQuality,
        maxWidth: viewport.width,
        maxHeight: viewport.height,
        everyNthFrame: 1,
      });

      logger.info(scopedLog(LogContext.RECORDING, 'screencast restarted'), {
        sessionId,
        frameCount,
        viewport,
        reason: newViewport ? 'viewport_update' : 'page_change',
      });
    };

    // Legacy wrapper for page changes (maintains existing API)
    const restartScreencastOnPage = async (newPage: Page) => {
      await restartScreencast(newPage);
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

    // Helper to send a frame (used for both live and buffered frames)
    const sendFrame = (
      frameData: string,
      _frameSessionId: number,
      isBuffered: boolean
    ): { buffer: Buffer; decodeMs: number } | null => {
      const ws = wsProvider.getWebSocket();
      if (!ws || ws.readyState !== 1) {
        return null;
      }

      const decodeStart = performance.now();
      const buffer = Buffer.from(frameData, 'base64');
      const decodeMs = performance.now() - decodeStart;

      const sendStart = performance.now();

      // Build frame with optional perf header
      let frameToSend: Buffer;
      if (config.includePerfHeaders) {
        const header = {
          frame_id: `${sessionId}-${frameCount + 1}`,
          capture_ms: decodeMs,
          compare_ms: 0,
          ws_send_ms: 0,
          frame_bytes: buffer.length,
          sent_at: Date.now(),
          buffered: isBuffered,
        };
        const headerJson = Buffer.from(JSON.stringify(header), 'utf8');
        const headerLen = Buffer.alloc(4);
        headerLen.writeUInt32BE(headerJson.length, 0);
        frameToSend = Buffer.concat([headerLen, headerJson, buffer]);
      } else {
        const timestampBuffer = Buffer.alloc(8);
        timestampBuffer.writeBigInt64BE(BigInt(Date.now()), 0);
        frameToSend = Buffer.concat([timestampBuffer, buffer]);
      }

      ws.send(frameToSend);

      // Also broadcast to direct frame server
      const directServer = (global as { directFrameServer?: DirectFrameServer }).directFrameServer;
      if (directServer?.hasSubscribers(sessionId)) {
        directServer.broadcast(buffer, sessionId);
      }

      const sendMs = performance.now() - sendStart;

      frameCount++;

      statsReporter.onFrameSent({
        captureMs: decodeMs,
        wsSendMs: sendMs,
        frameBytes: buffer.length,
      });

      metrics.frameCaptureLatency.observe({ session_id: sessionId }, decodeMs);
      metrics.frameE2ELatency.observe({ session_id: sessionId }, decodeMs + sendMs);

      if (isBuffered) {
        logger.debug(scopedLog(LogContext.RECORDING, 'sent buffered frame'), {
          sessionId,
          frameBytes: buffer.length,
        });
      }

      return { buffer, decodeMs };
    };

    // Setup frame handler (extracted so we can re-attach after page switch)
    const setupFrameHandler = () => {
      cdpSession.on('Page.screencastFrame', async (event: ScreencastFrameEvent) => {
        if (!isActive) return;

        const frameStart = performance.now();

        try {
          // Check WebSocket readiness
          if (!wsProvider.isReady()) {
            // Buffer this frame instead of dropping it
            // Only keep the latest frame to avoid memory buildup
            pendingFrame = { data: event.data, sessionId: event.sessionId };
            statsReporter.onFrameSkipped('ws_not_ready');
            // Still need to ACK the frame to prevent Chrome from stopping
            await ackWithTimeout(cdpSession, event.sessionId, cdpConfig.ackTimeoutMs);
            return;
          }

          // WebSocket is ready - first send any buffered frame
          if (pendingFrame) {
            sendFrame(pendingFrame.data, pendingFrame.sessionId, true);
            pendingFrame = null;
          }

          // Now send the current frame
          const result = sendFrame(event.data, event.sessionId, false);
          if (!result) {
            // WebSocket closed between ready check and send
            await ackWithTimeout(cdpSession, event.sessionId, cdpConfig.ackTimeoutMs);
            return;
          }

          const { buffer, decodeMs } = result;

          // Log periodically (if enabled)
          if (cdpConfig.frameLogInterval > 0 && frameCount % cdpConfig.frameLogInterval === 0) {
            logger.debug(scopedLog(LogContext.RECORDING, 'screencast stats'), {
              sessionId,
              frameCount,
              frameBytes: buffer.length,
              decodeMs: decodeMs.toFixed(2),
              totalMs: (performance.now() - frameStart).toFixed(2),
            });
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
      isViewportUpdatePending: () => viewportUpdatePending,

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

      updateViewport: async (width: number, height: number) => {
        if (!isActive) return;

        // Check if viewport change is significant enough to warrant restart
        const widthDiff = Math.abs(width - currentViewport.width);
        const heightDiff = Math.abs(height - currentViewport.height);

        if (widthDiff < VIEWPORT_CHANGE_THRESHOLD && heightDiff < VIEWPORT_CHANGE_THRESHOLD) {
          logger.debug(scopedLog(LogContext.RECORDING, 'viewport change below threshold, skipping restart'), {
            sessionId,
            current: currentViewport,
            requested: { width, height },
            threshold: VIEWPORT_CHANGE_THRESHOLD,
          });
          return;
        }

        // Prevent concurrent viewport updates
        if (viewportUpdatePending) {
          logger.debug(scopedLog(LogContext.RECORDING, 'viewport update already pending, skipping'), {
            sessionId,
            requested: { width, height },
          });
          return;
        }

        viewportUpdatePending = true;
        const updateStart = performance.now();

        try {
          logger.info(scopedLog(LogContext.RECORDING, 'updating viewport'), {
            sessionId,
            from: currentViewport,
            to: { width, height },
          });

          // Update Playwright viewport first
          await currentPage.setViewportSize({ width, height });

          // Restart screencast with new dimensions
          await restartScreencast(currentPage, { width, height });

          const updateMs = performance.now() - updateStart;
          logger.info(scopedLog(LogContext.RECORDING, 'viewport update complete'), {
            sessionId,
            viewport: { width, height },
            updateMs: updateMs.toFixed(2),
          });
        } catch (error) {
          logger.error(scopedLog(LogContext.RECORDING, 'viewport update failed'), {
            sessionId,
            error: error instanceof Error ? error.message : String(error),
            requested: { width, height },
          });
          throw error;
        } finally {
          viewportUpdatePending = false;
        }
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
