import { act } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { BoardContext, type BoardControllerValue } from "../lib/boardContext";
import { STRIP_PAGE_MS } from "../lib/fit";
import { renderWithProviders, screen } from "../test-utils/renderWithProviders";
import { SupportingStrip } from "./SupportingStrip";

vi.mock("../lib/media", () => ({ useLandscapeRoom: () => true, useReducedMotion: () => false }));

const TILE = 100;
const GAP = 10;
const COLUMNS = 6;

/** jsdom has no layout: tiles report their authored top and the strip a 200px budget. */
function stubLayout() {
  const original = window.getComputedStyle.bind(window);
  vi.spyOn(window, "getComputedStyle").mockImplementation((element: Element) =>
    element.classList.contains("cc-readings")
      ? ({ maxHeight: "200px", paddingTop: "0px", borderTopWidth: "0px", rowGap: `${GAP}px` } as CSSStyleDeclaration)
      : original(element),
  );
  vi.spyOn(Element.prototype, "getBoundingClientRect").mockImplementation(function (this: Element) {
    const top = Number((this as HTMLElement).dataset.top ?? 0);
    const height = this.matches("[data-top]") ? TILE : 0;
    return { top, bottom: top + height, left: 0, right: 100, width: 100, height, x: 0, y: top, toJSON: () => ({}) } as DOMRect;
  });
}

function renderStrip(count: number, holdBeat = vi.fn()) {
  const board = { holdBeat } as unknown as BoardControllerValue;
  const tiles = Array.from({ length: count }, (_, index) => <li key={index} data-top={Math.floor(index / COLUMNS) * (TILE + GAP)}>tile {index + 1}</li>);
  const view = renderWithProviders(<BoardContext.Provider value={board}><section><SupportingStrip>{tiles}</SupportingStrip></section></BoardContext.Provider>);
  return { ...view, holdBeat, shown: () => screen.getAllByText(/^tile \d+$/).map((tile) => tile.textContent) };
}

describe("SupportingStrip", () => { // [REQ:CC-P1-017]
  beforeEach(() => {
    vi.useFakeTimers();
    stubLayout();
  });
  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  it("pages tiles that overrun the strip budget, holding the column count and the beat until every page is shown", () => {
    const { holdBeat, shown, container } = renderStrip(12);
    const list = container.querySelector("[data-testid='metric-list']");
    expect(list).toHaveAttribute("data-density", "compact");
    expect(list).toHaveAttribute("data-paged", "true");
    expect(list?.getAttribute("style")).toContain("--strip-columns: 6");
    expect(screen.getByTestId("strip-pages")).toHaveTextContent("1 / 2");
    expect(shown()).toEqual(["tile 1", "tile 2", "tile 3", "tile 4", "tile 5", "tile 6"]);
    expect(holdBeat.mock.lastCall?.[1]).toBe(true);

    act(() => { vi.advanceTimersByTime(STRIP_PAGE_MS); });
    expect(screen.getByTestId("strip-pages")).toHaveTextContent("2 / 2");
    expect(shown()).toEqual(["tile 7", "tile 8", "tile 9", "tile 10", "tile 11", "tile 12"]);
    act(() => { vi.advanceTimersByTime(STRIP_PAGE_MS); });
    expect(holdBeat.mock.lastCall?.[1]).toBe(false);
  });

  it("shows every tile on one page, at normal density, when they fit", () => {
    const { holdBeat, shown, container } = renderStrip(6);
    const list = container.querySelector("[data-testid='metric-list']");
    expect(list).toHaveAttribute("data-density", "normal");
    expect(list).not.toHaveAttribute("data-paged");
    expect(screen.queryByTestId("strip-pages")).toBeNull();
    expect(shown()).toHaveLength(6);
    expect(holdBeat).not.toHaveBeenCalledWith(expect.any(String), true);
  });
});
