import { useEffect, useRef, useState, useCallback } from "react";
import { resolveWsBase } from "@vrooli/api-base";
import {
  parseWebSocketMessage,
  type WebSocketMessage,
} from "../lib/webSocketProtocol";
import {
  createWebSocketSubscriptionManager,
  type WebSocketSubscriptionManager,
} from "../lib/webSocketSubscriptions";
import { nextReconnectDelayMs, shouldReconnectAfterClose } from "../lib/webSocketConnection";

export type { WebSocketMessage } from "../lib/webSocketProtocol";

export type ConnectionStatus =
  | "connecting"
  | "connected"
  | "disconnected"
  | "error";

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
  const subscriptionManagerRef = useRef<WebSocketSubscriptionManager | null>(null);
  const reconnectTimeoutRef = useRef<number | null>(null);
  const reconnectAttemptsRef = useRef(0);
  const intentionalCloseRef = useRef(false);
  const onMessageRef = useRef(onMessage);
  const onStatusChangeRef = useRef(onStatusChange);

  if (subscriptionManagerRef.current === null) {
    subscriptionManagerRef.current = createWebSocketSubscriptionManager({
      isOpen: () => wsRef.current?.readyState === WebSocket.OPEN,
      send: (message) => {
        wsRef.current?.send(JSON.stringify(message));
      },
    });
  }

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
    if (
      !enabled ||
      wsRef.current?.readyState === WebSocket.OPEN ||
      wsRef.current?.readyState === WebSocket.CONNECTING
    ) {
      return;
    }

    try {
      updateStatus("connecting");
      setError(null);

      console.log(`[WebSocket] Connecting to ${wsUrl}`);
      const ws = new WebSocket(wsUrl);
      intentionalCloseRef.current = false;
      wsRef.current = ws;

      ws.onopen = () => {
        console.log("[WebSocket] Connected");
        updateStatus("connected");
        setError(null);
        reconnectAttemptsRef.current = 0;
        subscriptionManagerRef.current?.replayDesired();
      };

      ws.onmessage = (event) => {
        try {
          const data: unknown = JSON.parse(String(event.data));
          const normalized = parseWebSocketMessage(data);
          if (!normalized) {
            console.warn("[WebSocket] Ignoring unsupported message");
            return;
          }
          onMessageRef.current?.(normalized);
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
        const socketIsCurrent = wsRef.current === ws;
        if (!socketIsCurrent) {
          return;
        }
        const intentionalClose = intentionalCloseRef.current;
        wsRef.current = null;

        console.log("[WebSocket] Connection closed");
        updateStatus("disconnected");

        // Attempt to reconnect if enabled and within retry limits
        if (shouldReconnectAfterClose({
          enabled,
          intentionalClose,
          socketIsCurrent,
          reconnectAttempts: reconnectAttemptsRef.current,
          maxReconnectAttempts,
        })) {
          reconnectAttemptsRef.current += 1;
          const attempt = reconnectAttemptsRef.current;

          console.log(
            `[WebSocket] Reconnecting in ${reconnectInterval}ms (attempt ${attempt})`
          );

          const delay = nextReconnectDelayMs(reconnectInterval, attempt);

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
      intentionalCloseRef.current = true;
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

  const subscribe = useCallback((runId: string) => {
    subscriptionManagerRef.current?.subscribe(runId);
  }, []);

  const unsubscribe = useCallback((runId: string) => {
    subscriptionManagerRef.current?.unsubscribe(runId);
  }, []);

  const subscribeAll = useCallback(() => {
    subscriptionManagerRef.current?.subscribeAll();
  }, []);

  const unsubscribeAll = useCallback(() => {
    subscriptionManagerRef.current?.unsubscribeAll();
  }, []);

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
