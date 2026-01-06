/**
 * Script Injection Verification
 *
 * Provides functions to verify that the recording script was properly
 * injected and initialized in the browser page.
 *
 * ## Context Isolation Note
 *
 * With rebrowser-playwright, page.evaluate() runs in ISOLATED context
 * to evade bot detection. But the recording script runs in MAIN context
 * (injected via HTML). Window properties set in MAIN are NOT visible
 * from ISOLATED context.
 *
 * To verify script injection, we use CDP (Chrome DevTools Protocol) to
 * evaluate in the MAIN context directly.
 *
 * ## Usage
 *
 * ```typescript
 * import { verifyScriptInjection } from './verification';
 *
 * const result = await verifyScriptInjection(page);
 * if (!result.ready) {
 *   logger.error('Script injection failed', result);
 * }
 * ```
 *
 * ## Verification Markers
 *
 * The recording script sets these window properties:
 * - `__vrooli_recording_script_loaded`: true when script executes
 * - `__vrooli_recording_script_load_time`: timestamp of execution
 * - `__vrooli_recording_script_version`: script version string
 * - `__vrooli_recording_script_context`: 'MAIN' if in correct context
 * - `__vrooli_recording_ready`: true when fully initialized
 * - `__vrooli_recording_handlers_count`: number of registered handlers
 * - `__vrooli_recording_init_error`: error message if initialization failed
 *
 * @module recording/verification
 */

import type { Page, CDPSession } from 'rebrowser-playwright';

/**
 * Result of script injection verification.
 */
export interface InjectionVerification {
  /**
   * Whether the script was loaded (executed at all).
   * This is set immediately when the script starts executing.
   */
  loaded: boolean;

  /**
   * Timestamp when the script was loaded (ms since epoch).
   * Null if script was not loaded.
   */
  loadTime: number | null;

  /**
   * Version of the injected script.
   * Null if script was not loaded.
   */
  version: string | null;

  /**
   * Whether the script is fully initialized and ready.
   * This is set at the end of initialization after all handlers are registered.
   */
  ready: boolean;

  /**
   * Number of event handlers registered by the script.
   * Useful for verifying complete initialization.
   */
  handlersCount: number;

  /**
   * Whether the script is running in MAIN context (not ISOLATED).
   * This is critical for History API wrapping to work.
   */
  inMainContext: boolean;

  /**
   * Error message if verification failed or script had an error.
   */
  error?: string;

  /**
   * Error that occurred during script initialization.
   * Set if the script's try-catch caught an error.
   */
  initError?: string;
}

/**
 * Window interface with recording verification markers.
 * These are set by the recording script in the browser.
 */
interface RecordingWindow {
  __vrooli_recording_script_loaded?: boolean;
  __vrooli_recording_script_load_time?: number;
  __vrooli_recording_script_version?: string;
  __vrooli_recording_script_context?: string;
  __vrooli_recording_ready?: boolean;
  __vrooli_recording_handlers_count?: number;
  __vrooli_recording_init_error?: string;
}

/**
 * Verify that the recording script was properly injected and initialized.
 *
 * This function uses CDP (Chrome DevTools Protocol) to evaluate in the
 * MAIN execution context, bypassing rebrowser-playwright's context isolation.
 * This is necessary because the recording script runs in MAIN context, but
 * page.evaluate() runs in ISOLATED context with rebrowser-playwright.
 *
 * @param page - The Playwright page to verify
 * @returns Verification result with diagnostic details
 *
 * @example
 * ```typescript
 * const verification = await verifyScriptInjection(page);
 *
 * if (!verification.loaded) {
 *   // Script was never executed - injection failed
 * } else if (!verification.ready) {
 *   // Script started but didn't complete - initialization error
 * } else if (!verification.inMainContext) {
 *   // Script ran in wrong context - History API won't work
 * } else {
 *   // All good!
 * }
 * ```
 */
export async function verifyScriptInjection(page: Page): Promise<InjectionVerification> {
  try {
    // Use CDP to evaluate in MAIN context
    // This bypasses rebrowser-playwright's ISOLATED context isolation
    const client = await page.context().newCDPSession(page);

    try {
      // Get the main frame's execution context
      const { result } = await client.send('Runtime.evaluate', {
        expression: `(function() {
          return JSON.stringify({
            loaded: window.__vrooli_recording_script_loaded === true,
            loadTime: window.__vrooli_recording_script_load_time || null,
            version: window.__vrooli_recording_script_version || null,
            ready: window.__vrooli_recording_ready === true,
            handlersCount: window.__vrooli_recording_handlers_count || 0,
            inMainContext: window.__vrooli_recording_script_context === 'MAIN',
            initError: window.__vrooli_recording_init_error || undefined
          });
        })()`,
        returnByValue: true,
      });

      if (result.type === 'string' && result.value) {
        return JSON.parse(result.value) as InjectionVerification;
      }

      return {
        loaded: false,
        loadTime: null,
        version: null,
        ready: false,
        handlersCount: 0,
        inMainContext: false,
        error: 'CDP evaluation returned unexpected result',
      };
    } finally {
      await client.detach().catch(() => {});
    }
  } catch (error) {
    return {
      loaded: false,
      loadTime: null,
      version: null,
      ready: false,
      handlersCount: 0,
      inMainContext: false,
      error: error instanceof Error ? error.message : String(error),
    };
  }
}

/**
 * Verify script injection and throw if not ready.
 *
 * Convenience function that throws an error if verification fails.
 * Useful for assertions in tests or initialization flows.
 *
 * @param page - The Playwright page to verify
 * @param options - Verification options
 * @throws Error if script is not properly injected and ready
 */
export async function assertScriptInjected(
  page: Page,
  options: { requireMainContext?: boolean } = {}
): Promise<InjectionVerification> {
  const { requireMainContext = true } = options;

  const verification = await verifyScriptInjection(page);

  if (!verification.loaded) {
    throw new Error(
      `Recording script not loaded. ` +
        `Verification error: ${verification.error ?? 'unknown'}. ` +
        `Check that HTML injection is working.`
    );
  }

  if (!verification.ready) {
    throw new Error(
      `Recording script loaded but not ready. ` +
        `Init error: ${verification.initError ?? 'unknown'}. ` +
        `Handlers registered: ${verification.handlersCount}. ` +
        `Check for JavaScript errors in browser console.`
    );
  }

  if (requireMainContext && !verification.inMainContext) {
    throw new Error(
      `Recording script running in ISOLATED context instead of MAIN. ` +
        `History API wrapping will not work. ` +
        `Script version: ${verification.version ?? 'unknown'}. ` +
        `Check that script is injected via HTML, not page.evaluate().`
    );
  }

  return verification;
}

/**
 * Wait for script to be ready with timeout.
 *
 * Polls for script readiness, useful when script injection
 * may be delayed (e.g., after navigation).
 *
 * @param page - The Playwright page to verify
 * @param timeoutMs - Maximum time to wait (default: 5000ms)
 * @param pollIntervalMs - Polling interval (default: 100ms)
 * @returns Verification result when ready, or latest state on timeout
 */
export async function waitForScriptReady(
  page: Page,
  timeoutMs = 5000,
  pollIntervalMs = 100
): Promise<InjectionVerification> {
  const startTime = Date.now();

  while (Date.now() - startTime < timeoutMs) {
    const verification = await verifyScriptInjection(page);

    if (verification.ready) {
      return verification;
    }

    // If there was an init error, no point waiting
    if (verification.initError) {
      return verification;
    }

    await new Promise((resolve) => setTimeout(resolve, pollIntervalMs));
  }

  // Return final state on timeout
  return verifyScriptInjection(page);
}
