import { z } from 'zod';

/**
 * Schedule-related Zod schemas for API response validation.
 * These schemas provide runtime validation for workflow scheduling data.
 */

// Workflow schedule schema
export const WorkflowScheduleSchema = z.object({
  id: z.string(),
  workflow_id: z.string(),
  name: z.string(),
  description: z.string().optional(),
  cron_expression: z.string(),
  timezone: z.string(),
  is_active: z.boolean(),
  parameters: z.record(z.string(), z.unknown()).optional(),
  next_run_at: z.string().optional(),
  last_run_at: z.string().optional(),
  created_at: z.string().optional(),
  updated_at: z.string().optional(),
  // Computed fields from API
  workflow_name: z.string().optional(),
  next_run_human: z.string().optional(),
  last_run_status: z.string().optional(),
});

// Schedule occurrence schema (for calendar view)
export const ScheduleOccurrenceSchema = z.object({
  schedule_id: z.string(),
  schedule_name: z.string(),
  workflow_id: z.string(),
  workflow_name: z.string(),
  run_at: z.string(),
  is_active: z.boolean(),
  cron_expression: z.string(),
  timezone: z.string(),
});

// Schedule aggregate schema (for calendar view summary)
export const ScheduleAggregateSchema = z.object({
  schedule_id: z.string(),
  schedule_name: z.string(),
  total_runs: z.number(),
  truncated: z.boolean(),
  cron_expression: z.string(),
});

// List schedules response schema
export const ListSchedulesResponseSchema = z.object({
  schedules: z.array(WorkflowScheduleSchema),
});

// Single schedule response schema (for create/update/toggle)
export const ScheduleResponseSchema = z.object({
  schedule: WorkflowScheduleSchema,
});

// Occurrences response schema
export const OccurrencesResponseSchema = z.object({
  occurrences: z.array(ScheduleOccurrenceSchema),
  aggregates: z.record(z.string(), ScheduleAggregateSchema).optional(),
});

// Trigger schedule response schema
export const TriggerScheduleResponseSchema = z.object({
  execution_id: z.string(),
  triggered_at: z.string().optional(),
});

// Export types
export type WorkflowSchedule = z.infer<typeof WorkflowScheduleSchema>;
export type ScheduleOccurrence = z.infer<typeof ScheduleOccurrenceSchema>;
export type ScheduleAggregate = z.infer<typeof ScheduleAggregateSchema>;
export type ListSchedulesResponse = z.infer<typeof ListSchedulesResponseSchema>;
export type ScheduleResponse = z.infer<typeof ScheduleResponseSchema>;
export type OccurrencesResponse = z.infer<typeof OccurrencesResponseSchema>;
export type TriggerScheduleResponse = z.infer<typeof TriggerScheduleResponseSchema>;
