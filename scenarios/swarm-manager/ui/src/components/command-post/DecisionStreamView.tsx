/**
 * DecisionStreamView — Cross-item question stepper for Command Post.
 *
 * Aggregates pending questions from all backlog items into a single
 * "half asleep" flow: one question at a time, keyboard-driven, with
 * per-item context headers and snooze support.
 *
 * Mobile-first: 44px touch targets, safe-area-inset-bottom padding,
 * collapsible context panel, and maximized vertical content space.
 */
import { useState, useCallback, useRef, useEffect } from "react";
import { ChevronDown, ChevronLeft, ChevronRight, Loader2, SkipForward, ArrowLeft, Moon, CheckCircle2, Info } from "lucide-react";
import { cn } from "../../lib";
import { selectors } from "../../consts/selectors";
import { useBacklogStore } from "../../stores/backlog-store";
import { backlogService } from "../../services/backlog-service";
import { OTHER_KEY, parseWorkshopRound, buildWorkshopRoundContent } from "../../lib/workshop-files";
import {
  BACKLOG_KIND_ICONS,
  BACKLOG_KIND_LABELS,
  BACKLOG_STATUS_CHIP_COLORS,
  formatBacklogStatus,
} from "../../types";
import type { CrossItemQuestion } from "../../lib/command-post-utils";
import type { BacklogItem, BacklogKind } from "../../types";
import { QuestionAnswer, WorkshopQuestionView, ReviewQuestionView } from "../backlog/question-renderers";
import { TagList } from "../ui/tag-list";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface DecisionStreamResults {
  answeredCount: number;
  skippedCount: number;
  snoozedCount: number;
  unlockedItems: { kind: BacklogKind; name: string; title: string; action: "finalize" | "run" }[];
}

export interface DecisionStreamViewProps {
  questions: CrossItemQuestion[];
  onComplete: (results: DecisionStreamResults) => void;
  onBack: () => void;
  onSnoozeItem: (key: string) => void;
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/** Group questions by parent item key for batch completion. */
function groupByParent(questions: CrossItemQuestion[]): Map<string, CrossItemQuestion[]> {
  const map = new Map<string, CrossItemQuestion[]>();
  for (const ciq of questions) {
    const key = `${ciq.parentKind}/${ciq.parentName}`;
    const list = map.get(key);
    if (list) list.push(ciq);
    else map.set(key, [ciq]);
  }
  return map;
}

/** Build the snooze key matching snooze-utils pattern. */
function snoozeKey(kind: BacklogKind, name: string): string {
  return `backlog:${kind}/${name}`;
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function DecisionStreamView({
  questions,
  onComplete,
  onBack,
  onSnoozeItem,
}: DecisionStreamViewProps) {
  const [currentIndex, setCurrentIndex] = useState(0);
  const [localAnswers, setLocalAnswers] = useState<Map<string, QuestionAnswer>>(() => new Map());
  const [skippedIds, setSkippedIds] = useState<Set<string>>(() => new Set());
  const [snoozedItemKeys, setSnoozedItemKeys] = useState<Set<string>>(() => new Set());
  const [savingId, setSavingId] = useState<string | null>(null);
  const [saveError, setSaveError] = useState<string | null>(null);
  const [phase, setPhase] = useState<"answering" | "completing">("answering");
  const [contextExpanded, setContextExpanded] = useState(false);
  const [descExpanded, setDescExpanded] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);
  const prevParentRef = useRef<string>("");

  // Backlog store for item context lookup
  const backlogItems = useBacklogStore((s) => s.items);

  // Filter out questions from snoozed parent items
  const activeQuestions = questions.filter(
    (ciq) => !snoozedItemKeys.has(snoozeKey(ciq.parentKind, ciq.parentName)),
  );
  const total = activeQuestions.length;
  const safeIndex = Math.min(currentIndex, Math.max(0, total - 1));
  const current = activeQuestions[safeIndex] as CrossItemQuestion | undefined;
  const answer = current ? localAnswers.get(current.question.id) : undefined;

  // Look up full backlog item for context panel — try store first, then fetch
  const storeItem = current
    ? backlogItems.find((i) => i.kind === current.parentKind && i.name === current.parentName)
    : undefined;

  // Local cache for items fetched directly from the API (for items not in store,
  // e.g., archived items that still have pending questions on disk).
  const [fetchedItems, setFetchedItems] = useState<Map<string, BacklogItem>>(() => new Map());
  const fetchingRef = useRef<Set<string>>(new Set());

  const parentItemKey = current ? `${current.parentKind}/${current.parentName}` : "";
  const parentItem = storeItem ?? fetchedItems.get(parentItemKey);

  useEffect(() => {
    if (!current || storeItem || fetchedItems.has(parentItemKey) || fetchingRef.current.has(parentItemKey)) return;
    fetchingRef.current.add(parentItemKey);
    void backlogService.get(current.parentKind, current.parentName)
      .then((item) => {
        setFetchedItems((prev) => {
          const next = new Map(prev);
          next.set(parentItemKey, item);
          return next;
        });
      })
      .catch(() => {
        // Item may be deleted — leave as unavailable
      })
      .finally(() => {
        fetchingRef.current.delete(parentItemKey);
      });
  }, [current, storeItem, parentItemKey, fetchedItems]);

  // Auto-collapse context panel when navigating to a different parent item
  useEffect(() => {
    const parentKey = current ? `${current.parentKind}/${current.parentName}` : "";
    if (parentKey !== prevParentRef.current) {
      setContextExpanded(false);
      setDescExpanded(false);
      prevParentRef.current = parentKey;
    }
  }, [current]);

  // ---------------------------------------------------------------------------
  // Answer management
  // ---------------------------------------------------------------------------

  const updateAnswer = useCallback((questionId: string, patch: Partial<QuestionAnswer>) => {
    setLocalAnswers((prev) => {
      const next = new Map(prev);
      next.set(questionId, { ...prev.get(questionId), ...patch });
      return next;
    });
    setSaveError(null);
  }, []);

  // ---------------------------------------------------------------------------
  // Save (mirrors InlineQuestionStepper.saveAnswer)
  // ---------------------------------------------------------------------------

  const saveAnswer = useCallback(async (
    ciq: CrossItemQuestion,
    a: QuestionAnswer | undefined,
  ) => {
    if (!a) return;
    const q = ciq.question;
    setSavingId(q.id);
    setSaveError(null);
    try {
      if (q.source === "workshop" && q.round_number != null && a.selected?.trim()) {
        const roundNum = String(q.round_number).padStart(3, "0");
        const filePath = `workshop/round-${roundNum}.json`;
        const content = await backlogService.getFileContent(ciq.parentKind, ciq.parentName, filePath);
        const parsed = parseWorkshopRound(content);
        if (parsed.round) {
          const round = parsed.round;
          const item = (round.items ?? []).find((i) => i.id === q.id);
          if (item) {
            item.selected = a.selected === OTHER_KEY ? OTHER_KEY : a.selected;
            item.freeform = a.selected === OTHER_KEY ? (a.freeform ?? null) : null;
            item.notes = a.notes ?? null;
          }
          await backlogService.saveFileContent(
            ciq.parentKind, ciq.parentName, filePath,
            buildWorkshopRoundContent(round), "application/json",
          );
        }
      } else if (q.source === "review" && (a.reviewStatus === "approved" || a.reviewStatus === "flagged")) {
        await backlogService.batchReview(ciq.parentKind, ciq.parentName, [{
          id: q.id,
          type: q.review_type ?? "target",
          module_id: q.module_id,
          review_status: a.reviewStatus,
          review_comment: a.reviewComment ?? "",
        }]);
      }
    } catch {
      setSaveError("Save failed — will retry on next advance");
    } finally {
      setSavingId(null);
    }
  }, []);

  // ---------------------------------------------------------------------------
  // Completion — trigger workshopSave per item for auto-advance
  // ---------------------------------------------------------------------------

  const handleCompletion = useCallback(async () => {
    setPhase("completing");

    const answeredCount = Array.from(localAnswers.values()).filter((a) => {
      return (a.selected?.trim()) || a.reviewStatus === "approved" || a.reviewStatus === "flagged";
    }).length;

    const parentGroups = groupByParent(activeQuestions);
    const unlockedItems: DecisionStreamResults["unlockedItems"] = [];

    // For each parent item that had workshop answers, call workshopSave
    for (const [parentKey, groupQuestions] of parentGroups) {
      const workshopQ = groupQuestions.find(
        (ciq) => ciq.question.source === "workshop" && ciq.question.round_number != null,
      );
      if (!workshopQ || workshopQ.question.round_number == null) continue;

      // Check if any of this item's questions were answered
      const hasAnswers = groupQuestions.some((ciq) => {
        const a = localAnswers.get(ciq.question.id);
        return a && (a.selected?.trim() || a.reviewStatus === "approved" || a.reviewStatus === "flagged");
      });
      if (!hasAnswers) continue;

      try {
        const roundNumber = workshopQ.question.round_number;
        const roundNum = String(roundNumber).padStart(3, "0");
        const filePath = `workshop/round-${roundNum}.json`;
        const content = await backlogService.getFileContent(
          workshopQ.parentKind, workshopQ.parentName, filePath,
        );
        const result = await backlogService.workshopSave(
          workshopQ.parentKind, workshopQ.parentName, roundNumber, content,
        );
        if (result.autoAdvance?.triggered) {
          const [, name] = parentKey.split("/");
          unlockedItems.push({
            kind: workshopQ.parentKind,
            name: name ?? workshopQ.parentName,
            title: workshopQ.parentTitle,
            action: result.autoAdvance.nextMode === "finalize" ? "finalize" : "run",
          });
        }
      } catch {
        // Non-fatal: continue with other items
      }
    }

    onComplete({
      answeredCount,
      skippedCount: skippedIds.size,
      snoozedCount: snoozedItemKeys.size,
      unlockedItems,
    });
  }, [activeQuestions, localAnswers, skippedIds, snoozedItemKeys, onComplete]);

  // ---------------------------------------------------------------------------
  // Navigation
  // ---------------------------------------------------------------------------

  const isAllDone = useCallback(() => {
    return activeQuestions.every((ciq) => {
      if (skippedIds.has(ciq.question.id)) return true;
      const a = localAnswers.get(ciq.question.id);
      if (!a) return false;
      if (ciq.question.source === "workshop") return !!a.selected?.trim();
      return a.reviewStatus === "approved" || a.reviewStatus === "flagged";
    });
  }, [activeQuestions, skippedIds, localAnswers]);

  const advance = useCallback(async () => {
    if (!current) return;
    const a = localAnswers.get(current.question.id);
    if (a) {
      await saveAnswer(current, a);
    }
    if (safeIndex < total - 1) {
      setCurrentIndex(safeIndex + 1);
    }
    // Check if done after this advance
    if (isAllDone()) {
      handleCompletion();
    }
  }, [current, safeIndex, total, localAnswers, saveAnswer, isAllDone, handleCompletion]);

  const goBack = useCallback(() => {
    if (safeIndex > 0) {
      setCurrentIndex(safeIndex - 1);
      setSaveError(null);
    }
  }, [safeIndex]);

  const skip = useCallback(() => {
    if (!current) return;
    const newSkipped = new Set(skippedIds);
    newSkipped.add(current.question.id);
    setSkippedIds(newSkipped);
    if (safeIndex < total - 1) {
      setCurrentIndex(safeIndex + 1);
    }
    // Check completion with the new skipped set
    const allDone = activeQuestions.every((ciq) => {
      if (newSkipped.has(ciq.question.id)) return true;
      const a = localAnswers.get(ciq.question.id);
      if (!a) return false;
      if (ciq.question.source === "workshop") return !!a.selected?.trim();
      return a.reviewStatus === "approved" || a.reviewStatus === "flagged";
    });
    if (allDone) handleCompletion();
  }, [current, safeIndex, total, skippedIds, activeQuestions, localAnswers, handleCompletion]);

  const snoozeParent = useCallback(() => {
    if (!current) return;
    const key = snoozeKey(current.parentKind, current.parentName);
    const newSnoozed = new Set(snoozedItemKeys);
    newSnoozed.add(key);
    setSnoozedItemKeys(newSnoozed);
    onSnoozeItem(key);
    // Recalculate: the active list will shrink, safeIndex will auto-clamp
  }, [current, snoozedItemKeys, onSnoozeItem]);

  // ---------------------------------------------------------------------------
  // Keyboard shortcuts
  // ---------------------------------------------------------------------------

  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      // Suppress shortcuts when focus is in a textarea
      const tag = (e.target as HTMLElement)?.tagName;
      if (tag === "TEXTAREA" || tag === "INPUT") return;
      if (phase !== "answering" || !current) return;

      switch (e.key) {
        case "ArrowRight":
          e.preventDefault();
          advance();
          break;
        case "ArrowLeft":
          e.preventDefault();
          goBack();
          break;
        case "Enter":
          e.preventDefault();
          advance();
          break;
        case "s":
        case "S":
          e.preventDefault();
          snoozeParent();
          break;
        case "i":
        case "I":
          e.preventDefault();
          setContextExpanded((prev) => !prev);
          break;
        case "Escape":
          e.preventDefault();
          if (contextExpanded) {
            setContextExpanded(false);
          } else {
            onBack();
          }
          break;
        default: {
          // Number keys 1-9 to select workshop options
          const num = parseInt(e.key, 10);
          if (num >= 1 && num <= 9 && current.question.source === "workshop") {
            e.preventDefault();
            const options = current.question.options ?? [];
            // Options are 1-indexed for the user; OTHER is after all options
            if (num <= options.length) {
              const opt = options[num - 1];
              if (opt) {
                updateAnswer(current.question.id, {
                  selected: opt.key,
                  freeform: undefined,
                });
              }
            } else if (num === options.length + 1) {
              updateAnswer(current.question.id, { selected: OTHER_KEY });
            }
          }
          break;
        }
      }
    }

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [phase, current, advance, goBack, snoozeParent, onBack, updateAnswer, contextExpanded]);

  // ---------------------------------------------------------------------------
  // Completing phase
  // ---------------------------------------------------------------------------

  if (phase === "completing") {
    return (
      <div className="flex h-full items-center justify-center">
        <div className="flex flex-col items-center gap-3 text-slate-400">
          <Loader2 className="h-6 w-6 animate-spin text-cyan-400" />
          <p className="text-sm">Saving answers and checking auto-advance...</p>
        </div>
      </div>
    );
  }

  // ---------------------------------------------------------------------------
  // Empty state
  // ---------------------------------------------------------------------------

  if (total === 0) {
    return (
      <div className="flex h-full flex-col items-center justify-center gap-4">
        <CheckCircle2 className="h-10 w-10 text-emerald-400" />
        <p className="text-sm text-slate-300">No pending questions</p>
        <button
          type="button"
          onClick={onBack}
          className="flex min-h-[44px] items-center gap-1 rounded-lg border border-slate-600 px-4 py-2.5 text-sm text-slate-400 transition-colors hover:border-slate-500 hover:text-slate-200 active:bg-slate-800"
        >
          <ArrowLeft className="h-4 w-4" />
          Back to Command Post
        </button>
      </div>
    );
  }

  if (!current) return null;

  const isSaving = savingId === current.question.id;
  const isFirst = safeIndex === 0;
  const isLast = safeIndex === total - 1;
  const KindIcon = BACKLOG_KIND_ICONS[current.parentKind];
  const description = parentItem?.description ?? "";
  const descOverflows = description.length > 200 || description.includes("\n");

  return (
    <div ref={containerRef} className="flex h-full flex-col" data-testid={selectors.commandPost.decisionStream.container}>
      {/* Unified header — back, kind icon + title, counter + context toggle */}
      <div
        className="flex shrink-0 items-center gap-2 border-b border-slate-700/50 bg-slate-950 px-3"
        data-testid={selectors.commandPost.decisionStream.header}
      >
        <button
          type="button"
          onClick={onBack}
          className="flex min-h-[44px] shrink-0 items-center gap-1 rounded-lg py-2 pr-2 text-sm text-slate-400 transition-colors hover:bg-slate-800 hover:text-slate-200 active:bg-slate-700"
          data-testid={selectors.commandPost.decisionStream.backButton}
        >
          <ArrowLeft className="h-4 w-4" />
          <span className="hidden sm:inline">Command Post</span>
        </button>

        {/* Kind icon + title (center, truncated) */}
        <div className="flex min-w-0 flex-1 items-center gap-1.5 overflow-hidden">
          <KindIcon className="h-4 w-4 shrink-0 text-slate-500" aria-label={BACKLOG_KIND_LABELS[current.parentKind]} />
          <span className="truncate text-sm font-medium text-slate-200">
            {current.parentTitle}
          </span>
        </div>

        {/* Counter + context toggle */}
        <div className="flex shrink-0 items-center gap-1">
          <span
            className="text-xs tabular-nums text-slate-500"
            data-testid={selectors.commandPost.decisionStream.counter}
          >
            {safeIndex + 1}/{total}
          </span>
          <button
            type="button"
            onClick={() => setContextExpanded((prev) => !prev)}
            className="flex min-h-[44px] items-center rounded-lg p-2 text-slate-400 transition-colors hover:bg-slate-800 hover:text-slate-200 active:bg-slate-700"
            aria-label={contextExpanded ? "Hide item details" : "Show item details"}
            data-testid={selectors.commandPost.decisionStream.contextToggle}
          >
            <ChevronDown className={cn("h-4 w-4 transition-transform", contextExpanded && "rotate-180")} />
          </button>
        </div>
      </div>

      {/* Expandable item context panel */}
      {contextExpanded && (
        <div
          className="shrink-0 max-h-[40vh] overflow-y-auto border-b border-slate-700/50 bg-slate-900 px-3 py-2.5"
          data-testid={selectors.commandPost.decisionStream.contextPanel}
        >
          <div className="mx-auto max-w-2xl space-y-2">
            {parentItem ? (
              <>
                {/* Status + kind + priority row */}
                <div className="flex flex-wrap items-center gap-1.5">
                  <span className={cn("rounded px-1.5 py-0.5 text-[10px] font-medium", BACKLOG_STATUS_CHIP_COLORS[parentItem.status])}>
                    {formatBacklogStatus(parentItem.status)}
                  </span>
                  <span className="rounded bg-slate-700/60 px-1.5 py-0.5 text-[10px] font-medium text-slate-400">
                    {BACKLOG_KIND_LABELS[parentItem.kind]}
                  </span>
                  {parentItem.priority != null && (
                    <span className="rounded bg-slate-700/60 px-1.5 py-0.5 text-[10px] font-medium text-slate-400">
                      P{parentItem.priority}
                    </span>
                  )}
                  {parentItem.effort && (
                    <span className="rounded bg-slate-700/60 px-1.5 py-0.5 text-[10px] font-medium text-slate-400">
                      {parentItem.effort}
                    </span>
                  )}
                </div>

                {/* Description */}
                {description && (
                  <div>
                    <p className={cn("text-xs leading-relaxed text-slate-300", !descExpanded && "line-clamp-4")}>
                      {description}
                    </p>
                    {descOverflows && (
                      <button
                        type="button"
                        onClick={() => setDescExpanded((prev) => !prev)}
                        className="mt-0.5 text-[10px] font-medium text-blue-400 hover:text-blue-300"
                      >
                        {descExpanded ? "Show less" : "Show more\u2026"}
                      </button>
                    )}
                  </div>
                )}

                {/* Initiative */}
                {parentItem.initiative && (
                  <div className="flex items-center gap-1.5">
                    <Info className="h-3 w-3 shrink-0 text-slate-500" />
                    <span className="text-[10px] text-slate-500">Initiative:</span>
                    <span className="rounded bg-sky-500/15 px-1.5 py-0.5 text-[10px] font-medium text-sky-400">
                      {parentItem.initiative}
                    </span>
                  </div>
                )}

                {/* Tags */}
                {parentItem.tags && parentItem.tags.length > 0 && (
                  <TagList tags={parentItem.tags} maxTags={5} />
                )}
              </>
            ) : (
              <p className="text-xs text-slate-500">Item details not available</p>
            )}

            {/* Slug for reference */}
            <p className="text-[10px] font-mono text-slate-600">{current.parentKind}/{current.parentName}</p>
          </div>
        </div>
      )}

      {/* Question content */}
      <div
        className="flex-1 overflow-y-auto px-3 py-2"
        data-testid={selectors.commandPost.decisionStream.questionArea}
      >
        <div className="mx-auto max-w-2xl">
          {current.question.source === "workshop" ? (
            <WorkshopQuestionView
              question={current.question}
              answer={answer}
              disabled={isSaving}
              onUpdate={(patch) => updateAnswer(current.question.id, patch)}
            />
          ) : (
            <ReviewQuestionView
              question={current.question}
              answer={answer}
              disabled={isSaving}
              onUpdate={(patch) => updateAnswer(current.question.id, patch)}
            />
          )}
          {saveError && (
            <p className="mt-2 text-[10px] text-red-400">{saveError}</p>
          )}
        </div>
      </div>

      {/* Thin progress bar */}
      <div className="h-[3px] bg-slate-800" data-testid={selectors.commandPost.decisionStream.progressBar}>
        <div
          className="h-full bg-cyan-500/40 transition-all duration-300"
          style={{ width: `${((safeIndex + 1) / total) * 100}%` }}
        />
      </div>

      {/* Navigation row — 44px touch targets + safe area bottom */}
      <div
        className="border-t border-slate-700/50 px-3 py-2 pb-[calc(0.75rem+env(safe-area-inset-bottom))]"
        data-testid={selectors.commandPost.decisionStream.navBar}
      >
        <div className="mx-auto flex max-w-2xl items-center justify-between">
          <button
            type="button"
            disabled={isFirst}
            onClick={goBack}
            className={cn(
              "flex min-h-[44px] items-center gap-1 rounded-lg px-4 py-2.5 text-sm font-medium text-slate-400 transition-colors hover:bg-slate-800 hover:text-slate-200 active:bg-slate-700",
              isFirst && "opacity-30 cursor-not-allowed",
            )}
            data-testid={selectors.commandPost.decisionStream.navBack}
          >
            <ChevronLeft className="h-4 w-4" />
            Back
          </button>

          <div className="flex items-center gap-1">
            <button
              type="button"
              onClick={skip}
              className="flex min-h-[44px] items-center gap-1 rounded-lg px-3 py-2 text-xs text-slate-500 transition-colors hover:bg-slate-800 hover:text-slate-300 active:bg-slate-700"
              data-testid={selectors.commandPost.decisionStream.navSkip}
            >
              <SkipForward className="h-4 w-4" />
              Skip
            </button>
            <button
              type="button"
              onClick={snoozeParent}
              className="flex min-h-[44px] items-center gap-1 rounded-lg px-3 py-2 text-xs text-slate-500 transition-colors hover:bg-slate-800 hover:text-amber-400 active:bg-slate-700"
              title="Snooze this item (S)"
              data-testid={selectors.commandPost.decisionStream.navSnooze}
            >
              <Moon className="h-4 w-4" />
              Snooze
            </button>
          </div>

          <button
            type="button"
            disabled={isSaving}
            onClick={advance}
            className={cn(
              "flex min-h-[44px] items-center gap-1 rounded-lg px-4 py-2.5 text-sm font-medium text-slate-400 transition-colors hover:bg-slate-800 hover:text-slate-200 active:bg-slate-700",
              isSaving && "opacity-50 cursor-not-allowed",
            )}
            data-testid={selectors.commandPost.decisionStream.navNext}
          >
            {isSaving ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <>
                {isLast ? "Done" : "Next"}
                <ChevronRight className="h-4 w-4" />
              </>
            )}
          </button>
        </div>
      </div>
    </div>
  );
}
