import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { PhaseSelector } from "./PhaseSelector";
import { PHASES_FOR_GENERATION } from "../../lib/constants";

describe("PhaseSelector", () => {
  it("renders all phase options", () => {
    const onTogglePhase = vi.fn();
    render(<PhaseSelector selectedPhases={[]} onTogglePhase={onTogglePhase} />);

    for (const phase of PHASES_FOR_GENERATION) {
      expect(screen.getByText(phase.label)).toBeInTheDocument();
    }
  });

  it("shows selected styling for selected phases", () => {
    const onTogglePhase = vi.fn();
    render(
      <PhaseSelector
        selectedPhases={["unit", "playbooks"]}
        onTogglePhase={onTogglePhase}
      />
    );

    const unitButton = screen.getByText("Unit Tests").closest("button");
    expect(unitButton).toHaveClass("border-cyan-400");
    expect(unitButton).toHaveClass("bg-cyan-400/10");

    const playbooksButton = screen.getByText("E2E Playbooks").closest("button");
    expect(playbooksButton).toHaveClass("border-cyan-400");
  });

  it("shows unselected styling for non-selected phases", () => {
    const onTogglePhase = vi.fn();
    render(
      <PhaseSelector
        selectedPhases={["unit"]}
        onTogglePhase={onTogglePhase}
      />
    );

    const playbooksButton = screen.getByText("E2E Playbooks").closest("button");
    expect(playbooksButton).toHaveClass("border-white/10");
    expect(playbooksButton).not.toHaveClass("border-cyan-400");
  });

  it("calls onTogglePhase when a phase is clicked", () => {
    const onTogglePhase = vi.fn();
    render(<PhaseSelector selectedPhases={[]} onTogglePhase={onTogglePhase} />);

    const unitButton = screen.getByText("Unit Tests").closest("button");
    fireEvent.click(unitButton!);

    expect(onTogglePhase).toHaveBeenCalledWith("unit");
  });

  it("shows checkmark for selected phases", () => {
    const onTogglePhase = vi.fn();
    render(
      <PhaseSelector
        selectedPhases={["unit"]}
        onTogglePhase={onTogglePhase}
      />
    );

    const unitButton = screen.getByText("Unit Tests").closest("button");
    const svg = unitButton?.querySelector("svg");
    expect(svg).toBeInTheDocument();
  });

  it("does not show checkmark for unselected phases", () => {
    const onTogglePhase = vi.fn();
    render(
      <PhaseSelector
        selectedPhases={[]}
        onTogglePhase={onTogglePhase}
      />
    );

    const unitButton = screen.getByText("Unit Tests").closest("button");
    const svg = unitButton?.querySelector("svg");
    expect(svg).toBeNull();
  });

  it("has correct data-testid attribute", () => {
    const onTogglePhase = vi.fn();
    render(<PhaseSelector selectedPhases={[]} onTogglePhase={onTogglePhase} />);

    expect(screen.getByTestId("test-genie-phase-selector")).toBeInTheDocument();
  });

  it("renders phase descriptions", () => {
    const onTogglePhase = vi.fn();
    render(<PhaseSelector selectedPhases={[]} onTogglePhase={onTogglePhase} />);

    for (const phase of PHASES_FOR_GENERATION) {
      expect(screen.getByText(phase.description)).toBeInTheDocument();
    }
  });

  it("renders header content", () => {
    const onTogglePhase = vi.fn();
    render(<PhaseSelector selectedPhases={[]} onTogglePhase={onTogglePhase} />);

    expect(screen.getByText("Test phases")).toBeInTheDocument();
    expect(screen.getByText("Select phases")).toBeInTheDocument();
  });

  it("locks phases to unit when lockToUnit is true", () => {
    const onTogglePhase = vi.fn();
    render(
      <PhaseSelector
        selectedPhases={["unit"]}
        onTogglePhase={onTogglePhase}
        lockToUnit
      />
    );

    const playbooksButton = screen.getByText("E2E Playbooks").closest("button");
    expect(playbooksButton).toBeDisabled();
    fireEvent.click(playbooksButton!);
    expect(onTogglePhase).not.toHaveBeenCalled();

    const unitButton = screen.getByText("Unit Tests").closest("button");
    expect(unitButton).not.toBeDisabled();
  });

  it("uses task-specific copy when a generation task is selected", () => {
    const onTogglePhase = vi.fn();
    render(
      <PhaseSelector
        selectedPhases={[]}
        onTogglePhase={onTogglePhase}
        task="coverage"
      />
    );

    expect(screen.getByText("Add unit test coverage")).toBeInTheDocument();
    expect(screen.getByText("Add E2E playbook coverage")).toBeInTheDocument();
  });
});
