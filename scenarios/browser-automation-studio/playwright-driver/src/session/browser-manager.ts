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

import { accessSync, constants, existsSync, realpathSync, statSync } from 'fs';
import { isAbsolute } from 'path';
import type { Browser } from 'rebrowser-playwright';
import type { BrowserBackend } from '../types/session';
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
import { BrowserPool, NonRetryableBrowserLaunchError } from './browser-pool';
import { createAndroidCDPProxy } from './android-cdp-proxy';

// =============================================================================
// Types
// =============================================================================

/**
 * Browser health status for health endpoint.
 */
export interface BrowserStatus {
  healthy: boolean;
  /** Whether the version below came from a currently connected browser process. */
  runtime_identity_status?: 'reported' | 'unavailable';
  error?: string;
  family?: 'chromium';
  version?: string;
  audioCapability?: HostAudioCapability;
}

/** Pool key for the ordinary host-device browser with no fake media. */
const DEFAULT_BROWSER_KEY = getBrowserPoolKey();

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
      return { healthy: false, runtime_identity_status: 'unavailable', error: 'Browser not yet verified' };
    }

    if (this.browserError) {
      return { healthy: false, runtime_identity_status: 'unavailable', error: this.browserError };
    }

    const defaultBrowser = this.pool.get(DEFAULT_BROWSER_KEY);
    if (defaultBrowser) {
      return {
        healthy: true,
        runtime_identity_status: 'reported',
        family: 'chromium',
        version: defaultBrowser.version(),
        audioCapability: this.audioCapability,
      };
    }

    // A successful earlier launch is not evidence of a currently serving
    // binary. Idle eviction or a process exit can remove the default browser;
    // keep readiness separate from the unavailable live identity.
    return { healthy: true, runtime_identity_status: 'unavailable', audioCapability: this.audioCapability };
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
    audioStrategy: AudioStrategy = 'host_device',
    allowExtensionDebugging = false,
    extensionIsolationKey = '',
    backend: BrowserBackend = 'managed'
  ): Promise<Browser> {
    const wavPath = fakeMicrophoneWav?.trim() ?? '';
    const executablePath = backend === 'installed'
      ? resolveInstalledExecutablePath(this.config.browser.executablePath)
      : this.config.browser.executablePath || undefined;
    const key = getBrowserPoolKey(wavPath, audioStrategy, allowExtensionDebugging, extensionIsolationKey, backend, executablePath);
    if (allowExtensionDebugging && !extensionIsolationKey.trim()) {
      throw new Error('extension debugging requires a session-specific browser pool key');
    }
    if (wavPath) {
      this.validateFakeMicrophoneWav(wavPath);
    }

    const browser = await this.pool.getOrLaunch(key, () => this.launchBrowserInternal(key, executablePath, backend));
    this.updatePoolMetrics();
    return browser;
  }

  /** Connect to a browser owned by a controlled desktop target. */
  async connectOverCDP(endpoint: string, androidWebView = false): Promise<Browser> {
    if (!androidWebView) return playwrightProvider.chromium.connectOverCDP(endpoint);
    const proxy = await createAndroidCDPProxy(endpoint);
    try {
      const browser = await playwrightProvider.chromium.connectOverCDP(proxy.endpoint);
      browser.on('disconnected', () => {
        void proxy.close();
      });
      return browser;
    } catch (error) {
      await proxy.close();
      throw error;
    }
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
  private async launchBrowserInternal(poolKey: string, executablePath?: string, backend: BrowserBackend = 'managed'): Promise<Browser> {
    // Per-session fake media wins; the process-level env knob
    // (BAS_FAKE_MICROPHONE_FILE) remains as the default-browser fallback for
    // dedicated qualification drivers.
    const [, fakeMicrophoneKey = '', extensionDebuggingKey = ''] = poolKey.split('\u0000', 3);
    const [strategy] = poolKey.split('\u0000', 1) as [AudioStrategy];
    const extensionDebugging = extensionDebuggingKey.startsWith('enabled:');
    const extensionIsolationKey = extensionDebugging ? extensionDebuggingKey.slice('enabled:'.length) : '';
    const fakeMicrophoneFile = fakeMicrophoneKey || this.config.browser.fakeMicrophoneFile;
    const args = [...this.config.browser.args, ...getAudioLaunchArgs(strategy)];
    if (extensionDebugging && extensionIsolationKey) {
      // Chromium exposes Extensions.loadUnpacked only through the DevTools
      // pipe when this security-sensitive switch is enabled. Keep the switch
      // off normal browsers and isolate each extension-enabled session in its
      // own BAS-owned browser process so installs cannot bleed across leases.
      args.push('--enable-unsafe-extension-debugging');
    }
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
      executablePath: executablePath || 'auto',
      provider: playwrightProvider.name,
      capabilities: playwrightProvider.capabilities,
      fakeMicrophone: Boolean(fakeMicrophoneFile),
      dedicatedFakeMediaBrowser: Boolean(fakeMicrophoneKey),
    });

    const browser = await playwrightProvider.chromium.launch({
      headless: this.config.browser.headless,
      executablePath,
      args,
    });

    let version: string;
    if (backend === 'installed') {
      try {
        version = browser.version();
        validateInstalledBrowserVersion(version);
      } catch {
        await browser.close().catch(() => undefined);
        throw new NonRetryableBrowserLaunchError(
          'installed browser did not report a valid Chromium version'
        );
      }
    } else {
      version = browser.version();
    }
    logger.info('browser: launched', {
      version,
      provider: playwrightProvider.name,
      backend,
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

  /** Close only pooled Chromium processes whose contexts are all gone. */
  async recycleIdleBrowsers(idleForMs: number): Promise<number> {
    try {
      const recycled = await this.pool.closeIdle(idleForMs, async (key, browser) => {
        try {
          await browser.close();
          metrics.browserPoolRecycles.inc();
        } catch (error) {
          const message = error instanceof Error ? error.message : String(error);
          logger.warn('Failed to recycle idle browser', {
            error: message,
            browserKey: key === DEFAULT_BROWSER_KEY ? 'default' : key,
          });
          metrics.cleanupFailures.inc({ operation: 'browser_idle_recycle' });
          throw error;
        }
      });
      if (recycled > 0) logger.info('browser: recycled idle pooled browsers', { count: recycled });
      return recycled;
    } finally {
      this.updatePoolMetrics();
    }
  }

  updatePoolMetrics(): void {
    const usage = this.pool.usageSnapshot();
    metrics.browserPoolBrowsers.set(usage.browserCount);
    metrics.browserPoolContexts.set(usage.contextCount);
    metrics.browserPoolIdleBrowsers.set(usage.idleBrowserCount);
    metrics.browserPoolMaxIdleSeconds.set(usage.maxIdleSeconds);
  }

  /** True only when one managed browser/context is the entire driver-owned
   * browser pool. Used to keep process-tree measurements session attributable. */
  canAttributeProcessTree(browser: Browser): boolean {
    return this.pool.ownsExclusiveContext(browser);
  }

  /** Start the pool's idle grace period when a caller's browser work ends. */
  markPoolUsedNow(): void {
    this.pool.markUsed();
    this.updatePoolMetrics();
  }

  /**
   * Shutdown all browsers and cleanup resources.
   */
  async shutdown(): Promise<void> {
    try {
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
    } finally {
      this.updatePoolMetrics();
    }
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
  strategy: AudioStrategy = 'host_device',
  allowExtensionDebugging = false,
  extensionIsolationKey = '',
  backend: BrowserBackend = 'managed',
  executablePath = ''
): string {
  const extensionKey = allowExtensionDebugging
    ? `enabled:${extensionIsolationKey.trim()}`
    : 'disabled';
  return `${strategy}\u0000${fakeMicrophoneWav}\u0000${extensionKey}\u0000${backend}\u0000${executablePath}`;
}

/** Resolve only an explicitly configured, existing executable; never search or download. */
export function resolveInstalledExecutablePath(configuredPath?: string): string {
  const candidate = configuredPath?.trim() ?? '';
  if (!candidate || !isAbsolute(candidate)) {
    throw new Error('installed-binary backend requires an absolute BROWSER_EXECUTABLE_PATH');
  }
  let resolved: string;
  try {
    resolved = realpathSync(candidate);
    if (!statSync(resolved).isFile()) throw new Error('not a regular file');
    accessSync(resolved, constants.X_OK);
  } catch {
    throw new Error('configured BROWSER_EXECUTABLE_PATH is missing or not executable');
  }
  return resolved;
}

/** Require an owner-observed Chromium version before creating any session context. */
export function validateInstalledBrowserVersion(version: string): void {
  if (!/^\d+\.\d+(?:\.\d+){1,2}$/.test(version.trim())) {
    throw new NonRetryableBrowserLaunchError('installed browser did not report a valid Chromium version');
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
