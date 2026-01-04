/**
 * Recording Diagnostics Module
 *
 * Provides comprehensive diagnostics for the recording system.
 * Use this to:
 * 1. Verify recording is properly configured before user interaction
 * 2. Diagnose issues when recording doesn't work
 * 3. Generate structured diagnostic reports for debugging
 *
 * ## Quick Start
 *
 * ```typescript
 * import { runRecordingDiagnostics, RecordingDiagnosticLevel } from './diagnostics';
 *
 * // Run full diagnostics before starting recording
 * const result = await runRecordingDiagnostics(page, context, {
 *   level: RecordingDiagnosticLevel.FULL,
 *   logResults: true,
 * });
 *
 * if (!result.ready) {
 *   console.error('Recording not ready:', result.issues);
 * }
 * ```
 *
 * ## Integration with Context Initializer
 *
 * The context initializer can optionally run diagnostics after setup:
 *
 * ```typescript
 * const initializer = createRecordingContextInitializer({
 *   diagnosticsEnabled: true, // Enables detailed logging
 *   runSanityCheck: true,     // Runs diagnostics after first page load
 * });
 * ```
 *
 * @module recording/diagnostics
 */

import type { Page, BrowserContext } from 'rebrowser-playwright';
import type winston from 'winston';
import { verifyScriptInjection, type InjectionVerification } from './verification';
import type { RecordingContextInitializer, InjectionStats } from './context-initializer';
import { playwrightProvider } from '../playwright';
import { logger as defaultLogger, LogContext, scopedLog } from '../utils';

// =============================================================================
// Types
// =============================================================================

/**
 * Diagnostic level determines how thorough the checks are.
 */
export enum RecordingDiagnosticLevel {
  /** Just check if script is loaded and ready */
  QUICK = 'quick',
  /** Check script + verify it's in correct context */
  STANDARD = 'standard',
  /** Full check including event flow test */
  FULL = 'full',
}

/**
 * Diagnostic issue severity.
 */
export enum DiagnosticSeverity {
  /** Recording won't work at all */
  ERROR = 'error',
  /** Recording may have issues */
  WARNING = 'warning',
  /** Informational, not a problem */
  INFO = 'info',
}

/**
 * A single diagnostic issue found.
 */
export interface DiagnosticIssue {
  /** Unique identifier for the issue type */
  code: string;
  /** Human-readable description */
  message: string;
  /** How serious is this issue */
  severity: DiagnosticSeverity;
  /** Suggested fix or next step */
  suggestion?: string;
  /** Additional context */
  details?: Record<string, unknown>;
}

/**
 * Complete diagnostic result.
 */
export interface RecordingDiagnosticResult {
  /** Whether recording is ready to capture events */
  ready: boolean;
  /** Timestamp of the diagnostic run */
  timestamp: string;
  /** How long diagnostics took (ms) */
  durationMs: number;
  /** Diagnostic level that was run */
  level: RecordingDiagnosticLevel;
  /** Issues found (empty if all good) */
  issues: DiagnosticIssue[];
  /** Script injection verification result */
  scriptVerification?: InjectionVerification;
  /** Injection statistics from context initializer */
  injectionStats?: InjectionStats;
  /** Provider information */
  provider: {
    name: string;
    evaluateIsolated: boolean;
    exposeBindingIsolated: boolean;
  };
  /** Event flow test result (only for FULL level) */
  eventFlowTest?: {
    passed: boolean;
    eventSent: boolean;
    eventReceived: boolean;
    latencyMs?: number;
    error?: string;
  };
}

/**
 * Options for running diagnostics.
 */
export interface DiagnosticOptions {
  /** How thorough to be (default: STANDARD) */
  level?: RecordingDiagnosticLevel;
  /** Log results to console (default: false) */
  logResults?: boolean;
  /** Timeout for async checks in ms (default: 5000) */
  timeoutMs?: number;
  /** Logger instance */
  logger?: winston.Logger;
  /** Context initializer for injection stats */
  contextInitializer?: RecordingContextInitializer;
}

// =============================================================================
// Issue Codes
// =============================================================================

/**
 * Known diagnostic issue codes.
 * Use these for programmatic handling of issues.
 */
export const DIAGNOSTIC_CODES = {
  // Script loading issues
  SCRIPT_NOT_LOADED: 'SCRIPT_NOT_LOADED',
  SCRIPT_NOT_READY: 'SCRIPT_NOT_READY',
  SCRIPT_INIT_ERROR: 'SCRIPT_INIT_ERROR',
  SCRIPT_WRONG_CONTEXT: 'SCRIPT_WRONG_CONTEXT',
  SCRIPT_LOW_HANDLERS: 'SCRIPT_LOW_HANDLERS',

  // Injection issues
  INJECTION_NO_ATTEMPTS: 'INJECTION_NO_ATTEMPTS',
  INJECTION_ALL_FAILED: 'INJECTION_ALL_FAILED',
  INJECTION_HIGH_FAILURE_RATE: 'INJECTION_HIGH_FAILURE_RATE',

  // Event flow issues
  EVENT_SEND_FAILED: 'EVENT_SEND_FAILED',
  EVENT_NOT_RECEIVED: 'EVENT_NOT_RECEIVED',
  EVENT_HIGH_LATENCY: 'EVENT_HIGH_LATENCY',

  // Configuration issues
  PROVIDER_MISMATCH: 'PROVIDER_MISMATCH',
  CDP_NOT_AVAILABLE: 'CDP_NOT_AVAILABLE',
} as const;

// =============================================================================
// Diagnostic Functions
// =============================================================================

/**
 * Run recording diagnostics on a page.
 *
 * This is the main entry point for diagnosing recording issues.
 * It checks script injection, context, and optionally event flow.
 *
 * @param page - The Playwright page to diagnose
 * @param context - The browser context (for event flow test)
 * @param options - Diagnostic options
 * @returns Diagnostic result with issues found
 *
 * @example
 * ```typescript
 * // Quick check before recording
 * const result = await runRecordingDiagnostics(page, context);
 * if (!result.ready) {
 *   for (const issue of result.issues) {
 *     console.error(`[${issue.severity}] ${issue.code}: ${issue.message}`);
 *     if (issue.suggestion) {
 *       console.log(`  Suggestion: ${issue.suggestion}`);
 *     }
 *   }
 * }
 * ```
 */
export async function runRecordingDiagnostics(
  page: Page,
  context: BrowserContext,
  options: DiagnosticOptions = {}
): Promise<RecordingDiagnosticResult> {
  const {
    level = RecordingDiagnosticLevel.STANDARD,
    logResults = false,
    timeoutMs = 5000,
    logger = defaultLogger,
    contextInitializer,
  } = options;

  const startTime = Date.now();
  const issues: DiagnosticIssue[] = [];

  // Collect provider info
  const provider = {
    name: playwrightProvider.name,
    evaluateIsolated: playwrightProvider.capabilities.evaluateIsolated,
    exposeBindingIsolated: playwrightProvider.capabilities.exposeBindingIsolated,
  };

  // Get injection stats if available
  let injectionStats: InjectionStats | undefined;
  if (contextInitializer) {
    injectionStats = contextInitializer.getInjectionStats();
    checkInjectionStats(injectionStats, issues);
  }

  // Verify script injection via CDP
  let scriptVerification: InjectionVerification | undefined;
  try {
    scriptVerification = await verifyScriptInjection(page);
    checkScriptVerification(scriptVerification, issues, level);
  } catch (error) {
    issues.push({
      code: DIAGNOSTIC_CODES.CDP_NOT_AVAILABLE,
      message: 'Failed to verify script injection via CDP',
      severity: DiagnosticSeverity.ERROR,
      suggestion: 'CDP may not be available. Check browser launch options.',
      details: { error: error instanceof Error ? error.message : String(error) },
    });
  }

  // Run event flow test for FULL level
  let eventFlowTest: RecordingDiagnosticResult['eventFlowTest'];
  if (level === RecordingDiagnosticLevel.FULL) {
    eventFlowTest = await testEventFlow(page, context, timeoutMs, logger);
    checkEventFlowTest(eventFlowTest, issues);
  }

  const durationMs = Date.now() - startTime;

  // Determine if ready (no ERROR severity issues)
  const ready = !issues.some((i) => i.severity === DiagnosticSeverity.ERROR);

  const result: RecordingDiagnosticResult = {
    ready,
    timestamp: new Date().toISOString(),
    durationMs,
    level,
    issues,
    scriptVerification,
    injectionStats,
    provider,
    eventFlowTest,
  };

  // Log results if requested
  if (logResults) {
    logDiagnosticResult(result, logger);
  }

  return result;
}

/**
 * Check injection statistics for issues.
 */
function checkInjectionStats(stats: InjectionStats, issues: DiagnosticIssue[]): void {
  if (stats.attempted === 0) {
    issues.push({
      code: DIAGNOSTIC_CODES.INJECTION_NO_ATTEMPTS,
      message: 'No HTML injection attempts recorded',
      severity: DiagnosticSeverity.WARNING,
      suggestion: 'Navigate to a page first, then run diagnostics',
    });
    return;
  }

  if (stats.successful === 0) {
    issues.push({
      code: DIAGNOSTIC_CODES.INJECTION_ALL_FAILED,
      message: 'All HTML injection attempts failed',
      severity: DiagnosticSeverity.ERROR,
      suggestion: 'Check route interception setup and HTML structure of pages',
      details: { attempted: stats.attempted, failed: stats.failed },
    });
    return;
  }

  const failureRate = stats.failed / stats.attempted;
  if (failureRate > 0.2) {
    issues.push({
      code: DIAGNOSTIC_CODES.INJECTION_HIGH_FAILURE_RATE,
      message: `High injection failure rate: ${(failureRate * 100).toFixed(1)}%`,
      severity: DiagnosticSeverity.WARNING,
      suggestion: 'Some pages may not have recording enabled',
      details: { successful: stats.successful, failed: stats.failed, attempted: stats.attempted },
    });
  }
}

/**
 * Check script verification for issues.
 */
function checkScriptVerification(
  verification: InjectionVerification,
  issues: DiagnosticIssue[],
  level: RecordingDiagnosticLevel
): void {
  if (!verification.loaded) {
    issues.push({
      code: DIAGNOSTIC_CODES.SCRIPT_NOT_LOADED,
      message: 'Recording script not loaded on page',
      severity: DiagnosticSeverity.ERROR,
      suggestion: 'Check HTML injection route and that page was navigated via HTTP(S)',
      details: { error: verification.error },
    });
    return;
  }

  if (verification.initError) {
    issues.push({
      code: DIAGNOSTIC_CODES.SCRIPT_INIT_ERROR,
      message: `Script initialization error: ${verification.initError}`,
      severity: DiagnosticSeverity.ERROR,
      suggestion: 'Check browser console for JavaScript errors in recording script',
    });
    return;
  }

  if (!verification.ready) {
    issues.push({
      code: DIAGNOSTIC_CODES.SCRIPT_NOT_READY,
      message: 'Recording script loaded but not ready',
      severity: DiagnosticSeverity.ERROR,
      suggestion: 'Script may have crashed during initialization',
      details: { handlersCount: verification.handlersCount },
    });
    return;
  }

  // Context check only matters for STANDARD and FULL levels
  if (level !== RecordingDiagnosticLevel.QUICK && !verification.inMainContext) {
    issues.push({
      code: DIAGNOSTIC_CODES.SCRIPT_WRONG_CONTEXT,
      message: 'Recording script running in ISOLATED context instead of MAIN',
      severity: DiagnosticSeverity.ERROR,
      suggestion:
        'Script was likely injected via page.evaluate() instead of HTML. ' +
        'History API navigation events will not be captured.',
    });
    return;
  }

  // Check handler count
  const expectedMinHandlers = 7; // Core handlers
  if (verification.handlersCount < expectedMinHandlers) {
    issues.push({
      code: DIAGNOSTIC_CODES.SCRIPT_LOW_HANDLERS,
      message: `Low handler count: ${verification.handlersCount} (expected ${expectedMinHandlers}+)`,
      severity: DiagnosticSeverity.WARNING,
      suggestion: 'Some event types may not be captured',
      details: { handlersCount: verification.handlersCount, expected: expectedMinHandlers },
    });
  }
}

/**
 * Test the event flow from browser to Node.js.
 *
 * Note: context and logger are reserved for future enhancements
 * where we might set up a temporary route or add more logging.
 */
async function testEventFlow(
  page: Page,
  _context: BrowserContext,
  timeoutMs: number,
  _logger: winston.Logger
): Promise<RecordingDiagnosticResult['eventFlowTest']> {
  const testEventId = `diag-${Date.now()}`;
  let eventReceived = false;
  let receiveTime = 0;

  // Create a temporary route to capture the test event
  const testPromise = new Promise<void>((resolve) => {
    const timeout = setTimeout(resolve, timeoutMs);

    // We can't easily add a temporary route, so we'll use the page's console
    // to verify the event was at least sent
    page.on('console', (msg) => {
      if (msg.text().includes(testEventId)) {
        receiveTime = Date.now();
        eventReceived = true;
        clearTimeout(timeout);
        resolve();
      }
    });
  });

  // Send a test event from the browser
  const sendTime = Date.now();
  let eventSent = false;

  try {
    await page.evaluate((eventId: string) => {
      console.log(`[DiagnosticTest] ${eventId}`);
      // Also try sending via the recording event URL
      fetch('/__vrooli_recording_event__', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ actionType: 'diagnostic', testId: eventId }),
        keepalive: true,
      }).catch(() => {
        // Expected to fail if route not set up
      });
    }, testEventId);
    eventSent = true;
  } catch (error) {
    return {
      passed: false,
      eventSent: false,
      eventReceived: false,
      error: error instanceof Error ? error.message : String(error),
    };
  }

  // Wait for the event
  await testPromise;

  const latencyMs = eventReceived ? receiveTime - sendTime : undefined;

  return {
    passed: eventSent && eventReceived,
    eventSent,
    eventReceived,
    latencyMs,
  };
}

/**
 * Check event flow test results for issues.
 */
function checkEventFlowTest(
  test: RecordingDiagnosticResult['eventFlowTest'],
  issues: DiagnosticIssue[]
): void {
  if (!test) return;

  if (!test.eventSent) {
    issues.push({
      code: DIAGNOSTIC_CODES.EVENT_SEND_FAILED,
      message: 'Failed to send test event from browser',
      severity: DiagnosticSeverity.ERROR,
      suggestion: 'page.evaluate() may have failed',
      details: { error: test.error },
    });
    return;
  }

  if (!test.eventReceived) {
    issues.push({
      code: DIAGNOSTIC_CODES.EVENT_NOT_RECEIVED,
      message: 'Test event sent but not received by Node.js',
      severity: DiagnosticSeverity.WARNING,
      suggestion:
        'Event route may not be set up, or console message was not captured. ' +
        'This may be a false positive in the diagnostic.',
    });
    return;
  }

  if (test.latencyMs && test.latencyMs > 1000) {
    issues.push({
      code: DIAGNOSTIC_CODES.EVENT_HIGH_LATENCY,
      message: `High event latency: ${test.latencyMs}ms`,
      severity: DiagnosticSeverity.WARNING,
      suggestion: 'Events may be delayed. Check route handler performance.',
      details: { latencyMs: test.latencyMs },
    });
  }
}

/**
 * Log diagnostic result in a structured format.
 */
function logDiagnosticResult(result: RecordingDiagnosticResult, logger: winston.Logger): void {
  const status = result.ready ? 'READY' : 'NOT READY';

  logger.info(scopedLog(LogContext.RECORDING, `diagnostics: ${status}`), {
    ready: result.ready,
    level: result.level,
    durationMs: result.durationMs,
    issueCount: result.issues.length,
    provider: result.provider.name,
  });

  for (const issue of result.issues) {
    const logFn =
      issue.severity === DiagnosticSeverity.ERROR
        ? logger.error.bind(logger)
        : issue.severity === DiagnosticSeverity.WARNING
          ? logger.warn.bind(logger)
          : logger.info.bind(logger);

    logFn(scopedLog(LogContext.RECORDING, `diagnostic issue: ${issue.code}`), {
      message: issue.message,
      suggestion: issue.suggestion,
      details: issue.details,
    });
  }

  if (result.scriptVerification) {
    logger.debug(scopedLog(LogContext.RECORDING, 'script verification'), {
      loaded: result.scriptVerification.loaded,
      ready: result.scriptVerification.ready,
      inMainContext: result.scriptVerification.inMainContext,
      handlersCount: result.scriptVerification.handlersCount,
      version: result.scriptVerification.version,
    });
  }

  if (result.injectionStats) {
    logger.debug(scopedLog(LogContext.RECORDING, 'injection stats'), {
      ...result.injectionStats,
    });
  }

  if (result.eventFlowTest) {
    logger.debug(scopedLog(LogContext.RECORDING, 'event flow test'), {
      ...result.eventFlowTest,
    });
  }
}

/**
 * Quick check if recording is ready.
 *
 * Convenience function for simple ready checks without full diagnostics.
 *
 * @param page - The Playwright page to check
 * @returns true if recording appears ready
 */
export async function isRecordingReady(page: Page): Promise<boolean> {
  try {
    const verification = await verifyScriptInjection(page);
    return verification.loaded && verification.ready && verification.inMainContext;
  } catch {
    return false;
  }
}

/**
 * Wait for recording to be ready with timeout.
 *
 * Polls until recording is ready or timeout is reached.
 *
 * @param page - The Playwright page to check
 * @param timeoutMs - Maximum wait time (default: 5000ms)
 * @param pollIntervalMs - Polling interval (default: 100ms)
 * @returns true if ready before timeout
 */
export async function waitForRecordingReady(
  page: Page,
  timeoutMs = 5000,
  pollIntervalMs = 100
): Promise<boolean> {
  const startTime = Date.now();

  while (Date.now() - startTime < timeoutMs) {
    if (await isRecordingReady(page)) {
      return true;
    }
    await new Promise((resolve) => setTimeout(resolve, pollIntervalMs));
  }

  return false;
}

/**
 * Generate a diagnostic report string for logging or display.
 *
 * @param result - Diagnostic result to format
 * @returns Formatted report string
 */
export function formatDiagnosticReport(result: RecordingDiagnosticResult): string {
  const lines: string[] = [];

  lines.push('='.repeat(60));
  lines.push('RECORDING DIAGNOSTIC REPORT');
  lines.push('='.repeat(60));
  lines.push('');
  lines.push(`Status: ${result.ready ? '✅ READY' : '❌ NOT READY'}`);
  lines.push(`Timestamp: ${result.timestamp}`);
  lines.push(`Duration: ${result.durationMs}ms`);
  lines.push(`Level: ${result.level}`);
  lines.push('');

  lines.push('Provider:');
  lines.push(`  Name: ${result.provider.name}`);
  lines.push(`  Evaluate Isolated: ${result.provider.evaluateIsolated}`);
  lines.push(`  ExposeBinding Isolated: ${result.provider.exposeBindingIsolated}`);
  lines.push('');

  if (result.scriptVerification) {
    lines.push('Script Verification:');
    lines.push(`  Loaded: ${result.scriptVerification.loaded}`);
    lines.push(`  Ready: ${result.scriptVerification.ready}`);
    lines.push(`  In Main Context: ${result.scriptVerification.inMainContext}`);
    lines.push(`  Handlers Count: ${result.scriptVerification.handlersCount}`);
    lines.push(`  Version: ${result.scriptVerification.version || 'N/A'}`);
    if (result.scriptVerification.error) {
      lines.push(`  Error: ${result.scriptVerification.error}`);
    }
    if (result.scriptVerification.initError) {
      lines.push(`  Init Error: ${result.scriptVerification.initError}`);
    }
    lines.push('');
  }

  if (result.injectionStats) {
    lines.push('Injection Stats:');
    lines.push(`  Attempted: ${result.injectionStats.attempted}`);
    lines.push(`  Successful: ${result.injectionStats.successful}`);
    lines.push(`  Failed: ${result.injectionStats.failed}`);
    lines.push(`  Skipped: ${result.injectionStats.skipped}`);
    lines.push('');
  }

  if (result.eventFlowTest) {
    lines.push('Event Flow Test:');
    lines.push(`  Passed: ${result.eventFlowTest.passed}`);
    lines.push(`  Event Sent: ${result.eventFlowTest.eventSent}`);
    lines.push(`  Event Received: ${result.eventFlowTest.eventReceived}`);
    if (result.eventFlowTest.latencyMs !== undefined) {
      lines.push(`  Latency: ${result.eventFlowTest.latencyMs}ms`);
    }
    if (result.eventFlowTest.error) {
      lines.push(`  Error: ${result.eventFlowTest.error}`);
    }
    lines.push('');
  }

  if (result.issues.length > 0) {
    lines.push('Issues:');
    for (const issue of result.issues) {
      const icon =
        issue.severity === DiagnosticSeverity.ERROR
          ? '❌'
          : issue.severity === DiagnosticSeverity.WARNING
            ? '⚠️'
            : 'ℹ️';
      lines.push(`  ${icon} [${issue.code}] ${issue.message}`);
      if (issue.suggestion) {
        lines.push(`     → ${issue.suggestion}`);
      }
    }
    lines.push('');
  } else {
    lines.push('Issues: None');
    lines.push('');
  }

  lines.push('='.repeat(60));

  return lines.join('\n');
}
