/**
 * Store barrel exports.
 */

// Pipeline store
export {
  usePipelineStore,
  // Types
  type PipelineStage,
  type PipelineRunStatus,
  type PipelineErrorInfo,
  type StatusSubscriber,
  // Selectors
  selectIsRunning,
  selectCurrentStage,
  selectProgress,
  selectStageStatus,
  selectCanResume,
  selectStoppedAfterStage,
  selectIsSubmitting,
  selectIsBusy,
  // Preflight selectors
  selectPreflightValidationOk,
  selectPreflightReadinessOk,
  selectPreflightSecretsOk,
  selectPreflightOk,
  selectMissingSecrets,
  // Stage result selectors
  selectBundleResult,
  selectPreflightResult,
  selectGenerateResult,
  selectBuildResult,
  selectSmokeTestResult,
  selectDistributionResult,
  selectStageLogs,
  // Error selectors
  selectError,
  selectErrorInfo,
  selectHasError,
  // History selectors
  selectPipelineHistory,
  selectLatestPipelineId,
  // Preflight input selectors
  selectPreflightSecrets,
  selectPreflightOverride,
} from "./pipelineStore";

// Pipeline types (for direct import if needed)
export {
  type PipelineStore,
  type PipelineStoreState,
  type PipelineStoreActions,
  type PipelineErrorCategory,
  initialPipelineState,
} from "./pipelineTypes";

// Form store
export {
  useFormStore,
  // Types
  type FormStore,
  type FormStoreState,
  type FormStoreActions,
  type AppMetadataState,
  type DeploymentState,
  type OutputState,
  type PlatformsState,
  type ConnectionState,
  type ValidationError,
  type HydrateFormData,
  // Selectors
  selectIsBundled,
  selectConnectionDecision,
  selectRequiresRemoteConfig,
  selectAllowedServerTypes,
  selectSelectedPlatformsList,
  selectStandardOutputPath,
  selectStagingPreviewPath,
  selectIconPreviewUrl,
  selectIsCustomLocation,
} from "./formStore";

// Form types (for direct import if needed)
export {
  initialFormState,
  defaultAppMetadata,
  defaultDeployment,
  defaultOutput,
  defaultPlatforms,
  defaultConnection,
} from "./formTypes";

// Sidebar store
export {
  useSidebarStore,
  selectCollapsed,
  selectActiveSection,
  SECTION_IDS,
  SECTION_METADATA,
  type SectionId,
} from "./sidebarStore";
