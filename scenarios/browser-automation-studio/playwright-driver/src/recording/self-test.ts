/**
 * Recording Self-Test Module
 *
 * Provides automated end-to-end testing of the recording pipeline without
 * requiring human interaction. This is the key to debugging recording issues
 * without a human-in-the-loop.
 *
 * ## How It Works
 *
 * 1. Navigates to a special test page with interactive elements
 * 2. Starts a temporary recording session (if not already recording)
 * 3. Uses CDP to simulate REAL user interactions (not Playwright's page.click)
 * 4. Verifies events flow through the entire pipeline:
 *    - Browser script detects events
 *    - Events are sent via fetch
 *    - Route handler receives events
 *    - Event handler processes events
 * 5. Reports detailed diagnostics at each step
 *
 * ## Usage
 *
 * ```typescript
 * import { runRecordingPipelineTest } from './self-test';
 *
 * const result = await runRecordingPipelineTest(session, sessionManager);
 * if (!result.success) {
 *   console.error('Pipeline test failed:', result.failurePoint);
 *   console.log('Diagnostics:', result.diagnostics);
 * }
 * ```
 */

import type { Page, BrowserContext } from 'rebrowser-playwright';
import type { RecordingContextInitializer } from './context-initializer';
import { logger, scopedLog, LogContext } from '../utils';
import { verifyScriptInjection } from './verification';

// =============================================================================
// Types
// =============================================================================

/**
 * Result of the recording pipeline self-test.
 */
export interface PipelineTestResult {
  /** Whether the entire pipeline test passed */
  success: boolean;
  /** Timestamp of the test */
  timestamp: string;
  /** Total duration of the test in ms */
  durationMs: number;
  /** Where the test failed (if it failed) */
  failurePoint?: PipelineFailurePoint;
  /** Detailed message about the failure */
  failureMessage?: string;
  /** Suggestions for fixing the issue */
  suggestions?: string[];

  /** Individual step results */
  steps: PipelineStepResult[];

  /** Diagnostics collected during the test */
  diagnostics: PipelineTestDiagnostics;
}

/**
 * Where in the pipeline the test failed.
 */
export type PipelineFailurePoint =
  | 'navigation'
  | 'script_injection'
  | 'script_initialization'
  | 'event_detection'
  | 'event_capture'
  | 'event_send'
  | 'route_receive'
  | 'handler_process'
  | 'unknown';

/**
 * Result of a single pipeline step.
 */
export interface PipelineStepResult {
  /** Step name */
  name: string;
  /** Whether this step passed */
  passed: boolean;
  /** Duration of this step in ms */
  durationMs: number;
  /** Error message if failed */
  error?: string;
  /** Additional details */
  details?: Record<string, unknown>;
}

/**
 * Diagnostics collected during the pipeline test.
 */
export interface PipelineTestDiagnostics {
  /** URL navigated to */
  testPageUrl: string;
  /** Whether the test page was injected successfully */
  testPageInjected: boolean;

  /** Script verification before test */
  scriptStatusBefore?: ScriptStatus;
  /** Script verification after test */
  scriptStatusAfter?: ScriptStatus;

  /** Browser telemetry before interactions */
  telemetryBefore?: BrowserTelemetry;
  /** Browser telemetry after interactions */
  telemetryAfter?: BrowserTelemetry;

  /** Route handler stats before test */
  routeStatsBefore?: RouteStats;
  /** Route handler stats after test */
  routeStatsAfter?: RouteStats;

  /** Events captured during the test */
  eventsCaptured: CapturedEvent[];

  /** Console messages captured during the test */
  consoleMessages: ConsoleMessage[];
}

interface ScriptStatus {
  loaded: boolean;
  ready: boolean;
  inMainContext: boolean;
  handlersCount: number;
  version: string | null;
  isActive: boolean | null;
}

interface BrowserTelemetry {
  eventsDetected: number;
  eventsCaptured: number;
  eventsSent: number;
  eventsSendSuccess: number;
  eventsSendFailed: number;
  lastError: string | null;
}

interface RouteStats {
  eventsReceived: number;
  eventsProcessed: number;
  eventsDroppedNoHandler: number;
  eventsWithErrors: number;
}

interface CapturedEvent {
  actionType: string;
  timestamp: string;
  selector?: string;
}

interface ConsoleMessage {
  type: string;
  text: string;
}

// =============================================================================
// Test Page HTML
// =============================================================================

/**
 * HTML for the self-test page.
 * This page contains various interactive elements for testing.
 */
export const TEST_PAGE_HTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Recording Pipeline Test Page</title>
  <style>
    * { box-sizing: border-box; margin: 0; padding: 0; }
    body {
      font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
      padding: 40px;
      background: #f5f5f5;
      min-height: 100vh;
    }
    .container {
      max-width: 600px;
      margin: 0 auto;
      background: white;
      padding: 40px;
      border-radius: 12px;
      box-shadow: 0 2px 8px rgba(0,0,0,0.1);
    }
    h1 {
      color: #333;
      margin-bottom: 8px;
      font-size: 24px;
    }
    .subtitle {
      color: #666;
      margin-bottom: 32px;
      font-size: 14px;
    }
    .test-section {
      margin-bottom: 24px;
      padding: 16px;
      background: #fafafa;
      border-radius: 8px;
      border: 1px solid #eee;
    }
    .test-section h2 {
      font-size: 14px;
      color: #555;
      margin-bottom: 12px;
      text-transform: uppercase;
      letter-spacing: 0.5px;
    }
    button {
      padding: 12px 24px;
      font-size: 16px;
      cursor: pointer;
      border: none;
      border-radius: 6px;
      transition: all 0.2s;
    }
    #test-click-button {
      background: #4CAF50;
      color: white;
      width: 100%;
    }
    #test-click-button:hover {
      background: #45a049;
    }
    #test-click-button:active {
      transform: scale(0.98);
    }
    #test-input {
      width: 100%;
      padding: 12px;
      font-size: 16px;
      border: 2px solid #ddd;
      border-radius: 6px;
      outline: none;
      transition: border-color 0.2s;
    }
    #test-input:focus {
      border-color: #4CAF50;
    }
    #test-link {
      display: inline-block;
      color: #2196F3;
      text-decoration: none;
      font-size: 16px;
      padding: 8px 0;
    }
    #test-link:hover {
      text-decoration: underline;
    }
    .status {
      margin-top: 32px;
      padding: 16px;
      background: #e8f5e9;
      border-radius: 8px;
      font-size: 14px;
      color: #2e7d32;
    }
    .status-indicator {
      display: inline-block;
      width: 8px;
      height: 8px;
      background: #4CAF50;
      border-radius: 50%;
      margin-right: 8px;
      animation: pulse 2s infinite;
    }
    @keyframes pulse {
      0%, 100% { opacity: 1; }
      50% { opacity: 0.5; }
    }
    #event-log {
      margin-top: 16px;
      padding: 12px;
      background: #333;
      color: #0f0;
      font-family: monospace;
      font-size: 12px;
      border-radius: 4px;
      max-height: 150px;
      overflow-y: auto;
    }
    .log-entry {
      padding: 2px 0;
    }
  </style>
</head>
<body>
  <div class="container">
    <h1>Recording Pipeline Test</h1>
    <p class="subtitle">This page is used to test the recording pipeline automatically</p>

    <div class="test-section">
      <h2>Click Test</h2>
      <button id="test-click-button" data-testid="test-click-button">
        Click Me to Test Recording
      </button>
    </div>

    <div class="test-section">
      <h2>Input Test</h2>
      <input
        type="text"
        id="test-input"
        data-testid="test-input"
        placeholder="Type here to test input recording..."
      />
    </div>

    <div class="test-section">
      <h2>Navigation Test</h2>
      <a href="#test-anchor" id="test-link" data-testid="test-link">
        Click this link to test navigation
      </a>
    </div>

    <div class="status">
      <span class="status-indicator"></span>
      <strong>Ready for Testing</strong>
      <p style="margin-top: 8px; color: #555;">
        The recording script should detect interactions on this page.
      </p>
      <div id="event-log">
        <div class="log-entry">[Waiting for events...]</div>
      </div>
    </div>
  </div>

  <script>
    // Log events to the event log div for visual debugging
    const eventLog = document.getElementById('event-log');
    let eventCount = 0;

    function logEvent(type, target) {
      eventCount++;
      const entry = document.createElement('div');
      entry.className = 'log-entry';
      entry.textContent = '[' + new Date().toISOString().slice(11, 23) + '] ' + type + ': ' + target;
      eventLog.insertBefore(entry, eventLog.firstChild);
      if (eventLog.children.length > 20) {
        eventLog.removeChild(eventLog.lastChild);
      }
    }

    // Add listeners to log events (separate from recording script)
    document.getElementById('test-click-button').addEventListener('click', function() {
      logEvent('CLICK', 'test-click-button');
    });

    document.getElementById('test-input').addEventListener('input', function(e) {
      logEvent('INPUT', 'test-input: "' + e.target.value.slice(-10) + '"');
    });

    document.getElementById('test-link').addEventListener('click', function(e) {
      e.preventDefault(); // Don't actually navigate
      logEvent('CLICK', 'test-link');
    });
  </script>
</body>
</html>`;

/**
 * URL pattern for the test page.
 * The route handler will serve TEST_PAGE_HTML for this URL.
 */
export const TEST_PAGE_URL = '/__vrooli_recording_test__';

// =============================================================================
// Pipeline Test Implementation
// =============================================================================

/**
 * Run a complete recording pipeline test.
 *
 * This test:
 * 1. Navigates to the test page
 * 2. Verifies script injection
 * 3. Simulates real user interactions using CDP
 * 4. Verifies events flow through the entire pipeline
 * 5. Reports detailed diagnostics
 *
 * @param page - The Playwright page to test on
 * @param context - The browser context
 * @param contextInitializer - The recording context initializer
 * @param options - Test options
 */
export async function runRecordingPipelineTest(
  page: Page,
  _context: BrowserContext,
  contextInitializer: RecordingContextInitializer,
  options: {
    /** Whether to start a temporary recording if not already recording */
    autoStartRecording?: boolean;
    /** Timeout for the test in ms */
    timeoutMs?: number;
    /** Whether to capture console messages */
    captureConsole?: boolean;
  } = {}
): Promise<PipelineTestResult> {
  const {
    timeoutMs = 30000,
    captureConsole = true,
  } = options;

  const startTime = Date.now();
  const steps: PipelineStepResult[] = [];
  const eventsCaptured: CapturedEvent[] = [];
  const consoleMessages: ConsoleMessage[] = [];

  // Initialize diagnostics
  const diagnostics: PipelineTestDiagnostics = {
    testPageUrl: '',
    testPageInjected: false,
    eventsCaptured,
    consoleMessages,
  };

  // Helper to add step result
  const addStep = (name: string, passed: boolean, durationMs: number, error?: string, details?: Record<string, unknown>) => {
    steps.push({ name, passed, durationMs, error, details });
    logger.debug(scopedLog(LogContext.RECORDING, `pipeline test step: ${name}`), {
      passed,
      durationMs,
      error,
      ...details,
    });
  };

  // Set up console capture
  const consoleHandler = captureConsole ? (msg: { type: () => string; text: () => string }) => {
    consoleMessages.push({
      type: msg.type(),
      text: msg.text().slice(0, 500),
    });
  } : null;

  if (consoleHandler) {
    page.on('console', consoleHandler);
  }

  // Set up event capture (temporary handler to track received events)
  const originalHandler = contextInitializer.hasEventHandler();
  let receivedEvents: Array<{ actionType: string; timestamp: number }> = [];

  // Create a wrapper handler that tracks events
  const testEventHandler = (event: { actionType: string; url?: string; selectors?: unknown }) => {
    receivedEvents.push({
      actionType: event.actionType,
      timestamp: Date.now(),
    });
    eventsCaptured.push({
      actionType: event.actionType,
      timestamp: new Date().toISOString(),
      selector: typeof event.selectors === 'object' ? JSON.stringify(event.selectors).slice(0, 100) : undefined,
    });
  };

  // Set our test handler
  contextInitializer.setEventHandler(testEventHandler);

  try {
    // Get initial stats
    diagnostics.routeStatsBefore = contextInitializer.getRouteHandlerStats();

    // Step 1: Navigate to test page
    const navStart = Date.now();
    try {
      // Build the test page URL - we'll intercept this in the route
      const testPageFullUrl = `http://localhost${TEST_PAGE_URL}`;
      diagnostics.testPageUrl = testPageFullUrl;

      await page.goto(testPageFullUrl, { timeout: timeoutMs, waitUntil: 'domcontentloaded' });
      addStep('navigate_to_test_page', true, Date.now() - navStart);
    } catch (error) {
      addStep('navigate_to_test_page', false, Date.now() - navStart, error instanceof Error ? error.message : String(error));
      return buildResult(false, 'navigation', 'Failed to navigate to test page', steps, diagnostics, startTime);
    }

    // Wait for page to stabilize
    await sleep(500);

    // Step 2: Verify script injection via CDP
    const verifyStart = Date.now();
    try {
      const verification = await verifyScriptInjection(page);
      diagnostics.scriptStatusBefore = {
        loaded: verification.loaded,
        ready: verification.ready,
        inMainContext: verification.inMainContext,
        handlersCount: verification.handlersCount,
        version: verification.version,
        isActive: null, // Will query separately
      };

      if (!verification.loaded) {
        addStep('verify_script_injection', false, Date.now() - verifyStart, 'Script not loaded');
        return buildResult(false, 'script_injection', 'Recording script was not injected into the page', steps, diagnostics, startTime, [
          'Check that HTML route interception is working',
          'Verify the test page URL is being served correctly',
          'Check for CSP headers blocking script injection',
        ]);
      }

      if (!verification.ready) {
        addStep('verify_script_injection', false, Date.now() - verifyStart, 'Script not ready', { initError: verification.initError });
        return buildResult(false, 'script_initialization', 'Recording script loaded but failed to initialize', steps, diagnostics, startTime, [
          'Check browser console for JavaScript errors',
          'Verify the recording script has no syntax errors',
        ]);
      }

      if (!verification.inMainContext) {
        addStep('verify_script_injection', false, Date.now() - verifyStart, 'Script in wrong context');
        return buildResult(false, 'script_initialization', 'Recording script is running in ISOLATED context instead of MAIN', steps, diagnostics, startTime, [
          'Script must be injected via HTML, not page.evaluate()',
          'Check that HTML injection route is working correctly',
        ]);
      }

      diagnostics.testPageInjected = true;
      addStep('verify_script_injection', true, Date.now() - verifyStart, undefined, {
        handlersCount: verification.handlersCount,
        version: verification.version,
      });
    } catch (error) {
      addStep('verify_script_injection', false, Date.now() - verifyStart, error instanceof Error ? error.message : String(error));
      return buildResult(false, 'script_injection', 'Failed to verify script injection', steps, diagnostics, startTime);
    }

    // Step 3: Query browser telemetry before interactions
    const telemetryBeforeStart = Date.now();
    try {
      diagnostics.telemetryBefore = await queryBrowserTelemetry(page);
      diagnostics.scriptStatusBefore!.isActive = await queryIsActive(page);
      addStep('query_telemetry_before', true, Date.now() - telemetryBeforeStart, undefined, {
        telemetry: diagnostics.telemetryBefore,
        isActive: diagnostics.scriptStatusBefore!.isActive,
      });
    } catch (error) {
      addStep('query_telemetry_before', false, Date.now() - telemetryBeforeStart, error instanceof Error ? error.message : String(error));
      // Non-fatal, continue
    }

    // Step 4: Simulate a real click using CDP
    const clickStart = Date.now();
    receivedEvents = []; // Reset for tracking
    try {
      await simulateRealClick(page, '#test-click-button');
      await sleep(500); // Wait for event to propagate

      const clickReceived = receivedEvents.some(e => e.actionType === 'click');
      addStep('simulate_click', clickReceived, Date.now() - clickStart, clickReceived ? undefined : 'Click event not received', {
        eventsReceived: receivedEvents.length,
        eventTypes: receivedEvents.map(e => e.actionType),
      });

      if (!clickReceived) {
        // Query telemetry to understand where it failed
        const telemetryAfterClick = await queryBrowserTelemetry(page);
        diagnostics.telemetryAfter = telemetryAfterClick;

        if (telemetryAfterClick && diagnostics.telemetryBefore) {
          const detected = telemetryAfterClick.eventsDetected - diagnostics.telemetryBefore.eventsDetected;
          const captured = telemetryAfterClick.eventsCaptured - diagnostics.telemetryBefore.eventsCaptured;
          const sent = telemetryAfterClick.eventsSent - diagnostics.telemetryBefore.eventsSent;

          if (detected === 0) {
            return buildResult(false, 'event_detection', 'Click was simulated but NOT detected by recording script', steps, diagnostics, startTime, [
              'DOM event handlers may have been removed',
              'Event propagation might be stopped by page scripts',
              'The click target might be in a Shadow DOM or iframe',
            ]);
          }

          if (captured === 0) {
            return buildResult(false, 'event_capture', 'Click was detected but NOT captured (isActive may be false)', steps, diagnostics, startTime, [
              'Check if isActive flag is false in the recording script',
              'Recording may not have been properly started',
            ]);
          }

          if (sent === 0) {
            return buildResult(false, 'event_send', 'Click was captured but NOT sent via fetch', steps, diagnostics, startTime, [
              'The sendEvent function may be failing',
              'Check browser console for fetch errors',
            ]);
          }

          // Sent but not received by route handler
          const routeStatsAfter = contextInitializer.getRouteHandlerStats();
          const receivedByRoute = routeStatsAfter.eventsReceived - (diagnostics.routeStatsBefore?.eventsReceived || 0);

          if (receivedByRoute === 0) {
            return buildResult(false, 'route_receive', 'Click was sent but NOT received by route handler', steps, diagnostics, startTime, [
              'Service worker may be intercepting the request',
              'Route interception may not be working',
              'Check that the event URL pattern matches',
            ]);
          }

          return buildResult(false, 'handler_process', 'Click was received by route but NOT processed by handler', steps, diagnostics, startTime, [
            'Event handler may not be set',
            'Handler may be throwing an error',
          ]);
        }
      }
    } catch (error) {
      addStep('simulate_click', false, Date.now() - clickStart, error instanceof Error ? error.message : String(error));
      // Continue to try input test
    }

    // Step 5: Simulate typing using CDP
    const inputStart = Date.now();
    receivedEvents = []; // Reset for tracking
    try {
      // Focus the input first
      await page.focus('#test-input');
      await sleep(100);

      // Type using CDP
      await simulateRealType(page, 'test');
      await sleep(500); // Wait for events to propagate

      const inputReceived = receivedEvents.some(e => e.actionType === 'input' || e.actionType === 'change');
      addStep('simulate_input', inputReceived, Date.now() - inputStart, inputReceived ? undefined : 'Input event not received', {
        eventsReceived: receivedEvents.length,
        eventTypes: receivedEvents.map(e => e.actionType),
      });
    } catch (error) {
      addStep('simulate_input', false, Date.now() - inputStart, error instanceof Error ? error.message : String(error));
    }

    // Step 6: Query final telemetry and stats
    const finalStart = Date.now();
    try {
      diagnostics.telemetryAfter = await queryBrowserTelemetry(page);
      diagnostics.routeStatsAfter = contextInitializer.getRouteHandlerStats();
      diagnostics.scriptStatusAfter = {
        ...(diagnostics.scriptStatusBefore || { loaded: false, ready: false, inMainContext: false, handlersCount: 0, version: null }),
        isActive: await queryIsActive(page),
      };
      addStep('query_final_state', true, Date.now() - finalStart);
    } catch (error) {
      addStep('query_final_state', false, Date.now() - finalStart, error instanceof Error ? error.message : String(error));
    }

    // Determine overall success
    const clickStep = steps.find(s => s.name === 'simulate_click');
    const inputStep = steps.find(s => s.name === 'simulate_input');
    const overallSuccess = (clickStep?.passed || false) || (inputStep?.passed || false);

    if (!overallSuccess) {
      // Determine failure point from diagnostics
      return buildResult(false, 'unknown', 'No events were received during the test', steps, diagnostics, startTime, [
        'Check the detailed step results for more information',
        'Run the debug endpoint to see current state',
      ]);
    }

    return buildResult(true, undefined, undefined, steps, diagnostics, startTime);

  } finally {
    // Cleanup
    if (consoleHandler) {
      page.off('console', consoleHandler);
    }

    // Clear our test handler if no original handler
    if (!originalHandler) {
      contextInitializer.clearEventHandler();
    }
  }
}

// =============================================================================
// Helper Functions
// =============================================================================

function buildResult(
  success: boolean,
  failurePoint: PipelineFailurePoint | undefined,
  failureMessage: string | undefined,
  steps: PipelineStepResult[],
  diagnostics: PipelineTestDiagnostics,
  startTime: number,
  suggestions?: string[]
): PipelineTestResult {
  return {
    success,
    timestamp: new Date().toISOString(),
    durationMs: Date.now() - startTime,
    failurePoint,
    failureMessage,
    suggestions,
    steps,
    diagnostics,
  };
}

function sleep(ms: number): Promise<void> {
  return new Promise(resolve => setTimeout(resolve, ms));
}

async function queryBrowserTelemetry(page: Page): Promise<BrowserTelemetry | undefined> {
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
  return undefined;
}

async function queryIsActive(page: Page): Promise<boolean | null> {
  try {
    const client = await page.context().newCDPSession(page);
    try {
      const { result } = await client.send('Runtime.evaluate', {
        expression: `typeof window.__isRecordingActive === 'function' ? window.__isRecordingActive() : null`,
        returnByValue: true,
      });
      if (result.type === 'boolean') {
        return result.value;
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
 * Simulate a real click using CDP Input.dispatchMouseEvent.
 * This bypasses Playwright's click() which might not trigger DOM events the same way.
 */
async function simulateRealClick(page: Page, selector: string): Promise<void> {
  // Get element bounding box
  const element = await page.$(selector);
  if (!element) {
    throw new Error(`Element not found: ${selector}`);
  }

  const box = await element.boundingBox();
  if (!box) {
    throw new Error(`Element has no bounding box: ${selector}`);
  }

  // Calculate center of element
  const x = box.x + box.width / 2;
  const y = box.y + box.height / 2;

  // Use CDP to dispatch mouse events
  const client = await page.context().newCDPSession(page);
  try {
    // Mouse move
    await client.send('Input.dispatchMouseEvent', {
      type: 'mouseMoved',
      x,
      y,
    });

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

    logger.debug(scopedLog(LogContext.RECORDING, 'simulated real click'), { selector, x, y });
  } finally {
    await client.detach().catch(() => {});
  }
}

/**
 * Simulate real typing using CDP Input.dispatchKeyEvent.
 */
async function simulateRealType(page: Page, text: string): Promise<void> {
  const client = await page.context().newCDPSession(page);
  try {
    for (const char of text) {
      // Key down
      await client.send('Input.dispatchKeyEvent', {
        type: 'keyDown',
        text: char,
        key: char,
        code: `Key${char.toUpperCase()}`,
      });

      // Key up
      await client.send('Input.dispatchKeyEvent', {
        type: 'keyUp',
        key: char,
        code: `Key${char.toUpperCase()}`,
      });

      await sleep(50); // Small delay between keystrokes
    }

    logger.debug(scopedLog(LogContext.RECORDING, 'simulated real type'), { text });
  } finally {
    await client.detach().catch(() => {});
  }
}
