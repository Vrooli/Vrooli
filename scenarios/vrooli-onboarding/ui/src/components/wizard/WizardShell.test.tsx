import { cleanup, fireEvent, screen } from "../../test-utils";
import { afterEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "@vrooli/api-base/testing";
import { WizardShell, actForStep } from "./WizardShell";
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
  it.each([
    ["welcome", "decide"],
    ["plan", "decide"],
    ["scenarios", "adjust"],
    ["core-set", "adjust"],
    ["resources", "adjust"],
    ["operating-mode", "adjust"],
    ["host", "adjust"],
    ["apply", "commit"],
    [undefined, "commit"],
  ] as const)("maps %s to the %s operator act", (stepID, expected) => {
    expect(actForStep(stepID)).toBe(expected);
  });

  it("renders one compact progress affordance", () => {
    renderShell();
    expect(screen.getByLabelText("Adjust: Capabilities")).toBeInTheDocument();
    expect(screen.queryByRole("navigation", { name: "Wizard steps" })).not.toBeInTheDocument();
    expect(screen.getByText("2 / 3")).toBeInTheDocument();
    expect(screen.queryByTestId("wizard-target-chrome")).not.toBeInTheDocument();
  });

  it("uses the commit act label for the terminal step", () => {
    renderShell(2);
    expect(screen.getByLabelText("Commit: Apply")).toBeInTheDocument();
  });

  it("renders the setup fallback when no active step exists", () => {
    renderWithProviders(
      <WizardShell currentStep={0} steps={[]} onNext={vi.fn()} onPrev={vi.fn()}>
        <div>Content</div>
      </WizardShell>,
    );
    expect(screen.getByLabelText("Commit: Setup")).toBeInTheDocument();
  });

  it("allows navigation back through the library footer", () => {
    const onGoToStep = renderShell();
    const previousControls = screen.getAllByTestId("wizard-prev");
    fireEvent.click(previousControls[previousControls.length - 1]!);
    expect(onGoToStep).toHaveBeenCalledWith(0);
  });

  it("exposes every progress segment as a navigation control", () => {
    const onGoToStep = renderShell(0);
    const segments = screen.getAllByRole("button", { name: /^\d+\. / });
    expect(segments).toHaveLength(3);
    fireEvent.click(segments[1]!);
    expect(onGoToStep).toHaveBeenCalledWith(1);
  });

  it("falls back to directional callbacks when step segments are not directly navigable", () => {
    const onNext = vi.fn();
    const onPrev = vi.fn();
    renderWithProviders(
      <WizardShell currentStep={1} steps={steps} onNext={onNext} onPrev={onPrev}>
        <div>Content</div>
      </WizardShell>,
    );
    expect(screen.getAllByRole("button", { name: /^\d+\. / })[0]).toBeDisabled();
    const nextControls = screen.getAllByTestId("wizard-next");
    const previousControls = screen.getAllByTestId("wizard-prev");
    fireEvent.click(nextControls[nextControls.length - 1]!);
    fireEvent.click(previousControls[previousControls.length - 1]!);
    expect(onNext).toHaveBeenCalledTimes(1);
    expect(onPrev).toHaveBeenCalledTimes(1);
  });

  it("does not render a retry control when a failed save has no retry callback", () => {
    renderShell(1, "failed");
    expect(screen.queryByTestId("operator-state-save-retry")).not.toBeInTheDocument();
  });

  it("reports explicit save errors as alerts and measures selection indicators", () => {
    renderWithProviders(
      <WizardShell currentStep={1} steps={steps} onNext={vi.fn()} onPrev={vi.fn()} operatorStateError="Conflict from another session" operatorStateSaveState="conflict">
        <div data-rcl-selection-indicator="true">Content</div>
      </WizardShell>,
    );
    expect(screen.getByRole("alert")).toHaveTextContent("Conflict from another session");
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
