import { useEffect, useRef, useState, useCallback } from "react";
import { resolveWsBase } from "@vrooli/api-base";
import { create, fromJson, toJson } from "@bufbuild/protobuf";
import type { RunEvent, Task } from "../types";
import {
  AgentManagerWsClientMessageSchema,
  AgentManagerWsClientMessageType,
  AgentManagerWsMessageSchema,
  AgentManagerWsMessageType,
} from "@vrooli/proto-types/agent-manager/v1/domain/events_pb";

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

export type MessageHandler = (message: WebSocketMessage) => void;

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
  addMessageHandler: (handler: MessageHandler) => void;
  removeMessageHandler: (handler: MessageHandler) => void;
}

const protoReadOptions = { ignoreUnknownFields: true, protoFieldName: true };
const protoWriteOptions = { useProtoFieldName: true };

function parseProtoMessage(raw: unknown): WebSocketMessage | null {
  const message = fromJson(AgentManagerWsMessageSchema, raw as any, protoReadOptions);
  switch (message.type) {
    case AgentManagerWsMessageType.RUN_EVENT:
      if (message.payload.case !== "runEvent") return null;
      return {
        type: "run_event",
        runId: message.runId,
        payload: message.payload.value as RunEvent,
      };
    case AgentManagerWsMessageType.RUN_STATUS:
      if (message.payload.case !== "runStatus") return null;
      {
        const runId = message.payload.value.runId || message.runId;
        if (!runId) return null;
        return {
          type: "run_status",
          runId,
          payload: { id: runId, status: message.payload.value.status },
        };
      }
    case AgentManagerWsMessageType.TASK_STATUS:
      if (message.payload.case !== "taskStatus") return null;
      {
        const taskId = message.payload.value.taskId;
        if (!taskId) return null;
        return {
          type: "task_status",
          payload: { id: taskId, status: message.payload.value.status } as Partial<Task>,
        };
      }
    case AgentManagerWsMessageType.RUN_PROGRESS:
      if (message.payload.case !== "runProgress") return null;
      return {
        type: "run_progress",
        runId: message.runId,
        payload: message.payload.value,
      };
    case AgentManagerWsMessageType.CONNECTED:
      if (message.payload.case !== "connected") return null;
      return {
        type: "connected",
        payload: message.payload.value,
      };
    case AgentManagerWsMessageType.PONG:
      if (message.payload.case !== "pong") return null;
      return {
        type: "pong",
        payload: message.payload.value,
      };
    default:
      return null;
  }
}

function buildClientMessage(type: AgentManagerWsClientMessageType, runId?: string) {
  const payload = runId
    ? { payload: { case: "runSubscription" as const, value: { runId } } }
    : undefined;
  const message = create(AgentManagerWsClientMessageSchema, {
    type,
    ...(payload ?? {}),
  });
  return toJson(AgentManagerWsClientMessageSchema, message, protoWriteOptions) as Record<string, unknown>;
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
  const messageHandlersRef = useRef<Set<MessageHandler>>(new Set());

  // Keep refs fresh
  useEffect(() => {
    onMessageRef.current = onMessage;
  }, [onMessage]);

  useEffect(() => {
    onStatusChangeRef.current = onStatusChange;
  }, [onStatusChange]);

  const addMessageHandler = useCallback((handler: MessageHandler) => {
    messageHandlersRef.current.add(handler);
  }, []);

  const removeMessageHandler = useCallback((handler: MessageHandler) => {
    messageHandlersRef.current.delete(handler);
  }, []);

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
          const data = JSON.parse(event.data);
          const normalized = parseProtoMessage(data) ?? (data as WebSocketMessage);
          onMessageRef.current?.(normalized);
          // Call all registered message handlers
          messageHandlersRef.current.forEach((handler) => {
            try {
              handler(normalized);
            } catch (handlerErr) {
              console.error("[WebSocket] Handler error:", handlerErr);
            }
          });
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
      send(buildClientMessage(AgentManagerWsClientMessageType.SUBSCRIBE, runId));
    },
    [send]
  );

  const unsubscribe = useCallback(
    (runId: string) => {
      send(buildClientMessage(AgentManagerWsClientMessageType.UNSUBSCRIBE, runId));
    },
    [send]
  );

  const subscribeAll = useCallback(() => {
    send(buildClientMessage(AgentManagerWsClientMessageType.SUBSCRIBE_ALL));
  }, [send]);

  const unsubscribeAll = useCallback(() => {
    send(buildClientMessage(AgentManagerWsClientMessageType.UNSUBSCRIBE_ALL));
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
    addMessageHandler,
    removeMessageHandler,
  };
}
