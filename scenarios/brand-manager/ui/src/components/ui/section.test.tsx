import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { Section } from "./section";

// [REQ:BM-REQ-UI-DASHBOARD]

describe("Section", () => {
  it("renders children", () => {
    render(<Section>Content here</Section>);
    expect(screen.getByText("Content here")).toBeTruthy();
  });

  it("renders title when provided", () => {
    render(<Section title="My Section">Body</Section>);
    expect(screen.getByText("My Section")).toBeTruthy();
  });

  it("omits title when not provided", () => {
    const { container } = render(<Section>Body only</Section>);
    expect(container.querySelector("h2")).toBeNull();
  });

  it("applies data-testid", () => {
    render(<Section testId="test-section">Body</Section>);
    expect(screen.getByTestId("test-section")).toBeTruthy();
  });

  it("merges custom className", () => {
    render(<Section testId="sec" className="extra-class">Body</Section>);
    expect(screen.getByTestId("sec").className).toContain("extra-class");
    expect(screen.getByTestId("sec").className).toContain("rounded-xl");
  });

  it("renders as a semantic section element", () => {
    const { container } = render(<Section>Content</Section>);
    expect(container.querySelector("section")).toBeTruthy();
  });
});
