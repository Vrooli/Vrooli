/**
 * Frame Streaming Types
 *
 * Type definitions and constants for the frame streaming module.
 *
 * @module frame-streaming/types
 */

import type { Page } from 'playwright';
import type { PerfCollector } from '../performance';
import type { FpsControllerState, FpsControllerConfig } from '../fps';
import type { ScreencastHandle } from '../routes/record-mode/screencast-streaming';

// =============================================================================
// WebSocket Types
// =============================================================================

/**
 * Minimal WebSocket interface - subset of ws module for our needs.
 * Allows mocking in tests without importing full ws module.
 */
export interface FrameWebSocket {
  readyState: number;
  send(data: Buffer): void;
  close(): void;
  on(event: 'open', listener: () => void): void;
  on(event: 'close', listener: () => void): void;
  on(event: 'error', listener: (err: Error) => void): void;
}

// =============================================================================
// Frame Streaming State
// =============================================================================

/**
 * Frame streaming state for a session.
 * Uses WebSocket for efficient binary frame delivery.
 */
export interface FrameStreamState {
  /** Whether streaming is active */
  isStreaming: boolean;
  /** AbortController to stop the streaming loop */
  abortController: AbortController;
  /** Last frame buffer for quick byte-level comparison */
  lastFrameBuffer: Buffer | null;
  /** Frame quality (0-100) */
  quality: number;
  /** Target FPS (user-requested) */
  targetFps: number;
  /** Number of frames sent (for logging/metrics) */
  frameCount: number;
  /** WebSocket connection to API */
  ws: FrameWebSocket | null;
  /** WebSocket URL for reconnection */
  wsUrl: string;
  /** Consecutive failure count for circuit breaker */
  consecutiveFailures: number;
  /** Whether WebSocket is connected and ready */
  wsReady: boolean;
  /** Screenshot scale: 'css' for 1x, 'device' for devicePixelRatio */
  scale: 'css' | 'device';
  /** Performance collector - always created for runtime perf mode toggling */
  perfCollector: PerfCollector;
  /** Whether to include timing headers in frames (toggled via UI) */
  includePerfHeaders: boolean;
  /** FPS controller state (handles adaptive FPS logic) - only used for polling mode */
  fpsState: FpsControllerState;
  /** FPS controller config - only used for polling mode */
  fpsConfig: FpsControllerConfig;
  /** Whether using CDP screencast (vs legacy polling) */
  useScreencast: boolean;
  /** Handle to stop screencast and cleanup - only set when useScreencast is true */
  screencastHandle?: ScreencastHandle;
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
  getSession(sessionId: string): { page: Page };
}

// =============================================================================
// Constants
// =============================================================================

/** Max consecutive frame failures before pausing */
export const MAX_FRAME_FAILURES = 3;

/** WebSocket reconnection delay in ms */
export const WS_RECONNECT_DELAY_MS = 1000;

/** How often to log FPS changes (every N frames) */
export const FPS_LOG_INTERVAL = 30;

/** Screenshot timeout - reduced to allow faster adaptation */
export const SCREENSHOT_TIMEOUT_MS = 200;
