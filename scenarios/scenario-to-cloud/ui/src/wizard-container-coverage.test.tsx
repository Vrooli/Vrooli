import "@testing-library/jest-dom";
import { fireEvent, screen } from "@testing-library/react";
import { vi } from "vitest";
import { renderWithProviders } from "./test-utils/renderWithProviders";

const state = vi.hoisted(() => ({ step: 0, empty: false }));
vi.mock("./hooks/useDeployment", () => ({
  useDeployment: () => {
    const steps = state.empty ? [] : [
      { id: "manifest", label: "Manifest", description: "Configure" },
      { id: "secrets", label: "Secrets", description: "Secrets" },
      { id: "build", label: "Build", description: "Build" },
      { id: "preflight", label: "Preflight", description: "Preflight" },
      { id: "deploy", label: "Deploy", description: "Deploy" },
    ];
    return {
      currentStepIndex: state.step,
      currentStep: steps[state.step],
      steps,
      goToStep: vi.fn(), goNext: vi.fn(), goBack: vi.fn(), reset: vi.fn(),
      canProceed: true,
      manifestJson: "{}", parsedManifest: { ok: true, value: {} },
    };
  },
}));
vi.mock("./components/wizard/StepManifest", () => ({ StepManifest: () => <div>Manifest step</div> }));
vi.mock("./components/wizard/StepSecrets", () => ({ StepSecrets: () => <div>Secrets step</div> }));
vi.mock("./components/wizard/StepBuild", () => ({ StepBuild: () => <div>Build step</div> }));
vi.mock("./components/wizard/StepPreflight", () => ({ StepPreflight: () => <div>Preflight step</div> }));
vi.mock("./components/wizard/StepDeploy", () => ({ StepDeploy: () => <div>Deploy step</div> }));

import { WizardContainer } from "./components/wizard/WizardContainer";

describe("wizard container navigation", () => {
  afterEach(() => { state.step = 0; state.empty = false; });

  it("renders every step and exposes navigation actions", () => {
    const backToDashboard = vi.fn();
    const onViewDeployments = vi.fn();
    const { rerender } = renderWithProviders(<WizardContainer onBackToDashboard={backToDashboard} onViewDeployments={onViewDeployments} />);
    for (const [index, label] of ["Manifest", "Secrets", "Build", "Preflight", "Deploy"].entries()) {
      state.step = index;
      rerender(<WizardContainer onBackToDashboard={backToDashboard} onViewDeployments={onViewDeployments} />);
      expect(screen.getByText(`${label} step`)).toBeInTheDocument();
      if (index < 4) expect(screen.getByTestId("wizard-next-button")).toBeInTheDocument();
    }
    fireEvent.click(screen.getByTestId("wizard-dashboard-button"));
    fireEvent.click(screen.getByTestId("wizard-back-button"));
    fireEvent.click(screen.getByTestId("wizard-reset-button"));
    expect(backToDashboard).toHaveBeenCalledOnce();
  });

  it("renders nothing when the deployment has no current step", () => {
    state.empty = true;
    const { container } = renderWithProviders(<WizardContainer />);
    expect(container).toBeEmptyDOMElement();
  });
});
