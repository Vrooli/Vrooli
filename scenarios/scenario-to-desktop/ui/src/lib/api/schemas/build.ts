/**
 * Zod schemas for build status and result types.
 *
 * Aligned with proto definitions in:
 * packages/proto/gen/typescript/scenario-to-desktop/v1/domain/build_pb.ts
 */

import { z } from "zod";
import {
  PlatformSchema,
  BuildStatusSchema as BuildStatusEnumSchema,
  FrameworkSchema,
  TemplateTypeSchema,
} from "./common";

/**
 * Platform build status values.
 * @see PlatformBuildStatus enum in domain/build_pb.ts
 */
export const PlatformBuildStatusSchema = z.enum([
  "pending",
  "building",
  "ready",
  "failed",
  "skipped",
]);
export type PlatformBuildStatus = z.infer<typeof PlatformBuildStatusSchema>;

/**
 * Smoke test status values.
 * @see SmokeTestStatus enum in domain/build_pb.ts
 */
export const SmokeTestStatusEnumSchema = z.enum([
  "running",
  "passed",
  "failed",
]);
export type SmokeTestStatusEnum = z.infer<typeof SmokeTestStatusEnumSchema>;

/**
 * Individual platform build result.
 * @see PlatformBuildResult in domain/build_pb.ts
 *
 * Uses union types for backward compatibility with string values.
 */
export const PlatformBuildResultSchema = z.object({
  platform: z.union([PlatformSchema, z.string()]),
  status: z.union([PlatformBuildStatusSchema, z.string()]),
  started_at: z.string().optional(),
  completed_at: z.string().optional(),
  error_log: z.array(z.string()).optional(),
  artifact: z.string().optional(),
  file_size: z.number().optional(),
  skip_reason: z.string().optional(),
});
export type PlatformBuildResult = z.infer<typeof PlatformBuildResultSchema>;

/**
 * Overall build status response.
 * @see BuildStatusResponse in domain/build_pb.ts
 *
 * Uses union types for backward compatibility with string values.
 */
export const BuildStatusResponseSchema = z.object({
  build_id: z.string(),
  scenario_name: z.string(),
  status: z.union([BuildStatusEnumSchema, z.string()]),
  framework: z.union([FrameworkSchema, z.string()]).optional(),
  template_type: z.union([TemplateTypeSchema, z.string()]).optional(),
  platforms: z.array(z.union([PlatformSchema, z.string()])),
  requested_platforms: z
    .array(z.union([PlatformSchema, z.string()]))
    .optional(),
  platform_results: z.record(PlatformBuildResultSchema).optional(),
  output_path: z.string(),
  created_at: z.string(),
  completed_at: z.string().optional(),
  error_log: z.array(z.string()).optional(),
  build_log: z.array(z.string()).optional(),
  artifacts: z.record(z.string()).optional(),
});
export type BuildStatusResponse = z.infer<typeof BuildStatusResponseSchema>;

// Backward compatibility alias
export const BuildStatusSchema = BuildStatusResponseSchema;
export type BuildStatus = BuildStatusResponse;

/**
 * Smoke test status response.
 * @see SmokeTestStatusResponse in domain/build_pb.ts
 *
 * Uses union types for backward compatibility with string values.
 */
export const SmokeTestStatusResponseSchema = z.object({
  smoke_test_id: z.string(),
  scenario_name: z.string(),
  platform: z.union([PlatformSchema, z.string()]),
  status: z.union([SmokeTestStatusEnumSchema, z.string()]),
  artifact_path: z.string().optional(),
  started_at: z.string(),
  completed_at: z.string().optional(),
  logs: z.array(z.string()).optional(),
  error: z.string().optional(),
  telemetry_uploaded: z.boolean().optional(),
  telemetry_upload_error: z.string().optional(),
});
export type SmokeTestStatusResponse = z.infer<
  typeof SmokeTestStatusResponseSchema
>;

// Backward compatibility alias
export const SmokeTestStatusSchema = SmokeTestStatusResponseSchema;
export type SmokeTestStatus = SmokeTestStatusResponse;

/**
 * Desktop record for generated applications.
 */
export const DesktopRecordSchema = z.object({
  id: z.string(),
  build_id: z.string(),
  scenario_name: z.string(),
  app_display_name: z.string().optional(),
  template_type: z.string().optional(),
  framework: z.string().optional(),
  location_mode: z.string().optional(),
  output_path: z.string(),
  destination_path: z.string().optional(),
  staging_path: z.string().optional(),
  custom_path: z.string().optional(),
  deployment_mode: z.string().optional(),
  icon: z.string().optional(),
  created_at: z.string().optional(),
  updated_at: z.string().optional(),
});
export type DesktopRecord = z.infer<typeof DesktopRecordSchema>;

/**
 * Desktop records response with build status.
 */
export const ScreenRecordingViewSchema = z.object({
  recorded: z.boolean(),
  duration_ms: z.number().optional(),
  file_size_bytes: z.number().optional(),
  error: z.string().optional(),
});

export const DesktopRecordResponseSchema = z.object({
  records: z.array(
    z.object({
      record: DesktopRecordSchema,
      build_status: BuildStatusSchema.optional(),
      has_build: z.boolean(),
      build_state: z.string().optional(),
      smoke_test_id: z.string().optional(),
      screen_recording: ScreenRecordingViewSchema.optional(),
    }),
  ),
});
