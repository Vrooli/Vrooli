import { fireEvent, renderHook, act } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { getTabFromHash, useActiveTab } from "./useActiveTab";
import { useEscapeKey } from "./useEscapeKey";

vi.mock("@vrooli/iframe-bridge", () => ({ emitShortcutIntent: vi.fn() }));

describe("navigation hooks", () => {
  beforeEach(() => {
    window.location.hash = "";
  });

  it("reads every supported tab hash and defaults safely", () => {
    for (const [hash, tab] of [["#trends", "trends"], ["#timeline", "timeline"], ["#incidents", "incidents"], ["#docs?path=x", "docs"], ["#docs", "docs"], ["#other", "dashboard"]] as const) {
      window.location.hash = hash;
      expect(getTabFromHash()).toBe(tab);
    }
  });

  it("updates active tab and responds to hash changes", () => {
    const { result } = renderHook(() => useActiveTab());
    act(() => result.current.handleTabChange("docs"));
    expect(result.current.activeTab).toBe("docs");
    expect(window.location.hash).toBe("#docs");
    act(() => {
      window.location.hash = "#timeline";
      window.dispatchEvent(new HashChangeEvent("hashchange"));
    });
    expect(result.current.activeTab).toBe("timeline");
  });

  it("handles Escape only while enabled", () => {
    const onEscape = vi.fn();
    renderHook(() => useEscapeKey(onEscape));
    fireEvent.keyDown(window, { key: "Enter" });
    fireEvent.keyDown(window, { key: "Escape" });
    expect(onEscape).toHaveBeenCalledTimes(1);

    const disabled = vi.fn();
    renderHook(() => useEscapeKey(disabled, false));
    fireEvent.keyDown(window, { key: "Escape" });
    expect(disabled).not.toHaveBeenCalled();
  });
});
