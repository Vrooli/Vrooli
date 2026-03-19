/**
 * Tests for useMediaQuery / useIsMobile hooks.
 *
 * We stub window.matchMedia so tests are deterministic regardless of
 * the actual viewport.  Each test constructs its own MediaQueryList
 * mock so listener behaviour can be verified in isolation.
 */

import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useMediaQuery, useIsMobile, MOBILE_QUERY } from "./useMediaQuery";

/* ── helpers ─────────────────────────────────────────────────────── */

type ChangeListener = (e: MediaQueryListEvent) => void;

/** Create a controllable matchMedia stub. */
function createMatchMediaStub(initialMatch: boolean) {
  const listeners: ChangeListener[] = [];
  let currentMatches = initialMatch;

  const mql: MediaQueryList = {
    get matches() { return currentMatches; },
    media: "",
    onchange: null,
    addEventListener(_type: string, cb: EventListenerOrEventListenerObject) {
      if (typeof cb === "function") listeners.push(cb as ChangeListener);
    },
    removeEventListener(_type: string, cb: EventListenerOrEventListenerObject) {
      if (typeof cb === "function") {
        const idx = listeners.indexOf(cb as ChangeListener);
        if (idx !== -1) listeners.splice(idx, 1);
      }
    },
    addListener: vi.fn(),
    removeListener: vi.fn(),
    dispatchEvent: vi.fn(() => true),
  };

  /** Simulate a viewport change. */
  function fire(matches: boolean) {
    currentMatches = matches;
    for (const l of [...listeners]) {
      l({ matches } as MediaQueryListEvent);
    }
  }

  return { mql, fire, listeners };
}

/* ── setup / teardown ────────────────────────────────────────────── */

let originalMatchMedia: typeof window.matchMedia;

beforeEach(() => {
  originalMatchMedia = window.matchMedia;
});

afterEach(() => {
  window.matchMedia = originalMatchMedia;
});

/* ── useMediaQuery ───────────────────────────────────────────────── */

describe("useMediaQuery", () => {
  it("returns true when the query initially matches", () => {
    const { mql } = createMatchMediaStub(true);
    window.matchMedia = vi.fn(() => mql);

    const { result } = renderHook(() => useMediaQuery("(max-width: 767px)"));
    expect(result.current).toBe(true);
  });

  it("returns false when the query does not initially match", () => {
    const { mql } = createMatchMediaStub(false);
    window.matchMedia = vi.fn(() => mql);

    const { result } = renderHook(() => useMediaQuery("(max-width: 767px)"));
    expect(result.current).toBe(false);
  });

  it("updates when the media query starts matching", () => {
    const { mql, fire } = createMatchMediaStub(false);
    window.matchMedia = vi.fn(() => mql);

    const { result } = renderHook(() => useMediaQuery("(max-width: 767px)"));
    expect(result.current).toBe(false);

    act(() => fire(true));
    expect(result.current).toBe(true);
  });

  it("updates when the media query stops matching", () => {
    const { mql, fire } = createMatchMediaStub(true);
    window.matchMedia = vi.fn(() => mql);

    const { result } = renderHook(() => useMediaQuery("(max-width: 767px)"));
    expect(result.current).toBe(true);

    act(() => fire(false));
    expect(result.current).toBe(false);
  });

  it("cleans up listener on unmount", () => {
    const { mql, listeners } = createMatchMediaStub(false);
    window.matchMedia = vi.fn(() => mql);

    const { unmount } = renderHook(() => useMediaQuery("(max-width: 767px)"));
    expect(listeners).toHaveLength(1);

    unmount();
    expect(listeners).toHaveLength(0);
  });
});

/* ── useIsMobile ─────────────────────────────────────────────────── */

describe("useIsMobile", () => {
  it("delegates to useMediaQuery with MOBILE_QUERY", () => {
    const { mql } = createMatchMediaStub(true);
    const spy = vi.fn(() => mql);
    window.matchMedia = spy;

    renderHook(() => useIsMobile());

    // Should have been called with the canonical mobile query
    expect(spy).toHaveBeenCalledWith(MOBILE_QUERY);
  });

  it("returns true on a narrow viewport", () => {
    const { mql } = createMatchMediaStub(true);
    window.matchMedia = vi.fn(() => mql);

    const { result } = renderHook(() => useIsMobile());
    expect(result.current).toBe(true);
  });

  it("returns false on a wide viewport", () => {
    const { mql } = createMatchMediaStub(false);
    window.matchMedia = vi.fn(() => mql);

    const { result } = renderHook(() => useIsMobile());
    expect(result.current).toBe(false);
  });
});
