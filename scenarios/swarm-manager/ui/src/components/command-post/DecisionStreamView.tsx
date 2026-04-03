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
import { ChevronDown, ChevronLeft, ChevronRight, Loader2, SkipForward, ArrowLeft, Moon, CheckCircle2, Info } from "lucide-react";
import { cn } from "../../lib";
import { selectors } from "../../consts/selectors";
import {
  BACKLOG_KIND_ICONS,
  BACKLOG_KIND_LABELS,
  BACKLOG_STATUS_CHIP_COLORS,
  formatBacklogStatus,
} from "../../types";
import type { CrossItemQuestion } from "../../lib/command-post-utils";
import { WorkshopQuestionView, ReviewQuestionView } from "../backlog/question-renderers";
import { TagList } from "../ui/tag-list";
import { useDecisionStreamLogic } from "../../hooks/useDecisionStreamLogic";
import type { DecisionStreamResults } from "../../hooks/useDecisionStreamLogic";

export type { DecisionStreamResults };

export interface DecisionStreamViewProps {
  questions: CrossItemQuestion[];
  onComplete: (results: DecisionStreamResults) => void;
  onBack: () => void;
  onSnoozeItem: (key: string) => void;
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
  const {
    phase,
    current,
    answer,
    parentItem,
    total,
    safeIndex,
    savingId,
    saveError,
    contextExpanded,
    setContextExpanded,
    descExpanded,
    setDescExpanded,
    containerRef,
    updateAnswer,
    advance,
    goBack,
    skip,
    snoozeParent,
  } = useDecisionStreamLogic({ questions, onComplete, onBack, onSnoozeItem });

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
