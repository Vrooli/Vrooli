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

  it("shows 'Next Round' button when onRunWorkshop is provided", () => {
    render(<WorkshopPanel {...defaultProps} rounds={[makeRound()]} onRunWorkshop={vi.fn()} />);
    expect(screen.getByText("Next Round")).toBeInTheDocument();
  });

  it("hides 'Next Round' button when onRunWorkshop is not provided", () => {
    render(<WorkshopPanel {...defaultProps} rounds={[makeRound()]} />);
    expect(screen.queryByText("Next Round")).not.toBeInTheDocument();
  });

  it("fires onRunWorkshop when Next Round is clicked", () => {
    const onRunWorkshop = vi.fn();
    render(
      <WorkshopPanel {...defaultProps} rounds={[makeRound()]} onRunWorkshop={onRunWorkshop} />,
    );

    fireEvent.click(screen.getByText("Next Round"));
    expect(onRunWorkshop).toHaveBeenCalledTimes(1);
  });

  it("shows default running label when isRunningWorkshop is true", () => {
    render(
      <WorkshopPanel {...defaultProps} rounds={[makeRound()]} isRunningWorkshop onRunWorkshop={vi.fn()} />,
    );
    expect(screen.getByText("Running…")).toBeInTheDocument();
  });

  it("shows custom runningLabel when provided", () => {
    render(
      <WorkshopPanel {...defaultProps} rounds={[makeRound()]} isRunningWorkshop onRunWorkshop={vi.fn()} runningLabel="Running workshop…" />,
    );
    expect(screen.getByText("Running workshop…")).toBeInTheDocument();
  });

  it("renders finalize as a primary action alongside next round", () => {
    render(
      <WorkshopPanel
        {...defaultProps}
        rounds={[makeRound()]}
        primaryActionLabel="Finalize Plan"
        onPrimaryAction={vi.fn()}
        onRunWorkshop={vi.fn()}
      />,
    );

    expect(screen.getByText("Finalize Plan")).toBeInTheDocument();
    expect(screen.getByText("Next Round")).toBeInTheDocument();
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

  it("shows delete button on workshop items", () => {
    const round = makeRound({
      items: [
        { id: "d1", type: "decision", topic: "Scope", text: "Pick scope", options: [], selected: null },
        { id: "i1", type: "info", text: "Background context" },
      ],
    });
    render(<WorkshopPanel {...defaultProps} rounds={[round]} />);

    const deleteButtons = screen.getAllByTitle("Delete item");
    expect(deleteButtons).toHaveLength(2);
  });

  it("hides delete button when disabled", () => {
    const round = makeRound();
    render(<WorkshopPanel {...defaultProps} rounds={[round]} disabled />);

    expect(screen.queryAllByTitle("Delete item")).toHaveLength(0);
  });

  it("clicking delete removes item from rendered list", async () => {
    const round = makeRound({
      items: [
        { id: "d1", type: "decision", topic: "Question 1", text: "Q1", options: [], selected: null },
        { id: "d2", type: "decision", topic: "Question 2", text: "Q2", options: [], selected: null },
      ],
    });
    render(<WorkshopPanel {...defaultProps} rounds={[round]} />);

    expect(screen.getByText("Question 1")).toBeInTheDocument();
    expect(screen.getByText("Question 2")).toBeInTheDocument();

    const deleteButtons = screen.getAllByTitle("Delete item");
    const firstDeleteButton = deleteButtons[0];
    expect(firstDeleteButton).toBeDefined();
    if (!firstDeleteButton) {
      throw new Error("Delete button not found");
    }
    await fireEvent.click(firstDeleteButton);

    // Confirm the deletion.
    const confirmButton = screen.getByText("Delete");
    await fireEvent.click(confirmButton);

    expect(screen.queryByText("Question 1")).not.toBeInTheDocument();
    expect(screen.getByText("Question 2")).toBeInTheDocument();
  });

  it("shows round menu and fires onDeleteRound", async () => {
    const onDeleteRound = vi.fn();
    const round = makeRound({ round: 3 });
    render(
      <WorkshopPanel {...defaultProps} rounds={[round]} onDeleteRound={onDeleteRound} />,
    );

    // Click the round menu button
    const menuBtn = screen.getByTitle("Round actions");
    await fireEvent.click(menuBtn);

    // Click "Delete round"
    const deleteRoundBtn = screen.getByText("Delete round");
    await fireEvent.click(deleteRoundBtn);

    expect(onDeleteRound).toHaveBeenCalledWith(3);
  });

  it("hides round menu when disabled", () => {
    render(
      <WorkshopPanel {...defaultProps} rounds={[makeRound()]} onDeleteRound={vi.fn()} disabled />,
    );

    expect(screen.queryByTitle("Round actions")).not.toBeInTheDocument();
  });

  it("hides round menu when onDeleteRound is not provided", () => {
    render(<WorkshopPanel {...defaultProps} rounds={[makeRound()]} />);

    expect(screen.queryByTitle("Round actions")).not.toBeInTheDocument();
  });

  it("auto-saves after deleting an item", async () => {
    vi.useFakeTimers();
    const onSaveRound = vi.fn();
    const round = makeRound();
    render(
      <WorkshopPanel
        {...defaultProps}
        rounds={[round]}
        onSaveRound={onSaveRound}
      />,
    );

    const deleteButtons = screen.getAllByTitle("Delete item");
    const firstDeleteButton = deleteButtons[0];
    expect(firstDeleteButton).toBeDefined();
    if (!firstDeleteButton) {
      throw new Error("Delete button not found");
    }
    await fireEvent.click(firstDeleteButton);

    // Confirm the deletion.
    const confirmButton = screen.getByText("Delete");
    await fireEvent.click(confirmButton);

    // No manual save button — auto-save fires after debounce
    expect(screen.queryByText("Save Responses")).not.toBeInTheDocument();
    expect(onSaveRound).not.toHaveBeenCalled();

    // Advance past the 600ms debounce
    vi.advanceTimersByTime(700);
    expect(onSaveRound).toHaveBeenCalledTimes(1);
    expect(onSaveRound).toHaveBeenCalledWith(1, expect.any(String));

    vi.useRealTimers();
  });

  // --- Finalization indicator & post-finalize confirmation ---

  it("shows 'Plan finalized' badge when isFinalized is true", () => {
    render(
      <WorkshopPanel
        {...defaultProps}
        rounds={[makeRound()]}
        isFinalized
        deliverableLabel="Plan"
      />,
    );

    expect(screen.getByText("Plan finalized")).toBeInTheDocument();
  });

  it("shows 'Conclusion finalized' badge for research items", () => {
    render(
      <WorkshopPanel
        {...defaultProps}
        backlogKind="research"
        rounds={[makeRound()]}
        isFinalized
        deliverableLabel="Conclusion"
      />,
    );

    expect(screen.getByText("Conclusion finalized")).toBeInTheDocument();
  });

  it("does not show finalized badge when isFinalized is false", () => {
    render(
      <WorkshopPanel
        {...defaultProps}
        rounds={[makeRound()]}
        isFinalized={false}
      />,
    );

    expect(screen.queryByText("Plan finalized")).not.toBeInTheDocument();
  });

  it("fires onRunWorkshop directly when not finalized", () => {
    const onRunWorkshop = vi.fn();
    render(
      <WorkshopPanel
        {...defaultProps}
        rounds={[makeRound()]}
        onRunWorkshop={onRunWorkshop}
        isFinalized={false}
      />,
    );

    fireEvent.click(screen.getByText("Next Round"));
    // Should fire immediately — no confirmation dialog
    expect(onRunWorkshop).toHaveBeenCalledTimes(1);
  });

  it("shows confirmation dialog instead of firing immediately when finalized", () => {
    const onRunWorkshop = vi.fn();
    render(
      <WorkshopPanel
        {...defaultProps}
        rounds={[makeRound()]}
        onRunWorkshop={onRunWorkshop}
        isFinalized
        deliverableLabel="Plan"
      />,
    );

    fireEvent.click(screen.getByText("Next Round"));
    // Should NOT fire yet — confirmation dialog should appear
    expect(onRunWorkshop).not.toHaveBeenCalled();
    expect(screen.getByText("Start new workshop round?")).toBeInTheDocument();
    expect(screen.getByText(/plan has already been finalized/i)).toBeInTheDocument();
  });

  it("fires onRunWorkshop after confirming post-finalize dialog", () => {
    const onRunWorkshop = vi.fn();
    render(
      <WorkshopPanel
        {...defaultProps}
        rounds={[makeRound()]}
        onRunWorkshop={onRunWorkshop}
        isFinalized
      />,
    );

    fireEvent.click(screen.getByText("Next Round"));
    // Confirm in the dialog
    fireEvent.click(screen.getByText("Start Round"));
    expect(onRunWorkshop).toHaveBeenCalledTimes(1);
  });

  it("does not fire onRunWorkshop when cancelling post-finalize dialog", () => {
    const onRunWorkshop = vi.fn();
    render(
      <WorkshopPanel
        {...defaultProps}
        rounds={[makeRound()]}
        onRunWorkshop={onRunWorkshop}
        isFinalized
      />,
    );

    fireEvent.click(screen.getByText("Next Round"));
    // Cancel
    fireEvent.click(screen.getByText("Cancel"));
    expect(onRunWorkshop).not.toHaveBeenCalled();
    // Dialog should close
    expect(screen.queryByText("Start new workshop round?")).not.toBeInTheDocument();
  });
});
