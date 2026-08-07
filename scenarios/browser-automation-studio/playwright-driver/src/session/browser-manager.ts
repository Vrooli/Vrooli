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
import {
  detectHostAudioCapability,
  getAudioLaunchArgs,
  selectAudioStrategy,
  type AudioStrategy,
  type HostAudioCapability,
} from './audio';
import { BrowserPool } from './browser-pool';

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
  audioCapability?: HostAudioCapability;
}

/** Pool key for the ordinary host-device browser with no fake media. */
const DEFAULT_BROWSER_KEY = 'host_device\u0000';

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
  private readonly pool = new BrowserPool();
  private config: Config;

  private browserVerified = false;
  private browserError: string | null = null;

  private audioCapabilityPromise?: Promise<HostAudioCapability>;
  private audioCapability?: HostAudioCapability;

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

    const defaultBrowser = this.pool.get(DEFAULT_BROWSER_KEY);
    if (defaultBrowser) {
      return {
        healthy: true,
        version: defaultBrowser.version(),
        audioCapability: this.audioCapability,
      };
    }

    return { healthy: true, audioCapability: this.audioCapability };
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
  async getBrowser(
    fakeMicrophoneWav?: string,
    audioStrategy: AudioStrategy = 'host_device'
  ): Promise<Browser> {
    const wavPath = fakeMicrophoneWav?.trim() ?? '';
    const key = getBrowserPoolKey(wavPath, audioStrategy);
    if (wavPath) {
      this.validateFakeMicrophoneWav(wavPath);
    }

    return this.pool.getOrLaunch(key, () => this.launchBrowserInternal(key));
  }

  /** Connect to a browser owned by a controlled desktop target. */
  async connectOverCDP(endpoint: string): Promise<Browser> {
    return playwrightProvider.chromium.connectOverCDP(endpoint);
  }

  /**
   * Validate a per-session fake microphone WAV request.
   * The path must be absolute (the API layer resolves workflow-relative
   * fixture paths against the execution's project root) and readable.
   */
  private validateFakeMicrophoneWav(wavPath: string): void {
    if (!isAbsolute(wavPath)) {
      throw new Error(`fake_media.microphone_wav must be an absolute path, got "${wavPath}"`);
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
  private async launchBrowserInternal(poolKey: string): Promise<Browser> {
    // Per-session fake media wins; the process-level env knob
    // (BAS_FAKE_MICROPHONE_FILE) remains as the default-browser fallback for
    // dedicated qualification drivers.
    const [, fakeMicrophoneKey = ''] = poolKey.split('\u0000', 2);
    const fakeMicrophoneFile = fakeMicrophoneKey || this.config.browser.fakeMicrophoneFile;
    const [strategy] = poolKey.split('\u0000', 1) as [AudioStrategy];
    const args = [...this.config.browser.args, ...getAudioLaunchArgs(strategy)];
    if (fakeMicrophoneFile) {
      // Chromium still performs its normal getUserMedia capture path. These
      // flags only replace the physical device and permission prompt with a
      // deterministic WAV-backed device for an explicitly configured test run.
      args.push(
        '--use-fake-device-for-media-stream',
        '--use-fake-ui-for-media-stream',
        `--use-file-for-fake-audio-capture=${fakeMicrophoneFile}`
      );
    }
    logger.info('browser: launching', {
      headless: this.config.browser.headless,
      executablePath: this.config.browser.executablePath || 'auto',
      provider: playwrightProvider.name,
      capabilities: playwrightProvider.capabilities,
      fakeMicrophone: Boolean(fakeMicrophoneFile),
      dedicatedFakeMediaBrowser: Boolean(fakeMicrophoneKey),
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

  /** Measure host output once per driver process and reuse its durable verdict. */
  async getHostAudioCapability(): Promise<HostAudioCapability> {
    if (!this.audioCapabilityPromise) {
      this.audioCapabilityPromise = this.getBrowser()
        .then(detectHostAudioCapability)
        .then((capability) => {
          this.audioCapability = capability;
          logger.info('browser: host audio capability detected', capability);
          return capability;
        });
    }
    return this.audioCapabilityPromise;
  }

  async getAudioStrategy(): Promise<AudioStrategy> {
    return selectAudioStrategy(await this.getHostAudioCapability());
  }

  /**
   * Shutdown all browsers and cleanup resources.
   */
  async shutdown(): Promise<void> {
    await this.pool.closeAll(async (key, browser) => {
      await browser.close().catch((error: unknown) => {
        const message = error instanceof Error ? error.message : String(error);
        logger.warn('Failed to close browser', {
          error: message,
          browserKey: key === DEFAULT_BROWSER_KEY ? 'default' : key,
        });
        metrics.cleanupFailures.inc({ operation: 'browser_close' });
      });
    });
  }

  /**
   * Check if the default browser is currently connected.
   */
  isConnected(): boolean {
    return this.pool.get(DEFAULT_BROWSER_KEY) !== undefined;
  }
}

/** Chromium launch configuration is process-wide, so every launch axis is keyed. */
export function getBrowserPoolKey(
  fakeMicrophoneWav = '',
  strategy: AudioStrategy = 'host_device'
): string {
  return `${strategy}\u0000${fakeMicrophoneWav}`;
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
