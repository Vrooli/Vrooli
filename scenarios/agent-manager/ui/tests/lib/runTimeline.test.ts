import { test } from "vitest";
import assert from "node:assert/strict";
import {
  buildTimelineEntries,
  countTimelineEntriesByCategory,
  createDefaultTimelineFilterState,
  createShowAllTimelineFilterState,
  filterTimelineEntries,
  getTimelineModeLabel,
  getTimelineCategory,
  getTimelineEventSummary,
  isReasoningEvent,
  getTimelineCategoryLabel,
  getTimelineEventLabel,
  sortRunEvents,
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

test("timeline maps every operational evidence family to an investigator-facing category and summary", () => {
  const cases: Array<[string, unknown, string, string]> = [
    ["toolCall", { toolName: "Read" }, "tools", "Read"],
    ["toolResult", { toolName: "Write", success: false }, "tools", "Failed Write"],
    ["status", { oldStatus: "running", newStatus: "failed" }, "status", "running -> failed"],
    ["progress", { percentComplete: 42 }, "status", "Progress 42%"],
    ["artifact", { path: "report.json" }, "artifacts", "report.json"],
    ["metric", { name: "tokens", value: 12 }, "metrics", "tokens: 12"],
    ["cost", { totalCostUsd: 0.125 }, "metrics", "Cost update $0.1250"],
    ["error", { message: "runner crashed" }, "errors", "runner crashed"],
    ["rateLimit", { message: "slow down" }, "errors", "slow down"],
  ];
  for (const [kind, value, category, summary] of cases) {
    const entry = buildTimelineEntries([makeRunEvent({ data: { case: kind, value } })])[0] as TimelineEventEntry;
    assert.equal(entry.category, category);
    assert.equal(getTimelineEventSummary(entry), summary);
    assert.ok(getTimelineEventLabel(entry)); assert.ok(getTimelineCategoryLabel(entry.category));
  }
});

test("timeline excludes malformed or image-only messages while preserving valid attachments", () => {
  const entries = buildTimelineEntries([
    makeRunEvent({ id: "unknown-role", data: { case: "message", value: { role: "tool", content: "hidden" } } }),
    makeRunEvent({ id: "image-context", sequence: 2n, data: { case: "message", value: { role: "assistant", content: '<context type="image"></context>' } } }),
    makeRunEvent({ id: "attached", sequence: 3n, data: { case: "message", value: { role: "assistant", content: '<context type="image"></context>', attachments: [{ id: "a", fileName: "evidence.png", url: "https://e" }] } } }),
  ]);
  assert.deepEqual(entries.map((entry) => entry.id), ["attached"]);
  assert.equal(entries[0] && "attachments" in entries[0] ? entries[0].attachments[0]?.fileName : "", "evidence.png");
});

test("timeline filter and label helpers cover all operator display modes and fallback summaries", () => {
  assert.equal(getTimelineModeLabel("conversation"), "Conversation");
  assert.equal(getTimelineModeLabel("combined"), "Combined");
  assert.equal(getTimelineModeLabel("events"), "Events");
  const all = createShowAllTimelineFilterState();
  assert.equal(all.mode, "events");
  assert.equal(Object.values(all.categories).every(Boolean), true);

  const entries = buildTimelineEntries([
    makeRunEvent({ id: "message", data: { case: "message", value: { role: "system", content: "System guidance" } } }),
    makeRunEvent({ id: "log", sequence: 2n, data: { case: "log", value: {} } }),
    makeRunEvent({ id: "artifact", sequence: 3n, data: { case: "artifact", value: {} } }),
    makeRunEvent({ id: "metric", sequence: 4n, data: { case: "metric", value: {} } }),
    makeRunEvent({ id: "error", sequence: 5n, data: { case: "error", value: { code: "E_NETWORK" } } }),
    makeRunEvent({ id: "rate", sequence: 6n, data: { case: "rateLimit", value: {} } }),
    makeRunEvent({ id: "cost", sequence: 7n, data: { case: "cost", value: {} } }),
    makeRunEvent({ id: "compact", sequence: 8n, data: { case: "compaction", value: { trigger: "token limit" } } }),
  ]);
  const eventsOnly = filterTimelineEntries(entries, all);
  assert.deepEqual(eventsOnly.map((entry) => entry.id), ["log", "artifact", "metric", "error", "rate", "cost", "compact"]);
  const byId = new Map(entries.map((entry) => [entry.id, entry]));
  assert.equal(getTimelineEventSummary(byId.get("log") as TimelineEventEntry), "Log entry");
  assert.equal(getTimelineEventSummary(byId.get("artifact") as TimelineEventEntry), "Artifact created");
  assert.equal(getTimelineEventSummary(byId.get("metric") as TimelineEventEntry), "metric: 0");
  assert.equal(getTimelineEventSummary(byId.get("error") as TimelineEventEntry), "E_NETWORK");
  assert.equal(getTimelineEventSummary(byId.get("rate") as TimelineEventEntry), "Rate limited");
  assert.equal(getTimelineEventSummary(byId.get("cost") as TimelineEventEntry), "Cost update");
  assert.equal(getTimelineEventSummary(byId.get("compact") as TimelineEventEntry), "token limit");
});

test("timeline normalizes sparse runner events, unknown evidence, and equal-sequence ordering", () => {
  const unknown = makeRunEvent({
    id: "unknown",
    sequence: 3n,
    data: { case: "runnerNotice", value: {} } as never,
  });
  const noTimestamp = makeRunEvent({ id: "no-time", sequence: 5 as never });
  const earlyTimestamp = makeRunEvent({
    id: "early-time",
    sequence: 5 as never,
    timestamp: { seconds: 1 as never, nanos: 0 as never },
  });
  const lateTimestamp = makeRunEvent({
    id: "late-time",
    sequence: 5 as never,
    timestamp: { seconds: 1 as never, nanos: 500_000_000 as never },
  });

  assert.equal(isReasoningEvent(unknown), false);
  assert.equal(getTimelineCategory(unknown), "logs");
  const entry = buildTimelineEntries([unknown])[0] as TimelineEventEntry;
  assert.equal(entry.category, "logs");
  assert.equal(getTimelineEventSummary(entry), "Log");
  assert.deepEqual(sortRunEvents([lateTimestamp, earlyTimestamp, noTimestamp]).map((event) => event.id), [
    "early-time", "no-time", "late-time",
  ]);
});

test("timeline summaries retain runner fallbacks when telemetry fields are absent or falsey", () => {
  const entries = buildTimelineEntries([
    makeRunEvent({ id: "call", data: { case: "toolCall", value: {} } }),
    makeRunEvent({ id: "result", sequence: 2n, data: { case: "toolResult", value: { success: true } } }),
    makeRunEvent({ id: "status", sequence: 3n, data: { case: "status", value: {} } }),
    makeRunEvent({ id: "progress", sequence: 4n, data: { case: "progress", value: {} } }),
    makeRunEvent({ id: "artifact", sequence: 5n, data: { case: "artifact", value: { type: "diff" } } }),
    makeRunEvent({ id: "message-deleted", sequence: 6n, data: { case: "messageDeleted", value: {} } }),
  ]);
  const byId = new Map(entries.map((entry) => [entry.id, entry as TimelineEventEntry]));
  assert.equal(getTimelineEventSummary(byId.get("call")!), "Unknown tool");
  assert.equal(getTimelineEventSummary(byId.get("result")!), "Completed tool");
  assert.equal(getTimelineEventSummary(byId.get("status")!), "unknown -> unknown");
  assert.equal(getTimelineEventSummary(byId.get("progress")!), "Progress 0%");
  assert.equal(getTimelineEventSummary(byId.get("artifact")!), "diff");
  assert.equal(getTimelineEventSummary(byId.get("message-deleted")!), "Message  redacted");
});
