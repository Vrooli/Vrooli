/**
 * Session Module
 *
 * Exports hooks and utilities for managing recording session state.
 * These hooks extract logic from RecordingSession.tsx to improve maintainability.
 */

export { useExecutionModeState, type WorkflowNode, type WorkflowEdge } from './useExecutionModeState';
export { useRecordingModeStateTemplate, type RecordingModeStateConfig } from './useRecordingModeState';
