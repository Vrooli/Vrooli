/**
 * Human-Like Behavior Module
 *
 * Provides realistic timing and movement patterns for browser automation.
 */

import type { BehaviorSettings } from '../types/browser-profile';

export interface Point {
  x: number;
  y: number;
}

/**
 * Human-like behavior controller.
 * Provides timing delays and mouse movement simulation.
 */
export class HumanBehavior {
  private settings: Required<BehaviorSettings>;

  constructor(settings: BehaviorSettings) {
    // Apply defaults for any missing settings
    this.settings = {
      typing_delay_min: settings.typing_delay_min ?? 0,
      typing_delay_max: settings.typing_delay_max ?? 0,
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
   * Get a random delay for typing a character.
   */
  getTypingDelay(): number {
    return this.randomBetween(this.settings.typing_delay_min, this.settings.typing_delay_max);
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

/**
 * Sleep for specified milliseconds.
 */
export function sleep(ms: number): Promise<void> {
  if (ms <= 0) return Promise.resolve();
  return new Promise((resolve) => setTimeout(resolve, ms));
}

/**
 * Type text character by character with random delays.
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
