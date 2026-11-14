import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useFilePreviewDeepLink } from "./useFilePreviewDeepLink";
import { useFilePreviewDeepLinkStore, parseFilePreviewDeepLink, stripFilePreviewDeepLink } from "../stores/useFilePreviewDeepLinkStore";

describe("file preview deep links", () => {
  beforeEach(() => { useFilePreviewDeepLinkStore.getState().clear(); });

  it("parses a path and strips only the file parameter", () => {
    expect(parseFilePreviewDeepLink("?file=%2Fworkspace%2Fnotes%20draft.md&approval=ask-1")).toBe("/workspace/notes draft.md");
    expect(stripFilePreviewDeepLink("https://console.example/workspace?file=%2Ftmp%2Fnote.md&approval=ask-1&theme=dark#terminal"))
      .toBe("/workspace?approval=ask-1&theme=dark#terminal");
    expect(parseFilePreviewDeepLink("?approval=ask-1")).toBeNull();
  });

  it("waits for hydration before opening the active session", async () => {
    const open = vi.fn();
    const { rerender } = renderHook((props: { hydrated: boolean }) => useFilePreviewDeepLink({
      hydrated: props.hydrated,
      activeSessionId: "session-1",
      hasActiveSession: true,
      open,
    }), { initialProps: { hydrated: false } });
    act(() => { useFilePreviewDeepLinkStore.getState().request("/workspace/report.md"); });
    expect(open).not.toHaveBeenCalled();

    rerender({ hydrated: true });
    await waitFor(() => expect(open).toHaveBeenCalledWith("session-1", "/workspace/report.md"));
    expect(useFilePreviewDeepLinkStore.getState().path).toBeNull();
  });

  it("shows a retryable error when no active session exists and opens after retry", async () => {
    const open = vi.fn();
    const { result, rerender } = renderHook((props: { sessionId: string | null }) => useFilePreviewDeepLink({
      hydrated: true,
      activeSessionId: props.sessionId,
      hasActiveSession: Boolean(props.sessionId),
      open,
    }), { initialProps: { sessionId: null as string | null } });
    act(() => { useFilePreviewDeepLinkStore.getState().request("/workspace/report.md"); });
    await waitFor(() => expect(result.current.error).toMatch(/No active session/));
    expect(open).not.toHaveBeenCalled();

    act(() => { result.current.retry(); });
    rerender({ sessionId: "session-2" });
    await waitFor(() => expect(open).toHaveBeenCalledWith("session-2", "/workspace/report.md"));
    expect(useFilePreviewDeepLinkStore.getState().path).toBeNull();
  });
});
