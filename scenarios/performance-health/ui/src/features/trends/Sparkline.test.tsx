import { describe, expect, it } from "vitest";
import { screen } from "@testing-library/react";
import { renderWithProviders } from "../../test-utils";

import { Sparkline } from "./Sparkline";

describe("Sparkline", () => {
  it("renders an em-dash placeholder when there is nothing to plot", () => {
    renderWithProviders(<Sparkline values={[]} label="empty" testId="spark" />, { extraProviders: (children) => children });
    expect(screen.getByTestId("spark")).toHaveTextContent("—");
  });

  it("filters out non-finite values before plotting", () => {
    renderWithProviders(<Sparkline values={[NaN, Infinity]} label="bad" testId="spark" />, { extraProviders: (children) => children });
    expect(screen.getByTestId("spark")).toHaveTextContent("—");
  });

  it("renders a flat midline for a single sample", () => {
    renderWithProviders(<Sparkline values={[5]} label="single" testId="spark" />, { extraProviders: (children) => children });
    const svg = screen.getByTestId("spark");
    expect(svg.tagName.toLowerCase()).toBe("svg");
    expect(svg).toHaveAttribute("role", "img");
    expect(svg).toHaveAttribute("aria-label", "single");
  });

  it("renders a polyline for a multi-sample series", () => {
    const { container } = renderWithProviders(
      <Sparkline values={[1, 4, 2, 8]} label="series" testId="spark" />,
      { extraProviders: (children) => children },
    );
    const polyline = container.querySelector("polyline");
    expect(polyline).not.toBeNull();
    // 4 plotted points → 4 coordinate pairs.
    expect(polyline?.getAttribute("points")?.split(" ").length).toBe(4);
  });
});
