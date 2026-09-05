import { renderHook, act } from "@testing-library/react";
import { describe, it, expect, vi, afterEach } from "vitest";
import { useTouchControls } from "./useTouchControls";

type MockMediaQueryList = MediaQueryList & {
  setMatches: (matches: boolean) => void;
};

function installMatchMedia(initialMatches: Record<string, boolean>) {
  const queries = new Map<string, MockMediaQueryList>();

  vi.stubGlobal("matchMedia", vi.fn((query: string) => {
    const existing = queries.get(query);
    if (existing) return existing;
    const listeners = new Set<(event: MediaQueryListEvent) => void>();
    const mql = {
      media: query,
      matches: initialMatches[query] ?? false,
      onchange: null,
      addEventListener: vi.fn((_type: string, cb: (event: MediaQueryListEvent) => void) => {
        listeners.add(cb);
      }),
      removeEventListener: vi.fn((_type: string, cb: (event: MediaQueryListEvent) => void) => {
        listeners.delete(cb);
      }),
      addListener: vi.fn(),
      removeListener: vi.fn(),
      dispatchEvent: vi.fn(),
      setMatches(matches: boolean) {
        (mql as unknown as { matches: boolean }).matches = matches;
        const event = { matches, media: query } as MediaQueryListEvent;
        for (const cb of listeners) cb(event);
      },
    } as unknown as MockMediaQueryList;
    queries.set(query, mql);
    return mql;
  }));

  return queries;
}

describe("useTouchControls", () => {
  const originalMaxTouchPoints = navigator.maxTouchPoints;

  afterEach(() => {
    Object.defineProperty(navigator, "maxTouchPoints", {
      configurable: true,
      value: originalMaxTouchPoints,
    });
    vi.unstubAllGlobals();
  });

  it("returns true when the device reports touch points", () => {
    Object.defineProperty(navigator, "maxTouchPoints", {
      configurable: true,
      value: 5,
    });
    installMatchMedia({});

    const { result } = renderHook(() => useTouchControls());

    expect(result.current).toBe(true);
  });

  it("returns true for coarse pointer media queries on wide viewports", () => {
    Object.defineProperty(navigator, "maxTouchPoints", {
      configurable: true,
      value: 0,
    });
    installMatchMedia({ "(pointer: coarse)": true });

    const { result } = renderHook(() => useTouchControls());

    expect(result.current).toBe(true);
  });

  it("updates when pointer media changes", () => {
    Object.defineProperty(navigator, "maxTouchPoints", {
      configurable: true,
      value: 0,
    });
    const queries = installMatchMedia({ "(pointer: coarse)": false, "(hover: none)": false });

    const { result } = renderHook(() => useTouchControls());
    expect(result.current).toBe(false);

    act(() => {
      queries.get("(pointer: coarse)")?.setMatches(true);
    });

    expect(result.current).toBe(true);
  });
});
