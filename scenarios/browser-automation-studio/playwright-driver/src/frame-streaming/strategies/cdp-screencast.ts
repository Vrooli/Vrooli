import { encodeFrame, sameFrameSource, type FrameSource } from '../frame';
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
import { MAX_QUEUED_FRAME_BYTES } from '../types';

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

  isSupported(page: Page): Promise<boolean> {
    try {
      // CDP is only available for Chromium-based browsers
      const browserType = page.context().browser()?.browserType().name();
      return Promise.resolve(browserType === 'chromium');
    } catch {
      return Promise.resolve(false);
    }
  }

  async start(
    pageProvider: PageProvider,
    config: StreamingStrategyConfig,
    wsProvider: WebSocketProvider,
    statsReporter: FrameStatsReporter
  ): Promise<StreamingHandle> {
    if (config.scale === 'device')
      throw new Error('CDP screencast does not support device scale; use SDK polling');
    const cdpConfig = resolveCdpConfig(config.cdp);
    const { sessionId } = config;
    type Capture = {
      started: boolean;
      page: Page;
      cdp: CDPSession;
      generation: number;
      listener: (event: ScreencastFrameEvent) => void;
    };
    let capture: Capture | undefined;
    let generation = 0;
    let active = true;
    let currentPage = pageProvider();
    let currentViewport = currentPage.viewportSize() ?? { width: 1280, height: 720 };
    let currentQuality = config.quality;
    let targetFps = config.targetFps;
    let lastSentAt = -Infinity;
    let nextFrameAt = -Infinity;
    let frameDeadline: ReturnType<typeof setTimeout> | undefined;
    let frameCount = 0;
    let cdpFrameCount = 0;
    let lastCdpFrameTimestamp: number | undefined;
    let ackFailures = 0;
    let pendingFrame:
      | { owner: Capture; data: string; source: FrameSource; capturedAt: number }
      | undefined;
    let transition: Promise<void> = Promise.resolve();
    let stopping: Promise<void> | undefined;
    // eslint-disable-next-line prefer-const -- start only after CDP acquisition succeeds
    let pageCheckInterval: ReturnType<typeof setInterval> | undefined;

    const clearPending = (): void => {
      pendingFrame = undefined;
      if (frameDeadline) clearTimeout(frameDeadline);
      frameDeadline = undefined;
    };
    const owns = (owner: Capture): boolean =>
      active && capture === owner && generation === owner.generation;
    const disposeCapture = async (): Promise<void> => {
      const owner = capture;
      if (!owner) return;
      owner.cdp.off('Page.screencastFrame', owner.listener);
      try {
        await owner.cdp.send('Page.stopScreencast');
      } catch (error) {
        logger.debug(
          scopedLog(LogContext.RECORDING, 'stopScreencast failed (page may be closed)'),
          {
            sessionId,
            error: error instanceof Error ? error.message : String(error),
          }
        );
      }
      try {
        await owner.cdp.detach();
      } catch {
        /* The target may already be closed. */
      }
      capture = undefined;
    };

    const stopStreaming = (): Promise<void> => {
      if (stopping) return stopping;
      active = false;
      generation++;
      clearPending();
      if (pageCheckInterval) clearInterval(pageCheckInterval);
      // A late protocol acquisition still belongs to this transition. Its
      // generation check disposes it before the stop acknowledgement settles.
      stopping = transition.then(disposeCapture);
      return stopping;
    };

    const sendFrame = (pending: NonNullable<typeof pendingFrame>, buffered: boolean): boolean => {
      const { owner, data, source, capturedAt } = pending;
      if (
        !owns(owner) ||
        pageProvider() !== owner.page ||
        !sameFrameSource(config.sourceForPage(owner.page), source)
      )
        return false;
      const ws = wsProvider.getWebSocket();
      if (!ws || ws.readyState !== 1) return false;
      const decodeStart = performance.now();
      const buffer = Buffer.from(data, 'base64');
      const decodeMs = performance.now() - decodeStart;
      const sendStart = performance.now();
      const frame = encodeFrame(
        source,
        buffer,
        capturedAt,
        config.includePerfHeaders
          ? {
              frame_id: `${sessionId}-${frameCount + 1}`,
              capture_ms: decodeMs,
              compare_ms: 0,
              ws_send_ms: 0,
              frame_bytes: buffer.length,
              sent_at: Date.now(),
              buffered,
            }
          : undefined
      );
      if ((ws.bufferedAmount ?? 0) + frame.length > MAX_QUEUED_FRAME_BYTES) {
        statsReporter.onFrameSkipped('ws_backpressure');
        return false;
      }
      ws.send(frame);
      const sentAt = performance.now();
      lastSentAt = sentAt;
      const intervalMs = 1000 / targetFps;
      if (!Number.isFinite(nextFrameAt)) {
        nextFrameAt = sentAt + intervalMs;
      } else {
        nextFrameAt += intervalMs;
        if (nextFrameAt <= sentAt) {
          const missedIntervals = Math.floor((sentAt - nextFrameAt) / intervalMs) + 1;
          nextFrameAt += missedIntervals * intervalMs;
        }
      }
      const sendMs = performance.now() - sendStart;
      frameCount++;
      statsReporter.onFrameSent({
        captureMs: decodeMs,
        compareMs: 0,
        wsSendMs: sendMs,
        frameBytes: buffer.length,
      });
      metrics.frameCaptureLatency.observe({ session_id: sessionId }, decodeMs);
      metrics.frameE2ELatency.observe({ session_id: sessionId }, decodeMs + sendMs);
      if (cdpConfig.frameLogInterval > 0 && frameCount % cdpConfig.frameLogInterval === 0) {
        logger.debug(scopedLog(LogContext.RECORDING, 'screencast stats'), {
          sessionId,
          frameCount,
          frameBytes: buffer.length,
        });
      }
      return true;
    };

    const flushPending = (buffered = true): void => {
      if (!active || !pendingFrame || !wsProvider.isReady()) return;
      const remaining = nextFrameAt - performance.now();
      if (remaining > 0) {
        if (!frameDeadline) {
          frameDeadline = setTimeout(deliverPending, Math.ceil(remaining));
        }
        return;
      }
      if (sendFrame(pendingFrame, buffered)) clearPending();
    };

    const deliverPending = (): void => {
      frameDeadline = undefined;
      try {
        flushPending();
      } catch (error) {
        logger.warn(scopedLog(LogContext.RECORDING, 'buffered frame delivery failed'), {
          sessionId,
          error: String(error),
        });
      }
    };

    const handleFrame = async (owner: Capture, event: ScreencastFrameEvent): Promise<void> => {
      if (!owns(owner)) return;
      cdpFrameCount++;
      const observedFrameCount = cdpFrameCount;
      const cdpTimestamp = event.metadata.timestamp;
      const cdpFrameGapMs =
        cdpTimestamp !== undefined && lastCdpFrameTimestamp !== undefined
          ? (cdpTimestamp - lastCdpFrameTimestamp) * 1000
          : undefined;
      if (cdpTimestamp !== undefined) lastCdpFrameTimestamp = cdpTimestamp;
      try {
        if (pageProvider() !== owner.page) return;
        const source = config.sourceForPage(owner.page);
        if (!source) return;
        pendingFrame = { owner, data: event.data, source, capturedAt: Date.now() };
        if (!wsProvider.isReady()) statsReporter.onFrameSkipped('ws_not_ready');
        else flushPending(false);
      } catch (error) {
        logger.warn(scopedLog(LogContext.RECORDING, 'screencast frame delivery failed'), {
          sessionId,
          error: error instanceof Error ? error.message : String(error),
        });
      } finally {
        // Transport failure cannot withhold Chrome's ACK. The emitter owns it
        // even if another capture becomes current while delivery settles.
        try {
          const ackMs = await ackWithTimeout(owner.cdp, event.sessionId, cdpConfig.ackTimeoutMs);
          if (owns(owner)) ackFailures = 0;
          if (
            cdpConfig.frameLogInterval > 0 &&
            observedFrameCount % cdpConfig.frameLogInterval === 0
          ) {
            logger.debug(scopedLog(LogContext.RECORDING, 'screencast ACK timing'), {
              sessionId,
              cdpFrameCount: observedFrameCount,
              cdpFrameGapMs,
              ackMs,
            });
          }
        } catch (error) {
          if (owns(owner)) {
            ackFailures++;
            logger.warn(scopedLog(LogContext.RECORDING, 'screencast ACK failed'), {
              sessionId,
              error: error instanceof Error ? error.message : String(error),
              frameNumber: event.sessionId,
              ackFailures,
            });
            if (ackFailures >= cdpConfig.maxAckFailures) {
              logger.error(
                scopedLog(LogContext.RECORDING, 'CDP screencast ACK failures exceeded threshold'),
                {
                  sessionId,
                  ackFailures,
                  threshold: cdpConfig.maxAckFailures,
                }
              );
            }
          }
        }
      }
    };

    const changeCapture = (
      page: Page,
      viewport?: { width: number; height: number },
      quality = currentQuality
    ): Promise<boolean> => {
      const revision = ++generation;
      currentPage = page;
      lastCdpFrameTimestamp = undefined;
      clearPending();
      lastSentAt = -Infinity;
      const changed = transition.then(async () => {
        if (!active || generation !== revision) return false;
        await disposeCapture();
        if (!active || generation !== revision) return false;
        const cdp = await page.context().newCDPSession(page);
        const owner: Capture = {
          started: false,
          page,
          cdp,
          generation: revision,
          listener: (event) => {
            void handleFrame(owner, event);
          },
        };
        capture = owner;
        try {
          if (!owns(owner) || pageProvider() !== page) return false;
          cdp.on('Page.screencastFrame', owner.listener);
          const dimensions = viewport ?? page.viewportSize() ?? { width: 1280, height: 720 };
          await cdp.send('Page.startScreencast', {
            format: 'jpeg',
            quality,
            maxWidth: dimensions.width,
            maxHeight: dimensions.height,
            everyNthFrame: 1,
          });
          if (!owns(owner) || pageProvider() !== page) return false;
          currentViewport = dimensions;
          owner.started = true;
          return true;
        } finally {
          if (!owner.started) await disposeCapture();
        }
      });
      transition = changed.then(
        () => {},
        () => {}
      );
      return changed;
    };

    if (!(await changeCapture(currentPage)))
      throw new Error('Frame capture was stopped or replaced');
    pageCheckInterval = setInterval(() => {
      if (!active) return;
      try {
        const page = pageProvider();
        if (page !== currentPage && !page.isClosed()) {
          void changeCapture(page).catch((error: unknown) => {
            logger.warn(scopedLog(LogContext.RECORDING, 'screencast page change failed'), {
              sessionId,
              error: String(error),
            });
            void stopStreaming();
          });
        } else {
          // A stable page may emit no second frame when transport reconnects.
          flushPending();
        }
      } catch {
        void stopStreaming();
      }
    }, cdpConfig.pageCheckIntervalMs);
    return {
      getFrameCount: () => frameCount,
      isActive: () => active,
      updateQuality: async (quality: number): Promise<void> => {
        if (!active) throw new Error('Frame capture was stopped');
        const applying = changeCapture(currentPage, undefined, quality);
        const revision = generation;
        try {
          if (!(await applying) || revision !== generation) {
            throw new Error('Frame capture was stopped or replaced during quality update');
          }
          currentQuality = quality;
        } catch (error) {
          if (revision === generation) await stopStreaming();
          throw error;
        }
      },
      updatePerfMode: (enabled: boolean): void => {
        config.includePerfHeaders = enabled;
      },
      updateTargetFps: (fps: number): void => {
        targetFps = Math.min(Math.max(fps, 1), 60);
        nextFrameAt = Number.isFinite(lastSentAt) ? lastSentAt + 1000 / targetFps : -Infinity;
        if (frameDeadline) clearTimeout(frameDeadline);
        frameDeadline = undefined;
        if (pendingFrame) frameDeadline = setTimeout(deliverPending, 0);
      },
      updateViewport: async (page: Page): Promise<void> => {
        if (!active || pageProvider() !== page)
          throw new Error('Frame capture was stopped or its page changed');
        const viewport = page.viewportSize();
        if (!viewport) throw new Error('Applied viewport is unavailable');
        if (
          capture?.started &&
          owns(capture) &&
          page === currentPage &&
          viewport.width === currentViewport.width &&
          viewport.height === currentViewport.height
        )
          return;
        if (!(await changeCapture(page, viewport)))
          throw new Error('Frame capture was stopped or replaced during resize');
      },
      stop: stopStreaming,
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
): Promise<number> {
  const startedAt = performance.now();
  const ackPromise = cdpSession.send('Page.screencastFrameAck', {
    sessionId: frameSessionId,
  });

  let deadline: ReturnType<typeof setTimeout> | undefined;
  try {
    await Promise.race([
      ackPromise,
      new Promise<never>((_, reject) => {
        deadline = setTimeout(
          () => reject(new Error(`ACK timeout after ${timeoutMs}ms`)),
          timeoutMs
        );
      }),
    ]);
    return performance.now() - startedAt;
  } finally {
    if (deadline) clearTimeout(deadline);
  }
}

/**
 * Create a CDP screencast strategy instance.
 */
export function createCdpScreencastStrategy(): CdpScreencastStrategy {
  return new CdpScreencastStrategy();
}
