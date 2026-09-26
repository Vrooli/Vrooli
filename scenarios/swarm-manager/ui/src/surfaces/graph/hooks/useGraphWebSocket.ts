/**
 * useGraphWebSocket - Real-time invalidation for graph projections.
 *
 * The WebSocket is treated as a change signal only. All graph data continues
 * to come from the HTTP graph projection endpoint so the API remains the
 * single source of truth.
 */

import { useCallback, useEffect, useRef } from "react";
import { buildWsUrl } from "@vrooli/api-base";
import { useGraphDataStore, type GraphLens } from "../stores/graph-data-store";

interface InvalidateMessage {
  type: "invalidate";
  data: {
    lenses?: GraphLens[];
  };
  timestamp: number;
}

interface NodeMessage {
  type: "node-update" | "node-add" | "node-remove";
  data: {
    id?: string;
  };
  timestamp: number;
}

interface EdgeMessage {
  type: "edge-add" | "edge-remove";
  data: unknown;
  timestamp: number;
}

interface HeartbeatMessage {
  type: "heartbeat";
  data: unknown;
  timestamp: number;
}

type WSMessage = InvalidateMessage | NodeMessage | EdgeMessage | HeartbeatMessage;

const BACKOFF_BASE_MS = 1000;
const BACKOFF_MAX_MS = 30_000;
const BACKOFF_MULTIPLIER = 2;
const INVALIDATION_DEBOUNCE_MS = 150;
// The Plan board refetches a whole cross-entity projection, so its refresh is
// an order of magnitude more expensive than a graph snapshot refetch. A burst
// of mutations — a batch queue, a multi-item decision, an agent run changing
// several items — must collapse into one refetch rather than one per event.
const PLAN_INVALIDATION_DEBOUNCE_MS = 750;

export function invalidationDebounceFor(lens: GraphLens): number {
  return lens === "plan" ? PLAN_INVALIDATION_DEBOUNCE_MS : INVALIDATION_DEBOUNCE_MS;
}
// `buildWsUrl(..., { appendSuffix: true })` contributes the `/ws` prefix.
// This hook should only provide the graph stream path segment.
const WS_PATH = "/graph";

export interface UseGraphWebSocketOptions {
  enabled: boolean;
  lens: GraphLens;
  onNodePulse?: (nodeId: string) => void;
}

function extractNodeId(message: WSMessage): string | null {
  if (message.type !== "node-update" && message.type !== "node-add" && message.type !== "node-remove") {
    return null;
  }

  return typeof message.data.id === "string" ? message.data.id : null;
}

export function affectsLens(message: WSMessage, lens: GraphLens): boolean {
  if (message.type === "heartbeat") {
    return false;
  }

  if (message.type === "invalidate") {
    return Array.isArray(message.data.lenses) && message.data.lenses.includes(lens);
  }

  // Edge changes reshape the topology graph. The Plan board is a column
  // projection over items and gates and never renders an edge, so refetching
  // it on an edge event is work that cannot change what the operator sees.
  if (lens === "plan") {
    return false;
  }

  return message.type === "edge-add" || message.type === "edge-remove";
}

export function useGraphWebSocket({ enabled, lens, onNodePulse }: UseGraphWebSocketOptions) {
  const wsRef = useRef<WebSocket | null>(null);
  const hasOpenedRef = useRef(false);
  const retryCountRef = useRef(0);
  const retryTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const invalidateTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const enabledRef = useRef(enabled);
  const lensRef = useRef(lens);
  enabledRef.current = enabled;
  lensRef.current = lens;

  const fetchGraph = useGraphDataStore((s) => s.fetchGraph);

  const scheduleRefresh = useCallback(
    (delay = invalidationDebounceFor(lensRef.current)) => {
      // Clearing a pending timer is what collapses a burst: only the last
      // event in a window schedules the refetch that actually runs.
      if (invalidateTimerRef.current) {
        clearTimeout(invalidateTimerRef.current);
      }

      invalidateTimerRef.current = setTimeout(() => {
        invalidateTimerRef.current = null;
        void fetchGraph(lensRef.current, { silent: true, force: true });
      }, delay);
    },
    [fetchGraph],
  );

  const handleMessage = useCallback(
    (message: WSMessage) => {
      const nodeId = extractNodeId(message);
      if (nodeId) {
        onNodePulse?.(nodeId);
      }

      if (!affectsLens(message, lensRef.current)) {
        return;
      }

      scheduleRefresh();
    },
    [onNodePulse, scheduleRefresh],
  );

  const connect = useCallback(() => {
    if (!enabledRef.current) return;
    if (wsRef.current && (wsRef.current.readyState === 0 || wsRef.current.readyState === 1)) return;

    const ws = new WebSocket(buildWsUrl(WS_PATH, { appendSuffix: true }));
    wsRef.current = ws;

    ws.onopen = () => {
      hasOpenedRef.current = true;
      retryCountRef.current = 0;
    };

    ws.onmessage = (event) => {
      try {
        const parsed: unknown = JSON.parse(event.data as string);
        if (
          typeof parsed === "object" &&
          parsed !== null &&
          "type" in parsed &&
          typeof (parsed as { type: unknown }).type === "string" &&
          "data" in parsed
        ) {
          handleMessage(parsed as WSMessage);
        }
      } catch {
        // Ignore malformed messages.
      }
    };

    ws.onclose = () => {
      wsRef.current = null;
      if (!enabledRef.current) return;

      const shouldRefreshOnReconnect = hasOpenedRef.current;
      const delay = Math.min(
        BACKOFF_BASE_MS * Math.pow(BACKOFF_MULTIPLIER, retryCountRef.current),
        BACKOFF_MAX_MS,
      );
      retryCountRef.current += 1;

      retryTimerRef.current = setTimeout(() => {
        retryTimerRef.current = null;

        const reconnect = () => {
          if (enabledRef.current) {
            connect();
          }
        };

        if (!shouldRefreshOnReconnect) {
          reconnect();
          return;
        }

        void fetchGraph(lensRef.current, { silent: true, force: true }).finally(reconnect);
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
    hasOpenedRef.current = false;
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
