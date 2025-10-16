import { useEffect, useRef, useState, useCallback } from 'react';
import type { WebSocketEvent } from '../types/events';

export type ConnectionStatus = 'connecting' | 'connected' | 'disconnected' | 'error';

interface UseWebSocketOptions {
  url: string;
  onEvent: (event: WebSocketEvent) => void;
  enabled?: boolean;
  reconnectInterval?: number;
  maxReconnectAttempts?: number;
}

interface UseWebSocketReturn {
  status: ConnectionStatus;
  error: Error | null;
  reconnectAttempts: number;
}

export function useWebSocket({
  url,
  onEvent,
  enabled = true,
  reconnectInterval = 5000,
  maxReconnectAttempts = Infinity,
}: UseWebSocketOptions): UseWebSocketReturn {
  const [status, setStatus] = useState<ConnectionStatus>('disconnected');
  const [error, setError] = useState<Error | null>(null);
  const [reconnectAttempts, setReconnectAttempts] = useState(0);

  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimeoutRef = useRef<number | null>(null);
  const isMountedRef = useRef(true);
  const reconnectAttemptsRef = useRef(0);
  const onEventRef = useRef(onEvent);
  const reconnectIntervalRef = useRef(reconnectInterval);
  const maxReconnectAttemptsRef = useRef(maxReconnectAttempts);
  const enabledRef = useRef(enabled);

  // Keep refs fresh
  useEffect(() => {
    onEventRef.current = onEvent;
    reconnectIntervalRef.current = reconnectInterval;
    maxReconnectAttemptsRef.current = maxReconnectAttempts;
    enabledRef.current = enabled;
  }, [onEvent, reconnectInterval, maxReconnectAttempts, enabled]);

  const connect = useCallback(() => {
    if (!enabledRef.current || wsRef.current?.readyState === WebSocket.OPEN) {
      return;
    }

    try {
      setStatus('connecting');
      setError(null);

      const ws = new WebSocket(url);
      wsRef.current = ws;

      ws.onopen = () => {
        if (!isMountedRef.current) return;
        console.log('[WebSocket] Connected');
        setStatus('connected');
        setError(null);
        reconnectAttemptsRef.current = 0;
        setReconnectAttempts(0);
      };

      ws.onmessage = (event) => {
        if (!isMountedRef.current) return;

        try {
          const messages = event.data.split('\n').filter(Boolean);
          for (const message of messages) {
            const parsed = JSON.parse(message) as WebSocketEvent;
            onEventRef.current(parsed);
          }
        } catch (err) {
          console.error('[WebSocket] Failed to parse message:', err);
        }
      };

      ws.onerror = (evt) => {
        if (!isMountedRef.current) return;
        console.warn('[WebSocket] Error:', evt);
        const errorObj = new Error('WebSocket connection error');
        setError(errorObj);
        setStatus('error');
      };

      ws.onclose = (evt) => {
        if (!isMountedRef.current) return;
        console.log('[WebSocket] Disconnected:', evt.code, evt.reason);
        setStatus('disconnected');
        wsRef.current = null;

        // Attempt to reconnect if enabled and within retry limits
        if (enabledRef.current && reconnectAttemptsRef.current < maxReconnectAttemptsRef.current) {
          reconnectAttemptsRef.current += 1;
          setReconnectAttempts(reconnectAttemptsRef.current);

          // Exponential backoff with jitter
          const backoff = Math.min(
            reconnectIntervalRef.current * Math.pow(1.5, reconnectAttemptsRef.current - 1),
            30000, // Max 30 seconds
          );
          const jitter = Math.random() * 1000;
          const delay = backoff + jitter;

          console.log(
            `[WebSocket] Reconnecting in ${(delay / 1000).toFixed(1)}s (attempt ${reconnectAttemptsRef.current})`,
          );

          reconnectTimeoutRef.current = window.setTimeout(() => {
            if (isMountedRef.current && enabledRef.current) {
              connect();
            }
          }, delay);
        }
      };
    } catch (err) {
      console.error('[WebSocket] Failed to create connection:', err);
      setError(err instanceof Error ? err : new Error(String(err)));
      setStatus('error');
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [url]);

  const disconnect = useCallback(() => {
    if (reconnectTimeoutRef.current !== null) {
      clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }

    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }

    reconnectAttemptsRef.current = 0;
    setReconnectAttempts(0);
    setStatus('disconnected');
  }, []);

  useEffect(() => {
    isMountedRef.current = true;

    if (enabled) {
      connect();
    }

    return () => {
      isMountedRef.current = false;
      disconnect();
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [enabled, url]);

  return {
    status,
    error,
    reconnectAttempts,
  };
}
