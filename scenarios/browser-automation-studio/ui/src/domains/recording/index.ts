/**
 * Recording Domain
 *
 * Browser action recording and workflow generation.
 * This domain handles capturing browser interactions, displaying them
 * in a timeline, and converting them into reusable workflows.
 */

// Main component
export { RecordModePage as RecordingSession } from './RecordingSession';

// Timeline components
export { ActionTimeline } from './timeline/ActionTimeline';
export { SelectorEditor } from './timeline/SelectorEditor';
export { TimelineEventCard, UnifiedTimeline } from './timeline/TimelineEventCard';
export { RecordPreviewPanel } from './timeline/RecordPreviewPanel';
export { TimelineFullView } from './timeline/TimelineFullView';
export { TimelineSidebar } from './timeline/TimelineSidebar';

// Capture components
export { PlaywrightView } from './capture/PlaywrightView';
export { BrowserUrlBar } from './capture/BrowserUrlBar';
export { RecordingHeader } from './capture/RecordingHeader';
export { RecordActionsPanel } from './capture/RecordActionsPanel';
export { FloatingActionBar } from './capture/FloatingActionBar';
export { FloatingMiniPreview } from './capture/FloatingMiniPreview';
export { FrameStatsDisplay } from './capture/FrameStatsDisplay';
export { StreamSettings } from './capture/StreamSettings';
export { ErrorBanner, UnstableSelectorsBanner } from './capture/RecordModeBanners';
export { ClearActionsModal } from './capture/RecordModeModals';

// Conversion components
export { WorkflowCreationForm } from './conversion/WorkflowCreationForm';

// Hooks
export { useRecordMode } from './hooks/useRecordMode';
export { useUnifiedTimeline } from './hooks/useUnifiedTimeline';
export { useRecordingSession } from './hooks/useRecordingSession';
export { useSessionProfiles } from './hooks/useSessionProfiles';
export { useActionSelection } from './hooks/useActionSelection';
export { useTimelinePanel } from './hooks/useTimelinePanel';
export { useFrameStats } from './hooks/useFrameStats';
export { usePerfStats } from './hooks/usePerfStats';
export { useRecordModeLayout } from './hooks/useRecordModeLayout';
export type { UseUnifiedTimelineOptions, UseUnifiedTimelineReturn } from './hooks/useUnifiedTimeline';

// Utilities
export { mergeConsecutiveActions, getMergeDescription } from './utils/mergeActions';
export type { MergedAction, MergedActionMeta } from './utils/mergeActions';
export { mapClientToViewport } from './utils/coordinateMapping';
export type { Rect as CoordinateRect, Point as CoordinatePoint } from './utils/coordinateMapping';

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
  RecordingSessionProfile,
  ReplayPreviewResponse,
} from './types/types';

// Unified Timeline Types
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
