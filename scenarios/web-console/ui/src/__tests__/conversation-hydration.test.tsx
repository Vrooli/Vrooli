import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useConversationHydration } from "../hooks/useConversationHydration";
import { useConversationStore } from "../stores/useConversationStore";

const { mockGetConversationSession } = vi.hoisted(() => ({
  mockGetConversationSession: vi.fn(),
}));

vi.mock("../api/conversation", () => ({
  getConversationSession: mockGetConversationSession,
}));

describe("useConversationHydration", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    useConversationStore.setState({ sessions: {}, viewModes: {} });
    mockGetConversationSession.mockReset();
    vi.spyOn(console, "warn").mockImplementation(() => {});
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  it("does not mark a failed hydrate as complete and retries later", async () => {
    mockGetConversationSession
      .mockRejectedValueOnce(new Error("offline"))
      .mockResolvedValueOnce({
        events: [{
          id: "evt-1",
          sessionId: "s1",
          source: "test",
          role: "assistant",
          text: "hello",
          speechParagraphs: ["hello"],
          summarized: false,
          createdAt: "2026-06-25T12:00:00Z",
          sequence: 1,
          deliveryState: "delivered",
          ttsState: "idle",
          consumptionState: "new",
        }],
        cursor: { lastSeenSequence: 0, lastListenedSequence: 0 },
      });

    renderHook(() => useConversationHydration(["s1"]));

    await act(async () => {
      await Promise.resolve();
    });
    expect(useConversationStore.getState().sessions.s1?.hydrated).toBeUndefined();
    expect(console.warn).toHaveBeenCalledWith(
      "[web-console] conversation hydration failed",
      expect.objectContaining({ sessionId: "s1" }),
    );

    await act(async () => {
      vi.advanceTimersByTime(1000);
      await Promise.resolve();
    });

    const session = useConversationStore.getState().sessions.s1;
    expect(session?.hydrated).toBe(true);
    expect(session?.events).toHaveLength(1);
  });
});
