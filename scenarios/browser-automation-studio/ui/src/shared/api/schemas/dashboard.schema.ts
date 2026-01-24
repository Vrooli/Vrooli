/**
 * Dashboard API Response Schemas
 *
 * Zod schemas for validating dashboard-related API responses.
 * Used by dashboardStore.ts for runtime validation of workflows and executions.
 */

import { z } from 'zod';

// =============================================================================
// Workflow Schemas
// =============================================================================

/**
 * Schema for a single workflow in the list response.
 * Handles both snake_case (API) and camelCase (legacy) field names.
 */
export const WorkflowItemSchema = z.object({
  id: z.string(),
  name: z.string().optional(),
  project_id: z.string().optional(),
  projectId: z.string().optional(),
  updated_at: z.string().optional(),
  updatedAt: z.string().optional(),
  folder_path: z.string().optional(),
  folderPath: z.string().optional(),
});

/**
 * Schema for the workflows list API response.
 */
export const WorkflowsListResponseSchema = z.object({
  workflows: z.array(WorkflowItemSchema),
});

// =============================================================================
// Execution Schemas
// =============================================================================

/**
 * Valid execution status values (normalized to lowercase).
 * Named differently to avoid conflict with common.schema ExecutionStatusSchema.
 */
export const DashboardExecutionStatusSchema = z.enum([
  'pending',
  'running',
  'completed',
  'failed',
  'cancelled',
]);

/**
 * Schema for a single execution in the list response.
 * Handles both snake_case (API) and camelCase (legacy) field names.
 * Also handles proto enum format (e.g., "EXECUTION_STATUS_RUNNING").
 */
export const ExecutionItemSchema = z.object({
  id: z.string().optional(),
  executionId: z.string().optional(),
  execution_id: z.string().optional(),
  workflow_id: z.string().optional(),
  workflowId: z.string().optional(),
  status: z.union([z.string(), z.number()]).optional(),
  started_at: z.string().optional(),
  startedAt: z.string().optional(),
  completed_at: z.string().optional().nullable(),
  completedAt: z.string().optional().nullable(),
  error: z.string().optional().nullable(),
}).passthrough(); // Allow additional fields

/**
 * Schema for the executions list API response.
 */
export const ExecutionsListResponseSchema = z.object({
  executions: z.array(ExecutionItemSchema),
});

// =============================================================================
// Type Exports
// =============================================================================

export type WorkflowItem = z.infer<typeof WorkflowItemSchema>;
export type WorkflowsListResponse = z.infer<typeof WorkflowsListResponseSchema>;
export type ExecutionItem = z.infer<typeof ExecutionItemSchema>;
export type ExecutionsListResponse = z.infer<typeof ExecutionsListResponseSchema>;
