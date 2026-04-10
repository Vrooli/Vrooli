import { AlertTriangle, FolderX, FileX, X } from "lucide-react";
import { Button } from "./ui/button";
import { useIsMobile } from "../hooks";

interface DeleteConfirmationModalProps {
  isOpen: boolean;
  path: string;
  isDirectory: boolean;
  isLoading: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}

export function DeleteConfirmationModal({
  isOpen,
  path,
  isDirectory,
  isLoading,
  onConfirm,
  onCancel
}: DeleteConfirmationModalProps) {
  const isMobile = useIsMobile();

  if (!isOpen) return null;

  const filename = path.split("/").pop() || path;
  const Icon = isDirectory ? FolderX : FileX;

  const content = (
    <>
      <div className="flex items-center gap-3 text-red-400 mb-4">
        <AlertTriangle className="h-5 w-5 flex-shrink-0" />
        <span className="font-semibold">
          Delete {isDirectory ? "folder" : "file"}?
        </span>
      </div>

      <div className="flex items-center gap-2 text-sm text-slate-300 mb-4">
        <Icon className="h-4 w-4 flex-shrink-0 text-slate-500" />
        <span className="font-mono break-all">{path}</span>
      </div>

      <div className="space-y-2 text-sm text-slate-400">
        <p>This action cannot be undone.</p>
        {isDirectory ? (
          <p className="text-red-400/80">
            All files and subfolders inside this folder will be permanently deleted.
          </p>
        ) : (
          <p>
            Tracked files will show as "deleted" in git status. You can restore them with git checkout.
            Untracked files will be permanently deleted.
          </p>
        )}
      </div>
    </>
  );

  // Mobile: full-screen modal
  if (isMobile) {
    return (
      <div
        className="fixed inset-0 z-50 flex flex-col bg-slate-950 animate-in slide-in-from-bottom duration-200"
        role="dialog"
        aria-modal="true"
        aria-label={`Confirm delete ${filename}`}
      >
        {/* Header */}
        <div className="flex items-center justify-between border-b border-slate-800 px-4 py-4 pt-safe">
          <h2 className="text-base font-semibold text-slate-100">Confirm Delete</h2>
          <button
            type="button"
            className="h-11 w-11 inline-flex items-center justify-center rounded-full border border-slate-700 text-slate-300 hover:bg-slate-800/60 active:bg-slate-700 touch-target"
            onClick={onCancel}
            aria-label="Close"
            disabled={isLoading}
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto px-4 py-6">{content}</div>

        {/* Footer */}
        <div className="border-t border-slate-800 px-4 py-4 pb-safe flex gap-3">
          <Button
            variant="outline"
            size="sm"
            onClick={onCancel}
            disabled={isLoading}
            className="flex-1 h-12 text-sm touch-target"
          >
            Cancel
          </Button>
          <Button
            variant="destructive"
            size="sm"
            onClick={onConfirm}
            disabled={isLoading}
            className="flex-1 h-12 text-sm touch-target"
          >
            {isLoading ? "Deleting..." : "Delete"}
          </Button>
        </div>
      </div>
    );
  }

  // Desktop: centered modal
  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/80 px-4"
      role="dialog"
      aria-modal="true"
      aria-label={`Confirm delete ${filename}`}
      onClick={(e) => {
        if (e.target === e.currentTarget && !isLoading) {
          onCancel();
        }
      }}
    >
      <div className="w-full max-w-md rounded-xl border border-slate-800 bg-slate-950 shadow-xl">
        <div className="flex items-center justify-between border-b border-slate-800 px-4 py-3">
          <h2 className="text-sm font-semibold text-slate-100">Confirm Delete</h2>
          <button
            type="button"
            className="h-8 w-8 inline-flex items-center justify-center rounded-full border border-slate-700 text-slate-300 hover:bg-slate-800/60"
            onClick={onCancel}
            disabled={isLoading}
            aria-label="Close"
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        <div className="px-4 py-4">{content}</div>

        <div className="flex items-center justify-end gap-2 border-t border-slate-800 px-4 py-3">
          <Button variant="outline" size="sm" onClick={onCancel} disabled={isLoading} className="h-8 px-3">
            Cancel
          </Button>
          <Button variant="destructive" size="sm" onClick={onConfirm} disabled={isLoading} className="h-8 px-3">
            {isLoading ? "Deleting..." : "Delete"}
          </Button>
        </div>
      </div>
    </div>
  );
}
