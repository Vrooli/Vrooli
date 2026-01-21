/**
 * Custom hooks for scenario-to-desktop UI.
 */

export {
  useScenarioState,
  type UseScenarioStateOptions,
  type UseScenarioStateResult,
} from "./useScenarioState";

export {
  useAgentManagerStatus,
  useTasks,
  useTaskDetails,
  useCreateTask,
  useStopTask,
  usePipelineInvestigation,
} from "./useInvestigation";
export { useSigningConfig, type UseSigningConfigOptions, type UseSigningConfigResult } from "./useSigningConfig";
export { useUrlState, parseSearchParams, type ViewMode } from "./useUrlState";
export {
  usePipelineMutation,
  usePipelineStatus,
  usePlatformSelection,
  useWineCheck,
} from "./usePipelineButton";
export { useGeneratorModals, type ModalStates, type UseGeneratorModalsReturn } from "./useGeneratorModals";
export {
  useGeneratorFormState,
  type AppMetadata,
  type DeploymentState,
  type OutputState,
  type ConnectionState,
  type UseGeneratorFormStateReturn,
} from "./useGeneratorFormState";
