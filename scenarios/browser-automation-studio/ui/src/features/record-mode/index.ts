/**
 * Record Mode Feature - Backward Compatibility Re-exports
 *
 * @deprecated Import from '@/domains/recording' instead
 *
 * This module re-exports from the new domains/recording location
 * for backward compatibility during the migration.
 */

export {
  // Main component (with alias)
  RecordingSession as RecordModePage,
  // Timeline components
  ActionTimeline,
  SelectorEditor,
  TimelineEventCard,
  UnifiedTimeline,
  // Hooks
  useRecordMode,
  useUnifiedTimeline,
  // Utilities
  mergeConsecutiveActions,
  getMergeDescription,
  // Timeline converters
  recordedActionToTimelineItem,
  timelineEntryToTimelineItem,
  timelineEntryToRecordedAction,
  hasTimelineEntry,
  parseTimelineEntry,
} from '@/domains/recording';

export type {
  // Hook types
  UseUnifiedTimelineOptions,
  UseUnifiedTimelineReturn,
  // Utility types
  MergedAction,
  MergedActionMeta,
  // Core types
  RecordedAction,
  SelectorSet,
  SelectorCandidate,
  ElementMeta,
  BoundingBox,
  Point,
  ActionType,
  ActionPayload,
  RecordingState,
  StartRecordingResponse,
  StopRecordingResponse,
  GetActionsResponse,
  GenerateWorkflowResponse,
  SelectorValidation,
  // Unified timeline types
  TimelineItem,
  TimelineMode,
  TimelineEntry,
  TimelineEntryAggregates,
  ActionDefinition,
  ActionTelemetry,
  EventContext,
  WorkflowNodeV2,
  ActionMetadata,
} from '@/domains/recording';
