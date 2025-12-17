/**
 * Recording-Specific Action Type Utilities
 *
 * This module contains recording-specific logic that doesn't belong in the
 * canonical proto/action-type-utils.ts module.
 *
 * For action type utilities (ACTION_TYPE_MAP, actionTypeToString, etc.),
 * import from '../proto/action-type-utils' directly.
 *
 * @module recording/action-types
 */

import { ActionType, isSelectorOptional } from '../proto/action-type-utils';
import { adjustConfidenceForStrongType } from './selector-config';

/** Default confidence when selector info is missing or incomplete */
const DEFAULT_CONFIDENCE = 0.5;

/**
 * Calculate confidence score for an action based on selector quality.
 *
 * This is recording-specific logic that evaluates how reliable a recorded
 * selector is likely to be during replay.
 *
 * NOTE: Strong selector type confidence adjustment is delegated to
 * adjustConfidenceForStrongType() from selector-config.ts (single source of truth).
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
    return DEFAULT_CONFIDENCE;
  }

  // Use the confidence of the primary selector
  const primaryCandidate = selector.candidates.find(
    (c) => c.value === selector.primary
  );

  if (!primaryCandidate) {
    return DEFAULT_CONFIDENCE;
  }

  const baseConfidence = primaryCandidate.confidence ?? DEFAULT_CONFIDENCE;

  // Apply strong type confidence floor (single source of truth in selector-config)
  return adjustConfidenceForStrongType(primaryCandidate.type, baseConfidence);
}
