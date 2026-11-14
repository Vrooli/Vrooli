import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { saveWorkspaceLayout } from "../api/workspace";
import { useWorkspaceSync } from "./useWorkspaceSync";

vi.mock("../api/workspace", () => ({
  saveWorkspaceLayout: vi.fn(), updateWorkspacePane: vi.fn(), createTabGroup: vi.fn(),
  updateTabGroup: vi.fn(), deleteTabGroup: vi.fn(),
}));

describe("workspace reorder persistence lifecycle", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.mocked(saveWorkspaceLayout).mockReset().mockResolvedValue(undefined);
  });
  afterEach(() => { vi.useRealTimers(); });

  it("debounces to the latest order and does not save twice after completion", async () => {
    const { result, unmount } = renderHook(() => useWorkspaceSync());
    act(() => {
      result.current.syncPaneOrder(["a", "b"], "a");
      result.current.syncPaneOrder(["b", "a"], "b");
    });
    expect(saveWorkspaceLayout).not.toHaveBeenCalled();
    await act(async () => { await vi.advanceTimersByTimeAsync(300); });
    expect(saveWorkspaceLayout).toHaveBeenCalledTimes(1);
    expect(saveWorkspaceLayout).toHaveBeenCalledWith({ active_pane: "b", pane_order: ["b", "a"] });
    unmount();
    expect(saveWorkspaceLayout).toHaveBeenCalledTimes(1);
  });

  it("flushes the final order on unmount and leaves no delayed backend write", async () => {
    const { result, unmount } = renderHook(() => useWorkspaceSync());
    act(() => { result.current.syncPaneOrder(["b", "a"], "b"); });
    unmount();
    expect(saveWorkspaceLayout).toHaveBeenCalledTimes(1);
    expect(saveWorkspaceLayout).toHaveBeenCalledWith({ active_pane: "b", pane_order: ["b", "a"] });
    await act(async () => { await vi.advanceTimersByTimeAsync(300); });
    expect(saveWorkspaceLayout).toHaveBeenCalledTimes(1);
  });

  it("handles a failed final flush without leaving an unhandled rejection", async () => {
    const error = new Error("offline");
    vi.mocked(saveWorkspaceLayout).mockRejectedValueOnce(error);
    const log = vi.spyOn(console, "error").mockImplementation(() => {});
    const { result, unmount } = renderHook(() => useWorkspaceSync());
    act(() => { result.current.syncPaneOrder(["a"], "a"); });
    await act(async () => { unmount(); await Promise.resolve(); });
    expect(log).toHaveBeenCalledWith("Failed to sync pane order:", error);
    log.mockRestore();
  });
});
