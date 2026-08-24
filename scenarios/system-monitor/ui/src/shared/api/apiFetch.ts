// DOC: docs/internal/INTEROP_AUDIT.md
import { buildRootUrl, buildUrl as buildApiUrl } from '../../lib/api-client';
import type { APIError, ErrorDetail } from '../../types';

/**
 * Error subclass that also carries the normalized APIError fields.
 *
 * Thrown by the fetch helpers so that callers can `throw` a real `Error`
 * (satisfying lint/runtime expectations) while still reading `.error`,
 * `.detail`, and `.timestamp` exactly as before via `isApiError`/`toApiError`.
 */
export class ApiErrorException extends Error implements APIError {
  error: string;
  detail?: ErrorDetail;
  timestamp?: string;

  constructor(payload: APIError) {
    super(payload.error);
    this.name = 'ApiErrorException';
    this.error = payload.error;
    this.detail = payload.detail;
    this.timestamp = payload.timestamp;
  }
}

/** Merge a base header record with an optional `HeadersInit` into a plain object. */
function mergeHeaders(base: Record<string, string>, extra?: HeadersInit): Record<string, string> {
  const merged: Record<string, string> = { ...base };
  if (!extra) return merged;
  if (extra instanceof Headers) {
    extra.forEach((value, key) => { merged[key] = value; });
  } else if (Array.isArray(extra)) {
    for (const [key, value] of extra) merged[key] = value;
  } else {
    Object.assign(merged, extra);
  }
  return merged;
}

/** Parse an error response into a normalized APIError. */
async function parseErrorResponse(response: Response): Promise<APIError> {
  const errorText = await response.text();
  try {
    const parsed = JSON.parse(errorText) as Record<string, unknown>;
    // Unified format: { error: { code, message, retryable, ... } }
    if (parsed.error && typeof parsed.error === 'object' && 'code' in (parsed.error as Record<string, unknown>)) {
      const detail = parsed.error as ErrorDetail;
      return {
        error: detail.message,
        detail,
        timestamp: new Date().toISOString(),
      };
    }
    // Legacy or unexpected JSON — use what we can.
    return {
      error: typeof parsed.error === 'string' ? parsed.error : response.statusText,
      timestamp: new Date().toISOString(),
    };
  } catch {
    return {
      error: `HTTP ${response.status}: ${response.statusText}`,
      timestamp: new Date().toISOString(),
    };
  }
}

/** Build a network-error APIError (fetch itself threw). */
function networkError(): APIError {
  return {
    error: 'Unable to reach the server. Check your connection.',
    detail: { code: 'network', message: 'Unable to reach the server. Check your connection.', retryable: true, recovery: 'wait' },
    timestamp: new Date().toISOString(),
  };
}

/**
 * Shared fetch utility that handles:
 * - Prepending the API base URL
 * - JSON response parsing
 * - Error normalization into APIError
 * - Optional AbortSignal forwarding
 *
 * Use this instead of raw `fetch()` + `buildApiUrl()` in hooks and callbacks.
 * For components that need loading/error state, use `useApiCall` which wraps this.
 */
export async function apiFetch<T>(
  path: string,
  options?: RequestInit
): Promise<T> {
  let response: Response;
  try {
    response = await fetch(buildApiUrl(path), options);
  } catch (err) {
    if (err instanceof DOMException && err.name === 'AbortError') throw err;
    throw new ApiErrorException(networkError());
  }

  if (!response.ok) {
    throw new ApiErrorException(await parseErrorResponse(response));
  }

  try {
    return (await response.json()) as T;
  } catch {
    throw new ApiErrorException({
      error: 'Invalid response from server',
      detail: { code: 'internal', message: 'Failed to parse server response', retryable: false },
      timestamp: new Date().toISOString(),
    });
  }
}

/**
 * Fetch + proto-parse in one step.
 *
 * Works like `apiFetch` but passes the raw JSON through a proto `parser`
 * function (from proto-contracts.ts) before returning, giving callers a
 * fully-typed protobuf message shape.
 */
export async function protoFetch<T>(
  path: string,
  parser: (data: unknown) => T,
  options?: RequestInit,
): Promise<T> {
  const connectCall = buildConnectCall(path, options);
  const url = connectCall ? buildRootUrl(connectCall.procedure) : buildApiUrl(path);
  let response: Response;
  try {
    response = await fetch(url, {
      ...options,
      method: connectCall ? 'POST' : options?.method,
      body: connectCall ? JSON.stringify(connectCall.body) : options?.body,
      headers: mergeHeaders({ 'Content-Type': 'application/json' }, options?.headers),
    });
  } catch (err) {
    if (err instanceof DOMException && err.name === 'AbortError') throw err;
    throw new ApiErrorException(networkError());
  }
  if (!response.ok) {
    throw new ApiErrorException(await parseErrorResponse(response));
  }
  let json: unknown;
  try {
    json = await response.json();
  } catch {
    throw new ApiErrorException({
      error: 'Invalid response from server',
      detail: { code: 'internal', message: 'Failed to parse server response', retryable: false },
      timestamp: new Date().toISOString(),
    });
  }
  try {
    return parser(connectCall?.unwrap(json) ?? json);
  } catch {
    throw new ApiErrorException({
      error: 'Invalid response format',
      detail: { code: 'internal', message: 'Failed to decode server response', retryable: false },
      timestamp: new Date().toISOString(),
    });
  }
}

interface ConnectCall {
  procedure: string;
  body: Record<string, unknown>;
  unwrap(data: unknown): unknown;
}

const identity = (data: unknown): unknown => data;
const field = (name: string) => (data: unknown): unknown =>
  data && typeof data === 'object' ? (data as Record<string, unknown>)[name] : undefined;

function readJSONBody(options?: RequestInit): Record<string, unknown> {
  const body = options?.body;
  if (typeof body !== 'string' || body.trim() === '') return {};
  try {
    const parsed = JSON.parse(body);
    return parsed && typeof parsed === 'object' && !Array.isArray(parsed) ? parsed as Record<string, unknown> : {};
  } catch {
    return {};
  }
}

function secondsFromWindow(value: string | null): number | undefined {
  if (!value) return undefined;
  const numeric = Number(value);
  if (Number.isFinite(numeric) && numeric > 0) return Math.floor(numeric);
  const match = value.match(/^(\d+)(ms|s|m|h)$/);
  if (!match) return undefined;
  const amount = Number(match[1]);
  if (!Number.isFinite(amount) || amount <= 0) return undefined;
  switch (match[2]) {
    case 'ms':
      return Math.max(1, Math.ceil(amount / 1000));
    case 's':
      return amount;
    case 'm':
      return amount * 60;
    case 'h':
      return amount * 3600;
    default:
      return undefined;
  }
}

function boolParam(value: string | null): boolean {
  return value === '1' || value === 'true';
}

function intParam(value: string | null): number | undefined {
  if (!value) return undefined;
  const parsed = Number(value);
  return Number.isFinite(parsed) && parsed > 0 ? Math.floor(parsed) : undefined;
}

function buildConnectCall(rawPath: string, options?: RequestInit): ConnectCall | null {
  const url = new URL(rawPath, 'http://system-monitor.local');
  const path = url.pathname.replace(/^\/api\/v1/, '');
  const body = readJSONBody(options);
  const method = (options?.method ?? 'GET').toUpperCase();

  const metrics = '/vrooli.system_monitor.v1.metrics.MetricsService';
  const settings = '/vrooli.system_monitor.v1.settings.SettingsService';
  const reports = '/vrooli.system_monitor.v1.reports.ReportsService';
  const capacity = '/vrooli.system_monitor.v1.capacity.CapacityService';
  const investigations = '/vrooli.system_monitor.v1.investigations.InvestigationsService';
  const scripts = '/vrooli.system_monitor.v1.scripts.ScriptsService';

  switch (path) {
    case '/metrics/current':
      return { procedure: `${metrics}/GetCurrentMetrics`, body: { fresh: boolParam(url.searchParams.get('fresh')) }, unwrap: field('metrics') };
    case '/metrics/detailed':
      return { procedure: `${metrics}/GetDetailedMetrics`, body: {}, unwrap: field('metrics') };
    case '/metrics/processes':
      return { procedure: `${metrics}/GetProcessMonitor`, body: {}, unwrap: field('data') };
    case '/metrics/infrastructure':
      return { procedure: `${metrics}/GetInfrastructureMonitor`, body: {}, unwrap: field('data') };
    case '/metrics/timeline':
      return {
        procedure: `${metrics}/GetMetricsTimeline`,
        body: {
          windowSeconds: intParam(url.searchParams.get('window')),
          sampleIntervalSeconds: intParam(url.searchParams.get('interval')),
        },
        unwrap: field('timeline'),
      };
    case '/metrics/processes/timeline':
      return {
        procedure: `${metrics}/GetProcessTimeline`,
        body: {
          windowSeconds: secondsFromWindow(url.searchParams.get('window')),
          owner: url.searchParams.get('owner') ?? '',
          top: intParam(url.searchParams.get('top')),
        },
        unwrap: field('timeline'),
      };
    case '/settings':
      if (method === 'PUT' || method === 'POST') {
        return { procedure: `${settings}/UpdateSettings`, body: { settings: body }, unwrap: identity };
      }
      return { procedure: `${settings}/GetSettings`, body: {}, unwrap: identity };
    case '/settings/reset':
      return { procedure: `${settings}/ResetSettings`, body: {}, unwrap: identity };
    case '/maintenance/state':
      if (method === 'POST' || method === 'PUT') {
        return { procedure: `${settings}/SetMaintenanceState`, body, unwrap: identity };
      }
      return { procedure: `${settings}/GetMaintenanceState`, body: {}, unwrap: identity };
    case '/reports':
      return { procedure: `${reports}/ListReports`, body: {}, unwrap: identity };
    case '/reports/generate':
      return { procedure: `${reports}/GenerateReport`, body, unwrap: identity };
    case '/capacity/overview':
      return { procedure: `${capacity}/GetCapacityOverview`, body: {}, unwrap: identity };
    case '/capacity/reconcile':
      return { procedure: `${capacity}/ReconcileCapacity`, body: {}, unwrap: identity };
    case '/capacity/policy':
      if (method === 'POST' || method === 'PUT') {
        return { procedure: `${capacity}/SetCapacityPolicy`, body, unwrap: identity };
      }
      return { procedure: `${capacity}/GetCapacityPolicy`, body: {}, unwrap: identity };
    case '/investigations':
      return { procedure: `${investigations}/ListInvestigations`, body: { limit: intParam(url.searchParams.get('limit')) }, unwrap: field('investigations') };
    case '/investigations/latest':
    case '/investigations/agent/current':
      return { procedure: `${investigations}/GetLatestInvestigation`, body: {}, unwrap: field('investigation') };
    case '/investigations/trigger':
    case '/investigations/agent/spawn':
      return {
        procedure: `${investigations}/TriggerInvestigation`,
        body: {
          autoFix: Boolean(body.autoFix ?? body.auto_fix),
          note: typeof body.note === 'string' ? body.note : '',
        },
        unwrap: identity,
      };
    case '/investigations/cooldown':
      return { procedure: `${investigations}/GetCooldownStatus`, body: {}, unwrap: identity };
    case '/investigations/cooldown/reset':
      return { procedure: `${investigations}/ResetCooldown`, body: {}, unwrap: identity };
    case '/investigations/cooldown/period':
      return {
        procedure: `${investigations}/UpdateCooldownPeriod`,
        body: { cooldownPeriodSeconds: Number(body.cooldownPeriodSeconds ?? body.cooldown_period_seconds ?? 0) },
        unwrap: identity,
      };
    case '/investigations/triggers':
      return { procedure: `${investigations}/GetTriggers`, body: {}, unwrap: identity };
    case '/investigations/scripts':
      return { procedure: `${scripts}/ListScripts`, body: {}, unwrap: identity };
    default:
      break;
  }

  let match = path.match(/^\/reports\/([^/]+)$/);
  if (match?.[1]) {
    return { procedure: `${reports}/GetReport`, body: { id: decodeURIComponent(match[1]) }, unwrap: identity };
  }
  match = path.match(/^\/investigations\/agent\/([^/]+)\/status$/);
  if (match?.[1]) {
    return { procedure: `${investigations}/GetInvestigation`, body: { id: decodeURIComponent(match[1]) }, unwrap: field('investigation') };
  }
  match = path.match(/^\/investigations\/agent\/([^/]+)\/stop$/);
  if (match?.[1]) {
    return { procedure: `${investigations}/StopAgent`, body: { id: decodeURIComponent(match[1]) }, unwrap: identity };
  }
  match = path.match(/^\/investigations\/scripts\/([^/]+)$/);
  if (match?.[1]) {
    if (method === 'PUT') {
      return { procedure: `${scripts}/UpdateScript`, body: { id: decodeURIComponent(match[1]), content: body.content ?? '' }, unwrap: identity };
    }
    return { procedure: `${scripts}/GetScript`, body: { id: decodeURIComponent(match[1]) }, unwrap: identity };
  }
  match = path.match(/^\/investigations\/scripts\/([^/]+)\/execute$/);
  if (match?.[1]) {
    return { procedure: `${scripts}/ExecuteScript`, body: { id: decodeURIComponent(match[1]), content: body.content }, unwrap: identity };
  }

  return null;
}

/** Type guard: returns true when `err` is shaped like an APIError. */
export function isApiError(err: unknown): err is APIError {
  return err != null && typeof err === 'object' && 'error' in err;
}

/** Extract a human-readable message from any thrown value. */
export function extractErrorMessage(err: unknown, fallback = 'An unknown error occurred'): string {
  if (isApiError(err)) return err.error;
  if (err instanceof Error) return err.message;
  return fallback;
}

/** Normalize any thrown value into an APIError. */
export function toApiError(err: unknown): APIError {
  if (isApiError(err)) return err;
  if (err instanceof DOMException && err.name === 'AbortError') throw err;
  // Standard JS errors are NOT network errors — preserve their message
  if (err instanceof Error) {
    return {
      error: err.message,
      detail: { code: 'internal', message: err.message, retryable: false },
      timestamp: new Date().toISOString(),
    };
  }
  return networkError();
}
