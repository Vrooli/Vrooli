import type { ReactNode } from "react";
import { StepApply } from "./StepApply";
import { StepDerivedResources } from "./StepDerivedResources";
import { StepHostRequirements } from "./StepHostRequirements";
import { StepIntegrationsDeferred } from "./StepIntegrationsDeferred";
import { StepOperatingMode } from "./StepOperatingMode";
import { StepReadiness } from "./StepReadiness";
import { StepSelectScenarios } from "./StepSelectScenarios";
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
}

type StepRenderer = (props: StepRegistryProps) => ReactNode;

export const stepRegistry: Record<string, StepRenderer> = {
  welcome: () => <StepWelcome />,
  scenarios: ({ selectedScenarios, toggleScenario }) => (
    <StepSelectScenarios
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
    <StepDerivedResources
      selected={selectedScenarios}
      operatorState={operatorState}
      onToggle={setResourceEnabled}
    />
  ),
  credentials: () => <StepReadiness title="Credentials" />,
  integrations: () => <StepIntegrationsDeferred />,
  host: ({ setHostOptIn, setHostConfig }) => (
    <StepHostRequirements
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
  validation: () => <StepReadiness title="Validation" />,
};
