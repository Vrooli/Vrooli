import { describe, expect, it } from "vitest";
import type { RunEvent } from "../types";
import { getCostTotals } from "../components/RunDetailParts";
import { parseWebSocketMessage } from "./webSocketProtocol";

describe("workflow lifecycle websocket projection", () => {
  it("accepts metadata without inventing payload fields", () => {
    const message = parseWebSocketMessage({
      type: "AGENT_MANAGER_WS_MESSAGE_TYPE_WORKFLOW_LIFECYCLE",
      workflow_lifecycle: {
        execution_id: "1c7b2c39-bd80-4cd7-a5b2-b55f84ee3bd7",
        definition_digest: "sha256:abc",
        status: "waiting",
        node_id: "approval",
        journal_sequence: "4",
        journal_payload_digest: "sha256:def",
      },
    });
    expect(message?.type).toBe("workflow_lifecycle");
    expect(message?.payload).toMatchObject({ executionId: "1c7b2c39-bd80-4cd7-a5b2-b55f84ee3bd7", nodeId: "approval" });
    expect(message?.payload).not.toHaveProperty("prompt");
    expect(message?.payload).not.toHaveProperty("result");
  });
});

it("retains the canonical nanosecond run revision on status notifications", () => {
  const message = parseWebSocketMessage({
    type: "AGENT_MANAGER_WS_MESSAGE_TYPE_RUN_STATUS",
    run_status: { run_id: "run-1", status: "RUN_STATUS_COMPLETE", updated_at: "1970-01-01T00:00:01.000000011Z" },
  });
  expect(message?.payload).toMatchObject({ id: "run-1", status: 5, updatedAt: { seconds: 1n, nanos: 11 } });
});


describe("run event protobuf hydration", () => {
  // Actual RunEventToProto + protojson projection of NewLifecycleEvent:
  // lifecycle metadata has no data oneof in the current wire schema.
  const lifecycleWire = {
    id: "18e50bd8-1ef6-45a1-8460-2058ee9d11d4",
    run_id: "b8c6619c-a49d-4e6d-97db-78f738397d8d",
    event_type: "RUN_EVENT_TYPE_LIFECYCLE",
    timestamp: "2026-11-07T01:24:55.000000011Z",
    sequence: "1",
  };

  function receive(runEvent: unknown): RunEvent {
    const message = parseWebSocketMessage({
      type: "AGENT_MANAGER_WS_MESSAGE_TYPE_RUN_EVENT",
      run_event: runEvent,
    });
    expect(message?.type).toBe("run_event");
    return message!.payload as RunEvent;
  }

  it("keeps a valid metadata-only lifecycle event safe for the rendered cost fold", () => {
    const event = receive(lifecycleWire);
    expect(getCostTotals([event])).toMatchObject({ events: 0, totalCostUsd: 0 });
    expect(event.data).toEqual({ case: undefined });
    expect(event.sequence).toBe(1n);
    expect(event.timestamp).toMatchObject({ nanos: 11 });
  });

  it("hydrates and counts a positive cost payload alongside lifecycle metadata", () => {
    const lifecycle = receive(lifecycleWire);
    const cost = receive({
      ...lifecycleWire,
      event_type: "RUN_EVENT_TYPE_METRIC",
      sequence: "2",
      cost: { input_tokens: 7, output_tokens: 2, total_cost_usd: 0.25, model: "fixture-model" },
    });
    expect(cost.data.case).toBe("cost");
    expect(getCostTotals([lifecycle, cost])).toMatchObject({
      events: 1, inputTokens: 7, outputTokens: 2, totalCostUsd: 0.25, models: ["fixture-model"],
    });
  });

  it("rejects malformed protobuf fields instead of exposing a partially typed event", () => {
    expect(parseWebSocketMessage({
      type: "AGENT_MANAGER_WS_MESSAGE_TYPE_RUN_EVENT",
      run_event: { ...lifecycleWire, sequence: "not-an-integer" },
    })).toBeNull();
  });
});
