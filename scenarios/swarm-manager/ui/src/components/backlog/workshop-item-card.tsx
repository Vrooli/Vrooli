/**
 * Renders a single workshop item (decision or info).
 */
import { useState } from "react";
import { Trash2 } from "lucide-react";
import { cn } from "../../lib";
import { renderMarkdown } from "../../lib/render-markdown";
import { ClarifyButton } from "./clarify-button";
import { WorkshopQuestionView } from "./question-renderers";
import { useClarificationStore } from "../../stores/clarification-store";
import type { BacklogKind, WorkshopItem } from "../../types/domain";

interface WorkshopItemCardProps {
  item: WorkshopItem;
  disabled?: boolean;
  onUpdate?: (updated: WorkshopItem) => void;
  onDelete?: () => void;
  backlogKind?: BacklogKind;
  backlogName?: string;
  roundNumber?: number;
}

export function WorkshopItemCard({ item, disabled, onUpdate, onDelete, backlogKind, backlogName, roundNumber }: WorkshopItemCardProps) {
  const [localSelected, setLocalSelected] = useState(item.selected ?? "");
  const [localFreeform, setLocalFreeform] = useState(item.freeform ?? "");
  const [localNotes, setLocalNotes] = useState(item.notes ?? "");

  const [confirmDelete, setConfirmDelete] = useState(false);

  const clarificationStore = useClarificationStore();
  const isClarifyActive = clarificationStore.isOpen && clarificationStore.target?.itemId === item.id;

  const handleClarifyClick = () => {
    if (isClarifyActive) {
      clarificationStore.close();
    } else if (backlogKind && backlogName && roundNumber) {
      clarificationStore.open({
        backlogKind,
        backlogName,
        roundNumber,
        itemId: item.id,
        itemTopic: item.topic || item.text || "",
        clarificationId: item.clarification_id,
        currentItem: item,
      });
    }
  };

  // Info items — read-only text
  if (item.type === "info") {
    return (
      <div className="rounded-md border border-slate-700 bg-slate-800/30 p-3">
        <div className="grid grid-cols-[auto_minmax(0,1fr)_auto] items-start gap-x-2 gap-y-1">
          <span className="mt-0.5 rounded bg-blue-500/20 px-1.5 py-0.5 text-[10px] font-medium text-blue-400">
            Info
          </span>
          <div className="min-w-0">
            {item.topic && (
              <p className="break-words text-sm font-medium text-slate-300 [overflow-wrap:anywhere]">{item.topic}</p>
            )}
          </div>
          {onDelete && !disabled && (
            confirmDelete ? (
              <span className="flex items-center gap-1 shrink-0">
                <button
                  type="button"
                  onClick={() => { setConfirmDelete(false); onDelete(); }}
                  className="rounded px-1.5 py-0.5 text-[10px] font-medium text-red-400 bg-red-500/10 hover:bg-red-500/20 transition-colors"
                >
                  Delete
                </button>
                <button
                  type="button"
                  onClick={() => setConfirmDelete(false)}
                  className="rounded px-1.5 py-0.5 text-[10px] font-medium text-slate-400 hover:bg-slate-700 transition-colors"
                >
                  Cancel
                </button>
              </span>
            ) : (
              <button
                type="button"
                onClick={() => setConfirmDelete(true)}
                className="shrink-0 rounded p-1 text-slate-500 hover:text-red-400 hover:bg-red-500/10 transition-colors"
                title="Delete item"
              >
                <Trash2 className="h-3.5 w-3.5" />
              </button>
            )
          )}
        </div>
        {item.text && item.text !== item.topic && (
          <div
            className="prose-sm-slate mt-2 break-words text-sm leading-relaxed text-slate-300 [overflow-wrap:anywhere]"
            dangerouslySetInnerHTML={{ __html: renderMarkdown(item.text) }}
          />
        )}
        {item.context && (
          <div
            className="prose-sm-slate mt-2 break-words text-xs leading-relaxed text-slate-500 [overflow-wrap:anywhere]"
            dangerouslySetInnerHTML={{ __html: renderMarkdown(item.context) }}
          />
        )}
      </div>
    );
  }

  return (
    <div className={cn(
      "rounded-md border p-3",
      localSelected.trim() ? "border-emerald-500/20 bg-emerald-500/5" : "border-amber-500/20 bg-amber-500/5",
    )}>
      <div className="space-y-2">
        <WorkshopQuestionView
          question={item}
          answer={{
            selected: localSelected,
            freeform: localFreeform,
            notes: localNotes,
          }}
          disabled={!!disabled}
          onUpdate={(patch) => {
            if (patch.selected !== undefined) {
              setLocalSelected(patch.selected ?? "");
            }
            if (patch.freeform !== undefined) {
              setLocalFreeform(patch.freeform ?? "");
            }
            if (patch.notes !== undefined) {
              setLocalNotes(patch.notes ?? "");
            }
            onUpdate?.({
              ...item,
              selected: patch.selected !== undefined ? (patch.selected || null) : (localSelected || null),
              freeform: patch.freeform !== undefined ? (patch.freeform || null) : (localFreeform || null),
              notes: patch.notes !== undefined ? (patch.notes || null) : (localNotes || null),
            });
          }}
          actions={(
            <>
              {backlogKind && backlogName && roundNumber && (
                <ClarifyButton
                  disabled={disabled}
                  isActive={isClarifyActive}
                  hasClarification={!!item.clarification_id}
                  onClick={handleClarifyClick}
                />
              )}
              {onDelete && !disabled && (
                confirmDelete ? (
                  <span className="flex items-center gap-1 shrink-0">
                    <button
                      type="button"
                      onClick={() => { setConfirmDelete(false); onDelete(); }}
                      className="rounded px-1.5 py-0.5 text-[10px] font-medium text-red-400 bg-red-500/10 hover:bg-red-500/20 transition-colors"
                    >
                      Delete
                    </button>
                    <button
                      type="button"
                      onClick={() => setConfirmDelete(false)}
                      className="rounded px-1.5 py-0.5 text-[10px] font-medium text-slate-400 hover:bg-slate-700 transition-colors"
                    >
                      Cancel
                    </button>
                  </span>
                ) : (
                  <button
                    type="button"
                    onClick={() => setConfirmDelete(true)}
                    className="shrink-0 rounded p-1 text-slate-500 hover:text-red-400 hover:bg-red-500/10 transition-colors"
                    title="Delete item"
                  >
                    <Trash2 className="h-3.5 w-3.5" />
                  </button>
                )
              )}
            </>
          )}
        />

        {item.context_note && (
          <div className="mt-1 rounded border border-cyan-500/15 bg-cyan-500/5 px-2 py-1">
            <span className="text-[9px] font-medium text-cyan-400">Clarification note</span>
            <div
              className="prose-sm-slate mt-1 break-words text-xs text-slate-400 [overflow-wrap:anywhere]"
              dangerouslySetInnerHTML={{ __html: renderMarkdown(item.context_note) }}
            />
          </div>
        )}
      </div>
    </div>
  );
}
