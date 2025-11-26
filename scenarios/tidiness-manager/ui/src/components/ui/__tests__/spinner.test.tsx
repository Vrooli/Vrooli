import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { Spinner } from "../spinner";

describe("Spinner", () => {
  it("renders with default props", () => {
    render(<Spinner />);
    const spinner = screen.getByTestId("spinner");
    expect(spinner).toBeInTheDocument();
  });

  it("has proper accessibility attributes", () => {
    render(<Spinner />);
    const spinner = screen.getByRole("status");
    expect(spinner).toHaveAttribute("aria-label", "Loading");
    expect(screen.getByText("Loading...")).toBeInTheDocument();
  });

  it("renders sr-only text for screen readers", () => {
    render(<Spinner />);
    const srText = screen.getByText("Loading...");
    expect(srText).toHaveClass("sr-only");
  });

  describe("size variants", () => {
    it("renders small size", () => {
      render(<Spinner size="sm" />);
      const spinner = screen.getByTestId("spinner");
      expect(spinner).toHaveClass("h-4", "w-4", "border-2");
    });

    it("renders medium size (default)", () => {
      render(<Spinner size="md" />);
      const spinner = screen.getByTestId("spinner");
      expect(spinner).toHaveClass("h-8", "w-8", "border-3");
    });

    it("renders large size", () => {
      render(<Spinner size="lg" />);
      const spinner = screen.getByTestId("spinner");
      expect(spinner).toHaveClass("h-12", "w-12", "border-4");
    });

    it("defaults to medium size when no size prop provided", () => {
      render(<Spinner />);
      const spinner = screen.getByTestId("spinner");
      expect(spinner).toHaveClass("h-8", "w-8", "border-3");
    });
  });

  describe("styling", () => {
    it("has animate-spin class", () => {
      render(<Spinner />);
      const spinner = screen.getByTestId("spinner");
      expect(spinner).toHaveClass("animate-spin");
    });

    it("has rounded-full class", () => {
      render(<Spinner />);
      const spinner = screen.getByTestId("spinner");
      expect(spinner).toHaveClass("rounded-full");
    });

    it("has proper border styling", () => {
      render(<Spinner />);
      const spinner = screen.getByTestId("spinner");
      expect(spinner).toHaveClass("border-solid", "border-slate-600", "border-t-slate-100");
    });

    it("accepts custom className", () => {
      render(<Spinner className="custom-class" />);
      const spinner = screen.getByTestId("spinner");
      expect(spinner).toHaveClass("custom-class");
    });

    it("merges custom className with default classes", () => {
      render(<Spinner className="text-red-500" />);
      const spinner = screen.getByTestId("spinner");
      expect(spinner).toHaveClass("text-red-500", "animate-spin", "rounded-full");
    });
  });

  describe("additional props", () => {
    it("forwards additional HTML attributes", () => {
      render(<Spinner data-custom="test-value" />);
      const spinner = screen.getByTestId("spinner");
      expect(spinner).toHaveAttribute("data-custom", "test-value");
    });

    it("supports id prop", () => {
      render(<Spinner id="custom-spinner" />);
      const spinner = screen.getByTestId("spinner");
      expect(spinner).toHaveAttribute("id", "custom-spinner");
    });

    it("supports title prop", () => {
      render(<Spinner title="Loading data" />);
      const spinner = screen.getByTestId("spinner");
      expect(spinner).toHaveAttribute("title", "Loading data");
    });
  });
});
