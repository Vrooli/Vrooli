import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { WizardShell } from "./WizardShell";
import type { V2Step } from "../../types";

afterEach(cleanup);

const steps: V2Step[] = [
  { id: "welcome", ordinal: 0, title: "Welcome", route: "/", deferred: false },
  { id: "scenarios", ordinal: 1, title: "Capabilities", route: "/setup/scenarios", deferred: false },
  { id: "apply", ordinal: 2, title: "Apply", route: "/setup/apply", deferred: false },
];

function renderShell(currentStep = 1) {
  const onGoToStep = vi.fn();
  render(
    <WizardShell currentStep={currentStep} steps={steps} onNext={vi.fn()} onPrev={vi.fn()} onGoToStep={onGoToStep}>
      <div>Content</div>
    </WizardShell>,
  );
  return onGoToStep;
}

describe("WizardShell", () => {
  it("renders one compact progress affordance", () => {
    renderShell();
    expect(screen.getByLabelText("Adjust: Capabilities")).toBeInTheDocument();
    expect(screen.queryByRole("navigation", { name: "Wizard steps" })).not.toBeInTheDocument();
    expect(screen.getByText("2 / 3")).toBeInTheDocument();
    expect(screen.queryByTestId("wizard-target-chrome")).not.toBeInTheDocument();
  });

  it("allows navigation back through the library footer", () => {
    const onGoToStep = renderShell();
    const previousControls = screen.getAllByTestId("wizard-prev");
    fireEvent.click(previousControls[previousControls.length - 1]!);
    expect(onGoToStep).toHaveBeenCalledWith(0);
  });

  it("does not expose future steps as clickable controls", () => {
    renderShell(0);
    expect(screen.queryByRole("button", { name: /Capabilities/ })).not.toBeInTheDocument();
  });
});
