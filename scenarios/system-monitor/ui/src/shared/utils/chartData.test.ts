import { describe, it, expect } from 'vitest';

import { combineFlowSeries, combineMemorySeries } from './chartData';

describe('combineMemorySeries', () => {
  it('merges memory and swap readings that share a timestamp into one point', () => {
    const result = combineMemorySeries(
      [{ timestamp: '2026-08-21T20:00:00Z', value: 37.4 }],
      [{ timestamp: '2026-08-21T20:00:00Z', value: 33.1 }]
    );
    expect(result).toEqual([{ timestamp: '2026-08-21T20:00:00Z', memory: 37.4, swap: 33.1 }]);
  });

  // The whole point of the swap series is showing divergence: memory flat and
  // healthy while swap climbs. Both must survive the merge independently.
  it('keeps memory and swap independent when they diverge', () => {
    const result = combineMemorySeries(
      [
        { timestamp: '2026-08-21T20:00:00Z', value: 37 },
        { timestamp: '2026-08-21T20:00:05Z', value: 37 }
      ],
      [
        { timestamp: '2026-08-21T20:00:00Z', value: 10 },
        { timestamp: '2026-08-21T20:00:05Z', value: 46 }
      ]
    );
    expect(result.map(p => p.memory)).toEqual([37, 37]);
    expect(result.map(p => p.swap)).toEqual([10, 46]);
  });

  it('sorts by timestamp regardless of input order', () => {
    const result = combineMemorySeries(
      [
        { timestamp: '2026-08-21T20:00:10Z', value: 3 },
        { timestamp: '2026-08-21T20:00:00Z', value: 1 }
      ],
      []
    );
    expect(result.map(p => p.timestamp)).toEqual([
      '2026-08-21T20:00:00Z',
      '2026-08-21T20:00:10Z'
    ]);
  });

  // A missing swap reading must leave the key undefined so the line breaks,
  // rather than defaulting to 0 and drawing a false "swap emptied" cliff.
  it('leaves a series undefined at timestamps where it has no reading', () => {
    const result = combineMemorySeries(
      [{ timestamp: '2026-08-21T20:00:00Z', value: 37.4 }],
      []
    );
    expect(result[0].memory).toBe(37.4);
    expect(result[0].swap).toBeUndefined();
  });

  it('drops non-finite readings instead of plotting them', () => {
    const result = combineMemorySeries(
      [{ timestamp: '2026-08-21T20:00:00Z', value: Number.NaN }],
      [{ timestamp: '2026-08-21T20:00:00Z', value: 12 }]
    );
    expect(result[0].memory).toBeUndefined();
    expect(result[0].swap).toBe(12);
  });

  it('returns an empty array when neither series has data', () => {
    expect(combineMemorySeries(undefined, undefined)).toEqual([]);
  });
});

describe('combineFlowSeries', () => {
  it('preserves missing samples instead of inventing zero flow', () => {
    expect(combineFlowSeries([
      { key: 'swapTraffic', points: [{ timestamp: '2026-01-01T00:00:00Z', value: 12 }] },
      { key: 'majorFaults', points: [] }
    ])).toEqual([{ timestamp: '2026-01-01T00:00:00Z', swapTraffic: 12 }]);
  });

  it('merges the swap level and traffic series for the dual-axis chart', () => {
    expect(combineFlowSeries([
      { key: 'swapLevel', points: [{ timestamp: '2026-01-01T00:00:00Z', value: 45 }] },
      { key: 'swapTraffic', points: [{ timestamp: '2026-01-01T00:00:00Z', value: 128 }] }
    ])).toEqual([{ timestamp: '2026-01-01T00:00:00Z', swapLevel: 45, swapTraffic: 128 }]);
  });
});
