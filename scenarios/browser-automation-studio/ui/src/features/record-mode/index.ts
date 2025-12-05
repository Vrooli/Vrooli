/**
 * Record Mode Feature
 *
 * Exports for the Record Mode feature, which allows users to
 * record browser actions and generate workflows from them.
 */

// Components
export { RecordModePage } from './RecordModePage';
export { ActionTimeline } from './ActionTimeline';

// Hooks
export { useRecordMode } from './hooks/useRecordMode';

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
