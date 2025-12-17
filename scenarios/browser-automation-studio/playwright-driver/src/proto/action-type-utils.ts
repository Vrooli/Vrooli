/**
 * Action Type Utilities - Single Source of Truth
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
 * @module proto/action-type-utils
 */

import { ActionType } from '@vrooli/proto-types/browser-automation-studio/v1/actions/action_pb';

// =============================================================================
// ActionType → String Mappings
// =============================================================================

/**
 * Map ActionType enum to lowercase handler dispatch string.
 * Used for handler registration and lookup.
 */
const ACTION_TYPE_STRING_MAP: ReadonlyMap<ActionType, string> = new Map([
  [ActionType.NAVIGATE, 'navigate'],
  [ActionType.CLICK, 'click'],
  [ActionType.INPUT, 'input'],
  [ActionType.WAIT, 'wait'],
  [ActionType.ASSERT, 'assert'],
  [ActionType.SCROLL, 'scroll'],
  [ActionType.SELECT, 'select'],
  [ActionType.EVALUATE, 'evaluate'],
  [ActionType.KEYBOARD, 'keyboard'],
  [ActionType.HOVER, 'hover'],
  [ActionType.SCREENSHOT, 'screenshot'],
  [ActionType.FOCUS, 'focus'],
  [ActionType.BLUR, 'blur'],
  [ActionType.SUBFLOW, 'subflow'],
  [ActionType.EXTRACT, 'extract'],
  [ActionType.UPLOAD_FILE, 'uploadfile'],
  [ActionType.DOWNLOAD, 'download'],
  [ActionType.FRAME_SWITCH, 'frame-switch'],
  [ActionType.TAB_SWITCH, 'tab-switch'],
  [ActionType.COOKIE_STORAGE, 'cookie-storage'],
  [ActionType.SHORTCUT, 'shortcut'],
  [ActionType.DRAG_DROP, 'drag-drop'],
  [ActionType.GESTURE, 'gesture'],
  [ActionType.NETWORK_MOCK, 'network-mock'],
  [ActionType.ROTATE, 'rotate'],
]);

// =============================================================================
// String → ActionType Mappings
// =============================================================================

/**
 * Map lowercase strings to ActionType enum.
 * Includes aliases for handler dispatch flexibility.
 */
const STRING_TO_ACTION_TYPE_MAP: ReadonlyMap<string, ActionType> = new Map([
  // Core action types (canonical names)
  ['navigate', ActionType.NAVIGATE],
  ['click', ActionType.CLICK],
  ['input', ActionType.INPUT],
  ['wait', ActionType.WAIT],
  ['assert', ActionType.ASSERT],
  ['scroll', ActionType.SCROLL],
  ['select', ActionType.SELECT],
  ['evaluate', ActionType.EVALUATE],
  ['keyboard', ActionType.KEYBOARD],
  ['hover', ActionType.HOVER],
  ['screenshot', ActionType.SCREENSHOT],
  ['focus', ActionType.FOCUS],
  ['blur', ActionType.BLUR],
  ['subflow', ActionType.SUBFLOW],
  ['extract', ActionType.EXTRACT],
  ['download', ActionType.DOWNLOAD],
  ['shortcut', ActionType.SHORTCUT],

  // Upload file variations
  ['uploadfile', ActionType.UPLOAD_FILE],
  ['upload', ActionType.UPLOAD_FILE],

  // Frame switch variations
  ['frame-switch', ActionType.FRAME_SWITCH],
  ['frameswitch', ActionType.FRAME_SWITCH],

  // Tab switch variations
  ['tab-switch', ActionType.TAB_SWITCH],
  ['tabswitch', ActionType.TAB_SWITCH],
  ['tab', ActionType.TAB_SWITCH],
  ['tabs', ActionType.TAB_SWITCH],

  // Cookie storage variations
  ['cookie-storage', ActionType.COOKIE_STORAGE],
  ['cookiestorage', ActionType.COOKIE_STORAGE],

  // Drag drop variations
  ['drag-drop', ActionType.DRAG_DROP],
  ['dragdrop', ActionType.DRAG_DROP],
  ['drag', ActionType.DRAG_DROP],

  // Gesture variations
  ['gesture', ActionType.GESTURE],
  ['swipe', ActionType.GESTURE],
  ['pinch', ActionType.GESTURE],
  ['zoom', ActionType.GESTURE],

  // Network mock variations
  ['network-mock', ActionType.NETWORK_MOCK],
  ['networkmock', ActionType.NETWORK_MOCK],
  ['network', ActionType.NETWORK_MOCK],
  ['mock', ActionType.NETWORK_MOCK],
  ['intercept', ActionType.NETWORK_MOCK],

  // Rotate variations
  ['rotate', ActionType.ROTATE],
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
  return ACTION_TYPE_MAP[normalized] ?? ActionType.CLICK;
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
