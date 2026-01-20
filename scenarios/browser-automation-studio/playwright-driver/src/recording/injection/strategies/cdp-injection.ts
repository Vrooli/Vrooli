/**
 * CDP Injection Strategy
 *
 * Fallback strategy using Chrome DevTools Protocol (CDP) for script injection.
 *
 * ## How It Works
 *
 * Uses the CDP `Page.addScriptToEvaluateOnNewDocument` command to inject scripts
 * into pages. This provides:
 * 1. Full control over script injection timing
 * 2. Works at the browser level (bypasses Playwright abstractions)
 * 3. Excellent debugging visibility through CDP
 * 4. Script runs in MAIN context
 *
 * ## When to Use
 *
 * - As a fallback when other strategies fail
 * - When you need fine-grained control over injection
 * - Debugging injection issues (CDP provides detailed diagnostics)
 * - Chromium-only environments where CDP is always available
 *
 * ## When to Avoid
 *
 * - Firefox or WebKit (CDP is Chromium-only)
 * - When bot detection is a concern (CDP usage may be detectable)
 * - Production environments where simpler strategies work
 *
 * ## CDP Usage Note
 *
 * This strategy uses these CDP commands:
 * - `Page.addScriptToEvaluateOnNewDocument`: Register script for injection
 * - `Runtime.evaluate`: Verify injection (also used by verification.ts)
 *
 * CDP is NOT used for:
 * - Route interception (that's Playwright's job)
 * - Event capture (that's the recording script's job)
 * - Frame streaming (that's handled by cdp-screencast.ts)
 *
 * @module recording/injection/strategies/cdp-injection
 */

import type { BrowserContext, Page, CDPSession } from 'rebrowser-playwright';
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
 * CDP session tracker for cleanup.
 */
interface SessionTracker {
  page: Page;
  session: CDPSession;
  scriptIdentifier?: string;
}

/**
 * CDP Injection Strategy
 *
 * Injects the recording script using Chrome DevTools Protocol.
 * This is a fallback strategy with full control and debugging visibility.
 *
 * @example
 * ```typescript
 * const strategy = new CDPInjectionStrategy();
 * await strategy.initialize(context, {
 *   bindingName: '__vrooli_recordAction',
 *   logger: createLogger(),
 * });
 *
 * // Inject into a specific page
 * const page = await context.newPage();
 * const result = await strategy.injectScript(page, script);
 * console.log('Injection result:', result);
 * ```
 */
export class CDPInjectionStrategy implements InjectionStrategy {
  readonly name: InjectionStrategyName = 'cdp-injection';

  private initialized = false;
  private context: BrowserContext | null = null;
  private options: InjectionStrategyOptions | null = null;
  private stats: InjectionStrategyStats = createInitialStats();
  private firstInjectionFired = false;
  private sessions: SessionTracker[] = [];
  private initScript: string = '';

  /**
   * Initialize the strategy on a browser context.
   *
   * For CDP strategy, we set up listeners for new pages and prepare
   * the injection script. Actual injection happens per-page.
   *
   * @param context - Browser context to initialize on
   * @param options - Configuration options
   */
  async initialize(context: BrowserContext, options: InjectionStrategyOptions): Promise<void> {
    if (this.initialized) {
      options.logger.debug(
        scopedLog(LogContext.INJECTION, 'cdp-injection strategy already initialized, skipping')
      );
      return;
    }

    this.context = context;
    this.options = options;

    const { bindingName, logger, diagnosticsEnabled } = options;

    // Generate the recording script
    this.initScript = generateRecordingInitScript(bindingName);

    if (diagnosticsEnabled) {
      logger.debug(scopedLog(LogContext.INJECTION, 'cdp-injection strategy: preparing'), {
        bindingName,
        scriptLength: this.initScript.length,
      });
    }

    // Set up injection for existing pages
    for (const page of context.pages()) {
      await this.setupPageInjection(page).catch((err) => {
        logger.warn(scopedLog(LogContext.INJECTION, 'failed to setup CDP injection for existing page'), {
          url: page.url().slice(0, 80),
          error: err instanceof Error ? err.message : String(err),
        });
      });
    }

    // Set up injection for new pages
    context.on('page', (page: Page) => {
      this.setupPageInjection(page).catch((err) => {
        logger.warn(scopedLog(LogContext.INJECTION, 'failed to setup CDP injection for new page'), {
          url: page.url().slice(0, 80),
          error: err instanceof Error ? err.message : String(err),
        });
      });
    });

    this.initialized = true;

    logger.info(scopedLog(LogContext.INJECTION, 'cdp-injection strategy initialized'), {
      bindingName,
    });
  }

  /**
   * Set up CDP injection for a specific page.
   */
  private async setupPageInjection(page: Page): Promise<void> {
    if (!this.options) return;

    const { logger, diagnosticsEnabled, onFirstInjection } = this.options;
    const startTime = Date.now();

    try {
      // Create a CDP session for this page
      const session = await page.context().newCDPSession(page);

      // Register the script to be evaluated on new documents
      const response = await session.send('Page.addScriptToEvaluateOnNewDocument', {
        source: this.initScript,
        worldName: undefined, // Use main world (default)
        runImmediately: true, // Run immediately, not just on navigation
      });

      // Track the session for cleanup
      this.sessions.push({
        page,
        session,
        scriptIdentifier: response.identifier,
      });

      if (diagnosticsEnabled) {
        logger.debug(scopedLog(LogContext.INJECTION, 'cdp-injection: script registered'), {
          url: page.url().slice(0, 80),
          identifier: response.identifier,
        });
      }

      // If page already has content, inject immediately
      const currentUrl = page.url();
      if (currentUrl && currentUrl !== 'about:blank') {
        await this.injectIntoExistingPage(page, session);
      }

      // Listen for page loads to track injection success
      page.on('domcontentloaded', async () => {
        await this.handlePageLoad(page, startTime, onFirstInjection);
      });
    } catch (error) {
      const durationMs = Date.now() - startTime;
      updateStats(this.stats, false, durationMs);

      logger.error(scopedLog(LogContext.INJECTION, 'cdp-injection setup failed'), {
        url: page.url().slice(0, 80),
        error: error instanceof Error ? error.message : String(error),
      });

      throw error;
    }
  }

  /**
   * Inject into a page that already has content loaded.
   */
  private async injectIntoExistingPage(page: Page, session: CDPSession): Promise<void> {
    if (!this.options) return;

    const { logger, diagnosticsEnabled } = this.options;

    try {
      // Execute the script in the current page
      await session.send('Runtime.evaluate', {
        expression: this.initScript,
        awaitPromise: false,
        returnByValue: false,
      });

      if (diagnosticsEnabled) {
        logger.debug(scopedLog(LogContext.INJECTION, 'cdp-injection: injected into existing page'), {
          url: page.url().slice(0, 80),
        });
      }
    } catch (error) {
      logger.warn(scopedLog(LogContext.INJECTION, 'cdp-injection: failed to inject into existing page'), {
        url: page.url().slice(0, 80),
        error: error instanceof Error ? error.message : String(error),
      });
    }
  }

  /**
   * Handle page load event to track injection success.
   */
  private async handlePageLoad(
    page: Page,
    startTime: number,
    onFirstInjection?: () => void
  ): Promise<void> {
    if (!this.options) return;

    const { logger, diagnosticsEnabled } = this.options;

    try {
      // Verify the script was injected
      const verification = await verifyScriptInjection(page);

      const durationMs = Date.now() - startTime;
      const success = verification.loaded && verification.ready;

      updateStats(this.stats, success, durationMs);

      if (diagnosticsEnabled) {
        logger.debug(scopedLog(LogContext.INJECTION, 'cdp-injection result'), {
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
      logger.warn(scopedLog(LogContext.INJECTION, 'cdp-injection verification failed'), {
        url: page.url().slice(0, 80),
        error: error instanceof Error ? error.message : String(error),
      });
    }
  }

  /**
   * Inject script into a page.
   *
   * For CDP strategy, this executes the script directly in the page
   * using Runtime.evaluate.
   *
   * @param page - The page to inject into
   * @param script - The script to inject (defaults to recorded init script)
   * @returns Injection result
   */
  async injectScript(page: Page, script?: string): Promise<InjectionResult> {
    if (!this.initialized || !this.options) {
      return {
        success: false,
        strategy: this.name,
        error: 'Strategy not initialized',
        timestamp: new Date().toISOString(),
      };
    }

    const scriptToInject = script || this.initScript;
    const startTime = Date.now();

    try {
      // Create a new CDP session for this injection
      const session = await page.context().newCDPSession(page);

      try {
        // Execute the script
        await session.send('Runtime.evaluate', {
          expression: scriptToInject,
          awaitPromise: false,
          returnByValue: false,
        });

        // Verify it worked
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
      } finally {
        await session.detach().catch(() => {});
      }
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
   * Detaches all CDP sessions and removes script registrations.
   */
  async cleanup(): Promise<void> {
    const { logger } = this.options || {};

    // Detach all CDP sessions
    for (const tracker of this.sessions) {
      try {
        // Remove the script registration
        if (tracker.scriptIdentifier) {
          await tracker.session.send('Page.removeScriptToEvaluateOnNewDocument', {
            identifier: tracker.scriptIdentifier,
          }).catch(() => {});
        }

        // Detach the session
        await tracker.session.detach().catch(() => {});
      } catch (error) {
        logger?.warn(scopedLog(LogContext.INJECTION, 'error cleaning up CDP session'), {
          error: error instanceof Error ? error.message : String(error),
        });
      }
    }

    this.sessions = [];
    this.context = null;
    this.options = null;
    this.initialized = false;
    this.firstInjectionFired = false;
    this.initScript = '';
  }

  /**
   * Check if this strategy supports a given provider.
   *
   * CDP strategy only works with Chromium-based browsers.
   *
   * @param providerName - Name of the provider
   * @returns True if provider is Chromium-based
   */
  supportsProvider(providerName: string): boolean {
    // CDP is Chromium-only
    // Works with rebrowser-playwright and standard playwright on Chromium
    const chromiumProviders = [
      'rebrowser-playwright',
      'playwright',
      'chromium',
    ];

    const lowerName = providerName.toLowerCase();

    // Reject Firefox and WebKit explicitly
    if (lowerName.includes('firefox') || lowerName.includes('webkit')) {
      return false;
    }

    return chromiumProviders.some((p) => lowerName.includes(p));
  }
}

/**
 * Create a new CDPInjectionStrategy instance.
 */
export function createCDPInjectionStrategy(): CDPInjectionStrategy {
  return new CDPInjectionStrategy();
}
