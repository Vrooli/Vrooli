/**
 * External URL injection verification for an active recording session.
 */

import type { Page } from 'rebrowser-playwright';
import type { RecordingContextInitializer } from '../io/context-initializer';
import { logger, scopedLog, LogContext } from '../../utils';
import { verifyScriptInjection } from '../validation/verification';

/**
 * Result of the external URL injection test.
 */
export interface ExternalUrlTestResult {
  /** Whether the test passed */
  success: boolean;
  /** Timestamp of the test */
  timestamp: string;
  /** Duration in ms */
  durationMs: number;
  /** URL that was tested */
  testedUrl: string;
  /** Where the test failed (if it failed) */
  failurePoint?: 'fetch' | 'modify' | 'fulfill' | 'script_load' | 'script_ready' | 'context_wrong' | 'network' | 'timeout';
  /** Detailed failure message */
  failureMessage?: string;
  /** Suggestions for fixing */
  suggestions?: string[];
  /** Script verification results */
  verification?: {
    loaded: boolean;
    ready: boolean;
    inMainContext: boolean;
    handlersCount: number;
    version: string | null;
    error?: string;
  };
  /** Injection stats at time of test */
  injectionStats?: {
    attempted: number;
    successful: number;
    failed: number;
    avgInjectionTimeMs: number;
    lastInjectionAt: string | null;
  };
}

/**
 * Test the recording script injection on a real external URL.
 *
 * This tests the ACTUAL injection path that real users experience:
 * 1. context.route() intercepts the navigation request
 * 2. route.fetch() fetches the HTML from the external server
 * 3. HTML is modified to inject the recording script
 * 4. route.fulfill() serves the modified HTML to the browser
 *
 * This is critical because the standard pipeline test uses a dedicated route
 * that serves pre-built HTML, bypassing steps 2-3.
 *
 * @param page - The Playwright page to test on
 * @param contextInitializer - The recording context initializer
 * @param options - Test options
 */
export async function runExternalUrlInjectionTest(
  page: Page,
  contextInitializer: RecordingContextInitializer,
  options: {
    /** URL to test (default: https://example.com) */
    testUrl?: string;
    /** Timeout in ms */
    timeoutMs?: number;
  } = {}
): Promise<ExternalUrlTestResult> {
  const {
    testUrl = 'https://example.com',
    timeoutMs = 30000,
  } = options;

  const startTime = Date.now();

  logger.info(scopedLog(LogContext.RECORDING, 'starting external URL injection test'), {
    testUrl,
    timeoutMs,
  });

  // Get initial injection stats
  const statsBefore = contextInitializer.getInjectionStats();

  try {
    // Navigate to the external URL
    // This will trigger the route interception and injection path
    logger.info(scopedLog(LogContext.RECORDING, 'navigating to external URL'), { testUrl });

    await page.goto(testUrl, {
      waitUntil: 'domcontentloaded',
      timeout: timeoutMs,
    });

    // Wait a moment for script initialization
    await sleep(500);

    // Get injection stats after navigation
    const statsAfter = contextInitializer.getInjectionStats();
    const injectionAttempted = statsAfter.attempted > statsBefore.attempted;
    const injectionSucceeded = statsAfter.successful > statsBefore.successful;

    logger.info(scopedLog(LogContext.RECORDING, 'injection stats after navigation'), {
      before: statsBefore,
      after: statsAfter,
      attempted: injectionAttempted,
      succeeded: injectionSucceeded,
    });

    // Verify script injection
    const verification = await verifyScriptInjection(page);

    logger.info(scopedLog(LogContext.RECORDING, 'script verification result'), {
      loaded: verification.loaded,
      ready: verification.ready,
      inMainContext: verification.inMainContext,
      handlersCount: verification.handlersCount,
      version: verification.version,
      error: verification.error,
    });

    // Analyze results
    if (!injectionAttempted) {
      return {
        success: false,
        timestamp: new Date().toISOString(),
        durationMs: Date.now() - startTime,
        testedUrl: testUrl,
        failurePoint: 'fetch',
        failureMessage: 'Injection was not attempted - route interception may not have triggered',
        suggestions: [
          'Check that context.route("**/*") is set up before navigation',
          'Verify the URL is an HTTP(S) URL (not about:blank or data URL)',
          'Check if another route is handling this URL first',
        ],
        verification: {
          loaded: verification.loaded,
          ready: verification.ready,
          inMainContext: verification.inMainContext,
          handlersCount: verification.handlersCount,
          version: verification.version,
          error: verification.error,
        },
        injectionStats: statsAfter,
      };
    }

    if (!injectionSucceeded) {
      const failed = statsAfter.failed > statsBefore.failed;
      return {
        success: false,
        timestamp: new Date().toISOString(),
        durationMs: Date.now() - startTime,
        testedUrl: testUrl,
        failurePoint: failed ? 'fetch' : 'fulfill',
        failureMessage: failed
          ? 'Injection was attempted but route.fetch() failed - external server may be unreachable'
          : 'Injection stats show attempted but not successful - route.fulfill() may have failed',
        suggestions: [
          'Check network connectivity to the external URL',
          'Check for CORS or security restrictions',
          'Look at driver logs for detailed error messages',
          'The server may have returned a non-HTML response',
        ],
        verification: {
          loaded: verification.loaded,
          ready: verification.ready,
          inMainContext: verification.inMainContext,
          handlersCount: verification.handlersCount,
          version: verification.version,
          error: verification.error,
        },
        injectionStats: statsAfter,
      };
    }

    // Injection succeeded according to stats, but did the script actually load?
    if (!verification.loaded) {
      return {
        success: false,
        timestamp: new Date().toISOString(),
        durationMs: Date.now() - startTime,
        testedUrl: testUrl,
        failurePoint: 'script_load',
        failureMessage: 'Injection stats show success but script did not load in browser - HTML may not have been served correctly',
        suggestions: [
          'Check if route.fulfill() completed successfully',
          'The browser may have rejected the modified response',
          'Check for CSP headers blocking inline scripts',
          'The response body may not have been properly modified',
        ],
        verification: {
          loaded: verification.loaded,
          ready: verification.ready,
          inMainContext: verification.inMainContext,
          handlersCount: verification.handlersCount,
          version: verification.version,
          error: verification.error,
        },
        injectionStats: statsAfter,
      };
    }

    if (!verification.ready) {
      return {
        success: false,
        timestamp: new Date().toISOString(),
        durationMs: Date.now() - startTime,
        testedUrl: testUrl,
        failurePoint: 'script_ready',
        failureMessage: 'Script loaded but failed to initialize - may have crashed during setup',
        suggestions: [
          'Check browser console for JavaScript errors',
          'The recording script may conflict with page scripts',
          'Initialization error: ' + (verification.initError || 'unknown'),
        ],
        verification: {
          loaded: verification.loaded,
          ready: verification.ready,
          inMainContext: verification.inMainContext,
          handlersCount: verification.handlersCount,
          version: verification.version,
          error: verification.error,
        },
        injectionStats: statsAfter,
      };
    }

    if (!verification.inMainContext) {
      return {
        success: false,
        timestamp: new Date().toISOString(),
        durationMs: Date.now() - startTime,
        testedUrl: testUrl,
        failurePoint: 'context_wrong',
        failureMessage: 'Script is running in ISOLATED context instead of MAIN - History API events will NOT be captured',
        suggestions: [
          'Script must be injected via HTML modification, not page.evaluate()',
          'Check that HTML injection is working correctly',
          'This is a critical issue - recording will miss navigation events',
        ],
        verification: {
          loaded: verification.loaded,
          ready: verification.ready,
          inMainContext: verification.inMainContext,
          handlersCount: verification.handlersCount,
          version: verification.version,
          error: verification.error,
        },
        injectionStats: statsAfter,
      };
    }

    // All checks passed!
    logger.info(scopedLog(LogContext.RECORDING, 'external URL injection test PASSED'), {
      testUrl,
      durationMs: Date.now() - startTime,
      handlersCount: verification.handlersCount,
      version: verification.version,
    });

    return {
      success: true,
      timestamp: new Date().toISOString(),
      durationMs: Date.now() - startTime,
      testedUrl: testUrl,
      verification: {
        loaded: verification.loaded,
        ready: verification.ready,
        inMainContext: verification.inMainContext,
        handlersCount: verification.handlersCount,
        version: verification.version,
      },
      injectionStats: statsAfter,
    };

  } catch (error) {
    const isTimeout = error instanceof Error && error.message.includes('Timeout');
    const isNetwork = error instanceof Error && (
      error.message.includes('net::') ||
      error.message.includes('ERR_') ||
      error.message.includes('ECONNREFUSED')
    );

    logger.error(scopedLog(LogContext.RECORDING, 'external URL injection test FAILED'), {
      testUrl,
      error: error instanceof Error ? error.message : String(error),
      isTimeout,
      isNetwork,
    });

    return {
      success: false,
      timestamp: new Date().toISOString(),
      durationMs: Date.now() - startTime,
      testedUrl: testUrl,
      failurePoint: isTimeout ? 'timeout' : isNetwork ? 'network' : 'fetch',
      failureMessage: error instanceof Error ? error.message : String(error),
      suggestions: isTimeout
        ? ['Increase timeoutMs option', 'Check if the external URL is responding']
        : isNetwork
          ? ['Check network connectivity', 'The external URL may be blocked or down']
          : ['Check driver logs for more details'],
      injectionStats: contextInitializer.getInjectionStats(),
    };
  }
}


function sleep(ms: number): Promise<void> {
  return new Promise(resolve => setTimeout(resolve, ms));
}
