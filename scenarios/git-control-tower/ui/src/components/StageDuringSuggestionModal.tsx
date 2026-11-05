import { FullPageDrawer } from "@vrooli/react-component-library/FullPageDrawer/1";
import { Sparkles, X } from "lucide-react";
import { Button } from "./ui/button";
import { useIsMobile } from "../hooks";
import type { StagedSuggestionHold } from "../features/commit-message/messageSession";

interface StageDuringSuggestionModalProps {
  hold: StagedSuggestionHold | null;
  paths: string[];
  onConfirm: () => void;
  onCancel: () => void;
}

/** Staging changes the exact selection a commit suggestion describes. */
export function StageDuringSuggestionModal({ hold, paths, onConfirm, onCancel }: StageDuringSuggestionModalProps) {
  const isMobile = useIsMobile();
  if (!hold || paths.length === 0) return null;

  const drafting = hold.state === "drafting";
  const noun = paths.length === 1 ? "file" : "files";
  const title = drafting ? "A commit suggestion is being drafted" : "A commit suggestion is ready";
  const described = hold.paths.length ? `${hold.paths.length} staged ${hold.paths.length === 1 ? "file" : "files"}` : "the current staged selection";
  const content = (
    <>
      <div className="flex items-center gap-3 text-amber-300 mb-4">
        <Sparkles className="h-5 w-5 flex-shrink-0" />
        <span className="font-semibold">Stage {paths.length} more {noun}?</span>
      </div>
      <p className="text-sm text-slate-400 mb-4">
        {drafting ? "The drafting agent is" : "The suggestion is"} describing {described}. Staging more
        means it will no longer cover everything you commit. You can unstage the extra files from the
        suggestion later.
      </p>
      <ul className="max-h-64 overflow-y-auto space-y-1" aria-label="Files to stage">
        {paths.map((file) => (
          <li key={file} className="text-sm text-slate-300 font-mono whitespace-nowrap">
            {file}
          </li>
        ))}
      </ul>
    </>
  );

  if (isMobile) {
    return (
      <FullPageDrawer
        open
        onClose={onCancel}
        title={title}
        ariaLabel={title}
        closeLabel="Close"
        dismissAffordance="close"
        contentClassName="px-4 py-6 space-y-6"
        footer={
          <div className="flex w-full items-center gap-3">
            <Button variant="outline" size="sm" onClick={onCancel} className="flex-1 h-12" data-testid="stage-during-suggestion-cancel">
              Keep selection
            </Button>
            <Button size="sm" onClick={onConfirm} className="flex-1 h-12" data-testid="stage-during-suggestion-confirm">
              Stage anyway
            </Button>
          </div>
        }
      >
        {content}
      </FullPageDrawer>
    );
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/80 px-4" role="dialog" aria-modal="true" aria-label={title} onClick={(e) => e.target === e.currentTarget && onCancel()}>
      <div className="w-full max-w-md rounded-xl border border-slate-800 bg-slate-950 shadow-xl">
        <div className="flex items-center justify-between border-b border-slate-800 px-4 py-3">
          <h2 className="text-sm font-semibold text-slate-100">{title}</h2>
          <button type="button" className="h-8 w-8 inline-flex items-center justify-center rounded-full border border-slate-700 text-slate-300" onClick={onCancel} aria-label="Close"><X className="h-4 w-4" /></button>
        </div>
        <div className="px-4 py-4">{content}</div>
        <div className="flex items-center justify-end gap-2 border-t border-slate-800 px-4 py-3">
          <Button variant="outline" size="sm" onClick={onCancel} data-testid="stage-during-suggestion-cancel">Keep selection</Button>
          <Button size="sm" onClick={onConfirm} data-testid="stage-during-suggestion-confirm">Stage anyway</Button>
        </div>
      </div>
    </div>
  );
}
