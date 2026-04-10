/**
 * Tests for GeneratorFormFooter component.
 */

import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { GeneratorFormFooter } from "./GeneratorFormFooter";
import type { ValidationError } from ".";

describe("GeneratorFormFooter", () => {
  const defaultProps = {
    validationErrors: [] as ValidationError[],
    onDismissErrors: vi.fn(),
    isPending: false,
    isError: false,
    errorMessage: null,
    isUpdateMode: false,
  };

  it("renders submit button with 'Generate Desktop Application' text", () => {
    render(<GeneratorFormFooter {...defaultProps} />);

    expect(
      screen.getByRole("button", { name: "Generate Desktop Application" })
    ).toBeInTheDocument();
  });

  it("renders submit button with 'Update Desktop Application' in update mode", () => {
    render(<GeneratorFormFooter {...defaultProps} isUpdateMode={true} />);

    expect(
      screen.getByRole("button", { name: "Update Desktop Application" })
    ).toBeInTheDocument();
  });

  it("shows 'Generating...' when isPending is true", () => {
    render(<GeneratorFormFooter {...defaultProps} isPending={true} />);

    expect(screen.getByRole("button", { name: "Generating..." })).toBeInTheDocument();
  });

  it("disables button when isPending is true", () => {
    render(<GeneratorFormFooter {...defaultProps} isPending={true} />);

    expect(screen.getByRole("button")).toBeDisabled();
  });

  it("disables button when there are validation errors", () => {
    const errors: ValidationError[] = [
      { id: "scenario_required", message: "Scenario is required" },
    ];
    render(<GeneratorFormFooter {...defaultProps} validationErrors={errors} />);

    expect(screen.getByRole("button", { name: "Generate Desktop Application" })).toBeDisabled();
  });

  it("enables button when no pending state and no validation errors", () => {
    render(<GeneratorFormFooter {...defaultProps} />);

    expect(screen.getByRole("button")).toBeEnabled();
  });

  it("displays error message when isError and errorMessage provided", () => {
    render(
      <GeneratorFormFooter
        {...defaultProps}
        isError={true}
        errorMessage="Something went wrong"
      />
    );

    expect(screen.getByText(/Error:/)).toBeInTheDocument();
    expect(screen.getByText("Something went wrong")).toBeInTheDocument();
  });

  it("does not display error message when isError is false", () => {
    render(
      <GeneratorFormFooter
        {...defaultProps}
        isError={false}
        errorMessage="Something went wrong"
      />
    );

    expect(screen.queryByText(/Error:/)).not.toBeInTheDocument();
  });

  it("does not display error message when errorMessage is null", () => {
    render(
      <GeneratorFormFooter {...defaultProps} isError={true} errorMessage={null} />
    );

    expect(screen.queryByText(/Error:/)).not.toBeInTheDocument();
  });

  it("renders ValidationErrors component with errors", () => {
    const errors: ValidationError[] = [
      { id: "scenario_required", message: "Please select a scenario" },
    ];
    render(<GeneratorFormFooter {...defaultProps} validationErrors={errors} />);

    expect(screen.getByText("Please select a scenario")).toBeInTheDocument();
  });

  it("calls onDismissErrors when dismiss is triggered", () => {
    const onDismissErrors = vi.fn();
    const errors: ValidationError[] = [
      { id: "scenario_required", message: "Please select a scenario" },
    ];
    render(
      <GeneratorFormFooter
        {...defaultProps}
        validationErrors={errors}
        onDismissErrors={onDismissErrors}
      />
    );

    // Find the dismiss button (X icon button) - it's the one that's not the submit button
    const buttons = screen.getAllByRole("button");
    const dismissButton = buttons.find(
      (btn) => !btn.textContent?.includes("Desktop Application")
    );
    expect(dismissButton).toBeDefined();
    if (dismissButton) {
      fireEvent.click(dismissButton);
    }

    expect(onDismissErrors).toHaveBeenCalledTimes(1);
  });

  it("does not render ValidationErrors when no errors", () => {
    render(<GeneratorFormFooter {...defaultProps} validationErrors={[]} />);

    // The ValidationErrors component shouldn't show any content when empty
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });
});
