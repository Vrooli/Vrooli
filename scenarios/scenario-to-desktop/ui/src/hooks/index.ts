/**
 * Custom hooks for scenario-to-desktop UI.
 */

export {
  useScenarioState,
  type UseScenarioStateOptions,
  type UseScenarioStateResult,
} from "./useScenarioState";

export {
  useGeneratorFormState,
  type GeneratorFormState,
  type FormAction,
  type UseGeneratorFormStateResult,
  type ConnectionConfigPayload,
  DEFAULT_FORM_STATE,
} from "./useGeneratorFormState";

// Legacy type export for backwards compatibility
export type GeneratorDraftState = {
  selectedTemplate: string;
  appDisplayName: string;
  appDescription: string;
  iconPath: string;
  displayNameEdited: boolean;
  descriptionEdited: boolean;
  iconPathEdited: boolean;
  framework: string;
  serverType: string;
  deploymentMode: string;
  platforms: { win: boolean; mac: boolean; linux: boolean };
  locationMode: string;
  outputPath: string;
  proxyUrl: string;
  bundleManifestPath: string;
  serverPort: number;
  localServerPath: string;
  localApiEndpoint: string;
  autoManageTier1: boolean;
  vrooliBinaryPath: string;
  connectionResult: unknown;
  connectionError: string | null;
  preflightResult: unknown;
  preflightError: string | null;
  preflightOverride: boolean;
  preflightSecrets: Record<string, string>;
  preflightStartServices: boolean;
  preflightAutoRefresh: boolean;
  preflightSessionId: string | null;
  preflightSessionExpiresAt: string | null;
  preflightSessionTTL: number;
  deploymentManagerUrl: string | null;
  signingEnabledForBuild: boolean;
};

export type DraftTimestamps = {
  createdAt: string;
  updatedAt: string;
};
export {
  useAgentManagerStatus,
  useTasks,
  useTaskDetails,
  useCreateTask,
  useStopTask,
  usePipelineInvestigation,
} from "./useInvestigation";
export { usePreflightSession, type UsePreflightSessionOptions, type UsePreflightSessionResult } from "./usePreflightSession";
export { useSigningConfig, type UseSigningConfigOptions, type UseSigningConfigResult } from "./useSigningConfig";
