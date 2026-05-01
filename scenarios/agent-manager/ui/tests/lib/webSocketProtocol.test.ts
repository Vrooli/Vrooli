import { test } from "vitest";
import assert from "node:assert/strict";
import {
  buildWebSocketClientMessage,
  parseWebSocketMessage,
  WebSocketClientMessageType,
} from "../../src/lib/webSocketProtocol.js";

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

  assert.equal(parsed?.type, "run_event");
  assert.equal(parsed?.runId, "run-1");
  assert.equal(parsed?.payload && "sequence" in Object(parsed.payload) ? Object(parsed.payload).sequence : undefined, 7n);
  assert.equal(parsed?.payload && "eventType" in Object(parsed.payload) ? Object(parsed.payload).eventType : undefined, 2);
  assert.deepEqual(parsed?.payload && "data" in Object(parsed.payload) ? Object(parsed.payload).data : undefined, {
    case: "message",
    value: { role: "assistant", content: "done" },
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
