import { z } from 'zod';

/**
 * Recording-related Zod schemas for API response validation.
 * These schemas provide runtime validation for recording and recovery APIs.
 */

// Browser configuration schema
export const BrowserConfigSchema = z.object({
  viewportWidth: z.number(),
  viewportHeight: z.number(),
  userAgent: z.string().optional(),
});

// Recovery checkpoint schema - represents a saved recording state
// Note: createdAt and updatedAt are required for checkpoints since they must exist
export const RecoveryCheckpointSchema = z.object({
  sessionId: z.string(),
  workflowId: z.string().optional(),
  actionCount: z.number(),
  currentUrl: z.string(),
  createdAt: z.string(),
  updatedAt: z.string(),
  browserConfig: BrowserConfigSchema,
});

// Response from /api/recording/recovery/check
export const RecoveryCheckResponseSchema = z.object({
  checkpoint: RecoveryCheckpointSchema.optional().nullable(),
});

// Response from /api/recording/recovery/{sessionId}/resume
export const RecoveryResumeResponseSchema = z.object({
  success: z.boolean(),
  sessionId: z.string().optional(),
  message: z.string().optional(),
});

// Response from DELETE /api/recording/recovery/{sessionId}
export const RecoveryDeleteResponseSchema = z.object({
  success: z.boolean(),
  message: z.string().optional(),
});

// Recording session status
export const RecordingStatusSchema = z.enum([
  'initializing',
  'ready',
  'recording',
  'paused',
  'stopped',
  'error',
]);

// Recording import response (from /recordings/import)
export const RecordingImportResponseSchema = z.object({
  execution_id: z.string().optional(),
  executionId: z.string().optional(),
  workflow_id: z.string().optional(),
  workflowId: z.string().optional(),
});

// Export types from schemas
export type BrowserConfig = z.infer<typeof BrowserConfigSchema>;
export type RecoveryCheckpoint = z.infer<typeof RecoveryCheckpointSchema>;
export type RecoveryCheckResponse = z.infer<typeof RecoveryCheckResponseSchema>;
export type RecoveryResumeResponse = z.infer<typeof RecoveryResumeResponseSchema>;
export type RecoveryDeleteResponse = z.infer<typeof RecoveryDeleteResponseSchema>;
export type RecordingStatus = z.infer<typeof RecordingStatusSchema>;
export type RecordingImportResponse = z.infer<typeof RecordingImportResponseSchema>;
