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
import { useState, useCallback, useRef } from "react";
import { ChevronDown, ChevronLeft, ChevronRight, ExternalLink, Loader2, SkipForward, ArrowLeft, Moon, CheckCircle2, Info, Trash2 } from "lucide-react";
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
import { ClarifyButton } from "../backlog/clarify-button";
import { ScenarioBadge } from "../backlog/scenario-badge";
import { TagList } from "../ui/tag-list";
import { useDecisionStreamLogic } from "../../hooks/useDecisionStreamLogic";
import { useClarificationStore } from "../../stores/clarification-store";
import { ScenarioNavigatorPopover } from "./ScenarioNavigatorPopover";
import type { DecisionStreamResults } from "../../hooks/useDecisionStreamLogic";

export type { DecisionStreamResults };

export interface DecisionStreamViewProps {
  questions: CrossItemQuestion[];
  onComplete: (results: DecisionStreamResults) => void;
  onBack: () => void;
  onSnoozeItem: (key: string) => void;
  /** Navigate to a backlog item's detail page. */
  onOpenItem?: (kind: string, name: string) => void;
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function DecisionStreamView({
  questions,
  onComplete,
  onBack,
  onSnoozeItem,
  onOpenItem,
}: DecisionStreamViewProps) {
  // Navigator state
  const [navigatorOpen, setNavigatorOpen] = useState(false);
  const navigatorOpenRef = useRef(false);
  const toggleNavigator = useCallback(() => {
    setNavigatorOpen((prev) => {
      const next = !prev;
      navigatorOpenRef.current = next;
      return next;
    });
  }, []);

  const {
    phase,
    current,
    answer,
    parentItem,
    total,
    safeIndex,
    savingId,
    saveError,
    deletingId,
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
    deleteQuestion,
    parentGroups,
    jumpToParent,
    snoozeSpecificParent,
    localAnswers,
    skippedIds,
  } = useDecisionStreamLogic({ questions, onComplete, onBack, onSnoozeItem, navigatorOpenRef, toggleNavigator });

  // Clarification
  const clarificationStore = useClarificationStore();
  const isClarifyActive = current
    ? clarificationStore.isOpen && clarificationStore.target?.itemId === current.question.id
    : false;

  const handleClarifyClick = useCallback(() => {
    if (!current) return;
    const q = current.question;
    if (isClarifyActive) {
      clarificationStore.close();
    } else if (q.source === "workshop" && q.round_number != null) {
      clarificationStore.open({
        backlogKind: current.parentKind,
        backlogName: current.parentName,
        roundNumber: q.round_number,
        itemId: q.id,
        itemTopic: q.topic || q.text || "",
        clarificationId: q.clarification_id,
      });
    }
  }, [current, isClarifyActive, clarificationStore]);

  // Delete confirmation
  const [confirmDeleteId, setConfirmDeleteId] = useState<string | null>(null);

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

        {/* Counter + navigator + context toggle */}
        <div className="relative flex shrink-0 items-center gap-1">
          <button
            type="button"
            onClick={toggleNavigator}
            className="rounded px-1.5 py-0.5 text-xs tabular-nums text-slate-500 transition-colors hover:bg-slate-800 hover:text-slate-300"
            title="Scenario navigator (G)"
            data-testid={selectors.commandPost.decisionStream.navigatorButton}
          >
            <span data-testid={selectors.commandPost.decisionStream.counter}>
              {safeIndex + 1}/{total}
            </span>
          </button>
          <ScenarioNavigatorPopover
            isOpen={navigatorOpen}
            onClose={() => {
              setNavigatorOpen(false);
              navigatorOpenRef.current = false;
            }}
            parentGroups={parentGroups}
            currentParentKey={current ? `${current.parentKind}/${current.parentName}` : ""}
            localAnswers={localAnswers}
            skippedIds={skippedIds}
            onJumpTo={jumpToParent}
            onSnoozeParent={snoozeSpecificParent}
          />
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

                {/* Scenario */}
                {parentItem.acceptanceAllow && parentItem.acceptanceAllow.length > 0 && (
                  <div className="flex items-center gap-1.5">
                    <Info className="h-3 w-3 shrink-0 text-slate-500" />
                    <span className="text-[10px] text-slate-500">Scenario:</span>
                    <ScenarioBadge acceptanceAllow={parentItem.acceptanceAllow} />
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

            {/* Slug + open link */}
            <div className="flex items-center justify-between">
              <p className="text-[10px] font-mono text-slate-600">{current.parentKind}/{current.parentName}</p>
              {onOpenItem && (
                <button
                  type="button"
                  onClick={() => onOpenItem(current.parentKind, current.parentName)}
                  className="flex items-center gap-1 rounded px-1.5 py-0.5 text-[10px] font-medium text-cyan-400 transition-colors hover:bg-slate-800 hover:text-cyan-300"
                  data-testid={selectors.commandPost.decisionStream.openItemLink}
                >
                  <ExternalLink className="h-3 w-3" />
                  Open item
                </button>
              )}
            </div>
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
              actions={
                <div className="flex shrink-0 items-center gap-1">
                  {current.question.round_number != null && (
                    <ClarifyButton
                      disabled={isSaving || deletingId === current.question.id}
                      isActive={isClarifyActive}
                      hasClarification={!!current.question.clarification_id}
                      onClick={handleClarifyClick}
                    />
                  )}
                  {confirmDeleteId === current.question.id ? (
                    <span className="flex items-center gap-1 shrink-0">
                      <button
                        type="button"
                        onClick={() => {
                          setConfirmDeleteId(null);
                          void deleteQuestion(current);
                        }}
                        disabled={deletingId === current.question.id}
                        className="rounded px-1.5 py-0.5 text-[10px] font-medium text-red-400 bg-red-500/10 hover:bg-red-500/20 transition-colors disabled:opacity-50"
                        data-testid={selectors.commandPost.decisionStream.deleteConfirm}
                      >
                        {deletingId === current.question.id ? "Deleting..." : "Delete"}
                      </button>
                      <button
                        type="button"
                        onClick={() => setConfirmDeleteId(null)}
                        className="rounded px-1.5 py-0.5 text-[10px] font-medium text-slate-400 hover:bg-slate-700 transition-colors"
                      >
                        Cancel
                      </button>
                    </span>
                  ) : (
                    <button
                      type="button"
                      onClick={() => setConfirmDeleteId(current.question.id)}
                      disabled={isSaving || deletingId === current.question.id}
                      className="shrink-0 rounded p-1 text-slate-500 hover:text-red-400 hover:bg-red-500/10 transition-colors disabled:opacity-50"
                      title="Delete question"
                      data-testid={selectors.commandPost.decisionStream.deleteButton}
                    >
                      <Trash2 className="h-3.5 w-3.5" />
                    </button>
                  )}
                </div>
              }
            />
          ) : (
            <ReviewQuestionView
              question={current.question}
              answer={answer}
              disabled={isSaving}
              onUpdate={(patch) => updateAnswer(current.question.id, patch)}
            />
          )}

          {/* Context note from clarification */}
          {current.question.context_note && (
            <div className="mt-2 rounded border border-cyan-500/15 bg-cyan-500/5 px-2 py-1">
              <span className="text-[9px] font-medium text-cyan-400">Clarification note</span>
              <p className="text-xs text-slate-400">{current.question.context_note}</p>
            </div>
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
