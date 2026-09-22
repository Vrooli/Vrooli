/**
 * Frame Streaming Manager
 *
 * Orchestrates frame streaming for recording sessions.
 * Supports CDP screencast (preferred) and polling (fallback).
 *
 * This is a thin orchestrator that delegates to:
 * - strategies/ - Frame capture implementations (CDP screencast, polling)
 * - websocket/ - WebSocket connection management
 *
 * @module frame-streaming/manager
 */

import { logger, scopedLog, LogContext } from '../utils';
import { loadConfig } from '../config';
import { PerfCollector } from '../performance';
import {
  createCdpScreencastStrategy,
  createPollingStrategy,
  type StreamingHandle,
  type WebSocketProvider,
  type FrameStatsReporter,
} from './strategies';
import {
  createWebSocketConnectionManager,
  buildWebSocketUrl,
  type WebSocketConnectionManager,
} from './websocket';
import type {
  FrameStreamOptions,
  FrameStreamUpdateOptions,
  FrameStreamSettings,
  FrameStreamViewportOptions,
  FrameStreamViewportResult,
  SessionProvider,
} from './types';
import type { Page } from 'rebrowser-playwright';

// =============================================================================
// Module State
// =============================================================================

/**
 * Per-session streaming state.
 * Tracks WebSocket connection, strategy handle, and configuration.
 */
interface StreamingSession {
  generation: number;
  wsManager: WebSocketConnectionManager;
  strategyHandle: StreamingHandle | null;
  strategyName: string;
  perfCollector: PerfCollector;
  quality: number;
  targetFps: number;
  scale: 'css' | 'device';
  includePerfHeaders: boolean;
}

interface StreamingSlot {
  generation: number;
  stream?: StreamingSession;
  pending: Promise<void>;
}

/** One owner serializes acquisition and disposal, including pending starts. */
const slots = new Map<string, StreamingSlot>();

function currentStream(sessionId: string): StreamingSession | undefined {
  const slot = slots.get(sessionId);
  return slot?.stream?.generation === slot?.generation ? slot?.stream : undefined;
}

async function disposeStream(slot: StreamingSlot): Promise<void> {
  const stream = slot.stream;
  if (!stream) return;
  stream.wsManager.close();
  await stream.strategyHandle?.stop();
  // Keep a failed disposal owned so the next explicit stop/start can retry it.
  slot.stream = undefined;
}

// =============================================================================
// Public API
// =============================================================================

/**
 * Start frame streaming for a recording session.
 *
 * Uses CDP screencast by default for push-based frame delivery (30-60 FPS).
 * Falls back to polling-based screenshot capture if screencast fails.
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
  const slot: StreamingSlot = slots.get(sessionId) ?? { generation: 0, pending: Promise.resolve() };
  slots.set(sessionId, slot);
  const generation = ++slot.generation;
  const isCurrent = (): boolean => slot.generation === generation;
  slot.stream?.wsManager.close();
  slot.pending = slot.pending.then(async () => {
    if (!isCurrent()) return;
    await disposeStream(slot);
    if (!isCurrent()) return;

    const config = loadConfig();
    const fps = Math.min(Math.max(options.fps ?? 30, 1), 60);
    const stream: StreamingSession = {
      generation,
      wsManager: createWebSocketConnectionManager({
        url: buildWebSocketUrl(options.callbackUrl, sessionId), sessionId,
      }),
      strategyHandle: null,
      strategyName: 'none',
      perfCollector: PerfCollector.fromConfig(sessionId, config, fps),
      quality: options.quality ?? 65,
      targetFps: fps,
      scale: options.scale ?? 'css',
      includePerfHeaders: config.performance.enabled && config.performance.includeTimingHeaders,
    };
    slot.stream = stream;
    try {
      stream.wsManager.connect();
      await startWithStrategy(sessionId, sessionProvider, stream, config, isCurrent);
    } finally {
      if (!isCurrent() || !stream.strategyHandle) await disposeStream(slot);
    }
  }).catch((error: unknown) => {
    if (!slot.stream && isCurrent()) slots.delete(sessionId);
    logger.error(scopedLog(LogContext.RECORDING, 'frame streaming failed to start'), {
      sessionId, error: error instanceof Error ? error.message : String(error),
    });
  });
}

/** Invalidate transport immediately, then join pending acquisition and disposal. */
export function stopFrameStreaming(sessionId: string): Promise<void> {
  const slot = slots.get(sessionId);
  if (!slot) return Promise.resolve();
  const generation = ++slot.generation;
  slot.stream?.wsManager.close();
  const stopped = slot.pending.then(async () => {
    await disposeStream(slot);
    if (slot.generation === generation) slots.delete(sessionId);
  });
  // Observe rejection for the queue while returning it to the stop caller.
  slot.pending = stopped.catch(() => {});
  return stopped;
}

/**
 * Update frame streaming settings for an active session.
 *
 * @param sessionId - Session ID
 * @param options - Settings to update
 * @returns true if settings were updated, false if no active stream
 */
export function updateFrameStreamSettings(
  sessionId: string,
  options: FrameStreamUpdateOptions
): Promise<boolean> {
  const slot = slots.get(sessionId);
  if (!slot) return Promise.resolve(false);
  const generation = slot.generation;
  const requested = { ...options };
  const update = slot.pending.then(async () => {
    const session = currentStream(sessionId);
    if (slot.generation !== generation || !session?.strategyHandle?.isActive()) return false;
    const handle = session.strategyHandle;
    const quality = requested.quality === undefined ? session.quality : Math.min(Math.max(requested.quality, 1), 100);
    const fps = requested.fps === undefined ? session.targetFps : Math.min(Math.max(requested.fps, 1), 60);
    const perfMode = requested.perfMode ?? session.includePerfHeaders;
    const changed = quality !== session.quality || fps !== session.targetFps || perfMode !== session.includePerfHeaders;
    if (!changed) return false;

    // Check support before applying any part of the requested update.
    if (quality !== session.quality && !handle.updateQuality) throw new Error('Active stream does not support quality updates');
    if (fps !== session.targetFps && !handle.updateTargetFps) throw new Error('Active stream does not support FPS updates');
    if (perfMode !== session.includePerfHeaders && !handle.updatePerfMode) throw new Error('Active stream does not support performance header updates');
    if (quality !== session.quality) await handle.updateQuality!(quality);
    if (slot.generation !== generation || !handle.isActive()) return false;
    if (fps !== session.targetFps) handle.updateTargetFps!(fps);
    if (perfMode !== session.includePerfHeaders) handle.updatePerfMode!(perfMode);
    session.quality = quality;
    session.targetFps = fps;
    session.includePerfHeaders = perfMode;
    logger.info(scopedLog(LogContext.RECORDING, 'frame stream settings updated'), {
      sessionId, quality, targetFps: fps, perfMode, strategy: session.strategyName,
    });
    return true;
  });
  slot.pending = update.then(() => {}, () => {});
  return update;
}

/**
 * Get current frame streaming settings for a session.
 *
 * @param sessionId - Session ID
 * @returns Current settings or null if no active stream
 */
export function getFrameStreamSettings(sessionId: string): FrameStreamSettings | null {
  const session = currentStream(sessionId);
  if (!session) {
    return null;
  }

  return {
    quality: session.quality,
    fps: session.targetFps,
    scale: session.scale,
    currentFps: session.strategyHandle?.isActive() ? session.perfCollector.getAggregatedStats().actual_fps : 0,
    isStreaming: session.strategyHandle?.isActive() ?? false,
    perfMode: session.includePerfHeaders,
  };
}

/**
 * Update viewport dimensions for an active frame streaming session.
 * This may require restarting the screencast (for CDP strategy).
 *
 * @param sessionId - Session ID
 * @param options - New viewport dimensions
 * @returns Result of the viewport update operation
 */
export async function updateFrameStreamViewport(
  sessionId: string,
  options: FrameStreamViewportOptions
): Promise<FrameStreamViewportResult> {
  const session = currentStream(sessionId);
  if (!session || !session.strategyHandle?.isActive()) {
    return { success: false, error: 'No active stream for session' };
  }

  // Check if viewport update is supported by the strategy
  if (!session.strategyHandle.updateViewport) {
    return { success: false, error: 'Viewport update not supported by strategy' };
  }

  // Check if an update is already pending
  if (session.strategyHandle.isViewportUpdatePending?.()) {
    return { success: false, skipped: true, error: 'Viewport update already pending' };
  }

  try {
    await session.strategyHandle.updateViewport(options.width, options.height);

    logger.info(scopedLog(LogContext.RECORDING, 'frame stream viewport updated'), {
      sessionId,
      viewport: options,
      strategy: session.strategyName,
    });

    return { success: true };
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);

    logger.error(scopedLog(LogContext.RECORDING, 'frame stream viewport update failed'), {
      sessionId,
      viewport: options,
      error: message,
    });

    return { success: false, error: message };
  }
}

/**
 * Check if a viewport update is currently in progress for a session.
 *
 * @param sessionId - Session ID
 * @returns true if viewport update is pending, false otherwise
 */
export function isViewportUpdatePending(sessionId: string): boolean {
  const session = currentStream(sessionId);
  if (!session || !session.strategyHandle) {
    return false;
  }

  return session.strategyHandle.isViewportUpdatePending?.() ?? false;
}

// =============================================================================
// Internal Functions
// =============================================================================

/**
 * Start streaming with the appropriate strategy.
 * Tries CDP screencast first, falls back to polling if configured.
 */
async function startWithStrategy(
  sessionId: string,
  sessionProvider: SessionProvider,
  session: StreamingSession,
  config: ReturnType<typeof loadConfig>,
  isCurrent: () => boolean
): Promise<void> {
  // Create page provider function for multi-tab support
  // This is called on each frame capture to get the current active page
  const pageProvider = (): Page => {
    if (!isCurrent()) throw new Error('Frame stream was stopped or replaced');
    return sessionProvider.getSession(sessionId).page;
  };
  const page = pageProvider();

  // Create strategy instances
  const screencastStrategy = createCdpScreencastStrategy();
  const pollingStrategy = createPollingStrategy();

  // Screencast supplies CSS pixels. The SDK screenshot owner supplies device pixels.
  const useScreencast = session.scale === 'css' && config.frameStreaming.useScreencast
    && await screencastStrategy.isSupported(page);
  const strategies = useScreencast
    ? (config.frameStreaming.fallbackToPolling ? [screencastStrategy, pollingStrategy] : [screencastStrategy])
    : [pollingStrategy];

  // Create WebSocket provider adapter
  const wsProvider: WebSocketProvider = {
    getWebSocket: () => isCurrent() ? session.wsManager.getWebSocket() : null,
    isReady: () => isCurrent() && session.wsManager.isReady(),
  };

  // Create stats reporter adapter
  const statsReporter: FrameStatsReporter = {
    onFrameSent: (stats) => {
      if (!isCurrent()) return;
      session.perfCollector.recordFrame({
        captureMs: stats.captureMs,
        compareMs: stats.compareMs ?? 0,
        wsSendMs: stats.wsSendMs,
        frameBytes: stats.frameBytes,
        skipped: false,
      });

      // Log summary periodically
      if (session.perfCollector.shouldLogSummary()) {
        const aggregated = session.perfCollector.getAggregatedStats();
        logger.info(scopedLog(LogContext.RECORDING, 'frame perf summary'), {
          session_id: sessionId,
          frame_count: aggregated.frame_count,
          skipped_count: aggregated.skipped_count,
          capture_p50_ms: aggregated.capture_p50_ms,
          capture_p90_ms: aggregated.capture_p90_ms,
          e2e_p50_ms: aggregated.e2e_p50_ms,
          e2e_p90_ms: aggregated.e2e_p90_ms,
          bottleneck: aggregated.primary_bottleneck,
          strategy: session.strategyName,
        });
      }
    },
    onFrameSkipped: (reason) => {
      if (isCurrent() && reason === 'unchanged') {
        session.perfCollector.recordSkipped(0, 0);
      }
    },
  };

  for (const [index, strategy] of strategies.entries()) {
    if (!isCurrent()) return;
    session.strategyName = strategy.name;
    try {
      session.strategyHandle = await strategy.start(pageProvider, {
        sessionId,
        quality: session.quality,
        targetFps: session.targetFps,
        scale: session.scale,
        includePerfHeaders: session.includePerfHeaders,
      }, wsProvider, statsReporter);
      return;
    } catch (error) {
      if (!isCurrent()) return;
      if (index === strategies.length - 1) throw error;
      logger.warn(scopedLog(LogContext.RECORDING, 'CDP screencast failed, falling back to polling'), {
        sessionId, error: error instanceof Error ? error.message : String(error),
      });
    }
  }
}
