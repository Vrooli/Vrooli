import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { PhaseSelector } from "./PhaseSelector";

describe("PhaseSelector", () => {
  it("renders all phase options", () => {
    const onTogglePhase = vi.fn();
    render(<PhaseSelector selectedPhases={[]} onTogglePhase={onTogglePhase} />);

    expect(screen.getByText("Unit Tests")).toBeInTheDocument();
    expect(screen.getByText("Integration Tests")).toBeInTheDocument();
    expect(screen.getByText("E2E Playbooks")).toBeInTheDocument();
    expect(screen.getByText("Business Validation")).toBeInTheDocument();
  });

  it("shows selected styling for selected phases", () => {
    const onTogglePhase = vi.fn();
    render(
      <PhaseSelector
        selectedPhases={["unit", "integration"]}
        onTogglePhase={onTogglePhase}
      />
    );

    const unitButton = screen.getByText("Unit Tests").closest("button");
    expect(unitButton).toHaveClass("border-cyan-400");
    expect(unitButton).toHaveClass("bg-cyan-400/10");

    const integrationButton = screen.getByText("Integration Tests").closest("button");
    expect(integrationButton).toHaveClass("border-cyan-400");
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

    expect(screen.getByText(/Generate unit tests for individual functions/)).toBeInTheDocument();
    expect(screen.getByText(/Generate integration tests for component interactions/)).toBeInTheDocument();
  });

  it("renders header content", () => {
    const onTogglePhase = vi.fn();
    render(<PhaseSelector selectedPhases={[]} onTogglePhase={onTogglePhase} />);

    expect(screen.getByText("Test phases")).toBeInTheDocument();
    expect(screen.getByText("Select phases to generate")).toBeInTheDocument();
  });
});
