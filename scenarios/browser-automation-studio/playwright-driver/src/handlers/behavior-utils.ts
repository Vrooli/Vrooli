/**
 * Shared Behavior Utilities
 *
 * Common utilities for applying human-like behavior settings across handlers.
 * Centralizes behavior extraction and application logic.
 */

import type { BrowserContext } from 'playwright';
import type { BehaviorSettings } from '../types';
import { HumanBehavior, sleep } from '../browser-profile';
import { BEHAVIOR_SETTINGS_KEY } from '../session/context-builder';
import type { HandlerContext } from './base';

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
  page: import('playwright').Page,
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

  // If already at target or very close, just scroll directly
  if (totalDistance < 10) {
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

  // Calculate step delay
  const minDelay = options.minStepDelayMs ?? 10;
  const maxDelay = options.maxStepDelayMs ?? 30;

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
  page: import('playwright').Page,
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

  // Estimate scroll duration: ~300ms base + 1ms per 10 pixels
  const estimatedDuration = Math.min(1000, 300 + distance / 10);
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
  page: import('playwright').Page,
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

  // Generate natural mouse path
  const steps = options.steps ?? 15;
  const path = behavior.generateMousePath({ x: fromX, y: fromY }, { x: targetX, y: targetY }, steps);

  if (path.length <= 1) {
    await page.mouse.move(targetX, targetY);
    return;
  }

  // Calculate timing
  const totalDuration = options.durationMs ?? 150;
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
  page: import('playwright').Page,
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
export { sleep } from '../browser-profile';
