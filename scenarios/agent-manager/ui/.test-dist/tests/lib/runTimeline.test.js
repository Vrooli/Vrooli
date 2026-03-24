import test from "node:test";
import assert from "node:assert/strict";
import { buildTimelineEntries, countTimelineEntriesByCategory, createDefaultTimelineFilterState, filterTimelineEntries, getTimelineCategory, getTimelineEventSummary, isReasoningEvent, } from "../../src/lib/runTimeline.js";
const RUN_EVENT_TYPE_MESSAGE = 2;
const RUN_EVENT_TYPE_TOOL_CALL = 3;
const RUN_EVENT_TYPE_TOOL_RESULT = 4;
const RUN_EVENT_TYPE_STATUS = 5;
const RUN_EVENT_TYPE_LOG = 1;
const RUN_EVENT_TYPE_COMPACTION = 10;
const RUN_EVENT_TYPE_MESSAGE_DELETED = 9;
function makeEvent(overrides) {
    return {
        id: overrides.id ?? "event-1",
        runId: overrides.runId ?? "run-1",
        sequence: overrides.sequence ?? 1n,
        eventType: overrides.eventType ?? RUN_EVENT_TYPE_MESSAGE,
        timestamp: overrides.timestamp ?? { seconds: 1n, nanos: 0 },
        data: (overrides.data ?? { case: "message", value: { role: "assistant", content: "hi" } }),
    };
}
test("buildTimelineEntries preserves run sequence and marks deleted messages", () => {
    const entries = buildTimelineEntries([
        makeEvent({
            id: "delete-1",
            sequence: 3n,
            eventType: RUN_EVENT_TYPE_MESSAGE_DELETED,
            data: { case: "messageDeleted", value: { targetEventId: "msg-1" } },
        }),
        makeEvent({
            id: "tool-1",
            sequence: 2n,
            eventType: RUN_EVENT_TYPE_TOOL_CALL,
            data: { case: "toolCall", value: { toolName: "Read", toolCallId: "tool-1" } },
        }),
        makeEvent({
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
    const reasoning = makeEvent({
        eventType: RUN_EVENT_TYPE_LOG,
        data: { case: "log", value: { level: "debug", message: "Reasoning: inspect auth flow" } },
    });
    const generic = makeEvent({
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
        makeEvent({
            id: "msg-1",
            sequence: 1n,
            eventType: RUN_EVENT_TYPE_MESSAGE,
            data: { case: "message", value: { role: "user", content: "Please fix it" } },
        }),
        makeEvent({
            id: "status-1",
            sequence: 2n,
            eventType: RUN_EVENT_TYPE_STATUS,
            data: { case: "status", value: { oldStatus: "running", newStatus: "needs_review" } },
        }),
        makeEvent({
            id: "reasoning-1",
            sequence: 3n,
            eventType: RUN_EVENT_TYPE_LOG,
            data: { case: "log", value: { level: "debug", message: "Thinking: compare two approaches" } },
        }),
        makeEvent({
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
        makeEvent({
            id: "delete-1",
            eventType: RUN_EVENT_TYPE_MESSAGE_DELETED,
            data: { case: "messageDeleted", value: { targetEventId: "abc12345-def" } },
        }),
    ])[0];
    const compaction = buildTimelineEntries([
        makeEvent({
            id: "compact-1",
            eventType: RUN_EVENT_TYPE_COMPACTION,
            data: { case: "compaction", value: { summary: "Condensed previous debugging context" } },
        }),
    ])[0];
    assert.equal(getTimelineEventSummary(redaction), "Message abc12345 redacted");
    assert.equal(getTimelineEventSummary(compaction), "Condensed previous debugging context");
});
test("category counts match the built timeline entries", () => {
    const entries = buildTimelineEntries([
        makeEvent({
            id: "msg-1",
            eventType: RUN_EVENT_TYPE_MESSAGE,
            data: { case: "message", value: { role: "assistant", content: "done" } },
        }),
        makeEvent({
            id: "tool-1",
            eventType: RUN_EVENT_TYPE_TOOL_RESULT,
            data: { case: "toolResult", value: { toolName: "Read", success: true } },
        }),
        makeEvent({
            id: "tool-2",
            eventType: RUN_EVENT_TYPE_TOOL_CALL,
            data: { case: "toolCall", value: { toolName: "Write" } },
        }),
    ]);
    const counts = countTimelineEntriesByCategory(entries);
    assert.equal(counts.messages, 1);
    assert.equal(counts.tools, 2);
});
