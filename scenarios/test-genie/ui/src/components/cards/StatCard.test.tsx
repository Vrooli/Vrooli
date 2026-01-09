import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { StatCard } from "./StatCard";

describe("StatCard", () => {
  it("renders the label", () => {
    render(<StatCard label="Total Runs" value={42} />);
    expect(screen.getByText("Total Runs")).toBeInTheDocument();
  });

  it("renders numeric values", () => {
    render(<StatCard label="Count" value={42} />);
    expect(screen.getByText("42")).toBeInTheDocument();
  });

  it("pads single digit numbers with zero", () => {
    render(<StatCard label="Count" value={5} />);
    expect(screen.getByText("05")).toBeInTheDocument();
  });

  it("renders string values as-is", () => {
    render(<StatCard label="Status" value="Active" />);
    expect(screen.getByText("Active")).toBeInTheDocument();
  });

  it("renders description when provided", () => {
    render(
      <StatCard
        label="Queue"
        value={3}
        description="Pending requests"
      />
    );
    expect(screen.getByText("Pending requests")).toBeInTheDocument();
  });

  it("does not render description when not provided", () => {
    render(<StatCard label="Queue" value={3} />);
    // The article should only have label and value
    const article = screen.getByRole("article");
    expect(article.children).toHaveLength(2);
  });

  it("applies custom className", () => {
    render(<StatCard label="Test" value={1} className="custom-class" />);
    const article = screen.getByRole("article");
    expect(article).toHaveClass("custom-class");
  });

  it("has correct base styles", () => {
    render(<StatCard label="Test" value={1} />);
    const article = screen.getByRole("article");
    expect(article).toHaveClass("rounded-2xl");
    expect(article).toHaveClass("border-white/5");
  });

  it("renders zero correctly", () => {
    render(<StatCard label="Count" value={0} />);
    expect(screen.getByText("00")).toBeInTheDocument();
  });
});
