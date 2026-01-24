import { z } from 'zod';

/**
 * UX Metrics Zod schemas for API response validation.
 * These schemas provide runtime validation for friction analysis and metrics data.
 */

// Point schema
export const PointSchema = z.object({
  x: z.number(),
  y: z.number(),
});

// Timed point schema
export const TimedPointSchema = z.object({
  x: z.number(),
  y: z.number(),
  timestamp: z.string(),
});

// Cursor path schema
export const CursorPathSchema = z.object({
  step_index: z.number(),
  points: z.array(TimedPointSchema),
  total_distance_px: z.number(),
  duration_ms: z.number(),
  direct_distance_px: z.number(),
  directness: z.number(),
  zigzag_score: z.number(),
  average_speed_px_ms: z.number(),
  max_speed_px_ms: z.number(),
  hesitation_count: z.number(),
});

// Friction type enum
export const FrictionTypeSchema = z.enum([
  'excessive_time',
  'zigzag_path',
  'multiple_retries',
  'rapid_clicks',
  'long_hesitation',
  'back_navigation',
  'element_miss',
]);

// UX Metrics severity enum (for friction signals)
export const UXSeveritySchema = z.enum(['low', 'medium', 'high']);

// Friction signal schema
export const FrictionSignalSchema = z.object({
  type: FrictionTypeSchema,
  step_index: z.number(),
  severity: UXSeveritySchema,
  score: z.number(),
  description: z.string(),
  evidence: z.record(z.string(), z.unknown()).optional(),
});

// Step metrics schema
export const StepMetricsSchema = z.object({
  step_index: z.number(),
  node_id: z.string(),
  step_type: z.string(),
  time_to_action_ms: z.number(),
  action_duration_ms: z.number(),
  total_duration_ms: z.number(),
  cursor_path: CursorPathSchema.optional(),
  retry_count: z.number(),
  friction_signals: z.array(FrictionSignalSchema),
  friction_score: z.number(),
});

// Metrics summary schema
export const MetricsSummarySchema = z.object({
  high_friction_steps: z.array(z.number()),
  slowest_steps: z.array(z.number()),
  top_friction_types: z.array(z.string()),
  recommended_actions: z.array(z.string()),
});

// Execution metrics schema (main response)
export const ExecutionMetricsSchema = z.object({
  execution_id: z.string(),
  workflow_id: z.string(),
  computed_at: z.string(),
  total_duration_ms: z.number(),
  step_count: z.number(),
  successful_steps: z.number(),
  failed_steps: z.number(),
  total_retries: z.number(),
  avg_step_duration_ms: z.number(),
  total_cursor_distance_px: z.number(),
  overall_friction_score: z.number(),
  friction_signals: z.array(FrictionSignalSchema),
  step_metrics: z.array(StepMetricsSchema),
  summary: MetricsSummarySchema.optional(),
});

// Workflow metrics aggregate schema
export const WorkflowMetricsAggregateSchema = z.object({
  workflow_id: z.string(),
  execution_count: z.number(),
  avg_friction_score: z.number(),
  avg_duration_ms: z.number(),
  trend_direction: z.enum(['improving', 'degrading', 'stable']),
  high_friction_step_frequency: z.record(z.string(), z.number()),
});

// Export types
export type Point = z.infer<typeof PointSchema>;
export type TimedPoint = z.infer<typeof TimedPointSchema>;
export type CursorPath = z.infer<typeof CursorPathSchema>;
export type FrictionType = z.infer<typeof FrictionTypeSchema>;
export type UXSeverity = z.infer<typeof UXSeveritySchema>;
export type FrictionSignal = z.infer<typeof FrictionSignalSchema>;
export type StepMetrics = z.infer<typeof StepMetricsSchema>;
export type MetricsSummary = z.infer<typeof MetricsSummarySchema>;
export type ExecutionMetrics = z.infer<typeof ExecutionMetricsSchema>;
export type WorkflowMetricsAggregate = z.infer<typeof WorkflowMetricsAggregateSchema>;
