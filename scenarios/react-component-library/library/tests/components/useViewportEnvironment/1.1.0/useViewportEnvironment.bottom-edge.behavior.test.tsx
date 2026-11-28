import { act, cleanup, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  useViewportEnvironment,
  useViewportEnvironmentStyle,
} from "@vrooli/react-component-library/useViewportEnvironment/1.1.0";

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

describe("useViewportEnvironment 1.1.0 — does the app reach the screen's bottom edge", () => {
  let viewport: FakeVisualViewport;
  let frames: FrameRequestCallback[];
  const realGetComputedStyle = window.getComputedStyle.bind(window);

  const flushFrames = () => {
    while (frames.length > 0) frames.splice(0).forEach((callback) => callback(performance.now()));
  };
  const installed = (standalone: boolean) => {
    Object.defineProperty(navigator, "standalone", { configurable: true, value: standalone });
  };

  beforeEach(() => {
    viewport = new FakeVisualViewport();
    frames = [];
    Object.defineProperty(window, "innerWidth", { configurable: true, value: 393 });
    Object.defineProperty(window, "innerHeight", { configurable: true, value: 793 });
    Object.defineProperty(window, "visualViewport", { configurable: true, value: viewport });
    Object.defineProperty(window, "screen", { configurable: true, value: { width: 393, height: 852 } });
    // jsdom resolves (and keeps) no env(), so the safe-area probe — the hidden,
    // fixed element the environment measures — reads a 59px status bar.
    vi.spyOn(window, "getComputedStyle").mockImplementation((element: Element) => (
      element instanceof HTMLElement && element.style.visibility === "hidden" && element.style.position === "fixed"
        ? ({ paddingTop: "59px" } as CSSStyleDeclaration)
        : realGetComputedStyle(element)
    ));
    vi.spyOn(window, "requestAnimationFrame").mockImplementation((callback) => { frames.push(callback); return frames.length; });
    vi.spyOn(window, "cancelAnimationFrame").mockImplementation(() => undefined);
  });

  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
    installed(false);
    document.body.replaceChildren();
  });

  it("an installed app whose viewport stops short by exactly the status bar does not reach the bottom, so overlays reserve no home-indicator inset", () => {
    installed(true);
    const { result } = renderHook(() => ({ env: useViewportEnvironment(), style: useViewportEnvironmentStyle() }));
    expect(result.current.env.reachesScreenBottom).toBe(false);
    expect(result.current.style["--rcl-safe-bottom"]).toBe("0px");
  });

  it("a browser tab is trusted as reported and keeps the inset", () => {
    installed(false);
    const { result } = renderHook(() => ({ env: useViewportEnvironment(), style: useViewportEnvironmentStyle() }));
    expect(result.current.env.reachesScreenBottom).toBe(true);
    expect(result.current.style["--rcl-safe-bottom"]).toBe("env(safe-area-inset-bottom, 0px)");
  });

  it("an installed app that fills the screen keeps the inset; a gap it cannot explain is trusted too", () => {
    installed(true);
    viewport.height = 852;
    Object.defineProperty(window, "innerHeight", { configurable: true, value: 852 });
    const { result } = renderHook(() => useViewportEnvironment());
    expect(result.current.reachesScreenBottom).toBe(true);
    act(() => {
      viewport.height = 760;
      viewport.dispatchEvent(new Event("resize"));
      flushFrames();
    });
    expect(result.current.reachesScreenBottom).toBe(true);
  });

  it("the keyboard covers the bottom edge, so the inset does not apply while it is up", () => {
    installed(false);
    Object.defineProperty(window, "innerHeight", { configurable: true, value: 844 });
    viewport.height = 844;
    const input = document.createElement("input");
    document.body.append(input);
    input.focus();
    const { result } = renderHook(() => ({ env: useViewportEnvironment(), style: useViewportEnvironmentStyle() }));
    act(() => {
      viewport.height = 520;
      viewport.dispatchEvent(new Event("resize"));
      flushFrames();
    });
    expect(result.current.env.keyboardVisible).toBe(true);
    expect(result.current.env.reachesScreenBottom).toBe(false);
    expect(result.current.style["--rcl-safe-bottom"]).toBe("0px");
  });
});
