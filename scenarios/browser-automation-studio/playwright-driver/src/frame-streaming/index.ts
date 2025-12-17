/**
 * Frame Streaming Module
 *
 * Provides frame streaming capabilities for recording sessions.
 * Supports CDP screencast (preferred) and polling-based capture (fallback).
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
  FrameStreamState,
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
