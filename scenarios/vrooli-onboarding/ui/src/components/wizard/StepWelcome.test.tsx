// [REQ:REQ-P0-003] Welcome Step Component
import { render, screen } from "@testing-library/react";
// provider-free-exception: StepWelcome is static wizard content with no provider dependency.
import { StepWelcome } from "./StepWelcome";

describe("StepWelcome", () => {
  it("renders welcome heading", () => {
    render(<StepWelcome />);
    expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent("Welcome to Vrooli");
  });

  it("renders step-welcome test id", () => {
    render(<StepWelcome />);
    expect(screen.getByTestId("step-welcome")).toBeInTheDocument();
  });

  it("shows description text", () => {
    render(<StepWelcome />);
    expect(screen.getByText(/guide you through configuring/i)).toBeInTheDocument();
  });

  it("shows 3 info cards for wizard steps", () => {
    render(<StepWelcome />);
    expect(screen.getByText("Select Resources")).toBeInTheDocument();
    expect(screen.getByText("Review Configuration")).toBeInTheDocument();
    expect(screen.getByText("Generate Config")).toBeInTheDocument();
  });

  it("shows Get Started hint text", () => {
    render(<StepWelcome />);
    expect(screen.getByText(/click/i)).toBeInTheDocument();
    expect(screen.getByText("Get Started")).toBeInTheDocument();
  });

  it("renders Rocket icon with aria-hidden", () => {
    render(<StepWelcome />);
    const container = screen.getByTestId("step-welcome");
    const svg = container.querySelector("svg");
    expect(svg).toHaveAttribute("aria-hidden", "true");
  });

  it("has proper text contrast (slate-200 for body text)", () => {
    render(<StepWelcome />);
    const description = screen.getByText(/guide you through configuring/i);
    expect(description.className).toContain("text-slate-200");
  });
});
