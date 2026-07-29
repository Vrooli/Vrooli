// [REQ:REQ-P0-003] Wizard Shell Component
import { render, screen, fireEvent } from "@testing-library/react";
// provider-free-exception: WizardShell receives all state and callbacks as props and does not access app providers.
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

  it("renders step indicators for all onboarding V2 steps", () => {
    render(<WizardShell {...defaultProps}>Content</WizardShell>);
    expect(screen.getByTestId("step-indicator-0")).toBeInTheDocument();
    expect(screen.getByTestId("step-indicator-1")).toBeInTheDocument();
    expect(screen.getByTestId("step-indicator-2")).toBeInTheDocument();
    expect(screen.getByTestId("step-indicator-3")).toBeInTheDocument();
    expect(screen.getByTestId("step-indicator-4")).toBeInTheDocument();
    expect(screen.getByTestId("step-indicator-5")).toBeInTheDocument();
    expect(screen.getByTestId("step-indicator-6")).toBeInTheDocument();
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
    expect(progressBar).toHaveAttribute("aria-valuemax", "7");
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
    expect(screen.getByText("Step 2: Scenarios")).toBeInTheDocument();
  });

  it("renders mobile step counter", () => {
    render(<WizardShell {...defaultProps} currentStep={2}>Content</WizardShell>);
    expect(screen.getByText("3/8")).toBeInTheDocument();
  });

  it("renders mobile dot progress indicators", () => {
    render(<WizardShell {...defaultProps} currentStep={1}>Content</WizardShell>);
    const mobileProgress = screen.getByRole("list", { name: /step progress/i });
    expect(mobileProgress).toBeInTheDocument();
    const dots = mobileProgress.querySelectorAll("[role='listitem']");
    expect(dots).toHaveLength(8);
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

  it("calls onNext when Next button is clicked", () => {
    const onNext = vi.fn();
    render(<WizardShell {...defaultProps} onNext={onNext}>Content</WizardShell>);
    fireEvent.click(screen.getByTestId("wizard-next"));
    expect(onNext).toHaveBeenCalledTimes(1);
  });

  it("calls onPrev when Back button is clicked", () => {
    const onPrev = vi.fn();
    render(<WizardShell {...defaultProps} currentStep={1} onPrev={onPrev} showPrev={true}>Content</WizardShell>);
    fireEvent.click(screen.getByTestId("wizard-prev"));
    expect(onPrev).toHaveBeenCalledTimes(1);
  });

  it("progress bar width reflects step position", () => {
    render(<WizardShell {...defaultProps} currentStep={1}>Content</WizardShell>);
    const progressFill = screen.getByTestId("progress-bar");
    // Step 1 of 8 steps (0-indexed): 1/7 * 100 ≈ 14.29%
    expect(progressFill.style.width).toMatch(/14\.2/);
  });

  it("shows step number for non-completed future steps", () => {
    render(<WizardShell {...defaultProps} currentStep={0}>Content</WizardShell>);
    // Step 0 is current (shows "1"), later steps are future.
    expect(screen.getByTestId("step-indicator-0")).toHaveTextContent("1");
    expect(screen.getByTestId("step-indicator-3")).toHaveTextContent("4");
  });

  it("progress bar width is 0% on first step", () => {
    render(<WizardShell {...defaultProps} currentStep={0}>Content</WizardShell>);
    const progressFill = screen.getByTestId("progress-bar");
    expect(progressFill.style.width).toBe("0%");
  });

  it("progress bar width is 100% on the final V2 step", () => {
    render(<WizardShell {...defaultProps} currentStep={7}>Content</WizardShell>);
    const progressFill = screen.getByTestId("progress-bar");
    expect(progressFill.style.width).toBe("100%");
  });

  it("does not call onGoToStep for current step", () => {
    const goToStep = vi.fn();
    render(<WizardShell {...defaultProps} currentStep={1} onGoToStep={goToStep}>Content</WizardShell>);
    const step1 = screen.getByTestId("step-indicator-1");
    fireEvent.click(step1);
    expect(goToStep).not.toHaveBeenCalled();
  });
});
