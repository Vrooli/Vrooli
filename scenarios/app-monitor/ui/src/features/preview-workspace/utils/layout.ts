export type WorkspaceLayoutDescriptor = {
  className: string;
  columns: number;
  rows: number;
};

export type WorkspaceMinimapViewport = {
  topPercent: number;
  heightPercent: number;
  maxScrollable: number;
};

export type WorkspaceMinimapRowMarker = {
  rowIndex: number;
  topPercent: number;
  heightPercent: number;
};

const clampPaneCount = (count: number): number => {
  if (!Number.isFinite(count) || count < 1) {
    return 1;
  }
  return Math.floor(count);
};

const clamp = (value: number, min: number, max: number): number => {
  if (value < min) {
    return min;
  }
  if (value > max) {
    return max;
  }
  return value;
};

export const resolveWorkspaceLayout = (paneCount: number): WorkspaceLayoutDescriptor => {
  const normalizedCount = clampPaneCount(paneCount);
  const maxColumns = 2;

  return resolveWorkspaceLayoutWithMaxColumns(normalizedCount, maxColumns);
};

export const resolveWorkspaceLayoutWithMaxColumns = (
  paneCount: number,
  maxColumns: number,
): WorkspaceLayoutDescriptor => {
  const normalizedCount = clampPaneCount(paneCount);
  const safeMaxColumns = Math.max(1, Math.floor(maxColumns));

  if (normalizedCount <= 1 || safeMaxColumns <= 1) {
    return {
      className: 'preview-workspace__panes--single',
      columns: 1,
      rows: normalizedCount,
    };
  }

  const columns = Math.min(2, safeMaxColumns);
  return {
    className: normalizedCount === 2 ? 'preview-workspace__panes--double' : 'preview-workspace__panes--grid',
    columns,
    rows: Math.ceil(normalizedCount / columns),
  };
};

const normalizeFractions = (fractions: number[]): number[] => {
  const safeFractions = fractions.map((value) => (Number.isFinite(value) && value > 0 ? value : 0));
  const sum = safeFractions.reduce((acc, value) => acc + value, 0);
  if (sum <= 0) {
    return [];
  }
  return safeFractions.map((value) => value / sum);
};

export const reconcileTrackFractions = (
  currentFractions: number[],
  nextTrackCount: number,
): number[] => {
  if (!Number.isFinite(nextTrackCount) || nextTrackCount <= 0) {
    return [1];
  }

  const safeCount = Math.floor(nextTrackCount);
  if (safeCount === 1) {
    return [1];
  }

  const normalizedCurrent = normalizeFractions(currentFractions);
  if (normalizedCurrent.length === safeCount) {
    return normalizedCurrent;
  }

  if (normalizedCurrent.length === 0) {
    return Array.from({ length: safeCount }, () => 1 / safeCount);
  }

  if (normalizedCurrent.length > safeCount) {
    return normalizeFractions(normalizedCurrent.slice(0, safeCount));
  }

  const next = [...normalizedCurrent];
  while (next.length < safeCount) {
    const last = next[next.length - 1] ?? 1 / safeCount;
    next.push(last);
  }

  return normalizeFractions(next);
};

export const buildGridTrackTemplate = (
  fractions: number[],
  splitterSize: number,
): string => {
  if (fractions.length <= 1) {
    return 'minmax(0, 1fr)';
  }

  const safeSplitter = Number.isFinite(splitterSize) && splitterSize > 0 ? splitterSize : 6;
  const segments: string[] = [];
  fractions.forEach((fraction, index) => {
    const clampedFraction = Number.isFinite(fraction) && fraction > 0 ? fraction : 1;
    segments.push(`minmax(0, ${clampedFraction}fr)`);
    if (index < fractions.length - 1) {
      segments.push(`${safeSplitter}px`);
    }
  });
  return segments.join(' ');
};

export const resolveDropIndex = ({
  pointerX,
  pointerY,
  rect,
  columns,
  rows,
  paneCount,
}: {
  pointerX: number;
  pointerY: number;
  rect: DOMRect;
  columns: number;
  rows: number;
  paneCount: number;
}): number => {
  if (paneCount <= 1) {
    return 0;
  }

  const safeColumns = Math.max(1, Math.floor(columns));
  const safeRows = Math.max(1, Math.floor(rows));
  const rawX = (pointerX - rect.left) / rect.width;
  const rawY = (pointerY - rect.top) / rect.height;
  const x = Math.min(1, Math.max(0, rawX));
  const y = Math.min(1, Math.max(0, rawY));
  const edgeThreshold = 0.17;

  const clampIndex = (value: number) => Math.max(0, Math.min(paneCount - 1, value));

  if (x <= edgeThreshold && y <= edgeThreshold) {
    return 0;
  }
  if (x >= 1 - edgeThreshold && y <= edgeThreshold) {
    return clampIndex(safeColumns - 1);
  }
  if (x <= edgeThreshold && y >= 1 - edgeThreshold) {
    return clampIndex((safeRows - 1) * safeColumns);
  }
  if (x >= 1 - edgeThreshold && y >= 1 - edgeThreshold) {
    return clampIndex(safeRows * safeColumns - 1);
  }

  if (x <= edgeThreshold) {
    return clampIndex(Math.floor((safeRows - 1) / 2) * safeColumns);
  }
  if (x >= 1 - edgeThreshold) {
    return clampIndex(Math.floor((safeRows - 1) / 2) * safeColumns + (safeColumns - 1));
  }
  if (y <= edgeThreshold) {
    return clampIndex(Math.floor((safeColumns - 1) / 2));
  }
  if (y >= 1 - edgeThreshold) {
    return clampIndex((safeRows - 1) * safeColumns + Math.floor((safeColumns - 1) / 2));
  }

  const column = Math.min(safeColumns - 1, Math.floor(x * safeColumns));
  const row = Math.min(safeRows - 1, Math.floor(y * safeRows));
  return clampIndex((row * safeColumns) + column);
};

export const scrollTopFromWorkspaceMinimapPointer = (
  pointerOffsetY: number,
  railHeight: number,
  scrollHeight: number,
  clientHeight: number,
): number => {
  if (!Number.isFinite(railHeight) || railHeight <= 0) {
    return 0;
  }
  const ratio = clamp(pointerOffsetY / railHeight, 0, 1);
  const maxScrollable = Math.max(scrollHeight - clientHeight, 0);
  return ratio * maxScrollable;
};

export const workspaceViewportFromScrollMetrics = ({
  scrollTop,
  scrollHeight,
  clientHeight,
  minViewportPercent = 8,
}: {
  scrollTop: number;
  scrollHeight: number;
  clientHeight: number;
  minViewportPercent?: number;
}): WorkspaceMinimapViewport => {
  const safeScrollHeight = Math.max(1, Number.isFinite(scrollHeight) ? scrollHeight : 1);
  const safeClientHeight = Math.max(1, Number.isFinite(clientHeight) ? clientHeight : 1);
  const safeMinPercent = clamp(
    Number.isFinite(minViewportPercent) ? minViewportPercent : 8,
    1,
    100,
  );

  const maxScrollable = Math.max(safeScrollHeight - safeClientHeight, 0);
  const heightPercent = clamp((safeClientHeight / safeScrollHeight) * 100, safeMinPercent, 100);
  const maxTopPercent = Math.max(0, 100 - heightPercent);
  const safeScrollTop = clamp(Number.isFinite(scrollTop) ? scrollTop : 0, 0, maxScrollable);
  const topPercent = maxScrollable <= 0
    ? 0
    : (safeScrollTop / maxScrollable) * maxTopPercent;

  return {
    topPercent,
    heightPercent,
    maxScrollable,
  };
};

export const buildWorkspaceMinimapRowMarkers = (rowCount: number): WorkspaceMinimapRowMarker[] => {
  const safeRowCount = Number.isFinite(rowCount) ? Math.floor(rowCount) : 0;
  if (safeRowCount <= 0) {
    return [];
  }

  const rowHeight = 100 / safeRowCount;
  return Array.from({ length: safeRowCount }, (_, index) => ({
    rowIndex: index,
    topPercent: index * rowHeight,
    heightPercent: rowHeight,
  }));
};
