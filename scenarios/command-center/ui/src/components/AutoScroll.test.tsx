import { act } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { BoardContext, type BoardControllerValue } from "../lib/boardContext";
import { renderWithProviders, screen } from "../test-utils/renderWithProviders";
import { AUTOSCROLL_TIMING } from "../lib/fit";
import { AutoScroll } from "./AutoScroll";

vi.mock("../lib/media", () => ({ useLandscapeRoom: () => true, useReducedMotion: () => false }));

let ROW = 50;
let VIEWPORT = 200;
let contentHeight = 500;

/** jsdom has no layout: rows report their authored top, the viewport a fixed height. */
function stubLayout() {
  vi.spyOn(Element.prototype, "getBoundingClientRect").mockImplementation(function (this: Element) {
    const top = Number((this as HTMLElement).dataset.top ?? 0);
    const height = this.classList.contains("cc-autoscroll__content") ? contentHeight : this.matches("[data-top]") ? ROW : 0;
    return { top, bottom: top + height, left: 0, right: 100, width: 100, height, x: 0, y: top, toJSON: () => ({}) } as DOMRect;
  });
  vi.spyOn(HTMLElement.prototype, "clientHeight", "get").mockImplementation(function (this: HTMLElement) {
    return this.classList.contains("cc-autoscroll__viewport") ? VIEWPORT : 0;
  });
}

function renderList(count: number, holdBeat = vi.fn()) {
  const board = { holdBeat } as unknown as BoardControllerValue;
  const view = renderWithProviders(
    <BoardContext.Provider value={board}>
      <AutoScroll rowSelector="[data-top]" label="Rows">
        <ul>{Array.from({ length: count }, (_, index) => <li key={index} data-top={index * ROW}>row {index + 1}</li>)}</ul>
      </AutoScroll>
    </BoardContext.Provider>,
  );
  const content = () => view.container.querySelector<HTMLElement>(".cc-autoscroll__content");
  return { ...view, holdBeat, content };
}

const advance = (ms: number) => act(() => { vi.advanceTimersByTime(ms); });

describe("AutoScroll", () => { // [REQ:CC-P1-017]
  beforeEach(() => {
    vi.useFakeTimers();
    stubLayout();
  });
  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
    contentHeight = 500;
    ROW = 50;
    VIEWPORT = 200;
  });

  it("holds the first rows, steps one row at a time, fades back to the top and releases the beat after one pass", () => {
    const { holdBeat, content } = renderList(10);
    const held = () => holdBeat.mock.lastCall?.[1];
    expect(held()).toBe(true);
    expect(screen.getByTestId("autoscroll-position")).toHaveTextContent("1–4 of 10");
    expect(content()?.style.transform).toBe("translate3d(0, 0px, 0)");

    advance(AUTOSCROLL_TIMING.holdStartMs - 1);
    expect(content()?.style.transform).toBe("translate3d(0, 0px, 0)");
    advance(1);
    expect(content()?.style.transform).toBe("translate3d(0, -50px, 0)");
    for (let step = 0; step < 5; step += 1) advance(AUTOSCROLL_TIMING.stepMs);
    expect(content()?.style.transform).toBe("translate3d(0, -300px, 0)");
    expect(screen.getByTestId("autoscroll-position")).toHaveTextContent("7–10 of 10");
    expect(held()).toBe(true);

    advance(AUTOSCROLL_TIMING.holdEndMs);
    expect(content()?.closest("[data-phase]")).toHaveAttribute("data-phase", "fading");
    advance(AUTOSCROLL_TIMING.fadeMs);
    expect(content()?.style.transform).toBe("translate3d(0, 0px, 0)");
    advance(AUTOSCROLL_TIMING.fadeMs);
    expect(held()).toBe(false);

    // It has been read once; the list rests at the top rather than looping.
    const callsAfterPass = holdBeat.mock.calls.length;
    advance(60_000);
    expect(content()?.style.transform).toBe("translate3d(0, 0px, 0)");
    expect(holdBeat.mock.calls.slice(callsAfterPass).some((call) => call[1] === true)).toBe(false);
  });

  it("reserves the counter line when no row is wholly visible, so the viewport cannot oscillate", () => { // [REQ:CC-P1-017]
    // A row taller than the viewport means no row is wholly visible and the
    // counter has no range to name. The line must still hold its place, or its
    // appearance would resize the viewport it is measured from.
    ROW = 300;
    VIEWPORT = 100;
    contentHeight = 600;
    renderList(3);
    expect(screen.getByTestId("autoscroll-position")).toBeInTheDocument();
  });

  it("never moves or holds a list that fits", () => {
    contentHeight = 150;
    const { holdBeat, content } = renderList(3);
    advance(20_000);
    expect(content()?.style.transform).toBe("");
    expect(holdBeat).not.toHaveBeenCalledWith(expect.any(String), true);
    expect(screen.queryByTestId("autoscroll-position")).toBeNull();
  });
});
