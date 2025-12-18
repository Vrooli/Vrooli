// ============================================================================
// Backward Compatibility Re-export
// ============================================================================
// This file re-exports from the new domains/executions/ location.
// All imports from '@stores/executionStore' continue to work unchanged.
//
// Prefer importing from '@/domains/executions' for new code.

export {
  useExecutionStore,
  mapTimelineEntryToFrame,
  mapTimelineFrameFromProto,
} from '../domains/executions/store';

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
} from '../domains/executions/store';
