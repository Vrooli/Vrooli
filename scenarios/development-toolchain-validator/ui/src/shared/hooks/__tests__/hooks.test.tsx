/**
 * Hook tests.
 */
import { describe, expect, it, vi } from "vitest";
import { act, renderHook } from "@testing-library/react";

import { useMediaQuery, useIsMobile } from "../useMediaQuery";
import { useGlobalKeydown } from "../../../hooks/useGlobalKeydown";
import { useLocalStorage } from "../useLocalStorage";

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
  });

  it("falls back to the default when storage is empty", () => {
    const { result } = renderHook(() => useLocalStorage<number>("k2", 7));
    expect(result.current[0]).toBe(7);
  });
});
