import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { ReadinessBar } from "./readiness-bar";
import type { ReadinessIndicatorData } from "../../lib/maturity";
import type { ReadinessDimension } from "../../types/domain";

const allScores = (n: number): Record<ReadinessDimension, number> => ({
  problem_clarity: n,
  scope_defined: n,
  approach_solid: n,
  testable: n,
  risk_awareness: n,
});

const makeData = (overrides?: Partial<ReadinessIndicatorData>): ReadinessIndicatorData => ({
  rawScores: allScores(2),
  effectiveScores: allScores(2),
  roundsCompleted: 1,
  ready: false,
  pendingItems: 0,
  pendingSynthesis: false,
  hasPlan: false,
  nextNudge: null,
  ...overrides,
});

describe("ReadinessBar", () => {
  it("returns null when roundsCompleted is 0", () => {
    const { container } = render(
      <ReadinessBar data={makeData({ roundsCompleted: 0 })} />,
    );
    expect(container.innerHTML).toBe("");
  });

  it("renders 5 labeled dots when rounds > 0", () => {
    render(<ReadinessBar data={makeData({ roundsCompleted: 1 })} />);

    // Each dimension renders a dot with a title and initial letter
    const segments = screen.getAllByTitle(/\/3$/);
    expect(segments).toHaveLength(5);

    // Dimension initials are rendered
    expect(screen.getByText("P")).toBeInTheDocument();
    expect(screen.getByText("S")).toBeInTheDocument();
    expect(screen.getByText("A")).toBeInTheDocument();
    expect(screen.getByText("T")).toBeInTheDocument();
    expect(screen.getByText("R")).toBeInTheDocument();
  });

  it("score-0 uses ring outline instead of filled background", () => {
    render(
      <ReadinessBar data={makeData({ effectiveScores: allScores(0) })} />,
    );
    const dot = screen.getByText("P");
    expect(dot.className).toContain("ring-");
    expect(dot.className).not.toContain("bg-rose");
  });

  it("shows 'Ready' text when data.ready is true", () => {
    render(<ReadinessBar data={makeData({ ready: true })} />);
    expect(screen.getByText("Ready")).toBeInTheDocument();
  });

  it("does not show 'Ready' text when data.ready is false", () => {
    render(<ReadinessBar data={makeData({ ready: false })} />);
    expect(screen.queryByText("Ready")).not.toBeInTheDocument();
  });

  it("shows round badge R1, R2, etc.", () => {
    render(<ReadinessBar data={makeData({ roundsCompleted: 2 })} />);
    expect(screen.getByText("R2")).toBeInTheDocument();
  });

  it("applies className prop", () => {
    const { container } = render(
      <ReadinessBar data={makeData()} className="mt-4" />,
    );
    expect(container.firstChild).toHaveClass("mt-4");
  });
});
