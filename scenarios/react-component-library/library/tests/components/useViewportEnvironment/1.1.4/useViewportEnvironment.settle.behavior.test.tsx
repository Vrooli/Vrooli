import { act, cleanup, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useViewportEnvironment } from "@vrooli/react-component-library/useViewportEnvironment/1.1.4";

class FakeVisualViewport extends EventTarget {
  width = 393;
  height = 793;
  offsetLeft = 0;
  offsetTop = 0;
  pageLeft = 0;
  pageTop = 0;
  scale = 1;
  onresize = null;
  onscroll = null;
}

describe("useViewportEnvironment 1.1.4 — an installed app's late-settling frame", () => {
  let safeTop: string;
  const realGetComputedStyle = window.getComputedStyle.bind(window);
  const realRect = HTMLElement.prototype.getBoundingClientRect;

  beforeEach(() => {
    vi.useFakeTimers();
    safeTop = "0px";
    Object.defineProperty(navigator, "standalone", { configurable: true, value: true });
    Object.defineProperty(window, "innerWidth", { configurable: true, value: 393 });
    Object.defineProperty(window, "innerHeight", { configurable: true, value: 793 });
    Object.defineProperty(window, "visualViewport", { configurable: true, value: new FakeVisualViewport() });
    Object.defineProperty(window, "screen", { configurable: true, value: { width: 393, height: 852 } });
    const meta = document.createElement("meta");
    meta.name = "apple-mobile-web-app-status-bar-style";
    meta.content = "black-translucent";
    document.head.append(meta);
    // iOS resolves env(safe-area-inset-top) after first paint and fires no
    // resize when it does; 100lvh already reports the full screen.
    vi.spyOn(window, "getComputedStyle").mockImplementation((element: Element) => (
      element instanceof HTMLElement && element.style.visibility === "hidden" && element.style.position === "fixed"
        ? ({ paddingTop: safeTop } as CSSStyleDeclaration)
        : realGetComputedStyle(element)
    ));
    vi.spyOn(HTMLElement.prototype, "getBoundingClientRect").mockImplementation(function (this: HTMLElement) {
      return this.style.height === "100lvh" ? ({ height: 852 } as DOMRect) : realRect.call(this);
    });
    vi.spyOn(window, "requestAnimationFrame").mockImplementation((callback) => setTimeout(() => callback(performance.now()), 0) as unknown as number);
    vi.spyOn(window, "cancelAnimationFrame").mockImplementation(() => undefined);
  });

  afterEach(() => {
    cleanup();
    vi.useRealTimers();
    vi.restoreAllMocks();
    Object.defineProperty(navigator, "standalone", { configurable: true, value: false });
    document.head.replaceChildren();
    document.body.replaceChildren();
  });

  it("adopts the full-screen frame once the status-bar inset settles, without a resize event", () => {
    const { result } = renderHook(() => useViewportEnvironment());
    expect(result.current.frameHeight).toBe(793);
    safeTop = "59px";
    act(() => { vi.advanceTimersByTime(3000); });
    expect(result.current.frameHeight).toBe(852);
  });

  it("re-measures when the installed app returns to the foreground", () => {
    const { result } = renderHook(() => useViewportEnvironment());
    act(() => { vi.advanceTimersByTime(3000); });
    expect(result.current.frameHeight).toBe(793);
    safeTop = "59px";
    act(() => {
      document.dispatchEvent(new Event("visibilitychange"));
      vi.advanceTimersByTime(50);
    });
    expect(result.current.frameHeight).toBe(852);
  });
});
