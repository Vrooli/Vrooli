import { fireEvent, render, screen } from "@/test-utils";
import { describe, expect, it, vi } from "vitest";
import { GeneratorModalsContainer } from "./GeneratorModalsContainer";

vi.mock("../modals", () => ({
  ScenarioModal: ({
    open,
    onClose,
    onSelect,
  }: {
    open: boolean;
    onClose: () => void;
    onSelect: (name: string) => void;
  }) =>
    open ? (
      <>
        <button
          onClick={() => {
            onSelect("canvas-lab");
          }}
        >
          Select scenario
        </button>
        <button onClick={onClose}>Close scenario</button>
      </>
    ) : null,
  TemplateModal: ({
    open,
    onClose,
    onSelect,
  }: {
    open: boolean;
    onClose: () => void;
    onSelect: (name: string) => void;
  }) =>
    open ? (
      <>
        <button
          onClick={() => {
            onSelect("minimal");
          }}
        >
          Select template
        </button>
        <button onClick={onClose}>Close template</button>
      </>
    ) : null,
  FrameworkModal: ({
    open,
    onClose,
    onSelect,
  }: {
    open: boolean;
    onClose: () => void;
    onSelect: (name: string) => void;
  }) =>
    open ? (
      <>
        <button
          onClick={() => {
            onSelect("electron");
          }}
        >
          Select framework
        </button>
        <button onClick={onClose}>Close framework</button>
      </>
    ) : null,
  DeploymentModal: ({
    open,
    onClose,
    onChange,
  }: {
    open: boolean;
    onClose: () => void;
    onChange: (mode: string, serverType: string) => void;
  }) =>
    open ? (
      <>
        <button
          onClick={() => {
            onChange("bundled", "node");
          }}
        >
          Change deployment
        </button>
        <button onClick={onClose}>Close deployment</button>
      </>
    ) : null,
}));

describe("GeneratorModalsContainer", () => {
  it("routes selection and close actions to their owning state handlers", () => {
    const closeModal = vi.fn();
    const onScenarioSelect = vi.fn();
    const onTemplateSelect = vi.fn();
    const onFrameworkSelect = vi.fn();
    const onDeploymentChange = vi.fn();
    render(
      <GeneratorModalsContainer
        modals={{
          scenario: true,
          template: true,
          framework: true,
          deployment: true,
        }}
        closeModal={closeModal}
        loadingScenarios={false}
        scenarios={[]}
        selectedScenarioName=""
        onScenarioSelect={onScenarioSelect}
        selectedTemplate=""
        onTemplateSelect={onTemplateSelect}
        selectedFramework={"electron" as never}
        onFrameworkSelect={onFrameworkSelect}
        deploymentMode={"bundled"}
        serverType={"node"}
        allowedServerTypes={["node"]}
        onDeploymentChange={onDeploymentChange}
      />,
    );
    fireEvent.click(screen.getByRole("button", { name: "Select scenario" }));
    fireEvent.click(screen.getByRole("button", { name: "Select template" }));
    fireEvent.click(screen.getByRole("button", { name: "Select framework" }));
    fireEvent.click(screen.getByRole("button", { name: "Change deployment" }));
    fireEvent.click(screen.getByRole("button", { name: "Close deployment" }));
    expect(onScenarioSelect).toHaveBeenCalledWith("canvas-lab");
    expect(onTemplateSelect).toHaveBeenCalledWith("minimal");
    expect(onFrameworkSelect).toHaveBeenCalledWith("electron");
    expect(onDeploymentChange).toHaveBeenCalledWith("bundled", "node");
    expect(closeModal).toHaveBeenCalledWith("scenario");
    expect(closeModal).toHaveBeenCalledWith("template");
    expect(closeModal).toHaveBeenCalledWith("framework");
    expect(closeModal).toHaveBeenCalledWith("deployment");
  });
});
