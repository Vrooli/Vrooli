import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import * as api from "../api/conversation";
import { useConversationStore } from "../stores/useConversationStore";
import { loadConversationPageContaining, loadOlderConversationPage, refreshConversationSession, useConversationSession } from "./useConversationSession";

const event = { id: "e1", sessionId: "s", source: "agent", role: "assistant", text: "hello", speechParagraphs: ["hello"], summarized: false, createdAt: "now", deliveryState: "received", ttsState: "idle", consumptionState: "new", sequence: 1 } as const;
const page = { sessionId: "s", events: [event], cursor: { lastSeenSequence: 1, lastListenedSequence: 0 }, hasMore: true, oldestSequence: 1, totalCount: 1 };

describe("conversation session orchestration", () => {
  beforeEach(() => {
    useConversationStore.setState({ sessions: {} });
    vi.restoreAllMocks();
  });

  it("hydrates, refreshes gaps, loads older pages, and persists cursors", async () => {
    const get = vi.spyOn(api, "getConversationSession").mockResolvedValue(page as never);
    const update = vi.spyOn(api, "updateConversationCursor").mockResolvedValue({ lastSeenSequence: 2, lastListenedSequence: 1 });
    await expect(refreshConversationSession("s")).resolves.toBe(true);
    expect(useConversationStore.getState().sessions.s?.events).toHaveLength(1);
    await expect(refreshConversationSession("s")).resolves.toBe(true);
    await expect(loadOlderConversationPage("s")).resolves.toBe(false);
    await expect(loadConversationPageContaining("s", 1)).resolves.toBe(true);
    expect(get).toHaveBeenCalled();

    const { result } = renderHook(() => useConversationSession("s", { hydrate: false }));
    act(() => { result.current.appendConversationEvent({ ...event, id: "e2", sequence: 2 } as never); });
    await act(async () => { await result.current.persistCursor({ lastSeenSequence: 2 }); });
    expect(update).toHaveBeenCalledWith("s", { lastSeenSequence: 2 });
  });

  it("returns false on failed fetches and falls back when hook hydration fails", async () => {
    vi.spyOn(api, "getConversationSession").mockRejectedValue(new Error("offline"));
    await expect(refreshConversationSession("missing")).resolves.toBe(false);
    await expect(loadConversationPageContaining("s", 0)).resolves.toBe(false);
    const { unmount } = renderHook(() => useConversationSession("s"));
    await waitFor(() => expect(useConversationStore.getState().sessions.s).toBeDefined());
    expect(useConversationStore.getState().sessions.s?.events).toEqual([]);
    unmount();
  });
});
