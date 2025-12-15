/**
 * WebSocket Context
 * Manages real-time WebSocket connection for execution and workflow updates
 *
 * Note: Uses window.location.host to connect through Vite's proxy in development.
 * The proxy forwards /ws to the actual WebSocket server (see vite.config.ts).
 */

import React, { createContext, useContext, useEffect, useRef, useState, useCallback } from 'react';
import { logger } from '../utils/logger';

export interface WebSocketMessage {
  type: string;
  execution_id?: string;
  workflow_id?: string;
  status?: string;
  progress?: number;
  message?: string;
  data?: unknown;
  timestamp?: string;
}

/** Callback type for binary frame subscribers */
export type BinaryFrameCallback = (data: ArrayBuffer) => void;

interface WebSocketContextValue {
  isConnected: boolean;
  lastMessage: WebSocketMessage | null;
  /** @deprecated Use subscribeToBinaryFrames instead to avoid React re-renders */
  lastBinaryFrame: ArrayBuffer | null;
  send: (message: unknown) => void;
  subscribe: (executionId: string) => void;
  unsubscribe: () => void;
  reconnect: () => void;
  /** Subscribe to binary frames directly without triggering React state updates */
  subscribeToBinaryFrames: (callback: BinaryFrameCallback) => () => void;
}

const WebSocketContext = createContext<WebSocketContextValue | undefined>(undefined);

const MAX_RECONNECT_ATTEMPTS = 5;
const INITIAL_RECONNECT_DELAY = 1000;
const MAX_RECONNECT_DELAY = 30000;

/**
 * Build WebSocket URL using the current host.
 * In development, Vite's proxy forwards /ws to the WebSocket server.
 * In production, the same host serves both UI and WebSocket.
 */
function buildWebSocketUrl(): string {
  if (typeof window === 'undefined') {
    return '';
  }

  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  return `${protocol}//${window.location.host}/ws`;
}

export function WebSocketProvider({ children }: { children: React.ReactNode }) {
  const [isConnected, setIsConnected] = useState(false);
  const [lastMessage, setLastMessage] = useState<WebSocketMessage | null>(null);
  // Keep for backwards compatibility but deprecate
  const [lastBinaryFrame, setLastBinaryFrame] = useState<ArrayBuffer | null>(null);

  const wsRef = useRef<WebSocket | null>(null);
  const reconnectAttemptsRef = useRef(0);
  const reconnectDelayRef = useRef(INITIAL_RECONNECT_DELAY);
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Set of callbacks for binary frame subscribers (avoids React state updates)
  const binaryFrameCallbacksRef = useRef<Set<BinaryFrameCallback>>(new Set());

  const connect = useCallback(() => {
    const wsUrl = buildWebSocketUrl();
    if (!wsUrl) {
      logger.warn('WebSocket URL unavailable', { component: 'WebSocketContext', action: 'connect' });
      return;
    }

    logger.debug('Connecting', { component: 'WebSocketContext', action: 'connect', wsUrl });

    try {
      const ws = new WebSocket(wsUrl);
      wsRef.current = ws;

      ws.onopen = () => {
        logger.debug('Connected', { component: 'WebSocketContext', action: 'onopen' });
        setIsConnected(true);
        reconnectAttemptsRef.current = 0;
        reconnectDelayRef.current = INITIAL_RECONNECT_DELAY;
      };

      // Enable binary message handling
      ws.binaryType = 'arraybuffer';

      ws.onmessage = (event) => {
        // Check if this is a binary message (recording frame)
        if (event.data instanceof ArrayBuffer) {
          // Notify all subscribers directly without triggering React state
          // This is much more efficient for high-frequency frame updates
          const callbacks = binaryFrameCallbacksRef.current;
          if (callbacks.size > 0) {
            callbacks.forEach(callback => {
              try {
                callback(event.data);
              } catch (err) {
                logger.warn('Binary frame callback error', { component: 'WebSocketContext', action: 'onmessage' }, err);
              }
            });
          }
          // Also update state for backwards compatibility (deprecated)
          setLastBinaryFrame(event.data);
          return;
        }

        // Text message (JSON)
        try {
          const message = JSON.parse(event.data) as WebSocketMessage;
          setLastMessage(message);
        } catch (error) {
          logger.warn('Failed to parse WebSocket message', { component: 'WebSocketContext', action: 'onmessage' }, error);
        }
      };

      ws.onerror = (error) => {
        logger.warn('WebSocket error', { component: 'WebSocketContext', action: 'onerror' }, error);
      };

      ws.onclose = () => {
        logger.debug('Disconnected', { component: 'WebSocketContext', action: 'onclose' });
        setIsConnected(false);
        attemptReconnect();
      };
    } catch (error) {
      logger.warn('Failed to create WebSocket connection', { component: 'WebSocketContext', action: 'connect' }, error);
      attemptReconnect();
    }
  }, []);

  const attemptReconnect = useCallback(() => {
    if (reconnectAttemptsRef.current >= MAX_RECONNECT_ATTEMPTS) {
      logger.warn('Max reconnection attempts reached', { component: 'WebSocketContext', action: 'attemptReconnect' });
      return;
    }

    reconnectAttemptsRef.current++;
    const delay = reconnectDelayRef.current;

    logger.debug('Reconnecting', {
      component: 'WebSocketContext',
      action: 'attemptReconnect',
      delayMs: delay,
      attempt: reconnectAttemptsRef.current,
      maxAttempts: MAX_RECONNECT_ATTEMPTS,
    });

    reconnectTimeoutRef.current = setTimeout(() => {
      connect();
      // Exponential backoff
      reconnectDelayRef.current = Math.min(reconnectDelayRef.current * 2, MAX_RECONNECT_DELAY);
    }, delay);
  }, [connect]);

  const send = useCallback((message: unknown) => {
    const ws = wsRef.current;
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify(message));
    } else {
      logger.debug('Cannot send - not connected', { component: 'WebSocketContext', action: 'send' });
    }
  }, []);

  const subscribe = useCallback((executionId: string) => {
    send({ type: 'subscribe', execution_id: executionId });
  }, [send]);

  const unsubscribe = useCallback(() => {
    send({ type: 'unsubscribe' });
  }, [send]);

  const reconnect = useCallback(() => {
    // Close existing connection
    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }

    // Clear any pending reconnection
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }

    // Reset reconnection state
    reconnectAttemptsRef.current = 0;
    reconnectDelayRef.current = INITIAL_RECONNECT_DELAY;

    // Connect
    connect();
  }, [connect]);

  /**
   * Subscribe to binary frames directly without triggering React state updates.
   * This is much more efficient for high-frequency frame updates (~6+ FPS).
   * Returns an unsubscribe function.
   */
  const subscribeToBinaryFrames = useCallback((callback: BinaryFrameCallback) => {
    binaryFrameCallbacksRef.current.add(callback);
    return () => {
      binaryFrameCallbacksRef.current.delete(callback);
    };
  }, []);

  // Initial connection
  useEffect(() => {
    connect();

    // Cleanup on unmount
    return () => {
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
      }
      if (wsRef.current) {
        wsRef.current.close();
      }
    };
  }, [connect]);

  const value: WebSocketContextValue = {
    isConnected,
    lastMessage,
    lastBinaryFrame,
    send,
    subscribe,
    unsubscribe,
    reconnect,
    subscribeToBinaryFrames,
  };

  return <WebSocketContext.Provider value={value}>{children}</WebSocketContext.Provider>;
}

export function useWebSocket() {
  const context = useContext(WebSocketContext);
  if (!context) {
    throw new Error('useWebSocket must be used within a WebSocketProvider');
  }
  return context;
}
