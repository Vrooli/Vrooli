/**
 * Zod schemas for API response validation.
 *
 * These schemas provide runtime validation at the service layer boundary.
 * They should mirror the TypeScript types in ../types.ts and be kept in sync.
 *
 * Organization:
 * - pipeline.ts: Pipeline status and stage schemas
 * - build.ts: Build status and result schemas
 * - common.ts: Shared/reusable schemas (enums and base types)
 */

// Export from common first (base enums and shared types)
export {
  PlatformSchema,
  type Platform,
  StageNameSchema,
  type StageName,
  StageStatusSchema,
  type StageStatus,
  BuildStatusSchema as BuildStatusEnumSchema,
  type BuildStatus as BuildStatusEnum,
  UploadStatusSchema,
  type UploadStatus,
  DeploymentModeSchema,
  type DeploymentMode,
  FrameworkSchema,
  type Framework,
  TemplateTypeSchema,
  type TemplateType,
  ApiErrorResponseSchema,
  PlatformSelectionSchema,
  ValidationErrorSchema,
  ValidationWarningSchema,
  TemplateInfoSchema,
  ProbeResponseSchema,
} from "./common";

// Export from build (response schemas with aliases)
export {
  PlatformBuildStatusSchema,
  type PlatformBuildStatus,
  SmokeTestStatusEnumSchema,
  type SmokeTestStatusEnum,
  PlatformBuildResultSchema,
  type PlatformBuildResult,
  BuildStatusResponseSchema,
  type BuildStatusResponse,
  // Backward compatibility: BuildStatusSchema and BuildStatus refer to the response object
  BuildStatusSchema,
  type BuildStatus,
  SmokeTestStatusResponseSchema,
  type SmokeTestStatusResponse,
  SmokeTestStatusSchema,
  type SmokeTestStatus,
  DesktopRecordSchema,
  type DesktopRecord,
  DesktopRecordResponseSchema,
} from "./build";

// Export from misc (wine, telemetry, ports, inline response types)
export {
  ProxyHintSchema,
  ProxyHintsResponseSchema,
  BundleManifestResponseSchema,
  WineInstallMethodSchema,
  WineCheckResponseSchema,
  WineInstallStatusSchema,
  TelemetryInsightSessionSchema,
  TelemetryInsightSmokeTestSchema,
  TelemetryInsightErrorSchema,
  TelemetryInsightsSchema,
  TelemetrySummarySchema,
  TelemetryEventSchema,
  TelemetryTailEntrySchema,
  TelemetryTailResponseSchema,
  ScenarioPortResponseSchema,
  MoveRecordResponseSchema,
  StatusResponseSchema,
  InstallIdResponseSchema,
  OutputPathResponseSchema,
} from "./misc";

// Export from signing (signing config, validation, readiness, certificates)
export {
  WindowsSigningConfigSchema,
  MacOSSigningConfigSchema,
  LinuxSigningConfigSchema,
  SigningConfigSchema,
  SigningConfigResponseSchema,
  PlatformValidationSchema as SigningPlatformValidationSchema,
  SigningValidationErrorSchema,
  SigningValidationWarningSchema,
  SigningValidationResultSchema,
  PlatformStatusSchema as SigningPlatformStatusSchema,
  SigningReadinessResponseSchema,
  ToolDetectionResultSchema,
  DiscoveredCertificateSchema,
  GenerateKeyResponseSchema,
  DeleteSigningResponseSchema,
  DeletePlatformSigningResponseSchema,
  PrerequisitesResponseSchema,
  CertificateDiscoveryResponseSchema,
} from "./signing";

// Export from scenarios (state management, validation, listing)
export {
  FormStateSchema,
  InputFingerprintSchema,
  StageStateSchema,
  CompressedLogSchema,
  BuildArtifactSchema,
  ScenarioStateSchema,
  ScenarioStageStatusSchema,
  StateChangeSchema,
  ValidationStatusSchema,
  LoadStateResponseSchema,
  SaveStateResponseSchema,
  CheckStalenessResponseSchema,
  GetLogsResponseSchema,
  DesktopBuildArtifactSchema,
  DesktopConnectionConfigSchema,
  ScenarioDesktopStatusSchema,
  ScenariosResponseSchema,
  TemplateListResponseSchema,
} from "./scenarios";
