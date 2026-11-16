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
  type WebSocketMessageCallback,
} from './WebSocketContext';

const MAX_RECONNECT_ATTEMPTS = 5;
const INITIAL_RECONNECT_DELAY = 1000;
const MAX_RECONNECT_DELAY = 30000;

type RecordingSocketState = {
  socket: WebSocket | null;
  retryTimer: ReturnType<typeof setTimeout> | null;
  retries: number;
  delay: number;
  subscription: unknown | null;
  disposed: boolean;
};

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

function buildRecordingWebSocketUrl(sessionId: string): string {
  if (typeof window === 'undefined') return '';
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  return `${protocol}//${window.location.host}/ws/sessions/${encodeURIComponent(sessionId)}`;
}

export function WebSocketProvider({ children }: { children: React.ReactNode }) {
  const [isConnected, setIsConnected] = useState(false);

  const wsRef = useRef<WebSocket | null>(null);
  const reconnectAttemptsRef = useRef(0);
  const reconnectDelayRef = useRef(INITIAL_RECONNECT_DELAY);
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const messageCallbacksRef = useRef(new Set<WebSocketMessageCallback>());
  const recordingSocketsRef = useRef(new Map<string, RecordingSocketState>());

  const dispatchMessage = useCallback((event: MessageEvent<unknown>) => {
    if (typeof event.data !== 'string') return;
    try {
      const rawMessage: unknown = JSON.parse(event.data);
      const parsed = safeParse(LooseWebSocketMessageSchema, rawMessage, 'WebSocketMessage');
      if (parsed.success) {
        for (const callback of [...messageCallbacksRef.current]) {
          try {
            callback(parsed.data);
          } catch (error) {
            logger.warn('WebSocket subscriber failed', {component: 'WebSocketContext'}, error);
          }
        }
      } else {
        logger.warn('Invalid WebSocket message', {
          component: 'WebSocketContext',
          action: 'onmessage',
          issues: parsed.details.issues,
        });
      }
    } catch (err) {
      logger.warn('Failed to parse WebSocket message', {
        component: 'WebSocketContext',
        action: 'onmessage',
        error: err,
      });
    }
  }, []);

  const connectRecordingSocket = useCallback((sessionId: string, state: RecordingSocketState) => {
    if (state.disposed || state.socket || !state.subscription) return;
    const url = buildRecordingWebSocketUrl(sessionId);
    if (!url) return;
    try {
      const socket = new WebSocket(url);
      state.socket = socket;
      const owns = () => recordingSocketsRef.current.get(sessionId) === state && state.socket === socket;
      socket.onopen = () => {
        if (!owns()) return;
        state.retries = 0;
        state.delay = INITIAL_RECONNECT_DELAY;
        if (state.subscription) socket.send(JSON.stringify(state.subscription));
      };
      socket.onmessage = event => {
        if (!owns()) return;
        if (typeof event.data === 'string') {
          try {
            const message = JSON.parse(event.data) as {type?: unknown};
            if (message.type === 'session_access_revoked') {
              state.disposed = true;
              state.subscription = null;
              if (state.retryTimer) clearTimeout(state.retryTimer);
              state.retryTimer = null;
              state.socket = null;
              recordingSocketsRef.current.delete(sessionId);
              socket.close(1000, 'session access revoked');
            }
          } catch {
            // The normal dispatcher records malformed protocol payloads.
          }
        }
        dispatchMessage(event);
      };
      socket.onclose = event => {
        if (!owns()) return;
        state.socket = null;
        if (state.disposed || !state.subscription || event.code === 1008 || state.retries >= MAX_RECONNECT_ATTEMPTS) {
          recordingSocketsRef.current.delete(sessionId);
          return;
        }
        const delay = state.delay;
        state.delay = Math.min(state.delay * 2, MAX_RECONNECT_DELAY);
        state.retries += 1;
        state.retryTimer = setTimeout(() => {
          state.retryTimer = null;
          connectRecordingSocket(sessionId, state);
        }, delay);
      };
      socket.onerror = event => {
        if (owns()) logger.warn('Recording WebSocket error', {component: 'WebSocketContext', sessionId, event});
      };
    } catch (err) {
      logger.warn('Failed to connect recording WebSocket', {component: 'WebSocketContext', sessionId, err});
    }
  }, [dispatchMessage]);

  const sendRecordingMessage = useCallback((sessionId: string, message: unknown, type: string): boolean => {
    let state = recordingSocketsRef.current.get(sessionId);
    if (type === 'unsubscribe_recording') {
      if (!state) return false;
      state.subscription = null;
      if (state.retryTimer) clearTimeout(state.retryTimer);
      state.retryTimer = null;
      state.disposed = true;
      state.socket?.close(1000, 'recording subscription ended');
      state.socket = null;
      recordingSocketsRef.current.delete(sessionId);
      return true;
    }
    if (!state) {
      state = {socket: null, retryTimer: null, retries: 0, delay: INITIAL_RECONNECT_DELAY, subscription: null, disposed: false};
      recordingSocketsRef.current.set(sessionId, state);
    }
    if (type === 'subscribe_recording') state.subscription = message;
    if (!state.socket) connectRecordingSocket(sessionId, state);
    if (state.socket?.readyState === WebSocket.OPEN) {
      try {
        state.socket.send(JSON.stringify(message));
        return true;
      } catch (err) {
        logger.warn('Failed to send recording WebSocket message', {component: 'WebSocketContext', sessionId, err});
      }
    }
    return type === 'subscribe_recording';
  }, [connectRecordingSocket]);

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
      const owns = () => wsRef.current === ws;

      ws.onopen = () => {
        if (!owns()) return;
        logger.debug('Connected', { component: 'WebSocketContext', action: 'onopen' });
        setIsConnected(true);
        reconnectAttemptsRef.current = 0;
        reconnectDelayRef.current = INITIAL_RECONNECT_DELAY;
      };

      ws.onmessage = (event: MessageEvent<unknown>) => {
        // Frames have a separate viewer-owned transport. This connection owns JSON events.
        if (!owns() || typeof event.data !== 'string') return;

        dispatchMessage(event);
      };

      ws.onclose = (event) => {
        if (!owns()) return;
        wsRef.current = null;
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
            reconnectTimeoutRef.current = null;
            connect();
          }, delay);
        } else {
          logger.warn('Max reconnect attempts reached', { component: 'WebSocketContext', action: 'reconnect' });
        }
      };

      ws.onerror = (event) => {
        if (!owns()) return;
        logger.warn('WebSocket error', { component: 'WebSocketContext', action: 'onerror', event });
      };
    } catch (err) {
      logger.error('Failed to connect', { component: 'WebSocketContext', action: 'connect', err });
    }
  }, [dispatchMessage]);

  const disconnect = useCallback(() => {
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }

    const socket = wsRef.current;
    wsRef.current = null;
    socket?.close();
    for (const [sessionId, state] of recordingSocketsRef.current) {
      state.disposed = true;
      if (state.retryTimer) clearTimeout(state.retryTimer);
      state.socket?.close();
      recordingSocketsRef.current.delete(sessionId);
    }
    setIsConnected(false);
  }, []);

  const send = useCallback((message: unknown): boolean => {
    if (typeof message === 'object' && message !== null) {
      const candidate = message as {type?: unknown; session_id?: unknown};
      const type = typeof candidate.type === 'string' ? candidate.type : '';
      const sessionId = typeof candidate.session_id === 'string' ? candidate.session_id : '';
      if (sessionId && ['subscribe_recording', 'unsubscribe_recording', 'recording_input'].includes(type)) {
        return sendRecordingMessage(sessionId, message, type);
      }
    }
    if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) {
      logger.warn('Cannot send: WebSocket not connected', { component: 'WebSocketContext', action: 'send' });
      return false;
    }

    try {
      wsRef.current.send(JSON.stringify(message));
      return true;
    } catch (err) {
      logger.warn('Failed to send WebSocket message', { component: 'WebSocketContext', action: 'send', err });
      return false;
    }
  }, [sendRecordingMessage]);

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

  const subscribeToMessages = useCallback((callback: WebSocketMessageCallback) => {
    const callbacks = messageCallbacksRef.current;
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
    send,
    subscribe,
    unsubscribe,
    reconnect,
    subscribeToMessages,
  };

  return <WebSocketContext.Provider value={value}>{children}</WebSocketContext.Provider>;
}
