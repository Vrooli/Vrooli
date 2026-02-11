import { useCallback, useEffect, useMemo, useState } from 'react';
import { ChevronDown, ChevronUp, Eye, Grip, Layers, Maximize2, Minimize2, Pin, Plus, RotateCcw, Trash2, X, ZoomIn } from 'lucide-react';
import clsx from 'clsx';
import {
  PREVIEW_WORKSPACE_ZOOM_LEVELS,
  type PreviewWorkspaceZoomLevel,
  previewWorkspaceLimits,
  usePreviewWorkspaceStore,
} from '@/features/preview-workspace/state/previewWorkspaceStore';
import { useAppsStore } from '@/state/appsStore';
import { getAppDisplayName } from '@/components/tabSwitcher/tabSwitcherUtils';
import './WorkspaceManagerDialog.css';

type WorkspaceManagerDialogProps = {
  onClose: () => void;
};

export default function WorkspaceManagerDialog({ onClose }: WorkspaceManagerDialogProps) {
  const panes = usePreviewWorkspaceStore(state => state.panes);
  const interactionMode = usePreviewWorkspaceStore(state => state.interactionMode);
  const workspaceZoom = usePreviewWorkspaceStore(state => state.workspaceZoom);
  const pinnedPaneId = usePreviewWorkspaceStore(state => state.pinnedPaneId);
  const focusedPaneId = usePreviewWorkspaceStore(state => state.focusedPaneId);
  const addPane = usePreviewWorkspaceStore(state => state.addPane);
  const removePane = usePreviewWorkspaceStore(state => state.removePane);
  const movePaneToIndex = usePreviewWorkspaceStore(state => state.movePaneToIndex);
  const focusPane = usePreviewWorkspaceStore(state => state.focusPane);
  const setWorkspaceZoom = usePreviewWorkspaceStore(state => state.setWorkspaceZoom);
  const resetWorkspaceZoom = usePreviewWorkspaceStore(state => state.resetWorkspaceZoom);
  const clearPinnedPane = usePreviewWorkspaceStore(state => state.clearPinnedPane);
  const setInteractionMode = usePreviewWorkspaceStore(state => state.setInteractionMode);
  const resetLayout = usePreviewWorkspaceStore(state => state.resetLayout);
  const clearAllPanes = usePreviewWorkspaceStore(state => state.clearAllPanes);
  const apps = useAppsStore(state => state.apps);
  const [isWorkspaceFullscreen, setIsWorkspaceFullscreen] = useState(false);

  const paneCount = panes.length;
  const canAddPane = paneCount < previewWorkspaceLimits.maxPanes;
  const isWorkspaceEmpty = panes.length === 1 && panes[0]?.appId === null;

  const arrangeButtonLabel = interactionMode === 'arrange' ? 'Turn arrange off' : 'Turn arrange on';
  const arrangeButtonHint = interactionMode === 'arrange'
    ? 'Switch to browse mode.'
    : 'Enable drag-and-arrange mode.';

  const paneSummary = useMemo(() => {
    return `${paneCount} pane${paneCount === 1 ? '' : 's'} active`;
  }, [paneCount]);
  const workspaceZoomPercent = Math.round(workspaceZoom * 100);
  const canToggleWorkspaceFullscreen = typeof document !== 'undefined' && typeof document.exitFullscreen === 'function';

  useEffect(() => {
    if (typeof document === 'undefined') {
      return;
    }

    const syncWorkspaceFullscreenState = () => {
      const workspaceRoot = document.querySelector<HTMLElement>('.preview-workspace');
      setIsWorkspaceFullscreen(Boolean(workspaceRoot && document.fullscreenElement === workspaceRoot));
    };

    syncWorkspaceFullscreenState();
    document.addEventListener('fullscreenchange', syncWorkspaceFullscreenState);
    return () => {
      document.removeEventListener('fullscreenchange', syncWorkspaceFullscreenState);
    };
  }, []);

  const handleToggleArrange = () => {
    setInteractionMode(interactionMode === 'arrange' ? 'browse' : 'arrange');
    onClose();
  };

  const handleAddPane = () => {
    if (!canAddPane) {
      return;
    }
    const newPaneId = addPane(null);
    if (typeof document !== 'undefined') {
      window.requestAnimationFrame(() => {
        window.requestAnimationFrame(() => {
          const newPane = document.querySelector<HTMLElement>(`[data-preview-pane-id="${newPaneId}"]`);
          newPane?.scrollIntoView({
            behavior: 'smooth',
            block: 'nearest',
            inline: 'nearest',
          });
          const urlInput = newPane?.querySelector<HTMLInputElement>('input[aria-label="Preview URL"]');
          if (urlInput) {
            urlInput.focus({ preventScroll: true });
            urlInput.select();
          }
        });
      });
    }
    onClose();
  };

  const handleResetLayout = () => {
    resetLayout();
    onClose();
  };

  const handleClearAllPanes = () => {
    if (isWorkspaceEmpty) {
      return;
    }
    const shouldClear = typeof window === 'undefined'
      ? true
      : window.confirm('Clear all panes and reset to a single empty pane?');
    if (!shouldClear) {
      return;
    }
    clearAllPanes();
    onClose();
  };

  const handleUnpinPane = () => {
    clearPinnedPane();
    onClose();
  };

  const appsByKey = useMemo(() => {
    const map = new Map<string, string>();
    for (const app of apps) {
      map.set(app.id, getAppDisplayName(app));
      if (app.scenario_name) {
        map.set(app.scenario_name, getAppDisplayName(app));
      }
    }
    return map;
  }, [apps]);

  const getPaneLabel = useCallback((appId: string | null): string => {
    if (!appId) return 'Empty pane';
    return appsByKey.get(appId) ?? appId;
  }, [appsByKey]);

  const handleScrollToPane = useCallback((paneId: string) => {
    focusPane(paneId);
    onClose();
    window.requestAnimationFrame(() => {
      const paneEl = document.querySelector<HTMLElement>(`[data-preview-pane-id="${paneId}"]`);
      paneEl?.scrollIntoView({ behavior: 'smooth', block: 'nearest', inline: 'nearest' });
    });
  }, [focusPane, onClose]);

  const handleRemovePane = useCallback((paneId: string) => {
    removePane(paneId);
  }, [removePane]);

  const handleMoveUp = useCallback((paneId: string, index: number) => {
    if (index > 0) {
      movePaneToIndex(paneId, index - 1);
    }
  }, [movePaneToIndex]);

  const handleMoveDown = useCallback((paneId: string, index: number) => {
    if (index < panes.length - 1) {
      movePaneToIndex(paneId, index + 1);
    }
  }, [movePaneToIndex, panes.length]);

  const handleWorkspaceZoomChange = (value: string) => {
    const parsed = Number.parseFloat(value);
    if (!Number.isFinite(parsed)) {
      return;
    }
    setWorkspaceZoom(parsed as PreviewWorkspaceZoomLevel);
  };

  const handleToggleWorkspaceFullscreen = async () => {
    if (typeof document === 'undefined' || typeof document.exitFullscreen !== 'function') {
      return;
    }

    const workspaceRoot = document.querySelector<HTMLElement>('.preview-workspace');
    if (!workspaceRoot || typeof workspaceRoot.requestFullscreen !== 'function') {
      return;
    }

    try {
      if (document.fullscreenElement === workspaceRoot) {
        await document.exitFullscreen();
        onClose();
        return;
      }

      if (document.fullscreenElement) {
        await document.exitFullscreen();
      }
      await workspaceRoot.requestFullscreen();
      onClose();
    } catch {
      // Ignore fullscreen API failures (e.g., browser policy/user gesture constraints).
    }
  };

  return (
    <div className="workspace-manager">
      <header className="workspace-manager__header">
        <div>
          <h2>Workspace Manager</h2>
          <p>{paneSummary}</p>
        </div>
        <button
          type="button"
          className="workspace-manager__close"
          aria-label="Close workspace manager"
          onClick={onClose}
        >
          <X size={16} aria-hidden />
        </button>
      </header>

      <section className="workspace-manager__actions" aria-label="Workspace actions">
        <div className="workspace-manager__zoom" role="group" aria-label="Workspace zoom controls">
          <div className="workspace-manager__zoom-heading">
            <ZoomIn size={18} aria-hidden />
            <strong>Workspace zoom</strong>
          </div>
          <div className="workspace-manager__zoom-controls">
            <label htmlFor="workspace-zoom-level">All panes</label>
            <select
              id="workspace-zoom-level"
              value={workspaceZoom}
              onChange={(event) => handleWorkspaceZoomChange(event.target.value)}
            >
              {PREVIEW_WORKSPACE_ZOOM_LEVELS.map((level) => (
                <option key={level} value={level}>{`${Math.round(level * 100)}%`}</option>
              ))}
            </select>
            <button
              type="button"
              onClick={resetWorkspaceZoom}
              disabled={workspaceZoom === 1}
            >
              Reset
            </button>
          </div>
          <p>
            Active zoom: {workspaceZoomPercent}%.
            Pane device emulation zoom overrides this value.
          </p>
        </div>

        <button
          type="button"
          className={clsx(
            'workspace-manager__action',
            interactionMode === 'arrange' && 'workspace-manager__action--active',
          )}
          onClick={handleToggleArrange}
        >
          <Grip size={18} aria-hidden />
          <div>
            <strong>{arrangeButtonLabel}</strong>
            <span>{arrangeButtonHint}</span>
          </div>
        </button>

        <button
          type="button"
          className="workspace-manager__action"
          onClick={() => {
            void handleToggleWorkspaceFullscreen();
          }}
          disabled={!canToggleWorkspaceFullscreen}
        >
          {isWorkspaceFullscreen ? <Minimize2 size={18} aria-hidden /> : <Maximize2 size={18} aria-hidden />}
          <div>
            <strong>{isWorkspaceFullscreen ? 'Exit workspace fullscreen' : 'Enter workspace fullscreen'}</strong>
            <span>{canToggleWorkspaceFullscreen ? 'Expand the entire workspace view, not just one pane.' : 'Fullscreen is not supported in this environment.'}</span>
          </div>
        </button>

        <button
          type="button"
          className="workspace-manager__action"
          onClick={handleAddPane}
          disabled={!canAddPane}
        >
          <Plus size={18} aria-hidden />
          <div>
            <strong>Add pane</strong>
            <span>{canAddPane ? 'Create a new empty preview pane.' : `Maximum of ${previewWorkspaceLimits.maxPanes} panes reached.`}</span>
          </div>
        </button>

        <button
          type="button"
          className="workspace-manager__action"
          onClick={handleResetLayout}
        >
          <RotateCcw size={18} aria-hidden />
          <div>
            <strong>Reset layout</strong>
            <span>Clear pinning and restore default grid sizing.</span>
          </div>
        </button>

        <button
          type="button"
          className="workspace-manager__action"
          onClick={handleUnpinPane}
          disabled={!pinnedPaneId}
        >
          <Layers size={18} aria-hidden />
          <div>
            <strong>Unpin pane</strong>
            <span>{pinnedPaneId ? 'Return pinned pane to normal flow.' : 'No pane is currently pinned.'}</span>
          </div>
        </button>

        <button
          type="button"
          className={clsx('workspace-manager__action', 'workspace-manager__action--danger')}
          onClick={handleClearAllPanes}
          disabled={isWorkspaceEmpty}
        >
          <Trash2 size={18} aria-hidden />
          <div>
            <strong>Clear all panes</strong>
            <span>{isWorkspaceEmpty ? 'Workspace is already empty.' : 'Reset to one empty pane.'}</span>
          </div>
        </button>
      </section>

      <section className="workspace-manager__pane-list" aria-label="Pane list">
        <h3 className="workspace-manager__pane-list-heading">Panes</h3>
        <ol className="workspace-manager__pane-list-items">
          {panes.map((pane, index) => {
            const isFocused = pane.id === focusedPaneId;
            const isPinned = pane.id === pinnedPaneId;
            const canMoveUp = index > 0;
            const canMoveDown = index < panes.length - 1;
            const canRemove = panes.length > 1;
            return (
              <li
                key={pane.id}
                className={clsx(
                  'workspace-manager__pane-row',
                  isFocused && 'workspace-manager__pane-row--focused',
                )}
              >
                <span className="workspace-manager__pane-label">
                  {isPinned && <Pin size={12} aria-label="Pinned" className="workspace-manager__pane-pin" />}
                  <span className="workspace-manager__pane-name">{getPaneLabel(pane.appId)}</span>
                </span>
                <span className="workspace-manager__pane-controls">
                  <button
                    type="button"
                    aria-label={`Move pane ${index + 1} up`}
                    disabled={!canMoveUp}
                    onClick={() => handleMoveUp(pane.id, index)}
                  >
                    <ChevronUp size={14} aria-hidden />
                  </button>
                  <button
                    type="button"
                    aria-label={`Move pane ${index + 1} down`}
                    disabled={!canMoveDown}
                    onClick={() => handleMoveDown(pane.id, index)}
                  >
                    <ChevronDown size={14} aria-hidden />
                  </button>
                  <button
                    type="button"
                    aria-label={`Scroll to pane ${index + 1}`}
                    onClick={() => handleScrollToPane(pane.id)}
                  >
                    <Eye size={14} aria-hidden />
                  </button>
                  <button
                    type="button"
                    aria-label={`Remove pane ${index + 1}`}
                    disabled={!canRemove}
                    onClick={() => handleRemovePane(pane.id)}
                  >
                    <X size={14} aria-hidden />
                  </button>
                </span>
              </li>
            );
          })}
        </ol>
      </section>
    </div>
  );
}
