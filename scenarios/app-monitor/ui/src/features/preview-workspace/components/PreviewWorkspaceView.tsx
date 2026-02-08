import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import type { CSSProperties, PointerEvent as ReactPointerEvent } from 'react';
import clsx from 'clsx';
import { Command, Grip, Layers, Plus } from 'lucide-react';
import { useSearchParams } from 'react-router-dom';
import { useOverlayRouter } from '@/hooks/useOverlayRouter';
import { useAppsStore } from '@/state/appsStore';
import {
  previewWorkspaceLimits,
  usePreviewWorkspaceStore,
} from '../state/previewWorkspaceStore';
import {
  buildGridTrackTemplate,
  resolveDropIndex,
  resolveWorkspaceLayout,
} from '../utils/layout';
import { clearWorkspaceIntent, readWorkspaceIntent } from '../utils/navigationIntent';
import PreviewPane from './PreviewPane';
import './PreviewWorkspaceView.css';

const SPLITTER_SIZE = 8;
const MIN_COLUMN_PX = 240;
const MIN_ROW_PX = 180;

type ActiveResize = {
  axis: 'column' | 'row';
  index: number;
  startPointer: number;
  containerSize: number;
  startFractions: number[];
};

type ActiveArrangeDrag = {
  paneId: string;
  dropIndex: number;
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
  startFractions,
  index,
  delta,
  containerSize,
  splitterCount,
  minTrackPx,
}: {
  startFractions: number[];
  index: number;
  delta: number;
  containerSize: number;
  splitterCount: number;
  minTrackPx: number;
}): number[] => {
  if (index < 0 || index >= startFractions.length - 1) {
    return startFractions;
  }

  const usableSize = Math.max(1, containerSize - (splitterCount * SPLITTER_SIZE));
  const current = startFractions[index] ?? 0;
  const next = startFractions[index + 1] ?? 0;
  const pairFraction = current + next;
  const pairSize = pairFraction * usableSize;

  if (pairSize <= 0) {
    return startFractions;
  }

  const boundedMinTrackPx = Math.max(48, Math.min(minTrackPx, (pairSize / 2) - 1));
  if (!Number.isFinite(boundedMinTrackPx) || boundedMinTrackPx <= 0) {
    return startFractions;
  }

  const currentPx = current * usableSize;
  const nextCurrentPx = clamp(currentPx + delta, boundedMinTrackPx, pairSize - boundedMinTrackPx);
  const nextFraction = nextCurrentPx / usableSize;
  const pairedFraction = (pairSize - nextCurrentPx) / usableSize;

  const updated = [...startFractions];
  updated[index] = nextFraction;
  updated[index + 1] = pairedFraction;
  return updated;
};

export default function PreviewWorkspaceView() {
  const [searchParams, setSearchParams] = useSearchParams();
  const { openOverlay } = useOverlayRouter();
  const apps = useAppsStore((state) => state.apps);

  const panes = usePreviewWorkspaceStore((state) => state.panes);
  const layout = usePreviewWorkspaceStore((state) => state.layout);
  const interactionMode = usePreviewWorkspaceStore((state) => state.interactionMode);
  const focusedPaneId = usePreviewWorkspaceStore((state) => state.focusedPaneId);
  const columnFractions = usePreviewWorkspaceStore((state) => state.columnFractions);
  const rowFractions = usePreviewWorkspaceStore((state) => state.rowFractions);
  const addPane = usePreviewWorkspaceStore((state) => state.addPane);
  const removePane = usePreviewWorkspaceStore((state) => state.removePane);
  const movePaneToIndex = usePreviewWorkspaceStore((state) => state.movePaneToIndex);
  const setPaneApp = usePreviewWorkspaceStore((state) => state.setPaneApp);
  const focusPane = usePreviewWorkspaceStore((state) => state.focusPane);
  const setInteractionMode = usePreviewWorkspaceStore((state) => state.setInteractionMode);
  const setColumnFractions = usePreviewWorkspaceStore((state) => state.setColumnFractions);
  const setRowFractions = usePreviewWorkspaceStore((state) => state.setRowFractions);

  const panesContainerRef = useRef<HTMLDivElement | null>(null);
  const lastHandledIntentRef = useRef<string | null>(null);
  const [activeResize, setActiveResize] = useState<ActiveResize | null>(null);
  const [activeArrangeDrag, setActiveArrangeDrag] = useState<ActiveArrangeDrag | null>(null);

  const layoutDescriptor = useMemo(() => {
    return resolveWorkspaceLayout(layout, panes.length);
  }, [layout, panes.length]);

  useEffect(() => {
    const intent = readWorkspaceIntent(searchParams);
    if (!intent) {
      lastHandledIntentRef.current = null;
      return;
    }

    const intentSignature = `${intent.mode}:${intent.appId}:${searchParams.toString()}`;
    if (lastHandledIntentRef.current === intentSignature) {
      return;
    }
    lastHandledIntentRef.current = intentSignature;

    if (intent.mode === 'add-pane') {
      addPane(intent.appId);
    } else {
      const targetPaneId = focusedPaneId ?? panes[0]?.id ?? null;
      if (targetPaneId) {
        setPaneApp(targetPaneId, intent.appId);
        focusPane(targetPaneId);
      }
    }

    setSearchParams(clearWorkspaceIntent(searchParams), { replace: true });
  }, [addPane, focusPane, focusedPaneId, panes, searchParams, setPaneApp, setSearchParams]);

  useEffect(() => {
    if (!activeResize) {
      return;
    }

    const handlePointerMove = (event: PointerEvent) => {
      const delta = (activeResize.axis === 'column' ? event.clientX : event.clientY) - activeResize.startPointer;

      if (activeResize.axis === 'column') {
        const nextFractions = updateAdjacentFractions({
          startFractions: activeResize.startFractions,
          index: activeResize.index,
          delta,
          containerSize: activeResize.containerSize,
          splitterCount: Math.max(0, activeResize.startFractions.length - 1),
          minTrackPx: MIN_COLUMN_PX,
        });
        setColumnFractions(nextFractions);
        return;
      }

      const nextFractions = updateAdjacentFractions({
        startFractions: activeResize.startFractions,
        index: activeResize.index,
        delta,
        containerSize: activeResize.containerSize,
        splitterCount: Math.max(0, activeResize.startFractions.length - 1),
        minTrackPx: MIN_ROW_PX,
      });
      setRowFractions(nextFractions);
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
          dropIndex,
        };
      });
    };

    const stopDragging = () => {
      setActiveArrangeDrag((current) => {
        if (!current) {
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
  }, [activeArrangeDrag, layoutDescriptor.columns, layoutDescriptor.rows, movePaneToIndex, panes.length]);

  const canAddPane = panes.length < previewWorkspaceLimits.maxPanes;

  const startColumnResize = useCallback((index: number, event: ReactPointerEvent<HTMLButtonElement>) => {
    const containerNode = panesContainerRef.current;
    if (!containerNode) {
      return;
    }

    event.preventDefault();
    event.currentTarget.setPointerCapture(event.pointerId);
    setActiveResize({
      axis: 'column',
      index,
      startPointer: event.clientX,
      containerSize: containerNode.clientWidth,
      startFractions: columnFractions,
    });
  }, [columnFractions]);

  const startRowResize = useCallback((index: number, event: ReactPointerEvent<HTMLButtonElement>) => {
    const containerNode = panesContainerRef.current;
    if (!containerNode) {
      return;
    }

    event.preventDefault();
    event.currentTarget.setPointerCapture(event.pointerId);
    setActiveResize({
      axis: 'row',
      index,
      startPointer: event.clientY,
      containerSize: containerNode.clientHeight,
      startFractions: rowFractions,
    });
  }, [rowFractions]);

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

  const rowLineEnd = (layoutDescriptor.rows * 2).toString();
  const columnLineEnd = (layoutDescriptor.columns * 2).toString();

  return (
    <div className="preview-workspace">
      <header className="preview-workspace__header">
        <div className="preview-workspace__header-title">Workspace</div>

        <div className="preview-workspace__actions">
          <button
            type="button"
            className="preview-workspace__header-btn"
            onClick={() => openOverlay('tabs', {
              params: { segment: 'apps' },
            })}
            aria-label="Open tab switcher"
            title="Open tab switcher"
          >
            <Layers size={16} aria-hidden />
            <span>Tabs</span>
          </button>

          <button
            type="button"
            className="preview-workspace__header-btn"
            onClick={() => openOverlay('actions')}
            aria-label="Open command center"
            title="Open command center"
          >
            <Command size={16} aria-hidden />
            <span>Command Center</span>
          </button>

          <button
            type="button"
            className="preview-workspace__header-btn"
            onClick={() => addPane(null)}
            disabled={!canAddPane}
          >
            <Plus size={16} aria-hidden />
            <span>{canAddPane ? 'Add Pane' : `Max ${previewWorkspaceLimits.maxPanes} panes`}</span>
          </button>

          <button
            type="button"
            className={clsx(
              'preview-workspace__header-btn',
              interactionMode === 'arrange' && 'preview-workspace__header-btn--active',
            )}
            onClick={() => setInteractionMode(interactionMode === 'arrange' ? 'browse' : 'arrange')}
            aria-pressed={interactionMode === 'arrange'}
            aria-label="Toggle pane arrange mode"
            title="Toggle pane arrange mode"
          >
            <Grip size={16} aria-hidden />
            <span>{interactionMode === 'arrange' ? 'Arrange On' : 'Arrange Off'}</span>
          </button>
        </div>
      </header>

      <div
        ref={panesContainerRef}
        className={clsx(
          'preview-workspace__panes',
          layoutDescriptor.className,
          interactionMode === 'arrange' && 'preview-workspace__panes--arrange-mode',
          activeResize && 'preview-workspace__panes--resizing',
          activeArrangeDrag && 'preview-workspace__panes--dragging',
        )}
        style={{
          gridTemplateColumns: buildGridTrackTemplate(columnFractions, SPLITTER_SIZE),
          gridTemplateRows: buildGridTrackTemplate(rowFractions, SPLITTER_SIZE),
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
                isDropTarget && 'preview-workspace__pane-slot--drop-target',
              )}
              style={paneSlotStyles[index]}
            >
              <PreviewPane
                paneId={pane.id}
                appId={pane.appId}
                apps={apps}
                isFocused={focusedPaneId === pane.id}
                canRemove={panes.length > previewWorkspaceLimits.minPanes}
                isArrangeMode={interactionMode === 'arrange'}
                isBeingDragged={isDragging}
                onFocus={focusPane}
                onRemove={removePane}
                onArrangeDragStart={startArrangeDrag}
              />
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
            onPointerDown={(event) => startRowResize(index, event)}
            aria-label={`Resize row ${index + 1}`}
          />
        ))}
      </div>
    </div>
  );
}
