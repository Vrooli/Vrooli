/**
 * ScenarioNavigatorPopover — Shows parent items in the decision stream
 * as a popover, allowing quick navigation and per-parent snooze.
 */
import { useEffect, useRef } from "react";
import { Clock } from "lucide-react";
import { cn } from "../../lib";
import { selectors } from "../../consts/selectors";
import { BACKLOG_KIND_ICONS } from "../../types";
import type { BacklogKind } from "../../types";
import type { CrossItemQuestion } from "../../lib/command-post-utils";
import type { QuestionAnswer } from "../backlog/question-renderers";

export interface ScenarioNavigatorPopoverProps {
  isOpen: boolean;
  onClose: () => void;
  parentGroups: Map<string, CrossItemQuestion[]>;
  currentParentKey: string;
  localAnswers: Map<string, QuestionAnswer>;
  skippedIds: Set<string>;
  onJumpTo: (parentKey: string) => void;
  onSnoozeParent: (kind: BacklogKind, name: string) => void;
}

function parseParentKey(key: string): { kind: BacklogKind; name: string } {
  const slashIdx = key.indexOf("/");
  return {
    kind: key.slice(0, slashIdx) as BacklogKind,
    name: key.slice(slashIdx + 1),
  };
}

export function ScenarioNavigatorPopover({
  isOpen,
  onClose,
  parentGroups,
  currentParentKey,
  localAnswers,
  skippedIds,
  onJumpTo,
  onSnoozeParent,
}: ScenarioNavigatorPopoverProps) {
  const popoverRef = useRef<HTMLDivElement>(null);

  // Click-outside handler
  useEffect(() => {
    if (!isOpen) return;
    function handleClickOutside(e: MouseEvent) {
      if (popoverRef.current && !popoverRef.current.contains(e.target as Node)) {
        onClose();
      }
    }
    // Defer to next tick so the opening click doesn't immediately close
    const id = setTimeout(() => {
      document.addEventListener("mousedown", handleClickOutside);
    }, 0);
    return () => {
      clearTimeout(id);
      document.removeEventListener("mousedown", handleClickOutside);
    };
  }, [isOpen, onClose]);

  if (!isOpen) return null;

  const entries = Array.from(parentGroups.entries());

  return (
    <div
      ref={popoverRef}
      className="absolute right-0 top-full z-50 mt-1 max-h-64 min-w-[220px] overflow-y-auto rounded-lg border border-slate-700 bg-slate-800 shadow-lg"
      data-testid={selectors.commandPost.decisionStream.navigatorPopover}
    >
      {entries.length === 0 ? (
        <p className="px-3 py-2 text-xs text-slate-500">No items</p>
      ) : (
        entries.map(([parentKey, questions], idx) => {
          const { kind, name } = parseParentKey(parentKey);
          const KindIcon = BACKLOG_KIND_ICONS[kind];
          const title = questions[0]?.parentTitle ?? name;
          const isCurrent = parentKey === currentParentKey;

          // Count answered questions for this parent
          const answeredCount = questions.filter((ciq) => {
            if (skippedIds.has(ciq.question.id)) return false;
            const a = localAnswers.get(ciq.question.id);
            if (!a) return false;
            if (ciq.question.source === "workshop") return !!a.selected?.trim();
            return a.reviewStatus === "approved" || a.reviewStatus === "flagged";
          }).length;

          return (
            <div
              key={parentKey}
              className={cn(
                "flex items-center gap-2 px-3 py-2 cursor-pointer transition-colors hover:bg-slate-700/60",
                isCurrent && "bg-cyan-500/10 border-l-2 border-cyan-500/30",
                !isCurrent && "border-l-2 border-transparent",
              )}
              data-testid={selectors.commandPost.decisionStream.navigatorRow}
              onClick={() => {
                onJumpTo(parentKey);
                onClose();
              }}
            >
              {KindIcon && <KindIcon className="h-4 w-4 shrink-0 text-slate-500" />}
              <span className="min-w-0 flex-1 truncate text-xs text-slate-300">
                {idx + 1}. {title}
              </span>
              <span className="shrink-0 rounded bg-slate-700/60 px-1.5 py-0.5 text-[10px] tabular-nums text-slate-400">
                {answeredCount}/{questions.length}
              </span>
              <button
                type="button"
                className="shrink-0 rounded p-1 text-slate-500 transition-colors hover:bg-slate-600 hover:text-amber-400"
                title="Snooze this item"
                data-testid={selectors.commandPost.decisionStream.navigatorSnooze}
                onClick={(e) => {
                  e.stopPropagation();
                  onSnoozeParent(kind, name);
                }}
              >
                <Clock className="h-3.5 w-3.5" />
              </button>
            </div>
          );
        })
      )}
    </div>
  );
}
