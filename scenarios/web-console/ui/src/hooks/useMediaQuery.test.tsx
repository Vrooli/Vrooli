import { renderHook, act } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { useMediaQuery } from "./useMediaQuery";

describe("useMediaQuery", () => {
  it("tracks matchMedia changes and removes its listener", () => {
    let listener: ((event: MediaQueryListEvent) => void) | undefined;
    const remove = vi.fn();
    vi.stubGlobal("matchMedia", vi.fn(() => ({
      matches: false,
      addEventListener: (_: string, cb: (event: MediaQueryListEvent) => void) => { listener = cb; },
      removeEventListener: remove,
    })));
    const { result, unmount } = renderHook(() => useMediaQuery("(min-width: 1px)"));
    expect(result.current).toBe(false);
    act(() => listener?.({ matches: true } as MediaQueryListEvent));
    expect(result.current).toBe(true);
    unmount();
    expect(remove).toHaveBeenCalledOnce();
    vi.unstubAllGlobals();
  });
});
