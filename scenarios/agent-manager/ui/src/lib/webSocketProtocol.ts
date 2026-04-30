import type { RunEvent, Task } from "../types";

export interface WebSocketMessage {
  type: string;
  payload: unknown;
  runId?: string;
}

export enum WebSocketClientMessageType {
  Subscribe = "AGENT_MANAGER_WS_CLIENT_MESSAGE_TYPE_SUBSCRIBE",
  Unsubscribe = "AGENT_MANAGER_WS_CLIENT_MESSAGE_TYPE_UNSUBSCRIBE",
  SubscribeAll = "AGENT_MANAGER_WS_CLIENT_MESSAGE_TYPE_SUBSCRIBE_ALL",
  UnsubscribeAll = "AGENT_MANAGER_WS_CLIENT_MESSAGE_TYPE_UNSUBSCRIBE_ALL",
  Ping = "AGENT_MANAGER_WS_CLIENT_MESSAGE_TYPE_PING",
}

const RUN_STATUS_VALUES: Record<string, number> = {
  RUN_STATUS_UNSPECIFIED: 0,
  RUN_STATUS_PENDING: 1,
  RUN_STATUS_STARTING: 2,
  RUN_STATUS_RUNNING: 3,
  RUN_STATUS_NEEDS_REVIEW: 4,
  RUN_STATUS_COMPLETE: 5,
  RUN_STATUS_FAILED: 6,
  RUN_STATUS_CANCELLED: 7,
};

const TASK_STATUS_VALUES: Record<string, number> = {
  TASK_STATUS_UNSPECIFIED: 0,
  TASK_STATUS_PENDING: 1,
  TASK_STATUS_RUNNING: 2,
  TASK_STATUS_COMPLETE: 3,
  TASK_STATUS_FAILED: 4,
  TASK_STATUS_CANCELLED: 5,
};

const RUN_EVENT_TYPE_VALUES: Record<string, number> = {
  RUN_EVENT_TYPE_UNSPECIFIED: 0,
  RUN_EVENT_TYPE_LOG: 1,
  RUN_EVENT_TYPE_MESSAGE: 2,
  RUN_EVENT_TYPE_TOOL_CALL: 3,
  RUN_EVENT_TYPE_TOOL_RESULT: 4,
  RUN_EVENT_TYPE_STATUS: 5,
  RUN_EVENT_TYPE_METRIC: 6,
  RUN_EVENT_TYPE_ARTIFACT: 7,
  RUN_EVENT_TYPE_ERROR: 8,
  RUN_EVENT_TYPE_MESSAGE_DELETED: 9,
  RUN_EVENT_TYPE_COMPACTION: 10,
};

const RUN_EVENT_DATA_CASES: Record<string, string> = {
  log: "log",
  message: "message",
  message_deleted: "messageDeleted",
  tool_call: "toolCall",
  tool_result: "toolResult",
  status: "status",
  metric: "metric",
  artifact: "artifact",
  error: "error",
  progress: "progress",
  cost: "cost",
  rate_limit: "rateLimit",
  compaction: "compaction",
};

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function camelizeKey(key: string): string {
  return key.replace(/_([a-z])/g, (_, letter: string) => letter.toUpperCase());
}

function camelizeValue(value: unknown): unknown {
  if (Array.isArray(value)) {
    return value.map(camelizeValue);
  }
  if (!isRecord(value)) {
    return value;
  }

  return Object.fromEntries(
    Object.entries(value).map(([key, entryValue]) => [camelizeKey(key), camelizeValue(entryValue)])
  );
}

function enumValue(value: unknown, values: Record<string, number>): unknown {
  if (typeof value === "string") {
    return values[value] ?? value;
  }
  return value;
}

function normalizeTimestamp(value: unknown): unknown {
  if (typeof value !== "string") {
    return camelizeValue(value);
  }
  const parsed = Date.parse(value);
  if (Number.isNaN(parsed)) {
    return value;
  }
  return {
    seconds: BigInt(Math.trunc(parsed / 1000)),
    nanos: (parsed % 1000) * 1_000_000,
  };
}

function normalizeRunEvent(raw: unknown): RunEvent | null {
  if (!isRecord(raw)) return null;

  const dataEntry = Object.entries(RUN_EVENT_DATA_CASES).find(([wireName]) => raw[wireName] !== undefined);
  const normalized: Record<string, unknown> = {
    id: raw.id,
    runId: raw.run_id,
    sequence: typeof raw.sequence === "string" ? BigInt(raw.sequence) : raw.sequence,
    eventType: enumValue(raw.event_type, RUN_EVENT_TYPE_VALUES),
    timestamp: normalizeTimestamp(raw.timestamp),
  };

  if (dataEntry) {
    const [wireName, caseName] = dataEntry;
    normalized.data = {
      case: caseName,
      value: camelizeValue(raw[wireName]),
    };
  }

  return normalized as RunEvent;
}

export function parseWebSocketMessage(raw: unknown): WebSocketMessage | null {
  if (!isRecord(raw)) return null;

  switch (raw.type) {
    case "AGENT_MANAGER_WS_MESSAGE_TYPE_RUN_EVENT": {
      const event = normalizeRunEvent(raw.run_event);
      if (!event) return null;
      return {
        type: "run_event",
        runId: typeof raw.run_id === "string" ? raw.run_id : event.runId,
        payload: event,
      };
    }
    case "AGENT_MANAGER_WS_MESSAGE_TYPE_RUN_STATUS": {
      if (!isRecord(raw.run_status)) return null;
      const runId = typeof raw.run_status.run_id === "string" ? raw.run_status.run_id : raw.run_id;
      if (!runId) return null;

      const statusPayload: Record<string, unknown> = {
        id: runId,
        status: enumValue(raw.run_status.status, RUN_STATUS_VALUES),
      };
      if (raw.run_status.task_id) {
        statusPayload.taskId = raw.run_status.task_id;
      }
      if (raw.run_status.prompt_preview) {
        statusPayload.promptPreview = raw.run_status.prompt_preview;
      }

      return {
        type: "run_status",
        runId: String(runId),
        payload: statusPayload,
      };
    }
    case "AGENT_MANAGER_WS_MESSAGE_TYPE_TASK_STATUS":
      if (!isRecord(raw.task_status) || typeof raw.task_status.task_id !== "string") return null;
      return {
        type: "task_status",
        payload: {
          id: raw.task_status.task_id,
          status: enumValue(raw.task_status.status, TASK_STATUS_VALUES) as Task["status"],
        } satisfies Partial<Task>,
      };
    case "AGENT_MANAGER_WS_MESSAGE_TYPE_RUN_PROGRESS":
      if (!isRecord(raw.run_progress)) return null;
      return {
        type: "run_progress",
        runId: typeof raw.run_id === "string" ? raw.run_id : undefined,
        payload: camelizeValue(raw.run_progress),
      };
    case "AGENT_MANAGER_WS_MESSAGE_TYPE_CONNECTED":
      if (!isRecord(raw.connected)) return null;
      return {
        type: "connected",
        payload: camelizeValue(raw.connected),
      };
    case "AGENT_MANAGER_WS_MESSAGE_TYPE_PONG":
      if (!isRecord(raw.pong)) return null;
      return {
        type: "pong",
        payload: camelizeValue(raw.pong),
      };
    default:
      return null;
  }
}

export function buildWebSocketClientMessage(
  type: WebSocketClientMessageType,
  runId?: string
): Record<string, unknown> {
  return {
    type,
    ...(runId ? { run_subscription: { run_id: runId } } : {}),
  };
}
