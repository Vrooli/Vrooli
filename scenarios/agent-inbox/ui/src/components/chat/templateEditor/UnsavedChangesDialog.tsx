/**
 * Confirmation dialog shown when closing the template editor with unsaved changes.
 */

import { AlertTriangle } from "lucide-react";

interface UnsavedChangesDialogProps {
  dirtyCount: number;
  onKeepEditing: () => void;
  onDiscard: () => void;
}

export function UnsavedChangesDialog({
  dirtyCount,
  onKeepEditing,
  onDiscard,
}: UnsavedChangesDialogProps) {
  return (
    <div className="fixed inset-0 z-[60] flex items-center justify-center">
      <div
        className="absolute inset-0 bg-black/60"
        onClick={onKeepEditing}
      />
      <div className="relative bg-slate-900 border border-white/10 rounded-xl p-6 max-w-sm mx-4 shadow-xl">
        <div className="flex items-start gap-3 mb-4">
          <AlertTriangle className="h-6 w-6 text-amber-400 flex-shrink-0" />
          <div>
            <h3 className="text-lg font-semibold text-white">Unsaved Changes</h3>
            <p className="text-sm text-slate-400 mt-1">
              {dirtyCount > 1
                ? `You have unsaved changes in ${dirtyCount} templates. Are you sure you want to close without saving?`
                : "You have unsaved changes. Are you sure you want to close without saving?"}
            </p>
          </div>
        </div>
        <div className="flex justify-end gap-3">
          <button
            onClick={onKeepEditing}
            className="px-4 py-2 text-sm text-slate-400 hover:text-white transition-colors"
          >
            Keep Editing
          </button>
          <button
            onClick={onDiscard}
            className="px-4 py-2 text-sm font-medium bg-red-600 text-white rounded-lg hover:bg-red-500 transition-colors"
          >
            Discard {dirtyCount > 1 ? "All Changes" : "Changes"}
          </button>
        </div>
      </div>
    </div>
  );
}
