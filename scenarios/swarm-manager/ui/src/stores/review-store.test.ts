import { describe, it, expect, beforeEach } from "vitest";
import { useReviewStore } from "./review-store";
import type { RequestThread } from "../services/review-service";

beforeEach(() => {
  // Reset store to initial state between tests.
  useReviewStore.setState({
    requestPanelOpen: false,
    requestTarget: null,
    activeThreadId: null,
    activeThread: null,
    isCreating: false,
    isSending: false,
  });
});

const mockThread: RequestThread = {
  id: "rt-1",
  status: "pending",
  messages: [{ role: "user", content: "Need more screenshots", timestamp: "2026-04-01T10:00:00Z" }],
  created_at: "2026-04-01T10:00:00Z",
};

describe("useReviewStore", () => {
  it("openRequestPanel sets target and opens panel", () => {
    useReviewStore.getState().openRequestPanel(2, "ev-5");

    const state = useReviewStore.getState();
    expect(state.requestPanelOpen).toBe(true);
    expect(state.requestTarget).toEqual({ round: 2, evidenceId: "ev-5" });
    expect(state.activeThreadId).toBeNull();
    expect(state.activeThread).toBeNull();
  });

  it("openRequestPanel works without evidenceId", () => {
    useReviewStore.getState().openRequestPanel(1);

    const state = useReviewStore.getState();
    expect(state.requestPanelOpen).toBe(true);
    expect(state.requestTarget).toEqual({ round: 1, evidenceId: undefined });
  });

  it("closeRequestPanel resets all state", () => {
    useReviewStore.getState().openRequestPanel(1, "ev-1");
    useReviewStore.getState().setActiveThread(mockThread);
    useReviewStore.getState().setCreating(true);
    useReviewStore.getState().setSending(true);

    useReviewStore.getState().closeRequestPanel();

    const state = useReviewStore.getState();
    expect(state.requestPanelOpen).toBe(false);
    expect(state.requestTarget).toBeNull();
    expect(state.activeThreadId).toBeNull();
    expect(state.activeThread).toBeNull();
    expect(state.isCreating).toBe(false);
    expect(state.isSending).toBe(false);
  });

  it("setActiveThread updates thread and threadId", () => {
    useReviewStore.getState().setActiveThread(mockThread);

    const state = useReviewStore.getState();
    expect(state.activeThread).toBe(mockThread);
    expect(state.activeThreadId).toBe("rt-1");
  });

  it("setActiveThread with null clears both fields", () => {
    useReviewStore.getState().setActiveThread(mockThread);
    useReviewStore.getState().setActiveThread(null);

    const state = useReviewStore.getState();
    expect(state.activeThread).toBeNull();
    expect(state.activeThreadId).toBeNull();
  });

  it("setCreating and setSending update their flags", () => {
    useReviewStore.getState().setCreating(true);
    expect(useReviewStore.getState().isCreating).toBe(true);

    useReviewStore.getState().setSending(true);
    expect(useReviewStore.getState().isSending).toBe(true);

    useReviewStore.getState().setCreating(false);
    useReviewStore.getState().setSending(false);
    expect(useReviewStore.getState().isCreating).toBe(false);
    expect(useReviewStore.getState().isSending).toBe(false);
  });
});
