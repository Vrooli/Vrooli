import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import { RollupProgressBar, rollupTotal } from "./rollup-progress-bar";
import type { InitiativeRollup } from "../../types";

function makeRollup(overrides: Partial<InitiativeRollup> = {}): InitiativeRollup {
  const base = { completed: 0, inProgress: 0, failed: 0, pending: 0, total: 0, archived: 0, ...overrides };
  // Auto-compute total if not explicitly set
  if (!overrides.total) {
    base.total = base.completed + base.inProgress + base.failed + base.pending;
  }
  return base as InitiativeRollup;
}

describe("rollupTotal", () => {
  it("sums active progress fields", () => {
    expect(rollupTotal(makeRollup({ completed: 2, inProgress: 3, failed: 1, pending: 4 }))).toBe(10);
  });

  it("returns 0 for empty rollup", () => {
    expect(rollupTotal(makeRollup())).toBe(0);
  });

  it("excludes archived items from the total", () => {
    expect(rollupTotal(makeRollup({ completed: 2, pending: 1, archived: 5 }))).toBe(3);
  });
});

describe("RollupProgressBar", () => {
  it("renders nothing when total is 0", () => {
    const { container } = render(<RollupProgressBar rollup={makeRollup()} />);
    expect(container.firstChild).toBeNull();
  });

  it("renders segmented bar with correct segments", () => {
    render(<RollupProgressBar rollup={makeRollup({ completed: 5, pending: 5 })} />);
    const bar = screen.getByTestId("rollup-progress-bar");
    expect(bar).toBeDefined();
    // Two segments: completed and pending
    const segments = bar.querySelectorAll("[title]");
    expect(segments.length).toBe(2);
    expect(segments[0]?.getAttribute("title")).toBe("5 completed");
    expect(segments[1]?.getAttribute("title")).toBe("5 pending");
  });

  it("hides failed segment when count is 0", () => {
    render(<RollupProgressBar rollup={makeRollup({ completed: 3, pending: 2 })} />);
    const bar = screen.getByTestId("rollup-progress-bar");
    const titles = Array.from(bar.querySelectorAll("[title]")).map((el) => el.getAttribute("title"));
    expect(titles).not.toContain(expect.stringContaining("failed"));
  });

  it("shows numeric labels when showLabels is true", () => {
    render(<RollupProgressBar rollup={makeRollup({ completed: 2, inProgress: 1, pending: 3 })} showLabels />);
    expect(screen.getByText("2 completed")).toBeDefined();
    expect(screen.getByText("1 in progress")).toBeDefined();
    expect(screen.getByText("3 pending")).toBeDefined();
    expect(screen.getByText("6 total")).toBeDefined();
  });

  it("hides failed label when showLabels is true but failed count is 0", () => {
    render(<RollupProgressBar rollup={makeRollup({ completed: 1, pending: 1 })} showLabels />);
    expect(screen.queryByText(/failed/)).toBeNull();
  });

  it("shows failed label when count > 0", () => {
    render(<RollupProgressBar rollup={makeRollup({ failed: 2, pending: 1 })} showLabels />);
    expect(screen.getByText("2 failed")).toBeDefined();
  });

  it("does not show labels by default", () => {
    render(<RollupProgressBar rollup={makeRollup({ completed: 1, pending: 1 })} />);
    expect(screen.queryByText("1 completed")).toBeNull();
  });

  it("applies custom barHeight class", () => {
    render(<RollupProgressBar rollup={makeRollup({ completed: 1 })} barHeight="h-1" />);
    const bar = screen.getByTestId("rollup-progress-bar");
    const inner = bar.querySelector(".h-1");
    expect(inner).not.toBeNull();
  });

  it("ignores archived items when rendering active progress", () => {
    render(<RollupProgressBar rollup={makeRollup({ completed: 1, archived: 4 })} showLabels />);
    expect(screen.getByText("1 completed")).toBeDefined();
    expect(screen.getByText("1 total")).toBeDefined();
    expect(screen.queryByText(/archived/i)).toBeNull();
  });
});
