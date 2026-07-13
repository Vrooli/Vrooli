/**
 * Browser Manager
 *
 * Manages shared browser instance lifecycles including:
 * - Launch with configuration
 * - Verification (smoke test on startup)
 * - Connection monitoring
 * - Graceful shutdown
 *
 * BROWSER POOLING:
 * - One default browser serves ordinary sessions.
 * - Chromium fake media capture (--use-file-for-fake-audio-capture) is a
 *   process-wide launch flag, so sessions requesting a fake microphone WAV
 *   get a dedicated browser instance pooled per distinct WAV path.
 *
 * CONCURRENCY SAFETY:
 * - Uses per-key launch-promise locks to prevent concurrent browser launches
 * - Multiple concurrent calls to getBrowser() await the same launch promise
 * - Safe to call from multiple sessions simultaneously
 *
 * PLAYWRIGHT PROVIDER:
 * - Uses playwrightProvider for browser launch (see src/playwright/)
 * - Allows future switching between rebrowser-playwright and playwright
 * - Provider capabilities inform recording architecture decisions
 *
 * @module session/browser-manager
 */

import { existsSync, statSync } from 'fs';
import { isAbsolute } from 'path';
import type { Browser } from 'rebrowser-playwright';
import { playwrightProvider } from '../playwright';
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

/** Pool key for the default (no fake media) browser. */
const DEFAULT_BROWSER_KEY = '';

// =============================================================================
// BrowserManager
// =============================================================================

/**
 * Manages the shared browser instances used by all sessions.
 *
 * Browsers are lazily launched on first matching session creation and kept
 * alive until shutdown. This avoids the startup cost of launching
 * Chromium for each session.
 */
export class BrowserManager {
  /** Browsers keyed by fake microphone WAV path ('' = default browser). */
  private browsers = new Map<string, Browser>();
  private config: Config;

  private browserVerified = false;
  private browserError: string | null = null;

  /**
   * Per-key locks to prevent concurrent browser launches.
   * Hold promises that resolve when browser launch completes.
   * This prevents the race condition where multiple startSession() calls
   * could each launch their own browser instance for the same key.
   */
  private browserLaunchPromises = new Map<string, Promise<Browser>>();

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

    const defaultBrowser = this.browsers.get(DEFAULT_BROWSER_KEY);
    if (defaultBrowser && defaultBrowser.isConnected()) {
      return { healthy: true, version: defaultBrowser.version() };
    }

    return { healthy: true };
  }

  /**
   * Get or create a shared browser instance.
   *
   * @param fakeMicrophoneWav - Optional absolute WAV path to serve as a
   *   deterministic fake microphone. Sessions requesting fake media get a
   *   dedicated browser instance pooled per distinct WAV path, because the
   *   Chromium fake-capture flags are process-wide.
   *
   * Temporal hardening:
   * - Uses per-key locks (browserLaunchPromises) to prevent concurrent launches
   * - Multiple concurrent calls will all await the same launch promise
   * - If browser disconnects mid-launch, subsequent calls will retry
   */
  async getBrowser(fakeMicrophoneWav?: string): Promise<Browser> {
    const key = fakeMicrophoneWav?.trim() ?? DEFAULT_BROWSER_KEY;
    if (key !== DEFAULT_BROWSER_KEY) {
      this.validateFakeMicrophoneWav(key);
    }

    // Fast path: browser already exists and is connected
    const existing = this.browsers.get(key);
    if (existing && existing.isConnected()) {
      return existing;
    }

    // If another call is already launching this browser, wait for it
    const inFlight = this.browserLaunchPromises.get(key);
    if (inFlight) {
      logger.debug('browser: waiting for concurrent launch to complete');
      try {
        const browser = await inFlight;
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
    const launchPromise = this.launchBrowserInternal(key);
    this.browserLaunchPromises.set(key, launchPromise);

    try {
      const browser = await launchPromise;
      this.browsers.set(key, browser);
      return browser;
    } finally {
      // Clear the promise after launch completes (success or failure)
      // This allows retry on next call if launch failed
      this.browserLaunchPromises.delete(key);
    }
  }

  /**
   * Validate a per-session fake microphone WAV request.
   * The path must be absolute (the API layer resolves workflow-relative
   * fixture paths against the execution's project root) and readable.
   */
  private validateFakeMicrophoneWav(wavPath: string): void {
    if (!isAbsolute(wavPath)) {
      throw new Error(
        `fake_media.microphone_wav must be an absolute path, got "${wavPath}"`
      );
    }
    if (!existsSync(wavPath) || !statSync(wavPath).isFile()) {
      throw new Error(`fake_media.microphone_wav "${wavPath}" does not exist or is not a file`);
    }
  }

  /**
   * Internal browser launch implementation.
   * Separated from getBrowser() to make the locking logic clearer.
   *
   * Uses playwrightProvider.chromium for browser launch.
   * This allows switching between rebrowser-playwright (anti-detection)
   * and standard playwright in the future. See src/playwright/ for details.
   */
  private async launchBrowserInternal(fakeMicrophoneKey: string): Promise<Browser> {
    // Per-session fake media wins; the process-level env knob
    // (BAS_FAKE_MICROPHONE_FILE) remains as the default-browser fallback for
    // dedicated qualification drivers.
    const fakeMicrophoneFile = fakeMicrophoneKey || this.config.browser.fakeMicrophoneFile;
    const args = [...this.config.browser.args];
    if (fakeMicrophoneFile) {
      // Chromium still performs its normal getUserMedia capture path. These
      // flags only replace the physical device and permission prompt with a
      // deterministic WAV-backed device for an explicitly configured test run.
      args.push(
        '--use-fake-device-for-media-stream',
        '--use-fake-ui-for-media-stream',
        `--use-file-for-fake-audio-capture=${fakeMicrophoneFile}`,
      );
    }
    logger.info('browser: launching', {
      headless: this.config.browser.headless,
      executablePath: this.config.browser.executablePath || 'auto',
      provider: playwrightProvider.name,
      capabilities: playwrightProvider.capabilities,
      fakeMicrophone: Boolean(fakeMicrophoneFile),
      dedicatedFakeMediaBrowser: fakeMicrophoneKey !== DEFAULT_BROWSER_KEY,
    });

    const browser = await playwrightProvider.chromium.launch({
      headless: this.config.browser.headless,
      executablePath: this.config.browser.executablePath || undefined,
      args,
    });

    logger.info('browser: launched', {
      version: browser.version(),
      provider: playwrightProvider.name,
    });
    return browser;
  }

  /**
   * Shutdown all browsers and cleanup resources.
   */
  async shutdown(): Promise<void> {
    for (const [key, browser] of this.browsers) {
      await browser.close().catch((error: unknown) => {
        const message = error instanceof Error ? error.message : String(error);
        logger.warn('Failed to close browser', { error: message, browserKey: key || 'default' });
        metrics.cleanupFailures.inc({ operation: 'browser_close' });
      });
    }
    this.browsers.clear();
  }

  /**
   * Check if the default browser is currently connected.
   */
  isConnected(): boolean {
    return this.browsers.get(DEFAULT_BROWSER_KEY)?.isConnected() ?? false;
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
