import type { Machine } from '../../types';
import { isRecord } from '../../shared/utils/typeGuards';
import { apiFetch } from '../../shared/api/apiFetch';

function requiredString(record: Record<string, unknown>, field: string): string {
  const value = record[field];
  if (typeof value !== 'string') throw new TypeError(`machine.${field} must be a string`);
  return value;
}

function requiredBoolean(record: Record<string, unknown>, field: string): boolean {
  const value = record[field];
  if (typeof value !== 'boolean') throw new TypeError(`machine.${field} must be a boolean`);
  return value;
}

function optionalString(record: Record<string, unknown>, field: string): string | undefined {
  const value = record[field];
  if (value === undefined || value === null) return undefined;
  if (typeof value !== 'string') throw new TypeError(`machine.${field} must be a string`);
  return value;
}

function parseMachine(value: unknown): Machine {
  if (!isRecord(value)) throw new TypeError('machine must be an object');
  const heartbeatAge = value['heartbeat_age_seconds'];
  if (heartbeatAge !== undefined && heartbeatAge !== null && (typeof heartbeatAge !== 'number' || !Number.isFinite(heartbeatAge))) {
    throw new TypeError('machine.heartbeat_age_seconds must be a finite number');
  }
  const scopes = value['scopes'];
  if (scopes !== undefined && scopes !== null && (!Array.isArray(scopes) || !scopes.every(scope => typeof scope === 'string'))) {
    throw new TypeError('machine.scopes must be a string array');
  }
  const readinessValue = value['readiness'];
  if (readinessValue !== undefined && readinessValue !== null && !Array.isArray(readinessValue)) {
    throw new TypeError('machine.readiness must be an array');
  }
  const readiness = Array.isArray(readinessValue) ? readinessValue.map(item => {
    if (!isRecord(item) || typeof item['identity'] !== 'string' || typeof item['passed'] !== 'boolean') {
      throw new TypeError('machine readiness entry is malformed');
    }
    if (item['detail'] !== undefined && typeof item['detail'] !== 'string') throw new TypeError('machine readiness detail must be a string');
    if (item['recovery_action'] !== undefined && typeof item['recovery_action'] !== 'string') throw new TypeError('machine recovery action must be a string');
    return {
      identity: item['identity'],
      passed: item['passed'],
      detail: typeof item['detail'] === 'string' ? item['detail'] : undefined,
      recovery_action: typeof item['recovery_action'] === 'string' ? item['recovery_action'] : undefined,
    };
  }) : undefined;

  return {
    id: requiredString(value, 'id'),
    name: requiredString(value, 'name'),
    os: optionalString(value, 'os'),
    arch: optionalString(value, 'arch'),
    online: requiredBoolean(value, 'online'),
    heartbeat_fresh: requiredBoolean(value, 'heartbeat_fresh'),
    heartbeat_age_seconds: typeof heartbeatAge === 'number' ? heartbeatAge : undefined,
    dispatchable: requiredBoolean(value, 'dispatchable'),
    status: requiredString(value, 'status'),
    grant: optionalString(value, 'grant'),
    scopes: Array.isArray(scopes) ? scopes : undefined,
    readiness,
  };
}

export function parseMachineList(value: unknown): Machine[] {
  if (!Array.isArray(value)) throw new TypeError('machines response must be an array');
  return value.map(parseMachine);
}

export function fetchMachines(): Promise<Machine[]> {
  return apiFetch('/machines', parseMachineList);
}
