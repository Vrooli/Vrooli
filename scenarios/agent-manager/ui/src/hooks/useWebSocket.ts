import { useEffect, useRef, useState, useCallback } from "react";
import { resolveWsBase } from "@vrooli/api-base";

export type ConnectionStatus =
  | "connecting"
  | "connected"
  | "disconnected"
  | "error";

export interface WebSocketMessage {
  type: string;
  payload: unknown;
  runId?: string;
}

interface UseWebSocketOptions {
  enabled?: boolean;
  reconnectInterval?: number;
  maxReconnectAttempts?: number;
  onMessage?: (message: WebSocketMessage) => void;
  onStatusChange?: (status: ConnectionStatus) => void;
}

interface UseWebSocketReturn {
  status: ConnectionStatus;
  error: Error | null;
  send: (data: unknown) => void;
  subscribe: (runId: string) => void;
  unsubscribe: (runId: string) => void;
  subscribeAll: () => void;
  unsubscribeAll: () => void;
  reconnect: () => void;
}

export function useWebSocket(
  options: UseWebSocketOptions = {}
): UseWebSocketReturn {
  const {
    enabled = true,
    reconnectInterval = 5000,
    maxReconnectAttempts = Infinity,
    onMessage,
    onStatusChange,
  } = options;

  const [status, setStatus] = useState<ConnectionStatus>("disconnected");
  const [error, setError] = useState<Error | null>(null);

  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimeoutRef = useRef<number | null>(null);
  const reconnectAttemptsRef = useRef(0);
  const onMessageRef = useRef(onMessage);
  const onStatusChangeRef = useRef(onStatusChange);

  // Keep refs fresh
  useEffect(() => {
    onMessageRef.current = onMessage;
  }, [onMessage]);

  useEffect(() => {
    onStatusChangeRef.current = onStatusChange;
  }, [onStatusChange]);

  // Resolve WebSocket URL
  const wsUrl = resolveWsBase({
    appendSuffix: true,
    apiSuffix: "/api/v1/ws",
  });

  const updateStatus = useCallback((newStatus: ConnectionStatus) => {
    setStatus(newStatus);
    onStatusChangeRef.current?.(newStatus);
  }, []);

  const connect = useCallback(() => {
    if (!enabled || wsRef.current?.readyState === WebSocket.OPEN) {
      return;
    }

    try {
      updateStatus("connecting");
      setError(null);

      console.log(`[WebSocket] Connecting to ${wsUrl}`);
      const ws = new WebSocket(wsUrl);
      wsRef.current = ws;

      ws.onopen = () => {
        console.log("[WebSocket] Connected");
        updateStatus("connected");
        setError(null);
        reconnectAttemptsRef.current = 0;
      };

      ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data) as WebSocketMessage;
          onMessageRef.current?.(data);
        } catch (err) {
          console.error("[WebSocket] Failed to parse message:", err);
        }
      };

      ws.onerror = () => {
        const errorObj = new Error("WebSocket connection error");
        console.error("[WebSocket]", errorObj);
        setError(errorObj);
        updateStatus("error");
      };

      ws.onclose = () => {
        console.log("[WebSocket] Connection closed");
        updateStatus("disconnected");
        wsRef.current = null;

        // Attempt to reconnect if enabled and within retry limits
        if (
          enabled &&
          reconnectAttemptsRef.current < maxReconnectAttempts
        ) {
          reconnectAttemptsRef.current += 1;
          const attempt = reconnectAttemptsRef.current;

          console.log(
            `[WebSocket] Reconnecting in ${reconnectInterval}ms (attempt ${attempt})`
          );

          // Exponential backoff with jitter
          const backoff = Math.min(
            reconnectInterval * Math.pow(1.5, attempt - 1),
            30000 // Max 30 seconds
          );
          const jitter = Math.random() * 1000;
          const delay = backoff + jitter;

          reconnectTimeoutRef.current = window.setTimeout(() => {
            if (enabled) {
              connect();
            }
          }, delay);
        }
      };
    } catch (err) {
      const errorObj = err instanceof Error ? err : new Error(String(err));
      console.error("[WebSocket] Connection failed:", errorObj);
      setError(errorObj);
      updateStatus("error");
    }
  }, [wsUrl, enabled, reconnectInterval, maxReconnectAttempts, updateStatus]);

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
    updateStatus("disconnected");
  }, [updateStatus]);

  const send = useCallback((data: unknown) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(data));
    } else {
      console.warn("[WebSocket] Cannot send, not connected");
    }
  }, []);

  const subscribe = useCallback(
    (runId: string) => {
      send({ type: "subscribe", payload: { runId } });
    },
    [send]
  );

  const unsubscribe = useCallback(
    (runId: string) => {
      send({ type: "unsubscribe", payload: { runId } });
    },
    [send]
  );

  const subscribeAll = useCallback(() => {
    send({ type: "subscribe_all" });
  }, [send]);

  const unsubscribeAll = useCallback(() => {
    send({ type: "unsubscribe_all" });
  }, [send]);

  const reconnect = useCallback(() => {
    disconnect();
    connect();
  }, [disconnect, connect]);

  useEffect(() => {
    if (enabled) {
      connect();
    }

    return () => {
      disconnect();
    };
  }, [enabled, connect, disconnect]);

  return {
    status,
    error,
    send,
    subscribe,
    unsubscribe,
    subscribeAll,
    unsubscribeAll,
    reconnect,
  };
}
