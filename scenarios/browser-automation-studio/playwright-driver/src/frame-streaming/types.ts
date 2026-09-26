/**
 * Frame Streaming Types
 *
 * Type definitions and constants for the frame streaming module.
 *
 * @module frame-streaming/types
 */

import type { FrameSession } from './frame';

// =============================================================================
// WebSocket Types
// =============================================================================

/**
 * Minimal WebSocket interface - subset of ws module for our needs.
 * Allows mocking in tests without importing full ws module.
 */
export interface FrameWebSocket {
  readyState: number;
  /** Bytes waiting in the transport implementation's outbound queue. */
  bufferedAmount?: number;
  send(data: Buffer): void;
  close(): void;
  on(event: 'open', listener: () => void): void;
  on(event: 'close', listener: () => void): void;
  on(event: 'error', listener: (err: Error) => void): void;
}

// =============================================================================
// Options Types
// =============================================================================

/**
 * Options for starting frame streaming.
 */
export interface FrameStreamOptions {
  /** Callback URL to derive WebSocket URL from */
  callbackUrl: string;
  /** Frame quality 1-100 (default: 65) */
  quality?: number;
  /** Target FPS 1-60 (default: 30) */
  fps?: number;
  /** Screenshot scale: 'css' for 1x (default), 'device' for devicePixelRatio */
  scale?: 'css' | 'device';
  /** Carries the API request's test-storage routing context to the frame socket. */
  routedTestMode?: boolean;
}

/**
 * Options for updating frame stream settings.
 */
export interface FrameStreamUpdateOptions {
  /** New quality value 1-100 */
  quality?: number;
  /** New target FPS 1-60 */
  fps?: number;
  /** Whether to include performance headers in frames */
  perfMode?: boolean;
}

/**
 * Current frame stream settings.
 */
export interface FrameStreamSettings {
  quality: number;
  fps: number;
  scale: 'css' | 'device';
  currentFps: number;
  isStreaming: boolean;
  perfMode: boolean;
}

// =============================================================================
// Session Provider Interface
// =============================================================================

/**
 * Interface for accessing session data.
 * Allows frame streaming to be decoupled from SessionManager.
 */
export interface SessionProvider {
  getSession(sessionId: string): FrameSession;
}

// =============================================================================
// Constants
// =============================================================================

/** Max consecutive frame failures before pausing */
export const MAX_FRAME_FAILURES = 3;

/** WebSocket reconnection delay in ms */
export const WS_RECONNECT_DELAY_MS = 1000;

/** Keep at most one configured maximum-size recording frame queued per viewer. */
export const MAX_QUEUED_FRAME_BYTES = 12 * 1024 * 1024 + 4096;

/** How often to log FPS changes (every N frames) */
export const FPS_LOG_INTERVAL = 30;

/** Screenshot timeout - reduced to allow faster adaptation */
export const SCREENSHOT_TIMEOUT_MS = 200;
