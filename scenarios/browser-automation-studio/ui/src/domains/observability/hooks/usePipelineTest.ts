/**
 * usePipelineTest Hook
 *
 * Runs the automated recording pipeline self-test.
 * This test navigates to an internal test page, simulates user interactions,
 * and verifies that events flow through the entire recording pipeline.
 *
 * Key benefit: Eliminates human-in-the-loop debugging by automatically
 * detecting where in the pipeline events get lost.
 *
 * This is fully autonomous - it creates a temporary session if needed.
 */

import { useState, useCallback } from 'react';
import { getConfig } from '@/config';
import { logger } from '@/utils/logger';

/**
 * Result of a single step in the pipeline test.
 */
export interface PipelineStepResult {
  name: string;
  passed: boolean;
  duration_ms: number;
  error?: string;
  details?: Record<string, unknown>;
}

/**
 * Script verification status.
 */
export interface ScriptStatus {
  loaded: boolean;
  ready: boolean;
  inMainContext: boolean;
  handlersCount: number;
  version: string | null;
  isActive: boolean | null;
}

/**
 * Browser telemetry from the recording script.
 */
export interface BrowserTelemetry {
  eventsDetected: number;
  eventsCaptured: number;
  eventsSent: number;
  eventsSendSuccess: number;
  eventsSendFailed: number;
  lastError: string | null;
}

/**
 * Route handler statistics.
 */
export interface RouteStats {
  eventsReceived: number;
  eventsProcessed: number;
  eventsDroppedNoHandler: number;
  eventsWithErrors: number;
}

/**
 * Event captured during the test.
 */
export interface CapturedEvent {
  actionType: string;
  timestamp: string;
  selector?: string;
}

/**
 * Console message captured during the test.
 */
export interface ConsoleMessage {
  type: string;
  text: string;
}

/**
 * Diagnostics collected during the pipeline test.
 */
export interface PipelineTestDiagnostics {
  test_page_url: string;
  test_page_injected: boolean;
  script_status_before?: ScriptStatus;
  script_status_after?: ScriptStatus;
  telemetry_before?: BrowserTelemetry;
  telemetry_after?: BrowserTelemetry;
  route_stats_before?: RouteStats;
  route_stats_after?: RouteStats;
  events_captured: CapturedEvent[];
  console_messages: ConsoleMessage[];
}

/**
 * Possible failure points in the pipeline.
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
 * Response from the pipeline test endpoint.
 */
export interface PipelineTestResponse {
  success: boolean;
  timestamp: string;
  duration_ms: number;
  failure_point?: PipelineFailurePoint;
  failure_message?: string;
  suggestions?: string[];
  steps: PipelineStepResult[];
  diagnostics: PipelineTestDiagnostics;
  /** Whether a temporary session was created for the test */
  used_temp_session?: boolean;
  /** Session ID used for the test */
  session_id?: string;
}

/**
 * Request options for the pipeline test.
 */
export interface PipelineTestRequest {
  /** External URL to test injection on (default: https://example.com) */
  test_url?: string;
  timeout_ms?: number;
}

interface UsePipelineTestReturn {
  /** Run the autonomous pipeline test (no session required) */
  runTest: (options?: PipelineTestRequest) => Promise<PipelineTestResponse>;
  /** Whether a test is currently running */
  isRunning: boolean;
  /** The last test response */
  result: PipelineTestResponse | null;
  /** Any error from the last test */
  error: Error | null;
  /** Reset the state */
  reset: () => void;
}

export function usePipelineTest(): UsePipelineTestReturn {
  const [isRunning, setIsRunning] = useState(false);
  const [result, setResult] = useState<PipelineTestResponse | null>(null);
  const [error, setError] = useState<Error | null>(null);

  const runTest = useCallback(async (
    options: PipelineTestRequest = {}
  ): Promise<PipelineTestResponse> => {
    setIsRunning(true);
    setError(null);

    try {
      const config = await getConfig();
      // Use the autonomous endpoint that creates a session if needed
      const response = await fetch(`${config.API_URL}/observability/pipeline-test`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          test_url: options.test_url, // undefined = use default (example.com)
          timeout_ms: options.timeout_ms ?? 30000,
        }),
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.message || `Pipeline test failed: ${response.statusText}`);
      }

      const testResult = await response.json();
      setResult(testResult);
      return testResult;
    } catch (err) {
      const error = err instanceof Error ? err : new Error(String(err));
      logger.error('Pipeline test failed', { component: 'usePipelineTest' }, error);
      setError(error);
      throw error;
    } finally {
      setIsRunning(false);
    }
  }, []);

  const reset = useCallback(() => {
    setResult(null);
    setError(null);
    setIsRunning(false);
  }, []);

  return {
    runTest,
    isRunning,
    result,
    error,
    reset,
  };
}

export default usePipelineTest;
