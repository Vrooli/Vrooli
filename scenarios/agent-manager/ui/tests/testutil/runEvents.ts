import type { RunEvent } from "../../src/types.js";

export const RUN_EVENT_TYPE_LOG = 1;
export const RUN_EVENT_TYPE_MESSAGE = 2;
export const RUN_EVENT_TYPE_TOOL_CALL = 3;
export const RUN_EVENT_TYPE_TOOL_RESULT = 4;
export const RUN_EVENT_TYPE_STATUS = 5;
export const RUN_EVENT_TYPE_MESSAGE_DELETED = 9;
export const RUN_EVENT_TYPE_COMPACTION = 10;

export type RunEventOverrides = Omit<Partial<RunEvent>, "data"> & {
  data?: unknown;
};

export function makeRunEvent(overrides: RunEventOverrides = {}): RunEvent {
  return {
    id: overrides.id ?? "event-1",
    runId: overrides.runId ?? "run-1",
    sequence: overrides.sequence ?? 1n,
    eventType: overrides.eventType ?? RUN_EVENT_TYPE_MESSAGE,
    timestamp: overrides.timestamp ?? { seconds: 1n, nanos: 0 },
    data: (overrides.data ?? { case: "message", value: { role: "assistant", content: "hi" } }) as RunEvent["data"],
  } as RunEvent;
}

export function makeMessageEvent(id: string, sequence: bigint, content: string, runId = "run-1"): RunEvent {
  return makeRunEvent({
    id,
    runId,
    sequence,
    eventType: RUN_EVENT_TYPE_MESSAGE,
    data: { case: "message", value: { role: "assistant", content } },
  });
}

export function makeToolCallEvent(id: string, sequence: bigint, toolName: string, toolCallId: string): RunEvent {
  return makeRunEvent({
    id,
    sequence,
    eventType: RUN_EVENT_TYPE_TOOL_CALL,
    data: { case: "toolCall", value: { toolName, toolCallId, input: {} } },
  });
}

export function makeToolResultEvent(
  id: string,
  sequence: bigint,
  toolName: string,
  toolCallId: string,
  success = true
): RunEvent {
  return makeRunEvent({
    id,
    sequence,
    eventType: RUN_EVENT_TYPE_TOOL_RESULT,
    data: { case: "toolResult", value: { toolName, toolCallId, success, output: "ok", error: "" } },
  });
}
