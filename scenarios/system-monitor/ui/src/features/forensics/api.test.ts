import { describe, expect, it } from 'vitest';
import { parseForensicsSummary } from './api';

describe('parseForensicsSummary', () => {
  it('parses the REST casing and nested available/unavailable envelopes', () => {
    const result = parseForensicsSummary({
      generatedAt: '2026-01-01T00:00:00Z',
      pstore: { available: true, generatedAt: '2026-01-01T00:00:00Z', data: { path: '/sys/fs/pstore', entries: [] } },
      bootHistory: { available: true, generatedAt: '2026-01-01T00:00:00Z', data: { boots: [{ index: 0, bootId: 'boot-a', firstEntry: '2026-01-01T00:00:00Z', lastEntry: '2026-01-01T00:01:00Z', clean: true }] } },
      mce: { available: false, reason: 'not installed', generatedAt: '2026-01-01T00:00:00Z' },
      autoheal: { available: true, checks: [{ checkId: 'service', status: 'healthy' }] },
    });
    expect(result.bootHistory.data?.boots[0]?.bootId).toBe('boot-a');
    expect(result.mce.reason).toBe('not installed');
    expect(result.autoheal.checks?.[0]?.checkId).toBe('service');
  });

  it('rejects malformed envelope content', () => {
    expect(() => parseForensicsSummary({})).toThrow('forensics summary is malformed');
  });
});
