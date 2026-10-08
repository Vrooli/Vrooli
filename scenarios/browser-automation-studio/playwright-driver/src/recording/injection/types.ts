/** Types and diagnostics contracts for recording init-script injection. */

import type { BrowserContext, Page } from 'rebrowser-playwright';
import type winston from 'winston';

// =============================================================================
// Core Types
// =============================================================================

/** The only recording injection path supported by the driver. */
export type InjectionStrategyName = 'init-script';

/**
 * Result of a script injection attempt.
 */
export interface InjectionResult {
  /** Whether injection succeeded */
  success: boolean;
  /** Which injection path performed the injection */
  strategy: InjectionStrategyName;
  /** Error message if injection failed */
  error?: string;
  /** When the injection occurred */
  timestamp: string;
  /** Additional injection metadata */
  metadata?: Record<string, unknown>;
}

/**
 * Statistics tracking for an injection strategy.
 * Useful for diagnostics and debugging.
 */
export interface InjectionStrategyStats {
  /** Number of injection attempts */
  attempted: number;
  /** Number of successful injections */
  successful: number;
  /** Number of failed injections */
  failed: number;
  /** Average injection time in milliseconds */
  avgInjectionTimeMs: number;
  /** ISO timestamp of last injection, or null if none */
  lastInjectionAt: string | null;
}

/**
 * Options for initializing an injection strategy.
 */
export interface InjectionStrategyOptions {
  /** Name of the binding for event communication */
  bindingName: string;
  /** Logger instance for diagnostics */
  logger: winston.Logger;
  /** Enable verbose diagnostics logging */
  diagnosticsEnabled?: boolean;
  /** Callback when first successful injection occurs */
  onFirstInjection?: () => void;
}

// =============================================================================
// Strategy Interface
// =============================================================================

/**
 * Contract used by the init-script injector and its diagnostics.
 */
export interface InjectionStrategy {
  /**
   * Unique name identifying this strategy.
   */
  readonly name: InjectionStrategyName;

  /**
   * Initialize the strategy on a browser context.
   *
   * This is called once per context and sets up any context-level hooks
   * needed for the strategy to work (e.g., route handlers, init scripts).
   *
   * @param context - The browser context to initialize on
   * @param options - Configuration options
   */
  initialize(context: BrowserContext, options: InjectionStrategyOptions): Promise<void>;

  /**
   * Inject a script into a page.
   *
   * For context-level init-script registration, this confirms page readiness.
   *
   * @param page - The page to inject into
   * @param script - The JavaScript to inject
   * @returns Result of the injection attempt
   */
  injectScript(page: Page, script: string): Promise<InjectionResult>;

  /**
   * Verify that injection was successful on a page.
   *
   * This checks for verification markers set by the recording script
   * to confirm it's running in the correct context.
   *
   * @param page - The page to verify
   * @returns True if injection was verified successful
   */
  verify(page: Page): Promise<boolean>;

  /**
   * Get current statistics for this strategy.
   *
   * @returns Copy of current statistics
   */
  getStats(): InjectionStrategyStats;

  /**
   * Reset statistics to initial values.
   * Useful for clearing state between test runs.
   */
  resetStats(): void;

  /**
   * Clean up resources used by this strategy.
   *
   * Called when the strategy is no longer needed. Should release
   * any held resources (CDP sessions, route handlers, etc.).
   */
  cleanup(): Promise<void>;


}

// =============================================================================
// Helper Functions
// =============================================================================

/**
 * Create initial stats object for a strategy.
 */
export function createInitialStats(): InjectionStrategyStats {
  return {
    attempted: 0,
    successful: 0,
    failed: 0,
    avgInjectionTimeMs: 0,
    lastInjectionAt: null,
  };
}

/**
 * Clone stats to prevent external mutation.
 */
export function cloneStats(stats: InjectionStrategyStats): InjectionStrategyStats {
  return { ...stats };
}

/**
 * Update stats after an injection attempt.
 *
 * @param stats - Stats object to update (mutated in place)
 * @param success - Whether the injection succeeded
 * @param durationMs - How long the injection took
 */
export function updateStats(stats: InjectionStrategyStats, success: boolean, durationMs: number): void {
  stats.attempted++;
  if (success) {
    stats.successful++;
  } else {
    stats.failed++;
  }

  // Update rolling average
  const totalAttempts = stats.successful + stats.failed;
  stats.avgInjectionTimeMs =
    (stats.avgInjectionTimeMs * (totalAttempts - 1) + durationMs) / totalAttempts;

  stats.lastInjectionAt = new Date().toISOString();
}

/**
 * Reset stats to initial values.
 */
export function resetStats(stats: InjectionStrategyStats): void {
  stats.attempted = 0;
  stats.successful = 0;
  stats.failed = 0;
  stats.avgInjectionTimeMs = 0;
  stats.lastInjectionAt = null;
}
