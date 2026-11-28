import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { renderWithProviders as render } from "../../test-utils";
import { describe, expect, it } from "vitest";
import { screen } from "@testing-library/react";
import TopSafeArea from "../TopSafeArea";

/**
 * iOS 26 replaces its blurred top edge with a solid colour only when the
 * element under the middle of the top edge sits in a sticky or fixed bar that
 * spans the width (not the height) of the viewport, with one opaque colour and
 * no backdrop filter on the way up. `TopSafeArea` is that bar; the strip is
 * the hit element and carries the colour.
 */
describe("TopSafeArea as the iOS top edge bar", () => {
  it("is a sticky bar whose first child is the status strip", () => {
    render(
      <TopSafeArea testId="edge">
        <div>chrome</div>
      </TopSafeArea>,
    );
    const bar = screen.getByTestId("edge");
    expect(bar).toHaveAttribute("data-wc-top-edge-bar");
    expect(bar.className.split(/\s+/)).toEqual(expect.arrayContaining(["sticky", "top-0"]));
    // Between the strip and the bar nothing may add a second colour or a blur.
    expect(bar.className).not.toMatch(/\bbg-|backdrop-/);
    const strip = screen.getByTestId("edge-fill");
    expect(strip.parentElement).toBe(bar);
    expect(bar.firstElementChild).toBe(strip);
  });

  it("ships the status-bar style that lets iOS extend the bar's colour", () => {
    // `black-translucent` puts the page under the status bar, where the
    // installed iOS 26 app blurs the top chrome whatever bar sits there.
    const html = readFileSync(resolve(__dirname, "../../../index.html"), "utf8");
    const style = /name="apple-mobile-web-app-status-bar-style"\s+content="([^"]+)"/.exec(html)?.[1];
    expect(style).toBe("default");
  });

  it("stays the bar when callers add classes", () => {
    render(
      <TopSafeArea testId="edge" className="pb-1">
        <div>chrome</div>
      </TopSafeArea>,
    );
    expect(screen.getByTestId("edge").className).toMatch(/\bsticky\b/);
  });
});
