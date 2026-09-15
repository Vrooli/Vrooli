import { act, fireEvent, screen } from "@testing-library/react";
import { vi } from "vitest";
import type { ConversationEvent } from "../../api/conversation";

/**
 * Browser-like scroll geometry for the Messages scroll container in jsdom.
 *
 * - scrollHeight follows the virtualizer's inner spacer height (its totalSize);
 * - scrollTop clamps like a browser, and every write is recorded in
 *   `geo.writes` so a test can assert "the viewport did not move";
 * - a write delivers its `scroll` event on the next frame, as browsers do;
 * - requestAnimationFrame is a manual queue (`flushFrames`), so a test can
 *   hold a programmatic scroll open or flush it deterministically;
 * - ResizeObserver callbacks are captured so `growRow` can report a row
 *   measuring taller;
 * - message row wrappers (`[data-event-id]`, the element the virtualizer
 *   measures) report their virtual position as their rect, with the height
 *   a test set through `growRow` or, by default, their slot's height.
 */
export const CLIENT_HEIGHT = 800;

export const geo = {
  scrollTop: 0,
  writes: [] as number[],
  frames: [] as FrameRequestCallback[],
  /** Content height per event id, set by `growRow`. */
  rowHeights: new Map<string, number>(),
  observers: [] as { callback: ResizeObserverCallback; targets: Set<Element> }[],
};

export function isContainer(el: Element): boolean {
  return el.getAttribute("data-testid") === "messages-scroll";
}

function spacerHeight(el: Element): number {
  const spacer = el.firstElementChild as HTMLElement | null;
  return spacer ? parseFloat(spacer.style.height) || 0 : 0;
}

export function maxScrollTop(el: Element = container()): number {
  return Math.max(0, spacerHeight(el) - CLIENT_HEIGHT);
}

export function container(): HTMLElement {
  return screen.getByTestId("messages-scroll");
}

export function remaining(): number {
  return maxScrollTop() - geo.scrollTop;
}

export function flushFrames(limit = 50): void {
  for (let i = 0; i < limit && geo.frames.length > 0; i += 1) {
    const pending = geo.frames;
    geo.frames = [];
    for (const frame of pending) frame(performance.now());
  }
}

/** A user gesture: an input event first (as every real scroll gesture has), then the scroll. */
export function userScrollTo(top: number): void {
  const el = container();
  fireEvent.wheel(el, { deltaY: top < geo.scrollTop ? -100 : 100 });
  geo.scrollTop = Math.max(0, Math.min(top, maxScrollTop(el)));
  fireEvent.scroll(el);
}

/** Reports that the message row for `eventId` now measures `height` px. */
export function growRow(eventId: string, height: number): void {
  geo.rowHeights.set(eventId, height);
  const node = document.querySelector(`[data-event-id="${eventId}"]`);
  if (!node) throw new Error(`row ${eventId} is not rendered`);
  act(() => {
    for (const observer of geo.observers) {
      if (observer.targets.has(node)) observer.callback([], {} as ResizeObserver);
    }
    flushFrames();
  });
}

export function liveEvent(sequence: number): ConversationEvent {
  const text = `live reply ${String(sequence)}`;
  return {
    id: `live-${String(sequence)}`,
    sessionId: "sess-1",
    sequence,
    source: "claude_hook",
    role: "assistant",
    text,
    speechParagraphs: [text],
    summarized: false,
    createdAt: new Date(1_700_000_000_000 + sequence * 1000).toISOString(),
    deliveryState: "received",
    ttsState: "idle",
    consumptionState: "seen",
  };
}

/**
 * The height the virtualizer gave this row (up to the next rendered row). The
 * last rendered row reports 0, which the virtualizer reads as "keep the
 * estimate" — its slot is not observable from the DOM.
 */
function slotHeight(row: HTMLElement): number {
  const next = row.nextElementSibling as HTMLElement | null;
  return next?.dataset.eventId ? parseFloat(next.style.top) - parseFloat(row.style.top) : 0;
}

function rect(top: number, height: number): DOMRect {
  return { top, bottom: top + height, left: 0, right: 0, width: 0, height, x: 0, y: top, toJSON: () => ({}) } as DOMRect;
}

export function installScrollGeometry(): void {
  geo.scrollTop = 0;
  geo.writes = [];
  geo.frames = [];
  geo.rowHeights.clear();
  geo.observers.length = 0;

  vi.stubGlobal("requestAnimationFrame", (cb: FrameRequestCallback) => { geo.frames.push(cb); return geo.frames.length; });
  vi.stubGlobal("cancelAnimationFrame", () => undefined);
  vi.stubGlobal("ResizeObserver", class {
    private entry: { callback: ResizeObserverCallback; targets: Set<Element> };
    constructor(callback: ResizeObserverCallback) {
      this.entry = { callback, targets: new Set() };
      geo.observers.push(this.entry);
    }
    observe(target: Element) { this.entry.targets.add(target); }
    unobserve(target: Element) { this.entry.targets.delete(target); }
    disconnect() { this.entry.targets.clear(); }
  });

  Object.defineProperty(HTMLElement.prototype, "scrollHeight", {
    configurable: true,
    get(this: HTMLElement) { return isContainer(this) ? spacerHeight(this) : 0; },
  });
  Object.defineProperty(HTMLElement.prototype, "clientHeight", {
    configurable: true,
    get(this: HTMLElement) { return isContainer(this) ? CLIENT_HEIGHT : 0; },
  });
  Object.defineProperty(HTMLElement.prototype, "scrollTop", {
    configurable: true,
    get(this: HTMLElement) { return isContainer(this) ? geo.scrollTop : 0; },
    set(this: HTMLElement, value: number) {
      if (!isContainer(this)) return;
      const next = Math.max(0, Math.min(value, maxScrollTop(this)));
      geo.writes.push(next);
      if (next === geo.scrollTop) return;
      geo.scrollTop = next;
      // Browsers deliver the resulting scroll event on the next frame.
      geo.frames.push(() => { this.dispatchEvent(new Event("scroll")); });
    },
  });
  HTMLElement.prototype.scrollTo = function scrollTo(this: HTMLElement, arg?: ScrollToOptions | number) {
    const top = typeof arg === "number" ? arg : arg?.top ?? this.scrollTop;
    this.scrollTop = top;
  } as HTMLElement["scrollTo"];
  Element.prototype.getBoundingClientRect = function getBoundingClientRect(this: Element): DOMRect {
    if (isContainer(this)) return rect(0, CLIENT_HEIGHT);
    const html = this as HTMLElement;
    const eventId = html.dataset.eventId;
    if (eventId) {
      return rect(parseFloat(html.style.top) - geo.scrollTop, geo.rowHeights.get(eventId) ?? slotHeight(html));
    }
    return rect(0, 0);
  };
}

export function uninstallScrollGeometry(): void {
  const proto = HTMLElement.prototype as unknown as Record<string, unknown>;
  delete proto.scrollHeight;
  delete proto.clientHeight;
  delete proto.scrollTop;
  vi.unstubAllGlobals();
}
