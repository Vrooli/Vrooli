/** Verify init-script recording injection on a real external URL. */

import type { Page } from 'rebrowser-playwright';
import type { RecordingContextInitializer } from '../io/context-initializer';
import { logger, scopedLog, LogContext } from '../../utils';
import { verifyScriptInjection } from '../validation/verification';

export interface ExternalUrlTestResult {
  success: boolean;
  timestamp: string;
  durationMs: number;
  testedUrl: string;
  failurePoint?: 'script_load' | 'script_ready' | 'context_wrong' | 'network' | 'timeout';
  failureMessage?: string;
  suggestions?: string[];
  verification?: {
    loaded: boolean;
    ready: boolean;
    inMainContext: boolean;
    handlersCount: number;
    version: string | null;
    error?: string;
  };
  injectionStats?: {
    attempted: number;
    successful: number;
    failed: number;
    avgInjectionTimeMs: number;
    lastInjectionAt: string | null;
  };
}

/** Navigate to an external URL and verify the registered context init script. */
export async function runExternalUrlInjectionTest(
  page: Page,
  contextInitializer: RecordingContextInitializer,
  options: { testUrl?: string; timeoutMs?: number } = {}
): Promise<ExternalUrlTestResult> {
  const { testUrl = 'https://example.com', timeoutMs = 30000 } = options;
  const startTime = Date.now();
  logger.info(scopedLog(LogContext.RECORDING, 'starting external URL init-script verification'), {
    testUrl,
    timeoutMs,
  });

  const makeResult = (
    success: boolean,
    extra: Pick<ExternalUrlTestResult, 'failurePoint' | 'failureMessage' | 'suggestions' | 'verification'> = {}
  ): ExternalUrlTestResult => ({
    success,
    timestamp: new Date().toISOString(),
    durationMs: Date.now() - startTime,
    testedUrl: testUrl,
    injectionStats: contextInitializer.getInjectionStats(),
    ...extra,
  });

  try {
    await page.goto(testUrl, { waitUntil: 'domcontentloaded', timeout: timeoutMs });
    await sleep(500);
    const verification = await verifyScriptInjection(page);
    const verificationDetails = {
      loaded: verification.loaded,
      ready: verification.ready,
      inMainContext: verification.inMainContext,
      handlersCount: verification.handlersCount,
      version: verification.version,
      error: verification.error,
    };

    if (!verification.loaded) {
      return makeResult(false, {
        failurePoint: 'script_load',
        failureMessage: 'The recording init script did not load on the external page.',
        suggestions: [
          'Confirm the page belongs to the initialized browser context.',
          'Check browser console errors and page content security policy.',
        ],
        verification: verificationDetails,
      });
    }
    if (!verification.ready) {
      return makeResult(false, {
        failurePoint: 'script_ready',
        failureMessage: 'The recording init script loaded but did not finish initialization.',
        suggestions: ['Check browser console errors.', `Initialization error: ${verification.initError || 'unknown'}`],
        verification: verificationDetails,
      });
    }
    if (!verification.inMainContext) {
      return makeResult(false, {
        failurePoint: 'context_wrong',
        failureMessage: 'The recording script is not running in the main page context.',
        suggestions: ['Verify that the context.addInitScript() registration is active.'],
        verification: verificationDetails,
      });
    }

    logger.info(scopedLog(LogContext.RECORDING, 'external URL init-script verification passed'), {
      testUrl,
      handlersCount: verification.handlersCount,
      version: verification.version,
    });
    return makeResult(true, { verification: verificationDetails });
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    const isTimeout = message.includes('Timeout');
    const isNetwork = /net::|ERR_|ECONNREFUSED/.test(message);
    return makeResult(false, {
      failurePoint: isTimeout ? 'timeout' : isNetwork ? 'network' : 'script_load',
      failureMessage: message,
      suggestions: isNetwork
        ? ['Check network connectivity to the external URL.']
        : isTimeout
          ? ['Check whether the external page is reachable and completes navigation.']
          : ['Check the driver logs and browser console.'],
    });
  }
}

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}
