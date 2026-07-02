/**
 * Hook tests.
 */
import { describe, expect, it, vi } from "vitest";
import { act, renderHook } from "@testing-library/react";

import { useMediaQuery, useIsMobile } from "../useMediaQuery";
import { useGlobalKeydown } from "../../../hooks/useGlobalKeydown";
import { useLocalStorage } from "../useLocalStorage";
import { useAppViewport } from "../useAppViewport";

describe("useMediaQuery", () => {
  it("reports the current matchMedia state and updates on change", () => {
    const listeners = new Set<(e: MediaQueryListEvent) => void>();
    let currentMatches = false;
    const mql = {
      get matches() {
        return currentMatches;
      },
      media: "(max-width: 767px)",
      addEventListener: (_: string, cb: (e: MediaQueryListEvent) => void) => listeners.add(cb),
      removeEventListener: (_: string, cb: (e: MediaQueryListEvent) => void) => listeners.delete(cb),
      onchange: null,
    } as unknown as MediaQueryList;

    const spy = vi.spyOn(window, "matchMedia").mockImplementation(() => mql);
    try {
      const { result } = renderHook(() => useMediaQuery("(max-width: 767px)"));
      expect(result.current).toBe(false);

      act(() => {
        currentMatches = true;
        listeners.forEach((cb) => cb({ matches: true } as MediaQueryListEvent));
      });
      expect(result.current).toBe(true);
    } finally {
      spy.mockRestore();
    }
  });
});

describe("useIsMobile", () => {
  it("delegates to useMediaQuery with the canonical mobile query", () => {
    const spy = vi.spyOn(window, "matchMedia").mockImplementation((q) => {
      return {
        matches: q === "(max-width: 767px)",
        media: q,
        addEventListener: () => undefined,
        removeEventListener: () => undefined,
        onchange: null,
      } as unknown as MediaQueryList;
    });
    try {
      const { result } = renderHook(() => useIsMobile());
      expect(result.current).toBe(true);
    } finally {
      spy.mockRestore();
    }
  });
});

describe("useGlobalKeydown", () => {
  it("invokes the handler for each chord position and fires the consume reset only on match", () => {
    // Return true only when the sequence is "g g" — the hook resets the
    // buffer when the handler signals consumption.
    const handler = vi.fn((seq: string) => seq === "g g");
    renderHook(() => useGlobalKeydown(handler));

    document.dispatchEvent(new KeyboardEvent("keydown", { key: "g" }));
    document.dispatchEvent(new KeyboardEvent("keydown", { key: "g" }));
    const seqs = handler.mock.calls.map((c) => c[0]);
    expect(seqs).toContain("g");
    expect(seqs).toContain("g g");
  });

  it("ignores keydown events fired while focus is in an input", () => {
    const handler = vi.fn().mockReturnValue(false);
    renderHook(() => useGlobalKeydown(handler));

    const input = document.createElement("input");
    document.body.appendChild(input);
    input.focus();
    // Dispatching from the input bubbles to document with input as target.
    input.dispatchEvent(new KeyboardEvent("keydown", { key: "g", bubbles: true }));
    expect(handler).not.toHaveBeenCalled();
    input.remove();
  });

  it("normalizes modifiers and ignores already-prevented events", () => {
    const handler = vi.fn().mockReturnValue(false);
    renderHook(() => useGlobalKeydown(handler));

    const prevented = new KeyboardEvent("keydown", { key: "x", bubbles: true, cancelable: true });
    prevented.preventDefault();
    document.dispatchEvent(prevented);
    expect(handler).not.toHaveBeenCalled();

    document.dispatchEvent(new KeyboardEvent("keydown", { key: "Enter", shiftKey: true }));
    expect(handler).toHaveBeenLastCalledWith("shift+Enter", expect.any(KeyboardEvent));
    document.dispatchEvent(new KeyboardEvent("keydown", { key: "k", metaKey: true }));
    expect(handler).toHaveBeenLastCalledWith("shift+Enter meta+k", expect.any(KeyboardEvent));
  });
});

describe("useLocalStorage", () => {
  it("reads the initial value from storage and writes on update", () => {
    window.localStorage.setItem("k1", JSON.stringify("seed"));
    const { result } = renderHook(() => useLocalStorage<string>("k1", "default"));
    expect(result.current[0]).toBe("seed");

    act(() => {
      result.current[1]("changed");
    });
    expect(result.current[0]).toBe("changed");
    expect(JSON.parse(window.localStorage.getItem("k1") ?? "")).toBe("changed");

    act(() => {
      result.current[1]((prev) => `${prev}-again`);
    });
    expect(result.current[0]).toBe("changed-again");
  });

  it("falls back to the default when storage is empty", () => {
    const { result } = renderHook(() => useLocalStorage<number>("k2", 7));
    expect(result.current[0]).toBe(7);
  });

  it("falls back to the default when stored JSON is invalid", () => {
    window.localStorage.setItem("bad-json", "{");
    const { result } = renderHook(() => useLocalStorage("bad-json", "fallback"));
    expect(result.current[0]).toBe("fallback");
  });
});

describe("useAppViewport", () => {
  it("writes and cleans up viewport height listeners", () => {
    const listeners = new Map<string, EventListener>();
    const visualViewport = {
      height: 321,
      addEventListener: vi.fn((event: string, listener: EventListener) => {
        listeners.set(event, listener);
      }),
      removeEventListener: vi.fn(),
    };
    Object.defineProperty(window, "visualViewport", {
      configurable: true,
      value: visualViewport,
    });

    const { unmount } = renderHook(() => useAppViewport());
    expect(document.documentElement.style.getPropertyValue("--app-height")).toBe("321px");

    visualViewport.height = 654;
    act(() => {
      listeners.get("resize")?.(new Event("resize"));
    });
    expect(document.documentElement.style.getPropertyValue("--app-height")).toBe("654px");

    unmount();
    expect(visualViewport.removeEventListener).toHaveBeenCalled();
  });
});
