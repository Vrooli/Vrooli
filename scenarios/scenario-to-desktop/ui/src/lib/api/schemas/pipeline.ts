/**
 * Zod schemas for pipeline status and stage results.
 *
 * Aligned with proto definitions in:
 * packages/proto/gen/typescript/scenario-to-desktop/v1/pipeline/types_pb.ts
 */

import { z } from "zod";
import { PlatformBuildResultSchema } from "./build";
import {
  StageNameSchema,
  StageStatusSchema,
  DeploymentModeSchema,
  PlatformSchema,
  UploadStatusSchema,
} from "./common";

// Re-export for backward compatibility
export { StageNameSchema, StageStatusSchema };
export type StageName = z.infer<typeof StageNameSchema>;
export type StageStatus = z.infer<typeof StageStatusSchema>;

/**
 * Pipeline configuration for running a pipeline.
 * @see PipelineConfig in pipeline/types_pb.ts
 */
export const PipelineConfigSchema = z.object({
  scenario_name: z.string(),
  template_type: z.string().optional(),
  deployment_mode: DeploymentModeSchema.optional(),
  proxy_url: z.string().optional(),
  platforms: z.array(PlatformSchema).optional(),
  stop_after_stage: StageNameSchema.optional(),
  skip_stages: z.array(StageNameSchema).optional(),
  // Proto-aligned additional fields
  framework: z.string().optional(),
  webhook_url: z.string().optional(),
  bundle_manifest_path: z.string().optional(),
  clean: z.boolean().optional(),
  sign: z.boolean().optional(),
  publish: z.boolean().optional(),
  distribute: z.boolean().optional(),
  distribution_targets: z.array(z.string()).optional(),
  version: z.string().optional(),
  preflight_timeout_seconds: z.number().optional(),
  preflight_secrets: z.record(z.string()).optional(),
  resume_from_stage: StageNameSchema.optional(),
  parent_pipeline_id: z.string().optional(),
  idempotency_key: z.string().optional(),
});
export type PipelineConfig = z.infer<typeof PipelineConfigSchema>;

/**
 * Basic stage result without details.
 * Uses union type for backward compatibility with string values.
 */
export const PipelineStageResultSchema = z.object({
  status: z.union([StageStatusSchema, z.string()]),
  started_at: z.string().optional(),
  completed_at: z.string().optional(),
  error: z.string().optional(),
});

/**
 * Generate stage details.
 */
export const GenerateStageDetailsSchema = z.object({
  output_path: z.string().optional(),
  template_type: z.string().optional(),
  platforms: z.array(z.string()).optional(),
});

/**
 * Build stage details.
 */
export const BuildStageDetailsSchema = z.object({
  build_id: z.string().optional(),
  platform_results: z.record(PlatformBuildResultSchema).optional(),
  output_path: z.string().optional(),
});

/**
 * Bundle stage details.
 */
export const BundleStageDetailsSchema = z.object({
  bundle_path: z.string().optional(),
  bundle_size: z.number().optional(),
});

/**
 * Smoke test stage details.
 * @see SmokeTestStatus in domain/build_pb.ts
 */
export const SmokeTestStageDetailsSchema = z.object({
  smoke_test_id: z.string().optional(),
  platform: PlatformSchema.optional(),
  status: z.enum(["running", "passed", "failed"]).optional(),
  started_at: z.string().optional(),
  completed_at: z.string().optional(),
  logs: z.array(z.string()).optional(),
  error: z.string().optional(),
  telemetry_uploaded: z.boolean().optional(),
});

/**
 * Distribution upload status.
 * @see PlatformUpload in distribution/types_pb.ts
 *
 * Most fields are optional for backward compatibility with existing tests.
 */
export const DistributionPlatformUploadSchema = z.object({
  platform: z.union([PlatformSchema, z.string()]).optional(),
  target: z.string().optional(),
  status: z.union([UploadStatusSchema, z.string()]),
  url: z.string().optional(),
  error: z.string().optional(),
});

/**
 * Distribution stage details.
 */
export const DistributionStageDetailsSchema = z.object({
  targets: z.array(z.string()).optional(),
  uploads: z.array(DistributionPlatformUploadSchema).optional(),
});

/**
 * Verbose stage result with details.
 * Uses union type for backward compatibility with string values.
 */
export const VerboseStageResultSchema = z.object({
  status: z.union([StageStatusSchema, z.string()]),
  started_at: z.string().optional(),
  completed_at: z.string().optional(),
  error: z.string().optional(),
  details: z
    .union([
      GenerateStageDetailsSchema,
      BuildStageDetailsSchema,
      BundleStageDetailsSchema,
      SmokeTestStageDetailsSchema,
      DistributionStageDetailsSchema,
      z.record(z.unknown()),
    ])
    .optional(),
});

/**
 * Overall pipeline status.
 * @see PipelineStatus in pipeline/types_pb.ts
 *
 * Most fields are optional for backward compatibility with existing tests.
 */
export const PipelineStatusSchema = z.object({
  pipeline_id: z.string(),
  scenario_name: z.string().optional(),
  status: z.union([StageStatusSchema, z.string()]),
  current_stage: z.union([StageNameSchema, z.string()]).optional(),
  progress_percent: z.number().optional(),
  progress_message: z.string().optional(),
  stages: z.record(PipelineStageResultSchema).optional(),
  stage_order: z.array(z.union([StageNameSchema, z.string()])).optional(),
  config: PipelineConfigSchema.optional(),
  created_at: z.string().optional(),
  started_at: z.string().optional(),
  updated_at: z.string().optional(),
  completed_at: z.string().optional(),
  error: z.string().optional(),
  final_artifacts: z.record(z.string()).optional(),
  stopped_after_stage: z.union([StageNameSchema, z.string()]).optional(),
  parent_pipeline_id: z.string().optional(),
  idempotency_key: z.string().optional(),
});
export type PipelineStatus = z.infer<typeof PipelineStatusSchema>;

/**
 * Verbose pipeline status with stage details.
 * @see PipelineStatus in pipeline/types_pb.ts
 *
 * Most fields are optional for backward compatibility with existing tests.
 */
export const VerbosePipelineStatusSchema = z.object({
  pipeline_id: z.string(),
  scenario_name: z.string().optional(),
  status: z.union([StageStatusSchema, z.string()]),
  current_stage: z.union([StageNameSchema, z.string()]).optional(),
  progress_percent: z.number().optional(),
  progress_message: z.string().optional(),
  stages: z.record(VerboseStageResultSchema).optional(),
  stage_order: z.array(z.union([StageNameSchema, z.string()])).optional(),
  config: PipelineConfigSchema.optional(),
  created_at: z.string().optional(),
  started_at: z.string().optional(),
  updated_at: z.string().optional(),
  completed_at: z.string().optional(),
  error: z.string().optional(),
  final_artifacts: z.record(z.string()).optional(),
  stopped_after_stage: z.union([StageNameSchema, z.string()]).optional(),
  parent_pipeline_id: z.string().optional(),
  idempotency_key: z.string().optional(),
});
export type VerbosePipelineStatus = z.infer<typeof VerbosePipelineStatusSchema>;

/**
 * Pipeline run response.
 * @see PipelineRunResponse in pipeline/types_pb.ts
 */
export const PipelineRunResponseSchema = z.object({
  pipeline_id: z.string(),
  status_url: z.string().optional(),
  message: z.string().optional(),
});
export type PipelineRunResponse = z.infer<typeof PipelineRunResponseSchema>;

/**
 * Pipeline resume response.
 * @see PipelineResumeResponse in pipeline/types_pb.ts
 */
export const PipelineResumeResponseSchema = z.object({
  pipeline_id: z.string(),
  parent_pipeline_id: z.string(),
  status_url: z.string(),
  resume_from_stage: StageNameSchema,
  message: z.string().optional(),
});
export type PipelineResumeResponse = z.infer<typeof PipelineResumeResponseSchema>;

/**
 * Pipeline cancel response.
 * @see PipelineCancelResponse in pipeline/types_pb.ts
 */
export const PipelineCancelResponseSchema = z.object({
  status: z.string(),
  message: z.string().optional(),
});
export type PipelineCancelResponse = z.infer<typeof PipelineCancelResponseSchema>;

/**
 * Pipeline list item.
 * @see PipelineListItem in pipeline/types_pb.ts
 *
 * Uses union types for backward compatibility.
 */
export const PipelineListItemSchema = z.object({
  pipeline_id: z.string(),
  scenario_name: z.string().optional(),
  status: z.union([StageStatusSchema, z.string()]),
  progress_percent: z.number().optional(),
  current_stage: z.union([StageNameSchema, z.string()]).optional(),
  created_at: z.string().optional(),
  updated_at: z.string().optional(),
  completed_at: z.string().optional(),
  can_resume: z.boolean().optional(),
});
export type PipelineListItem = z.infer<typeof PipelineListItemSchema>;

/**
 * Pipeline list response.
 * @see PipelineListResponse in pipeline/types_pb.ts
 */
export const PipelineListResponseSchema = z.object({
  pipelines: z.array(PipelineListItemSchema),
  total: z.number().optional(),
});
export type PipelineListResponse = z.infer<typeof PipelineListResponseSchema>;
