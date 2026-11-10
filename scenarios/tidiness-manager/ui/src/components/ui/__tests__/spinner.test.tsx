import { describe, it, expect } from "vitest";
import { screen } from "@testing-library/react";
import { renderWithProviders } from "../../../test-utils";
import { Spinner } from "../spinner";

describe("Spinner", () => {
  it("renders with default props", () => {
    renderWithProviders(<Spinner />);
    const spinner = screen.getByTestId("spinner");
    expect(spinner).toBeInTheDocument();
  });

  it("has proper accessibility attributes", () => {
    renderWithProviders(<Spinner />);
    const spinner = screen.getByRole("status");
    expect(spinner).toHaveAttribute("aria-label", "Loading");
    expect(screen.getByText("Loading...")).toBeInTheDocument();
  });

  it("renders sr-only text for screen readers", () => {
    renderWithProviders(<Spinner />);
    const srText = screen.getByText("Loading...");
    expect(srText).toHaveClass("sr-only");
  });

  describe("size variants", () => {
    it("renders small size", () => {
      renderWithProviders(<Spinner size="sm" />);
      const spinner = screen.getByTestId("spinner");
      expect(spinner).toHaveClass("h-4", "w-4", "border-2");
    });

    it("renders medium size (default)", () => {
      renderWithProviders(<Spinner size="md" />);
      const spinner = screen.getByTestId("spinner");
      expect(spinner).toHaveClass("h-8", "w-8", "border-3");
    });

    it("renders large size", () => {
      renderWithProviders(<Spinner size="lg" />);
      const spinner = screen.getByTestId("spinner");
      expect(spinner).toHaveClass("h-12", "w-12", "border-4");
    });

    it("defaults to medium size when no size prop provided", () => {
      renderWithProviders(<Spinner />);
      const spinner = screen.getByTestId("spinner");
      expect(spinner).toHaveClass("h-8", "w-8", "border-3");
    });
  });

  describe("styling", () => {
    it("has animate-spin class", () => {
      renderWithProviders(<Spinner />);
      const spinner = screen.getByTestId("spinner");
      expect(spinner).toHaveClass("animate-spin");
    });

    it("has rounded-full class", () => {
      renderWithProviders(<Spinner />);
      const spinner = screen.getByTestId("spinner");
      expect(spinner).toHaveClass("rounded-full");
    });

    it("has proper border styling", () => {
      renderWithProviders(<Spinner />);
      const spinner = screen.getByTestId("spinner");
      expect(spinner).toHaveClass("border-solid", "border-slate-600", "border-t-slate-100");
    });

    it("accepts custom className", () => {
      renderWithProviders(<Spinner className="custom-class" />);
      const spinner = screen.getByTestId("spinner");
      expect(spinner).toHaveClass("custom-class");
    });

    it("merges custom className with default classes", () => {
      renderWithProviders(<Spinner className="text-red-500" />);
      const spinner = screen.getByTestId("spinner");
      expect(spinner).toHaveClass("text-red-500", "animate-spin", "rounded-full");
    });
  });

  describe("additional props", () => {
    it("forwards additional HTML attributes", () => {
      renderWithProviders(<Spinner data-custom="test-value" />);
      const spinner = screen.getByTestId("spinner");
      expect(spinner).toHaveAttribute("data-custom", "test-value");
    });

    it("supports id prop", () => {
      renderWithProviders(<Spinner id="custom-spinner" />);
      const spinner = screen.getByTestId("spinner");
      expect(spinner).toHaveAttribute("id", "custom-spinner");
    });

    it("supports title prop", () => {
      renderWithProviders(<Spinner title="Loading data" />);
      const spinner = screen.getByTestId("spinner");
      expect(spinner).toHaveAttribute("title", "Loading data");
    });
  });
});
