import type { RunEvent, Task } from "../types";
import { fromJson, type JsonValue } from "@bufbuild/protobuf";
import { RunEventSchema } from "@vrooli/proto-types/agent-manager/v1/domain/events_pb";
import { TimestampSchema } from "@bufbuild/protobuf/wkt";

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
  RUN_STATUS_PARKED: 8,
};

const TASK_STATUS_VALUES: Record<string, number> = {
  TASK_STATUS_UNSPECIFIED: 0,
  TASK_STATUS_PENDING: 1,
  TASK_STATUS_RUNNING: 2,
  TASK_STATUS_COMPLETE: 3,
  TASK_STATUS_FAILED: 4,
  TASK_STATUS_CANCELLED: 5,
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
  try {
    const { seconds, nanos } = fromJson(TimestampSchema, value);
    return { seconds, nanos };
  } catch {
    return value;
  }
}

function normalizeRunEvent(raw: unknown): RunEvent | null {
  if (!isRecord(raw)) return null;

  try {
    // Use the same protobuf hydration as HTTP. Valid metadata-only events
    // still carry the generated empty oneof required by timeline consumers.
    return fromJson(RunEventSchema, raw as JsonValue, { ignoreUnknownFields: true });
  } catch {
    return null;
  }
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
      if (raw.run_status.updated_at) {
        statusPayload.updatedAt = normalizeTimestamp(raw.run_status.updated_at);
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
    case "AGENT_MANAGER_WS_MESSAGE_TYPE_WORKFLOW_LIFECYCLE":
      if (!isRecord(raw.workflow_lifecycle) || typeof raw.workflow_lifecycle.execution_id !== "string") return null;
      return {
        type: "workflow_lifecycle",
        payload: camelizeValue(raw.workflow_lifecycle),
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
