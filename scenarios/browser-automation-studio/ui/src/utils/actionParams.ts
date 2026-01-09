/**
 * Action Params Utilities
 *
 * Helpers for extracting and updating typed params from V2 ActionDefinition.
 * These utilities enable the UI to work natively with the V2 action format.
 */

import {
  ACTION_TYPES,
  type ActionDefinition,
  type ActionTypeValue,
  type ActionMetadata,
  type NavigateParams,
  type ClickParams,
  type InputParams,
  type WaitParams,
  type AssertParams,
  type ScrollParams,
  type SelectParams,
  type EvaluateParams,
  type KeyboardParams,
  type HoverParams,
  type ScreenshotParams,
  type FocusParams,
  type BlurParams,
  type SubflowParams,
  type SetVariableParams,
  type LoopParams,
  type ConditionalParams,
  type ExtractParams,
  type UploadFileParams,
  type DownloadParams,
  type FrameSwitchParams,
  type TabSwitchParams,
  type CookieStorageParams,
  type ShortcutParams,
  type DragDropParams,
  type GestureParams,
  type NetworkMockParams,
  type RotateParams,
} from './actionBuilder';

// Re-export types for convenience
export type {
  ActionDefinition,
  ActionTypeValue,
  ActionMetadata,
  NavigateParams,
  ClickParams,
  InputParams,
  WaitParams,
  AssertParams,
  ScrollParams,
  SelectParams,
  EvaluateParams,
  KeyboardParams,
  HoverParams,
  ScreenshotParams,
  FocusParams,
  BlurParams,
  SubflowParams,
  SetVariableParams,
  LoopParams,
  ConditionalParams,
  ExtractParams,
  UploadFileParams,
  DownloadParams,
  FrameSwitchParams,
  TabSwitchParams,
  CookieStorageParams,
  ShortcutParams,
  DragDropParams,
  GestureParams,
  NetworkMockParams,
  RotateParams,
};

/**
 * Maps ACTION_TYPE_* enum values to their corresponding params field name.
 * e.g., ACTION_TYPE_CLICK -> 'click', ACTION_TYPE_INPUT -> 'input'
 */
export function getParamsFieldName(actionType: ActionTypeValue): string | null {
  switch (actionType) {
    case ACTION_TYPES.NAVIGATE:
      return 'navigate';
    case ACTION_TYPES.CLICK:
      return 'click';
    case ACTION_TYPES.INPUT:
      return 'input';
    case ACTION_TYPES.WAIT:
      return 'wait';
    case ACTION_TYPES.ASSERT:
      return 'assert';
    case ACTION_TYPES.SCROLL:
      return 'scroll';
    case ACTION_TYPES.SELECT:
      return 'selectOption';
    case ACTION_TYPES.EVALUATE:
      return 'evaluate';
    case ACTION_TYPES.KEYBOARD:
      return 'keyboard';
    case ACTION_TYPES.HOVER:
      return 'hover';
    case ACTION_TYPES.SCREENSHOT:
      return 'screenshot';
    case ACTION_TYPES.FOCUS:
      return 'focus';
    case ACTION_TYPES.BLUR:
      return 'blur';
    case ACTION_TYPES.SUBFLOW:
      return 'subflow';
    // Control flow
    case ACTION_TYPES.SET_VARIABLE:
      return 'setVariable';
    case ACTION_TYPES.LOOP:
      return 'loop';
    case ACTION_TYPES.CONDITIONAL:
      return 'conditional';
    // Data extraction
    case ACTION_TYPES.EXTRACT:
      return 'extract';
    // File operations
    case ACTION_TYPES.UPLOAD_FILE:
      return 'uploadFile';
    case ACTION_TYPES.DOWNLOAD:
      return 'download';
    // Context switching
    case ACTION_TYPES.FRAME_SWITCH:
      return 'frameSwitch';
    case ACTION_TYPES.TAB_SWITCH:
      return 'tabSwitch';
    // Storage
    case ACTION_TYPES.COOKIE_STORAGE:
      return 'cookieStorage';
    // Input actions
    case ACTION_TYPES.SHORTCUT:
      return 'shortcut';
    case ACTION_TYPES.DRAG_DROP:
      return 'dragDrop';
    case ACTION_TYPES.GESTURE:
      return 'gesture';
    // Network
    case ACTION_TYPES.NETWORK_MOCK:
      return 'networkMock';
    // Device
    case ACTION_TYPES.ROTATE:
      return 'rotate';
    default:
      return null;
  }
}

/**
 * Type mapping for action types to their params interfaces.
 */
export type ActionParamsMap = {
  [ACTION_TYPES.NAVIGATE]: NavigateParams;
  [ACTION_TYPES.CLICK]: ClickParams;
  [ACTION_TYPES.INPUT]: InputParams;
  [ACTION_TYPES.WAIT]: WaitParams;
  [ACTION_TYPES.ASSERT]: AssertParams;
  [ACTION_TYPES.SCROLL]: ScrollParams;
  [ACTION_TYPES.SELECT]: SelectParams;
  [ACTION_TYPES.EVALUATE]: EvaluateParams;
  [ACTION_TYPES.KEYBOARD]: KeyboardParams;
  [ACTION_TYPES.HOVER]: HoverParams;
  [ACTION_TYPES.SCREENSHOT]: ScreenshotParams;
  [ACTION_TYPES.FOCUS]: FocusParams;
  [ACTION_TYPES.BLUR]: BlurParams;
  [ACTION_TYPES.SUBFLOW]: SubflowParams;
  // Control flow
  [ACTION_TYPES.SET_VARIABLE]: SetVariableParams;
  [ACTION_TYPES.LOOP]: LoopParams;
  [ACTION_TYPES.CONDITIONAL]: ConditionalParams;
  // Data extraction
  [ACTION_TYPES.EXTRACT]: ExtractParams;
  // File operations
  [ACTION_TYPES.UPLOAD_FILE]: UploadFileParams;
  [ACTION_TYPES.DOWNLOAD]: DownloadParams;
  // Context switching
  [ACTION_TYPES.FRAME_SWITCH]: FrameSwitchParams;
  [ACTION_TYPES.TAB_SWITCH]: TabSwitchParams;
  // Storage
  [ACTION_TYPES.COOKIE_STORAGE]: CookieStorageParams;
  // Input actions
  [ACTION_TYPES.SHORTCUT]: ShortcutParams;
  [ACTION_TYPES.DRAG_DROP]: DragDropParams;
  [ACTION_TYPES.GESTURE]: GestureParams;
  // Network
  [ACTION_TYPES.NETWORK_MOCK]: NetworkMockParams;
  // Device
  [ACTION_TYPES.ROTATE]: RotateParams;
  [ACTION_TYPES.UNSPECIFIED]: Record<string, unknown>;
};

/**
 * Extracts typed params from an ActionDefinition based on its type.
 * Returns undefined if action is undefined or has no params for its type.
 */
export function extractParams<T>(action: ActionDefinition | undefined): T | undefined {
  if (!action?.type) {
    return undefined;
  }

  const fieldName = getParamsFieldName(action.type);
  if (!fieldName) {
    return undefined;
  }

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  return (action as any)[fieldName] as T | undefined;
}

/**
 * Extracts metadata from an ActionDefinition.
 */
export function extractMetadata(action: ActionDefinition | undefined): ActionMetadata | undefined {
  return action?.metadata;
}

/**
 * Creates an updated ActionDefinition with new params merged in.
 * Preserves existing params fields and only updates the relevant one.
 */
export function updateActionParams<T>(
  action: ActionDefinition | undefined,
  updates: Partial<T>,
): ActionDefinition {
  if (!action?.type) {
    throw new Error('Cannot update params: action has no type');
  }

  const fieldName = getParamsFieldName(action.type);
  if (!fieldName) {
    throw new Error(`Cannot update params: unknown action type ${action.type}`);
  }

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const existingParams = (action as any)[fieldName] as Record<string, unknown> | undefined;
  const updatedParams = { ...existingParams, ...updates } as Record<string, unknown>;

  // Remove undefined values from params
  for (const key of Object.keys(updatedParams)) {
    if (updatedParams[key] === undefined) {
      delete updatedParams[key];
    }
  }

  return {
    ...action,
    [fieldName]: updatedParams,
  };
}

/**
 * Creates an updated ActionDefinition with new metadata merged in.
 */
export function updateActionMetadata(
  action: ActionDefinition | undefined,
  updates: Partial<ActionMetadata>,
): ActionDefinition {
  if (!action?.type) {
    throw new Error('Cannot update metadata: action has no type');
  }

  const existingMetadata = action.metadata ?? {};
  const updatedMetadata = { ...existingMetadata, ...updates };

  // Remove undefined values
  for (const key of Object.keys(updatedMetadata)) {
    if ((updatedMetadata as Record<string, unknown>)[key] === undefined) {
      delete (updatedMetadata as Record<string, unknown>)[key];
    }
  }

  return {
    ...action,
    metadata: Object.keys(updatedMetadata).length > 0 ? updatedMetadata : undefined,
  };
}

/**
 * Creates a new ActionDefinition with a specific type and optional params.
 */
export function createAction<T extends ActionTypeValue>(
  type: T,
  params?: Partial<ActionParamsMap[T]>,
  metadata?: Partial<ActionMetadata>,
): ActionDefinition {
  const fieldName = getParamsFieldName(type);

  const action: ActionDefinition = { type };

  if (fieldName && params) {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (action as any)[fieldName] = params;
  }

  if (metadata && Object.keys(metadata).length > 0) {
    action.metadata = metadata as ActionMetadata;
  }

  return action;
}

/**
 * Gets the label from action metadata, or returns a default based on action type.
 */
export function getActionLabel(action: ActionDefinition | undefined): string {
  if (action?.metadata?.label) {
    return action.metadata.label;
  }

  if (!action?.type) {
    return 'Unknown';
  }

  // Default labels based on action type
  switch (action.type) {
    case ACTION_TYPES.NAVIGATE:
      return 'Navigate';
    case ACTION_TYPES.CLICK:
      return 'Click';
    case ACTION_TYPES.INPUT:
      return 'Type';
    case ACTION_TYPES.WAIT:
      return 'Wait';
    case ACTION_TYPES.ASSERT:
      return 'Assert';
    case ACTION_TYPES.SCROLL:
      return 'Scroll';
    case ACTION_TYPES.SELECT:
      return 'Select';
    case ACTION_TYPES.EVALUATE:
      return 'Evaluate';
    case ACTION_TYPES.KEYBOARD:
      return 'Keyboard';
    case ACTION_TYPES.HOVER:
      return 'Hover';
    case ACTION_TYPES.SCREENSHOT:
      return 'Screenshot';
    case ACTION_TYPES.FOCUS:
      return 'Focus';
    case ACTION_TYPES.BLUR:
      return 'Blur';
    case ACTION_TYPES.SUBFLOW:
      return 'Subflow';
    // Control flow
    case ACTION_TYPES.SET_VARIABLE:
      return 'Set Variable';
    case ACTION_TYPES.LOOP:
      return 'Loop';
    case ACTION_TYPES.CONDITIONAL:
      return 'Conditional';
    // Data extraction
    case ACTION_TYPES.EXTRACT:
      return 'Extract';
    // File operations
    case ACTION_TYPES.UPLOAD_FILE:
      return 'Upload File';
    case ACTION_TYPES.DOWNLOAD:
      return 'Download';
    // Context switching
    case ACTION_TYPES.FRAME_SWITCH:
      return 'Frame Switch';
    case ACTION_TYPES.TAB_SWITCH:
      return 'Tab Switch';
    // Storage
    case ACTION_TYPES.COOKIE_STORAGE:
      return 'Cookie/Storage';
    // Input actions
    case ACTION_TYPES.SHORTCUT:
      return 'Shortcut';
    case ACTION_TYPES.DRAG_DROP:
      return 'Drag & Drop';
    case ACTION_TYPES.GESTURE:
      return 'Gesture';
    // Network
    case ACTION_TYPES.NETWORK_MOCK:
      return 'Network Mock';
    // Device
    case ACTION_TYPES.ROTATE:
      return 'Rotate';
    default:
      return 'Unknown';
  }
}
