/**
 * Shared question renderers for workshop and review questions.
 * Extracted from inline-question-stepper.tsx for reuse in Command Post.
 */
import { CheckCircle2, AlertTriangle, Star } from "lucide-react";
import { cn } from "../../lib";
import { selectors } from "../../consts/selectors";
import { OTHER_KEY, filterAgentOther } from "../../lib/workshop-files";
import type { PendingQuestion, ReviewStatus } from "../../types";

/** Local answer state for a single question. */
export interface QuestionAnswer {
  // Workshop fields
  selected?: string;
  freeform?: string;
  notes?: string;
  // Review fields
  reviewStatus?: ReviewStatus;
  reviewComment?: string;
}

// ---------------------------------------------------------------------------
// Workshop question sub-renderer
// ---------------------------------------------------------------------------

export interface WorkshopQuestionViewProps {
  question: PendingQuestion;
  answer: QuestionAnswer | undefined;
  disabled: boolean;
  onUpdate: (patch: Partial<QuestionAnswer>) => void;
}

export function WorkshopQuestionView({ question, answer, disabled, onUpdate }: WorkshopQuestionViewProps) {
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

export interface ReviewQuestionViewProps {
  question: PendingQuestion;
  answer: QuestionAnswer | undefined;
  disabled: boolean;
  onUpdate: (patch: Partial<QuestionAnswer>) => void;
}

export function ReviewQuestionView({ question, answer, disabled, onUpdate }: ReviewQuestionViewProps) {
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
