import { z } from 'zod';

/**
 * Export-related Zod schemas for API response validation.
 * These schemas provide runtime validation for export operations.
 */

// Export format enum
export const ExportFormatSchema = z.enum(['mp4', 'gif', 'json', 'html']);

// Render source enum
export const RenderSourceSchema = z.enum(['auto', 'recorded_video', 'replay_frames']);

// Export status enum
export const ExportStatusEnumSchema = z.enum(['pending', 'processing', 'completed', 'failed']);

// Export stage enum
export const ExportStageSchema = z.enum([
  'preparing',
  'capturing',
  'encoding',
  'finalizing',
  'completed',
  'failed',
]);

// Server export response schema (from POST /export)
export const ServerExportResponseSchema = z.object({
  export_id: z.string(),
  execution_id: z.string(),
  status: z.literal('processing'),
  message: z.string().optional(),
});

// Export status response schema (from GET /exports/:id/status)
export const ExportStatusResponseSchema = z.object({
  export_id: z.string(),
  execution_id: z.string(),
  status: ExportStatusEnumSchema,
  format: z.string(),
  name: z.string(),
  storage_url: z.string().optional(),
  file_size_bytes: z.number().optional(),
  error: z.string().optional(),
});

// Export progress schema (from WebSocket)
export const ExportProgressSchema = z.object({
  export_id: z.string(),
  execution_id: z.string(),
  stage: ExportStageSchema,
  progress_percent: z.number(),
  status: z.enum(['processing', 'completed', 'failed']),
  storage_url: z.string().optional(),
  file_size_bytes: z.number().optional(),
  error: z.string().optional(),
  timestamp: z.string().optional(),
});

// Export types
export type ExportFormat = z.infer<typeof ExportFormatSchema>;
export type RenderSource = z.infer<typeof RenderSourceSchema>;
export type ExportStatusEnum = z.infer<typeof ExportStatusEnumSchema>;
export type ExportStage = z.infer<typeof ExportStageSchema>;
export type ServerExportResponse = z.infer<typeof ServerExportResponseSchema>;
export type ExportStatusResponse = z.infer<typeof ExportStatusResponseSchema>;
export type ExportProgress = z.infer<typeof ExportProgressSchema>;
