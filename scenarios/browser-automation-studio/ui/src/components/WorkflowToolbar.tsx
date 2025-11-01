import { useEffect, useMemo, useRef, useState } from 'react';
import {
  ChevronDown,
  Copy,
  Grid,
  Lock,
  Maximize2,
  MonitorSmartphone,
  Redo,
  Trash2,
  Undo,
  Unlock,
  ZoomIn,
  ZoomOut,
} from 'lucide-react';
import { useReactFlow } from 'reactflow';
import { usePopoverPosition } from '../hooks/usePopoverPosition';
import type { ExecutionViewportSettings } from '../stores/workflowStore';

const COMPACT_WIDTH_THRESHOLD = 560;
const menuButtonClass = 'flex w-full items-center gap-2 rounded-md px-3 py-2 text-sm text-gray-200 hover:bg-flow-node-hover transition-colors';

interface WorkflowToolbarProps {
  showGrid: boolean;
  onToggleGrid: () => void;
  locked: boolean;
  onToggleLock: () => void;
  onUndo: () => void;
  onRedo: () => void;
  canUndo: boolean;
  canRedo: boolean;
  onDuplicate: () => void;
  onDelete: () => void;
  graphWidth: number;
  onConfigureViewport: () => void;
  executionViewport?: ExecutionViewportSettings;
}

function WorkflowToolbar({
  showGrid,
  onToggleGrid,
  locked,
  onToggleLock,
  onUndo,
  onRedo,
  canUndo,
  canRedo,
  onDuplicate,
  onDelete,
  graphWidth,
  onConfigureViewport,
  executionViewport,
}: WorkflowToolbarProps) {
  const { zoomIn, zoomOut, fitView } = useReactFlow();
  const isCompact = graphWidth > 0 && graphWidth < COMPACT_WIDTH_THRESHOLD;

  const [showLayoutMenu, setShowLayoutMenu] = useState(false);
  const [showSelectionMenu, setShowSelectionMenu] = useState(false);

  const layoutButtonRef = useRef<HTMLButtonElement | null>(null);
  const layoutMenuRef = useRef<HTMLDivElement | null>(null);
  const selectionButtonRef = useRef<HTMLButtonElement | null>(null);
  const selectionMenuRef = useRef<HTMLDivElement | null>(null);

  const { floatingStyles: layoutMenuStyles } = usePopoverPosition(layoutButtonRef, layoutMenuRef, {
    isOpen: showLayoutMenu,
    placementPriority: ['bottom-start', 'top-start', 'bottom-end', 'top-end'],
  });
  const { floatingStyles: selectionMenuStyles } = usePopoverPosition(selectionButtonRef, selectionMenuRef, {
    isOpen: showSelectionMenu,
    placementPriority: ['bottom-start', 'top-start', 'bottom-end', 'top-end'],
  });

  useEffect(() => {
    if (!showLayoutMenu && !showSelectionMenu) {
      return;
    }

    const handlePointer = (event: MouseEvent | TouchEvent) => {
      const target = event.target as Node | null;
      if (showLayoutMenu) {
        const inTrigger = layoutButtonRef.current?.contains(target as Node);
        const inMenu = layoutMenuRef.current?.contains(target as Node);
        if (!inTrigger && !inMenu) {
          setShowLayoutMenu(false);
        }
      }
      if (showSelectionMenu) {
        const inTrigger = selectionButtonRef.current?.contains(target as Node);
        const inMenu = selectionMenuRef.current?.contains(target as Node);
        if (!inTrigger && !inMenu) {
          setShowSelectionMenu(false);
        }
      }
    };

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        if (showLayoutMenu) {
          setShowLayoutMenu(false);
        }
        if (showSelectionMenu) {
          setShowSelectionMenu(false);
        }
      }
    };

    document.addEventListener('mousedown', handlePointer);
    document.addEventListener('touchstart', handlePointer);
    document.addEventListener('keydown', handleKeyDown);
    return () => {
      document.removeEventListener('mousedown', handlePointer);
      document.removeEventListener('touchstart', handlePointer);
      document.removeEventListener('keydown', handleKeyDown);
    };
  }, [showLayoutMenu, showSelectionMenu]);

  useEffect(() => {
    if (!isCompact) {
      setShowLayoutMenu(false);
      setShowSelectionMenu(false);
    }
  }, [isCompact]);

  const viewportSummary = useMemo(() => {
    if (!executionViewport || !Number.isFinite(executionViewport.width) || !Number.isFinite(executionViewport.height)) {
      return '';
    }
    return `${executionViewport.width} × ${executionViewport.height}`;
  }, [executionViewport]);

  const handleFitView = () => {
    fitView();
    setShowLayoutMenu(false);
  };

  const handleToggleGridClick = () => {
    onToggleGrid();
    setShowLayoutMenu(false);
  };

  const handleToggleLockClick = () => {
    onToggleLock();
    setShowLayoutMenu(false);
  };

  const handleConfigureViewportClick = () => {
    setShowLayoutMenu(false);
    onConfigureViewport();
  };

  const handleDuplicateClick = () => {
    onDuplicate();
    setShowSelectionMenu(false);
  };

  const handleDeleteClick = () => {
    onDelete();
    setShowSelectionMenu(false);
  };

  return (
    <div className="absolute top-4 left-4 z-10 workflow-toolbar">
      <button
        type="button"
        onClick={() => zoomIn()}
        className="toolbar-button"
        title="Zoom in"
      >
        <ZoomIn size={18} />
      </button>

      <button
        type="button"
        onClick={() => zoomOut()}
        className="toolbar-button"
        title="Zoom out"
      >
        <ZoomOut size={18} />
      </button>

      {isCompact ? (
        <>
          <div className="w-px h-6 bg-gray-700" />

          <button
            type="button"
            ref={layoutButtonRef}
            onClick={() => setShowLayoutMenu((prev) => !prev)}
            className="toolbar-button flex items-center gap-1"
            aria-haspopup="menu"
            aria-expanded={showLayoutMenu}
            title={`Canvas options${viewportSummary ? ` (${viewportSummary})` : ''}`}
          >
            <Maximize2 size={16} />
            <ChevronDown size={14} className="text-gray-500" />
          </button>

          {showLayoutMenu && (
            <div
              ref={layoutMenuRef}
              style={{ ...layoutMenuStyles, zIndex: 40 }}
              className="mt-1 min-w-[200px] rounded-md border border-gray-700 bg-flow-node p-1 shadow-xl"
            >
              <button type="button" className={menuButtonClass} onClick={handleFitView}>
                <Maximize2 size={16} className="text-gray-400" />
                <span>Fit to view</span>
              </button>
              <button
                type="button"
                className={`${menuButtonClass} flex-col items-start`}
                onClick={handleConfigureViewportClick}
              >
                <span className="flex items-center gap-2 text-gray-200">
                  <MonitorSmartphone size={16} className="text-gray-400" />
                  <span>Set dimensions</span>
                </span>
                {viewportSummary && (
                  <span className="mt-1 text-[11px] text-gray-400">Current: {viewportSummary}</span>
                )}
              </button>
              <button type="button" className={menuButtonClass} onClick={handleToggleGridClick}>
                <Grid size={16} className="text-gray-400" />
                <span>{showGrid ? 'Hide grid' : 'Show grid'}</span>
              </button>
              <button type="button" className={menuButtonClass} onClick={handleToggleLockClick}>
                {locked ? <Lock size={16} className="text-gray-400" /> : <Unlock size={16} className="text-gray-400" />}
                <span>{locked ? 'Unlock editing' : 'Lock editing'}</span>
              </button>
            </div>
          )}

          <div className="w-px h-6 bg-gray-700" />
        </>
      ) : (
        <>
          <div className="w-px h-6 bg-gray-700" />

          <button
            type="button"
            onClick={handleFitView}
            className="toolbar-button"
            title="Fit view"
          >
            <Maximize2 size={18} />
          </button>

          <button
            type="button"
            onClick={handleConfigureViewportClick}
            className="toolbar-button"
            title={`Set execution dimensions${viewportSummary ? ` (${viewportSummary})` : ''}`}
          >
            <MonitorSmartphone size={18} />
          </button>

          <button
            type="button"
            onClick={handleToggleGridClick}
            className={`toolbar-button ${showGrid ? 'active' : ''}`}
            title={showGrid ? 'Hide grid' : 'Show grid'}
            aria-pressed={showGrid}
          >
            <Grid size={18} />
          </button>

          <button
            type="button"
            onClick={handleToggleLockClick}
            className="toolbar-button"
            title={locked ? 'Unlock editing' : 'Lock editing'}
          >
            {locked ? <Lock size={18} /> : <Unlock size={18} />}
          </button>

          <div className="w-px h-6 bg-gray-700" />
        </>
      )}

      <button
        type="button"
        onClick={onUndo}
        className={`toolbar-button ${!canUndo ? 'opacity-50 cursor-not-allowed' : ''}`}
        title="Undo"
        disabled={!canUndo}
      >
        <Undo size={18} />
      </button>

      <button
        type="button"
        onClick={onRedo}
        className={`toolbar-button ${!canRedo ? 'opacity-50 cursor-not-allowed' : ''}`}
        title="Redo"
        disabled={!canRedo}
      >
        <Redo size={18} />
      </button>

      {isCompact ? (
        <>
          <div className="w-px h-6 bg-gray-700" />
          <button
            type="button"
            ref={selectionButtonRef}
            onClick={() => setShowSelectionMenu((prev) => !prev)}
            className="toolbar-button flex items-center gap-1"
            aria-haspopup="menu"
            aria-expanded={showSelectionMenu}
            title="Selection actions"
          >
            <Copy size={16} />
            <ChevronDown size={14} className="text-gray-500" />
          </button>
          {showSelectionMenu && (
            <div
              ref={selectionMenuRef}
              style={{ ...selectionMenuStyles, zIndex: 40 }}
              className="mt-1 min-w-[180px] rounded-md border border-gray-700 bg-flow-node p-1 shadow-xl"
            >
              <button type="button" className={menuButtonClass} onClick={handleDuplicateClick}>
                <Copy size={16} className="text-gray-400" />
                <span>Duplicate selected</span>
              </button>
              <button
                type="button"
                className={`${menuButtonClass} text-red-300 hover:bg-red-500/10`}
                onClick={handleDeleteClick}
              >
                <Trash2 size={16} className="text-red-300" />
                <span>Delete selected</span>
              </button>
            </div>
          )}
        </>
      ) : (
        <>
          <div className="w-px h-6 bg-gray-700" />

          <button
            type="button"
            onClick={onDuplicate}
            className="toolbar-button"
            title="Duplicate selected"
          >
            <Copy size={18} />
          </button>

          <button
            type="button"
            onClick={onDelete}
            className="toolbar-button text-red-400 hover:text-red-300"
            title="Delete selected"
          >
            <Trash2 size={18} />
          </button>
        </>
      )}
    </div>
  );
}

export default WorkflowToolbar;
