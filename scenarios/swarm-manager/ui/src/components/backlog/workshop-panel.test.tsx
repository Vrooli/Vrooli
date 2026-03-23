import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { WorkshopPanel } from "./workshop-panel";
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
  items: [
    { id: "d1", type: "decision", topic: "Scope", text: "Pick scope", options: [], selected: null },
    { id: "i1", type: "info", text: "Background context" },
  ],
  ...overrides,
});

const defaultProps = {
  backlogKind: "idea" as const,
  backlogName: "test-item",
};

describe("WorkshopPanel", () => {
  it("shows empty state and Start button when rounds array is empty", () => {
    render(<WorkshopPanel {...defaultProps} rounds={[]} />);

    expect(screen.getByText("No workshop rounds yet")).toBeInTheDocument();
    expect(screen.getByText("Start Workshop")).toBeInTheDocument();
  });

  it("fires onRunWorkshop when Start button is clicked", () => {
    const onRunWorkshop = vi.fn();
    render(
      <WorkshopPanel {...defaultProps} rounds={[]} onRunWorkshop={onRunWorkshop} />,
    );

    fireEvent.click(screen.getByText("Start Workshop"));
    expect(onRunWorkshop).toHaveBeenCalledTimes(1);
  });

  it("disables Start button when disabled prop is true", () => {
    render(<WorkshopPanel {...defaultProps} rounds={[]} disabled />);

    const btn = screen.getByText("Start Workshop").closest("button");
    expect(btn).toBeDisabled();
  });

  it("renders round headers when rounds exist", () => {
    const rounds = [makeRound({ round: 1 }), makeRound({ round: 2 })];
    render(<WorkshopPanel {...defaultProps} rounds={rounds} />);

    expect(screen.getByText("Round 1")).toBeInTheDocument();
    expect(screen.getByText("Round 2")).toBeInTheDocument();
    expect(screen.getByText(/Workshop Rounds \(2\)/)).toBeInTheDocument();
  });

  it("latest round is expanded by default", () => {
    const rounds = [
      makeRound({ round: 1, items: [{ id: "old", type: "info", text: "Old context" }] }),
      makeRound({ round: 2, items: [{ id: "new", type: "info", text: "New context" }] }),
    ];
    render(<WorkshopPanel {...defaultProps} rounds={rounds} />);

    // Latest round's items should be visible
    expect(screen.getByText("New context")).toBeInTheDocument();
    // Older round's items should be hidden (collapsed)
    expect(screen.queryByText("Old context")).not.toBeInTheDocument();
  });

  it("toggling a collapsed round expands it", () => {
    const rounds = [
      makeRound({ round: 1, items: [{ id: "old", type: "info", text: "Old info" }] }),
      makeRound({ round: 2, items: [{ id: "new", type: "info", text: "New info" }] }),
    ];
    render(<WorkshopPanel {...defaultProps} rounds={rounds} />);

    // Round 1 is collapsed — click to expand
    fireEvent.click(screen.getByText("Round 1"));
    expect(screen.getByText("Old info")).toBeInTheDocument();
  });

  it("shows 'Next Round' button when rounds exist", () => {
    render(<WorkshopPanel {...defaultProps} rounds={[makeRound()]} />);
    expect(screen.getByText("Next Round")).toBeInTheDocument();
  });

  it("fires onRunWorkshop when Next Round is clicked", () => {
    const onRunWorkshop = vi.fn();
    render(
      <WorkshopPanel {...defaultProps} rounds={[makeRound()]} onRunWorkshop={onRunWorkshop} />,
    );

    fireEvent.click(screen.getByText("Next Round"));
    expect(onRunWorkshop).toHaveBeenCalledTimes(1);
  });

  it("shows 'Running...' text when isRunningWorkshop is true", () => {
    render(
      <WorkshopPanel {...defaultProps} rounds={[makeRound()]} isRunningWorkshop />,
    );
    expect(screen.getByText("Running...")).toBeInTheDocument();
  });

  it("shows pending decision count on round header", () => {
    const round = makeRound({
      items: [
        { id: "d1", type: "decision", selected: null },
        { id: "d2", type: "decision", selected: null },
        { id: "i1", type: "info", text: "note" },
      ],
    });
    render(<WorkshopPanel {...defaultProps} rounds={[round]} />);

    // The component shows pending decisions as "{count}D"
    expect(screen.getByText("2D")).toBeInTheDocument();
  });
});
