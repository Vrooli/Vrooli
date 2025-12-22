import { useEffect, useRef, useState, useCallback } from "react";
import { resolveWsBase } from "@vrooli/api-base";
import type { Run, RunEvent, Task } from "../types";

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

const WS_MESSAGE_PREFIX = "AGENT_MANAGER_WS_MESSAGE_TYPE_";
const WS_CLIENT_PREFIX = "AGENT_MANAGER_WS_CLIENT_MESSAGE_TYPE_";

function asRecord(value: unknown): Record<string, unknown> | null {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return null;
  }
  return value as Record<string, unknown>;
}

function getField<T>(record: Record<string, unknown> | null, ...keys: string[]): T | undefined {
  if (!record) return undefined;
  for (const key of keys) {
    if (key in record) {
      return record[key] as T;
    }
  }
  return undefined;
}

function normalizeEnumValue(value: unknown, prefix: string): string | undefined {
  if (typeof value !== "string") return undefined;
  if (value.startsWith(prefix)) {
    return value.slice(prefix.length).toLowerCase();
  }
  if (value.includes(prefix)) {
    const parts = value.split(prefix);
    return parts[parts.length - 1]?.toLowerCase();
  }
  return value.toLowerCase();
}

function mapRunEventData(value: unknown): RunEvent["data"] {
  const record = asRecord(value);
  if (!record) return value as RunEvent["data"];
  return {
    level: getField(record, "level"),
    message: getField(record, "message"),
    role: getField(record, "role"),
    content: getField(record, "content"),
    toolName: getField(record, "toolName", "tool_name"),
    input: getField(record, "input"),
    toolCallId: getField(record, "toolCallId", "tool_call_id"),
    output: getField(record, "output"),
    error: getField(record, "error"),
    success: getField(record, "success"),
    oldStatus:
      normalizeEnumValue(getField(record, "oldStatus", "old_status"), "RUN_STATUS_") ??
      getField(record, "oldStatus", "old_status"),
    newStatus:
      normalizeEnumValue(getField(record, "newStatus", "new_status"), "RUN_STATUS_") ??
      getField(record, "newStatus", "new_status"),
    reason: getField(record, "reason"),
    name: getField(record, "name"),
    value: getField(record, "value"),
    unit: getField(record, "unit"),
    tags: getField(record, "tags"),
    type: getField(record, "type"),
    path: getField(record, "path"),
    size: getField(record, "size"),
    mimeType: getField(record, "mimeType", "mime_type"),
    code: getField(record, "code"),
    retryable: getField(record, "retryable"),
    recovery:
      normalizeEnumValue(getField(record, "recovery"), "RECOVERY_ACTION_") ??
      getField(record, "recovery"),
    stackTrace: getField(record, "stackTrace", "stack_trace"),
    details: getField(record, "details"),
  } as RunEvent["data"];
}

function mapRunEvent(value: unknown): RunEvent {
  const record = asRecord(value);
  if (!record) return value as RunEvent;
  const payload =
    getField(record, "data") ??
    getField(record, "log") ??
    getField(record, "message") ??
    getField(record, "tool_call") ??
    getField(record, "tool_result") ??
    getField(record, "status") ??
    getField(record, "metric") ??
    getField(record, "artifact") ??
    getField(record, "error") ??
    getField(record, "progress") ??
    getField(record, "cost") ??
    getField(record, "rate_limit");
  return {
    id: String(getField(record, "id") ?? ""),
    runId: String(getField(record, "runId", "run_id") ?? ""),
    sequence: Number(getField(record, "sequence") ?? 0),
    eventType: (normalizeEnumValue(getField(record, "eventType", "event_type"), "RUN_EVENT_TYPE_") ??
      getField(record, "eventType", "event_type")) as RunEvent["eventType"],
    timestamp: String(getField(record, "timestamp") ?? ""),
    data: mapRunEventData(payload),
  };
}

function normalizeProtoWsMessage(raw: unknown): WebSocketMessage | null {
  const record = asRecord(raw);
  if (!record) return null;
  const typeValue = getField<string>(record, "type");
  if (!typeValue || !typeValue.startsWith(WS_MESSAGE_PREFIX)) {
    return null;
  }

  const normalized = normalizeEnumValue(typeValue, WS_MESSAGE_PREFIX) ?? "";
  const runId = getField<string>(record, "run_id");
  switch (normalized) {
    case "run_event":
      return {
        type: "run_event",
        runId,
        payload: mapRunEvent(getField(record, "run_event")),
      };
    case "run_status": {
      const status = getField(record, "run_status");
      const statusRecord = asRecord(status);
      const statusRunId = getField<string>(statusRecord, "run_id") ?? runId;
      return {
        type: "run_status",
        runId: statusRunId,
        payload: {
          id: statusRunId ?? "",
          status:
            normalizeEnumValue(getField(statusRecord, "status"), "RUN_STATUS_") ??
            getField(statusRecord, "status"),
        } as Partial<Run>,
      };
    }
    case "task_status": {
      const taskStatus = getField(record, "task_status");
      const taskRecord = asRecord(taskStatus);
      const taskId = getField<string>(taskRecord, "task_id") ?? "";
      return {
        type: "task_status",
        payload: {
          id: taskId,
          status:
            normalizeEnumValue(getField(taskRecord, "status"), "TASK_STATUS_") ??
            getField(taskRecord, "status"),
        } as Partial<Task>,
      };
    }
    case "run_progress":
      return {
        type: "run_progress",
        runId,
        payload: getField(record, "run_progress"),
      };
    case "connected":
      return {
        type: "connected",
        payload: getField(record, "connected"),
      };
    case "pong":
      return {
        type: "pong",
        payload: getField(record, "pong"),
      };
    default:
      return null;
  }
}

function normalizeWsMessage(raw: unknown): WebSocketMessage {
  return (
    normalizeProtoWsMessage(raw) ??
    (raw as WebSocketMessage)
  );
}

function buildClientMessage(type: string, payload?: Record<string, unknown>): Record<string, unknown> {
  const message: Record<string, unknown> = { type };
  if (payload) {
    Object.assign(message, payload);
  }
  return message;
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
          const normalized = normalizeWsMessage(data);
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
      send(
        buildClientMessage(`${WS_CLIENT_PREFIX}SUBSCRIBE`, {
          run_subscription: { run_id: runId },
        })
      );
    },
    [send]
  );

  const unsubscribe = useCallback(
    (runId: string) => {
      send(
        buildClientMessage(`${WS_CLIENT_PREFIX}UNSUBSCRIBE`, {
          run_subscription: { run_id: runId },
        })
      );
    },
    [send]
  );

  const subscribeAll = useCallback(() => {
    send(buildClientMessage(`${WS_CLIENT_PREFIX}SUBSCRIBE_ALL`));
  }, [send]);

  const unsubscribeAll = useCallback(() => {
    send(buildClientMessage(`${WS_CLIENT_PREFIX}UNSUBSCRIBE_ALL`));
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
