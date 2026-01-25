/**
 * WebSocket Provider
 * Manages WebSocket connection for execution and workflow updates.
 *
 * Note: Uses window.location.host to connect through Vite's proxy in development.
 * The proxy forwards /ws to the actual WebSocket server (see vite.config.ts).
 */

import React, { useCallback, useEffect, useRef, useState } from 'react';
import { logger } from '../utils/logger';
import { safeParse } from '../shared/api/safeParse';
import { LooseWebSocketMessageSchema } from '../shared/api/schemas';
import {
  WebSocketContext,
  type BinaryFrameCallback,
  type WebSocketMessage,
} from './WebSocketContext';

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

      ws.onmessage = (event: MessageEvent<string | ArrayBuffer>) => {
        // Check if this is a binary message (recording frame)
        if (event.data instanceof ArrayBuffer) {
          // Notify all subscribers directly without triggering React state
          // This is much more efficient for high-frequency frame updates
          const callbacks = binaryFrameCallbacksRef.current;
          if (callbacks.size > 0) {
            callbacks.forEach((callback) => {
              try {
                callback(event.data);
              } catch (err) {
                logger.warn('Binary frame callback error', { component: 'WebSocketContext', action: 'onmessage' }, err);
              }
            });
          }
          // Note: Removed setLastBinaryFrame() - it caused React re-renders on every frame
          // which severely degraded streaming performance. Use subscribeToBinaryFrames() instead.
          return;
        }

        if (typeof event.data !== 'string') {
          logger.warn('Unsupported WebSocket message type', { component: 'WebSocketContext', action: 'onmessage' });
          return;
        }

        // Text message (JSON)
        try {
          const rawMessage: unknown = JSON.parse(event.data);
          const parsed = safeParse(LooseWebSocketMessageSchema, rawMessage, 'WebSocketMessage');
          if (parsed.success) {
            setLastMessage(parsed.data);
          } else {
            logger.warn('Invalid WebSocket message', {
              component: 'WebSocketContext',
              action: 'onmessage',
              issues: parsed.issues,
            });
          }
        } catch (err) {
          logger.warn('Failed to parse WebSocket message', {
            component: 'WebSocketContext',
            action: 'onmessage',
            error: err,
          });
        }
      };

      ws.onclose = (event) => {
        logger.debug('Disconnected', {
          component: 'WebSocketContext',
          action: 'onclose',
          code: event.code,
          reason: event.reason,
        });
        setIsConnected(false);

        // Attempt to reconnect with exponential backoff
        if (reconnectAttemptsRef.current < MAX_RECONNECT_ATTEMPTS) {
          const delay = reconnectDelayRef.current;
          reconnectDelayRef.current = Math.min(reconnectDelayRef.current * 2, MAX_RECONNECT_DELAY);
          reconnectAttemptsRef.current += 1;

          logger.debug('Reconnecting', {
            component: 'WebSocketContext',
            action: 'reconnect',
            attempt: reconnectAttemptsRef.current,
            delay,
          });

          reconnectTimeoutRef.current = setTimeout(() => {
            connect();
          }, delay);
        } else {
          logger.warn('Max reconnect attempts reached', { component: 'WebSocketContext', action: 'reconnect' });
        }
      };

      ws.onerror = (event) => {
        logger.warn('WebSocket error', { component: 'WebSocketContext', action: 'onerror', event });
      };
    } catch (err) {
      logger.error('Failed to connect', { component: 'WebSocketContext', action: 'connect', err });
    }
  }, []);

  const disconnect = useCallback(() => {
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }

    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }
  }, []);

  const send = useCallback((message: unknown) => {
    if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) {
      logger.warn('Cannot send: WebSocket not connected', { component: 'WebSocketContext', action: 'send' });
      return;
    }

    try {
      wsRef.current.send(JSON.stringify(message));
    } catch (err) {
      logger.warn('Failed to send WebSocket message', { component: 'WebSocketContext', action: 'send', err });
    }
  }, []);

  const subscribe = useCallback((executionId: string) => {
    send({ type: 'subscribe', execution_id: executionId });
  }, [send]);

  const unsubscribe = useCallback(() => {
    send({ type: 'unsubscribe' });
  }, [send]);

  const reconnect = useCallback(() => {
    reconnectAttemptsRef.current = 0;
    reconnectDelayRef.current = INITIAL_RECONNECT_DELAY;
    disconnect();
    connect();
  }, [connect, disconnect]);

  const subscribeToBinaryFrames = useCallback((callback: BinaryFrameCallback) => {
    const callbacks = binaryFrameCallbacksRef.current;
    callbacks.add(callback);

    return () => {
      callbacks.delete(callback);
    };
  }, []);

  useEffect(() => {
    connect();

    return () => {
      disconnect();
    };
  }, [connect, disconnect]);

  const value = {
    isConnected,
    lastMessage,
    lastBinaryFrame: null,
    send,
    subscribe,
    unsubscribe,
    reconnect,
    subscribeToBinaryFrames,
  };

  return <WebSocketContext.Provider value={value}>{children}</WebSocketContext.Provider>;
}
