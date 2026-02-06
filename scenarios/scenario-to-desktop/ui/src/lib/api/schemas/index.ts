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
  StatusSchema,
  HealthResponseSchema,
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

// Export from pipeline (excluding re-exports that would conflict with common)
export {
  PipelineConfigSchema,
  type PipelineConfig,
  PipelineStageResultSchema,
  GenerateStageDetailsSchema,
  BuildStageDetailsSchema,
  BundleStageDetailsSchema,
  SmokeTestStageDetailsSchema,
  DeployArtifactResultSchema,
  DeployStageDetailsSchema,
  VerboseStageResultSchema,
  PipelineStatusSchema,
  type PipelineStatus,
  VerbosePipelineStatusSchema,
  type VerbosePipelineStatus,
  PipelineRunResponseSchema,
  type PipelineRunResponse,
  PipelineResumeResponseSchema,
  type PipelineResumeResponse,
  PipelineCancelResponseSchema,
  type PipelineCancelResponse,
  PipelineListItemSchema,
  type PipelineListItem,
  PipelineListResponseSchema,
  type PipelineListResponse,
} from "./pipeline";
