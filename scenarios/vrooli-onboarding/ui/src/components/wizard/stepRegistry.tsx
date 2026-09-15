import type { ReactNode } from "react";
import { StepApply } from "./StepApply";
import { DerivedResourceStep } from "./DerivedResourceStep";
import { HostRequirementStep } from "./HostRequirementStep";
import { StepIntegrationsDeferred } from "./StepIntegrationsDeferred";
import { StepOperatingMode } from "./StepOperatingMode";
import { StepCredentials } from "./StepCredentials";
import { StepReady } from "./StepReady";
import { ScenarioCatalogStep } from "./ScenarioCatalogStep";
import { StepCoreSet } from "./StepCoreSet";
import { StepWelcome } from "./StepWelcome";
import type { OperatorState } from "../../api/operatorstate";
import type { WizardStep } from "../../api/session";
import type { WizardProfileSession, WizardProfileSessionSaveRequest } from "../../api/session";

export interface StepRegistryProps {
  step: WizardStep;
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
  acceptRecommendation?: (profile?: string, scenarios?: string[]) => Promise<void>;
  onAdjustRecommendation?: () => void;
  profileSession?: WizardProfileSession | null;
  profileSessionBaseRevision?: string;
  onProfileSessionChange?: (draft: Omit<WizardProfileSessionSaveRequest, "expectedRevision">) => void;
  profileSessionError?: string | null;
  profileSessionSaveState?: "idle" | "saving" | "saved" | "failed" | "conflict";
  onRetryProfileSessionSave?: () => void;
}

type StepRenderer = (props: StepRegistryProps) => ReactNode;

export const stepRegistry: Record<string, StepRenderer> = {
  welcome: ({ acceptRecommendation, onAdjustRecommendation, target, profileSession, profileSessionBaseRevision, onProfileSessionChange, profileSessionError, profileSessionSaveState, onRetryProfileSessionSave }) => <StepWelcome target={target} onAccept={acceptRecommendation} onAdjust={onAdjustRecommendation} profileSession={profileSession} profileSessionBaseRevision={profileSessionBaseRevision} onProfileSessionChange={onProfileSessionChange} profileSessionError={profileSessionError} profileSessionSaveState={profileSessionSaveState} onRetryProfileSessionSave={onRetryProfileSessionSave} />,
  scenarios: ({ selectedScenarios, toggleScenario, target }) => (
    <ScenarioCatalogStep
      target={target}
      selected={selectedScenarios}
      onToggle={toggleScenario}
    />
  ),
  "core-set": ({ operatorState, setCoreSeed, target }) => (
    <StepCoreSet
      target={target}
      seed={new Set(operatorState?.core?.seed ?? [])}
      trustedBase={new Set(operatorState?.core?.trustedBase ?? [])}
      onChange={setCoreSeed}
    />
  ),
  resources: ({ selectedScenarios, operatorState, setResourceEnabled, target }) => (
    <DerivedResourceStep
      target={target}
      selected={selectedScenarios}
      operatorState={operatorState}
      onToggle={setResourceEnabled}
    />
  ),
  credentials: ({ target }) => <StepCredentials target={target} />,
  integrations: ({ target }) => <StepIntegrationsDeferred target={target} />,
  host: ({ setHostOptIn, setHostConfig, target }) => (
    <HostRequirementStep
      target={target}
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
    target,
  }) => (
    <StepOperatingMode
      target={target}
      selected={selectedScenarios}
      overrides={operatorState?.scenarios}
      onAutoRestart={setScenarioAutoRestart}
    />
  ),
  apply: ({ target }) => <StepApply target={target} />,
  validation: ({ target }) => <StepReady target={target} />,
};
