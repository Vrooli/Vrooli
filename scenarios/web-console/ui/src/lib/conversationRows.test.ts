import { beforeEach, describe, expect, it } from "vitest";
import type { ConversationEvent } from "../api/conversation";
import { getSessionRefetchSinceSequence, getSessionUnreadCount, useConversationStore, createConversationSessionState } from "../stores/useConversationStore";
import { foldLiveRow, isAgentReply, mergeProjectedRows } from "./conversationRows";
import { workSummary, workRowLine } from "../components/messages/workSummary";
import { noticeSummary } from "../components/messages/noticeSummary";
import { replyIndices } from "../components/messages/replyTrack";
import { strings } from "../consts/strings";

const at = (seconds: number) => new Date(Date.UTC(2026, 10, 26, 10, 0, seconds)).toISOString();

function row(sequence: number, overrides: Partial<ConversationEvent> = {}): ConversationEvent {
  return {
    id: `e${String(sequence)}`, sessionId: "s1", source: "codex_tailer", role: "assistant", text: `m${String(sequence)}`,
    speechParagraphs: [], summarized: false, createdAt: at(sequence), occurredAt: at(sequence), sequence,
    deliveryState: "received", ttsState: "idle", consumptionState: "unseen", ...overrides,
  };
}

describe("live rows fold into the projection", () => {
  it("extends the tail's work row with each tool row and starts a new one after a message", () => {
    let list = [row(1, { role: "user", kind: "message", author: "human" })];
    list = foldLiveRow(list, row(2, { kind: "tool_call", text: "" }));
    list = foldLiveRow(list, row(3, { kind: "tool_result", text: "" }));
    list = foldLiveRow(list, row(4, { kind: "context", text: "" }));
    expect(list.map((event) => event.id)).toEqual(["e1", "work::2"]);
    expect(list[1]?.sequence).toBe(4);
    expect(list[1]?.work).toMatchObject({ fromSequence: 2, toSequence: 4, toolCalls: 1, toolResults: 1, context: 1, startedAt: at(2), endedAt: at(4) });
    list = foldLiveRow(list, row(5, { kind: "message" }));
    list = foldLiveRow(list, row(6, { kind: "tool_call", text: "" }));
    expect(list.map((event) => event.id)).toEqual(["e1", "work::2", "e5", "work::6"]);
  });

  it("covers another thread's rows without counting them as this thread's work", () => {
    let list = [row(1, { kind: "message" })];
    list = foldLiveRow(list, row(2, { kind: "message", threadId: "hilbert" }));
    list = foldLiveRow(list, row(3, { kind: "tool_call", threadId: "hilbert" }));
    expect(list).toHaveLength(2);
    expect(list[1]?.work).toMatchObject({ fromSequence: 2, toSequence: 3, toolCalls: 0, toolResults: 0, context: 0 });
  });

  it("lets a later read of the same span replace the live one", () => {
    const live = foldLiveRow([row(1, { kind: "message" })], row(2, { kind: "tool_call" }));
    const liveWork = live[1];
    if (!liveWork?.work) throw new Error("expected a live work row");
    const server = { ...liveWork, sequence: 3, work: { ...liveWork.work, toSequence: 3, toolCalls: 2 } };
    const merged = mergeProjectedRows(live, [server, row(4, { kind: "message" })]);
    expect(merged.events.map((event) => `${event.id}@${String(event.sequence)}`)).toEqual(["e1@1", "work::2@3", "e4@4"]);
    expect(merged.added.map((event) => event.id)).toEqual(["e4"]);
    expect(mergeProjectedRows(merged.events, [server]).events).toBe(merged.events);
  });
});

describe("the store keeps the projection", () => {
  beforeEach(() => { useConversationStore.setState({ sessions: {}, echoes: {} }); });

  it("folds live tool rows, counts only the agent's replies as unread, and sees no gap across a work row", () => {
    useConversationStore.setState({ sessions: { s1: createConversationSessionState({ events: [row(1, { role: "user", kind: "message", author: "human" })], maxSequence: 1, hasSequenceGap: false }) } });
    const { appendEvent } = useConversationStore.getState();
    appendEvent(row(2, { kind: "tool_call", text: "" }));
    appendEvent(row(3, { kind: "tool_result", text: "" }));
    appendEvent(row(3, { id: "e3", kind: "tool_result", text: "" })); // a replay changes nothing
    appendEvent(row(4, { kind: "message", author: "agent" }));
    appendEvent(row(5, { kind: "message", author: "peer", threadId: "teammate:serving-fabric", role: "user" }));
    const state = useConversationStore.getState();
    const session = state.sessions.s1;
    expect(session?.events.map((event) => event.id)).toEqual(["e1", "work::2", "e4", "work::5"]);
    expect(getSessionUnreadCount(state, "s1")).toBe(1);
    expect(getSessionRefetchSinceSequence(state, "s1")).toBe(5);
    expect(session?.threads).toEqual([expect.objectContaining({ threadId: "teammate:serving-fabric", kind: "teammate", messageCount: 1 })]);
  });
});

describe("row helpers", () => {
  it("treats only the main thread agent's messages as replies (the reader's steps)", () => {
    const list = [
      row(1, { role: "user", kind: "message", author: "human" }),
      row(2, { kind: "work", work: { fromSequence: 2, toSequence: 2, toolCalls: 1, toolResults: 0, reasoning: 0, context: 0 } }),
      row(3, { kind: "message", author: "agent" }),
      row(4, { kind: "notice", author: "agent" }),
      row(5, { kind: "message", author: "agent", threadId: "hilbert" }),
      row(6),
    ];
    expect(replyIndices(list)).toEqual([2, 5]);
    expect(list.map((event) => isAgentReply(event))).toEqual([false, false, true, false, false, true]);
  });

  it("summarises a span as how long the agent took and how many tools it called", () => {
    const t = (key: string, params?: Record<string, number | string>) => `${key}${params ? JSON.stringify(params) : ""}`;
    const span = { fromSequence: 2, toSequence: 9, toolCalls: 10, toolResults: 10, reasoning: 0, context: 0, startedAt: at(2), endedAt: at(9) };
    expect(workSummary(span, t, at(0), at(20))).toEqual([
      `${strings.messagesPane.work.worked}{"duration":"${strings.messagesPane.duration.seconds}{\\"seconds\\":20}"}`,
      `${strings.messagesPane.work.toolCalls}{"count":10}`,
    ]);
    expect(workSummary({ ...span, toolCalls: 0, toolResults: 0, reasoning: 1 }, t, at(0), at(15))[0]).toContain(strings.messagesPane.work.thought);
    expect(workSummary({ ...span, toolCalls: 0, toolResults: 0 }, t)).toEqual([]);
    expect(workRowLine({ label: "exec", text: "[Tool call: exec]\n\"make test\"\nCall ID: call_1" })).toEqual({ title: "exec", detail: "make test" });
  });

  it("reads one line out of a notice's markup", () => {
    expect(noticeSummary({ label: "slash_command", text: "<command-name>/compact</command-name>\n<command-args>keep the plan</command-args>" })).toBe("/compact keep the plan");
    expect(noticeSummary({ label: "auto_review_verdict", text: "{\"risk_level\":\"low\",\"outcome\":\"allow\"}" })).toBe("low · allow");
    expect(noticeSummary({ label: "interrupted", text: "[Request interrupted by user]" })).toBe("[Request interrupted by user]");
  });
});
