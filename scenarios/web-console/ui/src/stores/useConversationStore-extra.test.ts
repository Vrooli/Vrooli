import { beforeEach, describe, expect, it } from "vitest";
import type { ConversationEvent } from "../api/conversation";
import {
  getSessionConversationCursor,
  getSessionRefetchSinceSequence,
  getSessionUnlistenedEvents,
  getSessionUnreadCount,
  getSessionViewMode,
  useConversationStore,
} from "./useConversationStore";

const event = (id: string, sequence: number, role: "assistant" | "user" = "assistant"): ConversationEvent => ({
  id, sessionId: "s", source: "test", role, text: id, speechParagraphs: [id], summarized: false,
  createdAt: "now", sequence, deliveryState: "received", ttsState: "idle", consumptionState: "unseen",
});

beforeEach(() => useConversationStore.setState({ sessions: {}, viewModes: {} }));

describe("conversation store state transitions", () => {
  it("hydrates, prepends, merges, updates, cursors and clears", () => {
    const store = useConversationStore.getState();
    store.hydrateSession("s", [event("e2", 2)], { lastSeenSequence: 0, lastListenedSequence: 0 }, { oldestSequence: 2, hasOlder: true, totalCount: 3 });
    store.prependEvents("s", [event("e1", 1), event("e2", 2)], { oldestSequence: 1, hasOlder: false, totalCount: 2 });
    store.mergeEvents("s", [event("e3", 3)], { lastSeenSequence: 1, lastListenedSequence: 0 });
    store.updateEvent("s", "e3", { summarized: true, speechParagraphs: ["short"], originalSpeechParagraphs: ["long"] });
    store.updateEvent("s", "missing", { summarized: true });
    store.updateCursor("s", { lastSeenSequence: 2, lastListenedSequence: 1 });
    store.setViewMode("s", "messages");
    const current = useConversationStore.getState();
    expect(current.sessions.s?.events.map((e) => e.id)).toEqual(["e1", "e2", "e3"]);
    expect(current.sessions.s?.events[2]).toMatchObject({ summarized: true, speechParagraphs: ["short"] });
    expect(getSessionUnreadCount(current, "s")).toBe(1);
    expect(getSessionUnlistenedEvents(current, "s")).toHaveLength(2);
    expect(getSessionViewMode(current, "s")).toBe("messages");
    expect(getSessionConversationCursor(current, "missing")).toEqual({ lastSeenSequence: 0, lastListenedSequence: 0 });
    expect(getSessionRefetchSinceSequence(current, "s")).toBe(3);
    store.clearSession("s");
    expect(getSessionViewMode(useConversationStore.getState(), "s")).toBe("terminal");
  });

  it("marks gaps and chooses prefix refetch", () => {
    const store = useConversationStore.getState();
    store.setSessionWindow("s", [event("e2", 2), event("e4", 4)], { lastSeenSequence: 0, lastListenedSequence: 0 }, { oldestSequence: 2, hasOlder: true, totalCount: 4 });
    const state = useConversationStore.getState();
    expect(getSessionRefetchSinceSequence(state, "s")).toBe(0);
    store.mergeEvents("s", [], undefined);
    expect(getSessionConversationEventsFallback(useConversationStore.getState(), "s")).toHaveLength(2);
  });
});

function getSessionConversationEventsFallback(state: ReturnType<typeof useConversationStore.getState>, id: string) {
  return state.sessions[id]?.events ?? [];
}
