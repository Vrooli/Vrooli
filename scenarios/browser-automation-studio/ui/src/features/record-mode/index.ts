/**
 * Record Mode Feature
 *
 * Exports for the Record Mode feature, which allows users to
 * record browser actions and generate workflows from them.
 */

// Components
export { RecordModePage } from './RecordModePage';
export { ActionTimeline } from './ActionTimeline';
export { SelectorEditor } from './SelectorEditor';
export { TimelineEventCard, UnifiedTimeline } from './components/TimelineEventCard';

// Hooks
export { useRecordMode } from './hooks/useRecordMode';
export { useUnifiedTimeline } from './hooks/useUnifiedTimeline';
export type { UseUnifiedTimelineOptions, UseUnifiedTimelineReturn } from './hooks/useUnifiedTimeline';

// Utilities
export { mergeConsecutiveActions, getMergeDescription } from './mergeActions';
export type { MergedAction, MergedActionMeta } from './mergeActions';

// Types
export type {
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
} from './types';

// Unified Timeline Types (for recording + execution modes)
export type {
  TimelineItem,
  TimelineMode,
  TimelineEntry,
  TimelineEntryAggregates,
  ActionDefinition,
  ActionTelemetry,
  EventContext,
  WorkflowNodeV2,
  ActionMetadata,
} from './types/timeline-unified';
export {
  recordedActionToTimelineItem,
  timelineEntryToTimelineItem,
  timelineEntryToRecordedAction,
  hasTimelineEntry,
  parseTimelineEntry,
} from './types/timeline-unified';
