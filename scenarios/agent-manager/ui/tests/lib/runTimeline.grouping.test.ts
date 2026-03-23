import test from "node:test";
import assert from "node:assert/strict";
import type { RunEvent } from "../../src/types.js";
import {
  buildTimelineEntries,
  buildToolGroupSummary,
  filterTimelineEntries,
  createDefaultTimelineFilterState,
  groupTimelineEntries,
  type TimelineToolGroup,
  type ToolCallPair,
} from "../../src/lib/runTimeline.js";

const RUN_EVENT_TYPE_MESSAGE = 2;
const RUN_EVENT_TYPE_TOOL_CALL = 3;
const RUN_EVENT_TYPE_TOOL_RESULT = 4;

type RunEventOverrides = Omit<Partial<RunEvent>, "data"> & {
  data?: unknown;
};

function makeEvent(overrides: RunEventOverrides): RunEvent {
  return {
    id: overrides.id ?? "event-1",
    runId: overrides.runId ?? "run-1",
    sequence: overrides.sequence ?? 1n,
    eventType: overrides.eventType ?? RUN_EVENT_TYPE_MESSAGE,
    timestamp: overrides.timestamp ?? { seconds: 1n, nanos: 0 },
    data: (overrides.data ?? { case: "message", value: { role: "assistant", content: "hi" } }) as RunEvent["data"],
  } as RunEvent;
}

function makeToolCall(id: string, seq: bigint, toolName: string, callId: string): RunEvent {
  return makeEvent({
    id,
    sequence: seq,
    eventType: RUN_EVENT_TYPE_TOOL_CALL,
    data: { case: "toolCall", value: { toolName, toolCallId: callId, input: {} } },
  });
}

function makeToolResult(id: string, seq: bigint, toolName: string, callId: string, success = true): RunEvent {
  return makeEvent({
    id,
    sequence: seq,
    eventType: RUN_EVENT_TYPE_TOOL_RESULT,
    data: { case: "toolResult", value: { toolName, toolCallId: callId, success, output: "ok", error: "" } },
  });
}

function makeMessage(id: string, seq: bigint, content: string): RunEvent {
  return makeEvent({
    id,
    sequence: seq,
    eventType: RUN_EVENT_TYPE_MESSAGE,
    data: { case: "message", value: { role: "assistant", content } },
  });
}

test("single tool call+result is not grouped", () => {
  const events = [
    makeToolCall("tc-1", 1n, "Edit", "call-1"),
    makeToolResult("tr-1", 2n, "Edit", "call-1"),
  ];
  const entries = buildTimelineEntries(events);
  const filtered = filterTimelineEntries(entries, createDefaultTimelineFilterState());
  const items = groupTimelineEntries(filtered);

  // Single pair -> ungrouped, emits call + result as separate entries
  assert.equal(items.length, 2);
  assert.equal(items[0]!.kind, "event");
  assert.equal(items[1]!.kind, "event");
});

test("two consecutive tool calls are grouped", () => {
  const events = [
    makeToolCall("tc-1", 1n, "Edit", "call-1"),
    makeToolResult("tr-1", 2n, "Edit", "call-1"),
    makeToolCall("tc-2", 3n, "Read", "call-2"),
    makeToolResult("tr-2", 4n, "Read", "call-2"),
  ];
  const entries = buildTimelineEntries(events);
  const filtered = filterTimelineEntries(entries, createDefaultTimelineFilterState());
  const items = groupTimelineEntries(filtered);

  assert.equal(items.length, 1);
  assert.equal(items[0]!.kind, "tool-group");
  const group = items[0] as TimelineToolGroup;
  assert.equal(group.pairs.length, 2);
  assert.equal(group.pairs[0]!.toolName, "Edit");
  assert.equal(group.pairs[1]!.toolName, "Read");
  assert.equal(group.summary, "Edit, Read");
});

test("tool calls split by message are not grouped", () => {
  const events = [
    makeToolCall("tc-1", 1n, "Edit", "call-1"),
    makeToolResult("tr-1", 2n, "Edit", "call-1"),
    makeMessage("msg-1", 3n, "Done editing"),
    makeToolCall("tc-2", 4n, "Read", "call-2"),
    makeToolResult("tr-2", 5n, "Read", "call-2"),
  ];
  const entries = buildTimelineEntries(events);
  const filtered = filterTimelineEntries(entries, createDefaultTimelineFilterState());
  const items = groupTimelineEntries(filtered);

  // Each single tool pair stays ungrouped, message in between
  assert.equal(items.length, 5); // call, result, message, call, result
  assert.equal(items[0]!.kind, "event");
  assert.equal(items[2]!.kind, "message");
  assert.equal(items[3]!.kind, "event");
});

test("mixed tool names produce correct summary with counts", () => {
  const events = [
    makeToolCall("tc-1", 1n, "Edit", "c1"),
    makeToolResult("tr-1", 2n, "Edit", "c1"),
    makeToolCall("tc-2", 3n, "Edit", "c2"),
    makeToolResult("tr-2", 4n, "Edit", "c2"),
    makeToolCall("tc-3", 5n, "Edit", "c3"),
    makeToolResult("tr-3", 6n, "Edit", "c3"),
    makeToolCall("tc-4", 7n, "Read", "c4"),
    makeToolResult("tr-4", 8n, "Read", "c4"),
    makeToolCall("tc-5", 9n, "Read", "c5"),
    makeToolResult("tr-5", 10n, "Read", "c5"),
  ];
  const entries = buildTimelineEntries(events);
  const filtered = filterTimelineEntries(entries, createDefaultTimelineFilterState());
  const items = groupTimelineEntries(filtered);

  assert.equal(items.length, 1);
  const group = items[0] as TimelineToolGroup;
  assert.equal(group.pairs.length, 5);
  assert.equal(group.summary, "Edit 3, Read 2");
});

test("orphan toolResult passes through ungrouped", () => {
  const events = [
    makeToolResult("tr-orphan", 1n, "Bash", "no-match"),
    makeToolCall("tc-1", 2n, "Edit", "c1"),
    makeToolResult("tr-1", 3n, "Edit", "c1"),
  ];
  const entries = buildTimelineEntries(events);
  const filtered = filterTimelineEntries(entries, createDefaultTimelineFilterState());
  const items = groupTimelineEntries(filtered);

  // Orphan result + single call/result pair (ungrouped)
  assert.equal(items.length, 3);
  assert.equal(items[0]!.kind, "event"); // orphan result
  assert.equal(items[1]!.kind, "event"); // call
  assert.equal(items[2]!.kind, "event"); // result
});

test("pending tool call (no result yet) stays ungrouped when solo", () => {
  const events = [
    makeToolCall("tc-1", 1n, "Bash", "c1"),
    // No result yet
  ];
  const entries = buildTimelineEntries(events);
  const filtered = filterTimelineEntries(entries, createDefaultTimelineFilterState());
  const items = groupTimelineEntries(filtered);

  assert.equal(items.length, 1);
  assert.equal(items[0]!.kind, "event");
});

test("pending tool calls group when 2+ consecutive", () => {
  const events = [
    makeToolCall("tc-1", 1n, "Edit", "c1"),
    makeToolCall("tc-2", 2n, "Read", "c2"),
    // No results yet
  ];
  const entries = buildTimelineEntries(events);
  const filtered = filterTimelineEntries(entries, createDefaultTimelineFilterState());
  const items = groupTimelineEntries(filtered);

  assert.equal(items.length, 1);
  assert.equal(items[0]!.kind, "tool-group");
  const group = items[0] as TimelineToolGroup;
  assert.equal(group.pairs.length, 2);
  assert.equal(group.pairs[0]!.result, undefined);
  assert.equal(group.pairs[1]!.result, undefined);
});

test("buildToolGroupSummary formats tool name counts", () => {
  const pairs: ToolCallPair[] = [
    { call: {} as never, toolName: "Edit" },
    { call: {} as never, toolName: "Edit" },
    { call: {} as never, toolName: "Edit" },
    { call: {} as never, toolName: "Read" },
    { call: {} as never, toolName: "Bash" },
  ];
  assert.equal(buildToolGroupSummary(pairs), "Edit 3, Read, Bash");
});

test("buildToolGroupSummary with single tool type", () => {
  const pairs: ToolCallPair[] = [
    { call: {} as never, toolName: "Edit" },
    { call: {} as never, toolName: "Edit" },
  ];
  assert.equal(buildToolGroupSummary(pairs), "Edit 2");
});

test("three consecutive tool calls form one group", () => {
  const events = [
    makeToolCall("tc-1", 1n, "Edit", "c1"),
    makeToolResult("tr-1", 2n, "Edit", "c1"),
    makeToolCall("tc-2", 3n, "Edit", "c2"),
    makeToolResult("tr-2", 4n, "Edit", "c2"),
    makeToolCall("tc-3", 5n, "Bash", "c3"),
    makeToolResult("tr-3", 6n, "Bash", "c3"),
  ];
  const entries = buildTimelineEntries(events);
  const filtered = filterTimelineEntries(entries, createDefaultTimelineFilterState());
  const items = groupTimelineEntries(filtered);

  assert.equal(items.length, 1);
  const group = items[0] as TimelineToolGroup;
  assert.equal(group.pairs.length, 3);
  assert.equal(group.summary, "Edit 2, Bash");
  assert.equal(group.id, "tool-group-tc-1");
});
