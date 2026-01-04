/**
 * Shared Behavior Utilities
 *
 * Common utilities for applying human-like behavior settings across handlers.
 * Centralizes behavior extraction and application logic.
 *
 * CONTROL SURFACE:
 * - Timeout resolution: Priority-based resolution with clear fallback chain
 * - Behavior settings: Human-like delays and pauses for anti-detection
 * - Mouse movement: Natural paths for realistic interaction patterns
 */

import type { BrowserContext } from 'rebrowser-playwright';
import type { BehaviorSettings } from '../types';
import { HumanBehavior, BEHAVIOR_SETTINGS_KEY } from '../browser-profile';
import { sleep } from '../utils';
import type { HandlerContext } from './base';
import type { Config } from '../config';
import {
  SCROLL_CLOSE_ENOUGH_THRESHOLD_PX,
  SCROLL_STEP_MIN_DELAY_MS,
  SCROLL_STEP_MAX_DELAY_MS,
  SMOOTH_SCROLL_BASE_DURATION_MS,
  SMOOTH_SCROLL_DISTANCE_FACTOR,
  SMOOTH_SCROLL_MAX_DURATION_MS,
  MOUSE_PATH_DEFAULT_STEPS,
  MOUSE_MOVEMENT_DEFAULT_DURATION_MS,
} from '../constants';

// =============================================================================
// TIMEOUT RESOLUTION - Decision Boundary for Timeout Selection
// =============================================================================

/**
 * Timeout resolution category.
 *
 * DECISION BOUNDARY: Different operations have different default timeouts
 * based on their expected duration and failure characteristics.
 *
 * Categories:
 * - default: General operations (click, type, hover) - 30s
 * - navigation: Page loads, navigation - 45s (networks can be slow)
 * - wait: Explicit waits, waitForSelector - 30s
 * - assertion: Assertions, checks - 15s (faster feedback on failures)
 * - replay: Replay mode actions - 10s (should be fast, user is watching)
 */
export type TimeoutCategory = 'default' | 'navigation' | 'wait' | 'assertion' | 'replay';

/**
 * Resolve timeout value using priority-based fallback chain.
 *
 * DECISION LOGIC (in priority order):
 * 1. Explicit parameter value (caller-specified)
 * 2. Config value for the category
 * 3. Hardcoded constant default (last resort)
 *
 * This design allows:
 * - Per-instruction override (params.timeoutMs)
 * - Per-deployment tuning (config)
 * - Sane defaults (constants)
 *
 * @param paramsTimeout - Timeout from instruction parameters (highest priority)
 * @param config - Configuration object with execution timeouts
 * @param category - The type of operation for category-based defaults
 * @returns Resolved timeout in milliseconds
 */
export function resolveTimeout(
  paramsTimeout: number | undefined,
  config: Config,
  category: TimeoutCategory = 'default'
): number {
  // Priority 1: Explicit parameter takes precedence
  if (paramsTimeout !== undefined && paramsTimeout > 0) {
    return paramsTimeout;
  }

  // Priority 2: Config value based on category
  switch (category) {
    case 'navigation':
      return config.execution.navigationTimeoutMs;
    case 'wait':
      return config.execution.waitTimeoutMs;
    case 'assertion':
      return config.execution.assertionTimeoutMs;
    case 'replay':
      return config.execution.replayActionTimeoutMs;
    case 'default':
    default:
      return config.execution.defaultTimeoutMs;
  }
}

/**
 * Resolve timeout from context with simplified signature.
 *
 * Convenience wrapper for handlers that have access to HandlerContext.
 *
 * @param paramsTimeout - Timeout from instruction parameters
 * @param context - Handler context containing config
 * @param category - The type of operation
 * @returns Resolved timeout in milliseconds
 */
export function resolveTimeoutFromContext(
  paramsTimeout: number | undefined,
  context: HandlerContext,
  category: TimeoutCategory = 'default'
): number {
  return resolveTimeout(paramsTimeout, context.config, category);
}

// =============================================================================
// BEHAVIOR SETTINGS - Human-like Behavior Utilities
// =============================================================================

/**
 * Get human behavior settings from handler context.
 *
 * Returns null if behavior is disabled or not configured.
 */
export function getBehaviorFromContext(context: HandlerContext): HumanBehavior | null {
  const browserContext = context.page.context();
  return getBehaviorFromBrowserContext(browserContext);
}

/**
 * Get human behavior settings from browser context directly.
 *
 * Useful when you have direct access to BrowserContext but not HandlerContext.
 */
export function getBehaviorFromBrowserContext(browserContext: BrowserContext): HumanBehavior | null {
  const settings = (browserContext as any)[BEHAVIOR_SETTINGS_KEY] as BehaviorSettings | undefined;
  if (!settings) return null;
  const behavior = new HumanBehavior(settings);
  return behavior.isEnabled() ? behavior : null;
}

/**
 * Apply pre-action delay with optional micro-pause.
 *
 * Standard pattern for human-like delay before an action.
 * Use the getDelay function to get the appropriate delay for the action type.
 *
 * @param behavior - HumanBehavior instance (can be null for no-op)
 * @param getDelay - Function to get delay from behavior (e.g., behavior.getClickDelay)
 * @returns Promise that resolves after delays are applied
 */
export async function applyPreActionDelay(
  behavior: HumanBehavior | null,
  getDelay: (b: HumanBehavior) => number
): Promise<void> {
  if (!behavior) return;

  const delay = getDelay(behavior);
  if (delay > 0) {
    await sleep(delay);
  }

  // Optional micro-pause
  if (behavior.shouldMicroPause()) {
    await sleep(behavior.getMicroPauseDuration());
  }
}

/**
 * Apply post-action micro-pause if enabled.
 *
 * Use after an action completes for natural pacing.
 *
 * @param behavior - HumanBehavior instance (can be null for no-op)
 */
export async function applyPostActionPause(behavior: HumanBehavior | null): Promise<void> {
  if (!behavior) return;

  if (behavior.shouldMicroPause()) {
    await sleep(behavior.getMicroPauseDuration());
  }
}

/**
 * Execute stepped scroll with human-like behavior.
 *
 * Scrolls in increments with delays between steps for natural appearance.
 *
 * @param page - Playwright Page instance
 * @param targetX - Target scroll X position
 * @param targetY - Target scroll Y position
 * @param behavior - HumanBehavior instance (can be null for instant scroll)
 * @param options - Additional scroll options
 */
export async function executeHumanScroll(
  page: import('rebrowser-playwright').Page,
  targetX: number,
  targetY: number,
  behavior: HumanBehavior | null,
  options: {
    /** Minimum step delay in ms */
    minStepDelayMs?: number;
    /** Maximum step delay in ms */
    maxStepDelayMs?: number;
  } = {}
): Promise<void> {
  // If no behavior or scroll speed is 0, do instant scroll
  if (!behavior) {
    await page.evaluate(
      ([x, y]) => {
        // @ts-expect-error - window is available in browser context
        window.scrollTo(x, y);
      },
      [targetX, targetY]
    );
    return;
  }

  // Get current scroll position
  const currentPosition = await page.evaluate(() => ({
    // @ts-expect-error - window is available in browser context
    x: window.scrollX || window.pageXOffset,
    // @ts-expect-error - window is available in browser context
    y: window.scrollY || window.pageYOffset,
  }));

  const deltaX = targetX - currentPosition.x;
  const deltaY = targetY - currentPosition.y;
  const totalDistance = Math.sqrt(deltaX * deltaX + deltaY * deltaY);

  // DECISION: If already at target or very close, use instant scroll
  // Threshold defined in constants.ts for tuning
  if (totalDistance < SCROLL_CLOSE_ENOUGH_THRESHOLD_PX) {
    await page.evaluate(
      ([x, y]) => {
        // @ts-expect-error - window is available in browser context
        window.scrollTo(x, y);
      },
      [targetX, targetY]
    );
    return;
  }

  // Calculate step size based on scroll speed settings
  const scrollSpeed = behavior.getScrollSpeed();
  const steps = Math.max(1, Math.ceil(totalDistance / scrollSpeed));
  const stepX = deltaX / steps;
  const stepY = deltaY / steps;

  // Calculate step delay using constants from constants.ts
  const minDelay = options.minStepDelayMs ?? SCROLL_STEP_MIN_DELAY_MS;
  const maxDelay = options.maxStepDelayMs ?? SCROLL_STEP_MAX_DELAY_MS;

  // Execute stepped scroll
  for (let i = 1; i <= steps; i++) {
    const nextX = Math.round(currentPosition.x + stepX * i);
    const nextY = Math.round(currentPosition.y + stepY * i);

    await page.evaluate(
      ([x, y]) => {
        // @ts-expect-error - window is available in browser context
        window.scrollTo(x, y);
      },
      [nextX, nextY]
    );

    // Add delay between steps (except for last step)
    if (i < steps) {
      const stepDelay = minDelay + Math.random() * (maxDelay - minDelay);
      await sleep(stepDelay);

      // Occasional micro-pause during scroll
      if (behavior.shouldMicroPause()) {
        await sleep(behavior.getMicroPauseDuration());
      }
    }
  }
}

/**
 * Execute smooth scroll using CSS scroll-behavior.
 *
 * Uses native browser smooth scrolling for a natural appearance.
 * Falls back to stepped scroll if smooth scrolling is not supported.
 *
 * @param page - Playwright Page instance
 * @param targetX - Target scroll X position
 * @param targetY - Target scroll Y position
 * @param behavior - HumanBehavior instance for micro-pauses
 */
export async function executeSmoothScroll(
  page: import('rebrowser-playwright').Page,
  targetX: number,
  targetY: number,
  behavior: HumanBehavior | null
): Promise<void> {
  // Apply pre-scroll micro-pause if enabled
  if (behavior?.shouldMicroPause()) {
    await sleep(behavior.getMicroPauseDuration());
  }

  // Use native smooth scroll
  await page.evaluate(
    ([x, y]) => {
      // @ts-expect-error - window is available in browser context
      window.scrollTo({
        left: x,
        top: y,
        behavior: 'smooth',
      });
    },
    [targetX, targetY]
  );

  // Wait for scroll to complete (approximate based on distance)
  const currentPosition = await page.evaluate(() => ({
    // @ts-expect-error - window is available in browser context
    x: window.scrollX || window.pageXOffset,
    // @ts-expect-error - window is available in browser context
    y: window.scrollY || window.pageYOffset,
  }));

  const distance = Math.sqrt(
    Math.pow(targetX - currentPosition.x, 2) + Math.pow(targetY - currentPosition.y, 2)
  );

  // DECISION: Estimate scroll duration based on distance
  // Formula: base + (distance / factor), capped at max
  // Constants defined in constants.ts for tuning
  const estimatedDuration = Math.min(
    SMOOTH_SCROLL_MAX_DURATION_MS,
    SMOOTH_SCROLL_BASE_DURATION_MS + distance / SMOOTH_SCROLL_DISTANCE_FACTOR
  );
  await sleep(estimatedDuration);
}

/**
 * Move mouse along a human-like path to target coordinates.
 *
 * Uses HumanBehavior's path generation for natural mouse movement.
 *
 * @param page - Playwright Page instance
 * @param targetX - Target X coordinate
 * @param targetY - Target Y coordinate
 * @param behavior - HumanBehavior instance for path generation
 * @param options - Movement options
 */
export async function moveMouseNaturally(
  page: import('rebrowser-playwright').Page,
  targetX: number,
  targetY: number,
  behavior: HumanBehavior | null,
  options: {
    /** Starting X coordinate (if known) */
    fromX?: number;
    /** Starting Y coordinate (if known) */
    fromY?: number;
    /** Total duration for movement in ms */
    durationMs?: number;
    /** Number of steps in the path */
    steps?: number;
  } = {}
): Promise<void> {
  // If no behavior or linear style, just move directly
  if (!behavior || behavior.getMouseMovementStyle() === 'linear') {
    await page.mouse.move(targetX, targetY);
    return;
  }

  // Get current mouse position or use provided starting point
  const fromX = options.fromX ?? 0;
  const fromY = options.fromY ?? 0;

  // Generate natural mouse path using constants from constants.ts
  const steps = options.steps ?? MOUSE_PATH_DEFAULT_STEPS;
  const path = behavior.generateMousePath({ x: fromX, y: fromY }, { x: targetX, y: targetY }, steps);

  if (path.length <= 1) {
    await page.mouse.move(targetX, targetY);
    return;
  }

  // Calculate timing using constants from constants.ts
  const totalDuration = options.durationMs ?? MOUSE_MOVEMENT_DEFAULT_DURATION_MS;
  const stepDuration = totalDuration / (path.length - 1);

  // Move along path
  for (let i = 1; i < path.length; i++) {
    const point = path[i];
    await page.mouse.move(point.x, point.y);
    if (stepDuration > 0 && i < path.length - 1) {
      await sleep(stepDuration);
    }
  }
}

/**
 * Get the center coordinates of an element.
 *
 * @param page - Playwright Page instance
 * @param selector - Element selector
 * @param timeout - Wait timeout in ms
 * @returns Center coordinates or null if element not found
 */
export async function getElementCenter(
  page: import('rebrowser-playwright').Page,
  selector: string,
  timeout?: number
): Promise<{ x: number; y: number } | null> {
  try {
    const element = await page.waitForSelector(selector, { timeout, state: 'visible' });
    if (!element) return null;

    const box = await element.boundingBox();
    if (!box) return null;

    return {
      x: box.x + box.width / 2,
      y: box.y + box.height / 2,
    };
  } catch {
    return null;
  }
}

// Re-export sleep for convenience
export { sleep } from '../utils';
