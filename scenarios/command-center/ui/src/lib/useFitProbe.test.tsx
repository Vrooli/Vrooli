import { useRef } from "react";
import { act } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../test-utils/renderWithProviders";
import { measureFit, useFitProbe } from "./useFitProbe";

const VIEWPORT = { width: 1280, height: 720 };

/** jsdom has no layout: each element reports the rect authored in data-rect ("top,left,right,bottom"). */
function stubLayout() {
  vi.spyOn(Element.prototype, "getBoundingClientRect").mockImplementation(function (this: Element) {
    const [top = 0, left = 0, right = 0, bottom = 0] = ((this as HTMLElement).dataset.rect ?? "0,0,0,0").split(",").map(Number);
    return { top, left, right, bottom, width: right - left, height: bottom - top, x: left, y: top, toJSON: () => ({}) } as DOMRect;
  });
  // measureFit reads only overflow from computed style.
  vi.spyOn(window, "getComputedStyle").mockImplementation((element: Element) => {
    const overflow = (element as HTMLElement).dataset.clip !== undefined ? "hidden" : "visible";
    return { overflowX: overflow, overflowY: overflow } as CSSStyleDeclaration;
  });
}

function room(html: string): HTMLElement {
  const element = document.createElement("main");
  element.innerHTML = html;
  return element;
}

const strip = (leaf: string) => `<section data-testid="room-supporting"><ul data-testid="metric-list" data-rect="580,40,1240,700">${leaf}</ul></section>`;

describe("measureFit — a landscape room reports its own fit", () => { // [REQ:CC-P1-017]
  beforeEach(stubLayout);
  afterEach(() => vi.restoreAllMocks());

  it("is ok when every figure is on screen and the hero ends above the strip", () => {
    const report = measureFit(room(`<section data-testid="room-hero"><span data-rect="100,40,400,560">66</span></section>${strip(`<li><span data-rect="600,40,200,640">29</span></li>`)}`), VIEWPORT);
    expect(report).toEqual({ state: "ok", detail: "" });
  });
  it("names a strip figure that leaves the screen", () => {
    const report = measureFit(room(`<section data-testid="room-hero"><span data-rect="100,40,400,560">66</span></section>${strip(`<li><span data-rect="690,40,200,760">Conversions</span></li>`)}`), VIEWPORT);
    expect(report.state).toBe("overflow");
    expect(report.detail).toContain('strip span "Conversions" leaves the screen');
  });
  it("names a hero that runs into the strip", () => {
    const report = measureFit(room(`<section data-testid="room-hero"><span data-rect="100,40,400,600">rank</span></section>${strip("")}`), VIEWPORT);
    expect(report).toEqual({ state: "overflow", detail: "hero runs 20px into the strip" });
  });
  it("counts only what a clipping ancestor draws, so an auto-scroll list's hidden rows are not an overflow", () => {
    const hero = `<section data-testid="room-hero"><div data-clip data-rect="300,40,600,540"><span data-rect="500,40,600,900">row 9</span></div></section>`;
    expect(measureFit(room(`${hero}${strip("")}`), VIEWPORT).state).toBe("ok");
  });
});

function Probe({ beatKey, dataKey }: { beatKey: string; dataKey: string }) {
  const ref = useRef<HTMLElement>(null);
  useFitProbe(ref, beatKey, dataKey);
  return (
    <main ref={ref} data-testid="room">
      <section data-testid="room-hero"><span data-rect="100,40,400,560">66</span></section>
    </main>
  );
}

describe("useFitProbe — stamps the room once the beat settles", () => { // [REQ:CC-P1-017]
  let portrait = false;
  beforeEach(() => {
    vi.useFakeTimers();
    stubLayout();
    portrait = false;
    vi.stubGlobal("matchMedia", () => ({ matches: portrait }));
    vi.stubGlobal("innerWidth", VIEWPORT.width);
    vi.stubGlobal("innerHeight", VIEWPORT.height);
  });
  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  const stamped = (view: ReturnType<typeof renderWithProviders>) => view.getByTestId("room");
  const settle = () => act(() => { vi.advanceTimersByTime(1200); });

  it("is pending while a new beat settles, then reports ok", () => {
    const view = renderWithProviders(<Probe beatKey="forge:0" dataKey="1" />);
    expect(stamped(view)).toHaveAttribute("data-fit", "pending");
    settle();
    expect(stamped(view)).toHaveAttribute("data-fit", "ok");
    view.rerender(<Probe beatKey="forge:1" dataKey="1" />);
    expect(stamped(view)).toHaveAttribute("data-fit", "pending");
  });
  it("keeps its verdict across a data refresh until the re-check lands", () => {
    const view = renderWithProviders(<Probe beatKey="forge:0" dataKey="1" />);
    settle();
    view.rerender(<Probe beatKey="forge:0" dataKey="2" />);
    expect(stamped(view)).toHaveAttribute("data-fit", "ok");
  });
  it("reports scroll in portrait, where the desk view may scroll", () => {
    portrait = true;
    const view = renderWithProviders(<Probe beatKey="forge:0" dataKey="1" />);
    settle();
    expect(stamped(view)).toHaveAttribute("data-fit", "scroll");
  });
});
