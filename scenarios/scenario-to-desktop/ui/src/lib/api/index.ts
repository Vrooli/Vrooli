// Re-export everything from each module for backward compatibility
// This allows existing imports from "../lib/api" to continue working

// Client (error handling)
export { ApiError, throwIfNotOk, buildUrl } from "./client";
export type { RecoveryAction, ApiErrorResponse } from "./client";

// Types (all type definitions)
export type {
  DocsDocument,
  DocsSection,
  DocsNavigation,
  DocsManifest,
  DocsContentResponse,
  HealthResponse,
  TemplateInfo,
  DesktopConfig,
  BundleValidationError,
  BundleValidationWarning,
  BundleMissingBinary,
  BundleMissingAsset,
  BundleInvalidChecksum,
  BundleValidationResult,
  BundlePreflightRequest,
  BundlePreflightSecret,
  BundlePreflightReady,
  BundlePreflightTelemetry,
  BundlePreflightRuntime,
  BundlePreflightServiceFingerprint,
  BundlePreflightLogTail,
  BundlePreflightCheck,
  BundlePreflightResponse,
  BundlePreflightStep,
  BundleManifestResponse,
  PlatformBuildResult,
  BuildStatus,
  SmokeTestStatus,
  ProbeResponse,
  ProxyHintsResponse,
  DesktopRecord,
  ScreenRecordingView,
  DesktopRecordResponse,
  WineInstallMethod,
  WineCheckResponse,
  WineInstallStatus,
  TelemetryUploadRequest,
  ScenarioPortResponse,
  SigningConfig,
  WindowsSigningConfig,
  MacOSSigningConfig,
  LinuxSigningConfig,
  SigningConfigResponse,
  SigningValidationError,
  ValidationWarning,
  PlatformValidation,
  SigningValidationResult,
  PlatformStatus,
  SigningReadinessResponse,
  ToolDetectionResult,
  DiscoveredCertificate,
  GenerateKeyResponse,
} from "./types";

// Pipeline types and functions
export type {
  PipelineConfig,
  PipelineStageResult,
  PipelineStatus,
  PipelineRunResponse,
  PipelineResumeResponse,
  BundleStageDetails,
  GenerateStageDetails,
  BuildPlatformResult,
  BuildStageDetails,
  SmokeTestStageDetails,
  DeployStageDetails,
  StageDetails,
  VerboseStageResult,
  VerbosePipelineStatus,
  GetPipelineStatusOptions,
  // Scenario-based pipeline management types
  ActivePipelineResponse,
  CreatePipelineResponse,
  ResetPipelineResponse,
  PipelineHistoryResponse,
  GetActivePipelineOptions,
  GetPipelineHistoryOptions,
  StartActivePipelineResponse,
} from "./pipeline";
export {
  runPipeline,
  getPipelineStatus,
  resumePipeline,
  cancelPipeline,
  listPipelines,
  runPreflightPipeline,
  extractPreflightResult,
  // Scenario-based pipeline management functions
  getActivePipeline,
  createNewPipeline,
  resetPipeline,
  getPipelineHistory,
  startActivePipeline,
} from "./pipeline";

// Scenario state types and functions
export type {
  PlatformSelection,
  FormState,
  InputFingerprint,
  StageState,
  CompressedLog,
  BuildArtifact,
  ScenarioState,
  StateChange,
  ScenarioStageStatus,
  StageStatus as ScenarioStageStatusLegacy,
  ValidationStatus,
  LoadStateResponse,
  SaveStateResponse,
  CheckStalenessResponse,
  GetLogsResponse,
  LoadStateOptions,
  SaveStateOptions,
} from "./scenarios";
export {
  fetchScenarioState,
  saveScenarioState,
  deleteScenarioState,
  checkStateStaleness,
  getScenarioLogs,
  invalidateScenarioStage,
  fetchScenarioDesktopStatus,
  fetchTemplates,
} from "./scenarios";

// Signing functions
export {
  fetchSigningConfig,
  saveSigningConfig,
  updatePlatformSigningConfig,
  deleteSigningConfig,
  deletePlatformSigningConfig,
  validateSigningConfig,
  checkSigningReadiness,
  fetchSigningPrerequisites,
  discoverCertificates,
  generateLinuxSigningKey,
} from "./signing";

// Task functions
export {
  getAgentManagerStatus,
  createTask,
  listTasks,
  getTask,
  stopTask,
} from "./tasks";

// Misc functions
export {
  getIconPreviewUrl,
  fetchHealth,
  fetchDocsManifest,
  fetchDocContent,
  fetchDesktopRecords,
  moveDesktopRecord,
  getDownloadUrl,
  deleteDesktopBuild,
  probeEndpoints,
  fetchProxyHints,
  fetchBundleManifest,
  checkWineStatus,
  startWineInstall,
  fetchWineInstallStatus,
  fetchTelemetryInsights,
  uploadTelemetry,
  deleteTelemetry,
  fetchTelemetrySummary,
  fetchTelemetryTail,
  getTelemetryDownloadUrl,
  fetchScenarioPort,
} from "./misc";

// Safe parsing utilities for runtime validation
export {
  safeParse,
  safeParseWithDefault,
  parseOrThrow,
  ValidationError,
  isValidationError,
  z,
} from "./safeParse";
export type { ParseResult, ZodSchema, ZodError } from "./safeParse";

// Zod schemas for runtime validation
export * from "./schemas";

// Proto parsing utilities
export {
  parseProtoStrict,
  parseProtoSafe,
  protoMessageToJson,
  timestampToDate,
  dateToTimestamp,
} from "./proto";

// Proto enum adapters for backward compatibility
export * from "./protoAdapters";

// ============================================================================
// Proto type re-exports for direct access to generated types
// ============================================================================

// Base shared types (enums)
export {
  Platform,
  StageName,
  StageStatus,
  BuildStatus as ProtoBuildStatus,
  UploadStatus as ProtoUploadStatus,
  DeploymentMode,
  Framework,
  TemplateType,
  // Schemas for validation
  PlatformSchema as ProtoPlatformSchema,
  StageNameSchema as ProtoStageNameSchema,
  StageStatusSchema as ProtoStageStatusSchema,
  BuildStatusSchema as ProtoBuildStatusSchema,
  UploadStatusSchema as ProtoUploadStatusSchema,
  DeploymentModeSchema as ProtoDeploymentModeSchema,
  FrameworkSchema as ProtoFrameworkSchema,
  TemplateTypeSchema as ProtoTemplateTypeSchema,
} from "@vrooli/proto-types/scenario-to-desktop/v1/base/shared_pb";

// Pipeline types
export type {
  PipelineConfig as ProtoPipelineConfig,
  PipelineStatus as ProtoPipelineStatus,
  StageResult as ProtoStageResult,
  PipelineRunRequest as ProtoPipelineRunRequest,
  PipelineRunResponse as ProtoPipelineRunResponse,
  PipelineCancelResponse as ProtoPipelineCancelResponse,
  PipelineResumeResponse as ProtoPipelineResumeResponse,
  PipelineListItem as ProtoPipelineListItem,
  PipelineListResponse as ProtoPipelineListResponse,
  GenerateResponse as ProtoGenerateResponse,
} from "@vrooli/proto-types/scenario-to-desktop/v1/pipeline/types_pb";

export {
  PipelineConfigSchema as ProtoPipelineConfigSchema,
  PipelineStatusSchema as ProtoPipelineStatusSchema,
  StageResultSchema as ProtoStageResultSchema,
  PipelineRunRequestSchema as ProtoPipelineRunRequestSchema,
  PipelineRunResponseSchema as ProtoPipelineRunResponseSchema,
  PipelineCancelResponseSchema as ProtoPipelineCancelResponseSchema,
  PipelineResumeResponseSchema as ProtoPipelineResumeResponseSchema,
  PipelineListItemSchema as ProtoPipelineListItemSchema,
  PipelineListResponseSchema as ProtoPipelineListResponseSchema,
  GenerateResponseSchema as ProtoGenerateResponseSchema,
} from "@vrooli/proto-types/scenario-to-desktop/v1/pipeline/types_pb";

// Build domain types
export type {
  PlatformBuildResult as ProtoPlatformBuildResult,
  BuildStatusResponse as ProtoBuildStatusResponse,
  SmokeTestStatusResponse as ProtoSmokeTestStatusResponse,
} from "@vrooli/proto-types/scenario-to-desktop/v1/domain/build_pb";

export {
  PlatformBuildStatus,
  SmokeTestStatus as ProtoSmokeTestStatus,
  PlatformBuildResultSchema as ProtoPlatformBuildResultSchema,
  BuildStatusResponseSchema as ProtoBuildStatusResponseSchema,
  SmokeTestStatusResponseSchema as ProtoSmokeTestStatusResponseSchema,
  PlatformBuildStatusSchema as ProtoPlatformBuildStatusSchema,
  SmokeTestStatusSchema as ProtoSmokeTestStatusSchema,
} from "@vrooli/proto-types/scenario-to-desktop/v1/domain/build_pb";

// Signing domain types
export type {
  SigningConfig as ProtoSigningConfig,
  WindowsSigningConfig as ProtoWindowsSigningConfig,
  MacOSSigningConfig as ProtoMacOSSigningConfig,
  LinuxSigningConfig as ProtoLinuxSigningConfig,
  SigningValidationResult as ProtoSigningValidationResult,
  PlatformValidation as ProtoPlatformValidation,
  CertificateInfo as ProtoCertificateInfo,
  PlatformStatus as ProtoPlatformStatus,
  ReadinessResponse as ProtoReadinessResponse,
  SigningConfigResponse as ProtoSigningConfigResponse,
} from "@vrooli/proto-types/scenario-to-desktop/v1/domain/signing_pb";

export {
  CertificateSource,
  SignAlgorithm,
  SigningConfigSchema as ProtoSigningConfigSchema,
  WindowsSigningConfigSchema as ProtoWindowsSigningConfigSchema,
  MacOSSigningConfigSchema as ProtoMacOSSigningConfigSchema,
  LinuxSigningConfigSchema as ProtoLinuxSigningConfigSchema,
  SigningValidationResultSchema as ProtoSigningValidationResultSchema,
  PlatformValidationSchema as ProtoPlatformValidationSchema,
  CertificateInfoSchema as ProtoCertificateInfoSchema,
  PlatformStatusSchema as ProtoPlatformStatusSchema,
  ReadinessResponseSchema as ProtoReadinessResponseSchema,
  SigningConfigResponseSchema as ProtoSigningConfigResponseSchema,
  CertificateSourceSchema,
  SignAlgorithmSchema,
} from "@vrooli/proto-types/scenario-to-desktop/v1/domain/signing_pb";

// Preflight domain types
export type {
  PreflightRequest as ProtoPreflightRequest,
  PreflightResponse as ProtoPreflightResponse,
  PreflightSecret as ProtoPreflightSecret,
  PreflightReady as ProtoPreflightReady,
  PreflightCheck as ProtoPreflightCheck,
  PreflightRuntime as ProtoPreflightRuntime,
  ServiceLogTail as ProtoServiceLogTail,
  ServiceFingerprint as ProtoServiceFingerprint,
  TelemetryInfo as ProtoTelemetryInfo,
  GPUInfo as ProtoGPUInfo,
  JobStep as ProtoJobStep,
  JobStartResponse as ProtoJobStartResponse,
  JobStatusResponse as ProtoJobStatusResponse,
  ManifestRequest as ProtoManifestRequest,
  ManifestResponse as ProtoManifestResponse,
} from "@vrooli/proto-types/scenario-to-desktop/v1/domain/preflight_pb";

export {
  PreflightStatus,
  CheckStatus,
  SecretClass,
  JobStatus,
  PreflightRequestSchema as ProtoPreflightRequestSchema,
  PreflightResponseSchema as ProtoPreflightResponseSchema,
  PreflightSecretSchema as ProtoPreflightSecretSchema,
  PreflightReadySchema as ProtoPreflightReadySchema,
  PreflightCheckSchema as ProtoPreflightCheckSchema,
  PreflightRuntimeSchema as ProtoPreflightRuntimeSchema,
  ServiceLogTailSchema as ProtoServiceLogTailSchema,
  ServiceFingerprintSchema as ProtoServiceFingerprintSchema,
  TelemetryInfoSchema as ProtoTelemetryInfoSchema,
  GPUInfoSchema as ProtoGPUInfoSchema,
  JobStepSchema as ProtoJobStepSchema,
  JobStartResponseSchema as ProtoJobStartResponseSchema,
  JobStatusResponseSchema as ProtoJobStatusResponseSchema,
  ManifestRequestSchema as ProtoManifestRequestSchema,
  ManifestResponseSchema as ProtoManifestResponseSchema,
  PreflightStatusSchema as ProtoPreflightStatusSchema,
  CheckStatusSchema as ProtoCheckStatusSchema,
  SecretClassSchema as ProtoSecretClassSchema,
  JobStatusSchema as ProtoJobStatusSchema,
} from "@vrooli/proto-types/scenario-to-desktop/v1/domain/preflight_pb";
