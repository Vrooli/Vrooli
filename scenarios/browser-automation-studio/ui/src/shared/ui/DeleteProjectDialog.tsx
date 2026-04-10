import { X, AlertTriangle } from "lucide-react";
import type { DeleteProjectDialogState } from "@hooks/useDeleteProjectDialog";
import ResponsiveDialog from "@shared/layout/ResponsiveDialog";

interface DeleteProjectDialogProps {
  /** Dialog state from useDeleteProjectDialog hook */
  state: DeleteProjectDialogState | null;
  /** Called when deleteFiles checkbox changes */
  onDeleteFilesChange: (value: boolean) => void;
  /** Called when dialog should close (true = confirmed, false = cancelled) */
  onClose: (confirmed: boolean) => void;
}

/**
 * Delete project confirmation dialog with option to delete files from disk.
 * Use with the useDeleteProjectDialog hook.
 */
export function DeleteProjectDialog({
  state,
  onDeleteFilesChange,
  onClose,
}: DeleteProjectDialogProps) {
  if (!state) return null;

  return (
    <ResponsiveDialog
      isOpen={state.isOpen}
      onDismiss={() => onClose(false)}
      ariaLabel="Delete project"
      role="alertdialog"
    >
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-lg font-semibold text-surface">Delete project?</h2>
        <button
          type="button"
          onClick={() => onClose(false)}
          className="p-1 text-subtle hover:text-surface hover:bg-gray-700 rounded-full transition-colors"
          aria-label="Close"
        >
          <X size={16} />
        </button>
      </div>

      <p className="text-sm text-gray-200 mb-4">
        Remove <span className="font-medium text-surface">{state.projectName}</span> from
        Browser Automation Studio? This will delete the project index and all workflow
        database entries.
      </p>

      <label className="flex items-start gap-3 p-3 rounded-lg bg-gray-900/50 border border-gray-700 cursor-pointer hover:border-gray-600 transition-colors">
        <input
          type="checkbox"
          checked={state.deleteFiles}
          onChange={(e) => onDeleteFilesChange(e.target.checked)}
          className="mt-0.5 w-4 h-4 rounded border-gray-600 bg-gray-800 text-red-500 focus:ring-red-500/40 focus:ring-offset-0"
        />
        <div className="flex-1">
          <span className="text-sm font-medium text-surface">
            Also delete project files from disk
          </span>
          <p className="text-xs text-gray-400 mt-1">
            This will permanently delete the workflows directory and BAS configuration
            files. The project folder itself will remain.
          </p>
        </div>
      </label>

      {state.deleteFiles && (
        <div className="mt-3 flex items-start gap-2 p-3 rounded-lg bg-red-500/10 border border-red-500/30">
          <AlertTriangle size={16} className="text-red-400 mt-0.5 flex-shrink-0" />
          <p className="text-xs text-red-300">
            This action cannot be undone. All workflow files will be permanently deleted.
          </p>
        </div>
      )}

      <div className="mt-6 flex items-center justify-end gap-2">
        <button
          type="button"
          onClick={() => onClose(false)}
          className="px-4 py-2 rounded-lg border border-gray-700 text-subtle hover:text-surface hover:border-flow-accent transition-colors"
        >
          Cancel
        </button>
        <button
          type="button"
          onClick={() => onClose(true)}
          className="px-4 py-2 rounded-lg bg-red-600 text-white hover:bg-red-500 transition-colors"
        >
          {state.deleteFiles ? "Delete Everything" : "Remove Project"}
        </button>
      </div>
    </ResponsiveDialog>
  );
}

export default DeleteProjectDialog;
