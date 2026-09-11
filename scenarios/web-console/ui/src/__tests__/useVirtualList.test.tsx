import { createRef, useRef } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { act, fireEvent, render, renderHook, screen, waitFor } from "@testing-library/react";
import { useVirtualList } from "../hooks/useVirtualList";

describe("useVirtualList", () => {
  it("keeps a 2500-item list bounded before its viewport is measured", () => {
    const scrollElementRef = createRef<HTMLElement>();
    const { result } = renderHook(() => useVirtualList({
      count: 2500,
      estimateSize: () => 120,
      overscan: 8,
      scrollElementRef,
    }));

    expect(result.current.viewportHeight).toBe(0);
    expect(result.current.virtualItems.length).toBeLessThanOrEqual(16);
  });

  it("keeps the small-list path unvirtualized", () => {
    const scrollElementRef = createRef<HTMLElement>();
    const { result } = renderHook(() => useVirtualList({
      count: 12,
      estimateSize: () => 120,
      scrollElementRef,
      enabled: false,
    }));

    expect(result.current.virtualItems).toHaveLength(12);
  });

  describe("resize compensation above the viewport", () => {
    const estimate = () => 120;
    const getKey = (index: number) => `k${String(index)}`;
    const heights = new Map<string, number>();
    const observers: { callback: ResizeObserverCallback; targets: Set<Element> }[] = [];
    let frames: FrameRequestCallback[] = [];

    function Harness() {
      const ref = useRef<HTMLDivElement>(null);
      const list = useVirtualList({ count: 50, estimateSize: estimate, getItemKey: getKey, overscan: 8, scrollElementRef: ref });
      return (
        <div ref={ref} data-testid="scroller">
          <div style={{ height: `${String(list.totalSize)}px` }}>
            {list.virtualItems.map((item) => (
              <div
                key={item.index}
                data-testid={`row-${String(item.index)}`}
                ref={(node) => { list.registerItem(item.index, node); }}
                style={{ position: "absolute", top: `${String(item.start)}px` }}
              />
            ))}
          </div>
        </div>
      );
    }

    afterEach(() => {
      heights.clear();
      observers.length = 0;
      frames = [];
      vi.unstubAllGlobals();
      vi.restoreAllMocks();
    });

    it("moves scrollTop in the same commit as the rows it compensates (no one-frame jump)", async () => {
      vi.stubGlobal("requestAnimationFrame", (cb: FrameRequestCallback) => { frames.push(cb); return frames.length; });
      vi.stubGlobal("cancelAnimationFrame", () => undefined);
      vi.stubGlobal("ResizeObserver", class {
        private entry: { callback: ResizeObserverCallback; targets: Set<Element> };
        constructor(callback: ResizeObserverCallback) {
          this.entry = { callback, targets: new Set() };
          observers.push(this.entry);
        }
        observe(target: Element) { this.entry.targets.add(target); }
        unobserve(target: Element) { this.entry.targets.delete(target); }
        disconnect() { this.entry.targets.clear(); }
      });
      vi.spyOn(Element.prototype, "getBoundingClientRect").mockImplementation(function (this: Element) {
        const height = heights.get(this.getAttribute("data-testid") ?? "") ?? 0;
        return { top: 0, bottom: height, left: 0, right: 0, width: 0, height, x: 0, y: 0, toJSON: () => ({}) } as DOMRect;
      });
      // Silence React's act() warning for the deliberate out-of-act frame below.
      vi.spyOn(console, "error").mockImplementation(() => undefined);

      render(<Harness />);
      const scroller = screen.getByTestId("scroller");
      let scrollTop = 0;
      Object.defineProperty(scroller, "clientHeight", { configurable: true, get: () => 400 });
      Object.defineProperty(scroller, "scrollTop", {
        configurable: true,
        get: () => scrollTop,
        set: (value: number) => { scrollTop = value; },
      });
      scrollTop = 1000;
      act(() => { fireEvent.scroll(scroller); });

      // Row 10 is on screen at 1200 - 1000 = 200 px from the viewport top.
      const screenOffset = () => parseFloat(screen.getByTestId("row-10").style.top) - scrollTop;
      expect(screenOffset()).toBe(200);

      // Row 2 (start 240, above the viewport) measures 200 px taller.
      heights.set("row-2", 320);
      const row2 = screen.getByTestId("row-2");
      for (const observer of observers) {
        if (observer.targets.has(row2)) observer.callback([], {} as ResizeObserver);
      }

      // The measurement frame runs, but React has not committed yet. The row on
      // screen must not have moved: scrollTop and row offsets change together.
      const pending = frames;
      frames = [];
      for (const frame of pending) frame(0);
      expect(screenOffset()).toBe(200);

      // After the commit the compensation has landed exactly once.
      await waitFor(() => { expect(scrollTop).toBe(1200); });
      expect(screenOffset()).toBe(200);
    });

    it("keeps the top of the row straddling the viewport edge where the reader left it", () => {
      vi.stubGlobal("requestAnimationFrame", (cb: FrameRequestCallback) => { frames.push(cb); return frames.length; });
      vi.stubGlobal("cancelAnimationFrame", () => undefined);
      vi.stubGlobal("ResizeObserver", class {
        private entry: { callback: ResizeObserverCallback; targets: Set<Element> };
        constructor(callback: ResizeObserverCallback) {
          this.entry = { callback, targets: new Set() };
          observers.push(this.entry);
        }
        observe(target: Element) { this.entry.targets.add(target); }
        unobserve(target: Element) { this.entry.targets.delete(target); }
        disconnect() { this.entry.targets.clear(); }
      });
      vi.spyOn(Element.prototype, "getBoundingClientRect").mockImplementation(function (this: Element) {
        const height = heights.get(this.getAttribute("data-testid") ?? "") ?? 0;
        return { top: 0, bottom: height, left: 0, right: 0, width: 0, height, x: 0, y: 0, toJSON: () => ({}) } as DOMRect;
      });

      render(<Harness />);
      const scroller = screen.getByTestId("scroller");
      let scrollTop = 0;
      Object.defineProperty(scroller, "clientHeight", { configurable: true, get: () => 400 });
      Object.defineProperty(scroller, "scrollTop", {
        configurable: true,
        get: () => scrollTop,
        set: (value: number) => { scrollTop = value; },
      });
      // Row 8 spans 960..1080; the viewport starts 40 px into it.
      scrollTop = 1000;
      act(() => { fireEvent.scroll(scroller); });

      heights.set("row-8", 400);
      const row8 = screen.getByTestId("row-8");
      act(() => {
        for (const observer of observers) {
          if (observer.targets.has(row8)) observer.callback([], {} as ResizeObserver);
        }
        const pending = frames;
        frames = [];
        for (const frame of pending) frame(0);
      });

      // Its top stays 40 px above the viewport; the rows below move down.
      expect(scrollTop).toBe(1000);
      expect(parseFloat(row8.style.top) - scrollTop).toBe(-40);
      expect(screen.getByTestId("row-9").style.top).toBe("1360px");
    });

    it("moves the rows below a visible row that measured taller, without any scroll", () => {
      vi.stubGlobal("requestAnimationFrame", (cb: FrameRequestCallback) => { frames.push(cb); return frames.length; });
      vi.stubGlobal("cancelAnimationFrame", () => undefined);
      vi.stubGlobal("ResizeObserver", class {
        private entry: { callback: ResizeObserverCallback; targets: Set<Element> };
        constructor(callback: ResizeObserverCallback) {
          this.entry = { callback, targets: new Set() };
          observers.push(this.entry);
        }
        observe(target: Element) { this.entry.targets.add(target); }
        unobserve(target: Element) { this.entry.targets.delete(target); }
        disconnect() { this.entry.targets.clear(); }
      });
      vi.spyOn(Element.prototype, "getBoundingClientRect").mockImplementation(function (this: Element) {
        const height = heights.get(this.getAttribute("data-testid") ?? "") ?? 0;
        return { top: 0, bottom: height, left: 0, right: 0, width: 0, height, x: 0, y: 0, toJSON: () => ({}) } as DOMRect;
      });

      render(<Harness />);
      const scroller = screen.getByTestId("scroller");
      Object.defineProperty(scroller, "clientHeight", { configurable: true, get: () => 400 });
      act(() => { fireEvent.scroll(scroller); });
      expect(screen.getByTestId("row-3").style.top).toBe("360px");

      // Row 1 (on screen) measures 300 px instead of 120; nothing scrolls.
      heights.set("row-1", 300);
      const row1 = screen.getByTestId("row-1");
      act(() => {
        for (const observer of observers) {
          if (observer.targets.has(row1)) observer.callback([], {} as ResizeObserver);
        }
        const pending = frames;
        frames = [];
        for (const frame of pending) frame(0);
      });

      // The row below must be placed after the taller row, not overlap it.
      expect(screen.getByTestId("row-3").style.top).toBe("540px");
    });
  });
});
