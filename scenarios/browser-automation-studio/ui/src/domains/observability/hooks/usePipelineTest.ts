/**
 * usePipelineTest Hook
 *
 * Runs the automated recording pipeline self-test.
 * This test navigates to an internal test page, simulates user interactions,
 * and verifies that events flow through the entire recording pipeline.
 *
 * Transport: Connect-RPC via `observability` client in src/api/observability.ts.
 */

import { useState, useCallback } from 'react';
import { observability } from '@/api/observability';
import { logger } from '@/utils/logger';

export interface PipelineStepResult {
  name: string;
  passed: boolean;
  duration_ms: number;
  error?: string;
  details?: Record<string, unknown>;
}

export interface ScriptStatus {
  loaded: boolean;
  ready: boolean;
  inMainContext: boolean;
  handlersCount: number;
  version: string | null;
  isActive: boolean | null;
}

export interface BrowserTelemetry {
  eventsDetected: number;
  eventsCaptured: number;
  eventsSent: number;
  eventsSendSuccess: number;
  eventsSendFailed: number;
  lastError: string | null;
}

export interface RouteStats {
  eventsReceived: number;
  eventsProcessed: number;
  eventsDroppedNoHandler: number;
  eventsWithErrors: number;
}

export interface CapturedEvent {
  actionType: string;
  timestamp: string;
  selector?: string;
}

export interface ConsoleMessage {
  type: string;
  text: string;
}

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

export interface PipelineTestResponse {
  success: boolean;
  timestamp: string;
  duration_ms: number;
  failure_point?: PipelineFailurePoint;
  failure_message?: string;
  suggestions?: string[];
  steps: PipelineStepResult[];
  diagnostics: PipelineTestDiagnostics;
  used_temp_session?: boolean;
  session_id?: string;
}

export interface PipelineTestRequest {
  test_url?: string;
  timeout_ms?: number;
}

interface UsePipelineTestReturn {
  runTest: (options?: PipelineTestRequest) => Promise<PipelineTestResponse>;
  isRunning: boolean;
  result: PipelineTestResponse | null;
  error: Error | null;
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
      const payload = await observability.runPipelineTest<PipelineTestResponse>({
        test_url: options.test_url,
        timeout_ms: options.timeout_ms ?? 30000,
      });
      setResult(payload);
      return payload;
    } catch (err: unknown) {
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
