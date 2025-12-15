import type { HandlerInstruction, ActionDefinition } from '../../src/types';
import {
  ActionType,
  NavigateWaitEvent,
  FrameSwitchAction,
  TabSwitchAction,
} from '@vrooli/proto-types/browser-automation-studio/v1/actions/action_pb';
import { create } from '@bufbuild/protobuf';
import {
  ActionDefinitionSchema,
  ClickParamsSchema,
  NavigateParamsSchema,
  InputParamsSchema,
  WaitParamsSchema,
  AssertParamsSchema,
  HoverParamsSchema,
  FocusParamsSchema,
  BlurParamsSchema,
  ScrollParamsSchema,
  ScreenshotParamsSchema,
  UploadFileParamsSchema,
  ExtractParamsSchema,
  EvaluateParamsSchema,
  DownloadParamsSchema,
  FrameSwitchParamsSchema,
  TabSwitchParamsSchema,
} from '@vrooli/proto-types/browser-automation-studio/v1/actions/action_pb';

/**
 * Convert waitUntil string to NavigateWaitEvent enum.
 */
function stringToNavigateWaitEvent(waitUntil: string | undefined): NavigateWaitEvent | undefined {
  if (!waitUntil) return undefined;
  switch (waitUntil.toLowerCase()) {
    case 'load': return NavigateWaitEvent.LOAD;
    case 'domcontentloaded': return NavigateWaitEvent.DOMCONTENTLOADED;
    case 'networkidle': return NavigateWaitEvent.NETWORKIDLE;
    default: return undefined;
  }
}

/**
 * Convert frame switch action string to FrameSwitchAction enum.
 */
function stringToFrameSwitchAction(action: string | undefined): FrameSwitchAction {
  if (!action) return FrameSwitchAction.UNSPECIFIED;
  switch (action.toLowerCase()) {
    case 'enter': return FrameSwitchAction.ENTER;
    case 'exit': return FrameSwitchAction.EXIT;
    case 'parent': return FrameSwitchAction.PARENT;
    default: return FrameSwitchAction.UNSPECIFIED;
  }
}

/**
 * Convert tab switch action string to TabSwitchAction enum.
 */
function stringToTabSwitchAction(action: string | undefined): TabSwitchAction {
  if (!action) return TabSwitchAction.UNSPECIFIED;
  switch (action.toLowerCase()) {
    case 'open': return TabSwitchAction.OPEN;
    case 'switch': return TabSwitchAction.SWITCH;
    case 'close': return TabSwitchAction.CLOSE;
    case 'list': return TabSwitchAction.LIST;
    default: return TabSwitchAction.UNSPECIFIED;
  }
}

/**
 * Factory for creating test HandlerInstruction objects with proper defaults.
 * HandlerInstruction is the handler-friendly wrapper that uses plain object params.
 *
 * MIGRATION NOTE: Tests should prefer using createTypedInstruction() which sets
 * the typed `action` field. The `params` field is only for backward compatibility.
 */
export function createTestInstruction(
  overrides: Partial<HandlerInstruction> = {}
): HandlerInstruction {
  return {
    index: 0,
    nodeId: 'test-node',
    type: 'test',
    params: {},
    ...overrides,
  };
}

/**
 * Create a typed instruction with ActionDefinition for testing handlers.
 * This is the preferred way to create test instructions - it uses the typed
 * proto action field and avoids Zod fallback.
 *
 * @example
 * // Click instruction
 * const clickInstr = createTypedInstruction('click', { selector: '#btn' });
 *
 * // Navigate instruction
 * const navInstr = createTypedInstruction('navigate', { url: 'https://example.com' });
 */
export function createTypedInstruction(
  actionType: string,
  params: Record<string, unknown>,
  overrides: Partial<Omit<HandlerInstruction, 'action' | 'type'>> = {}
): HandlerInstruction {
  const action = buildAction(actionType, params);
  return {
    index: 0,
    nodeId: 'test-node',
    type: actionType, // Keep for backward compat
    params: params,   // Keep for backward compat
    action,
    ...overrides,
  };
}

/**
 * Build a typed ActionDefinition from action type and params.
 */
function buildAction(actionType: string, params: Record<string, unknown>): ActionDefinition {
  const action = create(ActionDefinitionSchema, {});

  switch (actionType.toLowerCase()) {
    case 'click':
      action.type = ActionType.CLICK;
      action.params = {
        case: 'click',
        value: create(ClickParamsSchema, {
          selector: String(params.selector ?? ''),
          // Note: ClickParams in proto has delayMs (delay between clicks) but NOT timeoutMs
          // Timeout for click is controlled by config.execution.defaultTimeoutMs
          delayMs: params.delayMs != null ? Number(params.delayMs) : undefined,
          clickCount: params.clickCount != null ? Number(params.clickCount) : undefined,
        }),
      };
      break;

    case 'navigate':
      action.type = ActionType.NAVIGATE;
      action.params = {
        case: 'navigate',
        value: create(NavigateParamsSchema, {
          url: String(params.url ?? params.target ?? params.href ?? ''),
          timeoutMs: params.timeoutMs != null ? Number(params.timeoutMs) : undefined,
          waitUntil: stringToNavigateWaitEvent(params.waitUntil as string | undefined),
        }),
      };
      break;

    case 'type':
    case 'input':
      action.type = ActionType.INPUT;
      action.params = {
        case: 'input',
        value: create(InputParamsSchema, {
          selector: String(params.selector ?? ''),
          value: String(params.text ?? params.value ?? ''),
          delayMs: params.timeoutMs != null ? Number(params.timeoutMs) : undefined,
        }),
      };
      break;

    case 'wait':
      action.type = ActionType.WAIT;
      action.params = {
        case: 'wait',
        value: create(WaitParamsSchema, {
          waitFor: params.selector
            ? { case: 'selector', value: String(params.selector) }
            : { case: 'durationMs', value: Number(params.timeoutMs ?? params.ms ?? 1000) },
          timeoutMs: params.timeoutMs != null ? Number(params.timeoutMs) : undefined,
        }),
      };
      break;

    case 'assert':
      action.type = ActionType.ASSERT;
      action.params = {
        case: 'assert',
        value: create(AssertParamsSchema, {
          selector: String(params.selector ?? ''),
          mode: params.mode != null ? Number(params.mode) : undefined,
          timeoutMs: params.timeoutMs != null ? Number(params.timeoutMs) : undefined,
        }),
      };
      break;

    case 'hover':
      action.type = ActionType.HOVER;
      action.params = {
        case: 'hover',
        value: create(HoverParamsSchema, {
          selector: String(params.selector ?? ''),
          timeoutMs: params.timeoutMs != null ? Number(params.timeoutMs) : undefined,
        }),
      };
      break;

    case 'focus':
      action.type = ActionType.FOCUS;
      action.params = {
        case: 'focus',
        value: create(FocusParamsSchema, {
          selector: String(params.selector ?? ''),
          timeoutMs: params.timeoutMs != null ? Number(params.timeoutMs) : undefined,
        }),
      };
      break;

    case 'blur':
      action.type = ActionType.BLUR;
      action.params = {
        case: 'blur',
        value: create(BlurParamsSchema, {
          selector: params.selector != null ? String(params.selector) : undefined,
          timeoutMs: params.timeoutMs != null ? Number(params.timeoutMs) : undefined,
        }),
      };
      break;

    case 'scroll':
      action.type = ActionType.SCROLL;
      action.params = {
        case: 'scroll',
        value: create(ScrollParamsSchema, {
          selector: params.selector != null ? String(params.selector) : undefined,
          x: params.x != null ? Number(params.x) : undefined,
          y: params.y != null ? Number(params.y) : undefined,
        }),
      };
      break;

    case 'screenshot':
      action.type = ActionType.SCREENSHOT;
      action.params = {
        case: 'screenshot',
        value: create(ScreenshotParamsSchema, {
          fullPage: params.fullPage != null ? Boolean(params.fullPage) : undefined,
          quality: params.quality != null ? Number(params.quality) : undefined,
        }),
      };
      break;

    case 'uploadfile':
    case 'upload':
      action.type = ActionType.UPLOAD_FILE;
      action.params = {
        case: 'uploadFile',
        value: create(UploadFileParamsSchema, {
          selector: String(params.selector ?? ''),
          filePaths: Array.isArray(params.filePath)
            ? params.filePath.map(String)
            : params.filePath
              ? [String(params.filePath)]
              : Array.isArray(params.filePaths)
                ? params.filePaths.map(String)
                : [],
          timeoutMs: params.timeoutMs != null ? Number(params.timeoutMs) : undefined,
        }),
      };
      break;

    case 'extract':
      action.type = ActionType.EXTRACT;
      action.params = {
        case: 'extract',
        value: create(ExtractParamsSchema, {
          selector: String(params.selector ?? ''),
          timeoutMs: params.timeoutMs != null ? Number(params.timeoutMs) : undefined,
        }),
      };
      break;

    case 'evaluate':
      action.type = ActionType.EVALUATE;
      action.params = {
        case: 'evaluate',
        value: create(EvaluateParamsSchema, {
          expression: String(params.script ?? params.expression ?? ''),
          storeResult: params.storeResult != null ? String(params.storeResult) : undefined,
        }),
      };
      break;

    case 'download':
      action.type = ActionType.DOWNLOAD;
      action.params = {
        case: 'download',
        value: create(DownloadParamsSchema, {
          selector: params.selector != null ? String(params.selector) : undefined,
          url: params.url != null ? String(params.url) : undefined,
          savePath: params.savePath != null ? String(params.savePath) : undefined,
          timeoutMs: params.timeoutMs != null ? Number(params.timeoutMs) : undefined,
        }),
      };
      break;

    case 'frame-switch':
    case 'frameswitch':
      action.type = ActionType.FRAME_SWITCH;
      action.params = {
        case: 'frameSwitch',
        value: create(FrameSwitchParamsSchema, {
          action: stringToFrameSwitchAction(params.action as string),
          selector: params.selector != null ? String(params.selector) : undefined,
          frameId: params.frameId != null ? String(params.frameId) : undefined,
          frameUrl: params.frameUrl != null ? String(params.frameUrl) : undefined,
          timeoutMs: params.timeoutMs != null ? Number(params.timeoutMs) : undefined,
        }),
      };
      break;

    case 'tab-switch':
    case 'tabswitch':
    case 'tab':
    case 'tabs':
      action.type = ActionType.TAB_SWITCH;
      action.params = {
        case: 'tabSwitch',
        value: create(TabSwitchParamsSchema, {
          action: stringToTabSwitchAction(params.action as string),
          url: params.url != null ? String(params.url) : undefined,
          index: params.index != null ? Number(params.index) : undefined,
          title: params.title != null ? String(params.title) : undefined,
          urlPattern: params.urlPattern != null ? String(params.urlPattern) : undefined,
        }),
      };
      break;

    default:
      action.type = ActionType.UNSPECIFIED;
  }

  return action;
}
