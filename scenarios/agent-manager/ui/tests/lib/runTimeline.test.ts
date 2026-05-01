import { test } from "vitest";
import assert from "node:assert/strict";
import {
  buildTimelineEntries,
  countTimelineEntriesByCategory,
  createDefaultTimelineFilterState,
  filterTimelineEntries,
  getTimelineCategory,
  getTimelineEventSummary,
  isReasoningEvent,
  type TimelineEventEntry,
} from "../../src/lib/runTimeline.js";
import {
  makeRunEvent,
  RUN_EVENT_TYPE_COMPACTION,
  RUN_EVENT_TYPE_LOG,
  RUN_EVENT_TYPE_MESSAGE,
  RUN_EVENT_TYPE_MESSAGE_DELETED,
  RUN_EVENT_TYPE_STATUS,
  RUN_EVENT_TYPE_TOOL_CALL,
  RUN_EVENT_TYPE_TOOL_RESULT,
} from "../testutil/runEvents.js";

test("buildTimelineEntries preserves run sequence and marks deleted messages", () => {
  const entries = buildTimelineEntries([
    makeRunEvent({
      id: "delete-1",
      sequence: 3n,
      eventType: RUN_EVENT_TYPE_MESSAGE_DELETED,
      data: { case: "messageDeleted", value: { targetEventId: "msg-1" } },
    }),
    makeRunEvent({
      id: "tool-1",
      sequence: 2n,
      eventType: RUN_EVENT_TYPE_TOOL_CALL,
      data: { case: "toolCall", value: { toolName: "Read", toolCallId: "tool-1" } },
    }),
    makeRunEvent({
      id: "msg-1",
      sequence: 1n,
      eventType: RUN_EVENT_TYPE_MESSAGE,
      data: { case: "message", value: { role: "assistant", content: "hello" } },
    }),
  ]);

  assert.deepEqual(entries.map((entry) => entry.id), ["msg-1", "tool-1", "delete-1"]);
  assert.equal(entries[0]?.kind, "message");
  assert.equal(entries[0] && "deleted" in entries[0] ? entries[0].deleted : false, true);
  assert.equal(entries[2]?.kind, "event");
  assert.equal(entries[2] && "category" in entries[2] ? entries[2].category : "", "redactions");
});

test("reasoning logs classify separately from generic logs", () => {
  const reasoning = makeRunEvent({
    eventType: RUN_EVENT_TYPE_LOG,
    data: { case: "log", value: { level: "debug", message: "Reasoning: inspect auth flow" } },
  });
  const generic = makeRunEvent({
    id: "log-2",
    eventType: RUN_EVENT_TYPE_LOG,
    data: { case: "log", value: { level: "info", message: "phase: starting" } },
  });

  assert.equal(isReasoningEvent(reasoning), true);
  assert.equal(getTimelineCategory(reasoning), "reasoning");
  assert.equal(getTimelineCategory(generic), "logs");
});

test("default filter keeps messages, reasoning, tools, errors, and compaction", () => {
  const filters = createDefaultTimelineFilterState();
  const entries = buildTimelineEntries([
    makeRunEvent({
      id: "msg-1",
      sequence: 1n,
      eventType: RUN_EVENT_TYPE_MESSAGE,
      data: { case: "message", value: { role: "user", content: "Please fix it" } },
    }),
    makeRunEvent({
      id: "status-1",
      sequence: 2n,
      eventType: RUN_EVENT_TYPE_STATUS,
      data: { case: "status", value: { oldStatus: "running", newStatus: "needs_review" } },
    }),
    makeRunEvent({
      id: "reasoning-1",
      sequence: 3n,
      eventType: RUN_EVENT_TYPE_LOG,
      data: { case: "log", value: { level: "debug", message: "Thinking: compare two approaches" } },
    }),
    makeRunEvent({
      id: "compaction-1",
      sequence: 4n,
      eventType: RUN_EVENT_TYPE_COMPACTION,
      data: { case: "compaction", value: { summary: "Focused on auth state" } },
    }),
  ]);

  const visible = filterTimelineEntries(entries, filters);

  assert.deepEqual(visible.map((entry) => entry.id), ["msg-1", "reasoning-1", "compaction-1"]);
});

test("event summaries produce user-facing text for new timeline cards", () => {
  const redaction = buildTimelineEntries([
    makeRunEvent({
      id: "delete-1",
      eventType: RUN_EVENT_TYPE_MESSAGE_DELETED,
      data: { case: "messageDeleted", value: { targetEventId: "abc12345-def" } },
    }),
  ])[0] as TimelineEventEntry;

  const compaction = buildTimelineEntries([
    makeRunEvent({
      id: "compact-1",
      eventType: RUN_EVENT_TYPE_COMPACTION,
      data: { case: "compaction", value: { summary: "Condensed previous debugging context" } },
    }),
  ])[0] as TimelineEventEntry;

  assert.equal(getTimelineEventSummary(redaction), "Message abc12345 redacted");
  assert.equal(getTimelineEventSummary(compaction), "Condensed previous debugging context");
});

test("category counts match the built timeline entries", () => {
  const entries = buildTimelineEntries([
    makeRunEvent({
      id: "msg-1",
      eventType: RUN_EVENT_TYPE_MESSAGE,
      data: { case: "message", value: { role: "assistant", content: "done" } },
    }),
    makeRunEvent({
      id: "tool-1",
      eventType: RUN_EVENT_TYPE_TOOL_RESULT,
      data: { case: "toolResult", value: { toolName: "Read", success: true } },
    }),
    makeRunEvent({
      id: "tool-2",
      eventType: RUN_EVENT_TYPE_TOOL_CALL,
      data: { case: "toolCall", value: { toolName: "Write" } },
    }),
  ]);

  const counts = countTimelineEntriesByCategory(entries);
  assert.equal(counts.messages, 1);
  assert.equal(counts.tools, 2);
});
