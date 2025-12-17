/**
 * WebSocket Connection Manager
 *
 * Handles WebSocket lifecycle for frame streaming:
 * - Connection establishment
 * - Automatic reconnection
 * - Connection state tracking
 *
 * @module frame-streaming/websocket/connection
 */

import WebSocket from 'ws';
import { logger, scopedLog, LogContext } from '../../utils';
import type { FrameWebSocket } from '../types';
import { WS_RECONNECT_DELAY_MS } from '../types';

// Type cast for WebSocket constructor (ws module types are complex)
const WS: new (url: string) => FrameWebSocket = WebSocket as unknown as new (url: string) => FrameWebSocket;

/**
 * WebSocket connection state.
 */
export interface WebSocketConnectionState {
  /** Current WebSocket instance (may be null) */
  ws: FrameWebSocket | null;
  /** WebSocket URL */
  url: string;
  /** Whether WebSocket is connected and ready */
  isReady: boolean;
  /** Whether connection manager is active (not stopped) */
  isActive: boolean;
}

/**
 * Options for creating a WebSocket connection manager.
 */
export interface WebSocketConnectionOptions {
  /** WebSocket URL */
  url: string;
  /** Session ID for logging */
  sessionId: string;
  /** Reconnection delay in ms (default: WS_RECONNECT_DELAY_MS) */
  reconnectDelayMs?: number;
}

/**
 * WebSocket connection manager.
 *
 * Manages WebSocket lifecycle with automatic reconnection.
 */
export class WebSocketConnectionManager {
  private state: WebSocketConnectionState;
  private readonly sessionId: string;
  private readonly reconnectDelayMs: number;

  constructor(options: WebSocketConnectionOptions) {
    this.sessionId = options.sessionId;
    this.reconnectDelayMs = options.reconnectDelayMs ?? WS_RECONNECT_DELAY_MS;
    this.state = {
      ws: null,
      url: options.url,
      isReady: false,
      isActive: true,
    };
  }

  /**
   * Get current WebSocket (may be null if not connected).
   */
  getWebSocket(): FrameWebSocket | null {
    return this.state.ws;
  }

  /**
   * Check if WebSocket is ready to send data.
   */
  isReady(): boolean {
    return this.state.isReady && this.state.ws !== null && this.state.ws.readyState === 1;
  }

  /**
   * Check if connection manager is active (not stopped).
   */
  isActive(): boolean {
    return this.state.isActive;
  }

  /**
   * Connect to the WebSocket server.
   * Automatically handles reconnection on close.
   */
  connect(): void {
    if (!this.state.isActive) return;

    try {
      const ws = new WS(this.state.url);
      this.state.ws = ws;

      ws.on('open', () => {
        this.state.isReady = true;
        logger.info(scopedLog(LogContext.RECORDING, 'frame WebSocket connected'), {
          sessionId: this.sessionId,
        });
      });

      ws.on('close', () => {
        this.state.isReady = false;
        logger.debug(scopedLog(LogContext.RECORDING, 'frame WebSocket closed'), {
          sessionId: this.sessionId,
        });

        // Reconnect if still active
        if (this.state.isActive) {
          setTimeout(() => this.connect(), this.reconnectDelayMs);
        }
      });

      ws.on('error', (err: Error) => {
        this.state.isReady = false;
        logger.warn(scopedLog(LogContext.RECORDING, 'frame WebSocket error'), {
          sessionId: this.sessionId,
          error: err.message,
        });
      });
    } catch (err) {
      logger.warn(scopedLog(LogContext.RECORDING, 'failed to create frame WebSocket'), {
        sessionId: this.sessionId,
        error: err instanceof Error ? err.message : String(err),
      });

      // Retry connection
      if (this.state.isActive) {
        setTimeout(() => this.connect(), this.reconnectDelayMs);
      }
    }
  }

  /**
   * Close the WebSocket connection and stop reconnection attempts.
   */
  close(): void {
    this.state.isActive = false;
    this.state.isReady = false;

    if (this.state.ws) {
      try {
        this.state.ws.close();
      } catch {
        // Ignore close errors
      }
      this.state.ws = null;
    }
  }
}

/**
 * Build WebSocket URL from HTTP callback URL.
 *
 * Converts http://host:port/api/v1/recordings/live/{sessionId}/frame
 * to ws://host:port/ws/recording/{sessionId}/frames
 */
export function buildWebSocketUrl(callbackUrl: string, sessionId: string): string {
  try {
    const url = new URL(callbackUrl);
    const protocol = url.protocol === 'https:' ? 'wss:' : 'ws:';
    return `${protocol}//${url.host}/ws/recording/${sessionId}/frames`;
  } catch {
    // Fallback: assume localhost API
    return `ws://127.0.0.1:8080/ws/recording/${sessionId}/frames`;
  }
}

/**
 * Create a WebSocket connection manager.
 */
export function createWebSocketConnectionManager(
  options: WebSocketConnectionOptions
): WebSocketConnectionManager {
  return new WebSocketConnectionManager(options);
}
