/**
 * DecisionStreamView — Cross-item question stepper for Command Post.
 *
 * Aggregates pending questions from all backlog items into a single
 * "half asleep" flow: one question at a time, keyboard-driven, with
 * per-item context headers and snooze support.
 */
import { useState, useCallback, useRef, useEffect } from "react";
import { ChevronLeft, ChevronRight, Loader2, SkipForward, ArrowLeft, Moon, CheckCircle2 } from "lucide-react";
import { cn } from "../../lib";
import { backlogService } from "../../services/backlog-service";
import { OTHER_KEY, parseWorkshopRound, buildWorkshopRoundContent } from "../../lib/workshop-files";
import type { CrossItemQuestion } from "../../lib/command-post-utils";
import type { BacklogKind } from "../../types";
import { QuestionAnswer, WorkshopQuestionView, ReviewQuestionView } from "../backlog/question-renderers";

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
  const containerRef = useRef<HTMLDivElement>(null);

  // Filter out questions from snoozed parent items
  const activeQuestions = questions.filter(
    (ciq) => !snoozedItemKeys.has(snoozeKey(ciq.parentKind, ciq.parentName)),
  );
  const total = activeQuestions.length;
  const safeIndex = Math.min(currentIndex, Math.max(0, total - 1));
  const current = activeQuestions[safeIndex] as CrossItemQuestion | undefined;
  const answer = current ? localAnswers.get(current.question.id) : undefined;

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
        case "Escape":
          e.preventDefault();
          onBack();
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
  }, [phase, current, advance, goBack, snoozeParent, onBack, updateAnswer]);

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
          className="flex items-center gap-1 rounded-md border border-slate-600 px-3 py-1.5 text-xs text-slate-400 transition-colors hover:border-slate-500 hover:text-slate-200"
        >
          <ArrowLeft className="h-3.5 w-3.5" />
          Back to Command Post
        </button>
      </div>
    );
  }

  if (!current) return null;

  const isSaving = savingId === current.question.id;
  const isFirst = safeIndex === 0;
  const isLast = safeIndex === total - 1;

  return (
    <div ref={containerRef} className="flex h-full flex-col">
      {/* Top bar */}
      <div className="flex items-center justify-between border-b border-slate-700/50 px-3 py-1">
        <button
          type="button"
          onClick={onBack}
          className="flex items-center gap-1 text-xs text-slate-400 transition-colors hover:text-slate-200"
        >
          <ArrowLeft className="h-3.5 w-3.5" />
          Back to Command Post
        </button>
        <span className="text-[11px] text-slate-500">
          {safeIndex + 1} of {total}
        </span>
      </div>

      {/* Item context header */}
      <div className="border-b border-slate-700/30 bg-slate-800/40 px-3 py-1">
        <div className="flex items-center gap-2">
          <span className="shrink-0 rounded bg-slate-700 px-1.5 py-0.5 text-[10px] font-medium text-slate-400">
            {current.parentKind}
          </span>
          <span className="truncate text-sm font-medium text-slate-200">
            {current.parentTitle}
          </span>
          <span className="ml-auto shrink-0 text-[10px] text-slate-600">
            {current.parentName}
          </span>
        </div>
      </div>

      {/* Question content */}
      <div className="flex-1 overflow-y-auto px-3 py-2">
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

      {/* Navigation row */}
      <div className="border-t border-slate-700/50 px-3 py-1.5">
        <div className="mx-auto flex max-w-2xl items-center justify-between">
          <button
            type="button"
            disabled={isFirst}
            onClick={goBack}
            className={cn(
              "flex items-center gap-0.5 rounded-md px-2 py-1 text-xs text-slate-400 transition-colors hover:text-slate-200",
              isFirst && "opacity-30 cursor-not-allowed",
            )}
          >
            <ChevronLeft className="h-3.5 w-3.5" />
            Back
          </button>

          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={skip}
              className="text-[10px] text-slate-500 transition-colors hover:text-slate-300"
            >
              <SkipForward className="mr-0.5 inline h-3 w-3" />
              Skip
            </button>
            <button
              type="button"
              onClick={snoozeParent}
              className="flex items-center gap-0.5 text-[10px] text-slate-500 transition-colors hover:text-amber-400"
              title="Snooze this item (S)"
            >
              <Moon className="h-3 w-3" />
              Snooze
            </button>
          </div>

          <button
            type="button"
            disabled={isSaving}
            onClick={advance}
            className={cn(
              "flex items-center gap-0.5 rounded-md px-2 py-1 text-xs text-slate-400 transition-colors hover:text-slate-200",
              isSaving && "opacity-50 cursor-not-allowed",
            )}
          >
            {isSaving ? (
              <Loader2 className="h-3.5 w-3.5 animate-spin" />
            ) : (
              <>
                {isLast ? "Done" : "Next"}
                <ChevronRight className="h-3.5 w-3.5" />
              </>
            )}
          </button>
        </div>
      </div>

      {/* Progress bar */}
      <div className="relative flex h-6 items-center overflow-hidden bg-slate-900/60">
        <div
          className="absolute inset-y-0 left-0 bg-cyan-500/20 transition-all duration-300"
          style={{ width: `${((safeIndex + 1) / total) * 100}%` }}
        />
        <span className="relative z-10 mx-auto text-[10px] font-medium text-slate-400">
          {safeIndex + 1} of {total}
        </span>
      </div>
    </div>
  );
}
