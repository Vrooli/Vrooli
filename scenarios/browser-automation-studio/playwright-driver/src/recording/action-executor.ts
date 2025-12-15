/**
 * Action Executor Registry - Proto-Native Implementation
 *
 * Executes TimelineEntry actions directly without legacy type conversions.
 * Each action type has a registered executor that knows how to replay it.
 *
 * PROTO-FIRST ARCHITECTURE:
 * - Accepts TimelineEntry directly (no RecordedAction conversion)
 * - Uses proto ActionType enum (not string literals)
 * - Returns proto-aligned ActionReplayResult
 *
 * EXTENSION POINT: Adding a new action type
 * 1. Add executor using registerTimelineExecutor()
 * 2. Add tests in tests/unit/recording/action-executor.test.ts
 */

import type { Page } from 'playwright';
import type { TimelineEntry } from '../proto';
import { ActionType, MouseButton, KeyboardModifier } from '../proto';

// =============================================================================
// Types
// =============================================================================

/** Result of validating a selector */
export interface SelectorValidation {
  valid: boolean;
  matchCount: number;
  selector: string;
  error?: string;
}

/** Context provided to action executors */
export interface ExecutorContext {
  page: Page;
  timeout: number;
  validateSelector: (selector: string) => Promise<SelectorValidation>;
}

/** Error codes for action failures */
export type ActionErrorCode =
  | 'SELECTOR_NOT_FOUND'
  | 'SELECTOR_AMBIGUOUS'
  | 'ELEMENT_NOT_VISIBLE'
  | 'ELEMENT_NOT_ENABLED'
  | 'TIMEOUT'
  | 'NAVIGATION_FAILED'
  | 'UNSUPPORTED_ACTION'
  | 'MISSING_PARAMS'
  | 'UNKNOWN';

/** Result of replaying a single action */
export interface ActionReplayResult {
  entryId: string;
  sequenceNum: number;
  actionType: ActionType;
  success: boolean;
  durationMs: number;
  error?: {
    message: string;
    code: ActionErrorCode;
    matchCount?: number;
    selector?: string;
  };
  screenshotOnError?: string;
}

/** Function signature for timeline entry executors */
export type TimelineExecutor = (
  entry: TimelineEntry,
  context: ExecutorContext,
) => Promise<ActionReplayResult>;

// =============================================================================
// Registry
// =============================================================================

const executorRegistry = new Map<ActionType, TimelineExecutor>();

/** Register an executor for an action type */
export function registerTimelineExecutor(actionType: ActionType, executor: TimelineExecutor): void {
  if (executorRegistry.has(actionType)) {
    console.warn(`[ActionExecutor] Overwriting executor for action type: ${ActionType[actionType]}`);
  }
  executorRegistry.set(actionType, executor);
}

/** Get executor for an action type */
export function getTimelineExecutor(actionType: ActionType): TimelineExecutor | undefined {
  return executorRegistry.get(actionType);
}

/** Check if an executor is registered */
export function hasTimelineExecutor(actionType: ActionType): boolean {
  return executorRegistry.has(actionType);
}

/** Get all registered action types */
export function getRegisteredActionTypes(): ActionType[] {
  return Array.from(executorRegistry.keys());
}

// =============================================================================
// Helpers
// =============================================================================

/** Create base result for an entry */
function createBaseResult(entry: TimelineEntry): ActionReplayResult {
  return {
    entryId: entry.id,
    sequenceNum: entry.sequenceNum,
    actionType: entry.action?.type ?? ActionType.UNSPECIFIED,
    success: false,
    durationMs: 0,
  };
}

/** Create success result */
function successResult(base: ActionReplayResult): ActionReplayResult {
  return { ...base, success: true };
}

/** Create error result for selector validation failure */
function selectorErrorResult(
  base: ActionReplayResult,
  validation: SelectorValidation
): ActionReplayResult {
  const isNotFound = validation.matchCount === 0;
  return {
    ...base,
    error: {
      message: isNotFound
        ? `Element not found: ${validation.selector}`
        : `Multiple elements found (${validation.matchCount}): ${validation.selector}`,
      code: isNotFound ? 'SELECTOR_NOT_FOUND' : 'SELECTOR_AMBIGUOUS',
      matchCount: validation.matchCount,
      selector: validation.selector,
    },
  };
}

/** Create error result for missing selector */
function missingSelectorResult(base: ActionReplayResult, actionName: string): ActionReplayResult {
  return {
    ...base,
    error: {
      message: `${actionName} action missing selector`,
      code: 'MISSING_PARAMS',
    },
  };
}

/** Create error result for missing parameters */
function missingParamsResult(base: ActionReplayResult, message: string): ActionReplayResult {
  return {
    ...base,
    error: { message, code: 'MISSING_PARAMS' },
  };
}

/** Create error result for unsupported action */
function unsupportedResult(base: ActionReplayResult): ActionReplayResult {
  return {
    ...base,
    error: {
      message: `Unsupported action type for replay: ${ActionType[base.actionType]}`,
      code: 'UNSUPPORTED_ACTION',
    },
  };
}

/** Convert MouseButton enum to Playwright button string */
function mouseButtonToPlaywright(button?: MouseButton): 'left' | 'right' | 'middle' {
  switch (button) {
    case MouseButton.RIGHT:
      return 'right';
    case MouseButton.MIDDLE:
      return 'middle';
    case MouseButton.LEFT:
    default:
      return 'left';
  }
}

/** Convert KeyboardModifier array to Playwright modifiers */
function modifiersToPlaywright(modifiers?: KeyboardModifier[]): string[] {
  if (!modifiers) return [];
  return modifiers.map((m) => {
    switch (m) {
      case KeyboardModifier.CTRL:
        return 'Control';
      case KeyboardModifier.SHIFT:
        return 'Shift';
      case KeyboardModifier.ALT:
        return 'Alt';
      case KeyboardModifier.META:
        return 'Meta';
      default:
        return '';
    }
  }).filter(Boolean);
}

/** Validate selector and return result if invalid */
async function validateAndCheck(
  selector: string,
  context: ExecutorContext,
  base: ActionReplayResult
): Promise<ActionReplayResult | null> {
  const validation = await context.validateSelector(selector);
  if (!validation.valid) {
    return selectorErrorResult(base, validation);
  }
  return null;
}

// =============================================================================
// Executor Implementations
// =============================================================================

// Navigate executor
registerTimelineExecutor(ActionType.NAVIGATE, async (entry, context) => {
  const base = createBaseResult(entry);
  const params = entry.action?.params;

  if (params?.case !== 'navigate') {
    // Fallback to telemetry URL if params missing
    const url = entry.telemetry?.url;
    if (!url) {
      return missingParamsResult(base, 'Navigate action missing URL');
    }
    try {
      await context.page.goto(url, { timeout: context.timeout, waitUntil: 'networkidle' });
      return successResult(base);
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error);
      return {
        ...base,
        error: { message: `Navigation failed: ${message}`, code: 'NAVIGATION_FAILED' },
      };
    }
  }

  const url = params.value.url;
  if (!url) {
    return missingParamsResult(base, 'Navigate action missing URL');
  }

  try {
    await context.page.goto(url, { timeout: context.timeout, waitUntil: 'networkidle' });
    return successResult(base);
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    return {
      ...base,
      error: { message: `Navigation failed: ${message}`, code: 'NAVIGATION_FAILED' },
    };
  }
});

// Click executor
registerTimelineExecutor(ActionType.CLICK, async (entry, context) => {
  const base = createBaseResult(entry);
  const params = entry.action?.params;

  if (params?.case !== 'click') {
    return missingParamsResult(base, 'Click action missing params');
  }

  const selector = params.value.selector;
  if (!selector?.trim()) {
    return missingSelectorResult(base, 'Click');
  }

  const validationError = await validateAndCheck(selector, context, base);
  if (validationError) return validationError;

  const button = mouseButtonToPlaywright(params.value.button);
  const clickCount = params.value.clickCount || 1;
  const modifiers = modifiersToPlaywright(params.value.modifiers);

  await context.page.click(selector, {
    timeout: context.timeout,
    button,
    clickCount,
    modifiers: modifiers as Array<'Alt' | 'Control' | 'Meta' | 'Shift'>,
  });

  return successResult(base);
});

// Input (type) executor
registerTimelineExecutor(ActionType.INPUT, async (entry, context) => {
  const base = createBaseResult(entry);
  const params = entry.action?.params;

  if (params?.case !== 'input') {
    return missingParamsResult(base, 'Input action missing params');
  }

  const selector = params.value.selector;
  if (!selector?.trim()) {
    return missingSelectorResult(base, 'Input');
  }

  const validationError = await validateAndCheck(selector, context, base);
  if (validationError) return validationError;

  const text = params.value.value ?? '';

  // Clear first if specified
  if (params.value.clearFirst) {
    await context.page.fill(selector, '', { timeout: context.timeout });
  }

  // Use type() if delay specified, otherwise fill()
  if (params.value.delayMs && params.value.delayMs > 0) {
    await context.page.type(selector, text, {
      timeout: context.timeout,
      delay: params.value.delayMs,
    });
  } else {
    await context.page.fill(selector, text, { timeout: context.timeout });
  }

  return successResult(base);
});

// Scroll executor
registerTimelineExecutor(ActionType.SCROLL, async (entry, context) => {
  const base = createBaseResult(entry);
  const params = entry.action?.params;

  if (params?.case !== 'scroll') {
    return missingParamsResult(base, 'Scroll action missing params');
  }

  const selector = params.value.selector;
  const scrollX = params.value.x ?? 0;
  const scrollY = params.value.y ?? 0;

  if (selector?.trim()) {
    const validationError = await validateAndCheck(selector, context, base);
    if (validationError) return validationError;

    await context.page.evaluate(
      ({ sel, x, y }) => {
        // @ts-expect-error - document is available in browser context
        const element = document.querySelector(sel);
        if (element) {
          element.scrollTo(x, y);
        }
      },
      { sel: selector, x: scrollX, y: scrollY }
    );
  } else {
    await context.page.evaluate(
      ({ x, y }) => {
        // @ts-expect-error - window is available in browser context
        window.scrollTo(x, y);
      },
      { x: scrollX, y: scrollY }
    );
  }

  return successResult(base);
});

// Select executor
registerTimelineExecutor(ActionType.SELECT, async (entry, context) => {
  const base = createBaseResult(entry);
  const params = entry.action?.params;

  if (params?.case !== 'selectOption') {
    return missingParamsResult(base, 'Select action missing params');
  }

  const selector = params.value.selector;
  if (!selector?.trim()) {
    return missingSelectorResult(base, 'Select');
  }

  const validationError = await validateAndCheck(selector, context, base);
  if (validationError) return validationError;

  const selectBy = params.value.selectBy;
  if (!selectBy) {
    return missingParamsResult(base, 'Select action missing selection criteria');
  }

  switch (selectBy.case) {
    case 'value':
      await context.page.selectOption(selector, { value: selectBy.value }, { timeout: context.timeout });
      break;
    case 'label':
      await context.page.selectOption(selector, { label: selectBy.value }, { timeout: context.timeout });
      break;
    case 'index':
      await context.page.selectOption(selector, { index: selectBy.value }, { timeout: context.timeout });
      break;
    default:
      return missingParamsResult(base, 'Select action has invalid selection criteria');
  }

  return successResult(base);
});

// Keyboard executor
registerTimelineExecutor(ActionType.KEYBOARD, async (entry, context) => {
  const base = createBaseResult(entry);
  const params = entry.action?.params;

  if (params?.case !== 'keyboard') {
    return missingParamsResult(base, 'Keyboard action missing params');
  }

  const key = params.value.key;
  if (!key) {
    return missingParamsResult(base, 'Keyboard action missing key');
  }

  // Build key combo with modifiers
  const modifiers = modifiersToPlaywright(params.value.modifiers);
  const keyCombo = modifiers.length > 0 ? `${modifiers.join('+')}+${key}` : key;

  await context.page.keyboard.press(keyCombo);
  return successResult(base);
});

// Focus executor
registerTimelineExecutor(ActionType.FOCUS, async (entry, context) => {
  const base = createBaseResult(entry);
  const params = entry.action?.params;

  if (params?.case !== 'focus') {
    return missingParamsResult(base, 'Focus action missing params');
  }

  const selector = params.value.selector;
  if (!selector?.trim()) {
    return missingSelectorResult(base, 'Focus');
  }

  const validationError = await validateAndCheck(selector, context, base);
  if (validationError) return validationError;

  await context.page.focus(selector, { timeout: context.timeout });
  return successResult(base);
});

// Hover executor
registerTimelineExecutor(ActionType.HOVER, async (entry, context) => {
  const base = createBaseResult(entry);
  const params = entry.action?.params;

  if (params?.case !== 'hover') {
    return missingParamsResult(base, 'Hover action missing params');
  }

  const selector = params.value.selector;
  if (!selector?.trim()) {
    return missingSelectorResult(base, 'Hover');
  }

  const validationError = await validateAndCheck(selector, context, base);
  if (validationError) return validationError;

  await context.page.hover(selector, { timeout: context.timeout });
  return successResult(base);
});

// Blur executor
registerTimelineExecutor(ActionType.BLUR, async (entry, context) => {
  const base = createBaseResult(entry);
  const params = entry.action?.params;

  if (params?.case !== 'blur') {
    // Blur without selector just succeeds (blur active element)
    await context.page.evaluate(() => {
      // @ts-expect-error - document/HTMLElement are available in browser context
      if (document.activeElement instanceof HTMLElement) {
        // @ts-expect-error - document is available in browser context
        document.activeElement.blur();
      }
    });
    return successResult(base);
  }

  const selector = params.value.selector;
  if (selector?.trim()) {
    const validationError = await validateAndCheck(selector, context, base);
    if (validationError) return validationError;

    await context.page.evaluate((sel) => {
      // @ts-expect-error - document/HTMLElement are available in browser context
      const element = document.querySelector(sel);
      // @ts-expect-error - HTMLElement is available in browser context
      if (element instanceof HTMLElement) {
        element.blur();
      }
    }, selector);
  } else {
    await context.page.evaluate(() => {
      // @ts-expect-error - document/HTMLElement are available in browser context
      if (document.activeElement instanceof HTMLElement) {
        // @ts-expect-error - document is available in browser context
        document.activeElement.blur();
      }
    });
  }

  return successResult(base);
});

// Wait executor
registerTimelineExecutor(ActionType.WAIT, async (entry, context) => {
  const base = createBaseResult(entry);
  const params = entry.action?.params;

  if (params?.case !== 'wait') {
    return missingParamsResult(base, 'Wait action missing params');
  }

  // WaitParams uses oneof wait_for { duration_ms, selector }
  const waitFor = params.value.waitFor;
  if (!waitFor) {
    return missingParamsResult(base, 'Wait action missing wait condition');
  }

  if (waitFor.case === 'selector' && waitFor.value?.trim()) {
    // Wait for selector
    await context.page.waitForSelector(waitFor.value, { timeout: context.timeout });
  } else if (waitFor.case === 'durationMs' && waitFor.value > 0) {
    // Wait for duration
    await new Promise((resolve) => setTimeout(resolve, waitFor.value));
  }

  return successResult(base);
});

// Screenshot executor (no-op for replay, just succeeds)
registerTimelineExecutor(ActionType.SCREENSHOT, async (entry) => {
  const base = createBaseResult(entry);
  return successResult(base);
});

// Assert executor
registerTimelineExecutor(ActionType.ASSERT, async (entry, context) => {
  const base = createBaseResult(entry);
  const params = entry.action?.params;

  if (params?.case !== 'assert') {
    return missingParamsResult(base, 'Assert action missing params');
  }

  const selector = params.value.selector;
  if (!selector?.trim()) {
    return missingSelectorResult(base, 'Assert');
  }

  // For replay, we just verify the element exists
  const validationError = await validateAndCheck(selector, context, base);
  if (validationError) return validationError;

  return successResult(base);
});

// Evaluate executor
registerTimelineExecutor(ActionType.EVALUATE, async (entry, context) => {
  const base = createBaseResult(entry);
  const params = entry.action?.params;

  if (params?.case !== 'evaluate') {
    return missingParamsResult(base, 'Evaluate action missing params');
  }

  // EvaluateParams uses `expression` field, not `script`
  const expression = params.value.expression;
  if (!expression?.trim()) {
    return missingParamsResult(base, 'Evaluate action missing expression');
  }

  try {
    await context.page.evaluate(expression);
    return successResult(base);
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    return {
      ...base,
      error: { message: `Script evaluation failed: ${message}`, code: 'UNKNOWN' },
    };
  }
});

// =============================================================================
// Unsupported Action Handlers
// =============================================================================

// These action types are not supported for replay preview but we register
// them to provide clear error messages rather than "unknown action type"

const unsupportedActions = [
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

for (const actionType of unsupportedActions) {
  registerTimelineExecutor(actionType, async (entry) => {
    return unsupportedResult(createBaseResult(entry));
  });
}

// =============================================================================
// Main Executor Function
// =============================================================================

/**
 * Execute a TimelineEntry action.
 *
 * This is the main entry point for action execution. It looks up the
 * appropriate executor and runs it, handling errors and timing.
 */
export async function executeTimelineEntry(
  entry: TimelineEntry,
  context: ExecutorContext
): Promise<ActionReplayResult> {
  const actionType = entry.action?.type ?? ActionType.UNSPECIFIED;
  const base = createBaseResult(entry);
  const startTime = Date.now();

  try {
    const executor = getTimelineExecutor(actionType);

    if (!executor) {
      return {
        ...base,
        durationMs: Date.now() - startTime,
        error: {
          message: `No executor registered for action type: ${ActionType[actionType]}`,
          code: 'UNSUPPORTED_ACTION',
        },
      };
    }

    const result = await executor(entry, context);
    result.durationMs = Date.now() - startTime;
    return result;
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);

    // Classify common Playwright errors
    let code: ActionErrorCode = 'UNKNOWN';
    if (message.includes('waiting for selector') || message.includes('Timeout')) {
      code = 'TIMEOUT';
    } else if (message.includes('not visible')) {
      code = 'ELEMENT_NOT_VISIBLE';
    } else if (message.includes('not enabled') || message.includes('disabled')) {
      code = 'ELEMENT_NOT_ENABLED';
    }

    return {
      ...base,
      durationMs: Date.now() - startTime,
      error: { message, code },
    };
  }
}

