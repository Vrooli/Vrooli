/**
 * Route Injection Strategy
 *
 * Legacy strategy using Playwright's `context.route()` for HTML modification.
 *
 * ## How It Works
 *
 * Uses route interception to modify HTML responses:
 * 1. Registers `context.route('**\/*')` handler
 * 2. Intercepts document requests
 * 3. Fetches the original HTML
 * 4. Injects recording script into `<head>` or `<!DOCTYPE>`
 * 5. Returns modified HTML to browser
 *
 * ## When to Use
 *
 * - Standard Playwright (not rebrowser-playwright)
 * - Environments where route interception is proven to work
 * - When you need to modify HTML content (not just inject scripts)
 *
 * ## When to Avoid
 *
 * - rebrowser-playwright (route interception is BROKEN)
 * - Anti-detection environments (route interception may be detectable)
 * - High-traffic scenarios (adds latency to every document request)
 *
 * ## Known Issues with rebrowser-playwright
 *
 * rebrowser-playwright patches route interception for anti-detection:
 * - `context.route()` handlers may not be called consistently
 * - Redirects are not followed through the handler
 * - Some document requests bypass interception entirely
 *
 * This strategy is marked as DEPRECATED for rebrowser-playwright use cases.
 * Use `init-script` or `cdp-injection` strategies instead.
 *
 * @module recording/injection/strategies/route-injection
 * @deprecated For rebrowser-playwright environments. Use init-script strategy.
 */

import type { BrowserContext, Page } from 'rebrowser-playwright';
import {
  type InjectionStrategy,
  type InjectionStrategyName,
  type InjectionStrategyOptions,
  type InjectionResult,
  type InjectionStrategyStats,
  createInitialStats,
  cloneStats,
  resetStats as resetStatsHelper,
} from '../types';
import {
  setupHtmlInjectionRoute,
  type InjectionStats as HtmlInjectorStats,
} from '../../io/html-injector';
import { verifyScriptInjection } from '../../validation/verification';
import { LogContext, scopedLog } from '../../../utils';

/**
 * Route Injection Strategy
 *
 * Wraps the existing HTML injector for interface compliance.
 * This is a LEGACY strategy - prefer init-script for rebrowser-playwright.
 *
 * @deprecated For rebrowser-playwright environments. Use init-script strategy.
 *
 * @example
 * ```typescript
 * // Only use with standard Playwright
 * const strategy = new RouteInjectionStrategy();
 * await strategy.initialize(context, {
 *   bindingName: '__vrooli_recordAction',
 *   logger: createLogger(),
 * });
 *
 * // HTML injection happens automatically via route handler
 * const page = await context.newPage();
 * await page.goto('https://example.com');
 *
 * // Verify injection worked
 * const verified = await strategy.verify(page);
 * console.log('Injection verified:', verified);
 * ```
 */
export class RouteInjectionStrategy implements InjectionStrategy {
  readonly name: InjectionStrategyName = 'route-injection';

  private initialized = false;
  private options: InjectionStrategyOptions | null = null;
  private stats: InjectionStrategyStats = createInitialStats();
  private getHtmlStats: (() => HtmlInjectorStats) | null = null;
  private resetHtmlStats: (() => void) | null = null;

  /**
   * Initialize the strategy on a browser context.
   *
   * Sets up the HTML injection route handler using the existing
   * `setupHtmlInjectionRoute()` function.
   *
   * @param context - Browser context to initialize on
   * @param options - Configuration options
   */
  async initialize(context: BrowserContext, options: InjectionStrategyOptions): Promise<void> {
    if (this.initialized) {
      options.logger.debug(
        scopedLog(LogContext.INJECTION, 'route-injection strategy already initialized, skipping')
      );
      return;
    }

    this.options = options;

    const { bindingName, logger, diagnosticsEnabled, onFirstInjection } = options;

    // Log deprecation warning for rebrowser-playwright
    logger.warn(
      scopedLog(
        LogContext.INJECTION,
        'route-injection strategy is DEPRECATED for rebrowser-playwright. ' +
          'Route interception is broken with anti-detection patches. ' +
          'Use init-script strategy instead.'
      )
    );

    if (diagnosticsEnabled) {
      logger.debug(scopedLog(LogContext.INJECTION, 'route-injection strategy: setting up'), {
        bindingName,
      });
    }

    // Use the existing HTML injection setup
    const result = await setupHtmlInjectionRoute(context, {
      bindingName,
      logger,
      diagnosticsEnabled,
      onFirstInjection: () => {
        // Update our stats when first injection occurs
        this.syncStatsFromHtmlInjector();

        // Fire the callback
        if (onFirstInjection) {
          onFirstInjection();
        }
      },
    });

    // Store stat accessors
    this.getHtmlStats = result.getStats;
    this.resetHtmlStats = result.resetStats;

    this.initialized = true;

    logger.info(scopedLog(LogContext.INJECTION, 'route-injection strategy initialized'), {
      bindingName,
    });
  }

  /**
   * Sync stats from the HTML injector to our stats format.
   */
  private syncStatsFromHtmlInjector(): void {
    if (!this.getHtmlStats) return;

    const htmlStats = this.getHtmlStats();
    this.stats.attempted = htmlStats.attempted;
    this.stats.successful = htmlStats.successful;
    this.stats.failed = htmlStats.failed;
    this.stats.lastInjectionAt = new Date().toISOString();

    // Calculate average (rough approximation since HTML injector doesn't track timing)
    if (htmlStats.successful > 0) {
      this.stats.avgInjectionTimeMs = 50; // Estimate based on typical injection time
    }
  }

  /**
   * Inject script into a page.
   *
   * For route-injection, this is a no-op since injection happens automatically
   * via the route handler when pages navigate. This method verifies the injection.
   *
   * @param page - The page to inject into
   * @param _script - The script (unused - handled by route handler)
   * @returns Injection result
   */
  async injectScript(page: Page, _script: string): Promise<InjectionResult> {
    if (!this.initialized || !this.options) {
      return {
        success: false,
        strategy: this.name,
        error: 'Strategy not initialized',
        timestamp: new Date().toISOString(),
      };
    }

    const startTime = Date.now();

    try {
      // Sync stats from HTML injector
      this.syncStatsFromHtmlInjector();

      // Verify the script was injected via route handler
      const verification = await verifyScriptInjection(page);
      const durationMs = Date.now() - startTime;
      const success = verification.loaded && verification.ready;

      return {
        success,
        strategy: this.name,
        timestamp: new Date().toISOString(),
        error: success ? undefined : `Script not ready: loaded=${verification.loaded}, ready=${verification.ready}`,
        metadata: {
          verification: {
            loaded: verification.loaded,
            ready: verification.ready,
            inMainContext: verification.inMainContext,
            handlersCount: verification.handlersCount,
          },
          htmlInjectorStats: this.getHtmlStats?.() ?? null,
          durationMs,
        },
      };
    } catch (error) {
      return {
        success: false,
        strategy: this.name,
        error: error instanceof Error ? error.message : String(error),
        timestamp: new Date().toISOString(),
        metadata: { durationMs: Date.now() - startTime },
      };
    }
  }

  /**
   * Verify that injection was successful on a page.
   *
   * @param page - The page to verify
   * @returns True if script is loaded, ready, and in MAIN context
   */
  async verify(page: Page): Promise<boolean> {
    try {
      const verification = await verifyScriptInjection(page);
      return verification.loaded && verification.ready && verification.inMainContext;
    } catch {
      return false;
    }
  }

  /**
   * Get current statistics for this strategy.
   */
  getStats(): InjectionStrategyStats {
    // Sync before returning
    this.syncStatsFromHtmlInjector();
    return cloneStats(this.stats);
  }

  /**
   * Reset statistics to initial values.
   */
  resetStats(): void {
    resetStatsHelper(this.stats);
    if (this.resetHtmlStats) {
      this.resetHtmlStats();
    }
  }

  /**
   * Clean up resources.
   *
   * Note: Route handlers registered with context.route() cannot be removed
   * in Playwright. The handler will remain active for the context lifetime.
   */
  cleanup(): Promise<void> {
    // Route handlers cannot be removed in Playwright
    // Just clear our references
    this.options = null;
    this.initialized = false;
    this.getHtmlStats = null;
    this.resetHtmlStats = null;
    return Promise.resolve();
  }

  /**
   * Check if this strategy supports a given provider.
   *
   * Route injection only works reliably with standard Playwright.
   * It is BROKEN with rebrowser-playwright.
   *
   * @param providerName - Name of the provider
   * @returns True only for standard Playwright (not rebrowser)
   */
  supportsProvider(providerName: string): boolean {
    const lowerName = providerName.toLowerCase();

    // Explicitly NOT supported for rebrowser-playwright
    if (lowerName.includes('rebrowser')) {
      return false;
    }

    // Only support standard playwright
    return lowerName.includes('playwright');
  }
}

/**
 * Create a new RouteInjectionStrategy instance.
 *
 * @deprecated For rebrowser-playwright environments. Use createInitScriptInjectionStrategy().
 */
export function createRouteInjectionStrategy(): RouteInjectionStrategy {
  return new RouteInjectionStrategy();
}
