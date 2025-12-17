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
  type FrameStreamingStrategy,
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
  SessionProvider,
} from './types';

// =============================================================================
// Module State
// =============================================================================

/**
 * Per-session streaming state.
 * Tracks WebSocket connection, strategy handle, and configuration.
 */
interface StreamingSession {
  wsManager: WebSocketConnectionManager;
  strategyHandle: StreamingHandle | null;
  strategyName: string;
  perfCollector: PerfCollector;
  quality: number;
  targetFps: number;
  scale: 'css' | 'device';
  includePerfHeaders: boolean;
}

/** Active streaming sessions */
const sessions = new Map<string, StreamingSession>();

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
  // Stop any existing stream for this session
  void stopFrameStreaming(sessionId);

  const config = loadConfig();
  const quality = options.quality ?? 65;
  const fps = Math.min(Math.max(options.fps ?? 30, 1), 60);
  const scale = options.scale ?? 'css';
  const wsUrl = buildWebSocketUrl(options.callbackUrl, sessionId);

  // Create WebSocket connection manager
  const wsManager = createWebSocketConnectionManager({
    url: wsUrl,
    sessionId,
  });

  // Create performance collector
  const perfCollector = PerfCollector.fromConfig(sessionId, config, fps);

  // Create session state
  const session: StreamingSession = {
    wsManager,
    strategyHandle: null,
    strategyName: 'none',
    perfCollector,
    quality,
    targetFps: fps,
    scale,
    includePerfHeaders: config.performance.enabled && config.performance.includeTimingHeaders,
  };

  sessions.set(sessionId, session);

  // Connect WebSocket first
  wsManager.connect();

  // Start streaming with appropriate strategy
  void startWithStrategy(sessionId, sessionProvider, session, config);
}

/**
 * Stop frame streaming for a session.
 *
 * @param sessionId - Session ID to stop streaming for
 */
export async function stopFrameStreaming(sessionId: string): Promise<void> {
  const session = sessions.get(sessionId);
  if (!session) return;

  // Stop strategy
  if (session.strategyHandle) {
    try {
      await session.strategyHandle.stop();
    } catch (error) {
      logger.debug(scopedLog(LogContext.RECORDING, 'strategy stop error'), {
        sessionId,
        error: error instanceof Error ? error.message : String(error),
      });
    }
  }

  // Close WebSocket
  session.wsManager.close();

  const frameCount = session.strategyHandle?.getFrameCount() ?? 0;
  sessions.delete(sessionId);

  logger.info(scopedLog(LogContext.RECORDING, 'frame streaming stopped'), {
    sessionId,
    totalFrames: frameCount,
    strategy: session.strategyName,
  });
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
): boolean {
  const session = sessions.get(sessionId);
  if (!session || !session.strategyHandle?.isActive()) {
    return false;
  }

  let changed = false;

  if (options.quality !== undefined) {
    const newQuality = Math.min(Math.max(options.quality, 1), 100);
    if (newQuality !== session.quality) {
      session.quality = newQuality;
      session.strategyHandle.updateQuality?.(newQuality);
      changed = true;
    }
  }

  if (options.fps !== undefined) {
    const newFps = Math.min(Math.max(options.fps, 1), 60);
    if (newFps !== session.targetFps) {
      session.targetFps = newFps;
      session.strategyHandle.updateTargetFps?.(newFps);
      changed = true;
    }
  }

  if (options.perfMode !== undefined && options.perfMode !== session.includePerfHeaders) {
    session.includePerfHeaders = options.perfMode;
    changed = true;
  }

  if (changed) {
    logger.info(scopedLog(LogContext.RECORDING, 'frame stream settings updated'), {
      sessionId,
      quality: session.quality,
      targetFps: session.targetFps,
      perfMode: session.includePerfHeaders,
      strategy: session.strategyName,
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
  const session = sessions.get(sessionId);
  if (!session) {
    return null;
  }

  return {
    quality: session.quality,
    fps: session.targetFps,
    scale: session.scale,
    currentFps: session.targetFps, // Actual FPS tracking would need strategy cooperation
    isStreaming: session.strategyHandle?.isActive() ?? false,
    perfMode: session.includePerfHeaders,
  };
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
  config: ReturnType<typeof loadConfig>
): Promise<void> {
  const page = sessionProvider.getSession(sessionId).page;

  // Create strategy instances
  const screencastStrategy = createCdpScreencastStrategy();
  const pollingStrategy = createPollingStrategy();

  // Determine which strategy to use
  let strategy: FrameStreamingStrategy;
  if (config.frameStreaming.useScreencast && await screencastStrategy.isSupported(page)) {
    strategy = screencastStrategy;
  } else {
    strategy = pollingStrategy;
  }

  // Create WebSocket provider adapter
  const wsProvider: WebSocketProvider = {
    getWebSocket: () => session.wsManager.getWebSocket(),
    isReady: () => session.wsManager.isReady(),
  };

  // Create stats reporter adapter
  const statsReporter: FrameStatsReporter = {
    onFrameSent: (stats) => {
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
          strategy: strategy.name,
        });
      }
    },
    onFrameSkipped: (reason) => {
      if (reason === 'unchanged') {
        session.perfCollector.recordSkipped(0, 0);
      }
    },
  };

  // Try to start the strategy
  try {
    // Wait briefly for WebSocket to connect
    await new Promise((resolve) => setTimeout(resolve, 100));

    session.strategyHandle = await strategy.start(
      page,
      {
        sessionId,
        quality: session.quality,
        targetFps: session.targetFps,
        scale: session.scale,
        includePerfHeaders: session.includePerfHeaders,
      },
      wsProvider,
      statsReporter
    );
    session.strategyName = strategy.name;

    logger.info(scopedLog(LogContext.RECORDING, 'frame streaming started'), {
      sessionId,
      strategy: strategy.name,
      quality: session.quality,
      targetFps: session.targetFps,
      scale: session.scale,
    });
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);

    // If screencast failed and fallback is enabled, try polling
    if (strategy.name === 'cdp-screencast' && config.frameStreaming.fallbackToPolling) {
      logger.warn(scopedLog(LogContext.RECORDING, 'CDP screencast failed, falling back to polling'), {
        sessionId,
        error: message,
      });

      try {
        session.strategyHandle = await pollingStrategy.start(
          page,
          {
            sessionId,
            quality: session.quality,
            targetFps: session.targetFps,
            scale: session.scale,
            includePerfHeaders: session.includePerfHeaders,
          },
          wsProvider,
          statsReporter
        );
        session.strategyName = 'polling';

        logger.info(scopedLog(LogContext.RECORDING, 'frame streaming started (fallback)'), {
          sessionId,
          strategy: 'polling',
        });
      } catch (fallbackError) {
        logger.error(scopedLog(LogContext.RECORDING, 'polling fallback also failed'), {
          sessionId,
          error: fallbackError instanceof Error ? fallbackError.message : String(fallbackError),
        });
        sessions.delete(sessionId);
      }
    } else {
      logger.error(scopedLog(LogContext.RECORDING, 'frame streaming failed to start'), {
        sessionId,
        strategy: strategy.name,
        error: message,
        fallbackEnabled: config.frameStreaming.fallbackToPolling,
      });
      sessions.delete(sessionId);
    }
  }
}
