/**
 * ScenarioNavigatorPopover — Shows parent items in the decision stream
 * as a popover, allowing quick navigation and per-parent snooze.
 */
import { useCallback, useEffect, useRef } from "react";
import { Clock, X } from "lucide-react";
import { cn } from "../../lib";
import { selectors } from "../../consts/selectors";
import { BACKLOG_KIND_ICONS } from "../../types";
import type { BacklogKind } from "../../types";
import type { CrossItemQuestion } from "../../lib/command-post-utils";
import type { QuestionAnswer } from "../backlog/question-renderers";
import { useIsMobile } from "../../hooks/useMediaQuery";
import { BottomSheet } from "../ui/bottom-sheet";

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
  const isMobile = useIsMobile();

  const clampToViewport = useCallback(() => {
    const el = popoverRef.current;
    if (!el) return;
    el.style.transform = "";
    el.style.top = "";
    el.style.bottom = "";
    el.style.marginTop = "";
    el.style.marginBottom = "";

    const rect = el.getBoundingClientRect();
    const margin = 8;
    let shiftX = 0;
    if (rect.left < margin) shiftX = margin - rect.left;
    else if (rect.right > window.innerWidth - margin) shiftX = window.innerWidth - margin - rect.right;
    if (shiftX !== 0) el.style.transform = `translateX(${shiftX}px)`;
    if (rect.bottom > window.innerHeight - margin) {
      el.style.top = "auto";
      el.style.bottom = "100%";
      el.style.marginTop = "0";
      el.style.marginBottom = "4px";
    }
  }, []);

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

  const entries = Array.from(parentGroups.entries());

  useEffect(() => {
    if (!isOpen || isMobile) return;
    clampToViewport();
    const handle = () => clampToViewport();
    window.addEventListener("resize", handle);
    window.addEventListener("scroll", handle, true);
    return () => {
      window.removeEventListener("resize", handle);
      window.removeEventListener("scroll", handle, true);
    };
  }, [isOpen, isMobile, clampToViewport, entries.length]);

  if (!isOpen) return null;

  const content = (
    <>
      {entries.length === 0 ? (
        <p className="px-3 py-2 text-xs text-slate-500">No items</p>
      ) : (
        entries.map(([parentKey, questions], idx) => {
          const { kind, name } = parseParentKey(parentKey);
          const KindIcon = BACKLOG_KIND_ICONS[kind];
          const title = questions[0]?.parentTitle ?? name;
          const isCurrent = parentKey === currentParentKey;

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
                "flex cursor-pointer items-center gap-2 px-3 py-2 transition-colors hover:bg-slate-700/60",
                isCurrent && "border-l-2 border-cyan-500/30 bg-cyan-500/10",
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
    </>
  );

  if (isMobile) {
    return (
      <BottomSheet
        isOpen={isOpen}
        onClose={onClose}
        title="Decision navigator"
        description={`${entries.length} active item${entries.length === 1 ? "" : "s"}`}
        containerClassName="z-[90]"
        data-testid={selectors.commandPost.decisionStream.navigatorPopover}
      >
        <div className="-mx-4">{content}</div>
      </BottomSheet>
    );
  }

  return (
    <>
      <button
        type="button"
        className="fixed inset-0 z-[80] cursor-default bg-transparent"
        aria-label="Close decision navigator"
        onClick={onClose}
      />
      <div
        ref={popoverRef}
        className="absolute right-0 top-full z-[90] mt-1 w-[280px] max-w-[calc(100vw-1rem)] overflow-hidden rounded-lg border border-slate-700 bg-slate-800 shadow-lg"
        data-testid={selectors.commandPost.decisionStream.navigatorPopover}
      >
        <div className="flex items-center justify-between border-b border-slate-700/70 px-3 py-2">
          <p className="text-sm font-semibold text-slate-100">Decision navigator</p>
          <button type="button" className="rounded p-1 text-slate-400 hover:bg-slate-700 hover:text-slate-200" onClick={onClose}>
            <X className="h-4 w-4" />
          </button>
        </div>
        <div className="max-h-72 overflow-y-auto">{content}</div>
      </div>
    </>
  );
}
