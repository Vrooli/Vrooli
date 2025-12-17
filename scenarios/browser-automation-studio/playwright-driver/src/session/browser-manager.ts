/**
 * Browser Manager
 *
 * Manages the shared browser instance lifecycle including:
 * - Launch with configuration
 * - Verification (smoke test on startup)
 * - Connection monitoring
 * - Graceful shutdown
 *
 * CONCURRENCY SAFETY:
 * - Uses browserLaunchPromise lock to prevent concurrent browser launches
 * - Multiple concurrent calls to getBrowser() await the same launch promise
 * - Safe to call from multiple sessions simultaneously
 *
 * @module session/browser-manager
 */

import { chromium, Browser } from 'playwright';
import type { Config } from '../config';
import { logger, metrics } from '../utils';

// =============================================================================
// Types
// =============================================================================

/**
 * Browser health status for health endpoint.
 */
export interface BrowserStatus {
  healthy: boolean;
  error?: string;
  version?: string;
}

// =============================================================================
// BrowserManager
// =============================================================================

/**
 * Manages the shared browser instance used by all sessions.
 *
 * The browser is lazily launched on first session creation and kept
 * alive until shutdown. This avoids the startup cost of launching
 * Chromium for each session.
 */
export class BrowserManager {
  private browser: Browser | null = null;
  private config: Config;

  private browserVerified = false;
  private browserError: string | null = null;

  /**
   * Lock to prevent concurrent browser launches.
   * Holds a promise that resolves when browser launch completes.
   * This prevents the race condition where multiple startSession() calls
   * could each launch their own browser instance.
   */
  private browserLaunchPromise: Promise<Browser> | null = null;

  constructor(config: Config) {
    this.config = config;
  }

  /**
   * Verify that the browser can be launched.
   * Called during startup to catch Chromium issues early.
   * Returns null on success, error message on failure.
   */
  async verifyBrowserLaunch(): Promise<string | null> {
    if (this.browserVerified) {
      return this.browserError;
    }

    try {
      logger.info('browser: verifying launch capability');
      const browser = await this.getBrowser();

      // Verify we can create a context and page
      const context = await browser.newContext();
      const page = await context.newPage();

      // Verify basic navigation works
      await page.goto('about:blank');

      // Cleanup verification resources
      await page.close();
      await context.close();

      this.browserVerified = true;
      this.browserError = null;

      logger.info('browser: verification successful', {
        version: browser.version(),
      });

      return null;
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      this.browserError = errorMessage;
      this.browserVerified = true; // Mark as verified (we checked, it failed)

      logger.error('browser: verification failed', {
        error: errorMessage,
        hint: 'Check that Chromium is installed and sandbox settings are correct',
      });

      return errorMessage;
    }
  }

  /**
   * Get browser health status for health endpoint.
   */
  getBrowserStatus(): BrowserStatus {
    if (!this.browserVerified) {
      return { healthy: false, error: 'Browser not yet verified' };
    }

    if (this.browserError) {
      return { healthy: false, error: this.browserError };
    }

    if (this.browser && this.browser.isConnected()) {
      return { healthy: true, version: this.browser.version() };
    }

    return { healthy: true };
  }

  /**
   * Get or create shared browser instance.
   *
   * Temporal hardening:
   * - Uses a lock (browserLaunchPromise) to prevent concurrent browser launches
   * - Multiple concurrent calls will all await the same launch promise
   * - If browser disconnects mid-launch, subsequent calls will retry
   */
  async getBrowser(): Promise<Browser> {
    // Fast path: browser already exists and is connected
    if (this.browser && this.browser.isConnected()) {
      return this.browser;
    }

    // If another call is already launching the browser, wait for it
    if (this.browserLaunchPromise) {
      logger.debug('browser: waiting for concurrent launch to complete');
      try {
        const browser = await this.browserLaunchPromise;
        // Double-check it's still connected after await
        if (browser.isConnected()) {
          return browser;
        }
        // Browser disconnected during wait, fall through to launch new one
      } catch (error) {
        // Launch failed, fall through to try again
        logger.debug('browser: concurrent launch failed, will retry', {
          error: error instanceof Error ? error.message : String(error),
        });
      }
    }

    // Create the launch promise BEFORE starting the launch
    // This ensures concurrent calls will await this promise
    this.browserLaunchPromise = this.launchBrowserInternal();

    try {
      this.browser = await this.browserLaunchPromise;
      return this.browser;
    } finally {
      // Clear the promise after launch completes (success or failure)
      // This allows retry on next call if launch failed
      this.browserLaunchPromise = null;
    }
  }

  /**
   * Internal browser launch implementation.
   * Separated from getBrowser() to make the locking logic clearer.
   */
  private async launchBrowserInternal(): Promise<Browser> {
    logger.info('browser: launching', {
      headless: this.config.browser.headless,
      executablePath: this.config.browser.executablePath || 'auto',
    });

    const browser = await chromium.launch({
      headless: this.config.browser.headless,
      executablePath: this.config.browser.executablePath || undefined,
      args: this.config.browser.args,
    });

    logger.info('browser: launched', { version: browser.version() });
    return browser;
  }

  /**
   * Shutdown the browser and cleanup resources.
   */
  async shutdown(): Promise<void> {
    if (this.browser) {
      await this.browser.close().catch((err) => {
        logger.warn('Failed to close browser', { error: err.message });
        metrics.cleanupFailures.inc({ operation: 'browser_close' });
      });
      this.browser = null;
    }
  }

  /**
   * Check if browser is currently connected.
   */
  isConnected(): boolean {
    return this.browser?.isConnected() ?? false;
  }
}

/**
 * Create a BrowserManager instance.
 *
 * @param config - Application config with browser settings
 * @returns BrowserManager instance
 */
export function createBrowserManager(config: Config): BrowserManager {
  return new BrowserManager(config);
}
