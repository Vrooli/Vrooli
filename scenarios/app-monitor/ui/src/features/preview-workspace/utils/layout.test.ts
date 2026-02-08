import { describe, expect, it } from 'vitest';
import {
  reconcileTrackFractions,
  resolveDropIndex,
  resolveWorkspaceLayout,
  resolveWorkspaceLayoutWithMaxColumns,
} from './layout';

describe('resolveWorkspaceLayout', () => {
  it('returns single-column for one pane', () => {
    expect(resolveWorkspaceLayout(1)).toEqual({
      className: 'preview-workspace__panes--single',
      columns: 1,
      rows: 1,
    });
  });

  it('returns two columns for two panes', () => {
    expect(resolveWorkspaceLayout(2)).toEqual({
      className: 'preview-workspace__panes--double',
      columns: 2,
      rows: 1,
    });
  });

  it('returns two-by-two grid for four panes', () => {
    expect(resolveWorkspaceLayout(4)).toEqual({
      className: 'preview-workspace__panes--grid',
      columns: 2,
      rows: 2,
    });
  });

  it('caps columns to one when maxColumns is one', () => {
    expect(resolveWorkspaceLayoutWithMaxColumns(4, 1)).toEqual({
      className: 'preview-workspace__panes--single',
      columns: 1,
      rows: 4,
    });
  });
});

describe('reconcileTrackFractions', () => {
  it('returns [1] for invalid or single track counts', () => {
    expect(reconcileTrackFractions([0.2, 0.8], 0)).toEqual([1]);
    expect(reconcileTrackFractions([0.2, 0.8], 1)).toEqual([1]);
  });

  it('normalizes and expands fractions for additional tracks', () => {
    const result = reconcileTrackFractions([2, 1], 3);
    expect(result).toHaveLength(3);
    expect(result.reduce((sum, value) => sum + value, 0)).toBeCloseTo(1, 5);
  });
});

describe('resolveDropIndex', () => {
  const rect = {
    left: 0,
    top: 0,
    width: 100,
    height: 100,
  } as DOMRect;

  it('resolves center cell based on grid coordinates', () => {
    const result = resolveDropIndex({
      pointerX: 75,
      pointerY: 75,
      rect,
      columns: 2,
      rows: 2,
      paneCount: 4,
    });

    expect(result).toBe(3);
  });

  it('snaps top-left corner to first index', () => {
    const result = resolveDropIndex({
      pointerX: 3,
      pointerY: 2,
      rect,
      columns: 2,
      rows: 2,
      paneCount: 4,
    });

    expect(result).toBe(0);
  });
});
