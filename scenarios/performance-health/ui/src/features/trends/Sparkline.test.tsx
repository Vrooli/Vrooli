import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";

import { Sparkline } from "./Sparkline";

describe("Sparkline", () => {
  it("renders an em-dash placeholder when there is nothing to plot", () => {
    render(<Sparkline values={[]} label="empty" testId="spark" />);
    expect(screen.getByTestId("spark")).toHaveTextContent("—");
  });

  it("filters out non-finite values before plotting", () => {
    render(<Sparkline values={[NaN, Infinity]} label="bad" testId="spark" />);
    expect(screen.getByTestId("spark")).toHaveTextContent("—");
  });

  it("renders a flat midline for a single sample", () => {
    render(<Sparkline values={[5]} label="single" testId="spark" />);
    const svg = screen.getByTestId("spark");
    expect(svg.tagName.toLowerCase()).toBe("svg");
    expect(svg).toHaveAttribute("role", "img");
    expect(svg).toHaveAttribute("aria-label", "single");
  });

  it("renders a polyline for a multi-sample series", () => {
    const { container } = render(
      <Sparkline values={[1, 4, 2, 8]} label="series" testId="spark" />,
    );
    const polyline = container.querySelector("polyline");
    expect(polyline).not.toBeNull();
    // 4 plotted points → 4 coordinate pairs.
    expect(polyline?.getAttribute("points")?.split(" ").length).toBe(4);
  });
});
