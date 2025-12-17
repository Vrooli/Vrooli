/**
 * Frame Streaming Module
 *
 * Provides frame streaming capabilities for recording sessions.
 * Supports CDP screencast (preferred) and polling-based capture (fallback).
 *
 * Architecture:
 * - manager.ts - Thin orchestrator
 * - strategies/ - Frame capture implementations (CDP screencast, polling)
 * - websocket/ - WebSocket connection management
 *
 * @module frame-streaming
 */

// Public API
export {
  startFrameStreaming,
  stopFrameStreaming,
  updateFrameStreamSettings,
  getFrameStreamSettings,
} from './manager';

// Types
export type {
  FrameStreamOptions,
  FrameStreamUpdateOptions,
  FrameStreamSettings,
  FrameWebSocket,
  SessionProvider,
} from './types';

// Constants (exported for testing)
export {
  MAX_FRAME_FAILURES,
  WS_RECONNECT_DELAY_MS,
  FPS_LOG_INTERVAL,
  SCREENSHOT_TIMEOUT_MS,
} from './types';

// Strategy types (for advanced usage)
export type {
  FrameStreamingStrategy,
  StreamingStrategyConfig,
  StreamingHandle,
  WebSocketProvider,
  FrameStatsReporter,
  FrameStats,
} from './strategies';

// Strategy factories (for testing/custom usage)
export { createCdpScreencastStrategy, createPollingStrategy } from './strategies';

// WebSocket utilities
export { buildWebSocketUrl } from './websocket';
