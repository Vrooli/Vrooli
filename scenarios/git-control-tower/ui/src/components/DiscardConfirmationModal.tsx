import { AlertTriangle, FileX, RotateCcw, X } from "lucide-react";
import { Button } from "./ui/button";
import { useIsMobile } from "../hooks";

export interface DiscardFile {
  path: string;
  untracked: boolean;
}

interface DiscardConfirmationModalProps {
  isOpen: boolean;
  files: DiscardFile[];
  isLoading: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}

export function DiscardConfirmationModal({
  isOpen,
  files,
  isLoading,
  onConfirm,
  onCancel
}: DiscardConfirmationModalProps) {
  const isMobile = useIsMobile();

  if (!isOpen || files.length === 0) return null;

  const trackedFiles = files.filter((f) => !f.untracked);
  const untrackedFiles = files.filter((f) => f.untracked);

  const content = (
    <>
      <div className="flex items-center gap-3 text-red-400 mb-4">
        <AlertTriangle className="h-5 w-5 flex-shrink-0" />
        <span className="font-semibold">
          Discard {files.length} {files.length === 1 ? "change" : "changes"}?
        </span>
      </div>

      <p className="text-sm text-slate-400 mb-4">
        This action cannot be undone. The following files will be affected:
      </p>

      <div className="max-h-64 overflow-y-auto space-y-3">
        {trackedFiles.length > 0 && (
          <div>
            <div className="flex items-center gap-2 text-xs text-slate-500 mb-2">
              <RotateCcw className="h-3 w-3 flex-shrink-0" />
              <span>Revert to last commit ({trackedFiles.length})</span>
            </div>
            <ul className="space-y-1 overflow-x-auto">
              {trackedFiles.map((file) => (
                <li
                  key={file.path}
                  className="text-sm text-slate-300 font-mono pl-5 whitespace-nowrap"
                >
                  {file.path}
                </li>
              ))}
            </ul>
          </div>
        )}

        {untrackedFiles.length > 0 && (
          <div>
            <div className="flex items-center gap-2 text-xs text-red-400 mb-2">
              <FileX className="h-3 w-3 flex-shrink-0" />
              <span>Permanently delete ({untrackedFiles.length})</span>
            </div>
            <ul className="space-y-1 overflow-x-auto">
              {untrackedFiles.map((file) => (
                <li
                  key={file.path}
                  className="text-sm text-slate-300 font-mono pl-5 whitespace-nowrap"
                >
                  {file.path}
                </li>
              ))}
            </ul>
          </div>
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
        aria-label="Confirm discard"
      >
        {/* Header */}
        <div className="flex items-center justify-between border-b border-slate-800 px-4 py-4 pt-safe">
          <h2 className="text-base font-semibold text-slate-100">Confirm Discard</h2>
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
            {isLoading ? "Discarding..." : "Discard All"}
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
      aria-label="Confirm discard"
      onClick={(e) => {
        if (e.target === e.currentTarget && !isLoading) {
          onCancel();
        }
      }}
    >
      <div className="w-full max-w-2xl rounded-xl border border-slate-800 bg-slate-950 shadow-xl">
        <div className="flex items-center justify-between border-b border-slate-800 px-4 py-3">
          <h2 className="text-sm font-semibold text-slate-100">Confirm Discard</h2>
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
            {isLoading ? "Discarding..." : "Discard All"}
          </Button>
        </div>
      </div>
    </div>
  );
}
