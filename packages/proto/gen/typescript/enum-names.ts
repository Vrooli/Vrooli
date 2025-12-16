/**
 * Auto-generated enum name mappings for browser-automation-studio proto types.
 *
 * These helpers convert numeric proto enums to their canonical lowercase string names,
 * eliminating the need for manual mapping functions in UI code.
 *
 * Usage:
 *   import { ExecutionStatusName, ActionTypeName } from '@vrooli/proto-types/enum-names';
 *   const name = ExecutionStatusName[proto.status]; // 'pending', 'running', etc.
 *
 * @generated from packages/proto/schemas/browser-automation-studio/v1/
 */

import {
  ExecutionStatus,
  TriggerType,
  StepStatus,
  LogLevel,
  ArtifactType,
  ExportStatus,
  SelectorType,
  NetworkEventType,
  RecordingSource,
  WorkflowEdgeType,
  ValidationSeverity,
  ChangeSource,
  AssertionMode,
  HighlightColor,
} from './browser-automation-studio/v1/base/shared_pb';

import {
  ActionType,
  MouseButton,
  NavigateWaitEvent,
  NavigateDestinationType,
  WaitState,
  ScrollBehavior,
  KeyAction,
  KeyboardModifier,
  ExtractType,
  FrameSwitchAction,
  TabSwitchAction,
  CookieOperation,
  StorageType,
  GestureType,
  SwipeDirection,
  NetworkMockOperation,
  DeviceOrientation,
  CookieSameSite,
} from './browser-automation-studio/v1/actions/action_pb';

// =============================================================================
// BASE ENUM NAME MAPPINGS (from shared.proto)
// =============================================================================

/**
 * Maps ExecutionStatus enum values to lowercase string names.
 * @example ExecutionStatusName[ExecutionStatus.PENDING] === 'pending'
 */
export const ExecutionStatusName: Record<ExecutionStatus, string> = {
  [ExecutionStatus.UNSPECIFIED]: 'unspecified',
  [ExecutionStatus.PENDING]: 'pending',
  [ExecutionStatus.RUNNING]: 'running',
  [ExecutionStatus.COMPLETED]: 'completed',
  [ExecutionStatus.FAILED]: 'failed',
  [ExecutionStatus.CANCELLED]: 'cancelled',
};

/**
 * Maps TriggerType enum values to lowercase string names.
 */
export const TriggerTypeName: Record<TriggerType, string> = {
  [TriggerType.UNSPECIFIED]: 'unspecified',
  [TriggerType.MANUAL]: 'manual',
  [TriggerType.SCHEDULED]: 'scheduled',
  [TriggerType.API]: 'api',
  [TriggerType.WEBHOOK]: 'webhook',
};

/**
 * Maps StepStatus enum values to lowercase string names.
 */
export const StepStatusName: Record<StepStatus, string> = {
  [StepStatus.UNSPECIFIED]: 'unspecified',
  [StepStatus.PENDING]: 'pending',
  [StepStatus.RUNNING]: 'running',
  [StepStatus.COMPLETED]: 'completed',
  [StepStatus.FAILED]: 'failed',
  [StepStatus.CANCELLED]: 'cancelled',
  [StepStatus.SKIPPED]: 'skipped',
  [StepStatus.RETRYING]: 'retrying',
};

/**
 * Maps LogLevel enum values to lowercase string names.
 */
export const LogLevelName: Record<LogLevel, string> = {
  [LogLevel.UNSPECIFIED]: 'unspecified',
  [LogLevel.DEBUG]: 'debug',
  [LogLevel.INFO]: 'info',
  [LogLevel.WARN]: 'warn',
  [LogLevel.ERROR]: 'error',
};

/**
 * Maps ArtifactType enum values to lowercase string names.
 */
export const ArtifactTypeName: Record<ArtifactType, string> = {
  [ArtifactType.UNSPECIFIED]: 'unspecified',
  [ArtifactType.TIMELINE_FRAME]: 'timeline_frame',
  [ArtifactType.CONSOLE_LOG]: 'console_log',
  [ArtifactType.NETWORK_EVENT]: 'network_event',
  [ArtifactType.SCREENSHOT]: 'screenshot',
  [ArtifactType.DOM_SNAPSHOT]: 'dom_snapshot',
  [ArtifactType.TRACE]: 'trace',
  [ArtifactType.CUSTOM]: 'custom',
};

/**
 * Maps ExportStatus enum values to lowercase string names.
 */
export const ExportStatusName: Record<ExportStatus, string> = {
  [ExportStatus.UNSPECIFIED]: 'unspecified',
  [ExportStatus.READY]: 'ready',
  [ExportStatus.PENDING]: 'pending',
  [ExportStatus.ERROR]: 'error',
  [ExportStatus.UNAVAILABLE]: 'unavailable',
};

/**
 * Maps SelectorType enum values to lowercase string names.
 */
export const SelectorTypeName: Record<SelectorType, string> = {
  [SelectorType.UNSPECIFIED]: 'unspecified',
  [SelectorType.CSS]: 'css',
  [SelectorType.XPATH]: 'xpath',
  [SelectorType.ID]: 'id',
  [SelectorType.DATA_TESTID]: 'data-testid',
  [SelectorType.ARIA]: 'aria',
  [SelectorType.TEXT]: 'text',
  [SelectorType.ROLE]: 'role',
  [SelectorType.PLACEHOLDER]: 'placeholder',
  [SelectorType.ALT_TEXT]: 'alt-text',
  [SelectorType.TITLE]: 'title',
};

/**
 * Maps NetworkEventType enum values to lowercase string names.
 */
export const NetworkEventTypeName: Record<NetworkEventType, string> = {
  [NetworkEventType.UNSPECIFIED]: 'unspecified',
  [NetworkEventType.REQUEST]: 'request',
  [NetworkEventType.RESPONSE]: 'response',
  [NetworkEventType.FAILURE]: 'failure',
};

/**
 * Maps RecordingSource enum values to lowercase string names.
 */
export const RecordingSourceName: Record<RecordingSource, string> = {
  [RecordingSource.UNSPECIFIED]: 'unspecified',
  [RecordingSource.AUTO]: 'auto',
  [RecordingSource.MANUAL]: 'manual',
};

/**
 * Maps WorkflowEdgeType enum values to lowercase string names.
 */
export const WorkflowEdgeTypeName: Record<WorkflowEdgeType, string> = {
  [WorkflowEdgeType.UNSPECIFIED]: 'unspecified',
  [WorkflowEdgeType.DEFAULT]: 'default',
  [WorkflowEdgeType.SMOOTHSTEP]: 'smoothstep',
  [WorkflowEdgeType.STEP]: 'step',
  [WorkflowEdgeType.STRAIGHT]: 'straight',
  [WorkflowEdgeType.BEZIER]: 'bezier',
};

/**
 * Maps ValidationSeverity enum values to lowercase string names.
 */
export const ValidationSeverityName: Record<ValidationSeverity, string> = {
  [ValidationSeverity.UNSPECIFIED]: 'unspecified',
  [ValidationSeverity.ERROR]: 'error',
  [ValidationSeverity.WARNING]: 'warning',
  [ValidationSeverity.INFO]: 'info',
};

/**
 * Maps ChangeSource enum values to lowercase string names.
 */
export const ChangeSourceName: Record<ChangeSource, string> = {
  [ChangeSource.UNSPECIFIED]: 'unspecified',
  [ChangeSource.MANUAL]: 'manual',
  [ChangeSource.AUTOSAVE]: 'autosave',
  [ChangeSource.IMPORT]: 'import',
  [ChangeSource.AI_GENERATED]: 'ai_generated',
  [ChangeSource.RECORDING]: 'recording',
};

/**
 * Maps AssertionMode enum values to lowercase string names.
 */
export const AssertionModeName: Record<AssertionMode, string> = {
  [AssertionMode.UNSPECIFIED]: 'unspecified',
  [AssertionMode.EXISTS]: 'exists',
  [AssertionMode.NOT_EXISTS]: 'not_exists',
  [AssertionMode.VISIBLE]: 'visible',
  [AssertionMode.HIDDEN]: 'hidden',
  [AssertionMode.TEXT_EQUALS]: 'text_equals',
  [AssertionMode.TEXT_CONTAINS]: 'text_contains',
  [AssertionMode.ATTRIBUTE_EQUALS]: 'attribute_equals',
  [AssertionMode.ATTRIBUTE_CONTAINS]: 'attribute_contains',
};

/**
 * Maps HighlightColor enum values to lowercase string names.
 */
export const HighlightColorName: Record<HighlightColor, string> = {
  [HighlightColor.UNSPECIFIED]: 'unspecified',
  [HighlightColor.RED]: 'red',
  [HighlightColor.GREEN]: 'green',
  [HighlightColor.BLUE]: 'blue',
  [HighlightColor.YELLOW]: 'yellow',
  [HighlightColor.ORANGE]: 'orange',
  [HighlightColor.PURPLE]: 'purple',
  [HighlightColor.CYAN]: 'cyan',
  [HighlightColor.PINK]: 'pink',
  [HighlightColor.WHITE]: 'white',
  [HighlightColor.GRAY]: 'gray',
  [HighlightColor.BLACK]: 'black',
};

// =============================================================================
// ACTION ENUM NAME MAPPINGS (from action.proto)
// =============================================================================

/**
 * Maps ActionType enum values to lowercase string names.
 * @example ActionTypeName[ActionType.CLICK] === 'click'
 */
export const ActionTypeName: Record<ActionType, string> = {
  [ActionType.UNSPECIFIED]: 'unspecified',
  [ActionType.NAVIGATE]: 'navigate',
  [ActionType.CLICK]: 'click',
  [ActionType.INPUT]: 'input',
  [ActionType.WAIT]: 'wait',
  [ActionType.ASSERT]: 'assert',
  [ActionType.SCROLL]: 'scroll',
  [ActionType.SELECT]: 'select',
  [ActionType.EVALUATE]: 'evaluate',
  [ActionType.KEYBOARD]: 'keyboard',
  [ActionType.HOVER]: 'hover',
  [ActionType.SCREENSHOT]: 'screenshot',
  [ActionType.FOCUS]: 'focus',
  [ActionType.BLUR]: 'blur',
  [ActionType.SUBFLOW]: 'subflow',
  [ActionType.EXTRACT]: 'extract',
  [ActionType.UPLOAD_FILE]: 'upload_file',
  [ActionType.DOWNLOAD]: 'download',
  [ActionType.FRAME_SWITCH]: 'frame_switch',
  [ActionType.TAB_SWITCH]: 'tab_switch',
  [ActionType.COOKIE_STORAGE]: 'cookie_storage',
  [ActionType.SHORTCUT]: 'shortcut',
  [ActionType.DRAG_DROP]: 'drag_drop',
  [ActionType.GESTURE]: 'gesture',
  [ActionType.NETWORK_MOCK]: 'network_mock',
  [ActionType.ROTATE]: 'rotate',
  [ActionType.SET_VARIABLE]: 'set_variable',
  [ActionType.LOOP]: 'loop',
  [ActionType.CONDITIONAL]: 'conditional',
};

/**
 * Maps MouseButton enum values to lowercase string names.
 */
export const MouseButtonName: Record<MouseButton, string> = {
  [MouseButton.UNSPECIFIED]: 'unspecified',
  [MouseButton.LEFT]: 'left',
  [MouseButton.RIGHT]: 'right',
  [MouseButton.MIDDLE]: 'middle',
};

/**
 * Maps NavigateWaitEvent enum values to lowercase string names.
 */
export const NavigateWaitEventName: Record<NavigateWaitEvent, string> = {
  [NavigateWaitEvent.UNSPECIFIED]: 'unspecified',
  [NavigateWaitEvent.LOAD]: 'load',
  [NavigateWaitEvent.DOMCONTENTLOADED]: 'domcontentloaded',
  [NavigateWaitEvent.NETWORKIDLE]: 'networkidle',
};

/**
 * Maps NavigateDestinationType enum values to lowercase string names.
 */
export const NavigateDestinationTypeName: Record<NavigateDestinationType, string> = {
  [NavigateDestinationType.UNSPECIFIED]: 'unspecified',
  [NavigateDestinationType.URL]: 'url',
  [NavigateDestinationType.SCENARIO]: 'scenario',
};

/**
 * Maps WaitState enum values to lowercase string names.
 */
export const WaitStateName: Record<WaitState, string> = {
  [WaitState.UNSPECIFIED]: 'unspecified',
  [WaitState.ATTACHED]: 'attached',
  [WaitState.DETACHED]: 'detached',
  [WaitState.VISIBLE]: 'visible',
  [WaitState.HIDDEN]: 'hidden',
};

/**
 * Maps ScrollBehavior enum values to lowercase string names.
 */
export const ScrollBehaviorName: Record<ScrollBehavior, string> = {
  [ScrollBehavior.UNSPECIFIED]: 'unspecified',
  [ScrollBehavior.AUTO]: 'auto',
  [ScrollBehavior.SMOOTH]: 'smooth',
};

/**
 * Maps KeyAction enum values to lowercase string names.
 */
export const KeyActionName: Record<KeyAction, string> = {
  [KeyAction.UNSPECIFIED]: 'unspecified',
  [KeyAction.PRESS]: 'press',
  [KeyAction.DOWN]: 'down',
  [KeyAction.UP]: 'up',
};

/**
 * Maps KeyboardModifier enum values to lowercase string names.
 */
export const KeyboardModifierName: Record<KeyboardModifier, string> = {
  [KeyboardModifier.UNSPECIFIED]: 'unspecified',
  [KeyboardModifier.CTRL]: 'ctrl',
  [KeyboardModifier.SHIFT]: 'shift',
  [KeyboardModifier.ALT]: 'alt',
  [KeyboardModifier.META]: 'meta',
};

/**
 * Maps ExtractType enum values to lowercase string names.
 */
export const ExtractTypeName: Record<ExtractType, string> = {
  [ExtractType.UNSPECIFIED]: 'unspecified',
  [ExtractType.TEXT]: 'text',
  [ExtractType.INNER_HTML]: 'inner_html',
  [ExtractType.OUTER_HTML]: 'outer_html',
  [ExtractType.ATTRIBUTE]: 'attribute',
  [ExtractType.PROPERTY]: 'property',
  [ExtractType.VALUE]: 'value',
};

/**
 * Maps FrameSwitchAction enum values to lowercase string names.
 */
export const FrameSwitchActionName: Record<FrameSwitchAction, string> = {
  [FrameSwitchAction.UNSPECIFIED]: 'unspecified',
  [FrameSwitchAction.ENTER]: 'enter',
  [FrameSwitchAction.PARENT]: 'parent',
  [FrameSwitchAction.EXIT]: 'exit',
};

/**
 * Maps TabSwitchAction enum values to lowercase string names.
 */
export const TabSwitchActionName: Record<TabSwitchAction, string> = {
  [TabSwitchAction.UNSPECIFIED]: 'unspecified',
  [TabSwitchAction.OPEN]: 'open',
  [TabSwitchAction.SWITCH]: 'switch',
  [TabSwitchAction.CLOSE]: 'close',
  [TabSwitchAction.LIST]: 'list',
};

/**
 * Maps CookieOperation enum values to lowercase string names.
 */
export const CookieOperationName: Record<CookieOperation, string> = {
  [CookieOperation.UNSPECIFIED]: 'unspecified',
  [CookieOperation.GET]: 'get',
  [CookieOperation.SET]: 'set',
  [CookieOperation.DELETE]: 'delete',
  [CookieOperation.CLEAR]: 'clear',
};

/**
 * Maps StorageType enum values to lowercase string names.
 */
export const StorageTypeName: Record<StorageType, string> = {
  [StorageType.UNSPECIFIED]: 'unspecified',
  [StorageType.COOKIE]: 'cookie',
  [StorageType.LOCAL_STORAGE]: 'local_storage',
  [StorageType.SESSION_STORAGE]: 'session_storage',
};

/**
 * Maps GestureType enum values to lowercase string names.
 */
export const GestureTypeName: Record<GestureType, string> = {
  [GestureType.UNSPECIFIED]: 'unspecified',
  [GestureType.SWIPE]: 'swipe',
  [GestureType.PINCH]: 'pinch',
  [GestureType.ZOOM]: 'zoom',
  [GestureType.LONG_PRESS]: 'long_press',
  [GestureType.DOUBLE_TAP]: 'double_tap',
};

/**
 * Maps SwipeDirection enum values to lowercase string names.
 */
export const SwipeDirectionName: Record<SwipeDirection, string> = {
  [SwipeDirection.UNSPECIFIED]: 'unspecified',
  [SwipeDirection.UP]: 'up',
  [SwipeDirection.DOWN]: 'down',
  [SwipeDirection.LEFT]: 'left',
  [SwipeDirection.RIGHT]: 'right',
};

/**
 * Maps NetworkMockOperation enum values to lowercase string names.
 */
export const NetworkMockOperationName: Record<NetworkMockOperation, string> = {
  [NetworkMockOperation.UNSPECIFIED]: 'unspecified',
  [NetworkMockOperation.MOCK]: 'mock',
  [NetworkMockOperation.BLOCK]: 'block',
  [NetworkMockOperation.MODIFY_REQUEST]: 'modify_request',
  [NetworkMockOperation.MODIFY_RESPONSE]: 'modify_response',
  [NetworkMockOperation.CLEAR]: 'clear',
};

/**
 * Maps DeviceOrientation enum values to lowercase string names.
 */
export const DeviceOrientationName: Record<DeviceOrientation, string> = {
  [DeviceOrientation.UNSPECIFIED]: 'unspecified',
  [DeviceOrientation.PORTRAIT]: 'portrait',
  [DeviceOrientation.LANDSCAPE]: 'landscape',
};

/**
 * Maps CookieSameSite enum values to lowercase string names.
 */
export const CookieSameSiteName: Record<CookieSameSite, string> = {
  [CookieSameSite.UNSPECIFIED]: 'unspecified',
  [CookieSameSite.STRICT]: 'strict',
  [CookieSameSite.LAX]: 'lax',
  [CookieSameSite.NONE]: 'none',
};
