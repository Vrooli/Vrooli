import type { IncomingMessage, ServerResponse } from 'http';
import type { SessionManager } from '../session';
import { measureRealtimeAudio } from '../session';
import { sendJson } from '../middleware';
import { logger, scopedLog, LogContext } from '../utils';
import {
  RecordingDiagnosticLevel,
  type DiagnosticIssue,
  type RecordingDiagnosticResult,
} from '../recording';
import type { DiagnosticRunRequest, DiagnosticRunResponse, RecordingDiagnostics } from './types';

export interface DiagnosticsDependencies {
  sessionManager: SessionManager;
}
const record = (v: unknown): v is Record<string, unknown> =>
  typeof v === 'object' && v !== null && !Array.isArray(v);
const level = (v: unknown): 'quick' | 'standard' | 'full' | undefined =>
  v === 'quick' || v === 'standard' || v === 'full' ? v : undefined;
const diagnosticLevel = (value?: 'quick' | 'standard' | 'full'): RecordingDiagnosticLevel =>
  value === 'full'
    ? RecordingDiagnosticLevel.FULL
    : value === 'standard'
      ? RecordingDiagnosticLevel.STANDARD
      : RecordingDiagnosticLevel.QUICK;
const category = (
  code: string
): 'script' | 'injection' | 'event' | 'provider' | 'cdp' | 'general' =>
  code.startsWith('SCRIPT_')
    ? 'script'
    : code.startsWith('INJECTION_')
      ? 'injection'
      : code.startsWith('EVENT_')
        ? 'event'
        : code.startsWith('PROVIDER_')
          ? 'provider'
          : code.startsWith('CDP_')
            ? 'cdp'
            : 'general';
const transform = (result: RecordingDiagnosticResult): RecordingDiagnostics => ({
  ready: result.ready,
  timestamp: result.timestamp,
  durationMs: result.durationMs,
  level: result.level,
  checks: result.checks,
  provider: result.provider,
  eventFlowTest: result.eventFlowTest,
  issues: result.issues.map((issue: DiagnosticIssue) => ({
    severity: issue.severity as 'error' | 'warning' | 'info',
    category: category(issue.code),
    message: issue.message,
    suggestion: issue.suggestion,
  })),
});
const requestFrom = (body: string): DiagnosticRunRequest => {
  let value: unknown;
  try {
    value = JSON.parse(body || '{}');
  } catch {
    value = {};
  }
  const input = record(value) ? value : {};
  const options = record(input.options) ? input.options : undefined;
  return {
    type:
      input.type === 'recording' ||
      input.type === 'browser' ||
      input.type === 'audio' ||
      input.type === 'all'
        ? input.type
        : 'recording',
    session_id: typeof input.session_id === 'string' ? input.session_id : undefined,
    options: options
      ? {
          level: level(options.level),
          timeout_ms:
            typeof options.timeout_ms === 'number' && Number.isFinite(options.timeout_ms)
              ? options.timeout_ms
              : undefined,
        }
      : undefined,
  };
};
const recordingError = (
  message: string,
  value?: 'quick' | 'standard' | 'full'
): RecordingDiagnostics => ({
  ready: false,
  timestamp: new Date().toISOString(),
  durationMs: 0,
  level: diagnosticLevel(value),
  issues: [
    {
      severity: 'error',
      category: 'general',
      message,
      suggestion: 'Check the browser console for JavaScript errors',
    },
  ],
  checks: [],
  provider: { name: 'unknown', evaluateIsolated: false, exposeBindingIsolated: false },
});

export function handleDiagnosticsRun(
  req: IncomingMessage,
  res: ServerResponse,
  deps: DiagnosticsDependencies
): void {
  let body = '';
  req.on('data', (chunk: Buffer) => {
    body += chunk.toString();
  });
  req.on('end', () => {
    void (async (): Promise<void> => {
      const startedAt = new Date();
      try {
        const request = requestFrom(body);
        const results: DiagnosticRunResponse['results'] = {};
        logger.info(scopedLog(LogContext.HEALTH, 'running diagnostics'), {
          type: request.type,
          sessionId: request.session_id,
          level: request.options?.level,
        });
        if (request.type === 'recording' || request.type === 'all') {
          const id = request.session_id ?? deps.sessionManager.getAllSessionIds()[0];
          if (id)
            try {
              const session = deps.sessionManager.getSession(id);
              const { runRecordingDiagnostics } = await import('../recording');
              results.recording = transform(
                await runRecordingDiagnostics(session.page, session.context, {
                  level: diagnosticLevel(request.options?.level),
                  timeoutMs: request.options?.timeout_ms ?? 5000,
                  contextInitializer: session.recordingInitializer,
                })
              );
            } catch (error) {
              results.recording = recordingError(
                `Recording diagnostics failed: ${error instanceof Error ? error.message : String(error)}`,
                request.options?.level
              );
            }
          else
            results.recording = {
              ...recordingError(
                'No active browser sessions available for diagnostics',
                request.options?.level
              ),
              issues: [
                {
                  severity: 'warning',
                  category: 'general',
                  message: 'No active browser sessions available for diagnostics',
                  suggestion: 'Start a browser session first by navigating to a page',
                },
              ],
            };
        }
        if (request.type === 'audio' || request.type === 'all') {
          let id: string | undefined;
          let created = false;
          try {
            id = request.session_id ?? deps.sessionManager.getAllSessionIds()[0];
            if (!id) {
              id = (
                await deps.sessionManager.startSession({
                  execution_id: `audio-capability-${Date.now()}`,
                  workflow_id: 'audio-capability',
                  viewport: { width: 1280, height: 720 },
                  reuse_mode: 'fresh',
                })
              ).sessionId;
              created = true;
            }
            const session = deps.sessionManager.getSession(id);
            if (created) await session.page.goto('http://127.0.0.1:24485/health');
            const audio = await measureRealtimeAudio(
              session.page,
              request.options?.timeout_ms ?? 2000,
              session.audioCapability,
              session.audioStrategy
            );
            results.audio = {
              available: audio.available,
              current_time_delta: audio.currentTimeDelta,
              callback_count: audio.callbackCount,
              output_latency: audio.outputLatency,
              state: audio.state,
              duration_ms: audio.durationMs,
              finding: audio.finding,
              audio_strategy: session.audioStrategy,
              host_audio_outcome: session.audioCapability?.outcome,
              host_audio_reason: session.audioCapability?.reason,
            };
          } catch (error) {
            results.audio = {
              available: false,
              current_time_delta: 0,
              callback_count: 0,
              output_latency: null,
              state: 'error',
              duration_ms: 0,
              finding: `realtime_audio_unavailable: ${error instanceof Error ? error.message : String(error)}`,
            };
          } finally {
            if (created && id)
              await deps.sessionManager
                .closeSession(id)
                .catch((error) =>
                  logger.warn(
                    scopedLog(LogContext.HEALTH, 'audio capability session cleanup failed'),
                    { error: error instanceof Error ? error.message : String(error) }
                  )
                );
          }
        }
        const completedAt = new Date();
        sendJson(res, 200, {
          started_at: startedAt.toISOString(),
          completed_at: completedAt.toISOString(),
          duration_ms: completedAt.getTime() - startedAt.getTime(),
          results,
        });
      } catch (error) {
        logger.error(scopedLog(LogContext.HEALTH, 'diagnostics run failed'), {
          error: error instanceof Error ? error.message : String(error),
        });
        sendJson(res, 500, {
          error: 'Failed to run diagnostics',
          message: error instanceof Error ? error.message : String(error),
        });
      }
    })();
  });
}
