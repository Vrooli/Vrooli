import { beforeEach, describe, expect, it } from "vitest";
import type { ConversationEvent } from "../api/conversation";
import {
  getSessionConversationCursor,
  getSessionRefetchSinceSequence,
  getSessionUnlistenedEvents,
  getSessionUnreadCount,
  getSessionViewMode,
  useConversationStore,
  createConversationSessionState,
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

  it("preserves a live event when hydration returns a different baseline", () => {
    const store = useConversationStore.getState();
    store.appendEvent(event("live", 3));
    store.hydrateSession("s", [event("baseline", 1)], { lastSeenSequence: 0, lastListenedSequence: 0 });
    expect(useConversationStore.getState().sessions.s?.events.map((item) => item.id)).toEqual(["baseline", "live"]);
  });

  it("covers empty and duplicate prepend/merge paths and recomputes cursor metadata", () => {
    const store = useConversationStore.getState();
    store.prependEvents("new", [], { oldestSequence: 0, hasOlder: false, totalCount: 0 });
    store.prependEvents("new", [event("e1", 1)], { oldestSequence: 1, hasOlder: false, totalCount: 1 });
    store.prependEvents("new", [event("e1", 1)], { oldestSequence: 1, hasOlder: false, totalCount: 1 });
    store.mergeEvents("new", [event("e1", 1)], { lastSeenSequence: 1, lastListenedSequence: 0 });
    store.updateCursor("new", { lastSeenSequence: 1 });
    expect(getSessionUnreadCount(useConversationStore.getState(), "new")).toBe(0);

    useConversationStore.setState({
      sessions: { empty: createConversationSessionState({ events: [], cursor: { lastSeenSequence: 0, lastListenedSequence: 0 }, hydrated: false }) },
      viewModes: {},
    });
    useConversationStore.getState().mergeEvents("empty", [], undefined);
    expect(useConversationStore.getState().sessions.empty?.hydrated).toBe(true);
  });

  it("updates only existing events and supports sessions without cached metadata", () => {
    const store = useConversationStore.getState();
    store.updateEvent("missing", "e1", { summarized: true });
    store.hydrateSession("s", [event("e1", 1)], { lastSeenSequence: 0, lastListenedSequence: 0 });
    store.updateEvent("s", "missing", { summarized: true });
    store.updateEvent("s", "e1", { summarized: false });
    expect(useConversationStore.getState().sessions.s?.events[0]?.summarized).toBe(false);

    useConversationStore.setState({
      sessions: {
        legacy: createConversationSessionState({ events: [event("e2", 2), event("e4", 4)], cursor: { lastSeenSequence: 0, lastListenedSequence: 0 }, hydrated: true }),
      },
      viewModes: {},
    });
    expect(getSessionRefetchSinceSequence(useConversationStore.getState(), "legacy")).toBe(0);
    useConversationStore.setState({
      sessions: {
        legacy: createConversationSessionState({ events: [event("e1", 1), event("e2", 2)], cursor: { lastSeenSequence: 0, lastListenedSequence: 0 }, hydrated: true }),
      },
      viewModes: {},
    });
    expect(getSessionRefetchSinceSequence(useConversationStore.getState(), "legacy")).toBe(2);
  });

  it("reports missing, prefix, gap, and contiguous selector cases", () => {
    const state = useConversationStore.getState();
    expect(getSessionUnlistenedEvents(state, "missing")).toEqual([]);
    expect(getSessionUnreadCount(state, "missing")).toBe(0);
    expect(getSessionRefetchSinceSequence(state, "missing")).toBe(0);

    useConversationStore.setState({
      sessions: {
        prefix: createConversationSessionState({ events: [event("e3", 3)], cursor: { lastSeenSequence: 0, lastListenedSequence: 0 }, hydrated: true }),
        gap: createConversationSessionState({ events: [event("e1", 1), event("e3", 3)], cursor: { lastSeenSequence: 0, lastListenedSequence: 0 }, hydrated: true }),
        contiguous: createConversationSessionState({ events: [event("e1", 1), event("e2", 2)], cursor: { lastSeenSequence: 0, lastListenedSequence: 0 }, hydrated: true }),
      },
      viewModes: {},
    });
    expect(getSessionRefetchSinceSequence(useConversationStore.getState(), "prefix")).toBe(0);
    expect(getSessionRefetchSinceSequence(useConversationStore.getState(), "gap")).toBe(1);
    expect(getSessionRefetchSinceSequence(useConversationStore.getState(), "contiguous")).toBe(2);
  });
});

function getSessionConversationEventsFallback(state: ReturnType<typeof useConversationStore.getState>, id: string) {
  return state.sessions[id]?.events ?? [];
}
