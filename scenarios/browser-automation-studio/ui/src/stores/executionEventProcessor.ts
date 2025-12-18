// ============================================================================
// Backward Compatibility Re-export
// ============================================================================
// This file re-exports from the new domains/executions/utils/ location.
// All imports from '@stores/executionEventProcessor' continue to work unchanged.
//
// Prefer importing from '@/domains/executions' for new code.

export {
  processExecutionEvent,
  createId,
  parseTimestamp,
  stepLabel,
} from '../domains/executions/utils/eventProcessor';

export type {
  Screenshot,
  LogLevel,
  LogEntry,
  ExecutionEventHandlers,
  ExecutionEventContext,
} from '../domains/executions/utils/eventProcessor';
