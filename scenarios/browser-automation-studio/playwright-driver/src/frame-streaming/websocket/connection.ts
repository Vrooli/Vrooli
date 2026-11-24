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

type WebSocketOptions = { headers?: Record<string, string> };
const WS: new (url: string, options?: WebSocketOptions) => FrameWebSocket =
  WebSocket as new (url: string, options?: WebSocketOptions) => FrameWebSocket;

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
  routedTestMode?: boolean;
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
  private readonly routedTestMode: boolean;
  private reconnectTimer: ReturnType<typeof setTimeout> | undefined;

  constructor(options: WebSocketConnectionOptions) {
    this.sessionId = options.sessionId;
    this.reconnectDelayMs = options.reconnectDelayMs ?? WS_RECONNECT_DELAY_MS;
    this.routedTestMode = options.routedTestMode === true;
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
    this.clearReconnectTimer();

    try {
      const previous = this.state.ws;
      const ws = this.routedTestMode
        ? new WS(this.state.url, { headers: { 'X-Vrooli-Test-Mode': '1' } })
        : new WS(this.state.url);
      this.state.ws = ws;
      this.state.isReady = false;
      const ownsSocket = (): boolean => this.state.ws === ws;

      // Replacing a connection must release the old transport. Its listeners
      // are owner-fenced below, so its close event cannot schedule another retry.
      if (previous && previous !== ws && previous.readyState < 2) {
        try {
          previous.close();
        } catch {
          // Continue with the new socket if the retired transport rejects close.
        }
      }

      ws.on('open', () => {
        if (!ownsSocket()) return;
        this.state.isReady = true;
        logger.info(scopedLog(LogContext.RECORDING, 'frame WebSocket connected'), {
          sessionId: this.sessionId,
        });
      });

      ws.on('close', () => {
        if (!ownsSocket()) return;
        this.state.isReady = false;
        logger.debug(scopedLog(LogContext.RECORDING, 'frame WebSocket closed'), {
          sessionId: this.sessionId,
        });

        this.scheduleReconnect(ownsSocket);
      });

      ws.on('error', (err: Error) => {
        if (!ownsSocket()) return;
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

      this.scheduleReconnect();
    }
  }

  /**
   * Close the WebSocket connection and stop reconnection attempts.
   */
  close(): void {
    this.state.isActive = false;
    this.state.isReady = false;
    this.clearReconnectTimer();

    if (this.state.ws) {
      try {
        this.state.ws.close();
      } catch {
        // Ignore close errors
      }
      this.state.ws = null;
    }
  }

  private scheduleReconnect(isCurrent: () => boolean = () => true): void {
    if (!this.state.isActive || this.reconnectTimer !== undefined) return;
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = undefined;
      if (this.state.isActive && isCurrent()) this.connect();
    }, this.reconnectDelayMs);
  }

  private clearReconnectTimer(): void {
    if (this.reconnectTimer === undefined) return;
    clearTimeout(this.reconnectTimer);
    this.reconnectTimer = undefined;
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
