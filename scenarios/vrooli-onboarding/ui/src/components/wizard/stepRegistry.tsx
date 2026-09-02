import type { ReactNode } from "react";
import { StepApply } from "./StepApply";
import { DerivedResourceStep } from "./DerivedResourceStep";
import { HostRequirementStep } from "./HostRequirementStep";
import { StepIntegrationsDeferred } from "./StepIntegrationsDeferred";
import { StepOperatingMode } from "./StepOperatingMode";
import { StepReadiness } from "./StepReadiness";
import { ScenarioCatalogStep } from "./ScenarioCatalogStep";
import { StepCoreSet } from "./StepCoreSet";
import { StepWelcome } from "./StepWelcome";
import type { OperatorState, V2Step } from "../../types";

export interface StepRegistryProps {
  step: V2Step;
  selectedScenarios: Set<string>;
  operatorState: OperatorState | null;
  toggleScenario: (name: string) => void;
  setCoreSeed: (seed: string[]) => void;
  setScenarioAutoRestart: (name: string, autoRestart: boolean) => void;
  setHostOptIn: (
    kind: "host_tools" | "host_safeguards",
    name: string,
    optedIn: boolean,
  ) => void;
  setHostConfig: (
    kind: "host_tools" | "host_safeguards",
    name: string,
    config: Record<string, unknown>,
  ) => void;
  setResourceEnabled: (name: string, enabled: boolean) => void;
  target: string;
}

type StepRenderer = (props: StepRegistryProps) => ReactNode;

export const stepRegistry: Record<string, StepRenderer> = {
  welcome: () => <StepWelcome />,
  scenarios: ({ selectedScenarios, toggleScenario }) => (
    <ScenarioCatalogStep
      selected={selectedScenarios}
      onToggle={toggleScenario}
    />
  ),
  "core-set": ({ operatorState, setCoreSeed }) => (
    <StepCoreSet
      seed={new Set(operatorState?.core?.seed ?? [])}
      trustedBase={new Set(operatorState?.core?.trusted_base ?? [])}
      onChange={setCoreSeed}
    />
  ),
  resources: ({ selectedScenarios, operatorState, setResourceEnabled }) => (
    <DerivedResourceStep
      selected={selectedScenarios}
      operatorState={operatorState}
      onToggle={setResourceEnabled}
    />
  ),
  credentials: ({ target }) => <StepReadiness title="Credentials" target={target} />,
  integrations: () => <StepIntegrationsDeferred />,
  host: ({ setHostOptIn, setHostConfig }) => (
    <HostRequirementStep
      onTool={(name, value) => setHostOptIn("host_tools", name, value)}
      onSafeguard={(name, value) =>
        setHostOptIn("host_safeguards", name, value)
      }
      onHostConfig={setHostConfig}
    />
  ),
  "operating-mode": ({
    selectedScenarios,
    operatorState,
    setScenarioAutoRestart,
  }) => (
    <StepOperatingMode
      selected={selectedScenarios}
      overrides={operatorState?.scenarios}
      onAutoRestart={setScenarioAutoRestart}
    />
  ),
  apply: () => <StepApply />,
  validation: ({ target }) => <StepReadiness title="Validation" target={target} />,
};
