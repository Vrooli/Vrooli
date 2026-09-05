// [REQ:REQ-P0-003] Welcome Step Component
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach } from "vitest";
// provider-free-exception: StepWelcome is static wizard content with no provider dependency.
import { StepWelcome } from "./StepWelcome";

afterEach(cleanup);

describe("StepWelcome", () => {
  it("renders welcome heading", () => {
    render(<StepWelcome />);
    expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent("This machine is about to become a Vrooli node");
  });

  it("renders step-welcome test id", () => {
    render(<StepWelcome />);
    expect(screen.getByTestId("step-welcome")).toBeInTheDocument();
  });

  it("shows description text", () => {
    render(<StepWelcome />);
    expect(screen.getByText(/nothing is written until you approve it/i)).toBeInTheDocument();
  });

  it("states the three setup commitments", () => {
    render(<StepWelcome />);
    expect(screen.getByText("Installs local services")).toBeInTheDocument();
    expect(screen.getByText("Asks before touching the host")).toBeInTheDocument();
    expect(screen.getByText("Stays reversible")).toBeInTheDocument();
  });

  it("keeps decorative icons out of the accessibility tree", () => {
    render(<StepWelcome />);
    const container = screen.getByTestId("step-welcome");
    const svg = container.querySelector("svg");
    expect(svg).toHaveAttribute("aria-hidden", "true");
  });

  it("uses the semantic welcome lede treatment", () => {
    render(<StepWelcome />);
    const description = screen.getByText(/nothing is written until you approve it/i);
    expect(description.className).toContain("welcome-screen__lede");
  });
});
