import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { useIsDesktop, useIsMobile, useMediaQuery } from "./useMediaQuery";

type Listener = (event: MediaQueryListEvent) => void;

interface FakeMQL {
  matches: boolean;
  media: string;
  addEventListener: (type: "change", listener: Listener) => void;
  removeEventListener: (type: "change", listener: Listener) => void;
  trigger: (next: boolean) => void;
}

function installMatchMedia(initial: Record<string, boolean>) {
  const lists = new Map<string, FakeMQL>();
  function ensure(query: string): FakeMQL {
    let mql = lists.get(query);
    if (!mql) {
      const listeners = new Set<Listener>();
      mql = {
        matches: initial[query] ?? false,
        media: query,
        addEventListener: (_type, l) => listeners.add(l),
        removeEventListener: (_type, l) => listeners.delete(l),
        trigger(next: boolean) {
          this.matches = next;
          listeners.forEach((l) =>
            l({ matches: next, media: query } as MediaQueryListEvent),
          );
        },
      };
      lists.set(query, mql);
    }
    return mql;
  }
  vi.stubGlobal(
    "matchMedia",
    (query: string) => ensure(query) as unknown as MediaQueryList,
  );
  Object.defineProperty(window, "matchMedia", {
    configurable: true,
    writable: true,
    value: (query: string) => ensure(query) as unknown as MediaQueryList,
  });
  return { get: (q: string) => ensure(q) };
}

describe("useMediaQuery", () => {
  beforeEach(() => {
    vi.unstubAllGlobals();
  });
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("returns the current match value", () => {
    installMatchMedia({ "(max-width: 767px)": true });
    const { result } = renderHook(() => useMediaQuery("(max-width: 767px)"));
    expect(result.current).toBe(true);
  });

  it("updates when the media query changes", () => {
    const mm = installMatchMedia({ "(max-width: 767px)": false });
    const { result } = renderHook(() => useIsMobile());
    expect(result.current).toBe(false);
    act(() => mm.get("(max-width: 767px)").trigger(true));
    expect(result.current).toBe(true);
  });

  it("desktop helper resolves separately from mobile", () => {
    installMatchMedia({
      "(max-width: 767px)": false,
      "(min-width: 1024px)": true,
    });
    const { result: m } = renderHook(() => useIsMobile());
    const { result: d } = renderHook(() => useIsDesktop());
    expect(m.current).toBe(false);
    expect(d.current).toBe(true);
  });
});
