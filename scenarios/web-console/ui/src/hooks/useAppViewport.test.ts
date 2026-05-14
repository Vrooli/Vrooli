import { renderHook } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { useAppViewport } from "./useAppViewport";

/* ── Helpers ───────────────────────────────────────────────────────── */

/** Minimal stub for the VisualViewport API used by the hook. */
type MockVisualViewport = VisualViewport & {
  height: number;
  offsetTop: number;
  _fire: (type: string) => void;
};

function createMockVisualViewport(overrides: Partial<VisualViewport> = {}) {
  const listeners = new Map<string, Set<EventListener>>();
  return {
    height: 800,
    width: 400,
    offsetTop: 0,
    offsetLeft: 0,
    pageTop: 0,
    pageLeft: 0,
    scale: 1,
    addEventListener: vi.fn((type: string, cb: EventListener) => {
      let callbacks = listeners.get(type);
      if (!callbacks) {
        callbacks = new Set<EventListener>();
        listeners.set(type, callbacks);
      }
      callbacks.add(cb);
    }),
    removeEventListener: vi.fn((type: string, cb: EventListener) => {
      listeners.get(type)?.delete(cb);
    }),
    dispatchEvent: vi.fn(),
    onresize: null,
    onscroll: null,
    /** Fire all listeners registered for the given event type. */
    _fire(type: string) {
      for (const cb of listeners.get(type) ?? []) cb(new Event(type));
    },
    ...overrides,
  } as unknown as MockVisualViewport;
}

function getCssVar(name: string): string | undefined {
  return document.documentElement.style.getPropertyValue(name) || undefined;
}

/* ── Tests ─────────────────────────────────────────────────────────── */

describe("useAppViewport", () => {
  let mockVV: ReturnType<typeof createMockVisualViewport>;
  const originalInnerHeight = window.innerHeight;

  beforeEach(() => {
    vi.useFakeTimers();
    mockVV = createMockVisualViewport();
    Object.defineProperty(window, "visualViewport", {
      value: mockVV,
      writable: true,
      configurable: true,
    });
    Object.defineProperty(window, "innerHeight", {
      value: 800,
      writable: true,
      configurable: true,
    });
  });

  afterEach(() => {
    vi.useRealTimers();
    document.documentElement.style.removeProperty("--wc-app-height");
    document.documentElement.style.removeProperty("--wc-kb-height");
    document.documentElement.style.removeProperty("--wc-safe-top");
    document.documentElement.style.removeProperty("--wc-safe-bottom");
    Object.defineProperty(window, "innerHeight", {
      value: originalInnerHeight,
      writable: true,
      configurable: true,
    });
    vi.restoreAllMocks();
  });

  it("sets initial CSS vars on mount", () => {
    renderHook(() => useAppViewport());

    expect(getCssVar("--wc-app-height")).toBe("800px");
    expect(getCssVar("--wc-kb-height")).toBe("0px");
    expect(getCssVar("--wc-safe-top")).toBe("env(safe-area-inset-top)");
    expect(getCssVar("--wc-safe-bottom")).toBe("env(safe-area-inset-bottom)");
  });

  it("computes keyboard height correctly and clears safe-bottom", () => {
    // Simulate keyboard taking 300px: innerHeight stays 800, viewport shrinks to 500
    Object.defineProperty(window, "innerHeight", { value: 800, writable: true, configurable: true });
    mockVV.height = 500;
    mockVV.offsetTop = 0;

    renderHook(() => useAppViewport());

    expect(getCssVar("--wc-app-height")).toBe("500px");
    expect(getCssVar("--wc-kb-height")).toBe("300px");
    // Keyboard is open, so safe-bottom should be 0
    expect(getCssVar("--wc-safe-bottom")).toBe("0px");
  });

  it("accounts for offsetTop in keyboard height calculation", () => {
    Object.defineProperty(window, "innerHeight", { value: 800, writable: true, configurable: true });
    mockVV.height = 500;
    mockVV.offsetTop = 50;

    renderHook(() => useAppViewport());

    // kbHeight = max(0, 800 - 500 - 50) = 250
    expect(getCssVar("--wc-kb-height")).toBe("250px");
    expect(getCssVar("--wc-app-height")).toBe("500px");
  });

  it("updates vars on visualViewport resize event", () => {
    renderHook(() => useAppViewport());
    expect(getCssVar("--wc-app-height")).toBe("800px");

    // Simulate keyboard opening
    mockVV.height = 450;
    mockVV._fire("resize");

    expect(getCssVar("--wc-app-height")).toBe("450px");
    expect(getCssVar("--wc-kb-height")).toBe("350px");
  });

  it("updates vars on visualViewport scroll event", () => {
    renderHook(() => useAppViewport());

    mockVV.height = 600;
    mockVV.offsetTop = 20;
    mockVV._fire("scroll");

    expect(getCssVar("--wc-app-height")).toBe("600px");
    expect(getCssVar("--wc-kb-height")).toBe("180px");
  });

  it("polls on focusin for input elements", () => {
    renderHook(() => useAppViewport());

    // Initial state: no keyboard
    expect(getCssVar("--wc-kb-height")).toBe("0px");

    // Simulate keyboard opening after input focus
    const input = document.createElement("input");
    document.body.appendChild(input);
    mockVV.height = 500;
    input.dispatchEvent(new FocusEvent("focusin", { bubbles: true }));

    // Immediate update fires
    expect(getCssVar("--wc-app-height")).toBe("500px");

    // Simulate viewport continuing to change during animation
    mockVV.height = 480;
    vi.advanceTimersByTime(100);
    expect(getCssVar("--wc-app-height")).toBe("480px");

    // Advance through remaining poll intervals
    mockVV.height = 460;
    vi.advanceTimersByTime(400);
    expect(getCssVar("--wc-app-height")).toBe("460px");

    document.body.removeChild(input);
  });

  it("does not poll on focusin for non-input elements", () => {
    renderHook(() => useAppViewport());

    const div = document.createElement("div");
    document.body.appendChild(div);

    // Change viewport but focus a non-input element
    mockVV.height = 500;
    div.dispatchEvent(new FocusEvent("focusin", { bubbles: true }));

    // Should NOT have updated (still at initial value)
    expect(getCssVar("--wc-app-height")).toBe("800px");

    document.body.removeChild(div);
  });

  it("polls on focusout for input elements", () => {
    renderHook(() => useAppViewport());

    const textarea = document.createElement("textarea");
    document.body.appendChild(textarea);

    // Simulate keyboard closing after blur
    mockVV.height = 500;
    textarea.dispatchEvent(new FocusEvent("focusin", { bubbles: true }));
    expect(getCssVar("--wc-app-height")).toBe("500px");

    // Keyboard closes
    mockVV.height = 800;
    textarea.dispatchEvent(new FocusEvent("focusout", { bubbles: true }));
    expect(getCssVar("--wc-app-height")).toBe("800px");

    vi.advanceTimersByTime(500);
    document.body.removeChild(textarea);
  });

  it("calls window.scrollTo(0, 0) on every update", () => {
    const scrollToSpy = vi.spyOn(window, "scrollTo").mockImplementation(() => {});
    renderHook(() => useAppViewport());

    // Called once on mount
    expect(scrollToSpy).toHaveBeenCalledWith(0, 0);

    // Called again on resize
    mockVV.height = 500;
    mockVV._fire("resize");
    expect(scrollToSpy).toHaveBeenCalledTimes(2);
  });

  it("restores safe-bottom when keyboard closes", () => {
    renderHook(() => useAppViewport());
    expect(getCssVar("--wc-safe-bottom")).toBe("env(safe-area-inset-bottom)");

    // Open keyboard
    mockVV.height = 500;
    mockVV._fire("resize");
    expect(getCssVar("--wc-safe-bottom")).toBe("0px");

    // Close keyboard
    mockVV.height = 800;
    mockVV._fire("resize");
    expect(getCssVar("--wc-safe-bottom")).toBe("env(safe-area-inset-bottom)");
  });

  it("cleans up CSS vars and listeners on unmount", () => {
    const { unmount } = renderHook(() => useAppViewport());

    expect(getCssVar("--wc-app-height")).toBe("800px");
    expect(getCssVar("--wc-kb-height")).toBe("0px");
    expect(getCssVar("--wc-safe-top")).toBe("env(safe-area-inset-top)");
    expect(getCssVar("--wc-safe-bottom")).toBe("env(safe-area-inset-bottom)");

    unmount();

    expect(getCssVar("--wc-app-height")).toBeUndefined();
    expect(getCssVar("--wc-kb-height")).toBeUndefined();
    expect(getCssVar("--wc-safe-top")).toBeUndefined();
    expect(getCssVar("--wc-safe-bottom")).toBeUndefined();
    expect(mockVV.removeEventListener).toHaveBeenCalledWith("resize", expect.any(Function));
    expect(mockVV.removeEventListener).toHaveBeenCalledWith("scroll", expect.any(Function));
  });

  it("does not throw when visualViewport is unavailable", () => {
    Object.defineProperty(window, "visualViewport", {
      value: null,
      writable: true,
      configurable: true,
    });

    expect(() => {
      const { unmount } = renderHook(() => useAppViewport());
      unmount();
    }).not.toThrow();

    // No CSS vars should be set
    expect(getCssVar("--wc-app-height")).toBeUndefined();
    expect(getCssVar("--wc-kb-height")).toBeUndefined();
  });
});
