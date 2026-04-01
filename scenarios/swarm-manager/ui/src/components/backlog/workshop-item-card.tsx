/**
 * Renders a single workshop item (decision or info).
 */
import { useState } from "react";
import { Star, Trash2 } from "lucide-react";
import { cn } from "../../lib";
import { OTHER_KEY, filterAgentOther } from "../../lib/workshop-files";
import { ClarifyButton } from "./clarify-button";
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
      });
    }
  };

  const handleSelect = (key: string) => {
    const newSelected = key === OTHER_KEY ? OTHER_KEY : key;
    setLocalSelected(newSelected);
    onUpdate?.({
      ...item,
      selected: newSelected,
      freeform: key === OTHER_KEY ? localFreeform || null : null,
      notes: localNotes || null,
    });
  };

  const handleFreeformChange = (value: string) => {
    setLocalFreeform(value);
    onUpdate?.({ ...item, selected: OTHER_KEY, freeform: value || null, notes: localNotes || null });
  };

  const handleNotesChange = (value: string) => {
    setLocalNotes(value);
    onUpdate?.({ ...item, selected: localSelected || null, freeform: localFreeform || null, notes: value || null });
  };

  // Info items — read-only text
  if (item.type === "info") {
    return (
      <div className="rounded-md border border-slate-700 bg-slate-800/30 p-3">
        <div className="flex items-start gap-2">
          <span className="mt-0.5 rounded bg-blue-500/20 px-1.5 py-0.5 text-[10px] font-medium text-blue-400">
            Info
          </span>
          <p className="flex-1 text-sm text-slate-300">{item.topic || item.text}</p>
          {onDelete && !disabled && (
            <button
              type="button"
              onClick={onDelete}
              className="shrink-0 rounded p-1 text-slate-500 hover:text-red-400 hover:bg-red-500/10 transition-colors"
              title="Delete item"
            >
              <Trash2 className="h-3.5 w-3.5" />
            </button>
          )}
        </div>
        {item.context && (
          <p className="mt-1 ml-12 text-xs text-slate-500">{item.context}</p>
        )}
      </div>
    );
  }

  // Decision items
  const isResolved = !!localSelected.trim();
  const isOther = localSelected === OTHER_KEY;
  const options = filterAgentOther(item.options ?? []);

  return (
    <div className={cn(
      "rounded-md border p-3",
      isResolved ? "border-emerald-500/20 bg-emerald-500/5" : "border-amber-500/20 bg-amber-500/5",
    )}>
      <div className="space-y-2">
        {/* Header */}
        <div className="flex items-start gap-2">
          <span className={cn(
            "mt-0.5 rounded px-1.5 py-0.5 text-[10px] font-medium",
            isResolved ? "bg-emerald-500/20 text-emerald-400" : "bg-amber-500/20 text-amber-400",
          )}>
            D
          </span>
          <p className="flex-1 text-sm font-medium text-slate-200">{item.topic || item.text}</p>
          {backlogKind && backlogName && roundNumber && (
            <ClarifyButton
              disabled={disabled}
              isActive={isClarifyActive}
              onClick={handleClarifyClick}
            />
          )}
          {onDelete && !disabled && (
            <button
              type="button"
              onClick={onDelete}
              className="shrink-0 rounded p-1 text-slate-500 hover:text-red-400 hover:bg-red-500/10 transition-colors"
              title="Delete item"
            >
              <Trash2 className="h-3.5 w-3.5" />
            </button>
          )}
        </div>

        {item.context && (
          <p className="ml-7 text-xs text-slate-500">{item.context}</p>
        )}

        {/* Option cards + Other */}
        <div className="ml-7 space-y-1.5">
          {options.length > 0 && options.map((opt) => (
              <button
                key={opt.key}
                type="button"
                disabled={disabled}
                onClick={() => handleSelect(opt.key)}
                className={cn(
                  "w-full rounded-md border px-3 py-2 text-left transition-colors",
                  localSelected === opt.key
                    ? "border-emerald-500/40 bg-emerald-500/10"
                    : opt.recommended
                      ? "border-cyan-500/30 bg-cyan-500/[0.03] hover:border-cyan-500/50"
                      : "border-slate-600 bg-slate-800/50 hover:border-slate-500",
                  disabled && "opacity-50 cursor-not-allowed",
                )}
              >
                <div className="flex items-baseline gap-2">
                  <span className={cn(
                    "shrink-0 rounded px-1.5 py-0.5 text-[10px] font-bold",
                    localSelected === opt.key
                      ? "bg-emerald-500/20 text-emerald-400"
                      : "bg-slate-700 text-slate-400",
                  )}>
                    {opt.key}
                  </span>
                  <span className="text-sm text-slate-200">{opt.label}</span>
                  {opt.recommended && (
                    <span className="ml-auto flex items-center gap-0.5 rounded bg-cyan-500/15 px-1.5 py-0.5 text-[9px] font-medium text-cyan-400">
                      <Star className="h-2.5 w-2.5 fill-current" />
                      Recommended
                    </span>
                  )}
                </div>
                {opt.rationale && (
                  <p className="mt-1 ml-7 text-xs text-slate-500">{opt.rationale}</p>
                )}
              </button>
          ))}

          {/* Other / freeform option */}
            <button
              type="button"
              disabled={disabled}
              onClick={() => handleSelect(OTHER_KEY)}
              className={cn(
                "w-full rounded-md border px-3 py-2 text-left transition-colors",
                isOther
                  ? "border-emerald-500/40 bg-emerald-500/10"
                  : "border-slate-600 bg-slate-800/50 hover:border-slate-500",
                disabled && "opacity-50 cursor-not-allowed",
              )}
            >
              <div className="flex items-baseline gap-2">
                <span className={cn(
                  "shrink-0 rounded px-1.5 py-0.5 text-[10px] font-bold",
                  isOther ? "bg-emerald-500/20 text-emerald-400" : "bg-slate-700 text-slate-400",
                )}>
                  ...
                </span>
                <span className="text-sm text-slate-200">Other</span>
              </div>
            </button>

            {isOther && (
              <textarea
                className="w-full rounded-md border border-slate-600 bg-slate-800 px-3 py-2 text-base text-slate-200 placeholder-slate-500 focus:border-slate-500 focus:outline-none"
                placeholder="Describe your alternative..."
                value={localFreeform}
                onChange={(e) => handleFreeformChange(e.target.value)}
                disabled={disabled}
                rows={2}
              />
          )}
        </div>

        {/* Notes */}
        {isResolved && (
          <div className="ml-7">
            <textarea
              className="w-full rounded-md border border-slate-600 bg-slate-800 px-3 py-2 text-sm text-slate-200 placeholder-slate-500 focus:border-slate-500 focus:outline-none"
              placeholder="Notes (optional)..."
              value={localNotes}
              onChange={(e) => handleNotesChange(e.target.value)}
              disabled={disabled}
              rows={2}
            />
          </div>
        )}

        {/* Context note from previous clarification */}
        {item.context_note && (
          <div className="ml-7 mt-1 rounded border border-cyan-500/15 bg-cyan-500/5 px-2 py-1">
            <span className="text-[9px] font-medium text-cyan-400">Clarification note</span>
            <p className="text-xs text-slate-400">{item.context_note}</p>
          </div>
        )}
      </div>
    </div>
  );
}
