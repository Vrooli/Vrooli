/**
 * Action Builder Utility
 *
 * Converts V1-style node type/data to V2 action definitions.
 * This enables the UI to create nodes with proper V2 action fields,
 * reducing reliance on backend V1-to-V2 conversion.
 */

/**
 * V2 Action types that match the proto ActionType enum.
 */
export const ACTION_TYPES = {
  UNSPECIFIED: 'ACTION_TYPE_UNSPECIFIED',
  NAVIGATE: 'ACTION_TYPE_NAVIGATE',
  CLICK: 'ACTION_TYPE_CLICK',
  INPUT: 'ACTION_TYPE_INPUT',
  WAIT: 'ACTION_TYPE_WAIT',
  ASSERT: 'ACTION_TYPE_ASSERT',
  SCROLL: 'ACTION_TYPE_SCROLL',
  SELECT: 'ACTION_TYPE_SELECT',
  EVALUATE: 'ACTION_TYPE_EVALUATE',
  KEYBOARD: 'ACTION_TYPE_KEYBOARD',
  HOVER: 'ACTION_TYPE_HOVER',
  SCREENSHOT: 'ACTION_TYPE_SCREENSHOT',
  FOCUS: 'ACTION_TYPE_FOCUS',
  BLUR: 'ACTION_TYPE_BLUR',
  SUBFLOW: 'ACTION_TYPE_SUBFLOW',
} as const;

export type ActionTypeValue = (typeof ACTION_TYPES)[keyof typeof ACTION_TYPES];

/**
 * Maps V1 node type strings to V2 ACTION_TYPE_ enum values.
 */
export function nodeTypeToActionType(nodeType: string): ActionTypeValue {
  const normalized = nodeType.toLowerCase().trim();
  switch (normalized) {
    case 'navigate':
    case 'goto':
      return ACTION_TYPES.NAVIGATE;
    case 'click':
      return ACTION_TYPES.CLICK;
    case 'input':
    case 'type':
    case 'fill':
      return ACTION_TYPES.INPUT;
    case 'wait':
      return ACTION_TYPES.WAIT;
    case 'assert':
      return ACTION_TYPES.ASSERT;
    case 'scroll':
      return ACTION_TYPES.SCROLL;
    case 'select':
      return ACTION_TYPES.SELECT;
    case 'evaluate':
    case 'eval':
      return ACTION_TYPES.EVALUATE;
    case 'keyboard':
    case 'keypress':
      return ACTION_TYPES.KEYBOARD;
    case 'hover':
      return ACTION_TYPES.HOVER;
    case 'screenshot':
      return ACTION_TYPES.SCREENSHOT;
    case 'focus':
      return ACTION_TYPES.FOCUS;
    case 'blur':
      return ACTION_TYPES.BLUR;
    case 'subflow':
      return ACTION_TYPES.SUBFLOW;
    default:
      return ACTION_TYPES.UNSPECIFIED;
  }
}

/**
 * Maps V2 ACTION_TYPE_ enum values back to V1 node type strings.
 */
export function actionTypeToNodeType(actionType: ActionTypeValue): string {
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
      return 'select';
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
    default:
      return 'unknown';
  }
}

/**
 * V2 Action Definition structure matching the proto ActionDefinition.
 */
export interface ActionDefinition {
  type: ActionTypeValue;
  navigate?: NavigateParams;
  click?: ClickParams;
  input?: InputParams;
  wait?: WaitParams;
  assert?: AssertParams;
  scroll?: ScrollParams;
  selectOption?: SelectParams;
  evaluate?: EvaluateParams;
  keyboard?: KeyboardParams;
  hover?: HoverParams;
  screenshot?: ScreenshotParams;
  focus?: FocusParams;
  blur?: BlurParams;
  subflow?: SubflowParams;
  metadata?: ActionMetadata;
}

export interface NavigateParams {
  url?: string;
  waitForSelector?: string;
  timeoutMs?: number;
  waitUntil?: string;
  /** Navigate destination type: 'url' or 'scenario' */
  destinationType?: string;
  /** Scenario name for scenario-based navigation */
  scenario?: string;
  /** Path within scenario (e.g., '/dashboard') */
  scenarioPath?: string;
}

export interface ClickParams {
  selector: string;
  button?: string;
  clickCount?: number;
  delayMs?: number;
  modifiers?: string[];
  force?: boolean;
}

export interface InputParams {
  selector: string;
  value: string;
  isSensitive?: boolean;
  submit?: boolean;
  clearFirst?: boolean;
  delayMs?: number;
}

export interface WaitParams {
  durationMs?: number;
  selector?: string;
  state?: string;
  timeoutMs?: number;
}

export interface AssertParams {
  selector: string;
  mode: string;
  expected?: unknown;
  negated?: boolean;
  caseSensitive?: boolean;
  attributeName?: string;
}

export interface ScrollParams {
  selector?: string;
  x?: number;
  y?: number;
  deltaX?: number;
  deltaY?: number;
  behavior?: string;
}

export interface SelectParams {
  selector: string;
  value?: string;
  label?: string;
  index?: number;
  timeoutMs?: number;
}

export interface EvaluateParams {
  expression: string;
  storeResult?: string;
}

export interface KeyboardParams {
  key?: string;
  keys?: string[];
  modifiers?: string[];
  action?: string;
}

export interface HoverParams {
  selector: string;
  timeoutMs?: number;
}

export interface ScreenshotParams {
  fullPage?: boolean;
  selector?: string;
  quality?: number;
}

export interface FocusParams {
  selector: string;
  timeoutMs?: number;
}

export interface BlurParams {
  selector?: string;
  timeoutMs?: number;
}

export interface SubflowParams {
  workflowId?: string;
  workflowPath?: string;
  workflowVersion?: number;
  parameters?: Record<string, unknown>;
}

export interface ActionMetadata {
  label?: string;
  recordedAt?: string;
  originalEvent?: string;
}

/**
 * Helper to safely get string value from data
 */
function getString(data: Record<string, unknown>, key: string): string | undefined {
  const value = data[key];
  return typeof value === 'string' ? value : undefined;
}

/**
 * Helper to safely get number value from data
 */
function getNumber(data: Record<string, unknown>, key: string): number | undefined {
  const value = data[key];
  return typeof value === 'number' ? value : undefined;
}

/**
 * Helper to safely get boolean value from data
 */
function getBool(data: Record<string, unknown>, key: string): boolean | undefined {
  const value = data[key];
  return typeof value === 'boolean' ? value : undefined;
}

/**
 * Helper to safely get string array from data
 */
function getStringArray(data: Record<string, unknown>, key: string): string[] | undefined {
  const value = data[key];
  if (Array.isArray(value) && value.every((v) => typeof v === 'string')) {
    return value;
  }
  return undefined;
}

/**
 * Builds typed action params from V1 node data.
 */
function buildNavigateParams(data: Record<string, unknown>): NavigateParams {
  return {
    url: getString(data, 'url'),
    waitForSelector: getString(data, 'waitForSelector'),
    timeoutMs: getNumber(data, 'timeoutMs'),
    waitUntil: getString(data, 'waitUntil'),
    destinationType: getString(data, 'destinationType'),
    scenario: getString(data, 'scenario') ?? getString(data, 'scenarioName'),
    scenarioPath: getString(data, 'scenarioPath'),
  };
}

function buildClickParams(data: Record<string, unknown>): ClickParams {
  return {
    selector: getString(data, 'selector') ?? '',
    button: getString(data, 'button'),
    clickCount: getNumber(data, 'clickCount'),
    delayMs: getNumber(data, 'delayMs'),
    modifiers: getStringArray(data, 'modifiers'),
    force: getBool(data, 'force'),
  };
}

function buildInputParams(data: Record<string, unknown>): InputParams {
  return {
    selector: getString(data, 'selector') ?? '',
    value: getString(data, 'value') ?? getString(data, 'text') ?? '',
    isSensitive: getBool(data, 'isSensitive'),
    submit: getBool(data, 'submit'),
    clearFirst: getBool(data, 'clearFirst'),
    delayMs: getNumber(data, 'delayMs'),
  };
}

function buildWaitParams(data: Record<string, unknown>): WaitParams {
  return {
    durationMs: getNumber(data, 'durationMs') ?? getNumber(data, 'duration'),
    selector: getString(data, 'selector'),
    state: getString(data, 'state'),
    timeoutMs: getNumber(data, 'timeoutMs'),
  };
}

function buildAssertParams(data: Record<string, unknown>): AssertParams {
  return {
    selector: getString(data, 'selector') ?? '',
    mode: getString(data, 'mode') ?? getString(data, 'assertMode') ?? 'exists',
    expected: data.expected,
    negated: getBool(data, 'negated'),
    caseSensitive: getBool(data, 'caseSensitive'),
    attributeName: getString(data, 'attributeName'),
  };
}

function buildScrollParams(data: Record<string, unknown>): ScrollParams {
  return {
    selector: getString(data, 'selector'),
    x: getNumber(data, 'x'),
    y: getNumber(data, 'y'),
    deltaX: getNumber(data, 'deltaX'),
    deltaY: getNumber(data, 'deltaY'),
    behavior: getString(data, 'behavior'),
  };
}

function buildSelectParams(data: Record<string, unknown>): SelectParams {
  return {
    selector: getString(data, 'selector') ?? '',
    value: getString(data, 'value'),
    label: getString(data, 'label'),
    index: getNumber(data, 'index'),
    timeoutMs: getNumber(data, 'timeoutMs'),
  };
}

function buildEvaluateParams(data: Record<string, unknown>): EvaluateParams {
  return {
    expression: getString(data, 'expression') ?? '',
    storeResult: getString(data, 'storeResult'),
  };
}

function buildKeyboardParams(data: Record<string, unknown>): KeyboardParams {
  return {
    key: getString(data, 'key'),
    keys: getStringArray(data, 'keys'),
    modifiers: getStringArray(data, 'modifiers'),
    action: getString(data, 'action'),
  };
}

function buildHoverParams(data: Record<string, unknown>): HoverParams {
  return {
    selector: getString(data, 'selector') ?? '',
    timeoutMs: getNumber(data, 'timeoutMs'),
  };
}

function buildScreenshotParams(data: Record<string, unknown>): ScreenshotParams {
  return {
    fullPage: getBool(data, 'fullPage'),
    selector: getString(data, 'selector'),
    quality: getNumber(data, 'quality'),
  };
}

function buildFocusParams(data: Record<string, unknown>): FocusParams {
  return {
    selector: getString(data, 'selector') ?? '',
    timeoutMs: getNumber(data, 'timeoutMs'),
  };
}

function buildBlurParams(data: Record<string, unknown>): BlurParams {
  return {
    selector: getString(data, 'selector'),
    timeoutMs: getNumber(data, 'timeoutMs'),
  };
}

function buildSubflowParams(data: Record<string, unknown>): SubflowParams {
  return {
    workflowId: getString(data, 'workflowId'),
    workflowPath: getString(data, 'workflowPath'),
    workflowVersion: getNumber(data, 'workflowVersion'),
    parameters: data.parameters as Record<string, unknown> | undefined,
  };
}

/**
 * Builds a V2 ActionDefinition from V1 node type and data.
 *
 * @param nodeType - The V1 node type (e.g., "navigate", "click")
 * @param data - The V1 node data object containing action parameters
 * @returns A V2 ActionDefinition with typed params
 */
export function buildActionDefinition(
  nodeType: string,
  data: Record<string, unknown> = {},
): ActionDefinition {
  const actionType = nodeTypeToActionType(nodeType);
  const action: ActionDefinition = { type: actionType };

  switch (actionType) {
    case ACTION_TYPES.NAVIGATE:
      action.navigate = buildNavigateParams(data);
      break;
    case ACTION_TYPES.CLICK:
      action.click = buildClickParams(data);
      break;
    case ACTION_TYPES.INPUT:
      action.input = buildInputParams(data);
      break;
    case ACTION_TYPES.WAIT:
      action.wait = buildWaitParams(data);
      break;
    case ACTION_TYPES.ASSERT:
      action.assert = buildAssertParams(data);
      break;
    case ACTION_TYPES.SCROLL:
      action.scroll = buildScrollParams(data);
      break;
    case ACTION_TYPES.SELECT:
      action.selectOption = buildSelectParams(data);
      break;
    case ACTION_TYPES.EVALUATE:
      action.evaluate = buildEvaluateParams(data);
      break;
    case ACTION_TYPES.KEYBOARD:
      action.keyboard = buildKeyboardParams(data);
      break;
    case ACTION_TYPES.HOVER:
      action.hover = buildHoverParams(data);
      break;
    case ACTION_TYPES.SCREENSHOT:
      action.screenshot = buildScreenshotParams(data);
      break;
    case ACTION_TYPES.FOCUS:
      action.focus = buildFocusParams(data);
      break;
    case ACTION_TYPES.BLUR:
      action.blur = buildBlurParams(data);
      break;
    case ACTION_TYPES.SUBFLOW:
      action.subflow = buildSubflowParams(data);
      break;
  }

  // Add metadata if present
  const label = getString(data, 'label');
  if (label) {
    action.metadata = { label };
  }

  return action;
}

/**
 * Extracts the action type from an existing ActionDefinition.
 * Returns the V1 node type string for React Flow compatibility.
 */
export function getNodeTypeFromAction(action: ActionDefinition | undefined): string {
  if (!action?.type) {
    return 'navigate';
  }
  return actionTypeToNodeType(action.type);
}

/**
 * Checks if a node has a valid V2 action field.
 */
export function hasValidAction(node: { action?: ActionDefinition }): boolean {
  return !!node.action?.type && node.action.type !== ACTION_TYPES.UNSPECIFIED;
}
