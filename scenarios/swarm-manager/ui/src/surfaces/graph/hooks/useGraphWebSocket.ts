/**
 * useGraphWebSocket - Real-time graph updates via WebSocket.
 *
 * Connects to /ws/graph when enabled (Operations lens active).
 * Applies incoming messages to the graph-data-store.
 * Reconnects with exponential backoff on disconnect.
 * Falls back to full HTTP fetch + full-replace on reconnect.
 */

import { useEffect, useRef, useCallback } from "react";
import { buildWsUrl } from "@vrooli/api-base";
import { buildApiUrl } from "@vrooli/api-base";
import { useGraphDataStore } from "../stores/graph-data-store";
import type { Node, Edge } from "@xyflow/react";

/** WebSocket message from the server. */
interface WSMessage {
  type: "full-sync" | "node-update" | "node-add" | "node-remove" | "edge-add" | "edge-remove" | "heartbeat";
  data: unknown;
  timestamp: number;
}

/** Set of node IDs that recently received updates (for pulse animation). */
export type PulsedNodes = Set<string>;

const BACKOFF_BASE_MS = 1000;
const BACKOFF_MAX_MS = 30_000;
const BACKOFF_MULTIPLIER = 2;
const WS_PATH = "/ws/graph";
const GRAPH_API_PATH = "/graph";

export interface UseGraphWebSocketOptions {
  enabled: boolean;
  onNodePulse?: (nodeId: string) => void;
}

export function useGraphWebSocket({ enabled, onNodePulse }: UseGraphWebSocketOptions) {
  const wsRef = useRef<WebSocket | null>(null);
  const retryCountRef = useRef(0);
  const retryTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const enabledRef = useRef(enabled);
  enabledRef.current = enabled;

  const setGraphData = useGraphDataStore((s) => s.setGraphData);
  const setNodes = useGraphDataStore((s) => s.setNodes);
  const setEdges = useGraphDataStore((s) => s.setEdges);

  /** Full-fetch from HTTP API and replace store. */
  const fullFetch = useCallback(async () => {
    try {
      const url = buildApiUrl(`${GRAPH_API_PATH}?lens=operations`, { appendSuffix: true });
      const res = await fetch(url, { cache: "no-store" });
      if (!res.ok) return;
      const data = await res.json() as { nodes?: Node[]; edges?: Edge[] };
      if (data.nodes && data.edges) {
        setGraphData(data.nodes, data.edges);
      }
    } catch {
      // Silently fail — the WS will keep trying to reconnect.
    }
  }, [setGraphData]);

  /** Process an incoming WebSocket message. */
  const handleMessage = useCallback((msg: WSMessage) => {
    switch (msg.type) {
      case "full-sync": {
        const syncData = msg.data as { nodes?: Node[]; edges?: Edge[] } | undefined;
        if (syncData?.nodes && syncData?.edges) {
          setGraphData(syncData.nodes, syncData.edges);
        }
        break;
      }
      case "node-update": {
        const updatedNode = msg.data as Node | undefined;
        if (!updatedNode?.id) break;
        onNodePulse?.(updatedNode.id);
        useGraphDataStore.setState((state) => ({
          nodes: state.nodes.map((n) =>
            n.id === updatedNode.id ? { ...n, data: { ...n.data, ...(updatedNode.data as Record<string, unknown>) } } : n,
          ),
        }));
        break;
      }
      case "node-add": {
        const newNode = msg.data as Node | undefined;
        if (!newNode?.id) break;
        onNodePulse?.(newNode.id);
        useGraphDataStore.setState((state) => ({
          nodes: [...state.nodes.filter((n) => n.id !== newNode.id), newNode],
        }));
        break;
      }
      case "node-remove": {
        const removedNode = msg.data as { id?: string } | undefined;
        if (!removedNode?.id) break;
        useGraphDataStore.setState((state) => ({
          nodes: state.nodes.filter((n) => n.id !== removedNode.id),
        }));
        break;
      }
      case "edge-add": {
        const newEdge = msg.data as Edge | undefined;
        if (!newEdge?.id) break;
        useGraphDataStore.setState((state) => ({
          edges: [...state.edges.filter((e) => e.id !== newEdge.id), newEdge],
        }));
        break;
      }
      case "edge-remove": {
        const removedEdge = msg.data as { id?: string } | undefined;
        if (!removedEdge?.id) break;
        useGraphDataStore.setState((state) => ({
          edges: state.edges.filter((e) => e.id !== removedEdge.id),
        }));
        break;
      }
      case "heartbeat":
        // No-op — connection keepalive.
        break;
    }
  }, [setGraphData, onNodePulse]);

  /** Connect to the WebSocket. */
  const connect = useCallback(() => {
    if (!enabledRef.current) return;
    if (wsRef.current && wsRef.current.readyState <= WebSocket.OPEN) return;

    const url = buildWsUrl(WS_PATH, { appendSuffix: true });
    const ws = new WebSocket(url);
    wsRef.current = ws;

    ws.onopen = () => {
      retryCountRef.current = 0;
    };

    ws.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data as string) as WSMessage;
        handleMessage(msg);
      } catch {
        // Ignore malformed messages.
      }
    };

    ws.onclose = () => {
      wsRef.current = null;
      if (!enabledRef.current) return;

      // Exponential backoff reconnect.
      const delay = Math.min(
        BACKOFF_BASE_MS * Math.pow(BACKOFF_MULTIPLIER, retryCountRef.current),
        BACKOFF_MAX_MS,
      );
      retryCountRef.current++;

      retryTimerRef.current = setTimeout(() => {
        // Full-fetch before reconnecting to ensure state is fresh.
        void fullFetch().then(() => {
          if (enabledRef.current) connect();
        });
      }, delay);
    };

    ws.onerror = () => {
      // onclose will fire after onerror — reconnection handled there.
    };
  }, [handleMessage, fullFetch]);

  /** Disconnect and cleanup. */
  const disconnect = useCallback(() => {
    if (retryTimerRef.current) {
      clearTimeout(retryTimerRef.current);
      retryTimerRef.current = null;
    }
    if (wsRef.current) {
      wsRef.current.onclose = null; // Prevent reconnect on intentional close.
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
