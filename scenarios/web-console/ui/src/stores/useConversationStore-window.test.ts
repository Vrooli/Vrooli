import { beforeEach, describe, expect, it } from "vitest";
import type { ConversationEvent } from "../api/conversation";
import { getSessionRefetchSinceSequence, useConversationStore } from "./useConversationStore";

const cursor = { lastSeenSequence: 0, lastListenedSequence: 0 };

function range(from: number, to: number): ConversationEvent[] {
  return Array.from({ length: to - from + 1 }, (_, index) => {
    const sequence = from + index;
    return {
      id: `e${String(sequence)}`, sessionId: "s", source: "claude_hook", role: "assistant", text: `m${String(sequence)}`,
      speechParagraphs: [], summarized: false, createdAt: "now", sequence, deliveryState: "received", ttsState: "idle",
      consumptionState: "seen",
    };
  });
}

const since = () => getSessionRefetchSinceSequence(useConversationStore.getState(), "s");

describe("refetch point for a paged (windowed) session", () => {
  beforeEach(() => { useConversationStore.setState({ sessions: {} }); });

  it("[REQ:P0-017b] a tail window refetches only what is newer than its end", () => {
    // The server returned the newest page: 743..1242 of 1,242, with older history available.
    useConversationStore.getState().hydrateSession("s", range(743, 1242), cursor, { oldestSequence: 743, hasOlder: true, totalCount: 1242 });
    expect(since()).toBe(1242);
  });

  it("[REQ:P0-017b] a page loaded around a saved place refetches from its end, not from 0", () => {
    useConversationStore.getState().setSessionWindow("s", range(1, 500), cursor, { oldestSequence: 1, hasOlder: false, totalCount: 1242 });
    expect(since()).toBe(500);
    useConversationStore.getState().setSessionWindow("s", range(300, 799), cursor, { oldestSequence: 300, hasOlder: true, totalCount: 1242 });
    expect(since()).toBe(799);
  });

  it("[REQ:P0-017b] an older page prepended to a tail window keeps the tail refetch point", () => {
    const store = useConversationStore.getState();
    store.hydrateSession("s", range(743, 1242), cursor, { oldestSequence: 743, hasOlder: true, totalCount: 1242 });
    store.prependEvents("s", range(243, 742), { oldestSequence: 243, hasOlder: true, totalCount: 1242 });
    expect(since()).toBe(1242);
  });

  it("[REQ:P0-017b] a real gap inside the window is still backfilled from before the gap", () => {
    const store = useConversationStore.getState();
    store.hydrateSession("s", range(743, 1242), cursor, { oldestSequence: 743, hasOlder: true, totalCount: 1250 });
    for (const event of range(1245, 1245)) store.appendEvent(event);
    expect(since()).toBe(1242);
  });

  it("[REQ:P0-017b] events that do not start at the window start are still a prefix gap", () => {
    // Live events arrived before any page: nothing says where history starts.
    const store = useConversationStore.getState();
    for (const event of range(5, 5)) store.appendEvent(event);
    expect(since()).toBe(0);
  });
});
