/**
 * Action Executor Registry
 *
 * CHANGE AXIS: Recording Action Types - Replay Execution
 *
 * This module centralizes replay execution for recorded actions.
 * When adding a new action type:
 * 1. Add to ACTION_TYPES in action-types.ts (normalization, kind mapping)
 * 2. Register an executor here using registerActionExecutor()
 * 3. Add tests in tests/unit/recording/action-executor.test.ts
 *
 * This reduces shotgun surgery - no need to modify controller.ts switch statement.
 */

import type { Page } from 'playwright';
import type { RecordedAction, ActionReplayResult, SelectorValidation, ActionType } from './types';

/**
 * Context provided to action executors.
 */
export interface ActionExecutorContext {
  page: Page;
  timeout: number;
  validateSelector: (selector: string) => Promise<SelectorValidation>;
}

/**
 * Function signature for action executors.
 * Returns ActionReplayResult with success status and optional error.
 */
export type ActionExecutor = (
  action: RecordedAction,
  context: ActionExecutorContext,
  baseResult: ActionReplayResult
) => Promise<ActionReplayResult>;

/**
 * Registry of action executors keyed by action type.
 */
const executorRegistry = new Map<ActionType, ActionExecutor>();

/**
 * Register an executor for an action type.
 *
 * @param actionType - The action type to register
 * @param executor - The executor function
 */
export function registerActionExecutor(actionType: ActionType, executor: ActionExecutor): void {
  if (executorRegistry.has(actionType)) {
    console.warn(`[ActionExecutor] Overwriting executor for action type: ${actionType}`);
  }
  executorRegistry.set(actionType, executor);
}

/**
 * Get executor for an action type.
 * Returns undefined if no executor is registered.
 */
export function getActionExecutor(actionType: ActionType): ActionExecutor | undefined {
  return executorRegistry.get(actionType);
}

/**
 * Check if an executor is registered for an action type.
 */
export function hasActionExecutor(actionType: ActionType): boolean {
  return executorRegistry.has(actionType);
}

/**
 * Get all registered action types.
 */
export function getRegisteredActionTypes(): ActionType[] {
  return Array.from(executorRegistry.keys());
}

// ============================================================================
// Default Executors
// ============================================================================

/**
 * Helper to create error result with selector validation error.
 */
function selectorError(
  baseResult: ActionReplayResult,
  validation: SelectorValidation
): ActionReplayResult {
  return {
    ...baseResult,
    error: {
      message: validation.matchCount === 0
        ? `Element not found: ${validation.selector}`
        : `Multiple elements found (${validation.matchCount}): ${validation.selector}`,
      code: validation.matchCount === 0 ? 'SELECTOR_NOT_FOUND' : 'SELECTOR_AMBIGUOUS',
      matchCount: validation.matchCount,
      selector: validation.selector,
    },
  };
}

/**
 * Helper to check if selector is missing or empty.
 */
function isSelectorMissing(action: RecordedAction): boolean {
  return !action.selector?.primary || action.selector.primary.trim() === '';
}

// Click executor
registerActionExecutor('click', async (action, context, baseResult) => {
  if (isSelectorMissing(action)) {
    return {
      ...baseResult,
      error: { message: 'Click action missing selector', code: 'UNKNOWN' },
    };
  }

  const selector = action.selector!.primary;
  const validation = await context.validateSelector(selector);

  if (!validation.valid) {
    return selectorError(baseResult, validation);
  }

  await context.page.click(selector, { timeout: context.timeout });
  return { ...baseResult, success: true };
});

// Type executor
registerActionExecutor('type', async (action, context, baseResult) => {
  if (isSelectorMissing(action)) {
    return {
      ...baseResult,
      error: { message: 'Type action missing selector', code: 'UNKNOWN' },
    };
  }

  const selector = action.selector!.primary;
  const text = action.payload?.text || '';

  const validation = await context.validateSelector(selector);
  if (!validation.valid) {
    return selectorError(baseResult, validation);
  }

  await context.page.fill(selector, text, { timeout: context.timeout });
  return { ...baseResult, success: true };
});

// Navigate executor
registerActionExecutor('navigate', async (action, context, baseResult) => {
  const url = action.payload?.targetUrl || action.url;

  if (!url) {
    return {
      ...baseResult,
      error: { message: 'Navigate action missing URL', code: 'UNKNOWN' },
    };
  }

  try {
    await context.page.goto(url, { timeout: context.timeout, waitUntil: 'networkidle' });
    return { ...baseResult, success: true };
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    return {
      ...baseResult,
      error: {
        message: `Navigation failed: ${message}`,
        code: 'NAVIGATION_FAILED',
      },
    };
  }
});

// Scroll executor
registerActionExecutor('scroll', async (action, context, baseResult) => {
  const scrollY = action.payload?.scrollY ?? 0;
  const scrollX = action.payload?.scrollX ?? 0;

  if (action.selector?.primary && action.selector.primary.trim() !== '') {
    const selector = action.selector.primary;
    const validation = await context.validateSelector(selector);

    if (!validation.valid) {
      return selectorError(baseResult, validation);
    }

    await context.page.evaluate(`
      (function() {
        const element = document.querySelector(${JSON.stringify(selector)});
        if (element) {
          element.scrollTo(${scrollX}, ${scrollY});
        }
      })()
    `);
  } else {
    await context.page.evaluate(`window.scrollTo(${scrollX}, ${scrollY})`);
  }

  return { ...baseResult, success: true };
});

// Select executor
registerActionExecutor('select', async (action, context, baseResult) => {
  if (isSelectorMissing(action)) {
    return {
      ...baseResult,
      error: { message: 'Select action missing selector', code: 'UNKNOWN' },
    };
  }

  const selector = action.selector!.primary;
  const value = action.payload?.value || '';

  const validation = await context.validateSelector(selector);
  if (!validation.valid) {
    return selectorError(baseResult, validation);
  }

  await context.page.selectOption(selector, value, { timeout: context.timeout });
  return { ...baseResult, success: true };
});

// Keypress executor
registerActionExecutor('keypress', async (action, context, baseResult) => {
  const key = action.payload?.key;

  if (!key) {
    return {
      ...baseResult,
      error: { message: 'Keypress action missing key', code: 'UNKNOWN' },
    };
  }

  await context.page.keyboard.press(key);
  return { ...baseResult, success: true };
});

// Focus executor
registerActionExecutor('focus', async (action, context, baseResult) => {
  if (isSelectorMissing(action)) {
    return {
      ...baseResult,
      error: { message: 'Focus action missing selector', code: 'UNKNOWN' },
    };
  }

  const selector = action.selector!.primary;
  const validation = await context.validateSelector(selector);

  if (!validation.valid) {
    return selectorError(baseResult, validation);
  }

  await context.page.focus(selector, { timeout: context.timeout });
  return { ...baseResult, success: true };
});

// Hover executor
registerActionExecutor('hover', async (action, context, baseResult) => {
  if (isSelectorMissing(action)) {
    return {
      ...baseResult,
      error: { message: 'Hover action missing selector', code: 'UNKNOWN' },
    };
  }

  const selector = action.selector!.primary;
  const validation = await context.validateSelector(selector);

  if (!validation.valid) {
    return selectorError(baseResult, validation);
  }

  await context.page.hover(selector, { timeout: context.timeout });
  return { ...baseResult, success: true };
});

// Blur executor - no special handling needed
registerActionExecutor('blur', async (_action, _context, baseResult) => {
  return { ...baseResult, success: true };
});
