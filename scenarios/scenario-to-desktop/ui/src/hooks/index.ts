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
  useGeneratorPage,
  type UseGeneratorPageProps,
  type UseGeneratorPageReturn,
} from "./useGeneratorPage";
export {
  usePreflightSection,
  type UsePreflightSectionProps,
  type UsePreflightSectionReturn,
  type PreflightStepStatuses,
} from "./usePreflightSection";
export {
  useSigningPage,
  type UseSigningPageProps,
  type UseSigningPageReturn,
} from "./useSigningPage";

// Responsive breakpoint hooks
export { useMediaQuery, useIsMobile, MOBILE_QUERY } from "./useMediaQuery";

// New micro-hooks for decomposed architecture
export {
  useFormState,
  type UseFormStateProps,
  type UseFormStateReturn,
} from "./useFormState";
export {
  usePipelineActions,
  type UsePipelineActionsProps,
  type UsePipelineActionsReturn,
} from "./usePipelineActions";
export {
  useScenarioSync,
  type UseScenarioSyncProps,
  type UseScenarioSyncReturn,
  type PreflightSeed,
} from "./useScenarioSync";
