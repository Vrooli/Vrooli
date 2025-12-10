/**
 * Recording Action Types Registry
 *
 * CHANGE AXIS: Recording Action Types
 *
 * This module centralizes all recording action type definitions and mappings.
 * When adding a new action type:
 * 1. Add to ACTION_TYPES constant
 * 2. Add normalization mapping if needed
 * 3. Add proto kind mapping
 *
 * This is VOLATILE code - expected to evolve as new action types are added.
 */

import type { RecordedActionKind, TypedActionPayload, ActionType } from './types';
import type { RawBrowserEvent } from './types';

/**
 * All supported action types for recording.
 *
 * Note: This must be kept in sync with the ActionType union in types.ts.
 * The ActionType source of truth is in types.ts; this const provides
 * runtime access to the list.
 */
export const ACTION_TYPES: readonly ActionType[] = [
  'click',
  'type',
  'scroll',
  'navigate',
  'select',
  'hover',
  'focus',
  'blur',
  'keypress',
] as const;

/**
 * Map from raw event type to normalized ActionType.
 * Handles common variations in event naming.
 */
const ACTION_TYPE_ALIASES: Record<string, ActionType> = {
  // Direct mappings
  click: 'click',
  type: 'type',
  scroll: 'scroll',
  navigate: 'navigate',
  select: 'select',
  hover: 'hover',
  focus: 'focus',
  blur: 'blur',
  keypress: 'keypress',
  // Aliases
  input: 'type',
  change: 'select',
  keydown: 'keypress',
  keyup: 'keypress',
};

/**
 * Map from ActionType to proto-aligned RecordedActionKind.
 */
const ACTION_KIND_MAP: Record<ActionType, RecordedActionKind> = {
  navigate: 'RECORDED_ACTION_TYPE_NAVIGATE',
  click: 'RECORDED_ACTION_TYPE_CLICK',
  focus: 'RECORDED_ACTION_TYPE_CLICK',
  hover: 'RECORDED_ACTION_TYPE_CLICK',
  blur: 'RECORDED_ACTION_TYPE_CLICK',
  type: 'RECORDED_ACTION_TYPE_INPUT',
  keypress: 'RECORDED_ACTION_TYPE_INPUT',
  scroll: 'RECORDED_ACTION_TYPE_UNSPECIFIED',
  select: 'RECORDED_ACTION_TYPE_INPUT',
};

/**
 * Action types that don't require selectors.
 * Used for confidence calculation and validation.
 */
export const SELECTOR_OPTIONAL_ACTIONS: ReadonlySet<ActionType> = new Set([
  'scroll',
  'navigate',
]);

/**
 * Normalize raw action type string to valid ActionType.
 *
 * @param rawType - Raw action type from browser event
 * @returns Normalized ActionType (defaults to 'click' for unknown)
 */
export function normalizeActionType(rawType: string): ActionType {
  const normalized = rawType.toLowerCase();
  return ACTION_TYPE_ALIASES[normalized] ?? 'click';
}

/**
 * Get proto-aligned RecordedActionKind for an action type.
 *
 * @param actionType - Normalized action type
 * @returns Proto-aligned action kind
 */
export function toRecordedActionKind(actionType: ActionType): RecordedActionKind {
  return ACTION_KIND_MAP[actionType] ?? 'RECORDED_ACTION_TYPE_UNSPECIFIED';
}

/**
 * Check if action type is valid.
 *
 * @param type - Action type to validate
 * @returns True if valid ActionType
 */
export function isValidActionType(type: string): type is ActionType {
  return ACTION_TYPES.includes(type as ActionType);
}

/**
 * Check if action type requires a selector.
 *
 * @param actionType - Action type to check
 * @returns True if selector is optional for this action
 */
export function isSelectorOptional(actionType: ActionType): boolean {
  return SELECTOR_OPTIONAL_ACTIONS.has(actionType);
}

/**
 * Build typed action payload from raw browser event.
 *
 * @param kind - Proto-aligned action kind
 * @param raw - Raw browser event data
 * @returns Typed action payload or undefined
 */
export function buildTypedActionPayload(
  kind: RecordedActionKind,
  raw: RawBrowserEvent
): TypedActionPayload | undefined {
  const payload = (raw.payload || {}) as Record<string, unknown>;

  switch (kind) {
    case 'RECORDED_ACTION_TYPE_NAVIGATE': {
      const url = (payload.targetUrl as string) || raw.url;
      return {
        navigate: {
          url,
          waitForSelector: (payload.waitForSelector as string) || undefined,
          timeoutMs: (payload.timeoutMs as number) || undefined,
        },
      };
    }

    case 'RECORDED_ACTION_TYPE_CLICK': {
      return {
        click: {
          selector: raw.selector?.primary,
          button: (payload.button as 'left' | 'right' | 'middle') || undefined,
          clickCount: (payload.clickCount as number) || undefined,
          delayMs: (payload.delayMs as number) || (payload.delay as number) || undefined,
          scrollIntoView: (payload.scrollIntoView as boolean) || undefined,
        },
      };
    }

    case 'RECORDED_ACTION_TYPE_INPUT': {
      const value = (payload.text as string) ?? (payload.value as string) ?? '';
      return {
        input: {
          selector: raw.selector?.primary,
          value,
          isSensitive: (payload.isSensitive as boolean) || (payload.sensitive as boolean) || false,
          submit: (payload.submit as boolean) || undefined,
        },
      };
    }

    default:
      return undefined;
  }
}

/**
 * Calculate confidence score for an action based on selector quality.
 *
 * @param actionType - Normalized action type
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
