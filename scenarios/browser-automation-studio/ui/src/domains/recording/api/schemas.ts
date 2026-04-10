/**
 * Zod Schemas for Recording API Responses
 *
 * These schemas provide runtime validation for API responses,
 * replacing manual type guards with proper schema validation.
 */

import { z } from 'zod';

// ============================================================================
// Recording Lifecycle Schemas
// ============================================================================

/**
 * Response from start recording endpoint.
 */
export const StartRecordingResponseSchema = z.object({
  recording_id: z.string(),
  session_id: z.string(),
  started_at: z.string(),
});

export type StartRecordingResponse = z.infer<typeof StartRecordingResponseSchema>;

/**
 * Response from stop recording endpoint.
 */
export const StopRecordingResponseSchema = z.object({
  recording_id: z.string(),
  session_id: z.string(),
  action_count: z.number(),
  stopped_at: z.string(),
});

export type StopRecordingResponse = z.infer<typeof StopRecordingResponseSchema>;

/**
 * Response from generate workflow endpoint.
 */
export const GenerateWorkflowResponseSchema = z.object({
  workflow_id: z.string(),
  project_id: z.string(),
  name: z.string(),
  node_count: z.number(),
  action_count: z.number(),
});

export type GenerateWorkflowResponse = z.infer<typeof GenerateWorkflowResponseSchema>;

// ============================================================================
// Selector Validation Schemas
// ============================================================================

/**
 * Selector validation result.
 */
export const SelectorValidationSchema = z.object({
  valid: z.boolean(),
  match_count: z.number(),
  selector: z.string(),
  error: z.string().optional(),
});

export type SelectorValidation = z.infer<typeof SelectorValidationSchema>;

// ============================================================================
// Replay Preview Schemas
// ============================================================================

/**
 * Error codes for failed action replays.
 */
export const ActionReplayErrorCodeSchema = z.enum([
  'SELECTOR_NOT_FOUND',
  'SELECTOR_AMBIGUOUS',
  'ELEMENT_NOT_VISIBLE',
  'ELEMENT_NOT_ENABLED',
  'TIMEOUT',
  'NAVIGATION_FAILED',
  'UNKNOWN',
]);

export type ActionReplayErrorCode = z.infer<typeof ActionReplayErrorCodeSchema>;

/**
 * Error details for a failed action replay.
 */
export const ActionReplayErrorSchema = z.object({
  message: z.string(),
  code: ActionReplayErrorCodeSchema,
  match_count: z.number().optional(),
  selector: z.string().optional(),
});

export type ActionReplayError = z.infer<typeof ActionReplayErrorSchema>;

/**
 * Action types that can be recorded.
 */
export const ActionTypeSchema = z.enum([
  'navigate',
  'click',
  'input',
  'wait',
  'assert',
  'scroll',
  'select',
  'evaluate',
  'keyboard',
  'hover',
  'screenshot',
  'focus',
  'blur',
]);

export type ActionType = z.infer<typeof ActionTypeSchema>;

/**
 * Result of replaying a single action.
 */
export const ActionReplayResultSchema = z.object({
  action_id: z.string(),
  sequence_num: z.number(),
  action_type: ActionTypeSchema,
  success: z.boolean(),
  duration_ms: z.number(),
  error: ActionReplayErrorSchema.optional(),
  screenshot_on_error: z.string().optional(),
});

export type ActionReplayResult = z.infer<typeof ActionReplayResultSchema>;

/**
 * Response from replay preview.
 */
export const ReplayPreviewResponseSchema = z.object({
  success: z.boolean(),
  total_actions: z.number(),
  passed_actions: z.number(),
  failed_actions: z.number(),
  total_duration_ms: z.number(),
  stopped_early: z.boolean(),
  results: z.array(ActionReplayResultSchema),
});

export type ReplayPreviewResponse = z.infer<typeof ReplayPreviewResponseSchema>;

// ============================================================================
// Session Schemas
// ============================================================================

/**
 * Viewport source attribution.
 */
export const ViewportSourceSchema = z.enum([
  'requested',
  'fingerprint',
  'fingerprint_partial',
  'default',
]);

export type ViewportSource = z.infer<typeof ViewportSourceSchema>;

/**
 * Actual viewport from Playwright with source attribution.
 */
export const ActualViewportSchema = z.object({
  width: z.number(),
  height: z.number(),
  source: ViewportSourceSchema,
  reason: z.string().optional(),
});

export type ActualViewport = z.infer<typeof ActualViewportSchema>;

/**
 * Response from create session endpoint.
 */
export const CreateSessionResponseSchema = z.object({
  session_id: z.string(),
  session_profile_id: z.string().optional(),
  actual_viewport: ActualViewportSchema.optional(),
  initial_url: z.string().optional(),
});

export type CreateSessionResponse = z.infer<typeof CreateSessionResponseSchema>;

/**
 * Response from navigation endpoints.
 */
export const NavigationResponseSchema = z.object({
  url: z.string().optional(),
  can_go_back: z.boolean().optional(),
  can_go_forward: z.boolean().optional(),
});

export type NavigationResponse = z.infer<typeof NavigationResponseSchema>;

// ============================================================================
// Timeline Schemas
// ============================================================================

/**
 * Timeline entry types.
 */
export const TimelineEntryTypeSchema = z.enum(['action', 'page_event']);

export type TimelineEntryType = z.infer<typeof TimelineEntryTypeSchema>;

/**
 * Page event types.
 */
export const PageEventTypeSchema = z.enum(['page_created', 'page_navigated', 'page_closed']);

export type PageEventType = z.infer<typeof PageEventTypeSchema>;

/**
 * Timeline action from API.
 */
export const TimelineActionSchema = z.object({
  id: z.string(),
  actionType: z.string(),
  url: z.string().optional(),
  sequenceNum: z.number(),
  timestamp: z.string(),
  selector: z.object({
    primary: z.string(),
  }).optional(),
  payload: z.record(z.unknown()).optional(),
  confidence: z.number(),
  pageTitle: z.string().optional(),
});

export type TimelineAction = z.infer<typeof TimelineActionSchema>;

/**
 * Page event from timeline.
 */
export const TimelinePageEventSchema = z.object({
  id: z.string(),
  type: PageEventTypeSchema,
  pageId: z.string(),
  url: z.string().optional(),
  title: z.string().optional(),
  openerId: z.string().optional(),
  timestamp: z.string(),
});

export type TimelinePageEvent = z.infer<typeof TimelinePageEventSchema>;

/**
 * Unified timeline entry.
 */
export const TimelineEntrySchema = z.object({
  id: z.string(),
  type: TimelineEntryTypeSchema,
  timestamp: z.string(),
  pageId: z.string(),
  action: TimelineActionSchema.optional(),
  pageEvent: TimelinePageEventSchema.optional(),
});

export type TimelineEntry = z.infer<typeof TimelineEntrySchema>;

/**
 * API response for timeline.
 */
export const TimelineResponseSchema = z.object({
  entries: z.array(TimelineEntrySchema),
  hasMore: z.boolean(),
  totalEntries: z.number(),
});

export type TimelineResponse = z.infer<typeof TimelineResponseSchema>;

// ============================================================================
// AI Navigation Schemas
// ============================================================================

/**
 * AI navigation response from start navigation endpoint.
 */
export const AINavigateResponseSchema = z.object({
  navigation_id: z.string(),
  status: z.string(),
  model: z.string(),
  max_steps: z.number(),
  estimated_cost: z.number().optional(),
});

export type AINavigateResponse = z.infer<typeof AINavigateResponseSchema>;

/**
 * Browser action types for AI navigation.
 */
export const BrowserActionTypeSchema = z.enum([
  'click',
  'type',
  'scroll',
  'navigate',
  'hover',
  'select',
  'wait',
  'keypress',
  'done',
  'request_human',
]);

export type BrowserActionType = z.infer<typeof BrowserActionTypeSchema>;

/**
 * Scroll direction types.
 */
export const ScrollDirectionSchema = z.enum(['up', 'down', 'left', 'right']);

export type ScrollDirection = z.infer<typeof ScrollDirectionSchema>;

/**
 * Human intervention types.
 */
export const InterventionTypeSchema = z.enum([
  'captcha',
  'verification',
  'complex_interaction',
  'login_required',
  'other',
]);

export type InterventionType = z.infer<typeof InterventionTypeSchema>;

/**
 * Browser action from AI navigation.
 */
export const BrowserActionSchema = z.object({
  type: BrowserActionTypeSchema,
  elementId: z.number().optional(),
  coordinates: z.object({
    x: z.number(),
    y: z.number(),
  }).optional(),
  text: z.string().optional(),
  direction: ScrollDirectionSchema.optional(),
  url: z.string().optional(),
  key: z.string().optional(),
  result: z.string().optional(),
  success: z.boolean().optional(),
  reason: z.string().optional(),
  instructions: z.string().optional(),
  interventionType: InterventionTypeSchema.optional(),
});

export type BrowserAction = z.infer<typeof BrowserActionSchema>;

/**
 * Token usage for AI navigation step.
 */
export const TokensUsedSchema = z.object({
  promptTokens: z.number(),
  completionTokens: z.number(),
  totalTokens: z.number(),
});

export type TokensUsed = z.infer<typeof TokensUsedSchema>;

/**
 * AI navigation step event from WebSocket.
 */
export const AINavigationStepEventSchema = z.object({
  type: z.literal('ai_navigation_step'),
  navigationId: z.string(),
  sessionId: z.string(),
  stepNumber: z.number(),
  action: BrowserActionSchema,
  reasoning: z.string(),
  currentUrl: z.string(),
  goalAchieved: z.boolean(),
  tokensUsed: TokensUsedSchema,
  durationMs: z.number(),
  error: z.string().optional(),
  timestamp: z.string(),
});

export type AINavigationStepEvent = z.infer<typeof AINavigationStepEventSchema>;

/**
 * AI navigation complete status.
 */
export const AINavigationCompleteStatusSchema = z.enum([
  'completed',
  'failed',
  'aborted',
  'max_steps_reached',
  'loop_detected',
  'awaiting_human',
]);

export type AINavigationCompleteStatus = z.infer<typeof AINavigationCompleteStatusSchema>;

/**
 * AI navigation complete event from WebSocket.
 */
export const AINavigationCompleteEventSchema = z.object({
  type: z.literal('ai_navigation_complete'),
  navigationId: z.string(),
  sessionId: z.string(),
  status: AINavigationCompleteStatusSchema,
  totalSteps: z.number(),
  totalTokens: z.number(),
  totalDurationMs: z.number(),
  finalUrl: z.string(),
  error: z.string().optional(),
  summary: z.string().optional(),
  timestamp: z.string(),
});

export type AINavigationCompleteEvent = z.infer<typeof AINavigationCompleteEventSchema>;

/**
 * Trigger types for human intervention.
 */
export const HumanInterventionTriggerSchema = z.enum(['programmatic', 'ai_requested']);

export type HumanInterventionTrigger = z.infer<typeof HumanInterventionTriggerSchema>;

/**
 * AI navigation awaiting human event from WebSocket.
 */
export const AINavigationAwaitingHumanEventSchema = z.object({
  type: z.literal('ai_navigation_awaiting_human'),
  navigationId: z.string(),
  sessionId: z.string(),
  stepNumber: z.number(),
  reason: z.string(),
  instructions: z.string().optional(),
  interventionType: InterventionTypeSchema,
  trigger: HumanInterventionTriggerSchema,
  timestamp: z.string(),
});

export type AINavigationAwaitingHumanEvent = z.infer<typeof AINavigationAwaitingHumanEventSchema>;

/**
 * AI navigation resumed event from WebSocket.
 */
export const AINavigationResumedEventSchema = z.object({
  type: z.literal('ai_navigation_resumed'),
  navigationId: z.string(),
  sessionId: z.string(),
  timestamp: z.string(),
});

export type AINavigationResumedEvent = z.infer<typeof AINavigationResumedEventSchema>;

// ============================================================================
// History Schemas
// ============================================================================

/**
 * A single entry in the browser navigation history.
 */
export const HistoryEntrySchema = z.object({
  id: z.string(),
  url: z.string(),
  title: z.string(),
  timestamp: z.string(),
  thumbnail: z.string().optional(),
});

export type HistoryEntry = z.infer<typeof HistoryEntrySchema>;

/**
 * Settings for history capture behavior.
 */
export const HistorySettingsSchema = z.object({
  maxEntries: z.number(),
  retentionDays: z.number(),
  captureThumbnails: z.boolean(),
});

export type HistorySettings = z.infer<typeof HistorySettingsSchema>;

/**
 * Statistics about the history.
 */
export const HistoryStatsSchema = z.object({
  totalEntries: z.number(),
  oldestEntry: z.string().optional(),
  newestEntry: z.string().optional(),
});

export type HistoryStats = z.infer<typeof HistoryStatsSchema>;

/**
 * API response for history endpoints.
 */
export const HistoryResponseSchema = z.object({
  entries: z.array(HistoryEntrySchema),
  settings: HistorySettingsSchema,
  stats: HistoryStatsSchema,
});

export type HistoryResponse = z.infer<typeof HistoryResponseSchema>;

// ============================================================================
// Error Response Schema
// ============================================================================

/**
 * Standard API error response.
 */
export const ApiErrorResponseSchema = z.object({
  message: z.string().optional(),
  error: z.string().optional(),
  code: z.string().optional(),
  details: z.record(z.string()).optional(),
});

export type ApiErrorResponse = z.infer<typeof ApiErrorResponseSchema>;
