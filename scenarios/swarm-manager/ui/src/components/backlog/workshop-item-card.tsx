/**
 * Renders a single workshop item (decision or info).
 */
import { useMemo, useState } from "react";
import { CheckCircle2, ChevronDown, ChevronRight, MessageCircleMore, Trash2 } from "lucide-react";
import { cn } from "../../lib";
import { OTHER_KEY, filterAgentOther } from "../../lib/workshop-files";
import { renderInlineMarkdown, renderMarkdown } from "../../lib/render-markdown";
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
  expanded?: boolean;
  onToggleExpanded?: () => void;
}

function MarkdownPreview({
  content,
  className,
}: {
  content?: string | null;
  className?: string;
}) {
  if (!content?.trim()) return null;

  return (
    <div
      className={cn("prose-sm-slate break-words [overflow-wrap:anywhere]", className)}
      dangerouslySetInnerHTML={{ __html: renderMarkdown(content) }}
    />
  );
}

export function WorkshopItemCard({
  item,
  disabled,
  onUpdate,
  onDelete,
  backlogKind,
  backlogName,
  roundNumber,
  expanded = false,
  onToggleExpanded,
}: WorkshopItemCardProps) {
  const [localSelected, setLocalSelected] = useState(item.selected ?? "");
  const [localFreeform, setLocalFreeform] = useState(item.freeform ?? "");
  const [localNotes, setLocalNotes] = useState(item.notes ?? "");
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [rationaleOverrides, setRationaleOverrides] = useState<Map<string, boolean>>(() => new Map());

  const clarificationStore = useClarificationStore();
  const isClarifyActive = clarificationStore.isOpen && clarificationStore.target?.itemId === item.id;
  const isResolved = !!localSelected.trim();
  const isOther = localSelected === OTHER_KEY;
  const options = useMemo(() => filterAgentOther(item.options ?? []), [item.options]);
  const selectedOption = useMemo(
    () => options.find((opt) => opt.key === localSelected),
    [options, localSelected],
  );
  const title = item.topic?.trim() || item.text?.trim() || "Untitled decision";
  const body = item.text?.trim() && item.text?.trim() !== item.topic?.trim() ? item.text.trim() : "";
  const summaryLabel = isOther
    ? (localFreeform.trim() ? `Other · ${localFreeform.trim()}` : "Other")
    : selectedOption?.label ?? "";

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

  const handleSelectionChange = (nextSelected: string, nextFreeform?: string, nextNotes?: string) => {
    setLocalSelected(nextSelected);
    if (nextFreeform !== undefined) {
      setLocalFreeform(nextFreeform);
    }
    if (nextNotes !== undefined) {
      setLocalNotes(nextNotes);
    }
    onUpdate?.({
      ...item,
      selected: nextSelected || null,
      freeform: nextSelected === OTHER_KEY ? ((nextFreeform ?? localFreeform) || null) : null,
      notes: (nextNotes ?? localNotes) || null,
    });
  };

  const toggleRationale = (key: string) => {
    setRationaleOverrides((prev) => {
      const next = new Map(prev);
      const isSelectedOption = localSelected === key;
      const defaultExpanded = isSelectedOption;
      const currentExpanded = next.has(key) ? next.get(key) ?? defaultExpanded : defaultExpanded;

      next.set(key, !currentExpanded);
      return next;
    });
  };

  // Info items — read-only text
  if (item.type === "info") {
    return (
      <div className="rounded-md border border-slate-700 bg-slate-800/30 p-3">
        <div className="flex items-start gap-2">
          <button
            type="button"
            onClick={onToggleExpanded}
            className="mt-0.5 rounded p-0.5 text-slate-500 transition-colors hover:bg-slate-700/50 hover:text-slate-200"
            aria-label={expanded ? "Collapse info item" : "Expand info item"}
          >
            {expanded ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
          </button>
          <span className="mt-0.5 rounded bg-blue-500/20 px-1.5 py-0.5 text-[10px] font-medium text-blue-400">Info</span>
          <div className="min-w-0 flex-1">
            <div
              className={cn(
                "break-words text-sm font-medium text-slate-300 [overflow-wrap:anywhere]",
                !expanded && "line-clamp-2",
              )}
              dangerouslySetInnerHTML={{ __html: renderInlineMarkdown(title) }}
            />
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
        {expanded && (
          <>
            <MarkdownPreview content={body} className="mt-2 text-sm leading-relaxed text-slate-300" />
            <MarkdownPreview content={item.context} className="mt-2 text-xs leading-relaxed text-slate-500" />
          </>
        )}
      </div>
    );
  }

  return (
    <div className={cn(
      "rounded-md border",
      isResolved ? "border-emerald-500/20 bg-emerald-500/5" : "border-amber-500/20 bg-amber-500/5",
    )}>
      <div className="border-b border-white/5 px-3 py-2.5">
        <div className="flex items-center justify-between gap-2">
          <div className="flex min-w-0 items-center gap-1">
            <button
              type="button"
              onClick={onToggleExpanded}
              className="rounded p-0.5 text-slate-500 transition-colors hover:bg-slate-700/50 hover:text-slate-200"
              aria-label={expanded ? "Collapse decision" : "Expand decision"}
            >
              {expanded ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
            </button>
            <span className={cn(
              "rounded px-1.5 py-0.5 text-[10px] font-medium",
              isResolved ? "bg-emerald-500/20 text-emerald-400" : "bg-amber-500/20 text-amber-400",
            )}>
              D
            </span>
            {isResolved ? (
              <span className="rounded bg-emerald-500/10 px-1.5 py-0.5 text-[10px] font-medium text-emerald-300">
                Answered
              </span>
            ) : (
              <span className="rounded bg-amber-500/10 px-1.5 py-0.5 text-[10px] font-medium text-amber-300">
                Pending
              </span>
            )}
            {!!item.clarification_id && (
              <span className="rounded bg-cyan-500/10 px-1.5 py-0.5 text-[10px] font-medium text-cyan-300">
                Clarified
              </span>
            )}
          </div>
          <div className="flex shrink-0 items-center gap-1">
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
          </div>
        </div>
        <div className="mt-2 min-w-0">
          <div
            className={cn(
              "break-words text-base font-medium leading-snug text-slate-100 [overflow-wrap:anywhere]",
              !expanded && "line-clamp-3",
            )}
            dangerouslySetInnerHTML={{ __html: renderInlineMarkdown(title) }}
          />
          {summaryLabel && (
            <div
              className="mt-1 line-clamp-1 break-words text-xs text-slate-400 [overflow-wrap:anywhere]"
              dangerouslySetInnerHTML={{ __html: renderInlineMarkdown(summaryLabel) }}
            />
          )}
        </div>
      </div>

      {expanded && (
        <div className="space-y-3 px-3 py-3">
          <MarkdownPreview content={body} className="text-sm leading-relaxed text-slate-300" />
          <MarkdownPreview content={item.context} className="text-xs leading-relaxed text-slate-500" />

          <div className="space-y-2">
            {options.map((opt) => {
              const isSelected = localSelected === opt.key;
              const rationaleExpanded = rationaleOverrides.has(opt.key)
                ? rationaleOverrides.get(opt.key) === true
                : isSelected;
              const rationaleOverflows = (opt.rationale?.length ?? 0) > 140 || (opt.rationale?.includes("\n") ?? false);
              return (
                <button
                  key={opt.key}
                  type="button"
                  disabled={disabled}
                  onClick={() => handleSelectionChange(opt.key)}
                  className={cn(
                    "w-full rounded-md border px-3 py-2 text-left transition-colors",
                    isSelected
                      ? "border-emerald-500/40 bg-emerald-500/10"
                      : opt.recommended
                        ? "border-cyan-500/30 bg-cyan-500/[0.03] hover:border-cyan-500/50"
                        : "border-slate-600 bg-slate-800/50 hover:border-slate-500",
                    disabled && "opacity-50 cursor-not-allowed",
                  )}
                >
                  <div className="flex min-w-0 items-start gap-2">
                    <span className={cn(
                      "mt-0.5 shrink-0 rounded px-1.5 py-0.5 text-[10px] font-bold",
                      isSelected ? "bg-emerald-500/20 text-emerald-400" : "bg-slate-700 text-slate-400",
                    )}>
                      {opt.key}
                    </span>
                    <div className="min-w-0 flex-1">
                      <div
                        className="break-words text-sm text-slate-200 [overflow-wrap:anywhere]"
                        dangerouslySetInnerHTML={{ __html: renderInlineMarkdown(opt.label) }}
                      />
                    </div>
                    {opt.recommended && (
                      <span className="mt-0.5 shrink-0 rounded bg-cyan-500/15 px-1.5 py-0.5 text-[9px] font-medium text-cyan-400">
                        Rec
                      </span>
                    )}
                    {isSelected && (
                      <CheckCircle2 className="mt-0.5 h-4 w-4 shrink-0 text-emerald-400" />
                    )}
                  </div>
                  {!!opt.rationale && (
                    <div className="mt-1 pl-7">
                      <MarkdownPreview
                        content={opt.rationale}
                        className={cn("text-xs leading-relaxed text-slate-500", !rationaleExpanded && "line-clamp-2")}
                      />
                      {rationaleOverflows && (
                        <button
                          type="button"
                          disabled={disabled}
                          onClick={(e) => {
                            e.stopPropagation();
                            toggleRationale(opt.key);
                          }}
                          className="mt-1 text-[11px] font-medium text-blue-400 hover:text-blue-300"
                        >
                          {rationaleExpanded ? "Show less" : "Show more…"}
                        </button>
                      )}
                    </div>
                  )}
                </button>
              );
            })}

            <button
              type="button"
              disabled={disabled}
              onClick={() => handleSelectionChange(OTHER_KEY)}
              className={cn(
                "w-full rounded-md border px-3 py-2 text-left transition-colors",
                isOther
                  ? "border-emerald-500/40 bg-emerald-500/10"
                  : "border-slate-600 bg-slate-800/50 hover:border-slate-500",
                disabled && "opacity-50 cursor-not-allowed",
              )}
            >
              <div className="flex min-w-0 items-start gap-2">
                <span className={cn(
                  "mt-0.5 shrink-0 rounded px-1.5 py-0.5 text-[10px] font-bold",
                  isOther ? "bg-emerald-500/20 text-emerald-400" : "bg-slate-700 text-slate-400",
                )}>
                  ...
                </span>
                <div className="min-w-0 flex-1 text-sm text-slate-200">Other</div>
                {isOther && (
                  <CheckCircle2 className="mt-0.5 h-4 w-4 shrink-0 text-emerald-400" />
                )}
              </div>
            </button>

            {isOther && (
              <textarea
                className="w-full rounded-md border border-slate-600 bg-slate-800 px-3 py-2 text-sm text-slate-200 placeholder-slate-500 focus:border-slate-500 focus:outline-none"
                placeholder="Describe your alternative..."
                value={localFreeform}
                onChange={(e) => handleSelectionChange(OTHER_KEY, e.target.value)}
                disabled={disabled}
                rows={3}
              />
            )}
          </div>

          {isResolved && (
            <textarea
              className="w-full rounded-md border border-slate-600 bg-slate-800 px-3 py-2 text-sm text-slate-200 placeholder-slate-500 focus:border-slate-500 focus:outline-none"
              placeholder="Notes (optional)..."
              value={localNotes}
              onChange={(e) => handleSelectionChange(localSelected, undefined, e.target.value)}
              disabled={disabled}
              rows={2}
            />
          )}

          {item.context_note && (
            <div className="rounded border border-cyan-500/15 bg-cyan-500/5 px-2 py-1">
              <div className="flex items-center gap-1 text-[10px] font-medium text-cyan-400">
                <MessageCircleMore className="h-3 w-3" />
                Clarification note
              </div>
              <MarkdownPreview content={item.context_note} className="mt-1 text-xs text-slate-400" />
            </div>
          )}
        </div>
      )}
    </div>
  );
}
