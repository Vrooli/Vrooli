import { describe, expect, it } from 'vitest';
import {
  buildWorkspaceMinimapRowMarkers,
  reconcileTrackFractions,
  resolveDropIndex,
  resolveWorkspaceLayout,
  resolveWorkspaceLayoutWithMaxColumns,
  scrollTopFromWorkspaceMinimapPointer,
  workspaceViewportFromScrollMetrics,
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

describe('workspace minimap helpers', () => {
  it('maps minimap pointer offset to scrollTop', () => {
    const scrollTop = scrollTopFromWorkspaceMinimapPointer(100, 200, 3000, 500);
    expect(scrollTop).toBe(1250);
  });

  it('clamps pointer mapping when rail height is invalid', () => {
    expect(scrollTopFromWorkspaceMinimapPointer(50, 0, 1000, 400)).toBe(0);
  });

  it('builds viewport metrics from scroll values', () => {
    const viewport = workspaceViewportFromScrollMetrics({
      scrollTop: 600,
      scrollHeight: 3000,
      clientHeight: 600,
    });

    expect(viewport.heightPercent).toBe(20);
    expect(viewport.maxScrollable).toBe(2400);
    expect(viewport.topPercent).toBeCloseTo(20, 4);
  });

  it('builds evenly spaced row markers', () => {
    const markers = buildWorkspaceMinimapRowMarkers(3);
    expect(markers).toEqual([
      { rowIndex: 0, topPercent: 0, heightPercent: 100 / 3 },
      { rowIndex: 1, topPercent: 100 / 3, heightPercent: 100 / 3 },
      { rowIndex: 2, topPercent: (2 * 100) / 3, heightPercent: 100 / 3 },
    ]);
  });
});
