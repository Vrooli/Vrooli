import { FileStack, X } from "lucide-react";
import { Button } from "./ui/button";
import { useIsMobile } from "../hooks";

interface StageSameNameConfirmationModalProps {
  isOpen: boolean;
  files: string[];
  isLoading: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}

export function StageSameNameConfirmationModal({
  isOpen,
  files,
  isLoading,
  onConfirm,
  onCancel,
}: StageSameNameConfirmationModalProps) {
  const isMobile = useIsMobile();
  if (!isOpen || files.length === 0) return null;

  const firstFile = files[0] ?? "";
  const filename = firstFile.split("/").pop() || firstFile;
  const content = (
    <>
      <div className="flex items-center gap-3 text-emerald-400 mb-4">
        <FileStack className="h-5 w-5 flex-shrink-0" />
        <span className="font-semibold">Stage {files.length} changed files named {filename}?</span>
      </div>
      <p className="text-sm text-slate-400 mb-4">
        This stages matching files across the current repository only.
      </p>
      <ul className="max-h-64 overflow-y-auto space-y-1">
        {files.map((file) => (
          <li key={file} className="text-sm text-slate-300 font-mono whitespace-nowrap">
            {file}
          </li>
        ))}
      </ul>
    </>
  );

  if (isMobile) {
    return (
      <div className="fixed inset-0 z-50 flex flex-col bg-slate-950" role="dialog" aria-modal="true" aria-label={`Confirm staging ${filename} files`}>
        <div className="flex items-center justify-between border-b border-slate-800 px-4 py-4 pt-safe">
          <h2 className="text-base font-semibold text-slate-100">Confirm Stage</h2>
          <button type="button" className="h-11 w-11 inline-flex items-center justify-center rounded-full border border-slate-700 text-slate-300" onClick={onCancel} aria-label="Close" disabled={isLoading}>
            <X className="h-5 w-5" />
          </button>
        </div>
        <div className="flex-1 overflow-y-auto px-4 py-6">{content}</div>
        <div className="border-t border-slate-800 px-4 py-4 pb-safe flex gap-3">
          <Button variant="outline" size="sm" onClick={onCancel} disabled={isLoading} className="flex-1 h-12">Cancel</Button>
          <Button size="sm" onClick={onConfirm} disabled={isLoading} className="flex-1 h-12">{isLoading ? "Staging..." : "Stage Files"}</Button>
        </div>
      </div>
    );
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/80 px-4" role="dialog" aria-modal="true" aria-label={`Confirm staging ${filename} files`} onClick={(e) => e.target === e.currentTarget && !isLoading && onCancel()}>
      <div className="w-full max-w-md rounded-xl border border-slate-800 bg-slate-950 shadow-xl">
        <div className="flex items-center justify-between border-b border-slate-800 px-4 py-3">
          <h2 className="text-sm font-semibold text-slate-100">Confirm Stage</h2>
          <button type="button" className="h-8 w-8 inline-flex items-center justify-center rounded-full border border-slate-700 text-slate-300" onClick={onCancel} disabled={isLoading} aria-label="Close"><X className="h-4 w-4" /></button>
        </div>
        <div className="px-4 py-4">{content}</div>
        <div className="flex items-center justify-end gap-2 border-t border-slate-800 px-4 py-3">
          <Button variant="outline" size="sm" onClick={onCancel} disabled={isLoading}>Cancel</Button>
          <Button size="sm" onClick={onConfirm} disabled={isLoading}>{isLoading ? "Staging..." : "Stage Files"}</Button>
        </div>
      </div>
    </div>
  );
}
