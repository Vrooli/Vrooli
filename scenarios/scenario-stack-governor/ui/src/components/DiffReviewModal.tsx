import type { FixResult } from "../lib/api";
import { DiffViewer } from "./DiffViewer";
import { Button } from "./ui/button";

export function DiffReviewModal({
  open,
  results,
  onApply,
  onCancel,
  applying
}: {
  open: boolean;
  results: FixResult[];
  onApply: () => void;
  onCancel: () => void;
  applying: boolean;
}) {
  if (!open) return null;

  // Deduplicate by file_path — Makefile fix returns 3 results for same file
  const seen = new Set<string>();
  const uniqueResults = results.filter((r) => {
    if (!r.diff || !r.fixed) return false;
    if (seen.has(r.file_path)) return false;
    seen.add(r.file_path);
    return true;
  });

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60" onClick={onCancel}>
      <div
        className="w-full max-w-4xl max-h-[80vh] flex flex-col rounded-2xl border border-white/10 bg-slate-900/95 backdrop-blur"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="p-6 border-b border-white/10">
          <h3 className="text-lg font-semibold text-slate-50">Review Changes</h3>
          <p className="mt-1 text-sm text-slate-300">
            {uniqueResults.length} file{uniqueResults.length !== 1 ? "s" : ""} will be modified. Review the changes below before applying.
          </p>
        </div>
        <div className="flex-1 overflow-auto p-6 space-y-4">
          {uniqueResults.map((r) => (
            <DiffViewer key={r.file_path} before={r.diff!.before} after={r.diff!.after} filePath={r.file_path} />
          ))}
          {uniqueResults.length === 0 && (
            <p className="text-sm text-slate-400">No file changes to review.</p>
          )}
        </div>
        <div className="p-6 border-t border-white/10 flex justify-end gap-3">
          <Button variant="outline" size="sm" onClick={onCancel} disabled={applying}>
            Cancel
          </Button>
          <Button size="sm" onClick={onApply} disabled={applying || uniqueResults.length === 0}>
            {applying ? "Applying..." : "Apply"}
          </Button>
        </div>
      </div>
    </div>
  );
}
