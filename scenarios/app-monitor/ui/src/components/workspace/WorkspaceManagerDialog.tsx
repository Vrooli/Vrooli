import { useMemo } from 'react';
import { Grip, Layers, Plus, RotateCcw, Trash2, X, ZoomIn } from 'lucide-react';
import clsx from 'clsx';
import {
  PREVIEW_WORKSPACE_ZOOM_LEVELS,
  type PreviewWorkspaceZoomLevel,
  previewWorkspaceLimits,
  usePreviewWorkspaceStore,
} from '@/features/preview-workspace/state/previewWorkspaceStore';
import './WorkspaceManagerDialog.css';

type WorkspaceManagerDialogProps = {
  onClose: () => void;
};

export default function WorkspaceManagerDialog({ onClose }: WorkspaceManagerDialogProps) {
  const panes = usePreviewWorkspaceStore(state => state.panes);
  const interactionMode = usePreviewWorkspaceStore(state => state.interactionMode);
  const workspaceZoom = usePreviewWorkspaceStore(state => state.workspaceZoom);
  const pinnedPaneId = usePreviewWorkspaceStore(state => state.pinnedPaneId);
  const addPane = usePreviewWorkspaceStore(state => state.addPane);
  const setWorkspaceZoom = usePreviewWorkspaceStore(state => state.setWorkspaceZoom);
  const resetWorkspaceZoom = usePreviewWorkspaceStore(state => state.resetWorkspaceZoom);
  const clearPinnedPane = usePreviewWorkspaceStore(state => state.clearPinnedPane);
  const setInteractionMode = usePreviewWorkspaceStore(state => state.setInteractionMode);
  const resetLayout = usePreviewWorkspaceStore(state => state.resetLayout);
  const clearAllPanes = usePreviewWorkspaceStore(state => state.clearAllPanes);

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

  const handleToggleArrange = () => {
    setInteractionMode(interactionMode === 'arrange' ? 'browse' : 'arrange');
    onClose();
  };

  const handleAddPane = () => {
    if (!canAddPane) {
      return;
    }
    addPane(null);
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

  const handleWorkspaceZoomChange = (value: string) => {
    const parsed = Number.parseFloat(value);
    if (!Number.isFinite(parsed)) {
      return;
    }
    setWorkspaceZoom(parsed as PreviewWorkspaceZoomLevel);
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
    </div>
  );
}
