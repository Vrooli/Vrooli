/**
 * Init Script Injection Strategy
 *
 * RECOMMENDED strategy for rebrowser-playwright environments.
 *
 * ## How It Works
 *
 * Uses `context.addInitScript()` to register a script that runs on every new
 * document in the context. The script:
 * 1. Runs in MAIN execution context (not ISOLATED)
 * 2. Executes before any page JavaScript
 * 3. Can properly wrap History API (pushState, replaceState)
 * 4. Persists across navigations automatically
 *
 * ## When to Use
 *
 * - rebrowser-playwright environments (anti-detection mode)
 * - When route interception is broken or unreliable
 * - When you need consistent injection across all page loads
 * - Default recommendation for recording features
 *
 * ## When to Avoid
 *
 * - When you need per-page script customization
 * - When debugging injection (CDP offers more visibility)
 * - Standard Playwright where route-injection is proven to work
 *
 * ## Technical Details
 *
 * The `addInitScript()` API:
 * - Registers script at context creation time
 * - Script runs in MAIN context (critical for History API)
 * - Survives page reloads and navigations
 * - Cannot be removed once added (context-level)
 *
 * @module recording/injection/strategies/init-script-injection
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
  updateStats,
  resetStats as resetStatsHelper,
} from '../types';
import { generateRecordingInitScript } from '../../capture/init-script-generator';
import { verifyScriptInjection } from '../../validation/verification';
import { LogContext, scopedLog } from '../../../utils';

/**
 * Init Script Injection Strategy
 *
 * Injects the recording script using `context.addInitScript()`.
 * This is the RECOMMENDED strategy for rebrowser-playwright.
 *
 * @example
 * ```typescript
 * const strategy = new InitScriptInjectionStrategy();
 * await strategy.initialize(context, {
 *   bindingName: '__vrooli_recordAction',
 *   logger: createLogger(),
 * });
 *
 * // Script is automatically injected into all pages
 * const page = await context.newPage();
 * await page.goto('https://example.com');
 *
 * // Verify injection worked
 * const verified = await strategy.verify(page);
 * console.log('Injection verified:', verified);
 * ```
 */
export class InitScriptInjectionStrategy implements InjectionStrategy {
  readonly name: InjectionStrategyName = 'init-script';

  private initialized = false;
  private context: BrowserContext | null = null;
  private options: InjectionStrategyOptions | null = null;
  private stats: InjectionStrategyStats = createInitialStats();
  private firstInjectionFired = false;

  /**
   * Initialize the strategy on a browser context.
   *
   * Registers the recording init script using `context.addInitScript()`.
   * The script will run on every new document created in this context.
   *
   * @param context - Browser context to initialize on
   * @param options - Configuration options
   */
  async initialize(context: BrowserContext, options: InjectionStrategyOptions): Promise<void> {
    if (this.initialized) {
      options.logger.debug(
        scopedLog(LogContext.INJECTION, 'init-script strategy already initialized, skipping')
      );
      return;
    }

    this.context = context;
    this.options = options;

    const { bindingName, logger, diagnosticsEnabled } = options;

    // Generate the recording script
    const initScript = generateRecordingInitScript(bindingName);

    if (diagnosticsEnabled) {
      logger.debug(scopedLog(LogContext.INJECTION, 'init-script strategy: registering init script'), {
        bindingName,
        scriptLength: initScript.length,
      });
    }

    // Register the script with the context
    // This runs BEFORE any page JavaScript on every new document
    await context.addInitScript(initScript);

    // Track when pages are created to fire first injection callback
    context.on('page', (page: Page) => {
      this.handlePageCreated(page).catch((err) => {
        logger.warn(scopedLog(LogContext.INJECTION, 'error handling page creation'), {
          error: err instanceof Error ? err.message : String(err),
        });
      });
    });

    this.initialized = true;

    logger.info(scopedLog(LogContext.INJECTION, 'init-script strategy initialized'), {
      bindingName,
    });
  }

  /**
   * Handle page creation for tracking first injection.
   */
  private async handlePageCreated(page: Page): Promise<void> {
    if (!this.options) return;

    const { logger, diagnosticsEnabled, onFirstInjection } = this.options;

    // Wait a moment for the page to initialize
    // The init script should run immediately on document creation
    await page.waitForLoadState('domcontentloaded').catch(() => {});

    const startTime = Date.now();

    try {
      // Verify the script was injected
      const verification = await verifyScriptInjection(page);

      const durationMs = Date.now() - startTime;
      const success = verification.loaded && verification.ready;

      // Update stats
      updateStats(this.stats, success, durationMs);

      if (diagnosticsEnabled) {
        logger.debug(scopedLog(LogContext.INJECTION, 'init-script injection result'), {
          url: page.url().slice(0, 80),
          success,
          verification: {
            loaded: verification.loaded,
            ready: verification.ready,
            inMainContext: verification.inMainContext,
            handlersCount: verification.handlersCount,
          },
          durationMs,
        });
      }

      // Fire first injection callback
      if (success && !this.firstInjectionFired && onFirstInjection) {
        this.firstInjectionFired = true;
        setImmediate(() => {
          try {
            onFirstInjection();
          } catch (err) {
            logger.error(scopedLog(LogContext.INJECTION, 'first injection callback error'), {
              error: err instanceof Error ? err.message : String(err),
            });
          }
        });
      }
    } catch (error) {
      const durationMs = Date.now() - startTime;
      updateStats(this.stats, false, durationMs);

      logger.warn(scopedLog(LogContext.INJECTION, 'init-script verification failed'), {
        url: page.url().slice(0, 80),
        error: error instanceof Error ? error.message : String(error),
        durationMs,
      });
    }
  }

  /**
   * Inject script into a page.
   *
   * For init-script strategy, this is largely a no-op since the script is
   * automatically injected at context level. This method exists for interface
   * compliance and may be used to verify injection after navigation.
   *
   * @param page - The page to inject into
   * @param _script - The script (unused - already registered at context level)
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
      // For init-script, the script is already injected via context.addInitScript()
      // We just verify it's working
      const verification = await verifyScriptInjection(page);
      const durationMs = Date.now() - startTime;
      const success = verification.loaded && verification.ready;

      updateStats(this.stats, success, durationMs);

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
          durationMs,
        },
      };
    } catch (error) {
      const durationMs = Date.now() - startTime;
      updateStats(this.stats, false, durationMs);

      return {
        success: false,
        strategy: this.name,
        error: error instanceof Error ? error.message : String(error),
        timestamp: new Date().toISOString(),
        metadata: { durationMs },
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
    return cloneStats(this.stats);
  }

  /**
   * Reset statistics to initial values.
   */
  resetStats(): void {
    resetStatsHelper(this.stats);
  }

  /**
   * Clean up resources.
   *
   * For init-script strategy, there's nothing to clean up since
   * `addInitScript()` cannot be removed. The script remains registered
   * for the lifetime of the context.
   */
  async cleanup(): Promise<void> {
    // addInitScript cannot be removed - script stays registered
    // Just clear our references
    this.context = null;
    this.options = null;
    this.initialized = false;
    this.firstInjectionFired = false;
  }

  /**
   * Check if this strategy supports a given provider.
   *
   * Init-script strategy works with all Playwright providers including
   * rebrowser-playwright. It's the RECOMMENDED strategy for rebrowser.
   *
   * @param _providerName - Name of the provider
   * @returns Always true - works with all providers
   */
  supportsProvider(_providerName: string): boolean {
    // init-script works with all providers
    // It's the RECOMMENDED strategy for rebrowser-playwright
    return true;
  }
}

/**
 * Create a new InitScriptInjectionStrategy instance.
 */
export function createInitScriptInjectionStrategy(): InitScriptInjectionStrategy {
  return new InitScriptInjectionStrategy();
}
