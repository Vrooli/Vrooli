import type {
  BridgeLogEvent,
  BridgeNetworkEvent,
} from '@vrooli/iframe-bridge';

import type { ReportConsoleEntry, ReportNetworkEntry, ConsoleSeverity } from './reportTypes';

export const MAX_CONSOLE_TEXT_LENGTH = 2000;
export const MAX_NETWORK_URL_LENGTH = 2048;
export const MAX_NETWORK_ERROR_LENGTH = 1500;
export const MAX_NETWORK_REQUEST_ID_LENGTH = 128;

export const trimForPayload = (value: string, max: number): string => {
  if (typeof value !== 'string' || value.length === 0) {
    return value;
  }
  if (!Number.isFinite(max) || max <= 0 || value.length <= max) {
    return value;
  }
  return `${value.slice(0, max)}...`;
};

export const normalizeConsoleLevel = (level: string | null | undefined): ConsoleSeverity => {
  const normalized = (level ?? '').toString().toLowerCase();
  if (['error', 'err', 'fatal', 'severe'].includes(normalized)) {
    return 'error';
  }
  if (['warn', 'warning'].includes(normalized)) {
    return 'warn';
  }
  if (['info', 'information', 'notice'].includes(normalized)) {
    return 'info';
  }
  if (['debug', 'verbose'].includes(normalized)) {
    return 'debug';
  }
  if (normalized === 'trace') {
    return 'trace';
  }
  return 'log';
};

export const describeLogValue = (value: unknown): string => {
  if (value === null) {
    return 'null';
  }
  if (value === undefined) {
    return 'undefined';
  }
  if (typeof value === 'string') {
    return value;
  }
  if (typeof value === 'number' || typeof value === 'boolean') {
    return String(value);
  }
  try {
    return JSON.stringify(value);
  } catch {
    return String(value);
  }
};

export const buildConsoleEventBody = (event: BridgeLogEvent): string => {
  const segments: string[] = [];
  if (event.message) {
    segments.push(event.message);
  }
  if (Array.isArray(event.args) && event.args.length > 0) {
    segments.push(event.args.map(describeLogValue).join(' '));
  }
  if (event.context && Object.keys(event.context).length > 0) {
    segments.push(describeLogValue(event.context));
  }
  const body = segments.filter(Boolean).join(' ');
  return body || '(no console output)';
};

export const formatBridgeLogEvent = (event: BridgeLogEvent): string => {
  const timestamp = (() => {
    try {
      return new Date(event.ts).toLocaleTimeString();
    } catch {
      return String(event.ts);
    }
  })();
  const level = event.level.toUpperCase();
  const source = event.source;
  const body = buildConsoleEventBody(event);
  return `${timestamp} [${source}/${level}] ${body}`.trim();
};

export const toConsoleEntry = (event: BridgeLogEvent): ReportConsoleEntry => {
  const timestamp = (() => {
    try {
      return new Date(event.ts).toLocaleTimeString();
    } catch {
      return String(event.ts);
    }
  })();
  const body = trimForPayload(buildConsoleEventBody(event), MAX_CONSOLE_TEXT_LENGTH);
  return {
    display: formatBridgeLogEvent(event),
    severity: normalizeConsoleLevel(event.level),
    payload: {
      ts: event.ts,
      level: event.level,
      source: event.source,
      text: body,
    },
    timestamp,
    source: event.source,
    body,
  };
};

export const formatBridgeNetworkEvent = (event: BridgeNetworkEvent): string => {
  const timestamp = (() => {
    try {
      return new Date(event.ts).toLocaleTimeString();
    } catch {
      return String(event.ts);
    }
  })();

  const method = event.method?.toUpperCase() ?? 'GET';
  const url = event.url ?? '(unknown URL)';
  const statusLabel = typeof event.status === 'number' ? ` [${event.status}]` : '';
  const durationLabel = typeof event.durationMs === 'number'
    ? ` (${Math.max(0, Math.round(event.durationMs))}ms)`
    : '';
  return `${timestamp} ${method} ${url}${statusLabel}${durationLabel}`;
};

export const toNetworkEntry = (event: BridgeNetworkEvent): ReportNetworkEntry => {
  const sanitizedURL = trimForPayload((event.url || '').trim() || '(unknown URL)', MAX_NETWORK_URL_LENGTH);
  const sanitizedError = trimForPayload(event.error ?? '', MAX_NETWORK_ERROR_LENGTH).trim();
  const sanitizedRequestId = trimForPayload(event.requestId ?? '', MAX_NETWORK_REQUEST_ID_LENGTH).trim();
  const normalizedDuration = typeof event.durationMs === 'number' && Number.isFinite(event.durationMs)
    ? Math.max(0, Math.round(event.durationMs))
    : undefined;
  const method = (event.method ?? 'GET').toUpperCase();
  const statusLabel = typeof event.status === 'number'
    ? `HTTP ${event.status}`
    : typeof event.ok === 'boolean'
      ? (event.ok ? 'OK' : 'Error')
      : '—';
  const durationLabel = typeof normalizedDuration === 'number' ? `${normalizedDuration} ms` : null;
  const timestamp = (() => {
    try {
      return new Date(event.ts).toLocaleTimeString();
    } catch {
      return String(event.ts);
    }
  })();

  return {
    display: formatBridgeNetworkEvent({
      ...event,
      method,
      url: sanitizedURL,
      error: sanitizedError,
      requestId: sanitizedRequestId,
      durationMs: normalizedDuration,
    }),
    payload: {
      ts: event.ts,
      kind: event.kind,
      method,
      url: sanitizedURL,
      status: typeof event.status === 'number' ? event.status : undefined,
      ok: typeof event.ok === 'boolean' ? event.ok : undefined,
      durationMs: normalizedDuration,
      requestId: sanitizedRequestId || undefined,
      error: sanitizedError || undefined,
    },
    timestamp,
    method,
    statusLabel,
    durationLabel,
    errorText: sanitizedError || null,
  };
};
