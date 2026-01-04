/**
 * Human-Like Behavior Module
 *
 * Provides realistic timing and movement patterns for browser automation.
 * Includes enhanced typing variance that simulates human typing patterns
 * based on key positions, digraphs, and character types.
 */

import type { BehaviorSettings } from '../types/browser-profile';
import { sleep } from '../utils/timing';

export interface Point {
  x: number;
  y: number;
}

/**
 * Common digraphs that are typed faster due to muscle memory.
 * These are the most common two-letter combinations in English.
 * Stored as a Set for O(1) lookup.
 */
const FAST_DIGRAPHS = new Set([
  'th', 'he', 'in', 'er', 'an', 're', 'on', 'en', 'at', 'es',
  'ed', 'to', 'it', 'or', 'st', 'is', 'ar', 'nd', 'ti', 'ng',
  'te', 'al', 'nt', 'as', 'ha', 'ou', 'se', 'le', 'of', 'ea',
  've', 'me', 'de', 'hi', 'ri', 'ro', 'ic', 'ne', 'ea', 'ra',
  'ce', 'li', 'll', 'be', 'ma', 'si', 'om', 'ur', 'ca', 'el',
  'ta', 'la', 'ns', 'di', 'fo', 'ho', 'pe', 'ec', 'pr', 'no',
  'ct', 'us', 'ac', 'ot', 'il', 'tr', 'ly', 'nc', 'et', 'ut',
  'ss', 'so', 'rs', 'un', 'lo', 'wa', 'ge', 'ie', 'wh', 'ee',
  'wi', 'em', 'ad', 'ol', 'rt', 'po', 'we', 'na', 'ul', 'ni',
  'ts', 'mo', 'ow', 'pa', 'im', 'pl', 'ay', 'ds', 'id', 'am',
]);

/**
 * Shifted characters (require holding shift key) - slower to type.
 */
const SHIFTED_CHARS = new Set([
  '~', '!', '@', '#', '$', '%', '^', '&', '*', '(', ')', '_', '+',
  '{', '}', '|', ':', '"', '<', '>', '?',
  'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M',
  'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z',
]);

/**
 * Number characters - slightly slower (hand movement to number row).
 */
const NUMBER_CHARS = new Set(['0', '1', '2', '3', '4', '5', '6', '7', '8', '9']);

/**
 * Uncommon symbols - slower due to unfamiliarity.
 */
const UNCOMMON_SYMBOLS = new Set(['`', '~', '^', '{', '}', '[', ']', '|', '\\', '<', '>']);

/**
 * Human-like behavior controller.
 * Provides timing delays and mouse movement simulation with
 * enhanced typing variance that mimics real human typing patterns.
 */
export class HumanBehavior {
  private settings: Required<BehaviorSettings>;
  private lastChar: string | null = null;

  constructor(settings: BehaviorSettings) {
    // Apply defaults for any missing settings
    this.settings = {
      typing_delay_min: settings.typing_delay_min ?? 0,
      typing_delay_max: settings.typing_delay_max ?? 0,
      typing_start_delay_min: settings.typing_start_delay_min ?? 0,
      typing_start_delay_max: settings.typing_start_delay_max ?? 0,
      typing_paste_threshold: settings.typing_paste_threshold ?? 0,
      typing_variance_enabled: settings.typing_variance_enabled ?? true,
      mouse_movement_style: settings.mouse_movement_style ?? 'linear',
      mouse_jitter_amount: settings.mouse_jitter_amount ?? 0,
      click_delay_min: settings.click_delay_min ?? 0,
      click_delay_max: settings.click_delay_max ?? 0,
      scroll_style: settings.scroll_style ?? 'smooth',
      scroll_speed_min: settings.scroll_speed_min ?? 100,
      scroll_speed_max: settings.scroll_speed_max ?? 300,
      micro_pause_enabled: settings.micro_pause_enabled ?? false,
      micro_pause_min_ms: settings.micro_pause_min_ms ?? 0,
      micro_pause_max_ms: settings.micro_pause_max_ms ?? 0,
      micro_pause_frequency: settings.micro_pause_frequency ?? 0,
    };
  }

  /**
   * Check if any behavior modifications are enabled.
   */
  isEnabled(): boolean {
    return (
      this.settings.typing_delay_max > 0 ||
      this.settings.click_delay_max > 0 ||
      this.settings.micro_pause_enabled ||
      this.settings.mouse_movement_style !== 'linear'
    );
  }

  /**
   * Reset typing state. Call this at the start of a new typing session.
   */
  resetTypingState(): void {
    this.lastChar = null;
  }

  /**
   * Get a random delay before starting to type in a field.
   * Simulates the human pause before beginning to type.
   */
  getTypingStartDelay(): number {
    return this.randomBetween(
      this.settings.typing_start_delay_min,
      this.settings.typing_start_delay_max
    );
  }

  /**
   * Determine if text should be pasted instead of typed.
   * Long text is often pasted by humans rather than typed.
   *
   * @param textLength - Length of the text to be entered
   * @returns true if text should be pasted, false if typed
   */
  shouldPaste(textLength: number): boolean {
    const threshold = this.settings.typing_paste_threshold;

    // -1 means always paste
    if (threshold === -1) return true;

    // 0 or undefined means always type
    if (threshold <= 0) return false;

    // Otherwise, paste if text exceeds threshold
    return textLength > threshold;
  }

  /**
   * Get a random delay for typing a character (basic version).
   * Use getTypingDelayForChar() for enhanced variance.
   */
  getTypingDelay(): number {
    return this.randomBetween(this.settings.typing_delay_min, this.settings.typing_delay_max);
  }

  /**
   * Get a delay for typing a specific character with enhanced variance.
   * Takes into account:
   * - Digraph patterns (common pairs are faster)
   * - Shifted characters (slower)
   * - Numbers (slightly slower)
   * - Uncommon symbols (slower)
   * - Random variance for natural feel
   *
   * @param char - The character being typed
   * @returns Delay in milliseconds before typing the next character
   */
  getTypingDelayForChar(char: string): number {
    const baseDelay = this.getTypingDelay();

    // If variance is disabled or base delay is 0, return base delay
    if (!this.settings.typing_variance_enabled || baseDelay === 0) {
      this.lastChar = char;
      return baseDelay;
    }

    let multiplier = 1.0;

    // Check for fast digraph (previous + current char)
    if (this.lastChar) {
      const digraph = (this.lastChar + char).toLowerCase();
      if (FAST_DIGRAPHS.has(digraph)) {
        // Fast digraphs: 60-80% of normal delay
        multiplier *= this.randomBetween(60, 80) / 100;
      }
    }

    // Shifted characters are slower (need to hold shift)
    if (SHIFTED_CHARS.has(char)) {
      // Shifted: 130-160% of normal delay
      multiplier *= this.randomBetween(130, 160) / 100;
    }

    // Numbers are slightly slower (hand movement)
    if (NUMBER_CHARS.has(char)) {
      // Numbers: 110-130% of normal delay
      multiplier *= this.randomBetween(110, 130) / 100;
    }

    // Uncommon symbols are slower (unfamiliarity)
    if (UNCOMMON_SYMBOLS.has(char)) {
      // Uncommon: 140-180% of normal delay
      multiplier *= this.randomBetween(140, 180) / 100;
    }

    // Space is often typed quickly (thumb)
    if (char === ' ') {
      multiplier *= this.randomBetween(70, 90) / 100;
    }

    // Add ±15% random variance for natural feel
    const variance = this.randomBetween(85, 115) / 100;
    multiplier *= variance;

    // Update last character for next digraph check
    this.lastChar = char;

    // Apply multiplier and clamp to reasonable range
    const adjustedDelay = Math.round(baseDelay * multiplier);

    // Ensure delay is at least 5ms (realistic minimum) and not absurdly long
    const minDelay = Math.min(5, this.settings.typing_delay_min);
    const maxDelay = Math.max(this.settings.typing_delay_max * 2, 500);

    return Math.max(minDelay, Math.min(maxDelay, adjustedDelay));
  }

  /**
   * Get a random delay before clicking.
   */
  getClickDelay(): number {
    return this.randomBetween(this.settings.click_delay_min, this.settings.click_delay_max);
  }

  /**
   * Get scroll speed for this step.
   */
  getScrollSpeed(): number {
    return this.randomBetween(this.settings.scroll_speed_min, this.settings.scroll_speed_max);
  }

  /**
   * Determine if we should insert a micro-pause.
   */
  shouldMicroPause(): boolean {
    if (!this.settings.micro_pause_enabled) return false;
    return Math.random() < this.settings.micro_pause_frequency;
  }

  /**
   * Get the duration for a micro-pause.
   */
  getMicroPauseDuration(): number {
    return this.randomBetween(this.settings.micro_pause_min_ms, this.settings.micro_pause_max_ms);
  }

  /**
   * Get the mouse movement style.
   */
  getMouseMovementStyle(): 'linear' | 'bezier' | 'natural' {
    return this.settings.mouse_movement_style as 'linear' | 'bezier' | 'natural';
  }

  /**
   * Get the scroll style.
   */
  getScrollStyle(): 'smooth' | 'stepped' {
    return this.settings.scroll_style as 'smooth' | 'stepped';
  }

  /**
   * Generate a path for mouse movement from one point to another.
   */
  generateMousePath(from: Point, to: Point, steps: number = 20): Point[] {
    const style = this.settings.mouse_movement_style;
    const jitter = this.settings.mouse_jitter_amount;

    switch (style) {
      case 'bezier':
        return this.bezierPath(from, to, steps, jitter);
      case 'natural':
        return this.naturalPath(from, to, steps, jitter);
      default:
        return this.linearPath(from, to, steps);
    }
  }

  /**
   * Linear path - direct movement (fastest).
   */
  private linearPath(from: Point, to: Point, steps: number): Point[] {
    const points: Point[] = [];
    for (let i = 0; i <= steps; i++) {
      const t = i / steps;
      points.push({
        x: Math.round(from.x + (to.x - from.x) * t),
        y: Math.round(from.y + (to.y - from.y) * t),
      });
    }
    return points;
  }

  /**
   * Bezier curve path - smooth curved movement.
   */
  private bezierPath(from: Point, to: Point, steps: number, jitter: number): Point[] {
    const points: Point[] = [];

    // Generate control points for cubic bezier
    const distance = Math.sqrt(Math.pow(to.x - from.x, 2) + Math.pow(to.y - from.y, 2));
    const controlOffset = Math.min(distance * 0.3, 100);

    const cp1: Point = {
      x: from.x + (to.x - from.x) * 0.25 + this.randomBetween(-controlOffset, controlOffset),
      y: from.y + (to.y - from.y) * 0.1 + this.randomBetween(-controlOffset / 2, controlOffset / 2),
    };

    const cp2: Point = {
      x: from.x + (to.x - from.x) * 0.75 + this.randomBetween(-controlOffset, controlOffset),
      y: from.y + (to.y - from.y) * 0.9 + this.randomBetween(-controlOffset / 2, controlOffset / 2),
    };

    for (let i = 0; i <= steps; i++) {
      const t = i / steps;
      const point = this.cubicBezier(from, cp1, cp2, to, t);

      // Add jitter
      if (jitter > 0 && i > 0 && i < steps) {
        point.x += this.randomBetween(-jitter, jitter);
        point.y += this.randomBetween(-jitter, jitter);
      }

      points.push({
        x: Math.round(point.x),
        y: Math.round(point.y),
      });
    }

    return points;
  }

  /**
   * Natural path - bezier with variable speed (slower at start/end).
   */
  private naturalPath(from: Point, to: Point, steps: number, jitter: number): Point[] {
    // Generate more bezier points for smoother sampling
    const bezierPoints = this.bezierPath(from, to, steps * 2, jitter);
    const points: Point[] = [];

    // Use easing to select points (more points at start and end)
    for (let i = 0; i <= steps; i++) {
      const t = i / steps;
      const easedT = this.easeInOutQuad(t);
      const index = Math.min(Math.floor(easedT * (bezierPoints.length - 1)), bezierPoints.length - 1);
      points.push(bezierPoints[index]);
    }

    return points;
  }

  /**
   * Calculate point on cubic bezier curve.
   */
  private cubicBezier(p0: Point, p1: Point, p2: Point, p3: Point, t: number): Point {
    const mt = 1 - t;
    const mt2 = mt * mt;
    const mt3 = mt2 * mt;
    const t2 = t * t;
    const t3 = t2 * t;

    return {
      x: mt3 * p0.x + 3 * mt2 * t * p1.x + 3 * mt * t2 * p2.x + t3 * p3.x,
      y: mt3 * p0.y + 3 * mt2 * t * p1.y + 3 * mt * t2 * p2.y + t3 * p3.y,
    };
  }

  /**
   * Ease in-out quadratic function for natural speed variation.
   */
  private easeInOutQuad(t: number): number {
    return t < 0.5 ? 2 * t * t : 1 - Math.pow(-2 * t + 2, 2) / 2;
  }

  /**
   * Generate random number between min and max (inclusive).
   */
  private randomBetween(min: number, max: number): number {
    if (min >= max) return min;
    return Math.floor(Math.random() * (max - min + 1)) + min;
  }
}

// Re-export sleep from utils for backward compatibility
export { sleep } from '../utils/timing';

/**
 * Type text character by character with random delays.
 * Uses basic delay without enhanced variance.
 */
export async function typeWithDelay(
  typeChar: (char: string) => Promise<void>,
  text: string,
  behavior: HumanBehavior
): Promise<void> {
  for (const char of text) {
    await typeChar(char);
    const delay = behavior.getTypingDelay();
    if (delay > 0) {
      await sleep(delay);
    }

    // Occasional micro-pause during typing
    if (behavior.shouldMicroPause()) {
      await sleep(behavior.getMicroPauseDuration());
    }
  }
}

/**
 * Type text character by character with enhanced variance.
 * Uses character-aware delays for realistic typing patterns.
 */
export async function typeWithEnhancedVariance(
  typeChar: (char: string) => Promise<void>,
  text: string,
  behavior: HumanBehavior
): Promise<void> {
  // Reset typing state for fresh digraph tracking
  behavior.resetTypingState();

  for (const char of text) {
    await typeChar(char);
    const delay = behavior.getTypingDelayForChar(char);
    if (delay > 0) {
      await sleep(delay);
    }

    // Occasional micro-pause during typing (simulate thinking/hesitation)
    if (behavior.shouldMicroPause()) {
      await sleep(behavior.getMicroPauseDuration());
    }
  }
}

/**
 * Move mouse along a path with natural timing.
 */
export async function moveMouseAlongPath(
  moveMouse: (x: number, y: number) => Promise<void>,
  path: Point[],
  totalDurationMs: number = 200
): Promise<void> {
  if (path.length <= 1) return;

  const stepDuration = totalDurationMs / (path.length - 1);

  for (let i = 1; i < path.length; i++) {
    const point = path[i];
    await moveMouse(point.x, point.y);
    await sleep(stepDuration);
  }
}
