/**
 * Renders a single workshop item (decision or info).
 */
import { useState } from "react";
import { cn } from "../../lib";
import type { WorkshopItem } from "../../types/domain";

const OTHER_KEY = "__other__";

interface WorkshopItemCardProps {
  item: WorkshopItem;
  disabled?: boolean;
  onUpdate?: (updated: WorkshopItem) => void;
}

export function WorkshopItemCard({ item, disabled, onUpdate }: WorkshopItemCardProps) {
  const [localSelected, setLocalSelected] = useState(item.selected ?? "");
  const [localFreeform, setLocalFreeform] = useState(item.freeform ?? "");
  const [localNotes, setLocalNotes] = useState(item.notes ?? "");

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
          <p className="text-sm text-slate-300">{item.topic || item.text}</p>
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
  const options = item.options ?? [];

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
          <p className="text-sm font-medium text-slate-200">{item.topic || item.text}</p>
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
                className="w-full rounded-md border border-slate-600 bg-slate-800 px-3 py-2 text-sm text-slate-200 placeholder-slate-500 focus:border-slate-500 focus:outline-none"
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
      </div>
    </div>
  );
}
