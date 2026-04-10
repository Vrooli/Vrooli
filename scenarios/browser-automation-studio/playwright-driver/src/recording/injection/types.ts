/**
 * Injection Strategy Types
 *
 * This module defines the interface contract for injection strategies.
 * Each strategy implements a different mechanism for injecting the recording
 * script into browser pages.
 *
 * ## Strategy Overview
 *
 * | Strategy | How It Works | When to Use |
 * |----------|--------------|-------------|
 * | `init-script` | `context.addInitScript()` | RECOMMENDED for rebrowser-playwright |
 * | `cdp-injection` | `Page.addScriptToEvaluateOnNewDocument` | Fallback with full CDP control |
 * | `route-injection` | `context.route()` HTML modification | Standard playwright only |
 *
 * ## Why Multiple Strategies?
 *
 * - rebrowser-playwright breaks `context.route()` to evade bot detection
 * - Different providers/environments may have different capabilities
 * - Fallback strategies ensure recording works across configurations
 *
 * @module recording/injection/types
 */

import type { BrowserContext, Page } from 'rebrowser-playwright';
import type winston from 'winston';

// =============================================================================
// Core Types
// =============================================================================

/**
 * Names of available injection strategies.
 *
 * - `init-script`: Uses `context.addInitScript()` - RECOMMENDED for rebrowser-playwright
 * - `cdp-injection`: Uses CDP `Page.addScriptToEvaluateOnNewDocument` - Chromium fallback
 * - `route-injection`: Uses `context.route()` HTML modification - Standard playwright only
 */
export type InjectionStrategyName = 'init-script' | 'cdp-injection' | 'route-injection';

/**
 * Result of a script injection attempt.
 */
export interface InjectionResult {
  /** Whether injection succeeded */
  success: boolean;
  /** Which strategy performed the injection */
  strategy: InjectionStrategyName;
  /** Error message if injection failed */
  error?: string;
  /** When the injection occurred */
  timestamp: string;
  /** Additional strategy-specific metadata */
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
 * Interface for injection strategies.
 *
 * Each strategy implements a different mechanism for injecting the recording
 * script into browser pages. The interface provides a consistent API for:
 * - Initialization (context-level setup)
 * - Injection (page-level script injection)
 * - Verification (checking if injection worked)
 * - Statistics tracking (diagnostics)
 * - Cleanup (resource release)
 *
 * ## Implementing a New Strategy
 *
 * ```typescript
 * class MyInjectionStrategy implements InjectionStrategy {
 *   readonly name = 'my-strategy' as InjectionStrategyName;
 *
 *   async initialize(context, options) {
 *     // Set up context-level hooks
 *   }
 *
 *   async injectScript(page, script) {
 *     // Inject script into page
 *     return { success: true, strategy: this.name, timestamp: new Date().toISOString() };
 *   }
 *
 *   async verify(page) {
 *     // Check if injection worked
 *     return true;
 *   }
 *
 *   // ... other methods
 * }
 * ```
 *
 * ## Strategy Selection
 *
 * The factory selects strategies based on:
 * 1. `INJECTION_STRATEGY` environment variable
 * 2. Explicit `injectionStrategy` option
 * 3. Provider capabilities (auto-select for rebrowser-playwright)
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
   * Depending on the strategy, this may:
   * - Be a no-op if injection happens at context level (init-script)
   * - Inject into specific pages (cdp-injection)
   * - Modify HTML responses (route-injection)
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

  /**
   * Check if this strategy supports a given provider.
   *
   * @param providerName - Name of the playwright provider (e.g., 'rebrowser-playwright')
   * @returns True if this strategy works with the provider
   */
  supportsProvider(providerName: string): boolean;
}

// =============================================================================
// Factory Types
// =============================================================================

/**
 * Options for creating an injection strategy.
 */
export interface InjectionStrategyFactoryOptions {
  /** Explicitly select a strategy by name */
  strategyName?: InjectionStrategyName | 'auto';
  /** Provider name for capability checking */
  providerName?: string;
  /** Logger for diagnostics */
  logger?: winston.Logger;
}

/**
 * Options for auto-detecting a working strategy.
 */
export interface AutoDetectorOptions {
  /** Logger for diagnostics */
  logger?: winston.Logger;
  /** Order of strategies to try */
  strategyOrder?: InjectionStrategyName[];
  /** Timeout for each strategy verification (ms) */
  verificationTimeoutMs?: number;
}

/**
 * Result of auto-detection.
 */
export interface AutoDetectionResult {
  /** Strategy that was selected */
  strategy: InjectionStrategy | null;
  /** Name of the selected strategy, or null if all failed */
  strategyName: InjectionStrategyName | null;
  /** Strategies that were tried and their results */
  attempts: Array<{
    strategy: InjectionStrategyName;
    success: boolean;
    error?: string;
    durationMs: number;
  }>;
  /** Total time spent detecting */
  totalDurationMs: number;
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
