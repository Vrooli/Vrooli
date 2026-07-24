/**
 * Tests for ValidationErrors component.
 */

import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@/test-utils";
import { ValidationErrors } from "./ValidationErrors";
import type { ValidationError } from "../../domain/generator";

describe("ValidationErrors", () => {
  it("renders nothing when errors array is empty", () => {
    const { container } = render(<ValidationErrors errors={[]} />);
    expect(container).toBeEmptyDOMElement();
  });

  it("renders single error correctly", () => {
    const errors: ValidationError[] = [
      { id: "scenario", message: "Scenario name is required" },
    ];
    render(<ValidationErrors errors={errors} />);
    expect(screen.getByText("Scenario name is required")).toBeInTheDocument();
    expect(screen.getByText(/Please fix the following issue/)).toBeInTheDocument();
  });

  it("renders multiple errors correctly", () => {
    const errors: ValidationError[] = [
      { id: "scenario", message: "Scenario name is required" },
      { id: "platforms", message: "Select at least one platform" },
      { id: "output", message: "Output location is required" },
    ];
    render(<ValidationErrors errors={errors} />);
    expect(screen.getByText("Scenario name is required")).toBeInTheDocument();
    expect(screen.getByText("Select at least one platform")).toBeInTheDocument();
    expect(screen.getByText("Output location is required")).toBeInTheDocument();
    expect(screen.getByText(/Please fix the following issues/)).toBeInTheDocument();
  });

  it("renders dismiss button when onDismiss is provided", () => {
    const onDismiss = vi.fn();
    const errors: ValidationError[] = [
      { id: "test", message: "Test error" },
    ];
    render(<ValidationErrors errors={errors} onDismiss={onDismiss} />);

    const dismissButton = screen.getByRole("button");
    expect(dismissButton).toBeInTheDocument();
  });

  it("calls onDismiss when dismiss button is clicked", () => {
    const onDismiss = vi.fn();
    const errors: ValidationError[] = [
      { id: "test", message: "Test error" },
    ];
    render(<ValidationErrors errors={errors} onDismiss={onDismiss} />);

    const dismissButton = screen.getByRole("button");
    fireEvent.click(dismissButton);
    expect(onDismiss).toHaveBeenCalledTimes(1);
  });

  it("does not render dismiss button when onDismiss is not provided", () => {
    const errors: ValidationError[] = [
      { id: "test", message: "Test error" },
    ];
    render(<ValidationErrors errors={errors} />);

    expect(screen.queryByRole("button")).not.toBeInTheDocument();
  });

  it("applies custom className when provided", () => {
    const errors: ValidationError[] = [
      { id: "test", message: "Test error" },
    ];
    const { container } = render(
      <ValidationErrors errors={errors} className="custom-class" />
    );

    const wrapper = container.firstChild;
    expect(wrapper).toHaveClass("custom-class");
  });

  it("uses correct singular form for single issue", () => {
    const errors: ValidationError[] = [
      { id: "single", message: "Single error" },
    ];
    render(<ValidationErrors errors={errors} />);
    expect(screen.getByText(/issue before generating/)).toBeInTheDocument();
    expect(screen.queryByText(/issues before generating/)).not.toBeInTheDocument();
  });

  it("uses correct plural form for multiple issues", () => {
    const errors: ValidationError[] = [
      { id: "first", message: "First error" },
      { id: "second", message: "Second error" },
    ];
    render(<ValidationErrors errors={errors} />);
    expect(screen.getByText(/issues before generating/)).toBeInTheDocument();
  });
});
