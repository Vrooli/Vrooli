/**
 * useGraphWebSocket - Real-time invalidation for the operations graph.
 *
 * The WebSocket is treated as a change signal only. All graph data continues
 * to come from the HTTP graph projection endpoint so the API remains the
 * single source of truth.
 */

import { useEffect, useRef, useCallback } from "react";
import { buildWsUrl } from "@vrooli/api-base";
import { useGraphDataStore } from "../stores/graph-data-store";

interface WSMessage {
  type:
    | "full-sync"
    | "node-update"
    | "node-add"
    | "node-remove"
    | "edge-add"
    | "edge-remove"
    | "heartbeat";
  data: unknown;
  timestamp: number;
}

const BACKOFF_BASE_MS = 1000;
const BACKOFF_MAX_MS = 30_000;
const BACKOFF_MULTIPLIER = 2;
const INVALIDATION_DEBOUNCE_MS = 150;
const WS_PATH = "/ws/graph";

export interface UseGraphWebSocketOptions {
  enabled: boolean;
  onNodePulse?: (nodeId: string) => void;
}

function extractNodeId(message: WSMessage): string | null {
  if (message.type !== "node-update" && message.type !== "node-add" && message.type !== "node-remove") {
    return null;
  }

  if (typeof message.data !== "object" || message.data === null) {
    return null;
  }

  const id = (message.data as { id?: unknown }).id;
  return typeof id === "string" ? id : null;
}

export function useGraphWebSocket({ enabled, onNodePulse }: UseGraphWebSocketOptions) {
  const wsRef = useRef<WebSocket | null>(null);
  const retryCountRef = useRef(0);
  const retryTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const invalidateTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const enabledRef = useRef(enabled);
  enabledRef.current = enabled;

  const fetchGraph = useGraphDataStore((s) => s.fetchGraph);

  const scheduleRefresh = useCallback(
    (delay = INVALIDATION_DEBOUNCE_MS) => {
      if (invalidateTimerRef.current) {
        clearTimeout(invalidateTimerRef.current);
      }

      invalidateTimerRef.current = setTimeout(() => {
        invalidateTimerRef.current = null;
        void fetchGraph("operations", { silent: true });
      }, delay);
    },
    [fetchGraph],
  );

  const handleMessage = useCallback(
    (message: WSMessage) => {
      if (message.type === "heartbeat") {
        return;
      }

      const nodeId = extractNodeId(message);
      if (nodeId) {
        onNodePulse?.(nodeId);
      }

      scheduleRefresh();
    },
    [onNodePulse, scheduleRefresh],
  );

  const connect = useCallback(() => {
    if (!enabledRef.current) return;
    if (wsRef.current && wsRef.current.readyState <= WebSocket.OPEN) return;

    const ws = new WebSocket(buildWsUrl(WS_PATH, { appendSuffix: true }));
    wsRef.current = ws;

    ws.onopen = () => {
      retryCountRef.current = 0;
    };

    ws.onmessage = (event) => {
      try {
        handleMessage(JSON.parse(event.data as string) as WSMessage);
      } catch {
        // Ignore malformed messages.
      }
    };

    ws.onclose = () => {
      wsRef.current = null;
      if (!enabledRef.current) return;

      const delay = Math.min(
        BACKOFF_BASE_MS * Math.pow(BACKOFF_MULTIPLIER, retryCountRef.current),
        BACKOFF_MAX_MS,
      );
      retryCountRef.current += 1;

      retryTimerRef.current = setTimeout(() => {
        void fetchGraph("operations", { silent: true }).finally(() => {
          if (enabledRef.current) {
            connect();
          }
        });
      }, delay);
    };
  }, [fetchGraph, handleMessage]);

  const disconnect = useCallback(() => {
    if (retryTimerRef.current) {
      clearTimeout(retryTimerRef.current);
      retryTimerRef.current = null;
    }

    if (invalidateTimerRef.current) {
      clearTimeout(invalidateTimerRef.current);
      invalidateTimerRef.current = null;
    }

    if (wsRef.current) {
      wsRef.current.onclose = null;
      wsRef.current.close();
      wsRef.current = null;
    }

    retryCountRef.current = 0;
  }, []);

  useEffect(() => {
    if (enabled) {
      connect();
    } else {
      disconnect();
    }

    return disconnect;
  }, [enabled, connect, disconnect]);
}
