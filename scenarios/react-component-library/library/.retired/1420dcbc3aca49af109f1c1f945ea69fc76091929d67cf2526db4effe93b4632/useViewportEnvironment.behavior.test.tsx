import { act, cleanup, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  ViewportEnvironmentProvider,
  useViewportEnvironment,
  type ViewportEnvironmentSnapshot,
} from "./useViewportEnvironment.ingest";

class FakeVisualViewport extends EventTarget {
  width = 390;
  height = 760;
  offsetLeft = 0;
  offsetTop = 0;
  pageLeft = 0;
  pageTop = 0;
  scale = 1;
  onresize = null;
  onscroll = null;
}

describe("useViewportEnvironment", () => {
  let viewport: FakeVisualViewport;
  let frames: FrameRequestCallback[];

  const flushFrames = () => {
    while (frames.length > 0) {
      const pending = frames.splice(0);
      pending.forEach((callback) => callback(performance.now()));
    }
  };

  beforeEach(() => {
    viewport = new FakeVisualViewport();
    frames = [];
    Object.defineProperty(window, "innerWidth", {
      configurable: true,
      value: 390,
    });
    Object.defineProperty(window, "innerHeight", {
      configurable: true,
      value: 844,
    });
    Object.defineProperty(window, "visualViewport", {
      configurable: true,
      value: viewport,
    });
    vi.spyOn(window, "requestAnimationFrame").mockImplementation((callback) => {
      frames.push(callback);
      return frames.length;
    });
    vi.spyOn(window, "cancelAnimationFrame").mockImplementation(
      () => undefined,
    );
  });

  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
    document.body.replaceChildren();
  });

  it("does not classify an ordinary viewport resize as a keyboard", () => {
    const { result } = renderHook(() => useViewportEnvironment());
    act(() => {
      viewport.height = 620;
      viewport.dispatchEvent(new Event("resize"));
      flushFrames();
    });
    expect(result.current).toMatchObject({
      visibleHeight: 620,
      keyboardInset: 0,
      keyboardVisible: false,
    });
  });

  it("enters keyboard state only after editable focus and settled occlusion", () => {
    const input = document.createElement("input");
    document.body.append(input);
    input.focus();
    const { result } = renderHook(() => useViewportEnvironment());
    act(() => {
      viewport.height = 520;
      viewport.dispatchEvent(new Event("resize"));
      flushFrames();
    });
    expect(result.current).toMatchObject({
      keyboardVisible: true,
      keyboardInset: 324,
    });
    act(() => {
      input.blur();
      document.dispatchEvent(new FocusEvent("focusout", { bubbles: true }));
      flushFrames();
    });
    expect(result.current).toMatchObject({
      keyboardVisible: false,
      keyboardInset: 0,
    });
  });

  it("shares one listener source across consumers", () => {
    const add = vi.spyOn(viewport, "addEventListener");
    const first = renderHook(() => useViewportEnvironment());
    const second = renderHook(() => useViewportEnvironment());
    expect(add.mock.calls.filter(([type]) => type === "resize")).toHaveLength(
      1,
    );
    first.unmount();
    second.unmount();
  });

  it("uses an authoritative host override", () => {
    const override: ViewportEnvironmentSnapshot = {
      layoutWidth: 800,
      layoutHeight: 600,
      visibleWidth: 700,
      visibleHeight: 400,
      offsetLeft: 10,
      offsetTop: 20,
      scale: 1,
      keyboardInset: 180,
      keyboardVisible: true,
    };
    const wrapper = ({ children }: { children: React.ReactNode }) => (
      <ViewportEnvironmentProvider value={override}>
        {children}
      </ViewportEnvironmentProvider>
    );
    const { result } = renderHook(() => useViewportEnvironment(), { wrapper });
    expect(result.current).toEqual(override);
  });
});
