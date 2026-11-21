import { describe, expect, it } from 'vitest';
import { readMetricNavigationState } from './navigationState';

describe('readMetricNavigationState', () => {
  it('keeps valid overview section, condition, and sample context', () => {
    const state = readMetricNavigationState({
      overviewSection: 'memory-pressure',
      condition: {
        alert: { id: 'alert-1', timestamp: '2026-01-01T00:00:00Z', severity: 'warning', category: 'cpu', message: 'High CPU', resolved: false },
        revalidated: true,
        subject: 'node-a',
        target: 'cpu',
        windowStart: '2026-01-01T00:00:00Z',
      },
      sample: {
        selected: { timestamp: '2026-01-01T00:00:00Z', raw: 91 },
        zoom: { start: '2026-01-01T00:00:00Z', end: '2026-01-01T00:05:00Z' },
        subject: 'node-a',
        target: 'cpu',
      },
    });

    expect(state.overviewSection).toBe('memory-pressure');
    expect(state.condition?.alert.message).toBe('High CPU');
    expect(state.sample?.selected.raw).toBe(91);
    expect(state.sample?.zoom?.end).toBe('2026-01-01T00:05:00Z');
  });

  it('drops malformed navigation details without throwing', () => {
    expect(readMetricNavigationState(null)).toEqual({});
    expect(readMetricNavigationState({
      overviewSection: 7,
      condition: { subject: 'node-a', target: 'mystery', revalidated: 'yes' },
      sample: { selected: { timestamp: '', raw: Number.NaN }, subject: 'node-a', target: 'cpu' },
    })).toEqual({});
  });
});
