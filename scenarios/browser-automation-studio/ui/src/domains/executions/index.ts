/**
 * Executions domain - handles execution viewing, history, and live monitoring
 *
 * Structure:
 * - viewer/: ExecutionViewer and related components
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
  Screenshot,
  LogEntry,
  TimelineFrame,
  TimelineBoundingBox,
  TimelineRegion,
  TimelineRetryHistoryEntry,
  TimelineAssertion,
  TimelineArtifact,
  TimelineLog,
} from './store';

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
export { default as ExecutionViewer } from "./viewer/ExecutionViewer";
export { default as ExecutionHistory } from "./history/ExecutionHistory";
export { ExecutionPaneLayout } from "./viewer/ExecutionPaneLayout";
export { ExecutionLimitModal } from "./viewer/ExecutionLimitModal";
export { ExecutionsTab } from "./history/ExecutionsTab";

// Viewer configuration (export config, etc.)
export * from "./viewer";
