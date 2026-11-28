import { act, renderHook } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import type { DesignFrame } from "../design-types";
import { measurePaintedArea, useElementWidth, useFrameAudit } from "./useFrameAudit";

function rect(left: number, top: number, width: number, height: number): DOMRect {
  return {
    left,
    top,
    width,
    height,
    right: left + width,
    bottom: top + height,
    x: left,
    y: top,
    toJSON: () => ({}),
  };
}

function giveRect(element: HTMLElement, value: DOMRect): void {
  element.getBoundingClientRect = () => value;
}

describe("measurePaintedArea", () => {
  it("counts a mapped component with children once at its full footprint", () => {
    const page = document.createElement("div");
    const button = document.createElement("button");
    button.dataset.nodeId = "nav-home";
    button.dataset.nodeType = "button";
    const icon = document.createElement("span");
    icon.dataset.nodeId = "nav-home-icon";
    icon.dataset.nodeType = "image";
    const label = document.createElement("span");
    label.dataset.nodeId = "nav-home-label";
    label.dataset.nodeType = "text";
    button.append(icon, label);
    const artwork = document.createElement("span");
    artwork.dataset.nodeId = "artwork";
    artwork.dataset.nodeType = "image";
    page.append(button, artwork);

    giveRect(button, rect(0, 700, 390, 60));
    giveRect(icon, rect(12, 710, 24, 24));
    giveRect(label, rect(40, 710, 80, 20));
    giveRect(artwork, rect(0, 0, 200, 100));

    expect(measurePaintedArea(page, rect(0, 0, 390, 844), 1)).toEqual({
      rcl: 23_400,
      total: 43_400,
    });
  });

  it("clips custom leaves and counts nested RCL primitives only once at invalid scales", () => {
    const page = document.createElement("div");
    const customParent = document.createElement("div");
    customParent.dataset.nodeId = "section";
    customParent.dataset.nodeType = "stack";
    const mappedCard = document.createElement("div");
    mappedCard.dataset.nodeId = "card";
    mappedCard.dataset.nodeType = "card";
    const title = document.createElement("span");
    title.dataset.nodeId = "title";
    title.dataset.nodeType = "text";
    const clippedLeaf = document.createElement("span");
    clippedLeaf.dataset.nodeId = "decoration";
    clippedLeaf.dataset.nodeType = "spacer";
    const unknownLeaf = document.createElement("span");
    unknownLeaf.dataset.nodeId = "unknown";
    customParent.append(mappedCard, title, clippedLeaf);
    mappedCard.append(title);
    page.append(customParent, unknownLeaf);

    giveRect(customParent, rect(0, 0, 300, 300));
    giveRect(mappedCard, rect(20, 20, 100, 80));
    giveRect(title, rect(30, 30, 50, 20));
    giveRect(clippedLeaf, rect(290, 290, 40, 40));
    giveRect(unknownLeaf, rect(10, 10, 10, 10));

    expect(measurePaintedArea(page, rect(0, 0, 300, 300), 0)).toEqual({
      rcl: 8_000,
      total: 8_200,
    });
  });
});

const frame: DesignFrame = {
  id: "phone",
  name: "Phone",
  device: "phone",
  spec: { id: "page", type: "page", props: {}, children: [] },
};

function layoutElement(className?: string): HTMLElement {
  const element = document.createElement("div");
  if (className) element.className = className;
  return element;
}

function setDimension(
  element: HTMLElement,
  key: "clientWidth" | "scrollWidth",
  value: number,
): void {
  Object.defineProperty(element, key, { configurable: true, value });
}

function setRect(element: HTMLElement, value: DOMRect): void {
  element.getBoundingClientRect = () => value;
}

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("useFrameAudit", () => {
  it("reports an unmeasured frame when its rendered page has no layout width", () => {
    const root = layoutElement();
    const host = layoutElement();
    host.dataset.studioFrame = frame.id;
    const page = layoutElement("tpl-page");
    setDimension(page, "clientWidth", 0);
    host.append(page);
    root.append(host);
    const container = { current: root };
    const frames = [frame];
    const scales = { [frame.id]: 1 };

    const { result } = renderHook(() => useFrameAudit(container, frames, scales));

    expect(result.current.phone).toEqual({
      overflow: 0,
      buttons: null,
      painted: null,
      measured: false,
    });
  });

  it("measures overflow and visible touch targets and remeasures after an observed resize", () => {
    let resize: ResizeObserverCallback | undefined;
    let scheduled: FrameRequestCallback | undefined;
    const observed: Element[] = [];
    let disconnected = false;
    class Observer {
      constructor(callback: ResizeObserverCallback) {
        resize = callback;
      }
      observe = vi.fn((target: Element) => observed.push(target));
      unobserve = vi.fn();
      disconnect = vi.fn(() => {
        disconnected = true;
      });
    }
    vi.stubGlobal("ResizeObserver", Observer);
    vi.spyOn(window, "requestAnimationFrame").mockImplementation((callback) => {
      scheduled = callback;
      return 17;
    });
    vi.spyOn(window, "cancelAnimationFrame").mockImplementation(() => undefined);

    const root = layoutElement();
    const host = layoutElement();
    host.dataset.studioFrame = frame.id;
    const page = layoutElement("tpl-page");
    setDimension(page, "clientWidth", 100);
    setDimension(page, "scrollWidth", 102);
    setRect(page, rect(0, 0, 100, 100));
    const visible = document.createElement("button");
    setRect(visible, rect(0, 0, 60, 36));
    const sufficientlyTall = document.createElement("button");
    setRect(sufficientlyTall, rect(0, 40, 60, 100));
    const notLaidOut = document.createElement("button");
    setRect(notLaidOut, rect(0, 90, 60, 0));
    page.append(visible, sufficientlyTall, notLaidOut);
    host.append(page);
    root.append(host);
    const container = { current: root };
    const frames = [frame];
    const scales = { [frame.id]: 2 };

    const { result, unmount } = renderHook(() => useFrameAudit(container, frames, scales));

    expect(result.current.phone?.overflow).toBe(0);
    expect(result.current.phone?.buttons).toEqual({ total: 2, under: 1 });
    expect(observed).toEqual([page, host]);

    setDimension(page, "scrollWidth", 130);
    act(() => {
      resize?.([], {} as ResizeObserver);
      scheduled?.(0);
    });
    expect(result.current.phone?.overflow).toBe(30);

    setDimension(page, "scrollWidth", 160);
    act(() => {
      resize?.([], {} as ResizeObserver);
      scheduled?.(0);
    });
    expect(result.current.phone?.overflow).toBe(60);

    unmount();
    expect(disconnected).toBe(true);
  });
});

describe("useElementWidth", () => {
  it("updates immediately from layout and follows element resizes", () => {
    let resize: ResizeObserverCallback | undefined;
    class Observer {
      constructor(callback: ResizeObserverCallback) {
        resize = callback;
      }
      observe = vi.fn();
      disconnect = vi.fn();
    }
    vi.stubGlobal("ResizeObserver", Observer);
    const element = layoutElement();
    setDimension(element, "clientWidth", 320);
    const ref = { current: element };
    const { result, rerender, unmount } = renderHook(() => useElementWidth(ref));
    expect(result.current).toBe(320);

    setDimension(element, "clientWidth", 480);
    act(() => resize?.([], {} as ResizeObserver));
    expect(result.current).toBe(480);
    rerender();
    unmount();
  });
});
