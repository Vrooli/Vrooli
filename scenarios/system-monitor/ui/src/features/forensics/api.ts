import { apiFetch } from '../../shared/api/apiFetch';
import { isRecord } from '../../shared/utils/typeGuards';
import type { BootEntry, BootHistoryReport, ForensicsEnvelope, ForensicsRelevantCheck, ForensicsSummary, MCEReport, PstoreEntry, PstoreReport } from './types';

function requiredString(record: Record<string, unknown>, field: string): string {
  const value = record[field];
  if (typeof value !== 'string') throw new TypeError(`forensics.${field} must be a string`);
  return value;
}

function requiredNumber(record: Record<string, unknown>, field: string): number {
  const value = record[field];
  if (typeof value !== 'number' || !Number.isFinite(value)) throw new TypeError(`forensics.${field} must be finite`);
  return value;
}

function requiredBoolean(record: Record<string, unknown>, field: string): boolean {
  const value = record[field];
  if (typeof value !== 'boolean') throw new TypeError(`forensics.${field} must be boolean`);
  return value;
}

function optionalString(record: Record<string, unknown>, field: string): string | undefined {
  const value = record[field];
  if (value === undefined || value === null) return undefined;
  if (typeof value !== 'string') throw new TypeError(`forensics.${field} must be a string`);
  return value;
}

function parseEnvelope<T>(value: unknown, parseData: (value: unknown) => T): ForensicsEnvelope<T> {
  if (!isRecord(value)) throw new TypeError('forensics envelope must be an object');
  const reason = optionalString(value, 'reason');
  return {
    available: requiredBoolean(value, 'available'),
    reason,
    data: value['data'] === undefined || value['data'] === null ? undefined : parseData(value['data']),
    generatedAt: requiredString(value, 'generatedAt'),
  };
}

function parsePstore(value: unknown): PstoreReport {
  if (!isRecord(value) || !Array.isArray(value['entries'])) throw new TypeError('pstore report is malformed');
  const entries: PstoreEntry[] = value['entries'].map(entry => {
    if (!isRecord(entry)) throw new TypeError('pstore entry must be an object');
    return {
      name: requiredString(entry, 'name'),
      kind: requiredString(entry, 'kind'),
      size: requiredNumber(entry, 'size'),
      modified: requiredString(entry, 'modified'),
    };
  });
  return { path: requiredString(value, 'path'), entries };
}

function parseBootHistory(value: unknown): BootHistoryReport {
  if (!isRecord(value) || !Array.isArray(value['boots'])) throw new TypeError('boot history report is malformed');
  const boots: BootEntry[] = value['boots'].map(boot => {
    if (!isRecord(boot)) throw new TypeError('boot entry must be an object');
    return {
      index: requiredNumber(boot, 'index'),
      bootId: requiredString(boot, 'bootId'),
      firstEntry: requiredString(boot, 'firstEntry'),
      lastEntry: requiredString(boot, 'lastEntry'),
      clean: requiredBoolean(boot, 'clean'),
      reason: optionalString(boot, 'reason'),
    };
  });
  return { boots };
}

function parseMCE(value: unknown): MCEReport {
  if (!isRecord(value)) throw new TypeError('MCE report must be an object');
  return {
    window: requiredString(value, 'window'),
    uncorrected: requiredNumber(value, 'uncorrected'),
    corrected: requiredNumber(value, 'corrected'),
    rawSummary: optionalString(value, 'rawSummary'),
  };
}

export function parsePstoreEnvelope(value: unknown): ForensicsEnvelope<PstoreReport> {
  return parseEnvelope(value, parsePstore);
}

export function parseBootHistoryEnvelope(value: unknown): ForensicsEnvelope<BootHistoryReport> {
  return parseEnvelope(value, parseBootHistory);
}

export function parseMCEEnvelope(value: unknown): ForensicsEnvelope<MCEReport> {
  return parseEnvelope(value, parseMCE);
}

function parseCheck(value: unknown): ForensicsRelevantCheck {
  if (!isRecord(value)) throw new TypeError('autoheal check must be an object');
  const details = value['details'];
  if (details !== undefined && details !== null && !isRecord(details)) throw new TypeError('autoheal check details must be an object');
  return {
    checkId: requiredString(value, 'checkId'),
    status: requiredString(value, 'status'),
    message: optionalString(value, 'message'),
    category: optionalString(value, 'category'),
    details: isRecord(details) ? details : undefined,
    lastRunAt: optionalString(value, 'lastRunAt'),
  };
}

export function parseForensicsSummary(value: unknown): ForensicsSummary {
  if (!isRecord(value) || !isRecord(value['autoheal'])) throw new TypeError('forensics summary is malformed');
  const checksValue = value['autoheal']['checks'];
  if (checksValue !== undefined && !Array.isArray(checksValue)) throw new TypeError('autoheal checks must be an array');
  return {
    generatedAt: requiredString(value, 'generatedAt'),
    pstore: parsePstoreEnvelope(value['pstore']),
    bootHistory: parseBootHistoryEnvelope(value['bootHistory']),
    mce: parseMCEEnvelope(value['mce']),
    autoheal: {
      available: requiredBoolean(value['autoheal'], 'available'),
      reason: optionalString(value['autoheal'], 'reason'),
      checks: Array.isArray(checksValue) ? checksValue.map(parseCheck) : undefined,
    },
  };
}

export function fetchForensicsSummary(signal?: AbortSignal): Promise<ForensicsSummary> {
  return apiFetch('/forensics/summary', parseForensicsSummary, { signal });
}

export function fetchPstore(signal?: AbortSignal): Promise<ForensicsEnvelope<PstoreReport>> {
  return apiFetch('/forensics/pstore', parsePstoreEnvelope, { signal });
}

export function fetchBootHistory(signal?: AbortSignal): Promise<ForensicsEnvelope<BootHistoryReport>> {
  return apiFetch('/forensics/boot-history', parseBootHistoryEnvelope, { signal });
}

export function fetchMCE(signal?: AbortSignal): Promise<ForensicsEnvelope<MCEReport>> {
  return apiFetch('/forensics/mce', parseMCEEnvelope, { signal });
}
