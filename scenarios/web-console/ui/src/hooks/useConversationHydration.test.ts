import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import * as api from "../api/conversation";
import { UNKNOWN_CAPTURE } from "../api/messageCapture";
import { useConversationStore } from "../stores/useConversationStore";
import { refreshConversationSession } from "./useConversationSession";
import { useConversationHydration } from "./useConversationHydration";

function page(id: string): api.ConversationSessionResponse {
  return { sessionId: id, events: [], cursor: { lastSeenSequence: 4, lastListenedSequence: 2 },
    oldestSequence: 5, totalCount: 20, hasMore: true, capture: UNKNOWN_CAPTURE };
}

const pending = new Map<string, (response: api.ConversationSessionResponse) => void>();
async function finish(id: string) {
  await act(async () => { pending.get(id)?.(page(id)); pending.delete(id); await Promise.resolve(); });
}

describe("workspace conversation hydration", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    useConversationStore.setState({ sessions: {} });
    vi.spyOn(api, "getConversationSession").mockImplementation(id => new Promise(resolve => pending.set(id, resolve)));
  });
  afterEach(async () => {
    await act(async () => {
      for (const [id, resolve] of pending) resolve(page(id));
      pending.clear();
      await Promise.resolve();
    });
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  it("loads the active session first, coalesces foreground loads and bounds background work", async () => {
    const ids = ["a", "b", "c", "d", "active"];
    const { unmount } = renderHook(() => { useConversationHydration(ids, "active"); });
    const foreground = refreshConversationSession("active");
    expect(api.getConversationSession).toHaveBeenCalledTimes(1);
    expect(api.getConversationSession).toHaveBeenCalledWith("active", { limit: 500 });
    await finish("active");
    await expect(foreground).resolves.toMatchObject({ ok: true });
    await act(async () => { await vi.advanceTimersByTimeAsync(50); });
    expect([...pending.keys()]).toEqual(["a", "b"]);
    await finish("a");
    await act(async () => { await vi.advanceTimersByTimeAsync(50); });
    expect([...pending.keys()]).toEqual(["b", "c"]);
    await finish("b"); await finish("c");
    await act(async () => { await vi.advanceTimersByTimeAsync(50); });
    await finish("d");
    expect(Object.values(useConversationStore.getState().sessions)).toHaveLength(5);
    expect(useConversationStore.getState().sessions.d).toMatchObject({ hydrated: true, hasOlder: true, totalCount: 20, cursor: { lastSeenSequence: 4 } });
    unmount();
  });

  it("immediately loads a newly selected session while background pages are pending", async () => {
    const ids = ["a", "b", "c", "active"];
    const { rerender, unmount } = renderHook(({ active }) => { useConversationHydration(ids, active); }, { initialProps: { active: "active" } });
    await finish("active");
    await act(async () => { await vi.advanceTimersByTimeAsync(50); });
    expect([...pending.keys()]).toEqual(["a", "b"]);
    rerender({ active: "c" });
    expect([...pending.keys()]).toEqual(["a", "b", "c"]);
    await finish("c");
    expect(useConversationStore.getState().sessions.c?.hydrated).toBe(true);
    unmount();
  });

  it("records background failures, retries them and cancels queued work on unmount", async () => {
    vi.mocked(api.getConversationSession).mockRejectedValueOnce(new Error("offline"));
    const { unmount } = renderHook(() => { useConversationHydration(["retry"]); });
    await act(async () => { await vi.advanceTimersByTimeAsync(50); });
    expect(useConversationStore.getState().sessions.retry).toMatchObject({ status: "failed", hydrated: false });
    await act(async () => { await vi.advanceTimersByTimeAsync(1000); });
    expect(pending.has("retry")).toBe(true);
    await finish("retry");
    expect(useConversationStore.getState().sessions.retry?.status).toBe("loaded");
    unmount();
    await act(async () => { await vi.advanceTimersByTimeAsync(10_000); });
    expect(api.getConversationSession).toHaveBeenCalledTimes(2);
  });

  it("retains the background limit across session-list changes without reloading hydrated sessions", async () => {
    const { rerender, unmount } = renderHook(({ ids }) => { useConversationHydration(ids, "active"); }, { initialProps: { ids: ["active", "a", "b"] } });
    await finish("active");
    await act(async () => { await vi.advanceTimersByTimeAsync(50); });
    expect([...pending.keys()]).toEqual(["a", "b"]);
    rerender({ ids: ["active", "a", "b", "new"] });
    await act(async () => { await vi.advanceTimersByTimeAsync(50); });
    expect([...pending.keys()]).toEqual(["a", "b"]);
    expect(vi.mocked(api.getConversationSession).mock.calls.filter(([id]) => id === "active")).toHaveLength(1);
    await finish("a");
    expect(pending.size).toBeLessThanOrEqual(2);
    await finish("b");
    await act(async () => { await vi.advanceTimersByTimeAsync(50); });
    expect([...pending.keys()]).toEqual(["new"]);
    await finish("new");
    unmount();
  });

});
