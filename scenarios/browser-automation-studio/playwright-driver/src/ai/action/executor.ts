/**
 * Action Executor
 *
 * STABILITY: STABLE CORE
 *
 * This module executes BrowserAction objects via Playwright.
 * It translates high-level actions into Playwright operations.
 *
 * Supports human-like typing behavior via BehaviorSettings:
 * - Pre-typing delays (pause before starting to type)
 * - Character-by-character typing with variable delays
 * - Enhanced variance based on digraphs, shifted chars, etc.
 * - Paste threshold for long text
 * - Micro-pauses during typing
 */

import type { Page } from 'rebrowser-playwright';
import type {
  BrowserAction,
  ClickAction,
  TypeAction,
  ScrollAction,
  NavigateAction,
  HoverAction,
  SelectAction,
  WaitAction,
  KeyPressAction,
} from './types';
import type { ElementLabel } from '../vision-client/types';
import type {
  ActionExecutorInterface,
  ActionExecutionResult,
  ActionExecutionContext,
} from '../vision-agent/types';
import type { BehaviorSettings } from '../../types/browser-profile';
import { HumanBehavior, sleep } from '../../browser-profile/human-behavior';

/**
 * Default timeout for actions in milliseconds.
 */
const DEFAULT_ACTION_TIMEOUT = 30000;

/**
 * Default wait time in milliseconds.
 */
const DEFAULT_WAIT_MS = 1000;

/**
 * Configuration for ActionExecutor.
 */
export interface ActionExecutorConfig {
  /** Timeout for actions in ms */
  actionTimeout?: number;
  /** Whether to wait for navigation after clicks */
  waitForNavigation?: boolean;
  /** Behavior settings for human-like interactions */
  behaviorSettings?: BehaviorSettings;
  /** @deprecated Use behaviorSettings instead. Delay between typing characters in ms (0 for instant) */
  typeDelay?: number;
}

/**
 * Creates an ActionExecutor instance.
 *
 * TESTING SEAM: This function returns an interface that can be mocked.
 */
export function createActionExecutor(
  config: ActionExecutorConfig = {}
): ActionExecutorInterface {
  const actionTimeout = config.actionTimeout ?? DEFAULT_ACTION_TIMEOUT;
  // Create HumanBehavior instance if settings provided
  const behavior = config.behaviorSettings ? new HumanBehavior(config.behaviorSettings) : null;
  // Legacy fallback for typeDelay
  const legacyTypeDelay = config.typeDelay ?? 0;

  return {
    async execute(
      page: Page,
      action: BrowserAction,
      elementLabels?: ElementLabel[]
    ): Promise<ActionExecutionResult> {
      const startTime = Date.now();
      let context: ActionExecutionContext | undefined;

      try {
        switch (action.type) {
          case 'click':
            await executeClick(page, action, elementLabels, actionTimeout);
            break;

          case 'type':
            await executeType(page, action, elementLabels, actionTimeout, behavior, legacyTypeDelay);
            break;

          case 'scroll':
            // Capture scroll position before and after for loop detection
            context = await executeScrollWithContext(page, action);
            break;

          case 'navigate':
            await executeNavigate(page, action, actionTimeout);
            break;

          case 'hover':
            await executeHover(page, action, elementLabels, actionTimeout);
            break;

          case 'select':
            await executeSelect(page, action, elementLabels, actionTimeout);
            break;

          case 'wait':
            await executeWait(page, action, actionTimeout);
            break;

          case 'keypress':
            await executeKeyPress(page, action);
            break;

          case 'done':
            // Done action doesn't execute anything
            break;

          default:
            throw new Error(`Unknown action type: ${(action as BrowserAction).type}`);
        }

        return {
          success: true,
          newUrl: page.url(),
          durationMs: Date.now() - startTime,
          context,
        };
      } catch (error) {
        return {
          success: false,
          error: error instanceof Error ? error.message : String(error),
          newUrl: page.url(),
          durationMs: Date.now() - startTime,
          context,
        };
      }
    },
  };
}

/**
 * Get selector for an element by label ID.
 * Falls back to data attribute selector if element not found in labels.
 */
function getSelectorForElement(
  elementId: number,
  elementLabels?: ElementLabel[]
): string {
  if (elementLabels) {
    const label = elementLabels.find((l) => l.id === elementId);
    if (label?.selector) {
      return label.selector;
    }
  }

  // Fallback to data attribute
  return `[data-ai-label="${elementId}"]`;
}

/**
 * Execute a click action.
 */
async function executeClick(
  page: Page,
  action: ClickAction,
  elementLabels: ElementLabel[] | undefined,
  timeout: number
): Promise<void> {
  if (action.elementId !== undefined) {
    const selector = getSelectorForElement(action.elementId, elementLabels);

    // Wait for element to be visible
    await page.waitForSelector(selector, {
      timeout,
      state: 'visible',
    });

    const clickOptions: { button?: 'left' | 'right'; clickCount?: number } = {};

    if (action.variant === 'right') {
      clickOptions.button = 'right';
    } else if (action.variant === 'double') {
      clickOptions.clickCount = 2;
    }

    await page.click(selector, clickOptions);
  } else if (action.coordinates) {
    const { x, y } = action.coordinates;

    if (action.variant === 'double') {
      await page.mouse.dblclick(x, y);
    } else if (action.variant === 'right') {
      await page.mouse.click(x, y, { button: 'right' });
    } else {
      await page.mouse.click(x, y);
    }
  } else {
    throw new Error('Click action requires elementId or coordinates');
  }
}

/**
 * Execute a type action with human-like behavior.
 *
 * When HumanBehavior is provided:
 * - Applies pre-typing delay (pause before starting to type)
 * - Checks paste threshold (paste long text instead of typing)
 * - Types character-by-character with enhanced variance
 * - Adds micro-pauses during typing
 *
 * Falls back to legacy typeDelay or instant typing if no behavior provided.
 */
async function executeType(
  page: Page,
  action: TypeAction,
  elementLabels: ElementLabel[] | undefined,
  timeout: number,
  behavior: HumanBehavior | null,
  legacyTypeDelay: number
): Promise<void> {
  const text = action.text;

  if (action.elementId !== undefined) {
    const selector = getSelectorForElement(action.elementId, elementLabels);

    // Wait for element to be visible
    await page.waitForSelector(selector, {
      timeout,
      state: 'visible',
    });

    if (action.clearFirst) {
      // Triple-click to select all, then type
      await page.click(selector, { clickCount: 3 });
    } else {
      // Focus the element
      await page.click(selector);
    }

    // Apply human-like typing behavior if available
    if (behavior && behavior.isEnabled()) {
      await typeWithHumanBehavior(page, text, behavior, selector);
    } else if (legacyTypeDelay > 0) {
      // Legacy: uniform delay
      await page.keyboard.type(text, { delay: legacyTypeDelay });
    } else {
      // Instant typing
      await page.keyboard.type(text);
    }
  } else {
    // Type into currently focused element
    if (action.clearFirst) {
      // Select all with keyboard shortcut
      await page.keyboard.press('Control+a');
    }

    // Apply human-like typing behavior if available
    if (behavior && behavior.isEnabled()) {
      await typeWithHumanBehavior(page, text, behavior);
    } else if (legacyTypeDelay > 0) {
      // Legacy: uniform delay
      await page.keyboard.type(text, { delay: legacyTypeDelay });
    } else {
      // Instant typing
      await page.keyboard.type(text);
    }
  }
}

/**
 * Type text with human-like behavior patterns.
 *
 * This function implements realistic human typing by:
 * 1. Applying a pre-typing delay (thinking time before typing)
 * 2. Checking if text should be pasted instead of typed (for long text)
 * 3. Typing character-by-character with enhanced variance:
 *    - Common digraphs (e.g., "th", "er") are typed faster
 *    - Shifted characters (capitals, symbols) are typed slower
 *    - Numbers are slightly slower
 *    - Spaces are fast (thumb typing)
 * 4. Adding occasional micro-pauses (hesitations)
 */
async function typeWithHumanBehavior(
  page: Page,
  text: string,
  behavior: HumanBehavior,
  selector?: string
): Promise<void> {
  // 1. Apply pre-typing delay (human pause before starting to type)
  const startDelay = behavior.getTypingStartDelay();
  if (startDelay > 0) {
    await sleep(startDelay);
  }

  // 2. Check paste threshold - paste long text instead of typing
  if (behavior.shouldPaste(text.length)) {
    if (selector) {
      // Use fill() for pasting - it's instant and simulates paste
      await page.fill(selector, text);
    } else {
      // For focused element, use clipboard simulation
      // First clear any selection, then "paste" via keyboard.type with no delay
      await page.keyboard.type(text, { delay: 0 });
    }
    return;
  }

  // 3. Type character-by-character with enhanced variance
  behavior.resetTypingState();

  for (const char of text) {
    // Type the character
    await page.keyboard.type(char, { delay: 0 });

    // Get delay for this specific character (considers digraphs, shift, etc.)
    const delay = behavior.getTypingDelayForChar(char);
    if (delay > 0) {
      await sleep(delay);
    }

    // 4. Occasional micro-pause during typing (simulates thinking/hesitation)
    if (behavior.shouldMicroPause()) {
      await sleep(behavior.getMicroPauseDuration());
    }
  }
}

/**
 * Get current scroll position of the page.
 * Note: This runs in browser context via Playwright's evaluate.
 */
async function getScrollPosition(page: Page): Promise<{ x: number; y: number }> {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  return page.evaluate((): { x: number; y: number } => {
    // Browser globals are available in evaluate context
    // Using any to avoid DOM typing conflicts with Node.js environment
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const w = window as any;
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const d = document as any;
    return {
      x: w.scrollX ?? w.pageXOffset ?? d.documentElement?.scrollLeft ?? 0,
      y: w.scrollY ?? w.pageYOffset ?? d.documentElement?.scrollTop ?? 0,
    };
  });
}

/**
 * Execute a scroll action and capture context for loop detection.
 * Returns context with scroll position before/after to detect if scroll had effect.
 */
async function executeScrollWithContext(
  page: Page,
  action: ScrollAction
): Promise<ActionExecutionContext> {
  // Capture position before scroll
  const positionBefore = await getScrollPosition(page);

  // Execute the scroll
  await executeScroll(page, action);

  // Small delay to let scroll settle
  await new Promise((resolve) => setTimeout(resolve, 50));

  // Capture position after scroll
  const positionAfter = await getScrollPosition(page);

  return {
    scroll: {
      positionBefore,
      positionAfter,
    },
  };
}

/**
 * Execute a scroll action.
 */
async function executeScroll(page: Page, action: ScrollAction): Promise<void> {
  const viewport = page.viewportSize();
  const defaultAmount = viewport
    ? (action.direction === 'up' || action.direction === 'down'
        ? viewport.height * 0.8
        : viewport.width * 0.8)
    : 500;

  const amount = action.amount ?? defaultAmount;

  let deltaX = 0;
  let deltaY = 0;

  switch (action.direction) {
    case 'down':
      deltaY = amount;
      break;
    case 'up':
      deltaY = -amount;
      break;
    case 'right':
      deltaX = amount;
      break;
    case 'left':
      deltaX = -amount;
      break;
  }

  await page.mouse.wheel(deltaX, deltaY);
}

/**
 * Execute a navigate action.
 */
async function executeNavigate(
  page: Page,
  action: NavigateAction,
  timeout: number
): Promise<void> {
  await page.goto(action.url, {
    timeout,
    waitUntil: 'domcontentloaded',
  });
}

/**
 * Execute a hover action.
 */
async function executeHover(
  page: Page,
  action: HoverAction,
  elementLabels: ElementLabel[] | undefined,
  timeout: number
): Promise<void> {
  if (action.elementId !== undefined) {
    const selector = getSelectorForElement(action.elementId, elementLabels);

    await page.waitForSelector(selector, {
      timeout,
      state: 'visible',
    });

    await page.hover(selector);
  } else if (action.coordinates) {
    const { x, y } = action.coordinates;
    await page.mouse.move(x, y);
  } else {
    throw new Error('Hover action requires elementId or coordinates');
  }
}

/**
 * Execute a select action.
 */
async function executeSelect(
  page: Page,
  action: SelectAction,
  elementLabels: ElementLabel[] | undefined,
  timeout: number
): Promise<void> {
  const selector = getSelectorForElement(action.elementId, elementLabels);

  await page.waitForSelector(selector, {
    timeout,
    state: 'visible',
  });

  await page.selectOption(selector, action.value);
}

/**
 * Execute a wait action.
 */
async function executeWait(
  page: Page,
  action: WaitAction,
  timeout: number
): Promise<void> {
  if (action.selector) {
    await page.waitForSelector(action.selector, {
      timeout,
      state: 'visible',
    });
  } else {
    const waitTime = action.ms ?? DEFAULT_WAIT_MS;
    await new Promise((resolve) => setTimeout(resolve, waitTime));
  }
}

/**
 * Execute a keypress action.
 */
async function executeKeyPress(page: Page, action: KeyPressAction): Promise<void> {
  let key = action.key;

  // Build key combination with modifiers
  if (action.modifiers && action.modifiers.length > 0) {
    const modifierString = action.modifiers.join('+');
    key = `${modifierString}+${key}`;
  }

  await page.keyboard.press(key);
}

/**
 * Create a mock ActionExecutor for testing.
 *
 * @returns Mock executor that records calls
 */
export function createMockActionExecutor(): ActionExecutorInterface & {
  getCalls(): Array<{ action: BrowserAction; elementLabels?: ElementLabel[] }>;
  clearCalls(): void;
  setFailMode(shouldFail: boolean, errorMessage?: string): void;
} {
  const calls: Array<{ action: BrowserAction; elementLabels?: ElementLabel[] }> = [];
  let shouldFail = false;
  let failureMessage = 'Mock executor failure';

  return {
    async execute(
      _page: Page,
      action: BrowserAction,
      elementLabels?: ElementLabel[]
    ): Promise<ActionExecutionResult> {
      calls.push({ action, elementLabels });

      if (shouldFail) {
        return {
          success: false,
          error: failureMessage,
          durationMs: 10,
        };
      }

      return {
        success: true,
        newUrl: 'https://example.com',
        durationMs: 10,
      };
    },

    getCalls() {
      return [...calls];
    },

    clearCalls() {
      calls.length = 0;
    },

    setFailMode(fail: boolean, message?: string) {
      shouldFail = fail;
      if (message) {
        failureMessage = message;
      }
    },
  };
}
