import { useMemo } from 'react';
import { Grip, Layers, Plus, RotateCcw, Trash2, X } from 'lucide-react';
import clsx from 'clsx';
import {
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
  const pinnedPaneId = usePreviewWorkspaceStore(state => state.pinnedPaneId);
  const addPane = usePreviewWorkspaceStore(state => state.addPane);
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
