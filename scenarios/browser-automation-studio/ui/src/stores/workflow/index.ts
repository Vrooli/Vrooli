// ============================================================================
// Browser Automation Studio - Workflow Store
// ============================================================================
// This module provides the Zustand store for workflow management.
// Re-exports all types and the store hook for convenience.

// Types
export type {
  SaveErrorType,
  SaveErrorState,
  ViewportPreset,
  ExecutionViewportSettings,
  Workflow,
  WorkflowVersionSummary,
  WorkflowConflictMetadata,
  SaveWorkflowOptions,
  AutosaveOptions,
  WorkflowStore,
  WorkflowLoadState,
} from './types';

// Store
export { useWorkflowStore } from './workflowStore';

// Utilities (exported for testing and advanced use cases)
export {
  MIN_VIEWPORT_DIMENSION,
  MAX_VIEWPORT_DIMENSION,
  sanitizeViewportSettings,
  extractExecutionViewport,
} from './utils/viewport';

export {
  canvasToWorkflowDefinition,
  workflowDefinitionToCanvas,
  workflowPayloadToCanvas,
  workflowDefinitionToProto,
} from './utils/codec';

export {
  computeWorkflowFingerprint,
  stableSerialize,
} from './utils/fingerprint';

export {
  normalizeVersionSummary,
  buildWorkflowLoadState,
} from './utils/normalization';
