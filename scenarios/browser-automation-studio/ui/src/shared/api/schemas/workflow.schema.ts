import { z } from 'zod';
import {
  FlexibleTimestampSchema,
  MetadataSchema,
  NavigationWaitUntilSchema,
  SeveritySchema,
} from './common.schema';

/**
 * Workflow-related Zod schemas for API response validation.
 * These schemas mirror the TypeScript types in src/types/workflow.ts
 * and provide runtime validation at API boundaries.
 */

// ReactFlow Node schema (simplified for validation)
export const ReactFlowNodeSchema = z.object({
  id: z.string(),
  type: z.string().optional(),
  position: z.object({
    x: z.number(),
    y: z.number(),
  }),
  data: z.record(z.string(), z.unknown()).optional(),
  width: z.number().optional(),
  height: z.number().optional(),
  selected: z.boolean().optional(),
  dragging: z.boolean().optional(),
  hidden: z.boolean().optional(),
});

// ReactFlow Edge schema (simplified for validation)
export const ReactFlowEdgeSchema = z.object({
  id: z.string(),
  source: z.string(),
  target: z.string(),
  sourceHandle: z.string().nullable().optional(),
  targetHandle: z.string().nullable().optional(),
  type: z.string().optional(),
  animated: z.boolean().optional(),
  hidden: z.boolean().optional(),
  data: z.record(z.string(), z.unknown()).optional(),
});

// Workflow metadata schema
export const WorkflowMetadataTypedSchema = z.object({
  name: z.string().optional(),
  description: z.string().optional(),
  labels: z.record(z.string(), z.string()).optional(),
  version: z.string().optional(),
});

// Workflow settings schema
export const WorkflowSettingsTypedSchema = z.object({
  viewport_width: z.number().optional(),
  viewport_height: z.number().optional(),
  user_agent: z.string().optional(),
  locale: z.string().optional(),
  timeout_ms: z.number().optional(),
  entry_selector_timeout_ms: z.number().optional(),
  headless: z.boolean().optional(),
  navigation_wait_until: NavigationWaitUntilSchema.optional(),
  continue_on_error: z.boolean().optional(),
  slow_motion_ms: z.number().optional(),
  extras: z.record(z.string(), z.unknown()).optional(),
});

// Workflow validation issue schema
export const WorkflowValidationIssueSchema = z.object({
  severity: SeveritySchema,
  code: z.string(),
  message: z.string(),
  node_id: z.string().optional(),
  node_type: z.string().optional(),
  field: z.string().optional(),
  pointer: z.string().optional(),
  hint: z.string().optional(),
});

// Workflow validation stats schema
export const WorkflowValidationStatsSchema = z.object({
  node_count: z.number(),
  edge_count: z.number(),
  selector_count: z.number(),
  unique_selector_count: z.number(),
  element_wait_count: z.number(),
  has_metadata: z.boolean(),
  has_execution_viewport: z.boolean(),
});

// Workflow validation result schema
export const WorkflowValidationResultSchema = z.object({
  valid: z.boolean(),
  errors: z.array(WorkflowValidationIssueSchema),
  warnings: z.array(WorkflowValidationIssueSchema),
  stats: WorkflowValidationStatsSchema,
  schema_version: z.string(),
  checked_at: z.string(),
  duration_ms: z.number(),
});

// Workflow definition schema
export const WorkflowDefinitionSchema = z.object({
  metadata: MetadataSchema.nullable(),
  metadata_typed: WorkflowMetadataTypedSchema.nullable().optional(),
  settings: MetadataSchema.nullable(),
  settings_typed: WorkflowSettingsTypedSchema.nullable().optional(),
  nodes: z.array(ReactFlowNodeSchema),
  edges: z.array(ReactFlowEdgeSchema),
});

// Resilience settings schema
export const ResilienceSettingsSchema = z.object({
  maxAttempts: z.number().optional(),
  delayMs: z.number().optional(),
  backoffFactor: z.number().optional(),
  preconditionSelector: z.string().optional(),
  preconditionTimeoutMs: z.number().optional(),
  preconditionWaitMs: z.number().optional(),
  successSelector: z.string().optional(),
  successTimeoutMs: z.number().optional(),
  successWaitMs: z.number().optional(),
});

// Workflow list item schema (for list API responses)
export const WorkflowListItemSchema = z.object({
  id: z.string(),
  name: z.string().optional(),
  description: z.string().optional(),
  version: z.number().optional(),
  created_at: FlexibleTimestampSchema,
  updated_at: FlexibleTimestampSchema,
  project_id: z.string().optional(),
});

// List workflows response schema
export const ListWorkflowsResponseSchema = z.object({
  workflows: z.array(WorkflowListItemSchema),
});

// Save workflow response schema
export const SaveWorkflowResponseSchema = z.object({
  id: z.string(),
  version: z.number().optional(),
  updated_at: FlexibleTimestampSchema,
});

// Export types from schemas
export type ReactFlowNode = z.infer<typeof ReactFlowNodeSchema>;
export type ReactFlowEdge = z.infer<typeof ReactFlowEdgeSchema>;
export type WorkflowMetadataTyped = z.infer<typeof WorkflowMetadataTypedSchema>;
export type WorkflowSettingsTyped = z.infer<typeof WorkflowSettingsTypedSchema>;
export type WorkflowValidationIssue = z.infer<typeof WorkflowValidationIssueSchema>;
export type WorkflowValidationStats = z.infer<typeof WorkflowValidationStatsSchema>;
export type WorkflowValidationResult = z.infer<typeof WorkflowValidationResultSchema>;
export type WorkflowDefinition = z.infer<typeof WorkflowDefinitionSchema>;
export type ResilienceSettings = z.infer<typeof ResilienceSettingsSchema>;
export type WorkflowListItem = z.infer<typeof WorkflowListItemSchema>;
export type ListWorkflowsResponse = z.infer<typeof ListWorkflowsResponseSchema>;
export type SaveWorkflowResponse = z.infer<typeof SaveWorkflowResponseSchema>;
