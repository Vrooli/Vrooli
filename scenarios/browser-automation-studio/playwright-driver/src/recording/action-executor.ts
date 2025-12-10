/**
 * Action Executor Registry
 *
 * Centralizes replay execution for recorded actions. Each action type (click, type,
 * navigate, etc.) has a registered executor function that knows how to replay it.
 *
 * EXTENSION POINT: Adding a new action type
 * 1. Add to ACTION_TYPES in action-types.ts (normalization, kind mapping)
 * 2. Register an executor here using registerActionExecutor()
 * 3. Add tests in tests/unit/recording/action-executor.test.ts
 *
 * Benefits of the registry pattern:
 * - No switch statements to modify when adding action types
 * - Each executor is self-contained and testable
 * - Easy to extend without touching controller.ts
 */

import type { Page } from 'playwright';
import type { RecordedAction, ActionReplayResult, SelectorValidation, ActionType } from './types';

// =============================================================================
// Types
// =============================================================================

/** Context provided to action executors */
export interface ActionExecutorContext {
  page: Page;
  timeout: number;
  validateSelector: (selector: string) => Promise<SelectorValidation>;
}

/** Function signature for action executors */
export type ActionExecutor = (
  action: RecordedAction,
  context: ActionExecutorContext,
  baseResult: ActionReplayResult
) => Promise<ActionReplayResult>;

// =============================================================================
// Registry
// =============================================================================

const executorRegistry = new Map<ActionType, ActionExecutor>();

/** Register an executor for an action type */
export function registerActionExecutor(actionType: ActionType, executor: ActionExecutor): void {
  if (executorRegistry.has(actionType)) {
    console.warn(`[ActionExecutor] Overwriting executor for action type: ${actionType}`);
  }
  executorRegistry.set(actionType, executor);
}

/** Get executor for an action type (returns undefined if not registered) */
export function getActionExecutor(actionType: ActionType): ActionExecutor | undefined {
  return executorRegistry.get(actionType);
}

/** Check if an executor is registered for an action type */
export function hasActionExecutor(actionType: ActionType): boolean {
  return executorRegistry.has(actionType);
}

/** Get all registered action types */
export function getRegisteredActionTypes(): ActionType[] {
  return Array.from(executorRegistry.keys());
}

// =============================================================================
// Helpers
// =============================================================================

/** Create error result for selector validation failure */
function createSelectorErrorResult(
  baseResult: ActionReplayResult,
  validation: SelectorValidation
): ActionReplayResult {
  const isNotFound = validation.matchCount === 0;
  return {
    ...baseResult,
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

/** Check if action is missing a required selector */
function isMissingSelector(action: RecordedAction): boolean {
  return !action.selector?.primary || action.selector.primary.trim() === '';
}

/** Create error result for missing selector */
function createMissingSelectorResult(baseResult: ActionReplayResult, actionType: string): ActionReplayResult {
  return {
    ...baseResult,
    error: { message: `${actionType} action missing selector`, code: 'UNKNOWN' },
  };
}

// =============================================================================
// Executor Registrations
// =============================================================================

// Click executor
registerActionExecutor('click', async (action, context, baseResult) => {
  if (isMissingSelector(action)) {
    return createMissingSelectorResult(baseResult, 'Click');
  }

  const selector = action.selector!.primary;
  const validation = await context.validateSelector(selector);
  if (!validation.valid) {
    return createSelectorErrorResult(baseResult, validation);
  }

  await context.page.click(selector, { timeout: context.timeout });
  return { ...baseResult, success: true };
});

// Type executor
registerActionExecutor('type', async (action, context, baseResult) => {
  if (isMissingSelector(action)) {
    return createMissingSelectorResult(baseResult, 'Type');
  }

  const selector = action.selector!.primary;
  const text = action.payload?.text || '';

  const validation = await context.validateSelector(selector);
  if (!validation.valid) {
    return createSelectorErrorResult(baseResult, validation);
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
      return createSelectorErrorResult(baseResult, validation);
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
  if (isMissingSelector(action)) {
    return createMissingSelectorResult(baseResult, 'Select');
  }

  const selector = action.selector!.primary;
  const value = action.payload?.value || '';

  const validation = await context.validateSelector(selector);
  if (!validation.valid) {
    return createSelectorErrorResult(baseResult, validation);
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
  if (isMissingSelector(action)) {
    return createMissingSelectorResult(baseResult, 'Focus');
  }

  const selector = action.selector!.primary;
  const validation = await context.validateSelector(selector);
  if (!validation.valid) {
    return createSelectorErrorResult(baseResult, validation);
  }

  await context.page.focus(selector, { timeout: context.timeout });
  return { ...baseResult, success: true };
});

// Hover executor
registerActionExecutor('hover', async (action, context, baseResult) => {
  if (isMissingSelector(action)) {
    return createMissingSelectorResult(baseResult, 'Hover');
  }

  const selector = action.selector!.primary;
  const validation = await context.validateSelector(selector);
  if (!validation.valid) {
    return createSelectorErrorResult(baseResult, validation);
  }

  await context.page.hover(selector, { timeout: context.timeout });
  return { ...baseResult, success: true };
});

// Blur executor - no special handling needed
registerActionExecutor('blur', async (_action, _context, baseResult) => {
  return { ...baseResult, success: true };
});
