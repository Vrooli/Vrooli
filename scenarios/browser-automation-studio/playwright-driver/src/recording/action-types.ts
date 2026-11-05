/**
 * Recording Action Types and Compatibility
 *
 * This module consolidates all ActionType enum ↔ string conversion logic.
 * Previously duplicated in:
 *   - proto/index.ts (actionTypeToString)
 *   - recording/action-types.ts (actionTypeToString, normalizeToProtoActionType)
 *   - recording/handler-adapter.ts (actionTypeToString, stringToActionType)
 *
 * CHANGE AXIS: Adding a New Action Type
 * When adding a new action type to the proto schema:
 * 1. Add to ACTION_TYPE_STRING_MAP (ActionType → handler string)
 * 2. Add to STRING_TO_ACTION_TYPE_MAP (string → ActionType)
 * 3. Add aliases to ACTION_TYPE_MAP if browser sends different event names
 * 4. Add to SELECTOR_OPTIONAL_ACTIONS if selector is optional
 *
 * @module recording/action-types
 */

import { ActionType } from '@vrooli/proto-types/browser-automation-studio/v1/actions/action_pb';
import { logger, LogContext, scopedLog } from '../utils';
import { adjustConfidenceForStrongType } from './validation/selector-config';

// =============================================================================
// ActionType → String Mappings
// =============================================================================

/**
 * Action kinds implemented by this driver. The values come from the generated
 * proto enum; the list represents driver capability, not a second vocabulary.
 */
const SUPPORTED_ACTION_TYPES: readonly ActionType[] = Object.values(ActionType).filter(
  (value): value is ActionType => typeof value === 'number' && value !== ActionType.UNSPECIFIED,
);

const HANDLER_NAME_OVERRIDES = new Map<ActionType, string>([
  // Existing route registration uses this compact legacy spelling.
  [ActionType.UPLOAD_FILE, 'uploadfile'],
]);

const actionTypeToHandlerName = (actionType: ActionType): string =>
  HANDLER_NAME_OVERRIDES.get(actionType) ?? ActionType[actionType].toLowerCase().replace(/_/g, '-');

const ACTION_TYPE_STRING_MAP: ReadonlyMap<ActionType, string> = new Map(
  SUPPORTED_ACTION_TYPES.map((actionType) => [actionType, actionTypeToHandlerName(actionType)]),
);

// =============================================================================
// String → ActionType Mappings
// =============================================================================

/**
 * Map lowercase strings to ActionType enum.
 * Includes aliases for handler dispatch flexibility.
 */
const STRING_TO_ACTION_TYPE_MAP: ReadonlyMap<string, ActionType> = new Map([
  ...Array.from(ACTION_TYPE_STRING_MAP, ([actionType, handlerName]) => [handlerName, actionType] as const),
  ['upload', ActionType.UPLOAD_FILE],
  ['frameswitch', ActionType.FRAME_SWITCH],
  ['tabswitch', ActionType.TAB_SWITCH],
  ['tab', ActionType.TAB_SWITCH],
  ['tabs', ActionType.TAB_SWITCH],
  ['cookiestorage', ActionType.COOKIE_STORAGE],
  ['dragdrop', ActionType.DRAG_DROP],
  ['drag', ActionType.DRAG_DROP],
  ['swipe', ActionType.GESTURE],
  ['pinch', ActionType.GESTURE],
  ['zoom', ActionType.GESTURE],
  ['networkmock', ActionType.NETWORK_MOCK],
  ['network', ActionType.NETWORK_MOCK],
  ['mock', ActionType.NETWORK_MOCK],
  ['intercept', ActionType.NETWORK_MOCK],
  ['orientation', ActionType.ROTATE],
  ['device', ActionType.ROTATE],
]);

// =============================================================================
// Browser Event → ActionType Mappings
// =============================================================================

/**
 * CANONICAL ACTION TYPE MAP
 *
 * Maps browser event strings to proto ActionType.
 * Used by recording capture (proto/recording.ts) to convert raw browser events.
 *
 * This includes both direct mappings and browser event aliases (e.g., 'mousedown' → CLICK).
 *
 * @public Exported for use by proto/recording.ts and recording/action-types.ts
 */
export const ACTION_TYPE_MAP: Record<string, ActionType> = {
  // === Core action types (direct 1:1 mapping) ===
  click: ActionType.CLICK,
  navigate: ActionType.NAVIGATE,
  scroll: ActionType.SCROLL,
  select: ActionType.SELECT,
  hover: ActionType.HOVER,
  focus: ActionType.FOCUS,
  blur: ActionType.BLUR,
  wait: ActionType.WAIT,
  assert: ActionType.ASSERT,
  screenshot: ActionType.SCREENSHOT,
  evaluate: ActionType.EVALUATE,

  // === Input variations ===
  type: ActionType.INPUT,
  input: ActionType.INPUT,

  // === Keyboard variations ===
  keyboard: ActionType.KEYBOARD,
  keypress: ActionType.KEYBOARD,
  keydown: ActionType.KEYBOARD,
  keyup: ActionType.KEYBOARD,

  // === Browser event aliases ===
  // These are raw DOM event names that should map to our canonical types
  change: ActionType.SELECT,      // <select> change events
  mousedown: ActionType.CLICK,    // Mouse button press
  mouseup: ActionType.CLICK,      // Mouse button release
  dblclick: ActionType.CLICK,     // Double-click events
};

// =============================================================================
// Selector Optional Actions
// =============================================================================

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
  ActionType.CONDITIONAL,
]);

// =============================================================================
// Conversion Functions
// =============================================================================

/**
 * Convert ActionType enum to lowercase string for handler dispatch.
 *
 * This is the canonical function for converting ActionType to handler strings.
 * Returns lowercase strings suitable for handler registration/lookup.
 *
 * @param actionType - Proto ActionType enum value
 * @returns Lowercase handler dispatch string (e.g., 'click', 'navigate')
 *
 * @example
 * actionTypeToString(ActionType.CLICK) // 'click'
 * actionTypeToString(ActionType.TAB_SWITCH) // 'tab-switch'
 */
export function actionTypeToString(actionType: ActionType): string {
  return ACTION_TYPE_STRING_MAP.get(actionType) ?? 'unknown';
}

/**
 * Convert ActionType enum to display string for logging.
 *
 * Returns the enum name as a string (e.g., 'CLICK', 'TAB_SWITCH').
 * Useful for human-readable logging output.
 *
 * @param actionType - Proto ActionType enum value
 * @returns Uppercase enum name string
 *
 * @example
 * actionTypeToDisplayString(ActionType.CLICK) // 'CLICK'
 * actionTypeToDisplayString(ActionType.TAB_SWITCH) // 'TAB_SWITCH'
 */
export function actionTypeToDisplayString(actionType: ActionType): string {
  return ActionType[actionType] ?? 'UNKNOWN';
}

/**
 * Convert string to ActionType enum.
 *
 * This is the canonical function for parsing action type strings.
 * Handles multiple aliases for flexibility (e.g., 'tab-switch', 'tabswitch', 'tab', 'tabs').
 *
 * @param typeString - Action type string (case-insensitive)
 * @returns Proto ActionType enum value (UNSPECIFIED if not recognized)
 *
 * @example
 * stringToActionType('click') // ActionType.CLICK
 * stringToActionType('tab-switch') // ActionType.TAB_SWITCH
 * stringToActionType('tabswitch') // ActionType.TAB_SWITCH (alias)
 */
export function stringToActionType(typeString: string): ActionType {
  const normalized = typeString.toLowerCase();
  return STRING_TO_ACTION_TYPE_MAP.get(normalized) ?? ActionType.UNSPECIFIED;
}

/**
 * Normalize raw action type string to proto ActionType enum.
 *
 * Used by recording capture to convert browser event strings.
 * Defaults to CLICK for unknown types (backward compatibility).
 *
 * @param rawType - Raw action type from browser event
 * @returns Proto ActionType (defaults to CLICK for unknown)
 *
 * @example
 * normalizeToProtoActionType('mousedown') // ActionType.CLICK
 * normalizeToProtoActionType('keypress') // ActionType.KEYBOARD
 */
export function normalizeToProtoActionType(rawType: string): ActionType {
  const normalized = rawType.toLowerCase();
  const actionType = ACTION_TYPE_MAP[normalized];
  if (actionType !== undefined) {
    return actionType;
  }
  // LEGACY TELEMETRY: Track unknown action types that fall back to CLICK
  // This helps identify unmapped browser event types that need explicit handling
  logger.warn(scopedLog(LogContext.RECORDING, 'unknown action type defaulting to CLICK'), {
    rawType,
    normalized,
    telemetryReason: 'UNKNOWN_ACTION_TYPE_FALLBACK',
  });
  return ActionType.CLICK;
}

// =============================================================================
// Utility Functions
// =============================================================================

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
 * Get all supported action types.
 *
 * Returns all ActionType enum values that have handler support.
 * Useful for validation and documentation.
 */
export function getSupportedActionTypes(): ActionType[] {
  return Array.from(ACTION_TYPE_STRING_MAP.keys());
}

/**
 * Get all registered handler type strings.
 *
 * Returns all lowercase handler dispatch strings.
 * Useful for debugging and documentation.
 */
export function getRegisteredTypeStrings(): string[] {
  return Array.from(ACTION_TYPE_STRING_MAP.values());
}

// Re-export ActionType for convenience
export { ActionType };

// Recording-specific selector confidence policy.
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
