/**
 * Handler Instruction Types
 *
 * This module defines the handler-friendly wrapper types around proto CompiledInstruction.
 * These types bridge the gap between the proto wire format and what handlers expect.
 *
 * DESIGN:
 *   - HandlerInstruction wraps proto CompiledInstruction for handler consumption
 *   - toHandlerInstruction() converts from proto to handler format
 *   - getActionType() extracts the action type string from instruction
 *
 * WHY THIS EXISTS:
 *   Handlers need a consistent interface that doesn't change with proto evolution.
 *   This wrapper provides stability while the underlying proto types can evolve.
 *
 * @module proto/instruction
 */

import type {
  CompiledInstruction,
  StepTelemetryDirective,
} from '@vrooli/proto-types/browser-automation-studio/v1/execution/driver_pb';
import {
  ActionType,
  MouseButton,
  KeyboardModifier,
  NavigateWaitEvent,
  WaitState,
  ScrollBehavior,
  KeyAction,
  ExtractType,
  FrameSwitchAction,
  TabSwitchAction,
  CookieOperation,
  StorageType,
  CookieSameSite,
  GestureType,
  SwipeDirection,
  NetworkMockOperation,
  DeviceOrientation,
  type ActionDefinition,
  type AssertParams,
  type BlurParams,
  type ClickParams,
  type CookieStorageParams,
  type DownloadParams,
  type DragDropParams,
  type EvaluateParams,
  type ExtractParams,
  type FocusParams,
  type FrameSwitchParams,
  type GestureParams,
  type HoverParams,
  type InputParams,
  type KeyboardParams,
  type NavigateParams,
  type NetworkMockParams,
  type RotateParams,
  type ScreenshotParams,
  type ScrollParams,
  type SelectParams,
  type ShortcutParams,
  type TabSwitchParams,
  type UploadFileParams,
  type WaitParams,
} from '@vrooli/proto-types/browser-automation-studio/v1/actions/action_pb';
import { AssertionMode } from '@vrooli/proto-types/browser-automation-studio/v1/base/shared_pb';
import { jsonValueMapToPlain, jsonValueToPlain } from './utils';
import { actionTypeToString } from '../recording/action-types';

// =============================================================================
// HandlerInstruction Type
// =============================================================================

/**
 * HandlerInstruction is a handler-friendly wrapper around proto CompiledInstruction.
 *
 * The `action` field contains the typed ActionDefinition with strongly-typed params.
 * Handlers use `requireTypedParams()` to extract and validate params from action.
 *
 * @example
 * ```typescript
 * const typedParams = instruction.action ? getClickParams(instruction.action) : undefined;
 * const params = this.requireTypedParams(typedParams, 'click', instruction.nodeId);
 * ```
 */
export interface HandlerInstruction {
  /** Declared runtime attempt; absent only for direct handler callers. */
  attempt?: number;
  invocationId?: string;
  operationSequence?: number;
  /** Zero-based index in execution order */
  index: number;
  /** Node ID from the workflow definition (UUID) */
  nodeId: string;
  /** Optional preload HTML */
  preloadHtml?: string;
  /** Optional context data */
  context?: Record<string, unknown>;
  /** Optional metadata */
  metadata?: Record<string, string>;
  /**
   * Typed action definition from proto.
   * Contains the ActionType enum and strongly-typed params (navigate, click, etc.)
   * Always populated by the Go API - handlers should use requireTypedParams() to extract.
   */
  action: ActionDefinition;
  /**
   * Per-step telemetry collection intent from the API.
   * Absent means "use driver defaults", which keeps older API builds working.
   */
  telemetry?: StepTelemetryDirective;
}

// =============================================================================
// Conversion Functions
// =============================================================================

/**
 * Convert a proto CompiledInstruction to a HandlerInstruction.
 *
 * Preserves the typed action field, the sole execution representation.
 *
 * @param proto - Proto CompiledInstruction from API
 * @returns HandlerInstruction with typed action field
 */
export function toHandlerInstruction(proto: CompiledInstruction): HandlerInstruction {
	if (!proto.action || proto.action.type === ActionType.UNSPECIFIED) {
		throw new Error(`Instruction ${proto.nodeId} is missing a typed action`);
	}
	return {
		index: proto.index,
		nodeId: proto.nodeId,
		preloadHtml: proto.preloadHtml,
    context: jsonValueMapToPlain(proto.context),
    metadata: proto.metadata ? { ...proto.metadata } : undefined,
    // Typed action - the canonical representation
    action: proto.action,
    telemetry: proto.telemetry,
  };
}

/**
 * Get the handler dispatch key from the typed action.
 */
export function getActionType(instruction: HandlerInstruction): string {
	if (!instruction.action || instruction.action.type === ActionType.UNSPECIFIED) {
		return 'unknown';
	}

	return actionTypeToString(instruction.action.type);
}

// =============================================================================
// Generated action parameter access
// =============================================================================

function getActionParams<T>(action: ActionDefinition, actionType: ActionType): T | undefined {
  if (action.type !== actionType) return undefined;
  const enumName = ActionType[actionType];
  const generatedCase = enumName
    ?.toLowerCase()
    .replace(/_([a-z])/g, (_, letter: string) => letter.toUpperCase());
  // SELECT is represented by the select_option protobuf field, whose oneof
  // case is `selectOption` even though the action enum is simply SELECT.
  const expectedCase = actionType === ActionType.SELECT ? 'selectOption' : generatedCase;
  if (!expectedCase || action.params.case !== expectedCase) return undefined;
  return action.params.value as T;
}

function enumValueToLowerCamel(
  values: Record<number, string>,
  value: number | undefined,
  fallback?: string,
  overrides: Record<string, string> = {},
): string | undefined {
  if (value === undefined) return undefined;
  const name = values[value];
  if (!name || name === 'UNSPECIFIED') return fallback;
  return overrides[name] ?? name.toLowerCase().replace(/_([a-z])/g, (_, letter: string) => letter.toUpperCase());
}

function enumValueToPascalCase(
  values: Record<number, string>,
  value: number | undefined,
  overrides: Record<string, string> = {},
): string | undefined {
  const lowerCamel = enumValueToLowerCamel(values, value, undefined, overrides);
  return lowerCamel ? lowerCamel.charAt(0).toUpperCase() + lowerCamel.slice(1) : undefined;
}

function mouseButtonToString(button: MouseButton | undefined): 'left' | 'right' | 'middle' | undefined {
  return enumValueToLowerCamel(MouseButton, button) as 'left' | 'right' | 'middle' | undefined;
}

type BrowserModifier = 'Control' | 'Shift' | 'Alt' | 'Meta';

function keyboardModifiersToStrings(modifiers: KeyboardModifier[]): BrowserModifier[] {
  return modifiers.map(m => {
    const modifier = enumValueToPascalCase(KeyboardModifier, m, { CTRL: 'control' });
    if (!modifier || modifier === 'Unspecified') throw new Error(`Unsupported keyboard modifier: ${m}`);
    return modifier as BrowserModifier;
  });
}

function navigateWaitEventToString(event: NavigateWaitEvent | undefined): string | undefined {
  return enumValueToLowerCamel(NavigateWaitEvent, event);
}

function waitStateToString(state: WaitState | undefined): string | undefined {
  return enumValueToLowerCamel(WaitState, state);
}

function assertionModeToString(mode: AssertionMode): string {
  switch (mode) {
    case AssertionMode.EXISTS: return 'exists';
    case AssertionMode.NOT_EXISTS: return 'notexists';
    case AssertionMode.VISIBLE: return 'visible';
    case AssertionMode.HIDDEN: return 'hidden';
    case AssertionMode.TEXT_EQUALS: return 'text_equals';
    case AssertionMode.TEXT_CONTAINS: return 'text_contains';
    case AssertionMode.ATTRIBUTE_EQUALS: return 'attribute_equals';
    case AssertionMode.ATTRIBUTE_CONTAINS: return 'attribute_contains';
    default: return 'unsupported';
  }
}

function scrollBehaviorToString(behavior: ScrollBehavior | undefined): string | undefined {
  return enumValueToLowerCamel(ScrollBehavior, behavior);
}

function keyActionToString(action: KeyAction | undefined): string | undefined {
  return enumValueToLowerCamel(KeyAction, action);
}

function extractTypeToString(type: ExtractType | undefined): string | undefined {
  if (type === undefined) return undefined;
  return enumValueToLowerCamel(ExtractType, type, 'text', {
    INNER_HTML: 'innerHTML',
    OUTER_HTML: 'outerHTML',
  }) ?? 'text';
}

function frameSwitchActionToString(action: FrameSwitchAction): string {
  return enumValueToLowerCamel(FrameSwitchAction, action, 'enter') ?? 'enter';
}

function tabSwitchActionToString(action: TabSwitchAction): string {
  return enumValueToLowerCamel(TabSwitchAction, action, 'switch') ?? 'switch';
}

function cookieOperationToString(operation: CookieOperation): string {
  return enumValueToLowerCamel(CookieOperation, operation, 'get') ?? 'get';
}

function storageTypeToString(type: StorageType): string {
  return enumValueToLowerCamel(StorageType, type, 'cookie') ?? 'cookie';
}

function cookieSameSiteToString(sameSite: CookieSameSite | undefined): string | undefined {
  return enumValueToPascalCase(CookieSameSite, sameSite);
}

function gestureTypeToString(type: GestureType): string {
  return enumValueToLowerCamel(GestureType, type, 'swipe') ?? 'swipe';
}

function swipeDirectionToString(direction: SwipeDirection | undefined): string | undefined {
  return enumValueToLowerCamel(SwipeDirection, direction);
}

function networkMockOperationToString(operation: NetworkMockOperation): string {
  return enumValueToLowerCamel(NetworkMockOperation, operation, 'mock') ?? 'mock';
}

function deviceOrientationToString(orientation: DeviceOrientation): string {
  return enumValueToLowerCamel(DeviceOrientation, orientation, 'portrait') ?? 'portrait';
}
/** Extract ClickParams from ActionDefinition */
export function getClickParams(action: ActionDefinition): {
  selector: string;
  button?: 'left' | 'right' | 'middle';
  clickCount?: number;
  delayMs?: number;
  modifiers?: BrowserModifier[];
  force?: boolean;
  timeoutMs?: number;
} | undefined {
  const p = getActionParams<ClickParams>(action, ActionType.CLICK);
  if (p) {
    return {
      selector: p.selector,
      button: mouseButtonToString(p.button),
      clickCount: p.clickCount,
      delayMs: p.delayMs,
      modifiers: p.modifiers.length > 0 ? keyboardModifiersToStrings(p.modifiers) : undefined,
      force: p.force,
      timeoutMs: undefined,
    };
  }
  return undefined;
}

/** Extract InputParams from ActionDefinition */
export function getInputParams(action: ActionDefinition): {
  selector: string;
  value: string;
  isSensitive?: boolean;
  submit?: boolean;
  clearFirst?: boolean;
  delayMs?: number;
  timeoutMs?: number;
} | undefined {
  const p = getActionParams<InputParams>(action, ActionType.INPUT);
  if (p) {
    return {
      selector: p.selector,
      value: p.value,
      isSensitive: p.isSensitive,
      submit: p.submit,
      clearFirst: p.clearFirst,
      delayMs: p.delayMs,
      timeoutMs: undefined,
    };
  }
  return undefined;
}

/** Extract NavigateParams from ActionDefinition */
export function getNavigateParams(action: ActionDefinition): {
  url: string;
  waitForSelector?: string;
  timeoutMs?: number;
  waitUntil?: string;
} | undefined {
  const p = getActionParams<NavigateParams>(action, ActionType.NAVIGATE);
  if (p) {
    return {
      url: p.url,
      waitForSelector: p.waitForSelector,
      timeoutMs: p.timeoutMs,
      waitUntil: navigateWaitEventToString(p.waitUntil),
    };
  }
  return undefined;
}

/** Extract WaitParams from ActionDefinition */
export function getWaitParams(action: ActionDefinition): {
  selector?: string;
  durationMs?: number;
  state?: string;
  timeoutMs?: number;
} | undefined {
  const p = getActionParams<WaitParams>(action, ActionType.WAIT);
  if (p) {
    const result: {
      selector?: string;
      durationMs?: number;
      state?: string;
      timeoutMs?: number;
    } = {
      state: waitStateToString(p.state),
      timeoutMs: p.timeoutMs,
    };
    if (p.waitFor?.case === 'selector') {
      result.selector = p.waitFor.value;
    } else if (p.waitFor?.case === 'durationMs') {
      result.durationMs = p.waitFor.value;
    }
    return result;
  }
  return undefined;
}

/** Extract AssertParams from ActionDefinition */
export function getAssertParams(action: ActionDefinition): {
  selector: string;
  mode: string;
  expected?: unknown;
  negated?: boolean;
  caseSensitive?: boolean;
  attributeName?: string;
  timeoutMs?: number;
  failureMessage?: string;
} | undefined {
  const p = getActionParams<AssertParams>(action, ActionType.ASSERT);
  if (p) {
    return {
      selector: p.selector,
      mode: assertionModeToString(p.mode),
      expected: p.expected ? jsonValueToPlain(p.expected) : undefined,
      negated: p.negated,
      caseSensitive: p.caseSensitive,
      attributeName: p.attributeName,
      timeoutMs: p.timeoutMs,
      failureMessage: p.failureMessage,
    };
  }
  return undefined;
}

/** Extract HoverParams from ActionDefinition */
export function getHoverParams(action: ActionDefinition): {
  selector: string;
  timeoutMs?: number;
} | undefined {
  const p = getActionParams<HoverParams>(action, ActionType.HOVER);
  if (p) {
    return {
      selector: p.selector,
      timeoutMs: p.timeoutMs,
    };
  }
  return undefined;
}

/** Extract FocusParams from ActionDefinition */
export function getFocusParams(action: ActionDefinition): {
  selector: string;
  timeoutMs?: number;
} | undefined {
  const p = getActionParams<FocusParams>(action, ActionType.FOCUS);
  if (p) {
    return {
      selector: p.selector,
      timeoutMs: p.timeoutMs,
    };
  }
  return undefined;
}

/** Extract BlurParams from ActionDefinition */
export function getBlurParams(action: ActionDefinition): {
  selector?: string;
  timeoutMs?: number;
} | undefined {
  const p = getActionParams<BlurParams>(action, ActionType.BLUR);
  if (p) {
    return {
      selector: p.selector,
      timeoutMs: p.timeoutMs,
    };
  }
  return undefined;
}

/** Extract ScrollParams from ActionDefinition */
export function getScrollParams(action: ActionDefinition): {
  selector?: string;
  x?: number;
  y?: number;
  deltaX?: number;
  deltaY?: number;
  behavior?: string;
} | undefined {
  const p = getActionParams<ScrollParams>(action, ActionType.SCROLL);
  if (p) {
    return {
      selector: p.selector,
      x: p.x,
      y: p.y,
      deltaX: p.deltaX,
      deltaY: p.deltaY,
      behavior: scrollBehaviorToString(p.behavior),
    };
  }
  return undefined;
}

/** Extract SelectParams from ActionDefinition */
export function getSelectParams(action: ActionDefinition): {
  selector: string;
  value?: string;
  label?: string;
  index?: number;
  timeoutMs?: number;
} | undefined {
  const p = getActionParams<SelectParams>(action, ActionType.SELECT);
  if (p) {
    const result: {
      selector: string;
      value?: string;
      label?: string;
      index?: number;
      timeoutMs?: number;
    } = {
      selector: p.selector,
      timeoutMs: p.timeoutMs,
    };
    if (p.selectBy?.case === 'value') {
      result.value = p.selectBy.value;
    } else if (p.selectBy?.case === 'label') {
      result.label = p.selectBy.value;
    } else if (p.selectBy?.case === 'index') {
      result.index = p.selectBy.value;
    }
    return result;
  }
  return undefined;
}

/** Extract ScreenshotParams from ActionDefinition */
export function getScreenshotParams(action: ActionDefinition): {
  fullPage?: boolean;
  selector?: string;
  quality?: number;
} | undefined {
  const p = getActionParams<ScreenshotParams>(action, ActionType.SCREENSHOT);
  if (p) {
    return {
      fullPage: p.fullPage,
      selector: p.selector,
      quality: p.quality,
    };
  }
  return undefined;
}

/** Extract EvaluateParams from ActionDefinition */
export function getEvaluateParams(action: ActionDefinition): {
  expression: string;
  storeResult?: string;
} | undefined {
  const p = getActionParams<EvaluateParams>(action, ActionType.EVALUATE);
  if (p) {
    return {
      expression: p.expression,
      storeResult: p.storeResult,
    };
  }
  return undefined;
}

/** Extract KeyboardParams from ActionDefinition */
export function getKeyboardParams(action: ActionDefinition): {
  key?: string;
  keys?: string[];
  modifiers?: BrowserModifier[];
  action?: string;
} | undefined {
  const p = getActionParams<KeyboardParams>(action, ActionType.KEYBOARD);
  if (p) {
    return {
      key: p.key,
      keys: p.keys.length > 0 ? [...p.keys] : undefined,
      modifiers: p.modifiers.length > 0 ? keyboardModifiersToStrings(p.modifiers) : undefined,
      action: keyActionToString(p.action),
    };
  }
  return undefined;
}

/** Extract ExtractParams from ActionDefinition */
export function getExtractParams(action: ActionDefinition): {
  selector: string;
  extractType?: string;
  attributeName?: string;
  propertyName?: string;
  storeAs?: string;
  timeoutMs?: number;
} | undefined {
  const p = getActionParams<ExtractParams>(action, ActionType.EXTRACT);
  if (p) {
    return {
      selector: p.selector,
      extractType: extractTypeToString(p.extractType),
      attributeName: p.attributeName,
      propertyName: p.propertyName,
      storeAs: p.storeAs,
      timeoutMs: p.timeoutMs,
    };
  }
  return undefined;
}

/** Extract UploadFileParams from ActionDefinition */
export function getUploadFileParams(action: ActionDefinition): {
  selector: string;
  filePaths: string[];
  timeoutMs?: number;
} | undefined {
  const p = getActionParams<UploadFileParams>(action, ActionType.UPLOAD_FILE);
  if (p) {
    return {
      selector: p.selector,
      filePaths: [...p.filePaths],
      timeoutMs: p.timeoutMs,
    };
  }
  return undefined;
}

/** Extract DownloadParams from ActionDefinition */
export function getDownloadParams(action: ActionDefinition): {
  selector?: string;
  url?: string;
  savePath?: string;
  timeoutMs?: number;
} | undefined {
  const p = getActionParams<DownloadParams>(action, ActionType.DOWNLOAD);
  if (p) {
    return {
      selector: p.selector,
      url: p.url,
      savePath: p.savePath,
      timeoutMs: p.timeoutMs,
    };
  }
  return undefined;
}

/** Extract FrameSwitchParams from ActionDefinition */
export function getFrameSwitchParams(action: ActionDefinition): {
  action: string;
  selector?: string;
  frameId?: string;
  frameUrl?: string;
  timeoutMs?: number;
} | undefined {
  const p = getActionParams<FrameSwitchParams>(action, ActionType.FRAME_SWITCH);
  if (p) {
    return {
      action: frameSwitchActionToString(p.action),
      selector: p.selector,
      frameId: p.frameId,
      frameUrl: p.frameUrl,
      timeoutMs: p.timeoutMs,
    };
  }
  return undefined;
}

/** Extract TabSwitchParams from ActionDefinition */
export function getTabSwitchParams(action: ActionDefinition): {
  action: string;
  url?: string;
  index?: number;
  title?: string;
  urlPattern?: string;
} | undefined {
  const p = getActionParams<TabSwitchParams>(action, ActionType.TAB_SWITCH);
  if (p) {
    return {
      action: tabSwitchActionToString(p.action),
      url: p.url,
      index: p.index,
      title: p.title,
      urlPattern: p.urlPattern,
    };
  }
  return undefined;
}

/** Extract CookieStorageParams from ActionDefinition */
export function getCookieStorageParams(action: ActionDefinition): {
  operation: string;
  storageType: string;
  key?: string;
  name?: string;
  value?: string;
  cookieOptions?: {
    domain?: string;
    path?: string;
    expires?: number;
    httpOnly?: boolean;
    secure?: boolean;
    sameSite?: string;
  };
} | undefined {
  const p = getActionParams<CookieStorageParams>(action, ActionType.COOKIE_STORAGE);
  if (p) {
    return {
      operation: cookieOperationToString(p.operation),
      storageType: storageTypeToString(p.storageType),
      key: p.key,
      name: p.key,
      value: p.value,
      cookieOptions: p.cookieOptions ? {
        domain: p.cookieOptions.domain,
        path: p.cookieOptions.path,
        expires: p.cookieOptions.expires !== undefined ? Number(p.cookieOptions.expires) : undefined,
        httpOnly: p.cookieOptions.httpOnly,
        secure: p.cookieOptions.secure,
        sameSite: cookieSameSiteToString(p.cookieOptions.sameSite),
      } : undefined,
    };
  }
  return undefined;
}

/** Extract ShortcutParams from ActionDefinition */
export function getShortcutParams(action: ActionDefinition): {
  shortcut: string;
  selector?: string;
} | undefined {
  const p = getActionParams<ShortcutParams>(action, ActionType.SHORTCUT);
  if (p) {
    return {
      shortcut: p.shortcut,
      selector: p.selector,
    };
  }
  return undefined;
}

/** Extract DragDropParams from ActionDefinition */
export function getDragDropParams(action: ActionDefinition): {
  sourceSelector: string;
  targetSelector?: string;
  offsetX?: number;
  offsetY?: number;
  targetOffsetX?: number;
  targetOffsetY?: number;
  steps?: number;
  delayMs?: number;
  timeoutMs?: number;
} | undefined {
  const p = getActionParams<DragDropParams>(action, ActionType.DRAG_DROP);
  if (p) {
    return {
      sourceSelector: p.sourceSelector,
      targetSelector: p.targetSelector,
      offsetX: p.offsetX,
      offsetY: p.offsetY,
      targetOffsetX: p.targetOffsetX,
      targetOffsetY: p.targetOffsetY,
      steps: p.steps,
      delayMs: p.delayMs,
      timeoutMs: p.timeoutMs,
    };
  }
  return undefined;
}

/** Extract GestureParams from ActionDefinition */
export function getGestureParams(action: ActionDefinition): {
  gestureType: string;
  selector?: string;
  direction?: string;
  distance?: number;
  scale?: number;
  durationMs?: number;
  steps?: number;
  stepDelayMs?: number;
  traceLabel?: string;
  idleAfterMs?: number;
  wheelDeltaY?: number;
  ctrlKey?: boolean;
} | undefined {
  const p = getActionParams<GestureParams>(action, ActionType.GESTURE);
  if (p) {
    return {
      gestureType: gestureTypeToString(p.gestureType),
      selector: p.selector,
      direction: swipeDirectionToString(p.direction),
      distance: p.distance,
      scale: p.scale,
      durationMs: p.durationMs,
      steps: p.steps,
      stepDelayMs: p.stepDelayMs,
      traceLabel: p.traceLabel,
      idleAfterMs: p.idleAfterMs,
      wheelDeltaY: p.wheelDeltaY,
      ctrlKey: p.ctrlKey,
    };
  }
  return undefined;
}

/** Extract NetworkMockParams from ActionDefinition */
export function getNetworkMockParams(action: ActionDefinition): {
  operation: string;
  urlPattern: string;
  method?: string;
  statusCode?: number;
  headers?: Record<string, string>;
  body?: string;
  delayMs?: number;
} | undefined {
  const p = getActionParams<NetworkMockParams>(action, ActionType.NETWORK_MOCK);
  if (p) {
    return {
      operation: networkMockOperationToString(p.operation),
      urlPattern: p.urlPattern,
      method: p.method,
      statusCode: p.statusCode,
      headers: Object.keys(p.headers).length > 0 ? { ...p.headers } : undefined,
      body: p.body,
      delayMs: p.delayMs,
    };
  }
  return undefined;
}

/** Extract RotateParams from ActionDefinition */
export function getRotateParams(action: ActionDefinition): {
  orientation: string;
  angle?: number;
} | undefined {
  const p = getActionParams<RotateParams>(action, ActionType.ROTATE);
  if (p) {
    return {
      orientation: deviceOrientationToString(p.orientation),
      angle: p.angle,
    };
  }
  return undefined;
}
