/**
 * Container for all GeneratorForm modals.
 * Manages Scenario, Template, Framework, and Deployment modals.
 */

import {
  TemplateModal,
  ScenarioModal,
  FrameworkModal,
  DeploymentModal,
} from "../modals";
import type { ScenarioDesktopStatus } from "../scenario-inventory/types";
import type { DeploymentMode, ServerType } from "../../domain/deployment";
import type { DesktopFramework } from "../../domain/generator";

export interface GeneratorModalsContainerProps {
  // Modal visibility state
  modals: {
    scenario: boolean;
    template: boolean;
    framework: boolean;
    deployment: boolean;
  };
  closeModal: (
    modal: "scenario" | "template" | "framework" | "deployment",
  ) => void;
  // Scenario modal props
  loadingScenarios: boolean;
  scenarios: ScenarioDesktopStatus[];
  selectedScenarioName: string;
  onScenarioSelect: (name: string) => void;
  // Template modal props
  selectedTemplate: string;
  onTemplateSelect: (template: string) => void;
  // Framework modal props
  selectedFramework: DesktopFramework;
  onFrameworkSelect: (framework: DesktopFramework) => void;
  // Deployment modal props
  deploymentMode: DeploymentMode;
  serverType: ServerType;
  allowedServerTypes: ServerType[];
  onDeploymentChange: (mode: DeploymentMode, serverType?: ServerType) => void;
}

export function GeneratorModalsContainer({
  modals,
  closeModal,
  loadingScenarios,
  scenarios,
  selectedScenarioName,
  onScenarioSelect,
  selectedTemplate,
  onTemplateSelect,
  selectedFramework,
  onFrameworkSelect,
  deploymentMode,
  serverType,
  allowedServerTypes,
  onDeploymentChange,
}: GeneratorModalsContainerProps) {
  return (
    <>
      <ScenarioModal
        open={modals.scenario}
        loading={loadingScenarios}
        scenarios={scenarios}
        selectedScenarioName={selectedScenarioName}
        onClose={() => {
          closeModal("scenario");
        }}
        onSelect={(name) => {
          onScenarioSelect(name);
          closeModal("scenario");
        }}
      />
      <TemplateModal
        open={modals.template}
        selectedTemplate={selectedTemplate}
        onClose={() => {
          closeModal("template");
        }}
        onSelect={(template) => {
          onTemplateSelect(template);
          closeModal("template");
        }}
      />
      <FrameworkModal
        open={modals.framework}
        selectedFramework={selectedFramework}
        onClose={() => {
          closeModal("framework");
        }}
        onSelect={(nextFramework) => {
          onFrameworkSelect(nextFramework);
          closeModal("framework");
        }}
      />
      <DeploymentModal
        open={modals.deployment}
        deploymentMode={deploymentMode}
        serverType={serverType}
        allowedServerTypes={allowedServerTypes}
        onClose={() => {
          closeModal("deployment");
        }}
        onChange={(nextMode, nextServerType) => {
          onDeploymentChange(nextMode, nextServerType);
        }}
      />
    </>
  );
}
