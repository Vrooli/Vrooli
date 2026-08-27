/**
 * Shared question renderers for workshop and review questions.
 * Extracted from inline-question-stepper.tsx for reuse in Command Post.
 */
import { CheckCircle2, AlertTriangle, Star } from "lucide-react";
import { cn } from "../../lib";
import { selectors } from "../../consts/selectors";
import { OTHER_KEY, filterAgentOther } from "../../lib/workshop-files";
import { MarkdownRenderer } from "@vrooli/react-component-library/markdown-renderer/0/0.3.2";
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

export interface WorkshopQuestionLike {
  id: string;
  topic?: string | null;
  text?: string | null;
  context?: string | null;
  options?: { key: string; label: string; rationale: string; recommended?: boolean }[];
  selected?: string | null;
  freeform?: string | null;
  notes?: string | null;
}

interface MarkdownBlockProps {
  content?: string | null;
  className?: string;
}

function MarkdownBlock({ content, className }: MarkdownBlockProps) {
  if (!content?.trim()) return null;

  return (
    <MarkdownRenderer content={content} className={cn("prose-sm-slate break-words [overflow-wrap:anywhere]", className)} />
  );
}

interface InlineMarkdownProps {
  content?: string | null;
  className?: string;
}

function InlineMarkdown({ content, className }: InlineMarkdownProps) {
  if (!content?.trim()) return null;

  return (
    <MarkdownRenderer content={content} className={cn("min-w-0 break-words [overflow-wrap:anywhere]", className)} />
  );
}

// ---------------------------------------------------------------------------
// Workshop question sub-renderer
// ---------------------------------------------------------------------------

export interface WorkshopQuestionViewProps {
  question: WorkshopQuestionLike;
  answer: QuestionAnswer | undefined;
  disabled: boolean;
  onUpdate: (patch: Partial<QuestionAnswer>) => void;
  /** Optional action buttons rendered in the header row (e.g. clarify, delete). */
  actions?: React.ReactNode;
}

export function WorkshopQuestionView({ question, answer, disabled, onUpdate, actions }: WorkshopQuestionViewProps) {
  const selected = answer?.selected ?? question.selected ?? "";
  const freeform = answer?.freeform ?? question.freeform ?? "";
  const notes = answer?.notes ?? question.notes ?? "";
  const isOther = selected === OTHER_KEY;
  const isResolved = !!selected.trim();
  const options = filterAgentOther(question.options ?? []);
  const title = question.topic?.trim() || "";
  const body = question.text?.trim() || "";
  const hasSeparateBody = !!body && body !== title;

  const handleSelect = (key: string) => {
    onUpdate({
      selected: key,
      freeform: key === OTHER_KEY ? freeform : undefined,
      notes,
    });
  };

  return (
    <div className="space-y-2">
      <div className="grid grid-cols-[auto_minmax(0,1fr)_auto] items-start gap-x-2 gap-y-1">
        <span className="mt-0.5 shrink-0 rounded bg-amber-500/20 px-1.5 py-0.5 text-[10px] font-medium text-amber-400">
          D
        </span>
        {title ? (
          <InlineMarkdown
            content={title}
            className="text-sm font-medium leading-snug text-slate-200"
          />
        ) : (
          <div />
        )}
        <div className="flex shrink-0 items-start justify-end">{actions}</div>
      </div>
      {(!title && body) || hasSeparateBody ? (
        <MarkdownBlock
          content={body}
          className="text-sm leading-relaxed text-slate-300"
        />
      ) : null}
      <MarkdownBlock
        content={question.context}
        className="text-[11px] leading-relaxed text-slate-500"
      />
      <div className="space-y-1">
        {options.map((opt) => (
          <button
            key={opt.key}
            type="button"
            data-testid={selectors.questionStepper.workshopOption}
            disabled={disabled}
            onClick={() => handleSelect(opt.key)}
            className={cn(
              "w-full rounded-md border px-2 py-1.5 text-left transition-colors",
              selected === opt.key
                ? "border-emerald-500/40 bg-emerald-500/10"
                : opt.recommended
                  ? "border-cyan-500/30 bg-cyan-500/[0.03] hover:border-cyan-500/50"
                  : "border-slate-600 bg-slate-800/50 hover:border-slate-500",
              disabled && "opacity-50 cursor-not-allowed",
            )}
          >
            <div className="flex min-w-0 items-start gap-1.5">
              <span className={cn(
                "mt-0.5 shrink-0 rounded px-1 py-0.5 text-[9px] font-bold",
                selected === opt.key
                  ? "bg-emerald-500/20 text-emerald-400"
                  : "bg-slate-700 text-slate-400",
              )}>
                {opt.key}
              </span>
              <InlineMarkdown
                content={opt.label}
                className="flex-1 text-xs text-slate-200"
              />
              {opt.recommended && (
                <span className="ml-auto mt-0.5 flex shrink-0 items-center gap-0.5 rounded bg-cyan-500/15 px-1 py-0.5 text-[9px] font-medium text-cyan-400">
                  <Star className="h-2.5 w-2.5 fill-current" />
                  Rec
                </span>
              )}
            </div>
            <MarkdownBlock
              content={opt.rationale}
              className="mt-1 text-[10px] leading-snug text-slate-500"
            />
          </button>
        ))}
        <button
          type="button"
          disabled={disabled}
          onClick={() => handleSelect(OTHER_KEY)}
          className={cn(
            "w-full rounded-md border px-2 py-1.5 text-left transition-colors",
            isOther
              ? "border-emerald-500/40 bg-emerald-500/10"
              : "border-slate-600 bg-slate-800/50 hover:border-slate-500",
            disabled && "opacity-50 cursor-not-allowed",
          )}
        >
          <div className="flex min-w-0 items-start gap-1.5">
            <span className={cn(
              "mt-0.5 shrink-0 rounded px-1 py-0.5 text-[9px] font-bold",
              isOther ? "bg-emerald-500/20 text-emerald-400" : "bg-slate-700 text-slate-400",
            )}>
              ...
            </span>
            <span className="min-w-0 flex-1 break-words text-xs text-slate-200 [overflow-wrap:anywhere]">Other</span>
          </div>
        </button>
        {isOther && (
          <textarea
            className="w-full rounded-md border border-slate-600 bg-slate-800 px-2 py-1.5 text-xs text-slate-200 placeholder-slate-500 focus:border-slate-500 focus:outline-none"
            placeholder="Describe your alternative..."
            value={freeform}
            onChange={(e) => onUpdate({ selected: OTHER_KEY, freeform: e.target.value })}
            disabled={disabled}
            rows={2}
          />
        )}
      </div>

      {/* Notes — visible once any option is selected */}
      {isResolved && (
        <textarea
          className="w-full rounded-md border border-slate-600 bg-slate-800 px-2 py-1.5 text-xs text-slate-200 placeholder-slate-500 focus:border-slate-500 focus:outline-none"
          placeholder="Notes (optional)..."
          value={notes}
          onChange={(e) => onUpdate({ selected, freeform: isOther ? freeform : undefined, notes: e.target.value })}
          disabled={disabled}
          rows={2}
        />
      )}
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
  /** Optional action buttons rendered in the header row. */
  actions?: React.ReactNode;
}

export function ReviewQuestionView({ question, answer, disabled, onUpdate, actions }: ReviewQuestionViewProps) {
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
      <div className="grid grid-cols-[auto_minmax(0,1fr)_auto] items-start gap-x-2 gap-y-1">
        <span className="mt-0.5 shrink-0 rounded bg-slate-700 px-1.5 py-0.5 text-[10px] font-medium text-slate-400">
          {question.review_type === "requirement" ? "Req" : "Target"}
        </span>
        <div className="min-w-0 flex-1">
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
          <InlineMarkdown
            content={question.title}
            className="mt-0.5 text-sm font-medium leading-snug text-slate-200"
          />
        </div>
        <div className="flex shrink-0 items-start justify-end">{actions}</div>
      </div>
      <MarkdownBlock
        content={question.description}
        className="text-[11px] leading-relaxed text-slate-400"
      />

      {/* Approve / Flag buttons */}
      <div className="flex flex-wrap items-center gap-1.5">
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
        <div>
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
