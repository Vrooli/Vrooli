import { act, cleanup, renderHook } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { useIsTablet } from "./useMediaQuery";

describe("useMediaQuery helpers", () => {
  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
  });

  it("subscribes to media-query changes and removes the listener", () => {
    let listener: ((event: MediaQueryListEvent) => void) | undefined;
    const removeEventListener = vi.fn();
    vi.stubGlobal("matchMedia", () => ({
      matches: false,
      addEventListener: vi.fn((_type: string, callback: (event: MediaQueryListEvent) => void) => {
        listener = callback;
      }),
      removeEventListener,
    }));
    const { result, unmount } = renderHook(() => useIsTablet());
    expect(result.current).toBe(false);
    act(() => listener?.({ matches: true } as MediaQueryListEvent));
    expect(result.current).toBe(true);
    unmount();
    expect(removeEventListener).toHaveBeenCalledOnce();
  });
});
