import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { MiniBarChart } from "./mini-bar-chart";

const POINTS = [
  { key: "2026-05-04", label: "2026-05-04", value: 3 },
  { key: "2026-05-11", label: "2026-05-11", value: 7 },
  { key: "2026-05-18", label: "2026-05-18", value: 5 },
];

describe("MiniBarChart", () => {
  it("sizes the viewBox to a square-scaled pixel box (no aspect-ratio stretch)", () => {
    const { container } = render(<MiniBarChart points={POINTS} testId="chart" />);
    const svg = container.querySelector("svg");
    expect(svg).not.toBeNull();
    // Height matches the requested pixel height, and the viewBox uses the same
    // pixel height — so text renders 1:1 rather than stretched horizontally.
    const height = svg?.getAttribute("height");
    expect(svg?.getAttribute("viewBox")).toBe(`0 0 ${svg?.getAttribute("width")} ${height}`);
    // The distorting attribute must be gone.
    expect(svg?.getAttribute("preserveAspectRatio")).not.toBe("none");
  });

  it("renders compact week-start x-axis labels", () => {
    const { container } = render(<MiniBarChart points={POINTS} testId="chart" />);
    const texts = Array.from(container.querySelectorAll("text")).map((t) => t.textContent);
    // "M/D" labels derived from the ISO week-start dates.
    expect(texts).toContain("5/4");
    expect(texts).toContain("5/18");
  });
});
