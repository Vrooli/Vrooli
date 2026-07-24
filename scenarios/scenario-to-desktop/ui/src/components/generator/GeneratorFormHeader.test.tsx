/**
 * Tests for GeneratorFormHeader component.
 */

import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@/test-utils";
import { GeneratorFormHeader } from "./GeneratorFormHeader";
import type { ValidationStatus } from "../../lib/api";

describe("GeneratorFormHeader", () => {
  const mockValidationStatus: ValidationStatus = {
    scenario_name: "test-scenario",
    overall_status: "valid",
    stages: {
      bundle: { status: "valid" },
      preflight: { status: "valid" },
      generate: { status: "none" },
      build: { status: "none" },
      smoke_test: { status: "none" },
    },
  };

  const defaultProps = {
    scenarioName: "test-scenario",
    validationStatus: mockValidationStatus,
    createdLabel: null,
    updatedLabel: null,
    isSaving: false,
    onReset: vi.fn(),
  };

  it("renders reset button", () => {
    render(<GeneratorFormHeader {...defaultProps} />);

    expect(screen.getByRole("button", { name: "Reset progress" })).toBeInTheDocument();
  });

  it("disables reset button when no scenario name", () => {
    render(<GeneratorFormHeader {...defaultProps} scenarioName="" />);

    expect(screen.getByRole("button", { name: "Reset progress" })).toBeDisabled();
  });

  it("enables reset button when scenario name is provided", () => {
    render(<GeneratorFormHeader {...defaultProps} scenarioName="my-scenario" />);

    expect(screen.getByRole("button", { name: "Reset progress" })).toBeEnabled();
  });

  it("calls onReset when reset button is clicked", () => {
    const onReset = vi.fn();
    render(<GeneratorFormHeader {...defaultProps} onReset={onReset} />);

    fireEvent.click(screen.getByRole("button", { name: "Reset progress" }));

    expect(onReset).toHaveBeenCalledTimes(1);
  });

  it("shows created timestamp when provided", () => {
    render(
      <GeneratorFormHeader
        {...defaultProps}
        createdLabel="1/20/2026, 10:30:00 AM"
      />
    );

    expect(screen.getByText("Started 1/20/2026, 10:30:00 AM")).toBeInTheDocument();
  });

  it("shows updated timestamp when provided", () => {
    render(
      <GeneratorFormHeader
        {...defaultProps}
        updatedLabel="1/20/2026, 11:00:00 AM"
      />
    );

    expect(screen.getByText("Saved 1/20/2026, 11:00:00 AM")).toBeInTheDocument();
  });

  it("shows both timestamps when both are provided", () => {
    render(
      <GeneratorFormHeader
        {...defaultProps}
        createdLabel="1/20/2026, 10:30:00 AM"
        updatedLabel="1/20/2026, 11:00:00 AM"
      />
    );

    expect(screen.getByText("Started 1/20/2026, 10:30:00 AM")).toBeInTheDocument();
    expect(screen.getByText("Saved 1/20/2026, 11:00:00 AM")).toBeInTheDocument();
  });

  it("shows saving indicator when isSaving is true", () => {
    render(
      <GeneratorFormHeader
        {...defaultProps}
        updatedLabel="1/20/2026, 10:30:00 AM"
        isSaving={true}
      />
    );

    expect(screen.getByText("Saving...")).toBeInTheDocument();
  });

  it("does not show saving indicator when isSaving is false", () => {
    render(
      <GeneratorFormHeader
        {...defaultProps}
        updatedLabel="1/20/2026, 10:30:00 AM"
        isSaving={false}
      />
    );

    expect(screen.queryByText("Saving...")).not.toBeInTheDocument();
  });

  it("does not show timestamps section when no timestamps provided", () => {
    render(<GeneratorFormHeader {...defaultProps} />);

    expect(screen.queryByText(/Started/)).not.toBeInTheDocument();
    expect(screen.queryByText(/Saved/)).not.toBeInTheDocument();
  });

  it("handles null validationStatus", () => {
    render(
      <GeneratorFormHeader {...defaultProps} validationStatus={null} />
    );

    // Should not throw and should render the reset button
    expect(screen.getByRole("button", { name: "Reset progress" })).toBeInTheDocument();
  });
});
