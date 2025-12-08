import type {
  ExecutionEventMessage,
} from '../features/execution/ws/executionEvents';

export interface Screenshot {
  id: string;
  timestamp: Date;
  url: string;
  stepName: string;
}

export type LogLevel = 'info' | 'warning' | 'error' | 'success';

export interface LogEntry {
  id: string;
  timestamp: Date;
  level: LogLevel;
  message: string;
}

export interface ExecutionEventHandlers {
  updateExecutionStatus: (status: 'pending' | 'running' | 'completed' | 'failed' | 'cancelled', error?: string) => void;
  updateProgress: (progress: number, currentStep?: string) => void;
  addLog: (log: LogEntry) => void;
  addScreenshot: (screenshot: Screenshot) => void;
  recordHeartbeat: (step?: string, elapsedMs?: number) => void;
}

export interface ExecutionEventContext {
  fallbackTimestamp?: string;
  fallbackProgress?: number;
}

export const createId = () => {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID();
  }
  return Math.random().toString(36).slice(2);
};

export const parseTimestamp = (timestamp?: string | number) => {
  if (typeof timestamp === 'number' && Number.isFinite(timestamp)) {
    return new Date(timestamp);
  }
  if (typeof timestamp === 'string') {
    const trimmed = timestamp.trim();
    if (trimmed.length > 0) {
      const parsed = new Date(trimmed);
      if (!Number.isNaN(parsed.valueOf())) {
        return parsed;
      }
    }
  }
  return new Date();
};

export const stepLabel = (event: ExecutionEventMessage) =>
  event.step_node_id || event.step_type || (typeof event.step_index === 'number' ? `Step ${event.step_index + 1}` : 'Step');

export const processExecutionEvent = (
  handlers: ExecutionEventHandlers,
  event: ExecutionEventMessage,
  context: ExecutionEventContext = {},
) => {
  const eventTimestamp = event.timestamp ?? context.fallbackTimestamp;
  if (typeof event.progress === 'number') {
    handlers.updateProgress(event.progress, event.step_node_id ?? event.step_type);
  } else if (typeof context.fallbackProgress === 'number') {
    handlers.updateProgress(context.fallbackProgress, event.step_node_id ?? event.step_type);
  }

  const appendRetrySuffix = (baseMessage: string) => {
    const payload = event.payload as Record<string, unknown> | null | undefined;
    if (!payload) {
      return baseMessage;
    }
    const attemptRaw = payload['retry_attempt'] ?? payload['retryAttempt'];
    const attemptNumber = Number(attemptRaw);
    if (!Number.isFinite(attemptNumber) || attemptNumber <= 1) {
      return baseMessage;
    }
    const attempt = Math.trunc(attemptNumber);
    const maxRaw = payload['retry_max_attempts'] ?? payload['retryMaxAttempts'];
    const maxNumber = Number(maxRaw);
    const maxAttempts = Number.isFinite(maxNumber) && maxNumber > 0 ? Math.trunc(maxNumber) : undefined;
    const suffix = maxAttempts ? ` (attempt ${attempt}/${maxAttempts})` : ` (attempt ${attempt})`;
    return `${baseMessage}${suffix}`;
  };

  switch (event.type) {
    case 'execution.started':
      handlers.updateExecutionStatus('running');
      return;
    case 'execution.completed':
      handlers.updateExecutionStatus('completed');
      return;
    case 'execution.failed':
      handlers.updateExecutionStatus('failed', event.message);
      return;
    case 'execution.cancelled':
      handlers.updateExecutionStatus('cancelled', event.message ?? 'Execution cancelled');
      return;
    case 'execution.progress':
      if (typeof event.progress === 'number') {
        handlers.updateProgress(event.progress, event.step_node_id ?? event.step_type);
      }
      return;
    case 'step.started':
      handlers.addLog({
        id: createId(),
        level: 'info',
        message: event.message ?? `${stepLabel(event)} started`,
        timestamp: parseTimestamp(eventTimestamp),
      });
      return;
    case 'step.completed':
    case 'step.failed': {
      const baseMessage = event.message ?? `${stepLabel(event)} ${event.type === 'step.completed' ? 'completed' : 'failed'}`;
      const message = appendRetrySuffix(baseMessage);
      handlers.addLog({
        id: createId(),
        level: event.type === 'step.failed' ? 'error' : 'success',
        message,
        timestamp: parseTimestamp(eventTimestamp),
      });

      const assertion = event.payload && typeof event.payload === 'object' ? (event.payload as Record<string, unknown>)?.assertion : undefined;
      if (assertion && typeof assertion === 'object') {
        const assertionData = assertion as Record<string, unknown>;
        const selector = typeof assertionData.selector === 'string' ? assertionData.selector : undefined;
        const mode = typeof assertionData.mode === 'string' ? assertionData.mode : undefined;
        const assertionLabel = selector || mode || stepLabel(event);
        const success = assertionData.success !== false;
        const assertionMessage = assertionData.message;
        const detail = success
          ? `Assertion ${assertionLabel} passed`
          : `Assertion ${assertionLabel} failed${assertionMessage ? `: ${assertionMessage}` : ''}`;
        handlers.addLog({
          id: createId(),
          level: success ? 'success' : 'error',
          message: detail,
          timestamp: parseTimestamp(eventTimestamp),
        });
      }

      if (event.type === 'step.failed') {
        handlers.updateExecutionStatus('failed', message);
      }

      const domPreview = (() => {
        const payload = event.payload ?? {};
        const payloadObj = payload as Record<string, unknown>;
        const direct = typeof payloadObj.dom_snapshot_preview === 'string' ? payloadObj.dom_snapshot_preview : undefined;
        const camel = typeof payloadObj.domSnapshotPreview === 'string' ? payloadObj.domSnapshotPreview : undefined;
        return direct ?? camel;
      })();
      if (domPreview) {
        handlers.addLog({
          id: createId(),
          level: 'info',
          message: `DOM snapshot captured (${stepLabel(event)})`,
          timestamp: parseTimestamp(eventTimestamp),
        });
      }
      return;
    }
    case 'step.heartbeat':
      handlers.recordHeartbeat(
        stepLabel(event),
        typeof event.payload?.elapsed_ms === 'number' ? event.payload.elapsed_ms : undefined,
      );
      return;
    case 'step.screenshot': {
      const payload = event.payload ?? {};
      const url = (payload.url as string | undefined) ??
        ((payload.base64 as string | undefined) ? `data:image/png;base64,${payload.base64 as string}` : undefined);
      if (!url) {
        return;
      }

      const screenshot: Screenshot = {
        id: (payload.screenshot_id as string | undefined) ?? createId(),
        url,
        stepName: stepLabel(event),
        timestamp: parseTimestamp((payload.timestamp as string | undefined) ?? eventTimestamp),
      };

      handlers.addScreenshot(screenshot);
      return;
    }
    case 'step.log': {
      const payload = event.payload ?? {};
      const message = (payload.message as string | undefined) ?? event.message ?? 'Step log';
      const level = (payload.level as LogLevel | undefined) ?? 'info';
      handlers.addLog({
        id: createId(),
        level,
        message,
        timestamp: parseTimestamp(eventTimestamp),
      });
      return;
    }
    case 'step.telemetry': {
      const payload = event.payload ?? {};
      const payloadObj = payload as Record<string, unknown>;
      const consoleEntries = Array.isArray(payloadObj.console_logs)
        ? (payloadObj.console_logs as Array<Record<string, unknown>>)
        : [];
      const networkEntries = Array.isArray(payloadObj.network_events)
        ? (payloadObj.network_events as Array<Record<string, unknown>>)
        : [];

      const label = stepLabel(event);

      const toConsoleLevel = (type: unknown): LogLevel => {
        switch (typeof type === 'string' ? type.toLowerCase() : '') {
          case 'error':
            return 'error';
          case 'warning':
          case 'warn':
            return 'warning';
          case 'success':
            return 'success';
          default:
            return 'info';
        }
      };

      for (const entry of consoleEntries) {
        const type = typeof entry.type === 'string' ? entry.type : 'log';
        const text = typeof entry.text === 'string' ? entry.text : '';
        if (!text) {
          continue;
        }

        handlers.addLog({
          id: createId(),
          level: toConsoleLevel(type),
          message: `${label} console.${type}: ${text}`,
          timestamp: parseTimestamp((entry.timestamp as string | number | undefined) ?? eventTimestamp),
        });
      }

      for (const entry of networkEntries) {
        const type = typeof entry.type === 'string' ? entry.type.toLowerCase() : 'event';
        const method = typeof entry.method === 'string' ? entry.method : undefined;
        const status = typeof entry.status === 'number' ? entry.status : undefined;
        const url = typeof entry.url === 'string' ? entry.url : undefined;
        const failure = typeof entry.failure === 'string' ? entry.failure : undefined;

        const pieces: string[] = [];
        if (type === 'requestfailed') {
          pieces.push('Network failure');
        } else if (type === 'response') {
          pieces.push('Network response');
        } else {
          pieces.push('Network request');
        }
        if (method) {
          pieces.push(method.toUpperCase());
        }
        if (status) {
          pieces.push(String(status));
        }
        if (url) {
          pieces.push(url);
        }
        if (failure) {
          pieces.push(`(${failure})`);
        }

        handlers.addLog({
          id: createId(),
          level: type === 'requestfailed' ? 'error' : 'info',
          message: `${label} ${pieces.join(' ')}`,
          timestamp: parseTimestamp((entry.timestamp as string | number | undefined) ?? eventTimestamp),
        });
      }

      return;
    }
    default:
      return;
  }
};
