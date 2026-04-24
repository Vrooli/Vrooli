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

  it("requests since_sequence=<max local sequence> and merges only missing events", async () => {
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
