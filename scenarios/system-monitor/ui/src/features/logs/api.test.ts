import { describe, expect, it } from 'vitest';
import { parseBootsResponse, parseLogsResponse, parseUnitsResponse } from './api';

describe('log response parsers', () => {
  it('parses response envelopes and their entries', () => {
    expect(parseLogsResponse({
      available: true,
      generatedAt: '2026-01-01T00:00:00Z',
      direction: 'reverse',
      entries: [{ timestamp: '2026-01-01T00:00:00Z', realtime: 1, priority: 3, message: 'started', bootId: 'boot-a' }],
    }).entries?.[0]?.bootId).toBe('boot-a');
    expect(parseUnitsResponse({ available: true, generatedAt: '2026-01-01T00:00:00Z', units: ['sshd.service'] }).units).toEqual(['sshd.service']);
    expect(parseBootsResponse({ available: true, generatedAt: '2026-01-01T00:00:00Z', boots: [{ index: 0, bootId: 'boot-a', firstEntry: 'start', lastEntry: 'end' }] }).boots?.[0]?.bootId).toBe('boot-a');
  });

  it('rejects malformed records and enum fields', () => {
    expect(() => parseLogsResponse({ available: true, generatedAt: 'now', direction: 'sideways' })).toThrow('logs.direction is invalid');
    expect(() => parseUnitsResponse({ available: true, generatedAt: 'now', units: [1] })).toThrow('units.units must be a string array');
    expect(() => parseBootsResponse(null)).toThrow('log boots response must be an object');
  });
});
