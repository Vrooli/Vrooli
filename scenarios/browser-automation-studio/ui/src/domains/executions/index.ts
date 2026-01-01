/**
 * Executions domain - handles execution viewing, history, and live monitoring
 *
 * Structure:
 * - viewer/: Export/replay utilities and InlineExecutionViewer
 * - history/: ExecutionHistory and ExecutionsTab
 * - live/: WebSocket handling for live execution updates
 * - hooks/: React hooks for execution state
 * - utils/: Execution-related utilities
 * - store.ts: Execution state management
 */

// Store
export { useExecutionStore, mapTimelineEntryToFrame, mapTimelineFrameFromProto } from './store';
export type {
  Execution,
  ExecutionStore,
  ExecutionPage,
  Screenshot,
  LogEntry,
  TimelineFrame,
  TimelineBoundingBox,
  TimelineRegion,
  TimelineRetryHistoryEntry,
  TimelineAssertion,
  TimelineArtifact,
  TimelineLog,
  StartExecutionOptions,
  ExecutionSettingsOverrides,
  ArtifactProfile,
  ArtifactCollectionConfig,
  FrameStreamingConfig,
} from './store';

// Hooks
export { useExecutionEvents } from './hooks/useExecutionEvents';
export { useStartWorkflow } from './hooks/useStartWorkflow';
export type { UseStartWorkflowOptions, StartWorkflowParams } from './hooks/useStartWorkflow';

// Event processor utilities
export {
  processExecutionEvent,
  createId,
  parseTimestamp,
  stepLabel,
} from './utils/eventProcessor';
export type {
  ExecutionEventHandlers,
  ExecutionEventContext,
  LogLevel,
} from './utils/eventProcessor';

// Main components
export { default as ExecutionHistory } from "./history/ExecutionHistory";
export { ExecutionLimitModal } from "./viewer/ExecutionLimitModal";
export { ExecutionsTab } from "./history/ExecutionsTab";

// Inline execution viewer (for Project view)
export { InlineExecutionViewer } from "./InlineExecutionViewer";
export type { InlineExecutionViewerProps } from "./InlineExecutionViewer";
export { InlineExecutionHeader } from "./InlineExecutionHeader";
export type { InlineExecutionHeaderProps } from "./InlineExecutionHeader";

// Viewer configuration (export config, etc.)
export * from "./viewer";
