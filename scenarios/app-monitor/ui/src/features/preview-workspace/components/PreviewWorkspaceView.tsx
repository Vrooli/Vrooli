import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import type { CSSProperties, PointerEvent as ReactPointerEvent } from 'react';
import clsx from 'clsx';
import { useSearchParams } from 'react-router-dom';
import { useAppsStore } from '@/state/appsStore';
import { isAppMonitorScenarioId } from '@/utils/appPreview';
import {
  previewWorkspaceLimits,
  usePreviewWorkspaceStore,
} from '../state/previewWorkspaceStore';
import {
  buildWorkspaceMinimapRowMarkers,
  buildGridTrackTemplate,
  reconcileTrackFractions,
  resolveDropIndex,
  resolveWorkspaceLayoutWithMaxColumns,
  scrollTopFromWorkspaceMinimapPointer,
  workspaceViewportFromScrollMetrics,
} from '../utils/layout';
import { clearWorkspaceIntent, readWorkspaceIntent } from '../utils/navigationIntent';
import PreviewPane from './PreviewPane';
import './PreviewWorkspaceView.css';

const SPLITTER_SIZE = 8;
const MIN_COLUMN_PX = 240;
const MIN_ROW_PX = 240;
const PINNED_COLUMN_MIN_FRACTION = 0.32;
const MOBILE_BREAKPOINT = 960;
const PIN_ZONE_THRESHOLD = 0.16;
const MINIMAP_MIN_OVERFLOW_PX = 8;

type WorkspaceScrollMetrics = {
  scrollTop: number;
  scrollHeight: number;
  clientHeight: number;
};

type ActiveResize = {
  axis: 'column' | 'row';
  scope: 'normal' | 'pinned';
  index: number;
  startPointer: number;
  containerSize: number;
  startValues: number[];
};

type WorkspaceDropIntent = 'reorder' | 'pin-left' | 'pin-right' | 'scroll-column';

type ActiveArrangeDrag = {
  paneId: string;
  dropIndex: number;
  intent: WorkspaceDropIntent;
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

const updateAdjacentFractions = ({
  startValues,
  index,
  delta,
  containerSize,
  splitterCount,
  minTrackPx,
}: {
  startValues: number[];
  index: number;
  delta: number;
  containerSize: number;
  splitterCount: number;
  minTrackPx: number;
}): number[] => {
  if (index < 0 || index >= startValues.length - 1) {
    return startValues;
  }

  const usableSize = Math.max(1, containerSize - (splitterCount * SPLITTER_SIZE));
  const current = startValues[index] ?? 0;
  const next = startValues[index + 1] ?? 0;
  const pairFraction = current + next;
  const pairSize = pairFraction * usableSize;

  if (pairSize <= 0) {
    return startValues;
  }

  const boundedMinTrackPx = Math.max(48, Math.min(minTrackPx, (pairSize / 2) - 1));
  if (!Number.isFinite(boundedMinTrackPx) || boundedMinTrackPx <= 0) {
    return startValues;
  }

  const currentPx = current * usableSize;
  const nextCurrentPx = clamp(currentPx + delta, boundedMinTrackPx, pairSize - boundedMinTrackPx);
  const nextFraction = nextCurrentPx / usableSize;
  const pairedFraction = (pairSize - nextCurrentPx) / usableSize;

  const updated = [...startValues];
  updated[index] = nextFraction;
  updated[index + 1] = pairedFraction;
  return updated;
};

const isMobileViewport = (): boolean => {
  if (typeof window === 'undefined') {
    return false;
  }
  return window.innerWidth <= MOBILE_BREAKPOINT;
};

const resolvePinnedColumnFractions = (fractions: number[]): [number, number] => {
  const normalized = reconcileTrackFractions(fractions, 2);
  const primary = clamp(normalized[0] ?? 0.5, PINNED_COLUMN_MIN_FRACTION, 1 - PINNED_COLUMN_MIN_FRACTION);
  return [primary, 1 - primary];
};

export default function PreviewWorkspaceView() {
  const [searchParams, setSearchParams] = useSearchParams();
  const apps = useAppsStore((state) => state.apps);
  const loadApps = useAppsStore((state) => state.loadApps);
  const loadingInitial = useAppsStore((state) => state.loadingInitial);
  const hasInitialized = useAppsStore((state) => state.hasInitialized);
  const [mobileLayout, setMobileLayout] = useState<boolean>(isMobileViewport);
  const [pinnedRowFractions, setPinnedRowFractions] = useState<number[]>([1]);

  const panes = usePreviewWorkspaceStore((state) => state.panes);
  const interactionMode = usePreviewWorkspaceStore((state) => state.interactionMode);
  const isWorkspaceMinimapVisible = usePreviewWorkspaceStore((state) => state.isWorkspaceMinimapVisible);
  const focusedPaneId = usePreviewWorkspaceStore((state) => state.focusedPaneId);
  const pinnedPaneId = usePreviewWorkspaceStore((state) => state.pinnedPaneId);
  const pinnedColumn = usePreviewWorkspaceStore((state) => state.pinnedColumn);
  const columnFractions = usePreviewWorkspaceStore((state) => state.columnFractions);
  const rowFractions = usePreviewWorkspaceStore((state) => state.rowFractions);
  const addPane = usePreviewWorkspaceStore((state) => state.addPane);
  const removePane = usePreviewWorkspaceStore((state) => state.removePane);
  const movePaneToIndex = usePreviewWorkspaceStore((state) => state.movePaneToIndex);
  const setPaneApp = usePreviewWorkspaceStore((state) => state.setPaneApp);
  const focusPane = usePreviewWorkspaceStore((state) => state.focusPane);
  const pinPaneToColumn = usePreviewWorkspaceStore((state) => state.pinPaneToColumn);
  const clearPinnedPane = usePreviewWorkspaceStore((state) => state.clearPinnedPane);
  const setColumnFractions = usePreviewWorkspaceStore((state) => state.setColumnFractions);
  const setRowFractions = usePreviewWorkspaceStore((state) => state.setRowFractions);
  const setPaneViewState = usePreviewWorkspaceStore((state) => state.setPaneViewState);

  const panesContainerRef = useRef<HTMLDivElement | null>(null);
  const panesScrollRef = useRef<HTMLDivElement | null>(null);
  const pinnedScrollColumnRef = useRef<HTMLDivElement | null>(null);
  const minimapRailRef = useRef<HTMLDivElement | null>(null);
  const lastHandledIntentRef = useRef<string | null>(null);
  const [activeResize, setActiveResize] = useState<ActiveResize | null>(null);
  const [activeArrangeDrag, setActiveArrangeDrag] = useState<ActiveArrangeDrag | null>(null);
  const [workspaceScrollMetrics, setWorkspaceScrollMetrics] = useState<WorkspaceScrollMetrics>({
    scrollTop: 0,
    scrollHeight: 1,
    clientHeight: 1,
  });

  const maxColumns = mobileLayout ? 1 : 2;
  const layoutDescriptor = useMemo(() => {
    return resolveWorkspaceLayoutWithMaxColumns(panes.length, maxColumns);
  }, [maxColumns, panes.length]);
  const effectiveColumnFractions = useMemo(() => {
    return reconcileTrackFractions(columnFractions, layoutDescriptor.columns);
  }, [columnFractions, layoutDescriptor.columns]);
  const effectiveRowFractions = useMemo(() => {
    return reconcileTrackFractions(rowFractions, layoutDescriptor.rows);
  }, [layoutDescriptor.rows, rowFractions]);
  const pinnedColumnFractions = useMemo<[number, number]>(() => {
    return resolvePinnedColumnFractions(columnFractions);
  }, [columnFractions]);
  const pinnedPane = useMemo(() => {
    if (!pinnedPaneId || maxColumns < 2) {
      return null;
    }
    return panes.find((pane) => pane.id === pinnedPaneId) ?? null;
  }, [maxColumns, panes, pinnedPaneId]);
  const scrollColumnPanes = useMemo(() => {
    if (!pinnedPane) {
      return panes;
    }
    return panes.filter((pane) => pane.id !== pinnedPane.id);
  }, [panes, pinnedPane]);
  const isPinnedLayout = Boolean(pinnedPane && pinnedColumn && maxColumns >= 2);
  const effectivePinnedRowFractions = useMemo(() => {
    return reconcileTrackFractions(pinnedRowFractions, scrollColumnPanes.length || 1);
  }, [pinnedRowFractions, scrollColumnPanes.length]);
  const dragDropTargetPaneId = useMemo(() => {
    if (!activeArrangeDrag || activeArrangeDrag.intent === 'pin-left' || activeArrangeDrag.intent === 'pin-right') {
      return null;
    }
    return panes[activeArrangeDrag.dropIndex]?.id ?? null;
  }, [activeArrangeDrag, panes]);
  useEffect(() => {
    if (!hasInitialized && !loadingInitial) {
      void loadApps();
    }
  }, [hasInitialized, loadApps, loadingInitial]);

  useEffect(() => {
    if (typeof window === 'undefined') {
      return;
    }

    const onResize = () => {
      setMobileLayout(isMobileViewport());
    };
    onResize();
    window.addEventListener('resize', onResize);
    return () => {
      window.removeEventListener('resize', onResize);
    };
  }, []);

  useEffect(() => {
    if (maxColumns > 1) {
      return;
    }
    if (pinnedPaneId) {
      clearPinnedPane();
    }
  }, [clearPinnedPane, maxColumns, pinnedPaneId]);

  useEffect(() => {
    setPinnedRowFractions((current) => reconcileTrackFractions(current, scrollColumnPanes.length || 1));
  }, [scrollColumnPanes.length]);

  useEffect(() => {
    const intent = readWorkspaceIntent(searchParams);
    const shouldOpenLogs = searchParams.get('paneLogs') === '1';
    if (!intent) {
      if (shouldOpenLogs) {
        const targetPaneId = focusedPaneId ?? panes[0]?.id ?? null;
        if (targetPaneId) {
          setPaneViewState(targetPaneId, { isLogsVisible: true });
        }
        const nextParams = new URLSearchParams(searchParams);
        nextParams.delete('paneLogs');
        setSearchParams(nextParams, { replace: true });
      }
      lastHandledIntentRef.current = null;
      return;
    }

    const intentSignature = `${intent.mode}:${intent.appId}:${searchParams.toString()}`;
    if (lastHandledIntentRef.current === intentSignature) {
      return;
    }
    lastHandledIntentRef.current = intentSignature;

    if (isAppMonitorScenarioId(intent.appId)) {
      if (shouldOpenLogs) {
        const targetPaneId = focusedPaneId ?? panes[0]?.id ?? null;
        if (targetPaneId) {
          setPaneViewState(targetPaneId, { isLogsVisible: true });
        }
      }

      const nextParams = clearWorkspaceIntent(searchParams);
      nextParams.delete('paneLogs');
      setSearchParams(nextParams, { replace: true });
      return;
    }

    let nextFocusedPaneId: string | null = null;
    if (intent.mode === 'add-pane') {
      nextFocusedPaneId = addPane(intent.appId);
    } else {
      const targetPaneId = focusedPaneId ?? panes[0]?.id ?? null;
      if (targetPaneId) {
        setPaneApp(targetPaneId, intent.appId);
        focusPane(targetPaneId);
        nextFocusedPaneId = targetPaneId;
      }
    }

    if (shouldOpenLogs && nextFocusedPaneId) {
      setPaneViewState(nextFocusedPaneId, { isLogsVisible: true });
    }

    const nextParams = clearWorkspaceIntent(searchParams);
    nextParams.delete('paneLogs');
    setSearchParams(nextParams, { replace: true });
  }, [addPane, focusPane, focusedPaneId, panes, searchParams, setPaneApp, setPaneViewState, setSearchParams]);

  useEffect(() => {
    if (!activeResize) {
      return;
    }

    const handlePointerMove = (event: PointerEvent) => {
      const delta = (activeResize.axis === 'column' ? event.clientX : event.clientY) - activeResize.startPointer;

      if (activeResize.axis === 'column') {
        const nextFractions = updateAdjacentFractions({
          startValues: activeResize.startValues,
          index: activeResize.index,
          delta,
          containerSize: activeResize.containerSize,
          splitterCount: Math.max(0, activeResize.startValues.length - 1),
          minTrackPx: MIN_COLUMN_PX,
        });
        setColumnFractions(nextFractions);
        return;
      }

      const nextFractions = updateAdjacentFractions({
        startValues: activeResize.startValues,
        index: activeResize.index,
        delta,
        containerSize: activeResize.containerSize,
        splitterCount: Math.max(0, activeResize.startValues.length - 1),
        minTrackPx: MIN_ROW_PX,
      });
      if (activeResize.scope === 'pinned') {
        setPinnedRowFractions(nextFractions);
      } else {
        setRowFractions(nextFractions);
      }
    };

    const stopResize = () => {
      setActiveResize(null);
    };

    window.addEventListener('pointermove', handlePointerMove);
    window.addEventListener('pointerup', stopResize);
    window.addEventListener('pointercancel', stopResize);

    return () => {
      window.removeEventListener('pointermove', handlePointerMove);
      window.removeEventListener('pointerup', stopResize);
      window.removeEventListener('pointercancel', stopResize);
    };
  }, [activeResize, setColumnFractions, setRowFractions]);

  useEffect(() => {
    if (!activeArrangeDrag) {
      return;
    }

    const handlePointerMove = (event: PointerEvent) => {
      const containerNode = panesContainerRef.current;
      if (!containerNode) {
        return;
      }
      const rect = containerNode.getBoundingClientRect();
      const xRatio = clamp((event.clientX - rect.left) / Math.max(1, rect.width), 0, 1);
      const isLeftPinZone = maxColumns > 1 && xRatio <= PIN_ZONE_THRESHOLD;
      const isRightPinZone = maxColumns > 1 && xRatio >= (1 - PIN_ZONE_THRESHOLD);
      const isPinnedScrollZone = isPinnedLayout && !isLeftPinZone && !isRightPinZone;
      const pointerTarget = typeof document.elementFromPoint === 'function'
        ? document.elementFromPoint(event.clientX, event.clientY)
        : null;

      if (isLeftPinZone) {
        setActiveArrangeDrag((current) => {
          if (!current) {
            return null;
          }
          return {
            ...current,
            intent: 'pin-left',
          };
        });
        return;
      }

      if (isRightPinZone) {
        setActiveArrangeDrag((current) => {
          if (!current) {
            return null;
          }
          return {
            ...current,
            intent: 'pin-right',
          };
        });
        return;
      }

      if (isPinnedScrollZone) {
        const stackSlot = pointerTarget?.closest('[data-stack-pane-id]');
        const stackPaneId = stackSlot?.getAttribute('data-stack-pane-id')?.trim() ?? '';
        const hoveredPaneIndex = stackPaneId.length > 0
          ? panes.findIndex((pane) => pane.id === stackPaneId)
          : -1;
        const targetIndex = hoveredPaneIndex >= 0
          ? hoveredPaneIndex
          : scrollColumnPanes.length <= 0
            ? 0
            : panes.findIndex((pane) => pane.id === scrollColumnPanes[Math.min(
              scrollColumnPanes.length - 1,
              Math.max(0, Math.floor(clamp((event.clientY - rect.top) / Math.max(1, rect.height), 0, 0.9999) * scrollColumnPanes.length)),
            )]?.id);

        setActiveArrangeDrag((current) => {
          if (!current) {
            return null;
          }
          return {
            ...current,
            intent: 'scroll-column',
            dropIndex: Math.max(0, targetIndex),
          };
        });
        return;
      }

      const paneSlot = pointerTarget?.closest('[data-pane-index]');
      const paneIndexRaw = paneSlot?.getAttribute('data-pane-index');
      const hoveredPaneIndex = paneIndexRaw ? Number.parseInt(paneIndexRaw, 10) : Number.NaN;
      if (Number.isFinite(hoveredPaneIndex) && hoveredPaneIndex >= 0 && hoveredPaneIndex < panes.length) {
        setActiveArrangeDrag((current) => {
          if (!current) {
            return null;
          }
          return {
            ...current,
            intent: 'reorder',
            dropIndex: hoveredPaneIndex,
          };
        });
        return;
      }

      const dropIndex = resolveDropIndex({
        pointerX: event.clientX,
        pointerY: event.clientY,
        rect,
        columns: layoutDescriptor.columns,
        rows: layoutDescriptor.rows,
        paneCount: panes.length,
      });

      setActiveArrangeDrag((current) => {
        if (!current) {
          return null;
        }
        return {
          ...current,
          intent: 'reorder',
          dropIndex,
        };
      });
    };

    const stopDragging = () => {
      setActiveArrangeDrag((current) => {
        if (!current) {
          return null;
        }

        if (current.intent === 'pin-left') {
          pinPaneToColumn(current.paneId, 'left');
          return null;
        }

        if (current.intent === 'pin-right') {
          pinPaneToColumn(current.paneId, 'right');
          return null;
        }

        if (current.intent === 'scroll-column') {
          clearPinnedPane();
          movePaneToIndex(current.paneId, current.dropIndex);
          return null;
        }

        movePaneToIndex(current.paneId, current.dropIndex);
        return null;
      });
    };

    window.addEventListener('pointermove', handlePointerMove);
    window.addEventListener('pointerup', stopDragging);
    window.addEventListener('pointercancel', stopDragging);

    return () => {
      window.removeEventListener('pointermove', handlePointerMove);
      window.removeEventListener('pointerup', stopDragging);
      window.removeEventListener('pointercancel', stopDragging);
    };
  }, [
    activeArrangeDrag,
    clearPinnedPane,
    isPinnedLayout,
    layoutDescriptor.columns,
    layoutDescriptor.rows,
    maxColumns,
    movePaneToIndex,
    panes,
    pinPaneToColumn,
    scrollColumnPanes,
  ]);

  const startColumnResize = useCallback((index: number, event: ReactPointerEvent<HTMLButtonElement>) => {
    const containerNode = panesContainerRef.current;
    if (!containerNode) {
      return;
    }

    event.preventDefault();
    event.currentTarget.setPointerCapture(event.pointerId);
    setActiveResize({
      axis: 'column',
      scope: isPinnedLayout ? 'pinned' : 'normal',
      index,
      startPointer: event.clientX,
      containerSize: containerNode.clientWidth,
      startValues: isPinnedLayout ? pinnedColumnFractions : effectiveColumnFractions,
    });
  }, [effectiveColumnFractions, isPinnedLayout, pinnedColumnFractions]);

  const startRowResize = useCallback((index: number, scope: 'normal' | 'pinned', event: ReactPointerEvent<HTMLButtonElement>) => {
    const containerNode = panesContainerRef.current;
    if (!containerNode) {
      return;
    }

    event.preventDefault();
    event.currentTarget.setPointerCapture(event.pointerId);
    setActiveResize({
      axis: 'row',
      scope,
      index,
      startPointer: event.clientY,
      containerSize: containerNode.clientHeight,
      startValues: scope === 'pinned' ? effectivePinnedRowFractions : effectiveRowFractions,
    });
  }, [effectivePinnedRowFractions, effectiveRowFractions]);

  const startArrangeDrag = useCallback((paneId: string, event: ReactPointerEvent<HTMLButtonElement>) => {
    if (interactionMode !== 'arrange') {
      return;
    }

    const paneIndex = panes.findIndex((pane) => pane.id === paneId);
    if (paneIndex < 0) {
      return;
    }

    event.preventDefault();
    event.stopPropagation();
    event.currentTarget.setPointerCapture(event.pointerId);

    setActiveArrangeDrag({
      paneId,
      dropIndex: paneIndex,
      intent: 'reorder',
    });
  }, [interactionMode, panes]);

  const paneSlotStyles = useMemo(() => {
    return panes.map((_, index) => {
      const row = Math.floor(index / layoutDescriptor.columns);
      const column = index % layoutDescriptor.columns;
      return {
        gridColumn: `${(column * 2) + 1}`,
        gridRow: `${(row * 2) + 1}`,
      } satisfies CSSProperties;
    });
  }, [layoutDescriptor.columns, panes]);
  const scrollColumnSlotStyles = useMemo(() => {
    return scrollColumnPanes.map((_, index) => ({
      gridColumn: '1',
      gridRow: `${(index * 2) + 1}`,
    } satisfies CSSProperties));
  }, [scrollColumnPanes]);

  const rowLineEnd = (layoutDescriptor.rows * 2).toString();
  const columnLineEnd = (layoutDescriptor.columns * 2).toString();
  const viewportPaneHeight = typeof window === 'undefined'
    ? MIN_ROW_PX
    : Math.max(MIN_ROW_PX, window.innerHeight);
  const rowSplittersHeight = Math.max(0, layoutDescriptor.rows - 1) * SPLITTER_SIZE;
  // Critical: scale grid min-height by row count so each row can grow up to viewport height.
  // Without this, fr tracks are forced to share a single-viewport container and panes shrink
  // as rows are added instead of extending the scrollable surface.
  const minimumGridHeightPx = (viewportPaneHeight * layoutDescriptor.rows) + rowSplittersHeight;
  const minimapRowCount = useMemo(
    () => (isPinnedLayout ? scrollColumnPanes.length : layoutDescriptor.rows),
    [isPinnedLayout, layoutDescriptor.rows, scrollColumnPanes.length],
  );
  const minimapRowMarkers = useMemo(
    () => buildWorkspaceMinimapRowMarkers(minimapRowCount),
    [minimapRowCount],
  );
  const minimapViewport = useMemo(() => workspaceViewportFromScrollMetrics({
    scrollTop: workspaceScrollMetrics.scrollTop,
    scrollHeight: workspaceScrollMetrics.scrollHeight,
    clientHeight: workspaceScrollMetrics.clientHeight,
  }), [workspaceScrollMetrics.clientHeight, workspaceScrollMetrics.scrollHeight, workspaceScrollMetrics.scrollTop]);
  const hasScrollableOverflow = minimapViewport.maxScrollable > MINIMAP_MIN_OVERFLOW_PX;
  const showWorkspaceMinimap = isWorkspaceMinimapVisible && hasScrollableOverflow;
  const activeViewportRowIndex = useMemo(() => {
    if (minimapRowCount <= 0) {
      return -1;
    }
    const rowHeightPercent = 100 / minimapRowCount;
    const viewportCenterPercent = minimapViewport.topPercent + (minimapViewport.heightPercent / 2);
    const clampedCenter = clamp(viewportCenterPercent, 0, 99.9999);
    return Math.min(minimapRowCount - 1, Math.max(0, Math.floor(clampedCenter / rowHeightPercent)));
  }, [minimapRowCount, minimapViewport.heightPercent, minimapViewport.topPercent]);
  const resolveWorkspaceScrollNode = useCallback(() => {
    const preferredNode = isPinnedLayout ? pinnedScrollColumnRef.current : panesScrollRef.current;
    if (
      preferredNode
      && (preferredNode.scrollHeight - preferredNode.clientHeight) > MINIMAP_MIN_OVERFLOW_PX
    ) {
      return preferredNode;
    }

    if (typeof document !== 'undefined') {
      const rootScrollNode = document.scrollingElement as HTMLElement | null;
      if (
        rootScrollNode
        && (rootScrollNode.scrollHeight - rootScrollNode.clientHeight) > MINIMAP_MIN_OVERFLOW_PX
      ) {
        return rootScrollNode;
      }
    }

    return preferredNode;
  }, [isPinnedLayout]);

  useEffect(() => {
    const scrollNode = resolveWorkspaceScrollNode();
    if (!scrollNode || !isWorkspaceMinimapVisible) {
      setWorkspaceScrollMetrics({
        scrollTop: 0,
        scrollHeight: 1,
        clientHeight: 1,
      });
      return;
    }

    const updateMetrics = () => {
      const isRootScrollNode = typeof document !== 'undefined' && scrollNode === document.scrollingElement;
      const nextScrollTop = isRootScrollNode && typeof window !== 'undefined'
        ? window.scrollY
        : scrollNode.scrollTop;
      const nextClientHeight = isRootScrollNode && typeof window !== 'undefined'
        ? window.innerHeight
        : scrollNode.clientHeight;
      setWorkspaceScrollMetrics({
        scrollTop: nextScrollTop,
        scrollHeight: Math.max(1, scrollNode.scrollHeight),
        clientHeight: Math.max(1, nextClientHeight),
      });
    };

    updateMetrics();
    scrollNode.addEventListener('scroll', updateMetrics, { passive: true });
    window.addEventListener('scroll', updateMetrics, { passive: true, capture: true });
    window.addEventListener('resize', updateMetrics);

    return () => {
      scrollNode.removeEventListener('scroll', updateMetrics);
      window.removeEventListener('scroll', updateMetrics, { capture: true });
      window.removeEventListener('resize', updateMetrics);
    };
  }, [
    isPinnedLayout,
    isWorkspaceMinimapVisible,
    layoutDescriptor.columns,
    layoutDescriptor.rows,
    minimumGridHeightPx,
    panes.length,
    resolveWorkspaceScrollNode,
    scrollColumnPanes.length,
  ]);

  const jumpToWorkspaceMinimapPosition = useCallback((clientY: number) => {
    const rail = minimapRailRef.current;
    const scrollNode = resolveWorkspaceScrollNode();
    if (!rail || !scrollNode) {
      return;
    }

    const rect = rail.getBoundingClientRect();
    const pointerOffsetY = clamp(clientY - rect.top, 0, rect.height);
    const nextScrollTop = scrollTopFromWorkspaceMinimapPointer(
      pointerOffsetY,
      rect.height,
      scrollNode.scrollHeight,
      scrollNode.clientHeight,
    );
    scrollNode.scrollTo({ top: nextScrollTop, behavior: 'auto' });
  }, [resolveWorkspaceScrollNode]);

  const handleWorkspaceMinimapPointerDown = useCallback((event: ReactPointerEvent<HTMLDivElement>) => {
    event.preventDefault();
    jumpToWorkspaceMinimapPosition(event.clientY);

    const handleMove = (moveEvent: PointerEvent) => {
      jumpToWorkspaceMinimapPosition(moveEvent.clientY);
    };

    const handleUp = () => {
      window.removeEventListener('pointermove', handleMove);
      window.removeEventListener('pointerup', handleUp);
      window.removeEventListener('pointercancel', handleUp);
    };

    window.addEventListener('pointermove', handleMove);
    window.addEventListener('pointerup', handleUp);
    window.addEventListener('pointercancel', handleUp);
  }, [jumpToWorkspaceMinimapPosition]);

  const handleWorkspaceMinimapKeyDown = useCallback((event: React.KeyboardEvent<HTMLDivElement>) => {
    const scrollNode = resolveWorkspaceScrollNode();
    if (!scrollNode) {
      return;
    }

    if (event.key === 'ArrowDown' || event.key === 'PageDown') {
      event.preventDefault();
      const step = event.key === 'PageDown' ? scrollNode.clientHeight : 120;
      scrollNode.scrollTo({ top: scrollNode.scrollTop + step, behavior: 'auto' });
      return;
    }
    if (event.key === 'ArrowUp' || event.key === 'PageUp') {
      event.preventDefault();
      const step = event.key === 'PageUp' ? scrollNode.clientHeight : 120;
      scrollNode.scrollTo({ top: scrollNode.scrollTop - step, behavior: 'auto' });
      return;
    }
    if (event.key === 'Home') {
      event.preventDefault();
      scrollNode.scrollTo({ top: 0, behavior: 'auto' });
      return;
    }
    if (event.key === 'End') {
      event.preventDefault();
      scrollNode.scrollTo({ top: scrollNode.scrollHeight, behavior: 'auto' });
    }
  }, [resolveWorkspaceScrollNode]);
  const renderPane = (paneId: string, appId: string | null, isBeingDragged: boolean) => (
    <PreviewPane
      paneId={paneId}
      appId={appId}
      apps={apps}
      isFocused={focusedPaneId === paneId}
      canRemove={panes.length > previewWorkspaceLimits.minPanes}
      isArrangeMode={interactionMode === 'arrange'}
      isBeingDragged={isBeingDragged}
      onFocus={focusPane}
      onRemove={removePane}
      onArrangeDragStart={startArrangeDrag}
    />
  );

  return (
    <div
      className={clsx(
        'preview-workspace',
        showWorkspaceMinimap && 'preview-workspace--with-minimap',
      )}
    >
      {isPinnedLayout && pinnedPane ? (
        <div
          ref={panesContainerRef}
          className={clsx(
            'preview-workspace__pinned-layout',
            activeArrangeDrag && 'preview-workspace__pinned-layout--dragging',
            activeArrangeDrag?.intent === 'pin-left' && 'preview-workspace__pinned-layout--pin-left-target',
            activeArrangeDrag?.intent === 'pin-right' && 'preview-workspace__pinned-layout--pin-right-target',
            activeArrangeDrag?.intent === 'scroll-column' && 'preview-workspace__pinned-layout--scroll-target',
          )}
          style={{
            gridTemplateColumns: buildGridTrackTemplate(pinnedColumnFractions, SPLITTER_SIZE),
          }}
        >
          {pinnedColumn === 'right' ? (
            <>
              <div
                className="preview-workspace__scroll-column"
                ref={pinnedScrollColumnRef}
                style={{
                  gridTemplateRows: buildGridTrackTemplate(effectivePinnedRowFractions, SPLITTER_SIZE),
                }}
              >
                {scrollColumnPanes.map((pane, index) => {
                  const isDragging = activeArrangeDrag?.paneId === pane.id;
                  const isDropTarget = dragDropTargetPaneId === pane.id && !isDragging;
                  return (
                    <div
                      key={pane.id}
                      className={clsx(
                        'preview-workspace__stack-slot',
                        isDragging && 'preview-workspace__pane-slot--dragging',
                        isDropTarget && 'preview-workspace__pane-slot--drop-target',
                      )}
                      data-stack-pane-id={pane.id}
                      style={scrollColumnSlotStyles[index]}
                    >
                      {renderPane(pane.id, pane.appId, isDragging)}
                    </div>
                  );
                })}
                {Array.from({ length: Math.max(0, scrollColumnPanes.length - 1) }).map((_, index) => (
                  <button
                    key={`pinned-right-row-split-${index}`}
                    type="button"
                    className="preview-workspace__splitter preview-workspace__splitter--row"
                    style={{
                      gridRow: `${(index * 2) + 2}`,
                      gridColumn: `1 / 2`,
                    }}
                    onPointerDown={(event) => startRowResize(index, 'pinned', event)}
                    aria-label={`Resize row ${index + 1}`}
                  />
                ))}
              </div>
              <div className="preview-workspace__pinned-column">
                {renderPane(pinnedPane.id, pinnedPane.appId, activeArrangeDrag?.paneId === pinnedPane.id)}
              </div>
            </>
          ) : (
            <>
              <div className="preview-workspace__pinned-column">
                {renderPane(pinnedPane.id, pinnedPane.appId, activeArrangeDrag?.paneId === pinnedPane.id)}
              </div>
              <div
                className="preview-workspace__scroll-column"
                ref={pinnedScrollColumnRef}
                style={{
                  gridTemplateRows: buildGridTrackTemplate(effectivePinnedRowFractions, SPLITTER_SIZE),
                }}
              >
                {scrollColumnPanes.map((pane, index) => {
                  const isDragging = activeArrangeDrag?.paneId === pane.id;
                  const isDropTarget = dragDropTargetPaneId === pane.id && !isDragging;
                  return (
                    <div
                      key={pane.id}
                      className={clsx(
                        'preview-workspace__stack-slot',
                        isDragging && 'preview-workspace__pane-slot--dragging',
                        isDropTarget && 'preview-workspace__pane-slot--drop-target',
                      )}
                      data-stack-pane-id={pane.id}
                      style={scrollColumnSlotStyles[index]}
                    >
                      {renderPane(pane.id, pane.appId, isDragging)}
                    </div>
                  );
                })}
                {Array.from({ length: Math.max(0, scrollColumnPanes.length - 1) }).map((_, index) => (
                  <button
                    key={`pinned-left-row-split-${index}`}
                    type="button"
                    className="preview-workspace__splitter preview-workspace__splitter--row"
                    style={{
                      gridRow: `${(index * 2) + 2}`,
                      gridColumn: `1 / 2`,
                    }}
                    onPointerDown={(event) => startRowResize(index, 'pinned', event)}
                    aria-label={`Resize row ${index + 1}`}
                  />
                ))}
              </div>
            </>
          )}

          <button
            type="button"
            className="preview-workspace__splitter preview-workspace__splitter--column"
            style={{
              gridColumn: '2',
              gridRow: '1 / 2',
            }}
            onPointerDown={(event) => startColumnResize(0, event)}
            aria-label="Resize column 1"
          />
        </div>
      ) : (
        <div className="preview-workspace__panes-scroll" ref={panesScrollRef}>
          <div
            ref={panesContainerRef}
            className={clsx(
              'preview-workspace__panes',
              layoutDescriptor.className,
              interactionMode === 'arrange' && 'preview-workspace__panes--arrange-mode',
              activeResize && 'preview-workspace__panes--resizing',
              activeArrangeDrag && 'preview-workspace__panes--dragging',
              activeArrangeDrag?.intent === 'pin-left' && 'preview-workspace__panes--pin-left-target',
              activeArrangeDrag?.intent === 'pin-right' && 'preview-workspace__panes--pin-right-target',
            )}
            style={{
              gridTemplateColumns: buildGridTrackTemplate(effectiveColumnFractions, SPLITTER_SIZE),
              gridTemplateRows: buildGridTrackTemplate(effectiveRowFractions, SPLITTER_SIZE),
              // Keep track sizing deterministic during loading/fallback transitions.
              // A fixed computed grid height prevents intrinsic pane content from temporarily
              // expanding rows beyond the persisted fractions.
              height: `${minimumGridHeightPx}px`,
              minHeight: `${minimumGridHeightPx}px`,
            }}
          >
            {panes.map((pane, index) => {
              const isDragging = activeArrangeDrag?.paneId === pane.id;
              const isDropTarget = activeArrangeDrag?.dropIndex === index && !isDragging;

              return (
                <div
                  key={pane.id}
                  className={clsx(
                    'preview-workspace__pane-slot',
                    isDragging && 'preview-workspace__pane-slot--dragging',
                    isDropTarget && activeArrangeDrag?.intent === 'reorder' && 'preview-workspace__pane-slot--drop-target',
                  )}
                  style={paneSlotStyles[index]}
                  data-pane-index={index}
                >
                  {renderPane(pane.id, pane.appId, isDragging)}
                </div>
              );
            })}

            {Array.from({ length: Math.max(0, layoutDescriptor.columns - 1) }).map((_, index) => (
              <button
                key={`column-split-${index}`}
                type="button"
                className="preview-workspace__splitter preview-workspace__splitter--column"
                style={{
                  gridColumn: `${(index * 2) + 2}`,
                  gridRow: `1 / ${rowLineEnd}`,
                }}
                onPointerDown={(event) => startColumnResize(index, event)}
                aria-label={`Resize column ${index + 1}`}
              />
            ))}

            {Array.from({ length: Math.max(0, layoutDescriptor.rows - 1) }).map((_, index) => (
              <button
                key={`row-split-${index}`}
                type="button"
                className="preview-workspace__splitter preview-workspace__splitter--row"
                style={{
                  gridRow: `${(index * 2) + 2}`,
                  gridColumn: `1 / ${columnLineEnd}`,
                }}
                onPointerDown={(event) => startRowResize(index, 'normal', event)}
                aria-label={`Resize row ${index + 1}`}
              />
            ))}
          </div>
        </div>
      )}
      {showWorkspaceMinimap && (
        <aside className="preview-workspace__minimap" data-testid="workspace-minimap">
          <div
            ref={minimapRailRef}
            className="preview-workspace__minimap-rail"
            role="slider"
            tabIndex={0}
            aria-label="Workspace minimap"
            aria-valuemin={0}
            aria-valuemax={Math.round(minimapViewport.maxScrollable)}
            aria-valuenow={Math.round(workspaceScrollMetrics.scrollTop)}
            data-testid="workspace-minimap-rail"
            onPointerDown={handleWorkspaceMinimapPointerDown}
            onKeyDown={handleWorkspaceMinimapKeyDown}
          >
            <div className="preview-workspace__minimap-markers" aria-hidden="true">
              {minimapRowMarkers.map((marker) => (
                <div
                  key={`row-${marker.rowIndex}`}
                  className={clsx(
                    'preview-workspace__minimap-marker',
                    marker.rowIndex === activeViewportRowIndex && 'preview-workspace__minimap-marker--active',
                  )}
                  style={{
                    top: `${marker.topPercent}%`,
                    height: `${marker.heightPercent}%`,
                  }}
                />
              ))}
            </div>
            <div
              className="preview-workspace__minimap-viewport"
              data-testid="workspace-minimap-viewport"
              style={{
                top: `${minimapViewport.topPercent}%`,
                height: `${minimapViewport.heightPercent}%`,
              }}
            />
          </div>
        </aside>
      )}
    </div>
  );
}
