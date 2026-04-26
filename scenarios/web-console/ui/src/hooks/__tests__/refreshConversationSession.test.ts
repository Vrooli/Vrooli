import { describe, it, expect, vi, beforeEach } from "vitest";
import type { ConversationEvent } from "../../lib/api";
import { apiBaseMock } from "../../test-utils";

vi.mock("@vrooli/api-base", () => apiBaseMock());

import { refreshConversationSession } from "../useConversationSession";
import { useConversationStore } from "../../stores/useConversationStore";
import * as api from "../../lib/api";

const makeEvent = (id: string, sequence: number, text = "t"): ConversationEvent => ({
  id,
  sessionId: "s1",
  source: "codex_tailer",
  role: "assistant",
  text,
  speechParagraphs: [text],
  summarized: false,
  createdAt: new Date().toISOString(),
  sequence,
  deliveryState: "pending",
  ttsState: "idle",
  consumptionState: "unseen",
});

describe("refreshConversationSession", () => {
  beforeEach(() => {
    useConversationStore.setState({ sessions: {}, viewModes: {} });
  });

  it("requests since_sequence=<max local sequence> when the local store is gap-free and merges only missing events", async () => {
    useConversationStore.getState().hydrateSession("s1", [
      makeEvent("e1", 1, "one"),
      makeEvent("e2", 2, "two"),
    ], { lastSeenSequence: 0, lastListenedSequence: 0 });

    const spy = vi.spyOn(api, "getConversationSession").mockResolvedValue({
      sessionId: "s1",
      events: [makeEvent("e3", 3, "three"), makeEvent("e4", 4, "four")],
      cursor: { lastSeenSequence: 2, lastListenedSequence: 1 },
    });

    const ok = await refreshConversationSession("s1");
    expect(ok).toBe(true);
    expect(spy).toHaveBeenCalledWith("s1", { sinceSequence: 2 });

    const events = useConversationStore.getState().sessions["s1"]?.events ?? [];
    expect(events.map((e) => e.id)).toEqual(["e1", "e2", "e3", "e4"]);
    expect(useConversationStore.getState().sessions["s1"]?.cursor).toEqual({
      lastSeenSequence: 2,
      lastListenedSequence: 1,
    });
  });

  it("is idempotent: server returning already-known events does not duplicate", async () => {
    useConversationStore.getState().hydrateSession("s1", [
      makeEvent("e1", 1),
      makeEvent("e2", 2),
    ], { lastSeenSequence: 0, lastListenedSequence: 0 });

    vi.spyOn(api, "getConversationSession").mockResolvedValue({
      sessionId: "s1",
      events: [makeEvent("e1", 1), makeEvent("e2", 2)],
      cursor: { lastSeenSequence: 0, lastListenedSequence: 0 },
    });

    await refreshConversationSession("s1");
    await refreshConversationSession("s1");

    const events = useConversationStore.getState().sessions["s1"]?.events ?? [];
    expect(events).toHaveLength(2);
  });

  it("backfills internal sequence gaps by refetching from before the gap", async () => {
    // Local store has 1, 2, 4, 5 — sequence 3 is missing (e.g., from a
    // hydrate/append race or a dropped WS event the resync didn't recover).
    useConversationStore.getState().hydrateSession("s1", [
      makeEvent("e1", 1),
      makeEvent("e2", 2),
      makeEvent("e4", 4),
      makeEvent("e5", 5),
    ], { lastSeenSequence: 0, lastListenedSequence: 0 });

    const spy = vi.spyOn(api, "getConversationSession").mockResolvedValue({
      sessionId: "s1",
      events: [makeEvent("e3", 3), makeEvent("e4", 4), makeEvent("e5", 5), makeEvent("e6", 6)],
      cursor: { lastSeenSequence: 0, lastListenedSequence: 0 },
    });

    const ok = await refreshConversationSession("s1");
    expect(ok).toBe(true);
    // since_sequence must be 2 (the last contiguous sequence before the gap),
    // not 5 (the max), so the server returns the missing #3.
    expect(spy).toHaveBeenCalledWith("s1", { sinceSequence: 2 });

    const events = useConversationStore.getState().sessions["s1"]?.events ?? [];
    expect(events.map((e) => e.sequence)).toEqual([1, 2, 3, 4, 5, 6]);
  });

  it("refetches from 0 when the local store is missing the prefix (first sequence > 1)", async () => {
    useConversationStore.getState().hydrateSession("s1", [
      makeEvent("e3", 3),
      makeEvent("e4", 4),
    ], { lastSeenSequence: 0, lastListenedSequence: 0 });

    const spy = vi.spyOn(api, "getConversationSession").mockResolvedValue({
      sessionId: "s1",
      events: [makeEvent("e1", 1), makeEvent("e2", 2), makeEvent("e3", 3), makeEvent("e4", 4)],
      cursor: { lastSeenSequence: 0, lastListenedSequence: 0 },
    });

    await refreshConversationSession("s1");
    expect(spy).toHaveBeenCalledWith("s1", { sinceSequence: 0 });
    const events = useConversationStore.getState().sessions["s1"]?.events ?? [];
    expect(events.map((e) => e.sequence)).toEqual([1, 2, 3, 4]);
  });

  it("hydrateSession preserves WS-appended events that arrived during the in-flight GET", () => {
    // Simulate the race: WS event seq=3 lands first via appendEvent.
    useConversationStore.getState().appendEvent(makeEvent("e3", 3, "live"));

    // Then the initial GET response returns events 1..2 — these were the only
    // events that existed when the request was issued. Without merge-aware
    // hydrate, e3 would be wiped out, leaving a permanent gap.
    useConversationStore.getState().hydrateSession("s1", [
      makeEvent("e1", 1),
      makeEvent("e2", 2),
    ], { lastSeenSequence: 0, lastListenedSequence: 0 });

    const events = useConversationStore.getState().sessions["s1"]?.events ?? [];
    expect(events.map((e) => e.sequence)).toEqual([1, 2, 3]);
  });

  it("returns false and preserves store on fetch error", async () => {
    useConversationStore.getState().hydrateSession("s1", [makeEvent("e1", 1)], {
      lastSeenSequence: 0,
      lastListenedSequence: 0,
    });
    vi.spyOn(api, "getConversationSession").mockRejectedValue(new Error("network"));

    const ok = await refreshConversationSession("s1");
    expect(ok).toBe(false);

    const events = useConversationStore.getState().sessions["s1"]?.events ?? [];
    expect(events).toHaveLength(1);
  });
});
