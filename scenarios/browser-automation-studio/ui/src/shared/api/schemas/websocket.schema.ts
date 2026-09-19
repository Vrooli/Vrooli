import { z } from 'zod';

/**
 * WebSocket message Zod schemas for runtime validation.
 * Uses discriminated unions for type-safe message handling.
 */

// Execution started message
export const ExecutionStartedMessageSchema = z.object({
  type: z.literal('execution_started'),
  execution_id: z.string(),
  workflow_id: z.string().optional(),
  timestamp: z.string().optional(),
});

// Execution completed message
export const ExecutionCompletedMessageSchema = z.object({
  type: z.literal('execution_completed'),
  execution_id: z.string(),
  workflow_id: z.string().optional(),
  status: z.string(),
  timestamp: z.string().optional(),
  data: z.unknown().optional(),
});

// Execution failed message
export const ExecutionFailedMessageSchema = z.object({
  type: z.literal('execution_failed'),
  execution_id: z.string(),
  workflow_id: z.string().optional(),
  status: z.string().optional(),
  message: z.string().optional(),
  timestamp: z.string().optional(),
  data: z.unknown().optional(),
});

// Progress update message
export const ProgressMessageSchema = z.object({
  type: z.literal('progress'),
  execution_id: z.string().optional(),
  progress: z.number(),
  message: z.string().optional(),
  timestamp: z.string().optional(),
});

// Step started message
export const StepStartedMessageSchema = z.object({
  type: z.literal('step_started'),
  execution_id: z.string(),
  step_index: z.number().optional(),
  node_id: z.string().optional(),
  timestamp: z.string().optional(),
});

// Step completed message
export const StepCompletedMessageSchema = z.object({
  type: z.literal('step_completed'),
  execution_id: z.string(),
  step_index: z.number().optional(),
  node_id: z.string().optional(),
  status: z.string().optional(),
  timestamp: z.string().optional(),
  data: z.unknown().optional(),
});

// Export progress message
export const ExportProgressMessageSchema = z.object({
  type: z.literal('export_progress'),
  export_id: z.string(),
  execution_id: z.string(),
  stage: z.enum(['preparing', 'capturing', 'encoding', 'finalizing', 'completed', 'failed']),
  progress_percent: z.number(),
  status: z.enum(['processing', 'completed', 'failed']),
  storage_url: z.string().optional(),
  file_size_bytes: z.number().optional(),
  error: z.string().optional(),
  timestamp: z.string().optional(),
});

// UX metrics update message
export const UXMetricsUpdateMessageSchema = z.object({
  type: z.literal('ux_metrics_update'),
  execution_id: z.string(),
  step_index: z.number(),
  friction_score: z.number(),
  signals: z.array(z.object({
    type: z.string(),
    step_index: z.number(),
    severity: z.enum(['low', 'medium', 'high']),
    score: z.number(),
    description: z.string(),
    evidence: z.record(z.string(), z.unknown()).optional(),
  })).optional(),
  timestamp: z.string().optional(),
});

// Friction alert message
export const FrictionAlertMessageSchema = z.object({
  type: z.literal('friction_alert'),
  execution_id: z.string(),
  step_index: z.number(),
  signal: z.object({
    type: z.string(),
    step_index: z.number(),
    severity: z.enum(['low', 'medium', 'high']),
    score: z.number(),
    description: z.string(),
    evidence: z.record(z.string(), z.unknown()).optional(),
  }),
  timestamp: z.string().optional(),
});

// Recording frame message (metadata only - actual frame is binary)
export const RecordingFrameMessageSchema = z.object({
  type: z.literal('recording_frame'),
  execution_id: z.string(),
  frame_number: z.number().optional(),
  timestamp: z.string().optional(),
});

// Connection status message
export const ConnectionStatusMessageSchema = z.object({
  type: z.literal('connection_status'),
  status: z.enum(['connected', 'disconnected', 'reconnecting']),
  message: z.string().optional(),
  timestamp: z.string().optional(),
});

// Generic/unknown message (fallback)
// This schema keeps the session-scoped timeline entry and page-event payloads
// intact for their respective domain consumers.
export const GenericMessageSchema = z.object({
  type: z.string(),
  execution_id: z.string().optional(),
  workflow_id: z.string().optional(),
  session_id: z.string().optional(), // For timeline and page-event messages
  entry: z.unknown().optional(), // For typed timeline entry messages
  event: z.unknown().optional(), // For page_event messages
  status: z.string().optional(),
  progress: z.number().optional(),
  message: z.string().optional(),
  data: z.unknown().optional(),
  timestamp: z.string().optional(),
});

// Discriminated union of all message types
export const WebSocketMessageSchema = z.discriminatedUnion('type', [
  ExecutionStartedMessageSchema,
  ExecutionCompletedMessageSchema,
  ExecutionFailedMessageSchema,
  ProgressMessageSchema,
  StepStartedMessageSchema,
  StepCompletedMessageSchema,
  ExportProgressMessageSchema,
  UXMetricsUpdateMessageSchema,
  FrictionAlertMessageSchema,
  RecordingFrameMessageSchema,
  ConnectionStatusMessageSchema,
]);

// Loose schema for parsing unknown messages (accepts any type)
export const LooseWebSocketMessageSchema = GenericMessageSchema;

// Export types
export type ExecutionStartedMessage = z.infer<typeof ExecutionStartedMessageSchema>;
export type ExecutionCompletedMessage = z.infer<typeof ExecutionCompletedMessageSchema>;
export type ExecutionFailedMessage = z.infer<typeof ExecutionFailedMessageSchema>;
export type ProgressMessage = z.infer<typeof ProgressMessageSchema>;
export type StepStartedMessage = z.infer<typeof StepStartedMessageSchema>;
export type StepCompletedMessage = z.infer<typeof StepCompletedMessageSchema>;
export type ExportProgressMessage = z.infer<typeof ExportProgressMessageSchema>;
export type UXMetricsUpdateMessage = z.infer<typeof UXMetricsUpdateMessageSchema>;
export type FrictionAlertMessage = z.infer<typeof FrictionAlertMessageSchema>;
export type RecordingFrameMessage = z.infer<typeof RecordingFrameMessageSchema>;
export type ConnectionStatusMessage = z.infer<typeof ConnectionStatusMessageSchema>;
export type WebSocketMessage = z.infer<typeof WebSocketMessageSchema>;
export type GenericMessage = z.infer<typeof GenericMessageSchema>;
