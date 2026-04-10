import { z } from 'zod';

/**
 * Project-related Zod schemas for API response validation.
 * Used for project workflows and file tree endpoints.
 */

export const ProjectWorkflowStatsSchema = z.object({
  execution_count: z.number().optional(),
  executionCount: z.number().optional(),
  last_execution: z.string().optional(),
  lastExecution: z.string().optional(),
  success_rate: z.number().optional(),
  successRate: z.number().optional(),
}).passthrough();

export const ProjectWorkflowItemSchema = z.object({
  id: z.string(),
  name: z.string().optional(),
  description: z.string().optional(),
  folder_path: z.string().optional(),
  folderPath: z.string().optional(),
  created_at: z.string().optional(),
  createdAt: z.string().optional(),
  updated_at: z.string().optional(),
  updatedAt: z.string().optional(),
  project_id: z.string().optional(),
  projectId: z.string().optional(),
  nodes: z.array(z.unknown()).optional(),
  edges: z.array(z.unknown()).optional(),
  version: z.number().optional(),
  stats: ProjectWorkflowStatsSchema.optional(),
}).passthrough();

export const ProjectWorkflowsResponseSchema = z.object({
  workflows: z.array(ProjectWorkflowItemSchema),
});

export const ProjectEntryKindSchema = z.enum(['folder', 'workflow_file', 'asset_file']);

export const ProjectEntrySchema = z.object({
  id: z.string(),
  project_id: z.string(),
  path: z.string(),
  kind: ProjectEntryKindSchema,
  workflow_id: z.string().optional(),
  metadata: z.record(z.string(), z.unknown()).optional(),
}).passthrough();

export const ProjectEntriesResponseSchema = z.object({
  entries: z.array(ProjectEntrySchema),
});

export type ProjectWorkflowStats = z.infer<typeof ProjectWorkflowStatsSchema>;
export type ProjectWorkflowItem = z.infer<typeof ProjectWorkflowItemSchema>;
export type ProjectWorkflowsResponse = z.infer<typeof ProjectWorkflowsResponseSchema>;
export type ProjectEntryKind = z.infer<typeof ProjectEntryKindSchema>;
export type ProjectEntry = z.infer<typeof ProjectEntrySchema>;
export type ProjectEntriesResponse = z.infer<typeof ProjectEntriesResponseSchema>;
