import { describe, expect, it } from 'vitest';
import { formatRealizedDelta, sumCounts, topDimensions } from './DecisionTracePanel.helpers';
import type { DecisionTraceEntry } from '@/types/api';

function entry(overrides: Partial<DecisionTraceEntry> = {}): DecisionTraceEntry {
  return {
    iteration: 1,
    chosen_skill: 'refactor',
    heaviest_dimension: 'standards',
    dimension_scores: { standards: 8, tests: 4, docs: 0 },
    score_before: 8,
    realized_delta: 4,
    ...overrides,
  };
}

describe('formatRealizedDelta', () => {
  it('renders an improvement as a negative score change', () => {
    expect(formatRealizedDelta(entry({ realized_delta: 4 }))).toBe('−4.0 (improved)');
  });

  it('renders a regression as a positive score change', () => {
    expect(formatRealizedDelta(entry({ realized_delta: -2 }))).toBe('+2.0 (regressed)');
  });

  it('renders zero change explicitly', () => {
    expect(formatRealizedDelta(entry({ realized_delta: 0 }))).toBe('0.0 (no change)');
  });

  it('renders pending when the delta is not yet realized', () => {
    expect(formatRealizedDelta(entry({ realized_delta: undefined }))).toBe('pending');
  });
});

describe('topDimensions', () => {
  it('returns positive-scored dimensions heaviest-first, dropping zeros', () => {
    expect(topDimensions(entry())).toEqual([
      ['standards', 8],
      ['tests', 4],
    ]);
  });

  it('breaks ties alphabetically and respects the limit', () => {
    const e = entry({ dimension_scores: { b: 4, a: 4, c: 1 } });
    expect(topDimensions(e, 2)).toEqual([
      ['a', 4],
      ['b', 4],
    ]);
  });

  it('returns an empty list when there are no open dimensions', () => {
    expect(topDimensions(entry({ dimension_scores: {} }))).toEqual([]);
  });
});

describe('sumCounts', () => {
  it('totals a per-dimension count map', () => {
    expect(sumCounts({ standards: 2, tests: 3 })).toBe(5);
  });

  it('treats an absent map as zero', () => {
    expect(sumCounts(undefined)).toBe(0);
  });
});
