// [REQ:REQ-P0-003] Wizard Shell Component
import { render, screen, fireEvent } from "@testing-library/react";
import { vi } from "vitest";
import { WizardShell } from "./WizardShell";

const defaultProps = {
  currentStep: 0,
  onNext: vi.fn(),
  onPrev: vi.fn(),
};

describe("WizardShell", () => {
  it("renders wizard shell container", () => {
    render(<WizardShell {...defaultProps}>Content</WizardShell>);
    expect(screen.getByTestId("wizard-shell")).toBeInTheDocument();
  });

  it("renders children content", () => {
    render(<WizardShell {...defaultProps}><p>Test Content</p></WizardShell>);
    expect(screen.getByText("Test Content")).toBeInTheDocument();
  });

  it("renders step indicators for all 4 steps", () => {
    render(<WizardShell {...defaultProps}>Content</WizardShell>);
    expect(screen.getByTestId("step-indicator-0")).toBeInTheDocument();
    expect(screen.getByTestId("step-indicator-1")).toBeInTheDocument();
    expect(screen.getByTestId("step-indicator-2")).toBeInTheDocument();
    expect(screen.getByTestId("step-indicator-3")).toBeInTheDocument();
  });

  it("renders ordered list for step indicators", () => {
    render(<WizardShell {...defaultProps}>Content</WizardShell>);
    const stepList = screen.getByTestId("wizard-steps-desktop");
    expect(stepList).toBeInTheDocument();
    expect(stepList.tagName).toBe("OL");
  });

  it("marks current step with aria-current", () => {
    render(<WizardShell {...defaultProps} currentStep={1}>Content</WizardShell>);
    const desktopSteps = screen.getByTestId("wizard-steps-desktop");
    const listItems = desktopSteps.querySelectorAll("li");
    expect(listItems[1]).toHaveAttribute("aria-current", "step");
    expect(listItems[0]).not.toHaveAttribute("aria-current");
  });

  it("renders progress bar with correct aria attributes", () => {
    render(<WizardShell {...defaultProps} currentStep={2}>Content</WizardShell>);
    const progressBar = screen.getByRole("progressbar");
    expect(progressBar).toHaveAttribute("aria-valuenow", "2");
    expect(progressBar).toHaveAttribute("aria-valuemin", "0");
    expect(progressBar).toHaveAttribute("aria-valuemax", "3");
  });

  it("shows Next button by default", () => {
    render(<WizardShell {...defaultProps}>Content</WizardShell>);
    expect(screen.getByTestId("wizard-next")).toBeInTheDocument();
  });

  it("hides Next button when showNext is false", () => {
    render(<WizardShell {...defaultProps} showNext={false}>Content</WizardShell>);
    expect(screen.queryByTestId("wizard-next")).not.toBeInTheDocument();
  });

  it("shows Back button when showPrev is true and step > 0", () => {
    render(<WizardShell {...defaultProps} currentStep={1} showPrev={true}>Content</WizardShell>);
    expect(screen.getByTestId("wizard-prev")).toBeInTheDocument();
  });

  it("hides Back button on first step", () => {
    render(<WizardShell {...defaultProps} showPrev={false}>Content</WizardShell>);
    expect(screen.queryByTestId("wizard-prev")).not.toBeInTheDocument();
  });

  it("disables Next button when nextDisabled is true", () => {
    render(<WizardShell {...defaultProps} nextDisabled={true}>Content</WizardShell>);
    expect(screen.getByTestId("wizard-next")).toBeDisabled();
  });

  it("displays custom next label", () => {
    render(<WizardShell {...defaultProps} nextLabel="Generate Config">Content</WizardShell>);
    expect(screen.getByTestId("wizard-next")).toHaveTextContent("Generate Config");
  });

  it("Back button has accessible label", () => {
    render(<WizardShell {...defaultProps} currentStep={1} showPrev={true}>Content</WizardShell>);
    expect(screen.getByTestId("wizard-prev")).toHaveAttribute("aria-label", "Go to previous step");
  });

  it("Next button has accessible label matching nextLabel", () => {
    render(<WizardShell {...defaultProps} nextLabel="Get Started">Content</WizardShell>);
    expect(screen.getByTestId("wizard-next")).toHaveAttribute("aria-label", "Get Started");
  });

  it("shows checkmark for completed steps", () => {
    render(<WizardShell {...defaultProps} currentStep={2}>Content</WizardShell>);
    // Steps 0 and 1 should show checkmarks
    expect(screen.getByTestId("step-indicator-0")).toHaveTextContent("\u2713");
    expect(screen.getByTestId("step-indicator-1")).toHaveTextContent("\u2713");
    // Step 2 should show number
    expect(screen.getByTestId("step-indicator-2")).toHaveTextContent("3");
  });

  it("renders mobile compact step label with current step name", () => {
    render(<WizardShell {...defaultProps} currentStep={1}>Content</WizardShell>);
    expect(screen.getByText("Step 2: Select Resources")).toBeInTheDocument();
  });

  it("renders mobile step counter", () => {
    render(<WizardShell {...defaultProps} currentStep={2}>Content</WizardShell>);
    expect(screen.getByText("3/4")).toBeInTheDocument();
  });

  it("renders mobile dot progress indicators", () => {
    render(<WizardShell {...defaultProps} currentStep={1}>Content</WizardShell>);
    const mobileProgress = screen.getByRole("list", { name: /step progress/i });
    expect(mobileProgress).toBeInTheDocument();
    const dots = mobileProgress.querySelectorAll("[role='listitem']");
    expect(dots).toHaveLength(4);
  });

  it("makes completed step indicators clickable when onGoToStep is provided", () => {
    const goToStep = vi.fn();
    render(<WizardShell {...defaultProps} currentStep={2} onGoToStep={goToStep}>Content</WizardShell>);
    // Step 0 (completed) should be clickable
    const step0 = screen.getByTestId("step-indicator-0");
    fireEvent.click(step0);
    expect(goToStep).toHaveBeenCalledWith(0);
  });

  it("does not make future step indicators clickable", () => {
    const goToStep = vi.fn();
    render(<WizardShell {...defaultProps} currentStep={1} onGoToStep={goToStep}>Content</WizardShell>);
    // Step 2 (future) should not call onGoToStep
    const step2 = screen.getByTestId("step-indicator-2");
    fireEvent.click(step2);
    expect(goToStep).not.toHaveBeenCalled();
  });

  it("completed step indicators have accessible label for navigation", () => {
    const goToStep = vi.fn();
    render(<WizardShell {...defaultProps} currentStep={2} onGoToStep={goToStep}>Content</WizardShell>);
    const step0 = screen.getByTestId("step-indicator-0");
    expect(step0).toHaveAttribute("aria-label", "Go back to Welcome");
  });
});
