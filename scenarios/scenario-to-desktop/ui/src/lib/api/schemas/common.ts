/**
 * Common/shared Zod schemas used across multiple API endpoints.
 *
 * These schemas are aligned with proto definitions in:
 * packages/proto/gen/typescript/scenario-to-desktop/v1/base/shared_pb.ts
 */

import { z } from "zod";

/**
 * Platform identifiers used throughout the application.
 * @see Platform enum in base/shared_pb.ts
 */
export const PlatformSchema = z.enum(["win", "mac", "linux"]);
export type Platform = z.infer<typeof PlatformSchema>;

/**
 * Stage names for pipeline execution.
 * @see StageName enum in base/shared_pb.ts
 */
export const StageNameSchema = z.enum([
  "preflight",
  "generate",
  "build",
  "bundle",
  "smoke_test",
  "smoketest",
  "deploy",
  "distribution",
]);
export type StageName = z.infer<typeof StageNameSchema>;

/**
 * Stage execution status.
 * @see StageStatus enum in base/shared_pb.ts
 */
export const StageStatusSchema = z.enum([
  "pending",
  "running",
  "completed",
  "failed",
  "skipped",
  "cancelled",
]);
export type StageStatus = z.infer<typeof StageStatusSchema>;

/**
 * Overall build status.
 * @see BuildStatus enum in base/shared_pb.ts
 */
export const BuildStatusSchema = z.enum(["building", "ready", "partial", "failed"]);
export type BuildStatus = z.infer<typeof BuildStatusSchema>;

/**
 * Upload status for deploy.
 * @see UploadStatus enum in base/shared_pb.ts
 */
export const UploadStatusSchema = z.enum(["pending", "uploading", "completed", "failed"]);
export type UploadStatus = z.infer<typeof UploadStatusSchema>;

/**
 * Deployment mode for the desktop application.
 * @see DeploymentMode enum in base/shared_pb.ts
 */
export const DeploymentModeSchema = z.enum(["bundled", "proxy"]);
export type DeploymentMode = z.infer<typeof DeploymentModeSchema>;

/**
 * Desktop framework.
 * @see Framework enum in base/shared_pb.ts
 */
export const FrameworkSchema = z.enum(["electron", "tauri", "neutralino"]);
export type Framework = z.infer<typeof FrameworkSchema>;

/**
 * Application template type.
 * @see TemplateType enum in base/shared_pb.ts
 */
export const TemplateTypeSchema = z.enum(["basic", "advanced", "multi-window", "kiosk"]);
export type TemplateType = z.infer<typeof TemplateTypeSchema>;

/**
 * Standard status values for various operations.
 * @deprecated Use more specific status schemas (StageStatusSchema, BuildStatusSchema, etc.)
 */
export const StatusSchema = z.enum([
  "pending",
  "running",
  "building",
  "ready",
  "completed",
  "passed",
  "failed",
  "partial",
  "skipped",
  "cancelled",
]);

/**
 * Health check response schema.
 */
export const HealthResponseSchema = z.object({
  status: z.string(),
  service: z.string(),
  timestamp: z.string(),
});

/**
 * Generic error response structure.
 */
export const ApiErrorResponseSchema = z.object({
  error: z.string(),
  code: z.string().optional(),
  details: z.record(z.unknown()).optional(),
});

/**
 * Platform selection for build targets.
 */
export const PlatformSelectionSchema = z.object({
  win: z.boolean().optional().default(false),
  mac: z.boolean().optional().default(false),
  linux: z.boolean().optional().default(false),
});

/**
 * Validation error structure used across endpoints.
 */
export const ValidationErrorSchema = z.object({
  code: z.string(),
  platform: z.string().optional(),
  field: z.string().optional(),
  service: z.string().optional(),
  path: z.string().optional(),
  message: z.string(),
  remediation: z.string().optional(),
});

/**
 * Validation warning structure.
 */
export const ValidationWarningSchema = z.object({
  code: z.string(),
  platform: z.string().optional(),
  message: z.string(),
});

/**
 * Template info schema.
 */
export const TemplateInfoSchema = z.object({
  name: z.string(),
  description: z.string(),
  type: z.string(),
  framework: z.string(),
  use_cases: z.array(z.string()),
  features: z.array(z.string()),
  complexity: z.string(),
});

/**
 * Probe response for endpoint connectivity checks.
 */
export const ProbeResponseSchema = z.object({
  server: z.object({
    status: z.enum(["ok", "error", "skipped"]),
    status_code: z.number().optional(),
    message: z.string().optional(),
  }),
  api: z.object({
    status: z.enum(["ok", "error", "skipped"]),
    status_code: z.number().optional(),
    message: z.string().optional(),
  }),
});
