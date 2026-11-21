import { describe, expect, it } from 'vitest';
import { parseSystemHealthStatus } from './useHealthCheck';

describe('parseSystemHealthStatus', () => {
  it('keeps only valid health fields and nested connectivity details', () => {
    expect(parseSystemHealthStatus({
      status: 'ok',
      service: 'system-monitor',
      timestamp: 1,
      uptime: 12.5,
      processor_active: true,
      maintenance_state: 'active',
      api_connectivity: { connected: true, latency_ms: 3, error: { code: 'slow', message: 'delayed' } },
      checks: { database: 'ok' },
    })).toEqual({
      status: 'ok',
      service: 'system-monitor',
      timestamp: 1,
      uptime: 12.5,
      processor_active: true,
      maintenance_state: 'active',
      api_connectivity: { connected: true, latency_ms: 3, error: { code: 'slow', message: 'delayed' } },
      checks: { database: 'ok' },
    });
  });

  it('accepts omitted optional fields and explicit null connectivity errors', () => {
    expect(parseSystemHealthStatus({ api_connectivity: { error: null } })).toEqual({
      api_connectivity: { connected: undefined, latency_ms: undefined, error: null },
      status: undefined,
      service: undefined,
      timestamp: undefined,
      uptime: undefined,
      processor_active: undefined,
      maintenance_state: undefined,
      checks: undefined,
    });
    expect(parseSystemHealthStatus({ api_connectivity: {} }).api_connectivity?.error).toBeUndefined();
  });

  it.each([
    [null, 'health response must be an object'],
    [{ status: 1 }, 'health.status must be a string'],
    [{ service: 1 }, 'health.service must be a string'],
    [{ timestamp: {} }, 'health.timestamp must be text or a number'],
    [{ uptime: Number.POSITIVE_INFINITY }, 'health.uptime must be finite'],
    [{ processor_active: 1 }, 'health.processor_active must be boolean'],
    [{ maintenance_state: 1 }, 'health.maintenance_state must be a string'],
    [{ api_connectivity: [] }, 'health.api_connectivity must be an object'],
    [{ api_connectivity: { connected: 1 } }, 'health.api_connectivity.connected must be boolean'],
    [{ api_connectivity: { latency_ms: Number.NaN } }, 'health.api_connectivity.latency_ms must be finite'],
    [{ api_connectivity: { error: 'failed' } }, 'health.api_connectivity.error must be an object'],
    [{ api_connectivity: { error: { code: 1 } } }, 'health API error code must be a string'],
    [{ api_connectivity: { error: { message: 1 } } }, 'health API error message must be a string'],
    [{ checks: [] }, 'health.checks must be an object'],
  ])('rejects malformed input %#', (value, message) => {
    expect(() => parseSystemHealthStatus(value)).toThrow(message);
  });
});
