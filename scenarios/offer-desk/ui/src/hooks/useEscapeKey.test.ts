import { act, renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { useEscapeKey } from "./useEscapeKey";

describe("useEscapeKey", () => {
  it("invokes the callback only for Escape while enabled", async () => {
    const onEscape = vi.fn();
    const { rerender } = renderHook(({ enabled }) => useEscapeKey(enabled, onEscape), {
      initialProps: { enabled: true },
    });

    await act(() => document.dispatchEvent(new KeyboardEvent("keydown", { key: "Enter" })));
    expect(onEscape).not.toHaveBeenCalled();
    await act(() => document.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape" })));
    expect(onEscape).toHaveBeenCalledOnce();

    rerender({ enabled: false });
    await act(() => document.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape" })));
    expect(onEscape).toHaveBeenCalledOnce();
  });
});
