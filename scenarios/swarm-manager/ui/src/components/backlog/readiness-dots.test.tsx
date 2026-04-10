import { describe, it, expect } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { ReadinessDots } from "./readiness-dots";
import type { WorkshopRound, ReadinessDimension } from "../../types/domain";

const allScores = (n: number): Record<ReadinessDimension, number> => ({
  problem_clarity: n,
  scope_defined: n,
  approach_solid: n,
  testable: n,
  risk_awareness: n,
});

const makeRound = (overrides?: Partial<WorkshopRound>): WorkshopRound => ({
  round: 1,
  generated_at: "2026-03-20T00:00:00Z",
  readiness: allScores(2),
  items: [],
  ...overrides,
});

describe("ReadinessDots", () => {
  it("renders 5 dots with dimension initials", () => {
    render(<ReadinessDots round={makeRound()} />);

    expect(screen.getByText("P")).toBeInTheDocument();
    expect(screen.getByText("S")).toBeInTheDocument();
    expect(screen.getByText("A")).toBeInTheDocument();
    expect(screen.getByText("T")).toBeInTheDocument();
    expect(screen.getByText("R")).toBeInTheDocument();
  });

  it("score-0 dots use ring outline instead of filled background", () => {
    render(<ReadinessDots round={makeRound({ readiness: allScores(0) })} />);

    const pDot = screen.getByText("P");
    expect(pDot.className).toContain("ring-");
    expect(pDot.className).not.toContain("bg-rose");
    expect(pDot.className).not.toContain("bg-amber");
    expect(pDot.className).not.toContain("bg-emerald");
  });

  it("score-1 dots have rose background", () => {
    render(<ReadinessDots round={makeRound({ readiness: allScores(1) })} />);
    expect(screen.getByText("P").className).toContain("bg-rose");
  });

  it("score-2 dots have amber background", () => {
    render(<ReadinessDots round={makeRound({ readiness: allScores(2) })} />);
    expect(screen.getByText("P").className).toContain("bg-amber");
  });

  it("score-3 dots have emerald background", () => {
    render(<ReadinessDots round={makeRound({ readiness: allScores(3) })} />);
    expect(screen.getByText("P").className).toContain("bg-emerald");
  });

  it("clicking dots opens popover with dimension details", () => {
    render(<ReadinessDots round={makeRound()} />);

    expect(screen.queryByTestId("readiness-popover")).not.toBeInTheDocument();

    fireEvent.click(screen.getByTestId("readiness-dots-trigger"));

    expect(screen.getByTestId("readiness-popover")).toBeInTheDocument();
    expect(screen.getByText("Problem Clarity")).toBeInTheDocument();
    expect(screen.getByText("Scope")).toBeInTheDocument();
    expect(screen.getByText("Approach")).toBeInTheDocument();
    expect(screen.getByText("Testability")).toBeInTheDocument();
    expect(screen.getByText("Risk Awareness")).toBeInTheDocument();
  });

  it("popover shows score badges for each dimension", () => {
    render(<ReadinessDots round={makeRound({ readiness: allScores(2) })} />);
    fireEvent.click(screen.getByTestId("readiness-dots-trigger"));

    const badges = screen.getAllByText("2/3");
    expect(badges).toHaveLength(5);
  });

  it("popover shows round number", () => {
    render(<ReadinessDots round={makeRound({ round: 3 })} />);
    fireEvent.click(screen.getByTestId("readiness-dots-trigger"));

    expect(screen.getByText(/Round 3/)).toBeInTheDocument();
  });

  it("shows delta arrows when prevRound is provided", () => {
    const prevRound = makeRound({ round: 1, readiness: { ...allScores(1), problem_clarity: 2 } });
    const currRound = makeRound({ round: 2, readiness: { ...allScores(2), problem_clarity: 1 } });

    render(<ReadinessDots round={currRound} prevRound={prevRound} />);
    fireEvent.click(screen.getByTestId("readiness-dots-trigger"));

    // problem_clarity went 2→1 (down)
    const pcDelta = screen.getByTestId("readiness-delta-problem_clarity");
    expect(pcDelta.textContent).toBe("▼");

    // scope_defined went 1→2 (up)
    const sdDelta = screen.getByTestId("readiness-delta-scope_defined");
    expect(sdDelta.textContent).toBe("▲");
  });

  it("does not show delta arrows when scores are unchanged", () => {
    const prevRound = makeRound({ round: 1, readiness: allScores(2) });
    const currRound = makeRound({ round: 2, readiness: allScores(2) });

    render(<ReadinessDots round={currRound} prevRound={prevRound} />);
    fireEvent.click(screen.getByTestId("readiness-dots-trigger"));

    expect(screen.queryByTestId("readiness-delta-problem_clarity")).not.toBeInTheDocument();
  });

  it("does not show delta arrows when no prevRound", () => {
    render(<ReadinessDots round={makeRound()} />);
    fireEvent.click(screen.getByTestId("readiness-dots-trigger"));

    expect(screen.queryByTestId("readiness-delta-problem_clarity")).not.toBeInTheDocument();
  });

  it("clicking trigger again closes the popover", () => {
    render(<ReadinessDots round={makeRound()} />);

    fireEvent.click(screen.getByTestId("readiness-dots-trigger"));
    expect(screen.getByTestId("readiness-popover")).toBeInTheDocument();

    fireEvent.click(screen.getByTestId("readiness-dots-trigger"));
    expect(screen.queryByTestId("readiness-popover")).not.toBeInTheDocument();
  });

  it("handles mixed scores correctly", () => {
    const readiness = {
      problem_clarity: 0,
      scope_defined: 1,
      approach_solid: 2,
      testable: 3,
      risk_awareness: 0,
    };
    render(<ReadinessDots round={makeRound({ readiness })} />);

    // P and R should be ring (score 0)
    expect(screen.getByText("P").className).toContain("ring-");
    expect(screen.getByText("R").className).toContain("ring-");
    // S should be rose (score 1)
    expect(screen.getByText("S").className).toContain("bg-rose");
    // A should be amber (score 2)
    expect(screen.getByText("A").className).toContain("bg-amber");
    // T should be emerald (score 3)
    expect(screen.getByText("T").className).toContain("bg-emerald");
  });
});
