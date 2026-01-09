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
  // Control flow
  SET_VARIABLE: 'ACTION_TYPE_SET_VARIABLE',
  LOOP: 'ACTION_TYPE_LOOP',
  CONDITIONAL: 'ACTION_TYPE_CONDITIONAL',
  // Data extraction
  EXTRACT: 'ACTION_TYPE_EXTRACT',
  // File operations
  UPLOAD_FILE: 'ACTION_TYPE_UPLOAD_FILE',
  DOWNLOAD: 'ACTION_TYPE_DOWNLOAD',
  // Context switching
  FRAME_SWITCH: 'ACTION_TYPE_FRAME_SWITCH',
  TAB_SWITCH: 'ACTION_TYPE_TAB_SWITCH',
  // Storage
  COOKIE_STORAGE: 'ACTION_TYPE_COOKIE_STORAGE',
  // Input actions
  SHORTCUT: 'ACTION_TYPE_SHORTCUT',
  DRAG_DROP: 'ACTION_TYPE_DRAG_DROP',
  GESTURE: 'ACTION_TYPE_GESTURE',
  // Network
  NETWORK_MOCK: 'ACTION_TYPE_NETWORK_MOCK',
  // Device
  ROTATE: 'ACTION_TYPE_ROTATE',
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
    case 'script':
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
    // Control flow
    case 'setvariable':
    case 'set_variable':
    case 'setvar':
      return ACTION_TYPES.SET_VARIABLE;
    case 'usevariable':
    case 'use_variable':
    case 'usevar':
      return ACTION_TYPES.SET_VARIABLE; // UseVariable uses same type as SetVariable
    case 'loop':
      return ACTION_TYPES.LOOP;
    case 'conditional':
    case 'if':
    case 'branch':
      return ACTION_TYPES.CONDITIONAL;
    // Data extraction
    case 'extract':
      return ACTION_TYPES.EXTRACT;
    // File operations
    case 'uploadfile':
    case 'upload_file':
    case 'upload':
      return ACTION_TYPES.UPLOAD_FILE;
    case 'download':
      return ACTION_TYPES.DOWNLOAD;
    // Context switching
    case 'frameswitch':
    case 'frame_switch':
    case 'frame':
      return ACTION_TYPES.FRAME_SWITCH;
    case 'tabswitch':
    case 'tab_switch':
    case 'tab':
      return ACTION_TYPES.TAB_SWITCH;
    // Storage (cookies and localStorage)
    case 'setcookie':
    case 'set_cookie':
    case 'getcookie':
    case 'get_cookie':
    case 'clearcookie':
    case 'clear_cookie':
    case 'setstorage':
    case 'set_storage':
    case 'getstorage':
    case 'get_storage':
    case 'clearstorage':
    case 'clear_storage':
    case 'cookie':
    case 'storage':
    case 'cookiestorage':
    case 'cookie_storage':
      return ACTION_TYPES.COOKIE_STORAGE;
    // Input actions
    case 'shortcut':
      return ACTION_TYPES.SHORTCUT;
    case 'dragdrop':
    case 'drag_drop':
    case 'drag':
      return ACTION_TYPES.DRAG_DROP;
    case 'gesture':
      return ACTION_TYPES.GESTURE;
    // Network
    case 'networkmock':
    case 'network_mock':
    case 'mock':
      return ACTION_TYPES.NETWORK_MOCK;
    // Device
    case 'rotate':
      return ACTION_TYPES.ROTATE;
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
  // Control flow
  setVariable?: SetVariableParams;
  loop?: LoopParams;
  conditional?: ConditionalParams;
  // Data extraction
  extract?: ExtractParams;
  // File operations
  uploadFile?: UploadFileParams;
  download?: DownloadParams;
  // Context switching
  frameSwitch?: FrameSwitchParams;
  tabSwitch?: TabSwitchParams;
  // Storage
  cookieStorage?: CookieStorageParams;
  // Input actions
  shortcut?: ShortcutParams;
  dragDrop?: DragDropParams;
  gesture?: GestureParams;
  // Network
  networkMock?: NetworkMockParams;
  // Device
  rotate?: RotateParams;
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

// Control flow params
export interface SetVariableParams {
  name?: string;
  value?: string;
  sourceType?: string;
  valueType?: string;
  expression?: string;
  selector?: string;
  extractType?: string;
  attribute?: string;
  storeAs?: string;
  timeoutMs?: number;
  allMatches?: boolean;
  // UseVariable-specific fields (shares same action type)
  transform?: string;
  required?: boolean;
}

export interface LoopParams {
  loopType?: string;
  arraySource?: string;
  count?: number;
  maxIterations?: number;
  itemVariable?: string;
  indexVariable?: string;
  conditionType?: string;
  conditionVariable?: string;
  conditionOperator?: string;
  conditionValue?: string;
  conditionExpression?: string;
  iterationTimeoutMs?: number;
  totalTimeoutMs?: number;
}

export interface ConditionalParams {
  conditionType?: string;
  expression?: string;
  selector?: string;
  variable?: string;
  operator?: string;
  value?: string;
  negate?: boolean;
  timeoutMs?: number;
  pollIntervalMs?: number;
}

// Data extraction params
export interface ExtractParams {
  selector?: string;
  extractType?: string;
  attribute?: string;
  storeAs?: string;
  allMatches?: boolean;
  timeoutMs?: number;
}

// File operation params
export interface UploadFileParams {
  selector?: string;
  filePath?: string;
  filePaths?: string[];
  timeoutMs?: number;
}

export interface DownloadParams {
  url?: string;
  savePath?: string;
  timeoutMs?: number;
}

// Context switching params
export interface FrameSwitchParams {
  switchBy?: string;
  selector?: string;
  name?: string;
  urlMatch?: string;
  index?: number;
  timeoutMs?: number;
}

export interface TabSwitchParams {
  switchBy?: string;
  index?: number;
  titleMatch?: string;
  urlMatch?: string;
  waitForNew?: boolean;
  closeOld?: boolean;
  timeoutMs?: number;
}

// Storage params (cookies and localStorage)
export interface CookieStorageParams {
  operation?: string;
  storageType?: string;
  name?: string;
  value?: string;
  url?: string;
  domain?: string;
  path?: string;
  sameSite?: string;
  secure?: boolean;
  httpOnly?: boolean;
  ttlSeconds?: number;
  expiresAt?: string;
  storeAs?: string;
  timeoutMs?: number;
  waitForMs?: number;
}

// Input action params
export interface ShortcutParams {
  keys?: string[];
  modifiers?: string[];
  timeoutMs?: number;
}

export interface DragDropParams {
  sourceSelector?: string;
  targetSelector?: string;
  sourcePosition?: { x?: number; y?: number };
  targetPosition?: { x?: number; y?: number };
  steps?: number;
  timeoutMs?: number;
}

export interface GestureParams {
  gestureType?: string;
  selector?: string;
  startPosition?: { x?: number; y?: number };
  endPosition?: { x?: number; y?: number };
  duration?: number;
  timeoutMs?: number;
}

// Network params
export interface NetworkMockParams {
  urlPattern?: string;
  method?: string;
  responseStatus?: number;
  responseBody?: string;
  responseHeaders?: Record<string, string>;
  delay?: number;
}

// Device params
export interface RotateParams {
  orientation?: string;
  angle?: number;
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

// Control flow builders
function buildSetVariableParams(data: Record<string, unknown>): SetVariableParams {
  return {
    name: getString(data, 'name'),
    value: getString(data, 'value'),
    sourceType: getString(data, 'sourceType'),
    valueType: getString(data, 'valueType'),
    expression: getString(data, 'expression'),
    selector: getString(data, 'selector'),
    extractType: getString(data, 'extractType'),
    attribute: getString(data, 'attribute'),
    storeAs: getString(data, 'storeAs'),
    timeoutMs: getNumber(data, 'timeoutMs'),
    allMatches: getBool(data, 'allMatches'),
  };
}

function buildLoopParams(data: Record<string, unknown>): LoopParams {
  return {
    loopType: getString(data, 'loopType'),
    arraySource: getString(data, 'arraySource'),
    count: getNumber(data, 'count'),
    maxIterations: getNumber(data, 'maxIterations'),
    itemVariable: getString(data, 'itemVariable'),
    indexVariable: getString(data, 'indexVariable'),
    conditionType: getString(data, 'conditionType'),
    conditionVariable: getString(data, 'conditionVariable'),
    conditionOperator: getString(data, 'conditionOperator'),
    conditionValue: getString(data, 'conditionValue'),
    conditionExpression: getString(data, 'conditionExpression'),
    iterationTimeoutMs: getNumber(data, 'iterationTimeoutMs'),
    totalTimeoutMs: getNumber(data, 'totalTimeoutMs'),
  };
}

function buildConditionalParams(data: Record<string, unknown>): ConditionalParams {
  return {
    conditionType: getString(data, 'conditionType'),
    expression: getString(data, 'expression'),
    selector: getString(data, 'selector'),
    variable: getString(data, 'variable'),
    operator: getString(data, 'operator'),
    value: getString(data, 'value'),
    negate: getBool(data, 'negate'),
    timeoutMs: getNumber(data, 'timeoutMs'),
    pollIntervalMs: getNumber(data, 'pollIntervalMs'),
  };
}

// Data extraction builder
function buildExtractParams(data: Record<string, unknown>): ExtractParams {
  return {
    selector: getString(data, 'selector'),
    extractType: getString(data, 'extractType'),
    attribute: getString(data, 'attribute'),
    storeAs: getString(data, 'storeAs'),
    allMatches: getBool(data, 'allMatches'),
    timeoutMs: getNumber(data, 'timeoutMs'),
  };
}

// File operation builders
function buildUploadFileParams(data: Record<string, unknown>): UploadFileParams {
  return {
    selector: getString(data, 'selector'),
    filePath: getString(data, 'filePath'),
    filePaths: getStringArray(data, 'filePaths'),
    timeoutMs: getNumber(data, 'timeoutMs'),
  };
}

function buildDownloadParams(data: Record<string, unknown>): DownloadParams {
  return {
    url: getString(data, 'url'),
    savePath: getString(data, 'savePath'),
    timeoutMs: getNumber(data, 'timeoutMs'),
  };
}

// Context switching builders
function buildFrameSwitchParams(data: Record<string, unknown>): FrameSwitchParams {
  return {
    switchBy: getString(data, 'switchBy'),
    selector: getString(data, 'selector'),
    name: getString(data, 'name'),
    urlMatch: getString(data, 'urlMatch'),
    index: getNumber(data, 'index'),
    timeoutMs: getNumber(data, 'timeoutMs'),
  };
}

function buildTabSwitchParams(data: Record<string, unknown>): TabSwitchParams {
  return {
    switchBy: getString(data, 'switchBy'),
    index: getNumber(data, 'index'),
    titleMatch: getString(data, 'titleMatch'),
    urlMatch: getString(data, 'urlMatch'),
    waitForNew: getBool(data, 'waitForNew'),
    closeOld: getBool(data, 'closeOld'),
    timeoutMs: getNumber(data, 'timeoutMs'),
  };
}

// Storage builder
function buildCookieStorageParams(data: Record<string, unknown>): CookieStorageParams {
  return {
    operation: getString(data, 'operation'),
    storageType: getString(data, 'storageType'),
    name: getString(data, 'name'),
    value: getString(data, 'value'),
    url: getString(data, 'url'),
    domain: getString(data, 'domain'),
    path: getString(data, 'path'),
    sameSite: getString(data, 'sameSite'),
    secure: getBool(data, 'secure'),
    httpOnly: getBool(data, 'httpOnly'),
    ttlSeconds: getNumber(data, 'ttlSeconds'),
    expiresAt: getString(data, 'expiresAt'),
    storeAs: getString(data, 'storeAs'),
    timeoutMs: getNumber(data, 'timeoutMs'),
    waitForMs: getNumber(data, 'waitForMs'),
  };
}

// Input action builders
function buildShortcutParams(data: Record<string, unknown>): ShortcutParams {
  return {
    keys: getStringArray(data, 'keys'),
    modifiers: getStringArray(data, 'modifiers'),
    timeoutMs: getNumber(data, 'timeoutMs'),
  };
}

function buildDragDropParams(data: Record<string, unknown>): DragDropParams {
  const sourcePos = data.sourcePosition as { x?: number; y?: number } | undefined;
  const targetPos = data.targetPosition as { x?: number; y?: number } | undefined;
  return {
    sourceSelector: getString(data, 'sourceSelector'),
    targetSelector: getString(data, 'targetSelector'),
    sourcePosition: sourcePos,
    targetPosition: targetPos,
    steps: getNumber(data, 'steps'),
    timeoutMs: getNumber(data, 'timeoutMs'),
  };
}

function buildGestureParams(data: Record<string, unknown>): GestureParams {
  const startPos = data.startPosition as { x?: number; y?: number } | undefined;
  const endPos = data.endPosition as { x?: number; y?: number } | undefined;
  return {
    gestureType: getString(data, 'gestureType'),
    selector: getString(data, 'selector'),
    startPosition: startPos,
    endPosition: endPos,
    duration: getNumber(data, 'duration'),
    timeoutMs: getNumber(data, 'timeoutMs'),
  };
}

// Network builder
function buildNetworkMockParams(data: Record<string, unknown>): NetworkMockParams {
  return {
    urlPattern: getString(data, 'urlPattern'),
    method: getString(data, 'method'),
    responseStatus: getNumber(data, 'responseStatus'),
    responseBody: getString(data, 'responseBody'),
    responseHeaders: data.responseHeaders as Record<string, string> | undefined,
    delay: getNumber(data, 'delay'),
  };
}

// Device builder
function buildRotateParams(data: Record<string, unknown>): RotateParams {
  return {
    orientation: getString(data, 'orientation'),
    angle: getNumber(data, 'angle'),
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
    // Control flow
    case ACTION_TYPES.SET_VARIABLE:
      action.setVariable = buildSetVariableParams(data);
      break;
    case ACTION_TYPES.LOOP:
      action.loop = buildLoopParams(data);
      break;
    case ACTION_TYPES.CONDITIONAL:
      action.conditional = buildConditionalParams(data);
      break;
    // Data extraction
    case ACTION_TYPES.EXTRACT:
      action.extract = buildExtractParams(data);
      break;
    // File operations
    case ACTION_TYPES.UPLOAD_FILE:
      action.uploadFile = buildUploadFileParams(data);
      break;
    case ACTION_TYPES.DOWNLOAD:
      action.download = buildDownloadParams(data);
      break;
    // Context switching
    case ACTION_TYPES.FRAME_SWITCH:
      action.frameSwitch = buildFrameSwitchParams(data);
      break;
    case ACTION_TYPES.TAB_SWITCH:
      action.tabSwitch = buildTabSwitchParams(data);
      break;
    // Storage
    case ACTION_TYPES.COOKIE_STORAGE:
      action.cookieStorage = buildCookieStorageParams(data);
      break;
    // Input actions
    case ACTION_TYPES.SHORTCUT:
      action.shortcut = buildShortcutParams(data);
      break;
    case ACTION_TYPES.DRAG_DROP:
      action.dragDrop = buildDragDropParams(data);
      break;
    case ACTION_TYPES.GESTURE:
      action.gesture = buildGestureParams(data);
      break;
    // Network
    case ACTION_TYPES.NETWORK_MOCK:
      action.networkMock = buildNetworkMockParams(data);
      break;
    // Device
    case ACTION_TYPES.ROTATE:
      action.rotate = buildRotateParams(data);
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
