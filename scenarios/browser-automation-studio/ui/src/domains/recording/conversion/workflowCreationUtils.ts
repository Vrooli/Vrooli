/** Describes a contiguous range of selected steps */
export interface SelectionRange {
  start: number;
  end: number;
  count: number;
}

/**
 * Compute contiguous ranges from sorted indices.
 * E.g., [0, 1, 2, 5, 6, 10] -> [{start: 0, end: 2}, {start: 5, end: 6}, {start: 10, end: 10}]
 */
export function computeSelectionRanges(indices: number[]): SelectionRange[] {
  if (indices.length === 0) return [];

  const sorted = [...indices].sort((a, b) => a - b);
  const ranges: SelectionRange[] = [];
  const first = sorted[0];
  if (first === undefined) return ranges;

  let rangeStart = first;
  let rangeEnd = first;

  for (let i = 1; i < sorted.length; i++) {
    const current = sorted[i];
    if (current === undefined) continue;
    if (current === rangeEnd + 1) {
      rangeEnd = current;
    } else {
      ranges.push({ start: rangeStart, end: rangeEnd, count: rangeEnd - rangeStart + 1 });
      rangeStart = current;
      rangeEnd = current;
    }
  }
  ranges.push({ start: rangeStart, end: rangeEnd, count: rangeEnd - rangeStart + 1 });

  return ranges;
}
