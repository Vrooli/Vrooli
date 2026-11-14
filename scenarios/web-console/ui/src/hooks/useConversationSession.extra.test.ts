import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import * as api from "../api/conversation";
import type { ConversationEvent } from "../api/conversation";
import { useConversationStore } from "../stores/useConversationStore";
import { loadConversationPageContaining, loadOlderConversationPage, refreshConversationSession, useConversationSession } from "./useConversationSession";

const event: ConversationEvent = { id: "e1", sessionId: "s", source: "agent", role: "assistant", text: "hello", speechParagraphs: ["hello"], summarized: false, createdAt: "now", deliveryState: "received", ttsState: "idle", consumptionState: "new", sequence: 1 };
const page = { sessionId: "s", events: [event], cursor: { lastSeenSequence: 1, lastListenedSequence: 0 }, hasMore: true, oldestSequence: 1, totalCount: 1 };

describe("conversation session orchestration", () => {
  beforeEach(() => {
    useConversationStore.setState({ sessions: {} });
    vi.restoreAllMocks();
  });

  it("hydrates, refreshes gaps, loads older pages, and persists cursors", async () => {
    const get = vi.spyOn(api, "getConversationSession").mockResolvedValue(page as never);
    const update = vi.spyOn(api, "updateConversationCursor").mockResolvedValue({ lastSeenSequence: 2, lastListenedSequence: 1 });
    await expect(refreshConversationSession("s")).resolves.toMatchObject({ ok: true, addedEvents: 1 });
    expect(useConversationStore.getState().sessions.s?.events).toHaveLength(1);
    // A second refresh returning the same event adds nothing — and says so,
    // which is what lets the pane report "up to date" instead of going quiet.
    await expect(refreshConversationSession("s")).resolves.toMatchObject({ ok: true, addedEvents: 0 });
    await expect(loadOlderConversationPage("s")).resolves.toBe(false);
    await expect(loadConversationPageContaining("s", 1)).resolves.toBe(true);
    expect(get).toHaveBeenCalled();

    const { result } = renderHook(() => useConversationSession("s", { hydrate: false }));
    act(() => { result.current.appendConversationEvent({ ...event, id: "e2", sequence: 2 } as never); });
    await act(async () => { await result.current.persistCursor({ lastSeenSequence: 2 }); });
    expect(update).toHaveBeenCalledWith("s", { lastSeenSequence: 2 });
  });

  it("reports a typed failure on failed fetches instead of an empty session", async () => {
    vi.spyOn(api, "getConversationSession").mockRejectedValue(new Error("offline"));
    await expect(refreshConversationSession("missing")).resolves.toMatchObject({
      ok: false,
      error: { message: "offline", retryable: true },
    });
    await expect(loadConversationPageContaining("s", 0)).resolves.toBe(false);
    const { unmount } = renderHook(() => useConversationSession("s"));
    await waitFor(() => expect(useConversationStore.getState().sessions.s).toBeDefined());
    // The critical distinction: a failed load leaves a session marked failed,
    // never a hydrated one with zero events, which the view would have shown
    // as "no messages yet".
    const session = useConversationStore.getState().sessions.s;
    expect(session?.events).toEqual([]);
    expect(session?.status).toBe("failed");
    expect(session?.hydrated).toBe(false);
    unmount();
  });

  it("handles bounded-page guards, duplicate events, and optimistic cursor failures", async () => {
    expect(await loadConversationPageContaining("s", 0)).toBe(false);
    expect(await loadOlderConversationPage("missing")).toBe(false);
    useConversationStore.getState().hydrateSession("s", [event], page.cursor, { oldestSequence: 1, hasOlder: true, totalCount: 1 });
    vi.spyOn(api, "getConversationSession").mockResolvedValue({ ...page, events: [{ ...event }], hasMore: false } as never);
    expect(await loadOlderConversationPage("s")).toBe(false);
    expect(await loadConversationPageContaining("s", 5)).toBe(false);
    vi.spyOn(api, "getConversationSession").mockRejectedValue(new Error("offline"));
    expect(await loadOlderConversationPage("s")).toBe(false);
    const update = vi.spyOn(api, "updateConversationCursor").mockRejectedValue(new Error("offline"));
    const { result } = renderHook(() => useConversationSession("s", { hydrate: false }));
    await act(async () => { await result.current.persistCursor({ lastListenedSequence: 3 }); });
    expect(update).toHaveBeenCalledWith("s", { lastListenedSequence: 3 });
  });
});
