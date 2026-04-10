import { z } from 'zod';

/**
 * Common Zod schemas for shared types used across API responses.
 * These provide runtime validation for API data.
 */

// Timestamp schema - accepts ISO 8601 strings
export const TimestampSchema = z.string().datetime({ offset: true }).optional();

// Flexible timestamp that accepts any string (for API responses that may not be ISO 8601)
export const FlexibleTimestampSchema = z.string().optional();

// Metadata schema - Record<string, unknown> with specific handling
export const MetadataSchema = z.record(z.string(), z.unknown()).optional();

// Severity enum for validation issues
export const SeveritySchema = z.enum(['error', 'warning']);

// Navigation wait condition enum
export const NavigationWaitUntilSchema = z.enum(['domcontentloaded', 'networkidle', 'load']);

// Execution status enum
export const ExecutionStatusSchema = z.enum([
  'pending',
  'running',
  'completed',
  'failed',
  'cancelled',
  'paused',
]);

// Generic success response
export const SuccessResponseSchema = z.object({
  success: z.boolean().optional(),
  updated_at: FlexibleTimestampSchema,
});

// Generic error response
export const ErrorResponseSchema = z.object({
  error: z.string(),
  code: z.string().optional(),
  details: z.unknown().optional(),
});

// Pagination schema for list responses
export const PaginationSchema = z.object({
  page: z.number().optional(),
  page_size: z.number().optional(),
  total_items: z.number().optional(),
  total_pages: z.number().optional(),
});

export type Timestamp = z.infer<typeof TimestampSchema>;
export type Metadata = z.infer<typeof MetadataSchema>;
export type Severity = z.infer<typeof SeveritySchema>;
export type NavigationWaitUntil = z.infer<typeof NavigationWaitUntilSchema>;
export type ExecutionStatus = z.infer<typeof ExecutionStatusSchema>;
export type SuccessResponse = z.infer<typeof SuccessResponseSchema>;
export type ErrorResponse = z.infer<typeof ErrorResponseSchema>;
export type Pagination = z.infer<typeof PaginationSchema>;
