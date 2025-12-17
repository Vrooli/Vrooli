/**
 * Recording Action Types Registry
 *
 * PROTO-FIRST ARCHITECTURE:
 * This module re-exports ActionType utilities from the canonical source
 * (proto/action-type-utils.ts) and provides recording-specific functions.
 *
 * CHANGE AXIS: Recording Action Types
 * When adding a new action type:
 * 1. Add to the proto schema (action.proto)
 * 2. Regenerate proto types
 * 3. Update proto/action-type-utils.ts with new mappings
 *
 * @module recording/action-types
 */

// =============================================================================
// RE-EXPORTS FROM CANONICAL SOURCE
// =============================================================================
// All ActionType utilities are now consolidated in proto/action-type-utils.ts
// These re-exports maintain backward compatibility for existing imports.

export {
  // Data exports
  ACTION_TYPE_MAP,
  SELECTOR_OPTIONAL_ACTIONS,
  // Conversion functions
  normalizeToProtoActionType,
  actionTypeToString,
  actionTypeToDisplayString,
  stringToActionType,
  // Utility functions
  isValidActionType,
  isSelectorOptional,
  getSupportedActionTypes,
  // ActionType enum re-export for convenience
  ActionType,
} from '../proto/action-type-utils';

// =============================================================================
// RECORDING-SPECIFIC FUNCTIONS
// =============================================================================

// Import ActionType for use in this file
import { ActionType, isSelectorOptional } from '../proto/action-type-utils';

/**
 * Calculate confidence score for an action based on selector quality.
 *
 * This is recording-specific logic that evaluates how reliable a recorded
 * selector is likely to be during replay.
 *
 * @param actionType - Proto ActionType
 * @param selector - Selector set from raw event
 * @returns Confidence score 0-1
 */
export function calculateActionConfidence(
  actionType: ActionType,
  selector?: { primary: string; candidates?: Array<{ type: string; value: string; confidence?: number }> }
): number {
  // Actions without selectors don't have selector-based confidence issues
  if (isSelectorOptional(actionType)) {
    return 1;
  }

  if (!selector || !selector.candidates || selector.candidates.length === 0) {
    return 0.5;
  }

  // Use the confidence of the primary selector
  const primaryCandidate = selector.candidates.find(
    (c) => c.value === selector.primary
  );

  if (!primaryCandidate) {
    return 0.5;
  }

  // If we found a stable signal (data-testid/id/aria/data-*), bump to a safe floor
  // to avoid flashing "unstable" warnings on otherwise solid selectors.
  const strongTypes = ['data-testid', 'id', 'aria', 'data-attr'];
  if (strongTypes.includes(primaryCandidate.type) && (primaryCandidate.confidence ?? 0.5) < 0.85) {
    return Math.max(primaryCandidate.confidence ?? 0.5, 0.85);
  }

  return primaryCandidate.confidence ?? 0.5;
}
