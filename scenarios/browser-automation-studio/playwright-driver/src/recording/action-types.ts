/**
 * Recording Action Types Registry
 *
 * PROTO-FIRST ARCHITECTURE:
 * This module provides utilities for working with proto ActionType enum
 * and converting between browser events and proto types.
 *
 * CHANGE AXIS: Recording Action Types
 * When adding a new action type:
 * 1. Add to the proto schema (action.proto)
 * 2. Regenerate proto types
 * 3. Add normalization mapping here if browser sends different name
 */

import { ActionType } from '../proto';

/**
 * Map from raw browser event type string to proto ActionType enum.
 * Handles common variations in event naming from the browser.
 */
const ACTION_TYPE_FROM_STRING: Record<string, ActionType> = {
  // Direct mappings
  click: ActionType.CLICK,
  type: ActionType.INPUT,
  input: ActionType.INPUT,
  scroll: ActionType.SCROLL,
  navigate: ActionType.NAVIGATE,
  select: ActionType.SELECT,
  hover: ActionType.HOVER,
  focus: ActionType.FOCUS,
  blur: ActionType.BLUR,
  keypress: ActionType.KEYBOARD,
  keyboard: ActionType.KEYBOARD,
  wait: ActionType.WAIT,
  assert: ActionType.ASSERT,
  screenshot: ActionType.SCREENSHOT,
  evaluate: ActionType.EVALUATE,
  // Aliases for browser event types
  change: ActionType.SELECT,
  keydown: ActionType.KEYBOARD,
  keyup: ActionType.KEYBOARD,
  mousedown: ActionType.CLICK,
  mouseup: ActionType.CLICK,
  dblclick: ActionType.CLICK,
};

/**
 * Action types that don't require selectors.
 * Used for confidence calculation and validation.
 */
export const SELECTOR_OPTIONAL_ACTIONS: ReadonlySet<ActionType> = new Set([
  ActionType.SCROLL,
  ActionType.NAVIGATE,
  ActionType.WAIT,
  ActionType.KEYBOARD,
  ActionType.SCREENSHOT,
  ActionType.EVALUATE,
]);

/**
 * Normalize raw action type string to proto ActionType enum.
 *
 * @param rawType - Raw action type from browser event
 * @returns Proto ActionType (defaults to CLICK for unknown)
 */
export function normalizeToProtoActionType(rawType: string): ActionType {
  const normalized = rawType.toLowerCase();
  return ACTION_TYPE_FROM_STRING[normalized] ?? ActionType.CLICK;
}

/**
 * Convert proto ActionType enum to string name.
 * Useful for logging and debugging.
 *
 * @param actionType - Proto ActionType enum value
 * @returns Human-readable string name
 */
export function actionTypeToString(actionType: ActionType): string {
  return ActionType[actionType] ?? 'UNKNOWN';
}

/**
 * Check if action type is valid (not UNSPECIFIED).
 *
 * @param actionType - Action type to validate
 * @returns True if valid (not UNSPECIFIED)
 */
export function isValidActionType(actionType: ActionType): boolean {
  return actionType !== ActionType.UNSPECIFIED;
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
 * Calculate confidence score for an action based on selector quality.
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

/**
 * Get all supported proto action types.
 * Useful for validation and documentation.
 */
export function getSupportedActionTypes(): ActionType[] {
  return [
    ActionType.NAVIGATE,
    ActionType.CLICK,
    ActionType.INPUT,
    ActionType.WAIT,
    ActionType.ASSERT,
    ActionType.SCROLL,
    ActionType.SELECT,
    ActionType.EVALUATE,
    ActionType.KEYBOARD,
    ActionType.HOVER,
    ActionType.SCREENSHOT,
    ActionType.FOCUS,
    ActionType.BLUR,
    ActionType.SUBFLOW,
    ActionType.EXTRACT,
    ActionType.UPLOAD_FILE,
    ActionType.DOWNLOAD,
    ActionType.FRAME_SWITCH,
    ActionType.TAB_SWITCH,
    ActionType.COOKIE_STORAGE,
    ActionType.SHORTCUT,
    ActionType.DRAG_DROP,
    ActionType.GESTURE,
    ActionType.NETWORK_MOCK,
    ActionType.ROTATE,
  ];
}
