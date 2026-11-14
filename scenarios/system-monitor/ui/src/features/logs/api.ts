import { apiFetch } from '../../shared/api/apiFetch';
import { isRecord } from '../../shared/utils/typeGuards';
import {
  type BootRecord,
  type BootsResponse,
  DEFAULT_LIMIT,
  type LogDirection,
  type LogEntry,
  type LogQueryFilters,
  type LogsResponse,
  MAX_LIMIT,
  type UnitsResponse,
} from './types';

function requiredString(record: Record<string, unknown>, field: string): string {
  const value = record[field];
  if (typeof value !== 'string') throw new TypeError(`logs.${field} must be a string`);
  return value;
}

function requiredNumber(record: Record<string, unknown>, field: string): number {
  const value = record[field];
  if (typeof value !== 'number' || !Number.isFinite(value)) throw new TypeError(`logs.${field} must be finite`);
  return value;
}

function optionalString(record: Record<string, unknown>, field: string): string | undefined {
  const value = record[field];
  if (value === undefined || value === null) return undefined;
  if (typeof value !== 'string') throw new TypeError(`logs.${field} must be a string`);
  return value;
}

function optionalNumber(record: Record<string, unknown>, field: string): number | undefined {
  const value = record[field];
  if (value === undefined || value === null) return undefined;
  if (typeof value !== 'number' || !Number.isFinite(value)) throw new TypeError(`logs.${field} must be finite`);
  return value;
}

function parseLogEntry(value: unknown): LogEntry {
  if (!isRecord(value)) throw new TypeError('log entry must be an object');
  return {
    timestamp: requiredString(value, 'timestamp'),
    realtime: requiredNumber(value, 'realtime'),
    priority: requiredNumber(value, 'priority'),
    unit: optionalString(value, 'unit'),
    userUnit: optionalString(value, 'userUnit'),
    identifier: optionalString(value, 'identifier'),
    hostname: optionalString(value, 'hostname'),
    pid: optionalNumber(value, 'pid'),
    bootId: optionalString(value, 'bootId'),
    cursor: optionalString(value, 'cursor'),
    message: requiredString(value, 'message'),
    raw: optionalString(value, 'raw'),
  };
}

export function parseLogsResponse(value: unknown): LogsResponse {
  if (!isRecord(value)) throw new TypeError('logs response must be an object');
  const available = value['available'];
  const entries = value['entries'];
  const direction = optionalString(value, 'direction');
  const limit = optionalNumber(value, 'limit');
  if (typeof available !== 'boolean') throw new TypeError('logs.available must be boolean');
  if (entries !== undefined && !Array.isArray(entries)) throw new TypeError('logs.entries must be an array');
  if (direction !== undefined && direction !== 'forward' && direction !== 'reverse') throw new TypeError('logs.direction is invalid');
  return {
    available,
    reason: optionalString(value, 'reason'),
    error: optionalString(value, 'error'),
    entries: Array.isArray(entries) ? entries.map(parseLogEntry) : undefined,
    nextCursor: optionalString(value, 'nextCursor'),
    direction,
    limit,
    generatedAt: requiredString(value, 'generatedAt'),
  };
}

export function parseUnitsResponse(value: unknown): UnitsResponse {
  if (!isRecord(value)) throw new TypeError('log units response must be an object');
  const available = value['available'];
  const units = value['units'];
  if (typeof available !== 'boolean') throw new TypeError('units.available must be boolean');
  if (units !== undefined && (!Array.isArray(units) || !units.every(unit => typeof unit === 'string'))) throw new TypeError('units.units must be a string array');
  return {
    available,
    reason: optionalString(value, 'reason'),
    units: Array.isArray(units) ? units : undefined,
    generatedAt: requiredString(value, 'generatedAt'),
  };
}

function parseBootRecord(value: unknown): BootRecord {
  if (!isRecord(value)) throw new TypeError('boot record must be an object');
  return {
    index: requiredNumber(value, 'index'),
    bootId: requiredString(value, 'bootId'),
    firstEntry: requiredString(value, 'firstEntry'),
    lastEntry: requiredString(value, 'lastEntry'),
  };
}

export function parseBootsResponse(value: unknown): BootsResponse {
  if (!isRecord(value)) throw new TypeError('log boots response must be an object');
  const available = value['available'];
  const boots = value['boots'];
  if (typeof available !== 'boolean') throw new TypeError('boots.available must be boolean');
  if (boots !== undefined && !Array.isArray(boots)) throw new TypeError('boots.boots must be an array');
  return {
    available,
    reason: optionalString(value, 'reason'),
    boots: Array.isArray(boots) ? boots.map(parseBootRecord) : undefined,
    generatedAt: requiredString(value, 'generatedAt'),
  };
}

export interface FetchLogsArgs {
  filters: LogQueryFilters;
  cursor?: string;
  direction?: LogDirection;
  signal?: AbortSignal;
}

export function buildLogsQueryString(args: FetchLogsArgs): string {
  const { filters, cursor, direction } = args;
  const params = new URLSearchParams();
  for (const u of filters.units) params.append('unit', u);
  if (filters.kernel) params.set('kernel', 'true');
  if (filters.since) params.set('since', filters.since);
  if (filters.until) params.set('until', filters.until);
  if (filters.priority) params.set('priority', filters.priority);
  if (filters.grep) params.set('grep', filters.grep);
  if (filters.boot) params.set('boot', filters.boot);
  const limit = Math.min(Math.max(1, filters.limit || DEFAULT_LIMIT), MAX_LIMIT);
  params.set('limit', String(limit));
  if (cursor) params.set('cursor', cursor);
  if (direction) params.set('direction', direction);
  return params.toString();
}

export function fetchLogs(args: FetchLogsArgs): Promise<LogsResponse> {
  const qs = buildLogsQueryString(args);
  return apiFetch(`/logs?${qs}`, parseLogsResponse, { signal: args.signal });
}

export function fetchUnits(signal?: AbortSignal): Promise<UnitsResponse> {
  return apiFetch('/logs/units', parseUnitsResponse, { signal });
}

export function fetchBoots(signal?: AbortSignal): Promise<BootsResponse> {
  return apiFetch('/logs/boots', parseBootsResponse, { signal });
}
