import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { useState } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { VirtualList } from "../../../../components/VirtualList/versions/1.3.3/VirtualList";

type Row = { id: string; label: string };
const initialRows: Row[] = Array.from({ length: 30 }, (_, index) => ({
  id: `row-${index}`,
  label: `Row ${index}`,
}));
const earlierRows: Row[] = [
  { id: "earlier-0", label: "Earlier 0" },
  { id: "earlier-1", label: "Earlier 1" },
];

class ResizeObserverStub {
  static instances: ResizeObserverStub[] = [];
  callback: ResizeObserverCallback;
  target?: Element;

  constructor(callback: ResizeObserverCallback) {
    this.callback = callback;
    ResizeObserverStub.instances.push(this);
  }

  observe(target: Element) {
    this.target = target;
  }

  disconnect() {}

  emit() {
    this.callback([], this as unknown as ResizeObserver);
  }
}

function rowTop(row: HTMLElement, viewport: HTMLElement) {
  const match = row.style.transform.match(/translateY\((-?[\d.]+)px\)/);
  expect(match).not.toBeNull();
  return Number(match?.[1]) - viewport.scrollTop;
}

function HistoryHarness() {
  const [rows, setRows] = useState(initialRows);
  return (
    <div
      ref={(container) => {
        if (!container) return;
        const viewport = container.querySelector<HTMLElement>("[data-rcl-virtual-list-viewport]");
        const content = container.querySelector<HTMLElement>("[data-rcl-virtual-list-content]");
        if (!viewport || !content || viewport.dataset.testMetricsReady) return;
        let scrollTop = viewport.scrollTop;
        Object.defineProperties(viewport, {
          clientHeight: { configurable: true, value: 360 },
          scrollHeight: {
            configurable: true,
            get: () => Number.parseFloat(content.style.height) || 0,
          },
          scrollTop: {
            configurable: true,
            get: () => scrollTop,
            set: (next: number) => {
              const maximum = Math.max(0, Number.parseFloat(content.style.height) - 360);
              scrollTop = Math.min(Math.max(0, next), maximum);
            },
          },
        });
        viewport.dataset.testMetricsReady = "true";
      }}
    >
      <button type="button" onClick={() => setRows((current) => [...earlierRows, ...current])}>
        Load earlier rows
      </button>
      <VirtualList
        items={rows}
        getItemKey={(row) => row.id}
        height={360}
        initialScrollTop={450}
        estimateItemHeight={100}
        overscan={1}
        followEnd={false}
        renderItem={(row) => <span>{row.label}</span>}
      />
    </div>
  );
}

describe("VirtualList draft scroll anchoring", () => {
  afterEach(() => {
    cleanup();
    ResizeObserverStub.instances = [];
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
  });

  it("preserves the user's visible row through a prepend and a resize above it", async () => {
    vi.stubGlobal("ResizeObserver", ResizeObserverStub);
    const rowHeights = new Map<string, number>();
    const originalRect = Element.prototype.getBoundingClientRect;
    vi.spyOn(Element.prototype, "getBoundingClientRect").mockImplementation(function (this: Element) {
      const key = (this as HTMLElement).dataset?.rclVirtualListKey;
      if (key !== undefined) {
        const height = rowHeights.get(key) ?? 100;
        return {
          x: 0,
          y: 0,
          top: 0,
          right: 100,
          bottom: height,
          left: 0,
          width: 100,
          height,
          toJSON: () => ({}),
        } as DOMRect;
      }
      return originalRect.call(this);
    });

    render(<HistoryHarness />);
    const viewport = document.querySelector<HTMLElement>("[data-rcl-virtual-list-viewport]");
    if (!viewport) throw new Error("VirtualList viewport did not render");
    const anchor = document.querySelector<HTMLElement>("[data-rcl-virtual-list-key='row-5']");
    expect(anchor).not.toBeNull();

    viewport.scrollTop = 550;
    await act(async () => {
      fireEvent.scroll(viewport);
      await new Promise((resolve) => setTimeout(resolve, 30));
    });
    expect(viewport.scrollTop).toBe(550);
    const originalAnchorTop = rowTop(anchor!, viewport);

    fireEvent.click(screen.getByRole("button", { name: "Load earlier rows" }));
    expect(viewport.scrollTop).toBe(750);
    const anchoredAfterPrepend = document.querySelector<HTMLElement>("[data-rcl-virtual-list-key='row-5']");
    expect(anchoredAfterPrepend).not.toBeNull();
    expect(rowTop(anchoredAfterPrepend!, viewport)).toBe(originalAnchorTop);

    const rowFourObserver = ResizeObserverStub.instances.find(
      (observer) => (observer.target as HTMLElement | undefined)?.dataset.rclVirtualListKey === "row-4",
    );
    expect(rowFourObserver).toBeDefined();
    rowHeights.set("row-4", 140);
    act(() => rowFourObserver?.emit());

    expect(viewport.scrollTop).toBe(790);
    const anchoredAfterResize = document.querySelector<HTMLElement>("[data-rcl-virtual-list-key='row-5']");
    expect(anchoredAfterResize).not.toBeNull();
    expect(rowTop(anchoredAfterResize!, viewport)).toBe(originalAnchorTop);
  });
});
