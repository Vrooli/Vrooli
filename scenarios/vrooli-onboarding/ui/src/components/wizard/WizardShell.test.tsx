import { cleanup, fireEvent, screen } from "../../test-utils";
import { afterEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "@vrooli/api-base/testing";
import { WizardShell } from "./WizardShell";
import type { WizardStep } from "../../api/session";

afterEach(cleanup);

const steps: WizardStep[] = [
  { id: "welcome", ordinal: 0, title: "Welcome", route: "/", deferred: false },
  { id: "scenarios", ordinal: 1, title: "Capabilities", route: "/setup/scenarios", deferred: false },
  { id: "apply", ordinal: 2, title: "Apply", route: "/setup/apply", deferred: false },
];

function renderShell(currentStep = 1, saveState?: "idle" | "saving" | "saved" | "failed" | "conflict", onRetry?: () => void) {
  const onGoToStep = vi.fn();
  renderWithProviders(
    <WizardShell currentStep={currentStep} steps={steps} onNext={vi.fn()} onPrev={vi.fn()} onGoToStep={onGoToStep} operatorStateSaveState={saveState} onRetryOperatorStateSave={onRetry}>
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

  it.each([
    ["saving", "Saving preferences…"],
    ["saved", "Preferences saved locally. Host effects require review and apply."],
    ["failed", "Preferences not saved. Your edit is retained for retry."],
    ["conflict", "Preferences conflict. Your edit is retained for rebase and retry."],
  ] as const)("exposes the %s durable save state", (state, message) => {
    const onRetry = vi.fn();
    renderShell(1, state, onRetry);
    expect(screen.getByTestId("operator-state-save-state")).toHaveTextContent(message);
    if (state === "failed" || state === "conflict") fireEvent.click(screen.getByTestId("operator-state-save-retry"));
    if (state === "failed" || state === "conflict") expect(onRetry).toHaveBeenCalledTimes(1);
  });
});
