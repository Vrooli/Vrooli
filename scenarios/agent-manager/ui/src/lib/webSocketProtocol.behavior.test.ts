import { test } from "vitest";
import assert from "node:assert/strict";
import { create } from "@bufbuild/protobuf";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";
import { RunEventSchema } from "@vrooli/proto-types/agent-manager/v1/domain/events_pb";
import { RunEventType } from "@vrooli/proto-types/agent-manager/v1/domain/types_pb";
import {
  buildWebSocketClientMessage,
  parseWebSocketMessage,
  WebSocketClientMessageType,
} from "./webSocketProtocol.js";

test("buildWebSocketClientMessage creates typed subscribe messages", () => {
  assert.deepEqual(buildWebSocketClientMessage(WebSocketClientMessageType.Subscribe, "run-1"), {
    type: "AGENT_MANAGER_WS_CLIENT_MESSAGE_TYPE_SUBSCRIBE",
    run_subscription: { run_id: "run-1" },
  });
});

test("buildWebSocketClientMessage creates typed subscribe-all messages", () => {
  assert.deepEqual(buildWebSocketClientMessage(WebSocketClientMessageType.SubscribeAll), {
    type: "AGENT_MANAGER_WS_CLIENT_MESSAGE_TYPE_SUBSCRIBE_ALL",
  });
});

test("parseWebSocketMessage normalizes run status payloads", () => {
  const wire = {
    type: "AGENT_MANAGER_WS_MESSAGE_TYPE_RUN_STATUS",
    run_id: "run-1",
    run_status: {
      run_id: "run-1",
      status: "RUN_STATUS_RUNNING",
      task_id: "task-1",
      prompt_preview: "Investigate websocket flow",
    },
  };

  const parsed = parseWebSocketMessage(wire);

  assert.deepEqual(parsed, {
    type: "run_status",
    runId: "run-1",
    payload: {
      id: "run-1",
      status: 3,
      taskId: "task-1",
      promptPreview: "Investigate websocket flow",
    },
  });
});

test("parseWebSocketMessage normalizes run event payloads", () => {
  const wire = {
    type: "AGENT_MANAGER_WS_MESSAGE_TYPE_RUN_EVENT",
    run_id: "run-1",
    run_event: {
      id: "event-1",
      run_id: "run-1",
      sequence: "7",
      event_type: "RUN_EVENT_TYPE_MESSAGE",
      timestamp: "2026-04-30T12:34:56Z",
      message: { role: "assistant", content: "done" },
    },
  };

  const parsed = parseWebSocketMessage(wire);

  // A live event is hydrated exactly as the HTTP route hydrates it: a typed
  // RunEvent message, defaults included, so timeline consumers read one shape.
  assert.deepEqual(parsed, {
    type: "run_event",
    runId: "run-1",
    payload: create(RunEventSchema, {
      id: "event-1",
      runId: "run-1",
      sequence: 7n,
      eventType: RunEventType.MESSAGE,
      timestamp: timestampFromDate(new Date("2026-04-30T12:34:56Z")),
      data: { case: "message", value: { role: "assistant", content: "done" } },
    }),
  });
});

test("parseWebSocketMessage normalizes task status payloads", () => {
  const wire = {
    type: "AGENT_MANAGER_WS_MESSAGE_TYPE_TASK_STATUS",
    task_status: {
      task_id: "task-1",
      status: "TASK_STATUS_RUNNING",
    },
  };

  assert.deepEqual(parseWebSocketMessage(wire), {
    type: "task_status",
    payload: {
      id: "task-1",
      status: 2,
    },
  });
});

test("parseWebSocketMessage rejects unsupported envelopes", () => {
  assert.equal(parseWebSocketMessage({ type: "AGENT_MANAGER_WS_MESSAGE_TYPE_RUN_STATUS" }), null);
});

test("parseWebSocketMessage preserves optional run-status fields only when the wire payload supplies them", () => {
  assert.deepEqual(parseWebSocketMessage({
    type: "AGENT_MANAGER_WS_MESSAGE_TYPE_RUN_STATUS",
    run_id: "fallback-run",
    run_status: { status: "RUN_STATUS_FAILED" },
  }), {
    type: "run_status", runId: "fallback-run", payload: { id: "fallback-run", status: 6 },
  });
  assert.equal(parseWebSocketMessage({
    type: "AGENT_MANAGER_WS_MESSAGE_TYPE_RUN_STATUS",
    run_status: { status: "RUN_STATUS_RUNNING" },
  }), null);
});

test("parseWebSocketMessage normalizes operational progress and connection envelopes", () => {
  assert.deepEqual(parseWebSocketMessage({
    type: "AGENT_MANAGER_WS_MESSAGE_TYPE_RUN_PROGRESS", run_id: "run-9",
    run_progress: { percent_complete: 75, active_tool: { tool_name: "bash" } },
  }), {
    type: "run_progress", runId: "run-9", payload: { percentComplete: 75, activeTool: { toolName: "bash" } },
  });
  assert.deepEqual(parseWebSocketMessage({
    type: "AGENT_MANAGER_WS_MESSAGE_TYPE_CONNECTED", connected: { connection_id: "socket-1" },
  }), { type: "connected", payload: { connectionId: "socket-1" } });
  assert.deepEqual(parseWebSocketMessage({
    type: "AGENT_MANAGER_WS_MESSAGE_TYPE_PONG", pong: { sent_at: "now" },
  }), { type: "pong", payload: { sentAt: "now" } });
});

test("parseWebSocketMessage rejects malformed typed payloads and preserves unknown enum values", () => {
  assert.equal(parseWebSocketMessage(null), null);
  assert.equal(parseWebSocketMessage([]), null);
  assert.equal(parseWebSocketMessage({ type: "AGENT_MANAGER_WS_MESSAGE_TYPE_TASK_STATUS", task_status: { status: "TASK_STATUS_RUNNING" } }), null);
  assert.equal(parseWebSocketMessage({ type: "AGENT_MANAGER_WS_MESSAGE_TYPE_WORKFLOW_LIFECYCLE", workflow_lifecycle: {} }), null);
  assert.equal(parseWebSocketMessage({ type: "AGENT_MANAGER_WS_MESSAGE_TYPE_RUN_PROGRESS" }), null);
  assert.deepEqual(parseWebSocketMessage({
    type: "AGENT_MANAGER_WS_MESSAGE_TYPE_TASK_STATUS", task_status: { task_id: "task-9", status: "FUTURE_STATUS" },
  }), { type: "task_status", payload: { id: "task-9", status: "FUTURE_STATUS" } });
});

test("parseWebSocketMessage maps alternate event payloads and rejects events HTTP hydration would reject", () => {
  const parsed = parseWebSocketMessage({
    type: "AGENT_MANAGER_WS_MESSAGE_TYPE_RUN_EVENT",
    run_event: {
      id: "event-log", run_id: "run-4", sequence: 3, event_type: "RUN_EVENT_TYPE_LOG",
      log: { level: "warn", message: "degraded" },
    },
  });
  assert.deepEqual(parsed, {
    type: "run_event", runId: "run-4", payload: create(RunEventSchema, {
      id: "event-log", runId: "run-4", sequence: 3n, eventType: RunEventType.LOG,
      data: { case: "log", value: { level: "warn", message: "degraded" } },
    }),
  });
  // A timestamp that is not RFC 3339 cannot become the typed Timestamp the
  // RunEvent contract promises, so the event is rejected, not half-typed.
  assert.equal(parseWebSocketMessage({
    type: "AGENT_MANAGER_WS_MESSAGE_TYPE_RUN_EVENT",
    run_event: { id: "event-bad", run_id: "run-4", sequence: 4, event_type: "RUN_EVENT_TYPE_LOG", timestamp: "not-a-timestamp" },
  }), null);
  assert.equal(parseWebSocketMessage({ type: "AGENT_MANAGER_WS_MESSAGE_TYPE_RUN_EVENT", run_event: null }), null);
});
