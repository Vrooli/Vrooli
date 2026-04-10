/**
 * Inline question stepper for the All tab unified feed.
 * Shows one pending question at a time (TurboTax-style) with back/forward
 * navigation, auto-save on advance, and a flush-bottom progress bar.
 */
import { useState, useCallback } from "react";
import { ChevronLeft, ChevronRight, Loader2, SkipForward } from "lucide-react";
import { cn } from "../../lib";
import { selectors } from "../../consts/selectors";
import { backlogService } from "../../services/backlog-service";
import { OTHER_KEY, parseWorkshopRound, buildWorkshopRoundContent } from "../../lib/workshop-files";
import type { PendingQuestion, BacklogKind } from "../../types";
import { QuestionAnswer, WorkshopQuestionView, ReviewQuestionView } from "./question-renderers";

/** Result passed to onAllAnswered when workshop auto-advance is evaluated. */
export interface StepperCompletionResult {
  autoAdvance?: {
    triggered: boolean;
    runId?: string;
    taskId?: string;
    reason: string;
    nextMode?: "workshop" | "finalize";
    pending?: boolean;
    advanceAt?: string;
    delaySeconds?: number;
  };
}

interface InlineQuestionStepperProps {
  questions: PendingQuestion[];
  backlogKind: BacklogKind;
  backlogName: string;
  onAllAnswered: (result: StepperCompletionResult) => void;
}

export function InlineQuestionStepper({
  questions,
  backlogKind,
  backlogName,
  onAllAnswered,
}: InlineQuestionStepperProps) {
  // Snapshot the questions on mount so background summary refreshes don't
  // shift the list or index out from under the user mid-session.
  const [stableQuestions] = useState(() => questions);

  const [currentIndex, setCurrentIndex] = useState(0);
  const [localAnswers, setLocalAnswers] = useState<Map<string, QuestionAnswer>>(() => new Map());
  const [skippedIds, setSkippedIds] = useState<Set<string>>(() => new Set());
  const [savingId, setSavingId] = useState<string | null>(null);
  const [saveError, setSaveError] = useState<string | null>(null);

  const question = stableQuestions[currentIndex];
  const answer = question ? localAnswers.get(question.id) : undefined;
  const total = stableQuestions.length;

  const updateAnswer = useCallback((questionId: string, patch: Partial<QuestionAnswer>) => {
    setLocalAnswers((prev) => {
      const next = new Map(prev);
      next.set(questionId, { ...prev.get(questionId), ...patch });
      return next;
    });
    setSaveError(null);
  }, []);

  /** Save the current question's answer to the backend. */
  const saveAnswer = useCallback(async (q: PendingQuestion, a: QuestionAnswer | undefined) => {
    if (!a) return;
    setSavingId(q.id);
    setSaveError(null);
    try {
      if (q.source === "workshop" && q.round_number != null && a.selected?.trim()) {
        // Read-modify-write the workshop round file.
        const roundNum = String(q.round_number).padStart(3, "0");
        const filePath = `workshop/round-${roundNum}.json`;
        const content = await backlogService.getFileContent(backlogKind, backlogName, filePath);
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
            backlogKind, backlogName, filePath,
            buildWorkshopRoundContent(round), "application/json",
          );
        }
      } else if (q.source === "review" && (a.reviewStatus === "approved" || a.reviewStatus === "flagged")) {
        await backlogService.batchReview(backlogKind, backlogName, [{
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
  }, [backlogKind, backlogName]);

  const advance = useCallback(async () => {
    const q = stableQuestions[currentIndex] as PendingQuestion | undefined;
    if (!q) return;
    const a = localAnswers.get(q.id);
    // Auto-save if there's an answer
    if (a) {
      await saveAnswer(q, a);
    }
    if (currentIndex < total - 1) {
      setCurrentIndex(currentIndex + 1);
    }
  }, [currentIndex, total, stableQuestions, localAnswers, saveAnswer]);

  const goBack = useCallback(() => {
    if (currentIndex > 0) {
      setCurrentIndex(currentIndex - 1);
      setSaveError(null);
    }
  }, [currentIndex]);

  const skip = useCallback(() => {
    if (!question) return;
    const newSkipped = new Set(skippedIds);
    newSkipped.add(question.id);
    setSkippedIds(newSkipped);
    if (currentIndex < total - 1) {
      setCurrentIndex(currentIndex + 1);
    }
  }, [currentIndex, total, question, skippedIds]);

  /** Finish: save the last answer and close the stepper (no auto-advance). */
  const finish = useCallback(async () => {
    const q = stableQuestions[currentIndex] as PendingQuestion | undefined;
    if (!q) return;
    const a = localAnswers.get(q.id);
    if (a) {
      await saveAnswer(q, a);
    }
    onAllAnswered({});
  }, [currentIndex, stableQuestions, localAnswers, saveAnswer, onAllAnswered]);

  if (!question) return null;

  const isSaving = savingId === question.id;
  const isFirst = currentIndex === 0;
  const isLast = currentIndex === total - 1;

  return (
    <div
      data-testid={selectors.questionStepper.container}
      className="mt-3 rounded-lg border border-amber-500/20 bg-amber-500/[0.03] p-3"
      onClick={(e) => { e.preventDefault(); e.stopPropagation(); }}
    >
      {/* Question content */}
      <div className="min-h-[140px]">
        {question.source === "workshop" ? (
          <WorkshopQuestionView
            question={question}
            answer={answer}
            disabled={isSaving}
            onUpdate={(patch) => updateAnswer(question.id, patch)}
          />
        ) : (
          <ReviewQuestionView
            question={question}
            answer={answer}
            disabled={isSaving}
            onUpdate={(patch) => updateAnswer(question.id, patch)}
          />
        )}
      </div>

      {/* Save error */}
      {saveError && (
        <p className="mt-1 text-[10px] text-red-400">{saveError}</p>
      )}

      {/* Navigation row */}
      <div className="mt-2 flex items-center justify-between">
        <button
          type="button"
          data-testid={selectors.questionStepper.prevButton}
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

        <button
          type="button"
          data-testid={selectors.questionStepper.skipButton}
          onClick={skip}
          className="text-[10px] text-slate-500 transition-colors hover:text-slate-300"
        >
          <SkipForward className="mr-0.5 inline h-3 w-3" />
          Skip
        </button>

        <button
          type="button"
          data-testid={selectors.questionStepper.nextButton}
          disabled={isSaving}
          onClick={isLast ? finish : advance}
          className={cn(
            "flex items-center gap-0.5 rounded-md px-2 py-1 text-xs text-slate-400 transition-colors hover:text-slate-200",
            isSaving && "opacity-50 cursor-not-allowed",
          )}
        >
          {isSaving ? (
            <Loader2 className="h-3.5 w-3.5 animate-spin" />
          ) : (
            <>
              {isLast ? "Finish" : "Next"}
              <ChevronRight className="h-3.5 w-3.5" />
            </>
          )}
        </button>
      </div>

      {/* Progress bar — flush to bottom edge of stepper container */}
      <div
        data-testid={selectors.questionStepper.progress}
        className="relative -mx-3 -mb-3 mt-2 flex h-6 items-center overflow-hidden rounded-b-lg bg-slate-900/60"
      >
        <div
          className="absolute inset-y-0 left-0 bg-cyan-500/20 transition-all duration-300"
          style={{ width: `${((currentIndex + 1) / total) * 100}%` }}
        />
        <span className="relative z-10 mx-auto text-[10px] font-medium text-slate-400">
          {currentIndex + 1} of {total}
        </span>
      </div>
    </div>
  );
}

