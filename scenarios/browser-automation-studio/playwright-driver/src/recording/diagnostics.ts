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
 * Status of a single diagnostic check.
 */
export type DiagnosticCheckStatus = 'passed' | 'failed' | 'warning' | 'skipped';

/**
 * A single diagnostic check that was performed.
 * Always reported, even when passing, to show what was validated.
 */
export interface DiagnosticCheck {
  /** Unique identifier for the check */
  id: string;
  /** Human-readable name of the check */
  name: string;
  /** Category for grouping (e.g., 'script', 'events', 'context') */
  category: 'script' | 'context' | 'events' | 'telemetry';
  /** Status of this check */
  status: DiagnosticCheckStatus;
  /** Brief description of what was checked */
  description: string;
  /** Value or result of the check (for display) */
  value?: string;
  /** Additional details if failed or warning */
  details?: string;
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
  /** All checks performed with their status (always present) */
  checks: DiagnosticCheck[];
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
  eventFlowTest?: EventFlowTestResult;
}

/**
 * Browser-side telemetry from recording script.
 */
export interface BrowserTelemetry {
  eventsDetected: number;
  eventsCaptured: number;
  eventsSent: number;
  eventsSendSuccess: number;
  eventsSendFailed: number;
  lastEventAt: number | null;
  lastEventType: string | null;
  lastError: string | null;
}

/**
 * Result of real click simulation test.
 */
export interface RealClickTestResult {
  /** Whether click simulation was attempted */
  attempted: boolean;
  /** Whether click event was detected by recording script */
  clickDetected: boolean;
  /** Whether event was sent via fetch */
  eventSent: boolean;
  /** Telemetry before click */
  telemetryBefore?: BrowserTelemetry;
  /** Telemetry after click */
  telemetryAfter?: BrowserTelemetry;
  /** Error if test failed */
  error?: string;
}

/**
 * Extended event flow test result with detailed diagnostics.
 */
export interface EventFlowTestResult {
  /** Whether the overall test passed */
  passed: boolean;
  /** Whether the console test event was sent */
  eventSent: boolean;
  /** Whether the console test event was received by Playwright */
  eventReceived: boolean;
  /** Round-trip latency for console event (ms) */
  latencyMs?: number;
  /** Error message if test failed */
  error?: string;

  // === Extended diagnostic info ===

  /** Page URL at time of test */
  pageUrl?: string;
  /** Whether page is a valid HTTP(S) page */
  pageValid?: boolean;

  /** Script status from CDP evaluation in MAIN context */
  scriptStatus?: {
    loaded: boolean;
    ready: boolean;
    inMainContext: boolean;
    handlersCount: number;
    version?: string | null;
  };

  /** Result of testing the actual fetch event path */
  fetchTest?: {
    sent: boolean;
    status: number | null;
    error: string | null;
  };

  /** Console capture statistics during test */
  consoleCapture?: {
    count: number;
    hasErrors: boolean;
    /** Sample of captured messages for debugging */
    samples?: Array<{ type: string; text: string }>;
  };

  /** Browser-side telemetry from recording script */
  browserTelemetry?: BrowserTelemetry;

  /** Result of real click simulation test */
  realClickTest?: RealClickTestResult;
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

  // Real click test issues
  CLICK_NOT_DETECTED: 'CLICK_NOT_DETECTED',
  CLICK_NOT_SENT: 'CLICK_NOT_SENT',
  CLICK_TEST_FAILED: 'CLICK_TEST_FAILED',

  // Configuration issues
  PROVIDER_MISMATCH: 'PROVIDER_MISMATCH',
  CDP_NOT_AVAILABLE: 'CDP_NOT_AVAILABLE',
} as const;

// =============================================================================
// Diagnostic Functions
// =============================================================================

/**
 * Build the checks array showing all validations performed.
 * This provides visibility into what was tested even when everything passes.
 */
function buildChecksArray(
  scriptVerification: InjectionVerification | undefined,
  injectionStats: InjectionStats | undefined,
  eventFlowTest: EventFlowTestResult | undefined,
  level: RecordingDiagnosticLevel
): DiagnosticCheck[] {
  const checks: DiagnosticCheck[] = [];

  // === Script Checks ===
  if (scriptVerification) {
    checks.push({
      id: 'script-loaded',
      name: 'Script Loaded',
      category: 'script',
      status: scriptVerification.loaded ? 'passed' : 'failed',
      description: 'Recording script is present on the page',
      value: scriptVerification.loaded ? 'Yes' : 'No',
      details: scriptVerification.error,
    });

    checks.push({
      id: 'script-ready',
      name: 'Script Ready',
      category: 'script',
      status: scriptVerification.ready ? 'passed' : scriptVerification.loaded ? 'failed' : 'skipped',
      description: 'Script initialized successfully',
      value: scriptVerification.ready ? 'Yes' : 'No',
      details: scriptVerification.initError || undefined,
    });

    checks.push({
      id: 'script-version',
      name: 'Script Version',
      category: 'script',
      status: scriptVerification.version ? 'passed' : 'warning',
      description: 'Recording script version',
      value: scriptVerification.version || 'Unknown',
    });

    checks.push({
      id: 'handlers-registered',
      name: 'Event Handlers',
      category: 'script',
      status: scriptVerification.handlersCount >= 7 ? 'passed' : scriptVerification.handlersCount > 0 ? 'warning' : 'failed',
      description: 'DOM event handlers registered',
      value: `${scriptVerification.handlersCount} handlers`,
      details: scriptVerification.handlersCount < 7 ? `Expected 7+, found ${scriptVerification.handlersCount}` : undefined,
    });
  }

  // === Context Checks ===
  if (scriptVerification) {
    checks.push({
      id: 'main-context',
      name: 'Main Context',
      category: 'context',
      status: scriptVerification.inMainContext ? 'passed' : 'failed',
      description: 'Script running in MAIN execution context',
      value: scriptVerification.inMainContext ? 'MAIN' : 'ISOLATED',
      details: !scriptVerification.inMainContext ? 'Script must run in MAIN context to capture History API events' : undefined,
    });
  }

  // === Injection Stats Checks ===
  if (injectionStats) {
    const successRate = injectionStats.attempted > 0
      ? ((injectionStats.successful / injectionStats.attempted) * 100).toFixed(0)
      : '0';

    checks.push({
      id: 'injection-attempts',
      name: 'Injection Attempts',
      category: 'script',
      status: injectionStats.attempted > 0 ? 'passed' : 'warning',
      description: 'HTML documents processed for script injection',
      value: `${injectionStats.attempted} attempts`,
      details: injectionStats.attempted === 0 ? 'No pages loaded yet' : undefined,
    });

    checks.push({
      id: 'injection-success',
      name: 'Injection Success Rate',
      category: 'script',
      status: parseFloat(successRate) >= 80 ? 'passed' : parseFloat(successRate) >= 50 ? 'warning' : 'failed',
      description: 'Percentage of successful script injections',
      value: `${successRate}% (${injectionStats.successful}/${injectionStats.attempted})`,
      details: parseFloat(successRate) < 80 ? 'Some pages may not have recording enabled' : undefined,
    });
  }

  // === Event Flow Checks (FULL level only) ===
  if (level === RecordingDiagnosticLevel.FULL && eventFlowTest) {
    // Page validity
    checks.push({
      id: 'page-valid',
      name: 'Page Type',
      category: 'context',
      status: eventFlowTest.pageValid ? 'passed' : 'failed',
      description: 'Page supports script injection (HTTP/HTTPS)',
      value: eventFlowTest.pageValid ? 'Valid' : 'Invalid',
      details: !eventFlowTest.pageValid ? 'about:blank and chrome:// pages cannot be recorded' : undefined,
    });

    // Fetch path test
    if (eventFlowTest.fetchTest) {
      checks.push({
        id: 'event-route',
        name: 'Event Route',
        category: 'events',
        status: eventFlowTest.fetchTest.sent && eventFlowTest.fetchTest.status === 200 ? 'passed' : 'failed',
        description: 'Event communication path (fetch to route handler)',
        value: eventFlowTest.fetchTest.status !== null ? `HTTP ${eventFlowTest.fetchTest.status}` : 'Failed',
        details: eventFlowTest.fetchTest.error || undefined,
      });
    }

    // Console capture
    if (eventFlowTest.consoleCapture) {
      checks.push({
        id: 'console-capture',
        name: 'Console Capture',
        category: 'events',
        status: eventFlowTest.eventReceived ? 'passed' : 'warning',
        description: 'Browser console events captured by Playwright',
        value: `${eventFlowTest.consoleCapture.count} messages`,
        details: eventFlowTest.consoleCapture.hasErrors ? 'Console errors detected during test' : undefined,
      });
    }

    // Browser telemetry
    if (eventFlowTest.browserTelemetry) {
      const tel = eventFlowTest.browserTelemetry;
      checks.push({
        id: 'telemetry-detected',
        name: 'Events Detected',
        category: 'telemetry',
        status: 'passed',
        description: 'DOM events detected by recording script',
        value: `${tel.eventsDetected} total`,
      });

      checks.push({
        id: 'telemetry-captured',
        name: 'Events Captured',
        category: 'telemetry',
        status: tel.eventsCaptured > 0 || tel.eventsDetected === 0 ? 'passed' : 'warning',
        description: 'Events passed isActive check',
        value: `${tel.eventsCaptured} captured`,
        details: tel.eventsDetected > 0 && tel.eventsCaptured === 0 ? 'Events detected but not captured - isActive may be false' : undefined,
      });

      checks.push({
        id: 'telemetry-sent',
        name: 'Events Sent',
        category: 'telemetry',
        status: tel.eventsSendFailed === 0 ? 'passed' : 'warning',
        description: 'Events sent via fetch',
        value: `${tel.eventsSent} sent, ${tel.eventsSendFailed} failed`,
        details: tel.lastError || undefined,
      });
    }

    // Real click test
    if (eventFlowTest.realClickTest) {
      const rct = eventFlowTest.realClickTest;
      if (rct.attempted) {
        checks.push({
          id: 'real-click-detected',
          name: 'Click Detection',
          category: 'events',
          status: rct.clickDetected ? 'passed' : 'failed',
          description: 'Simulated click detected by recording script',
          value: rct.clickDetected ? 'Detected' : 'Not detected',
          details: !rct.clickDetected ? 'DOM event handlers may not be firing' : undefined,
        });

        checks.push({
          id: 'real-click-sent',
          name: 'Click Event Sent',
          category: 'events',
          status: rct.eventSent ? 'passed' : rct.clickDetected ? 'failed' : 'skipped',
          description: 'Click event sent through fetch',
          value: rct.eventSent ? 'Sent' : rct.clickDetected ? 'Not sent' : 'Skipped',
          details: rct.clickDetected && !rct.eventSent ? 'Click detected but not sent - check isActive state' : undefined,
        });
      }
    }
  } else if (level !== RecordingDiagnosticLevel.FULL) {
    // Add placeholder for skipped checks
    checks.push({
      id: 'event-flow-test',
      name: 'Event Flow Test',
      category: 'events',
      status: 'skipped',
      description: 'Full event flow validation',
      value: 'Skipped',
      details: 'Run a FULL scan to test event flow',
    });
  }

  return checks;
}

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

  // Build checks array showing all validations performed
  const checks = buildChecksArray(scriptVerification, injectionStats, eventFlowTest, level);

  const result: RecordingDiagnosticResult = {
    ready,
    timestamp: new Date().toISOString(),
    durationMs,
    level,
    checks,
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
 * Query browser telemetry via CDP.
 */
async function queryBrowserTelemetry(page: Page): Promise<BrowserTelemetry | null> {
  try {
    const client = await page.context().newCDPSession(page);
    try {
      const { result } = await client.send('Runtime.evaluate', {
        expression: `JSON.stringify(window.__vrooli_recording_telemetry || null)`,
        returnByValue: true,
      });
      if (result.type === 'string' && result.value && result.value !== 'null') {
        return JSON.parse(result.value);
      }
    } finally {
      await client.detach().catch(() => {});
    }
  } catch {
    // Ignore errors
  }
  return null;
}

/**
 * Perform a real click simulation test using CDP Input.dispatchMouseEvent.
 * This tests the full event flow from DOM event → recording script → fetch.
 */
async function performRealClickTest(
  page: Page,
  logger: winston.Logger,
  logPrefix: string
): Promise<RealClickTestResult> {
  const result: RealClickTestResult = {
    attempted: false,
    clickDetected: false,
    eventSent: false,
  };

  try {
    // Get telemetry before click
    result.telemetryBefore = (await queryBrowserTelemetry(page)) || undefined;
    const beforeDetected = result.telemetryBefore?.eventsDetected ?? 0;
    const beforeSent = result.telemetryBefore?.eventsSent ?? 0;

    logger.debug(scopedLog(LogContext.RECORDING, `${logPrefix} real click test: before`), {
      eventsDetected: beforeDetected,
      eventsSent: beforeSent,
    });

    // Get viewport dimensions to click in the center
    const viewport = page.viewportSize();
    const x = viewport ? Math.floor(viewport.width / 2) : 100;
    const y = viewport ? Math.floor(viewport.height / 2) : 100;

    // Simulate a real click using CDP Input.dispatchMouseEvent
    const client = await page.context().newCDPSession(page);
    try {
      result.attempted = true;

      // Mouse down
      await client.send('Input.dispatchMouseEvent', {
        type: 'mousePressed',
        x,
        y,
        button: 'left',
        clickCount: 1,
      });

      // Mouse up
      await client.send('Input.dispatchMouseEvent', {
        type: 'mouseReleased',
        x,
        y,
        button: 'left',
        clickCount: 1,
      });

      logger.debug(scopedLog(LogContext.RECORDING, `${logPrefix} real click test: click sent`), { x, y });

      // Wait a bit for the event to be processed
      await new Promise((resolve) => setTimeout(resolve, 200));

      // Get telemetry after click
      result.telemetryAfter = (await queryBrowserTelemetry(page)) || undefined;
      const afterDetected = result.telemetryAfter?.eventsDetected ?? 0;
      const afterSent = result.telemetryAfter?.eventsSent ?? 0;

      // Check if click was detected
      result.clickDetected = afterDetected > beforeDetected;
      result.eventSent = afterSent > beforeSent;

      logger.debug(scopedLog(LogContext.RECORDING, `${logPrefix} real click test: after`), {
        eventsDetected: afterDetected,
        eventsSent: afterSent,
        clickDetected: result.clickDetected,
        eventSent: result.eventSent,
      });
    } finally {
      await client.detach().catch(() => {});
    }
  } catch (error) {
    result.error = error instanceof Error ? error.message : String(error);
    logger.debug(scopedLog(LogContext.RECORDING, `${logPrefix} real click test failed`), {
      error: result.error,
    });
  }

  return result;
}

/**
 * Test the event flow from browser to Node.js.
 *
 * This comprehensive test verifies:
 * 1. Page is a valid HTTP(S) page (not about:blank or chrome://)
 * 2. Recording script is loaded and ready in MAIN context (via CDP)
 * 3. The actual fetch event path (/__vrooli_recording_event__) works
 * 4. Console messages can be captured by Playwright
 *
 * Uses CDP for MAIN context access since rebrowser-playwright runs
 * page.evaluate() in ISOLATED context.
 */
async function testEventFlow(
  page: Page,
  _context: BrowserContext,
  timeoutMs: number,
  logger: winston.Logger
): Promise<EventFlowTestResult> {
  const testEventId = `diag-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
  const logPrefix = 'event flow test:';

  // === Step 1: Check page state ===
  let pageUrl: string;
  try {
    pageUrl = page.url();
  } catch (error) {
    logger.debug(scopedLog(LogContext.RECORDING, `${logPrefix} failed to get page URL`), {
      error: error instanceof Error ? error.message : String(error),
    });
    return {
      passed: false,
      eventSent: false,
      eventReceived: false,
      error: 'Failed to get page URL - page may be closed',
    };
  }

  const isBlankPage = pageUrl === 'about:blank' || pageUrl === '';
  const isChromePage = pageUrl.startsWith('chrome:') || pageUrl.startsWith('chrome-extension:');
  const isValidPage = pageUrl.startsWith('http://') || pageUrl.startsWith('https://');

  logger.debug(scopedLog(LogContext.RECORDING, `${logPrefix} page state`), {
    url: pageUrl.slice(0, 100),
    isBlankPage,
    isChromePage,
    isValidPage,
  });

  // Early return for pages where recording cannot work
  if (isBlankPage || isChromePage) {
    const reason = isBlankPage ? 'about:blank' : 'chrome:// URL';
    logger.debug(scopedLog(LogContext.RECORDING, `${logPrefix} skipping - invalid page type`), { reason });
    return {
      passed: false,
      eventSent: false,
      eventReceived: false,
      pageUrl,
      pageValid: false,
      error: `Page is ${reason} - recording script cannot be injected on these pages`,
    };
  }

  // === Step 2: Set up console capture for ALL messages ===
  const capturedConsole: Array<{ type: string; text: string }> = [];
  let eventReceived = false;
  let receiveTime = 0;

  const consoleHandler = (msg: import('rebrowser-playwright').ConsoleMessage) => {
    const text = msg.text();
    capturedConsole.push({ type: msg.type(), text: text.slice(0, 200) });
    if (text.includes(testEventId)) {
      receiveTime = Date.now();
      eventReceived = true;
    }
  };
  page.on('console', consoleHandler);

  logger.debug(scopedLog(LogContext.RECORDING, `${logPrefix} console listener attached`));

  // === Step 3: Use CDP to check recording globals in MAIN context ===
  // This fixes the bug where we checked for __VROOLI_RECORDER__ which doesn't exist
  let scriptStatus: EventFlowTestResult['scriptStatus'] | undefined;

  try {
    const client = await page.context().newCDPSession(page);
    try {
      const { result } = await client.send('Runtime.evaluate', {
        expression: `(function() {
          return JSON.stringify({
            loaded: window.__vrooli_recording_script_loaded === true,
            ready: window.__vrooli_recording_ready === true,
            inMainContext: window.__vrooli_recording_script_context === 'MAIN',
            handlersCount: window.__vrooli_recording_handlers_count || 0,
            version: window.__vrooli_recording_script_version || null
          });
        })()`,
        returnByValue: true,
      });

      if (result.type === 'string' && result.value) {
        scriptStatus = JSON.parse(result.value);
        logger.debug(scopedLog(LogContext.RECORDING, `${logPrefix} CDP script status`), scriptStatus);
      }
    } finally {
      await client.detach().catch(() => {});
    }
  } catch (error) {
    logger.debug(scopedLog(LogContext.RECORDING, `${logPrefix} CDP script check failed`), {
      error: error instanceof Error ? error.message : String(error),
    });
  }

  // === Step 4: Test the actual recording fetch event path via CDP ===
  let fetchTest: EventFlowTestResult['fetchTest'] | undefined;

  try {
    const client = await page.context().newCDPSession(page);
    try {
      const { result } = await client.send('Runtime.evaluate', {
        expression: `(async function() {
          try {
            const response = await fetch('/__vrooli_recording_event__', {
              method: 'POST',
              headers: { 'Content-Type': 'application/json' },
              body: JSON.stringify({
                actionType: 'diagnostic-test',
                timestamp: Date.now(),
                testId: '${testEventId}',
                url: window.location.href
              }),
              keepalive: true,
            });
            return JSON.stringify({ sent: true, status: response.status, error: null });
          } catch (e) {
            return JSON.stringify({ sent: false, status: null, error: e.message });
          }
        })()`,
        returnByValue: true,
        awaitPromise: true,
      });

      if (result.type === 'string' && result.value) {
        fetchTest = JSON.parse(result.value);
        logger.debug(scopedLog(LogContext.RECORDING, `${logPrefix} fetch path test result`), fetchTest);
      }
    } finally {
      await client.detach().catch(() => {});
    }
  } catch (error) {
    logger.debug(scopedLog(LogContext.RECORDING, `${logPrefix} fetch path test failed`), {
      error: error instanceof Error ? error.message : String(error),
    });
    fetchTest = {
      sent: false,
      status: null,
      error: error instanceof Error ? error.message : String(error),
    };
  }

  // === Step 5: Send console test event via CDP in MAIN context ===
  const sendTime = Date.now();
  let eventSent = false;

  try {
    const client = await page.context().newCDPSession(page);
    try {
      await client.send('Runtime.evaluate', {
        expression: `console.log('[DiagnosticTest] ${testEventId}')`,
      });
      eventSent = true;
      logger.debug(scopedLog(LogContext.RECORDING, `${logPrefix} console test event sent via CDP`));
    } finally {
      await client.detach().catch(() => {});
    }
  } catch (error) {
    logger.debug(scopedLog(LogContext.RECORDING, `${logPrefix} console send via CDP failed`), {
      error: error instanceof Error ? error.message : String(error),
    });
  }

  // === Step 6: Wait for console message or timeout ===
  if (eventSent && !eventReceived) {
    const waitStart = Date.now();
    const pollInterval = 50;

    while (Date.now() - waitStart < timeoutMs && !eventReceived) {
      await new Promise((resolve) => setTimeout(resolve, pollInterval));
    }

    logger.debug(scopedLog(LogContext.RECORDING, `${logPrefix} console wait complete`), {
      eventReceived,
      waitedMs: Date.now() - waitStart,
      capturedCount: capturedConsole.length,
    });
  }

  // === Step 7: Cleanup console listener ===
  page.off('console', consoleHandler);

  // === Step 8: Query browser telemetry ===
  let browserTelemetry: BrowserTelemetry | undefined;
  try {
    const client = await page.context().newCDPSession(page);
    try {
      const { result } = await client.send('Runtime.evaluate', {
        expression: `JSON.stringify(window.__vrooli_recording_telemetry || null)`,
        returnByValue: true,
      });
      if (result.type === 'string' && result.value && result.value !== 'null') {
        browserTelemetry = JSON.parse(result.value);
        logger.debug(scopedLog(LogContext.RECORDING, `${logPrefix} browser telemetry`), browserTelemetry);
      }
    } finally {
      await client.detach().catch(() => {});
    }
  } catch (error) {
    logger.debug(scopedLog(LogContext.RECORDING, `${logPrefix} browser telemetry query failed`), {
      error: error instanceof Error ? error.message : String(error),
    });
  }

  // === Step 9: Real click simulation test ===
  let realClickTest: RealClickTestResult | undefined;
  if (scriptStatus?.ready && scriptStatus?.inMainContext) {
    realClickTest = await performRealClickTest(page, logger, logPrefix);
  }

  // === Step 10: Analyze results and determine pass/fail ===
  const latencyMs = eventReceived ? receiveTime - sendTime : undefined;

  // Script is ready if loaded, ready flag set, and running in MAIN context
  const scriptReady = scriptStatus?.loaded && scriptStatus?.ready && scriptStatus?.inMainContext;

  // Fetch works if we got a 200 response from the route handler
  const fetchWorks = fetchTest?.sent && fetchTest?.status === 200;

  // Pass if EITHER script is ready OR fetch path works
  // (Script status is the authoritative check, fetch is the actual event path test)
  const passed = scriptReady || fetchWorks;

  // Build error message for failures
  let errorMessage: string | undefined;
  if (!passed) {
    const reasons: string[] = [];
    if (!scriptStatus) {
      reasons.push('CDP script check failed');
    } else if (!scriptStatus.loaded) {
      reasons.push('script not loaded');
    } else if (!scriptStatus.ready) {
      reasons.push('script not ready');
    } else if (!scriptStatus.inMainContext) {
      reasons.push('script not in MAIN context');
    }

    if (!fetchTest) {
      reasons.push('fetch test failed to run');
    } else if (!fetchTest.sent) {
      reasons.push(`fetch failed: ${fetchTest.error}`);
    } else if (fetchTest.status !== 200) {
      reasons.push(`fetch returned ${fetchTest.status}`);
    }

    errorMessage = reasons.join('; ');
  }

  // Log final result
  logger.debug(scopedLog(LogContext.RECORDING, `${logPrefix} final result`), {
    passed,
    scriptReady,
    fetchWorks,
    eventSent,
    eventReceived,
    capturedConsoleCount: capturedConsole.length,
    hasConsoleErrors: capturedConsole.some((c) => c.type === 'error'),
  });

  // Build console capture info with samples for debugging
  const consoleCapture: EventFlowTestResult['consoleCapture'] = capturedConsole.length > 0
    ? {
        count: capturedConsole.length,
        hasErrors: capturedConsole.some((c) => c.type === 'error'),
        samples: capturedConsole.slice(0, 5), // First 5 messages for debugging
      }
    : undefined;

  return {
    passed,
    eventSent,
    eventReceived,
    latencyMs,
    error: errorMessage,
    pageUrl,
    pageValid: isValidPage,
    scriptStatus,
    fetchTest,
    consoleCapture,
    browserTelemetry,
    realClickTest,
  };
}

/**
 * Check event flow test results for issues.
 * Provides detailed diagnostics based on the comprehensive test results.
 */
function checkEventFlowTest(
  test: RecordingDiagnosticResult['eventFlowTest'],
  issues: DiagnosticIssue[]
): void {
  if (!test) return;

  // === Check 1: Invalid page type ===
  if (test.pageValid === false) {
    issues.push({
      code: DIAGNOSTIC_CODES.EVENT_SEND_FAILED,
      message: `Page type not supported for recording: ${test.pageUrl?.slice(0, 50) || 'unknown'}`,
      severity: DiagnosticSeverity.ERROR,
      suggestion:
        'Navigate to an HTTP or HTTPS page to enable recording. ' +
        'Pages like about:blank, chrome://, or chrome-extension:// URLs do not support script injection.',
      details: { pageUrl: test.pageUrl },
    });
    return;
  }

  // === Check 2: Script not loaded ===
  if (test.scriptStatus && !test.scriptStatus.loaded) {
    issues.push({
      code: DIAGNOSTIC_CODES.SCRIPT_NOT_LOADED,
      message: 'Recording script not loaded on page',
      severity: DiagnosticSeverity.ERROR,
      suggestion:
        'The script injection via route interception may have failed. ' +
        'Check that the page was loaded through normal navigation (not dynamically injected). ' +
        'Service workers or CSP headers might be blocking injection.',
      details: {
        pageUrl: test.pageUrl,
        scriptStatus: test.scriptStatus,
        fetchTest: test.fetchTest,
      },
    });
    return;
  }

  // === Check 3: Script loaded but not ready ===
  if (test.scriptStatus && test.scriptStatus.loaded && !test.scriptStatus.ready) {
    issues.push({
      code: DIAGNOSTIC_CODES.SCRIPT_NOT_READY,
      message: 'Recording script loaded but failed to initialize',
      severity: DiagnosticSeverity.ERROR,
      suggestion:
        'The script started executing but crashed during initialization. ' +
        'Check browser console for JavaScript errors. ' +
        'A conflicting script on the page may have interfered.',
      details: {
        handlersCount: test.scriptStatus.handlersCount,
        version: test.scriptStatus.version,
        consoleCapture: test.consoleCapture,
      },
    });
    return;
  }

  // === Check 4: Script not in MAIN context ===
  if (test.scriptStatus && test.scriptStatus.loaded && !test.scriptStatus.inMainContext) {
    issues.push({
      code: DIAGNOSTIC_CODES.SCRIPT_WRONG_CONTEXT,
      message: 'Recording script running in wrong execution context',
      severity: DiagnosticSeverity.ERROR,
      suggestion:
        'The script is not in MAIN context. This means History API navigation events ' +
        'will not be captured. The script should be injected via HTML route interception, ' +
        'not via page.evaluate() or addInitScript().',
      details: {
        scriptStatus: test.scriptStatus,
      },
    });
    return;
  }

  // === Check 5: Fetch path test failed ===
  if (test.fetchTest && !test.fetchTest.sent) {
    issues.push({
      code: DIAGNOSTIC_CODES.EVENT_SEND_FAILED,
      message: 'Event communication path test failed',
      severity: DiagnosticSeverity.ERROR,
      suggestion:
        'The fetch to /__vrooli_recording_event__ failed. ' +
        'The route handler may not be set up, or CORS/CSP policies may be blocking the request.',
      details: {
        fetchError: test.fetchTest.error,
        pageUrl: test.pageUrl,
      },
    });
    return;
  }

  // === Check 6: Fetch returned non-200 ===
  if (test.fetchTest && test.fetchTest.sent && test.fetchTest.status !== 200) {
    issues.push({
      code: DIAGNOSTIC_CODES.EVENT_SEND_FAILED,
      message: `Event route returned status ${test.fetchTest.status}`,
      severity: DiagnosticSeverity.WARNING,
      suggestion:
        'The route handler responded but with an error status. ' +
        'Check playwright-driver logs for route handler errors.',
      details: {
        status: test.fetchTest.status,
      },
    });
  }

  // === If test passed, check for warnings ===
  if (test.passed) {
    // Warn about high latency
    if (test.latencyMs && test.latencyMs > 1000) {
      issues.push({
        code: DIAGNOSTIC_CODES.EVENT_HIGH_LATENCY,
        message: `High event latency: ${test.latencyMs}ms`,
        severity: DiagnosticSeverity.WARNING,
        suggestion: 'Events may be delayed. Check route handler performance.',
        details: { latencyMs: test.latencyMs },
      });
    }

    // Warn about console errors during test
    if (test.consoleCapture?.hasErrors) {
      issues.push({
        code: DIAGNOSTIC_CODES.SCRIPT_INIT_ERROR,
        message: 'Console errors detected during diagnostic test',
        severity: DiagnosticSeverity.WARNING,
        suggestion: 'JavaScript errors occurred during the test. Check browser console for details.',
        details: {
          consoleCount: test.consoleCapture.count,
          samples: test.consoleCapture.samples,
        },
      });
    }

    // Warn about low handler count
    if (test.scriptStatus && test.scriptStatus.handlersCount < 7) {
      issues.push({
        code: DIAGNOSTIC_CODES.SCRIPT_LOW_HANDLERS,
        message: `Low handler count: ${test.scriptStatus.handlersCount} (expected 7+)`,
        severity: DiagnosticSeverity.WARNING,
        suggestion: 'Some event types may not be captured. Script may have partially initialized.',
        details: { handlersCount: test.scriptStatus.handlersCount },
      });
    }

    // === CRITICAL: Check real click test results ===
    // This is the definitive test of whether DOM events are being captured
    if (test.realClickTest) {
      if (test.realClickTest.error) {
        issues.push({
          code: DIAGNOSTIC_CODES.CLICK_TEST_FAILED,
          message: `Real click simulation test failed: ${test.realClickTest.error}`,
          severity: DiagnosticSeverity.WARNING,
          suggestion: 'CDP Input.dispatchMouseEvent may not be supported or the page may have blocked the event.',
          details: { error: test.realClickTest.error },
        });
      } else if (test.realClickTest.attempted && !test.realClickTest.clickDetected) {
        // This is the CRITICAL case - click was simulated but not detected
        issues.push({
          code: DIAGNOSTIC_CODES.CLICK_NOT_DETECTED,
          message: 'CRITICAL: Real click was simulated but NOT detected by recording script',
          severity: DiagnosticSeverity.ERROR,
          suggestion:
            'DOM event handlers are not firing. This is the root cause of recording not working. ' +
            'Possible causes: (1) Event listeners were removed by page scripts, ' +
            '(2) Event propagation is being stopped, ' +
            '(3) The click target is in a Shadow DOM or iframe, ' +
            '(4) The recording script was replaced after initialization.',
          details: {
            telemetryBefore: test.realClickTest.telemetryBefore,
            telemetryAfter: test.realClickTest.telemetryAfter,
          },
        });
      } else if (test.realClickTest.clickDetected && !test.realClickTest.eventSent) {
        // Click detected but not sent - isActive might be false
        issues.push({
          code: DIAGNOSTIC_CODES.CLICK_NOT_SENT,
          message: 'Click was detected but event was NOT sent',
          severity: DiagnosticSeverity.ERROR,
          suggestion:
            'The recording script detected the click but did not send it. ' +
            'This usually means isActive is false (recording not started) ' +
            'or captureAction is failing before sendEvent is called.',
          details: {
            telemetryBefore: test.realClickTest.telemetryBefore,
            telemetryAfter: test.realClickTest.telemetryAfter,
          },
        });
      }
    }

    return;
  }

  // === Fallback: Test failed but no specific reason identified ===
  issues.push({
    code: DIAGNOSTIC_CODES.EVENT_NOT_RECEIVED,
    message: 'Recording subsystem connectivity test failed',
    severity: DiagnosticSeverity.ERROR,
    suggestion: test.error || 'Unknown failure. Check the diagnostic details for more information.',
    details: {
      error: test.error,
      pageUrl: test.pageUrl,
      scriptStatus: test.scriptStatus,
      fetchTest: test.fetchTest,
      consoleCapture: test.consoleCapture,
    },
  });
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
    const eft = result.eventFlowTest;
    logger.debug(scopedLog(LogContext.RECORDING, 'event flow test summary'), {
      passed: eft.passed,
      eventSent: eft.eventSent,
      eventReceived: eft.eventReceived,
      latencyMs: eft.latencyMs,
      error: eft.error,
      pageUrl: eft.pageUrl?.slice(0, 80),
      pageValid: eft.pageValid,
    });

    if (eft.scriptStatus) {
      logger.debug(scopedLog(LogContext.RECORDING, 'event flow: script status (CDP)'), {
        loaded: eft.scriptStatus.loaded,
        ready: eft.scriptStatus.ready,
        inMainContext: eft.scriptStatus.inMainContext,
        handlersCount: eft.scriptStatus.handlersCount,
        version: eft.scriptStatus.version,
      });
    }

    if (eft.fetchTest) {
      logger.debug(scopedLog(LogContext.RECORDING, 'event flow: fetch path test'), {
        sent: eft.fetchTest.sent,
        status: eft.fetchTest.status,
        error: eft.fetchTest.error,
      });
    }

    if (eft.consoleCapture) {
      logger.debug(scopedLog(LogContext.RECORDING, 'event flow: console capture'), {
        count: eft.consoleCapture.count,
        hasErrors: eft.consoleCapture.hasErrors,
        samples: eft.consoleCapture.samples?.map((s) => `[${s.type}] ${s.text.slice(0, 50)}`),
      });
    }
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
