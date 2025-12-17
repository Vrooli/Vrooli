/**
 * Proto Param Extractors
 *
 * Extract typed parameters from proto ActionDefinition messages.
 * These functions provide type-safe access to action params for handlers.
 *
 * DESIGN:
 * - Each get*Params function extracts the typed params from a specific action type
 * - Enum values are converted to strings for handler compatibility
 * - Returns undefined if the action doesn't match the expected type
 *
 * CHANGE AXIS: Adding New Action Types
 * When adding a new action type:
 * 1. Add the proto message in action.proto
 * 2. Create a get*Params function following the existing pattern
 * 3. Export from this file
 * 4. Re-export from proto/index.ts
 */

import {
  type ActionDefinition,
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
} from '@vrooli/proto-types/browser-automation-studio/v1/actions/action_pb';
import { AssertionMode } from '@vrooli/proto-types/browser-automation-studio/v1/base/shared_pb';
import { jsonValueToPlain } from './utils';

// =============================================================================
// ENUM TO STRING CONVERTERS (internal helpers)
// =============================================================================

function mouseButtonToString(button: MouseButton | undefined): string | undefined {
  if (button === undefined) return undefined;
  switch (button) {
    case MouseButton.LEFT: return 'left';
    case MouseButton.RIGHT: return 'right';
    case MouseButton.MIDDLE: return 'middle';
    default: return undefined;
  }
}

function keyboardModifiersToStrings(modifiers: KeyboardModifier[]): string[] {
  return modifiers.map(m => {
    switch (m) {
      case KeyboardModifier.CTRL: return 'ctrl';
      case KeyboardModifier.SHIFT: return 'shift';
      case KeyboardModifier.ALT: return 'alt';
      case KeyboardModifier.META: return 'meta';
      default: return '';
    }
  }).filter(Boolean);
}

function navigateWaitEventToString(event: NavigateWaitEvent | undefined): string | undefined {
  if (event === undefined) return undefined;
  switch (event) {
    case NavigateWaitEvent.LOAD: return 'load';
    case NavigateWaitEvent.DOMCONTENTLOADED: return 'domcontentloaded';
    case NavigateWaitEvent.NETWORKIDLE: return 'networkidle';
    default: return undefined;
  }
}

function waitStateToString(state: WaitState | undefined): string | undefined {
  if (state === undefined) return undefined;
  switch (state) {
    case WaitState.ATTACHED: return 'attached';
    case WaitState.DETACHED: return 'detached';
    case WaitState.VISIBLE: return 'visible';
    case WaitState.HIDDEN: return 'hidden';
    default: return undefined;
  }
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
    default: return 'exists';
  }
}

function scrollBehaviorToString(behavior: ScrollBehavior | undefined): string | undefined {
  if (behavior === undefined) return undefined;
  switch (behavior) {
    case ScrollBehavior.AUTO: return 'auto';
    case ScrollBehavior.SMOOTH: return 'smooth';
    default: return undefined;
  }
}

function keyActionToString(action: KeyAction | undefined): string | undefined {
  if (action === undefined) return undefined;
  switch (action) {
    case KeyAction.PRESS: return 'press';
    case KeyAction.DOWN: return 'down';
    case KeyAction.UP: return 'up';
    default: return undefined;
  }
}

function extractTypeToString(type: ExtractType | undefined): string | undefined {
  if (type === undefined) return undefined;
  switch (type) {
    case ExtractType.TEXT: return 'text';
    case ExtractType.INNER_HTML: return 'innerHTML';
    case ExtractType.OUTER_HTML: return 'outerHTML';
    case ExtractType.ATTRIBUTE: return 'attribute';
    case ExtractType.PROPERTY: return 'property';
    case ExtractType.VALUE: return 'value';
    default: return 'text';
  }
}

function frameSwitchActionToString(action: FrameSwitchAction): string {
  switch (action) {
    case FrameSwitchAction.ENTER: return 'enter';
    case FrameSwitchAction.PARENT: return 'parent';
    case FrameSwitchAction.EXIT: return 'exit';
    default: return 'enter';
  }
}

function tabSwitchActionToString(action: TabSwitchAction): string {
  switch (action) {
    case TabSwitchAction.OPEN: return 'open';
    case TabSwitchAction.SWITCH: return 'switch';
    case TabSwitchAction.CLOSE: return 'close';
    case TabSwitchAction.LIST: return 'list';
    default: return 'switch';
  }
}

function cookieOperationToString(operation: CookieOperation): string {
  switch (operation) {
    case CookieOperation.GET: return 'get';
    case CookieOperation.SET: return 'set';
    case CookieOperation.DELETE: return 'delete';
    case CookieOperation.CLEAR: return 'clear';
    default: return 'get';
  }
}

function storageTypeToString(type: StorageType): string {
  switch (type) {
    case StorageType.COOKIE: return 'cookie';
    case StorageType.LOCAL_STORAGE: return 'localStorage';
    case StorageType.SESSION_STORAGE: return 'sessionStorage';
    default: return 'cookie';
  }
}

function cookieSameSiteToString(sameSite: CookieSameSite | undefined): string | undefined {
  if (sameSite === undefined) return undefined;
  switch (sameSite) {
    case CookieSameSite.STRICT: return 'Strict';
    case CookieSameSite.LAX: return 'Lax';
    case CookieSameSite.NONE: return 'None';
    default: return undefined;
  }
}

function gestureTypeToString(type: GestureType): string {
  switch (type) {
    case GestureType.SWIPE: return 'swipe';
    case GestureType.PINCH: return 'pinch';
    case GestureType.ZOOM: return 'zoom';
    case GestureType.LONG_PRESS: return 'longPress';
    case GestureType.DOUBLE_TAP: return 'doubleTap';
    default: return 'swipe';
  }
}

function swipeDirectionToString(direction: SwipeDirection | undefined): string | undefined {
  if (direction === undefined) return undefined;
  switch (direction) {
    case SwipeDirection.UP: return 'up';
    case SwipeDirection.DOWN: return 'down';
    case SwipeDirection.LEFT: return 'left';
    case SwipeDirection.RIGHT: return 'right';
    default: return undefined;
  }
}

function networkMockOperationToString(operation: NetworkMockOperation): string {
  switch (operation) {
    case NetworkMockOperation.MOCK: return 'mock';
    case NetworkMockOperation.BLOCK: return 'block';
    case NetworkMockOperation.MODIFY_REQUEST: return 'modifyRequest';
    case NetworkMockOperation.MODIFY_RESPONSE: return 'modifyResponse';
    case NetworkMockOperation.CLEAR: return 'clear';
    default: return 'mock';
  }
}

function deviceOrientationToString(orientation: DeviceOrientation): string {
  switch (orientation) {
    case DeviceOrientation.PORTRAIT: return 'portrait';
    case DeviceOrientation.LANDSCAPE: return 'landscape';
    default: return 'portrait';
  }
}

// =============================================================================
// TYPED PARAM EXTRACTORS
// =============================================================================

/**
 * Extract typed params from ActionDefinition.
 *
 * These functions provide type-safe access to action params.
 * Handlers use these via requireTypedParams() in BaseHandler.
 *
 * NOTE: Enums are converted to strings for handler compatibility.
 */

/** Extract ClickParams from ActionDefinition */
export function getClickParams(action: ActionDefinition): {
  selector: string;
  button?: string;
  clickCount?: number;
  delayMs?: number;
  modifiers?: string[];
  force?: boolean;
  timeoutMs?: number;
} | undefined {
  if (action.params?.case === 'click' && action.params.value) {
    const p = action.params.value;
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
  if (action.params?.case === 'input' && action.params.value) {
    const p = action.params.value;
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
  if (action.params?.case === 'navigate' && action.params.value) {
    const p = action.params.value;
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
  if (action.params?.case === 'wait' && action.params.value) {
    const p = action.params.value;
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
} | undefined {
  if (action.params?.case === 'assert' && action.params.value) {
    const p = action.params.value;
    return {
      selector: p.selector,
      mode: assertionModeToString(p.mode),
      expected: p.expected ? jsonValueToPlain(p.expected) : undefined,
      negated: p.negated,
      caseSensitive: p.caseSensitive,
      attributeName: p.attributeName,
      timeoutMs: p.timeoutMs,
    };
  }
  return undefined;
}

/** Extract HoverParams from ActionDefinition */
export function getHoverParams(action: ActionDefinition): {
  selector: string;
  timeoutMs?: number;
} | undefined {
  if (action.params?.case === 'hover' && action.params.value) {
    const p = action.params.value;
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
  if (action.params?.case === 'focus' && action.params.value) {
    const p = action.params.value;
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
  if (action.params?.case === 'blur' && action.params.value) {
    const p = action.params.value;
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
  if (action.params?.case === 'scroll' && action.params.value) {
    const p = action.params.value;
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
  if (action.params?.case === 'selectOption' && action.params.value) {
    const p = action.params.value;
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
  if (action.params?.case === 'screenshot' && action.params.value) {
    const p = action.params.value;
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
  if (action.params?.case === 'evaluate' && action.params.value) {
    const p = action.params.value;
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
  modifiers?: string[];
  action?: string;
} | undefined {
  if (action.params?.case === 'keyboard' && action.params.value) {
    const p = action.params.value;
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
  if (action.params?.case === 'extract' && action.params.value) {
    const p = action.params.value;
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
  if (action.params?.case === 'uploadFile' && action.params.value) {
    const p = action.params.value;
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
  if (action.params?.case === 'download' && action.params.value) {
    const p = action.params.value;
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
  if (action.params?.case === 'frameSwitch' && action.params.value) {
    const p = action.params.value;
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
  if (action.params?.case === 'tabSwitch' && action.params.value) {
    const p = action.params.value;
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
  if (action.params?.case === 'cookieStorage' && action.params.value) {
    const p = action.params.value;
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
  if (action.params?.case === 'shortcut' && action.params.value) {
    const p = action.params.value;
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
  steps?: number;
  delayMs?: number;
  timeoutMs?: number;
} | undefined {
  if (action.params?.case === 'dragDrop' && action.params.value) {
    const p = action.params.value;
    return {
      sourceSelector: p.sourceSelector,
      targetSelector: p.targetSelector,
      offsetX: p.offsetX,
      offsetY: p.offsetY,
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
} | undefined {
  if (action.params?.case === 'gesture' && action.params.value) {
    const p = action.params.value;
    return {
      gestureType: gestureTypeToString(p.gestureType),
      selector: p.selector,
      direction: swipeDirectionToString(p.direction),
      distance: p.distance,
      scale: p.scale,
      durationMs: p.durationMs,
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
  if (action.params?.case === 'networkMock' && action.params.value) {
    const p = action.params.value;
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
  if (action.params?.case === 'rotate' && action.params.value) {
    const p = action.params.value;
    return {
      orientation: deviceOrientationToString(p.orientation),
      angle: p.angle,
    };
  }
  return undefined;
}
