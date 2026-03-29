/**
 * Inline question stepper for the All tab unified feed.
 * Shows one pending question at a time (TurboTax-style) with back/forward
 * navigation, auto-save on advance, and a flush-bottom progress bar.
 */
import { useState, useCallback, useRef } from "react";
import { ChevronLeft, ChevronRight, Loader2, SkipForward, CheckCircle2, AlertTriangle, Star } from "lucide-react";
import { cn } from "../../lib";
import { selectors } from "../../consts/selectors";
import { backlogService } from "../../services/backlog-service";
import { OTHER_KEY, filterAgentOther, parseWorkshopRound, buildWorkshopRoundContent } from "../../lib/workshop-files";
import type { PendingQuestion, BacklogKind, ReviewStatus } from "../../types";

/** Result passed to onAllAnswered when workshop auto-advance is evaluated. */
export interface StepperCompletionResult {
  autoAdvance?: {
    triggered: boolean;
    runId?: string;
    taskId?: string;
    reason: string;
    nextMode?: "workshop" | "finalize";
  };
}

interface InlineQuestionStepperProps {
  questions: PendingQuestion[];
  backlogKind: BacklogKind;
  backlogName: string;
  onAllAnswered: (result: StepperCompletionResult) => void;
}

/** Local answer state for a single question. */
interface QuestionAnswer {
  // Workshop fields
  selected?: string;
  freeform?: string;
  notes?: string;
  // Review fields
  reviewStatus?: ReviewStatus;
  reviewComment?: string;
}

export function InlineQuestionStepper({
  questions,
  backlogKind,
  backlogName,
  onAllAnswered,
}: InlineQuestionStepperProps) {
  const [currentIndex, setCurrentIndex] = useState(0);
  const [localAnswers, setLocalAnswers] = useState<Map<string, QuestionAnswer>>(() => new Map());
  const [skippedIds, setSkippedIds] = useState<Set<string>>(() => new Set());
  const [savingId, setSavingId] = useState<string | null>(null);
  const [saveError, setSaveError] = useState<string | null>(null);
  const completionFired = useRef(false);

  const question = questions[currentIndex];
  const answer = question ? localAnswers.get(question.id) : undefined;
  const total = questions.length;

  const updateAnswer = useCallback((questionId: string, patch: Partial<QuestionAnswer>) => {
    setLocalAnswers((prev) => {
      const next = new Map(prev);
      next.set(questionId, { ...prev.get(questionId), ...patch });
      return next;
    });
    setSaveError(null);
  }, []);

  /** Check if all questions are answered or skipped. If so, trigger workshopSave for auto-advance. */
  const checkCompletion = useCallback((answers: Map<string, QuestionAnswer>, skipped: Set<string>) => {
    if (completionFired.current) return;
    const allDone = questions.every((q) => {
      if (skipped.has(q.id)) return true;
      const a = answers.get(q.id);
      if (!a) return false;
      if (q.source === "workshop") return !!a.selected?.trim();
      return a.reviewStatus === "approved" || a.reviewStatus === "flagged";
    });
    if (!allDone) return;
    completionFired.current = true;

    // Find the workshop round number so we can call workshopSave() for auto-advance.
    const workshopQ = questions.find((q) => q.source === "workshop" && q.round_number != null);
    if (!workshopQ || workshopQ.round_number == null) {
      // No workshop questions (review-only) — complete without auto-advance.
      onAllAnswered({});
      return;
    }

    const roundNumber = workshopQ.round_number;
    const roundNum = String(roundNumber).padStart(3, "0");
    const filePath = `workshop/round-${roundNum}.json`;

    // Read the saved round file and call workshopSave() to trigger auto-advance evaluation.
    (async () => {
      try {
        const content = await backlogService.getFileContent(backlogKind, backlogName, filePath);
        const result = await backlogService.workshopSave(backlogKind, backlogName, roundNumber, content);
        onAllAnswered({ autoAdvance: result.autoAdvance });
      } catch {
        // If workshopSave fails, still notify parent that all questions are answered.
        onAllAnswered({});
      }
    })();
  }, [questions, onAllAnswered, backlogKind, backlogName]);

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
    const q = questions[currentIndex] as PendingQuestion | undefined;
    if (!q) return;
    const a = localAnswers.get(q.id);
    // Auto-save if there's an answer
    if (a) {
      await saveAnswer(q, a);
    }
    if (currentIndex < total - 1) {
      setCurrentIndex(currentIndex + 1);
    }
    checkCompletion(localAnswers, skippedIds);
  }, [currentIndex, total, questions, localAnswers, skippedIds, saveAnswer, checkCompletion]);

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
    checkCompletion(localAnswers, newSkipped);
  }, [currentIndex, total, question, skippedIds, localAnswers, checkCompletion]);

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

// ---------------------------------------------------------------------------
// Workshop question sub-renderer
// ---------------------------------------------------------------------------

interface WorkshopQuestionViewProps {
  question: PendingQuestion;
  answer: QuestionAnswer | undefined;
  disabled: boolean;
  onUpdate: (patch: Partial<QuestionAnswer>) => void;
}

function WorkshopQuestionView({ question, answer, disabled, onUpdate }: WorkshopQuestionViewProps) {
  const selected = answer?.selected ?? question.selected ?? "";
  const freeform = answer?.freeform ?? question.freeform ?? "";
  const isOther = selected === OTHER_KEY;
  const options = filterAgentOther(question.options ?? []);

  const handleSelect = (key: string) => {
    onUpdate({
      selected: key,
      freeform: key === OTHER_KEY ? freeform : undefined,
    });
  };

  return (
    <div className="space-y-2">
      <div className="flex items-start gap-2">
        <span className="mt-0.5 shrink-0 rounded bg-amber-500/20 px-1.5 py-0.5 text-[10px] font-medium text-amber-400">
          D
        </span>
        <p className="text-sm font-medium text-slate-200">{question.topic || question.text}</p>
      </div>
      {question.context && (
        <p className="ml-7 text-[11px] text-slate-500">{question.context}</p>
      )}
      <div className="ml-7 space-y-1">
        {options.map((opt) => (
          <button
            key={opt.key}
            type="button"
            data-testid={selectors.questionStepper.workshopOption}
            disabled={disabled}
            onClick={() => handleSelect(opt.key)}
            className={cn(
              "w-full rounded-md border px-2.5 py-1.5 text-left transition-colors",
              selected === opt.key
                ? "border-emerald-500/40 bg-emerald-500/10"
                : opt.recommended
                  ? "border-cyan-500/30 bg-cyan-500/[0.03] hover:border-cyan-500/50"
                  : "border-slate-600 bg-slate-800/50 hover:border-slate-500",
              disabled && "opacity-50 cursor-not-allowed",
            )}
          >
            <div className="flex items-baseline gap-1.5">
              <span className={cn(
                "shrink-0 rounded px-1 py-0.5 text-[9px] font-bold",
                selected === opt.key
                  ? "bg-emerald-500/20 text-emerald-400"
                  : "bg-slate-700 text-slate-400",
              )}>
                {opt.key}
              </span>
              <span className="text-xs text-slate-200">{opt.label}</span>
              {opt.recommended && (
                <span className="ml-auto flex items-center gap-0.5 rounded bg-cyan-500/15 px-1 py-0.5 text-[9px] font-medium text-cyan-400">
                  <Star className="h-2.5 w-2.5 fill-current" />
                  Rec
                </span>
              )}
            </div>
            {opt.rationale && (
              <p className="mt-0.5 ml-5 text-[10px] text-slate-500">{opt.rationale}</p>
            )}
          </button>
        ))}
        <button
          type="button"
          disabled={disabled}
          onClick={() => handleSelect(OTHER_KEY)}
          className={cn(
            "w-full rounded-md border px-2.5 py-1.5 text-left transition-colors",
            isOther
              ? "border-emerald-500/40 bg-emerald-500/10"
              : "border-slate-600 bg-slate-800/50 hover:border-slate-500",
            disabled && "opacity-50 cursor-not-allowed",
          )}
        >
          <div className="flex items-baseline gap-1.5">
            <span className={cn(
              "shrink-0 rounded px-1 py-0.5 text-[9px] font-bold",
              isOther ? "bg-emerald-500/20 text-emerald-400" : "bg-slate-700 text-slate-400",
            )}>
              ...
            </span>
            <span className="text-xs text-slate-200">Other</span>
          </div>
        </button>
        {isOther && (
          <textarea
            className="w-full rounded-md border border-slate-600 bg-slate-800 px-2.5 py-1.5 text-xs text-slate-200 placeholder-slate-500 focus:border-slate-500 focus:outline-none"
            placeholder="Describe your alternative..."
            value={freeform}
            onChange={(e) => onUpdate({ selected: OTHER_KEY, freeform: e.target.value })}
            disabled={disabled}
            rows={2}
          />
        )}
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Review question sub-renderer
// ---------------------------------------------------------------------------

interface ReviewQuestionViewProps {
  question: PendingQuestion;
  answer: QuestionAnswer | undefined;
  disabled: boolean;
  onUpdate: (patch: Partial<QuestionAnswer>) => void;
}

function ReviewQuestionView({ question, answer, disabled, onUpdate }: ReviewQuestionViewProps) {
  const status: ReviewStatus = answer?.reviewStatus ?? (question.review_status as ReviewStatus) ?? "unreviewed";
  const comment = answer?.reviewComment ?? question.review_comment ?? "";
  const showComment = status === "flagged";

  const CRITICALITY_COLORS: Record<string, string> = {
    P0: "text-red-400 border-red-500/30 bg-red-500/10",
    P1: "text-orange-400 border-orange-500/30 bg-orange-500/10",
    P2: "text-green-400 border-green-500/30 bg-green-500/10",
  };

  return (
    <div className="space-y-2">
      <div className="flex items-start gap-2">
        <span className="mt-0.5 shrink-0 rounded bg-slate-700 px-1.5 py-0.5 text-[10px] font-medium text-slate-400">
          {question.review_type === "requirement" ? "Req" : "Target"}
        </span>
        <div className="min-w-0">
          <div className="flex items-center gap-2">
            <span className="text-[10px] font-mono text-slate-500">{question.id}</span>
            {question.criticality && (
              <span className={cn(
                "rounded border px-1 py-0.5 text-[9px] font-medium",
                CRITICALITY_COLORS[question.criticality] ?? "text-slate-400 border-slate-600 bg-slate-700/50",
              )}>
                {question.criticality}
              </span>
            )}
          </div>
          <p className="mt-0.5 text-sm font-medium text-slate-200">{question.title}</p>
        </div>
      </div>
      {question.description && (
        <p className="ml-7 text-[11px] text-slate-400">{question.description}</p>
      )}

      {/* Approve / Flag buttons */}
      <div className="ml-7 flex items-center gap-1.5">
        <button
          type="button"
          data-testid={selectors.questionStepper.reviewApprove}
          disabled={disabled}
          onClick={() => onUpdate({ reviewStatus: "approved", reviewComment: comment })}
          className={cn(
            "flex items-center gap-1 rounded-md border px-2 py-1 text-[11px] font-medium transition-colors",
            status === "approved"
              ? "border-emerald-500/40 bg-emerald-500/10 text-emerald-400"
              : "border-slate-600 bg-slate-800/50 text-slate-400 hover:border-emerald-500/40 hover:text-emerald-400",
            disabled && "opacity-50 cursor-not-allowed",
          )}
        >
          <CheckCircle2 className="h-3 w-3" />
          Approve
        </button>
        <button
          type="button"
          data-testid={selectors.questionStepper.reviewFlag}
          disabled={disabled}
          onClick={() => onUpdate({ reviewStatus: "flagged", reviewComment: comment })}
          className={cn(
            "flex items-center gap-1 rounded-md border px-2 py-1 text-[11px] font-medium transition-colors",
            status === "flagged"
              ? "border-amber-500/40 bg-amber-500/10 text-amber-400"
              : "border-slate-600 bg-slate-800/50 text-slate-400 hover:border-amber-500/40 hover:text-amber-400",
            disabled && "opacity-50 cursor-not-allowed",
          )}
        >
          <AlertTriangle className="h-3 w-3" />
          Flag
        </button>
      </div>

      {showComment && (
        <div className="ml-7">
          <textarea
            className="w-full rounded-md border border-slate-600 bg-slate-800 px-2.5 py-1.5 text-xs text-slate-200 placeholder-slate-500 focus:border-slate-500 focus:outline-none"
            placeholder="Comment (optional)..."
            value={comment}
            onChange={(e) => onUpdate({ reviewStatus: "flagged", reviewComment: e.target.value })}
            disabled={disabled}
            rows={2}
          />
        </div>
      )}
    </div>
  );
}
