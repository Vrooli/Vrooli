import { describe, it, expect } from 'vitest';
import { computeSelectionRanges, type SelectionRange } from './WorkflowCreationForm';

describe('computeSelectionRanges', () => {
  it('returns empty array for empty input', () => {
    expect(computeSelectionRanges([])).toEqual([]);
  });

  it('handles single index', () => {
    const result = computeSelectionRanges([5]);
    expect(result).toEqual([{ start: 5, end: 5, count: 1 }]);
  });

  it('groups contiguous indices into a single range', () => {
    const result = computeSelectionRanges([0, 1, 2, 3, 4]);
    expect(result).toEqual([{ start: 0, end: 4, count: 5 }]);
  });

  it('creates separate ranges for non-contiguous indices', () => {
    const result = computeSelectionRanges([0, 1, 2, 5, 6, 10]);
    expect(result).toEqual([
      { start: 0, end: 2, count: 3 },
      { start: 5, end: 6, count: 2 },
      { start: 10, end: 10, count: 1 },
    ]);
  });

  it('handles unsorted input by sorting first', () => {
    const result = computeSelectionRanges([5, 2, 3, 1, 4]);
    expect(result).toEqual([{ start: 1, end: 5, count: 5 }]);
  });

  it('handles single-item gaps', () => {
    const result = computeSelectionRanges([0, 2, 4, 6]);
    expect(result).toEqual([
      { start: 0, end: 0, count: 1 },
      { start: 2, end: 2, count: 1 },
      { start: 4, end: 4, count: 1 },
      { start: 6, end: 6, count: 1 },
    ]);
  });

  it('correctly counts items in each range', () => {
    const result = computeSelectionRanges([10, 11, 12, 20, 21, 22, 23, 24]);
    expect(result).toEqual([
      { start: 10, end: 12, count: 3 },
      { start: 20, end: 24, count: 5 },
    ]);
  });
});
