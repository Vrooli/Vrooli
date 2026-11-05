import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act, cleanup } from "@testing-library/react";
import { useIsMobile } from "./useIsMobile";

type MockMql = {
  matches: boolean;
  addEventListener: ReturnType<typeof vi.fn>;
  removeEventListener: ReturnType<typeof vi.fn>;
};

let mql: MockMql;
let changeListener: (() => void) | null = null;

function setupMatchMedia(matches: boolean) {
  mql = {
    matches,
    addEventListener: vi.fn((_event: string, cb: () => void) => {
      changeListener = cb;
    }),
    removeEventListener: vi.fn(),
  };
  Object.defineProperty(window, "matchMedia", {
    writable: true,
    value: vi.fn().mockReturnValue(mql),
  });
}

afterEach(() => {
  cleanup();
  changeListener = null;
});

describe("useIsMobile", () => {
  describe("desktop viewport", () => {
    beforeEach(() => setupMatchMedia(false));

    it("returns false when matchMedia.matches is false", () => {
      const { result } = renderHook(() => useIsMobile());
      expect(result.current).toBe(false);
    });

    it("registers a change listener on mount", () => {
      renderHook(() => useIsMobile());
      expect(mql.addEventListener).toHaveBeenCalledWith("change", expect.any(Function));
    });

    it("removes the change listener on unmount", () => {
      const { unmount } = renderHook(() => useIsMobile());
      unmount();
      expect(mql.removeEventListener).toHaveBeenCalledWith("change", expect.any(Function));
    });

    it("switches to true when the MQL fires a change event", () => {
      const { result } = renderHook(() => useIsMobile());
      expect(result.current).toBe(false);
      act(() => {
        mql.matches = true;
        if (changeListener) changeListener();
      });
      expect(result.current).toBe(true);
    });
  });

  describe("mobile viewport", () => {
    beforeEach(() => setupMatchMedia(true));

    it("returns true when matchMedia.matches is true", () => {
      const { result } = renderHook(() => useIsMobile());
      expect(result.current).toBe(true);
    });

    it("switches to false when the MQL fires a change event", () => {
      const { result } = renderHook(() => useIsMobile());
      expect(result.current).toBe(true);
      act(() => {
        mql.matches = false;
        if (changeListener) changeListener();
      });
      expect(result.current).toBe(false);
    });
  });
});
