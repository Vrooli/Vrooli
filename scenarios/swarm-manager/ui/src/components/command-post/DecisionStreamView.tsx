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
import { ChevronLeft, ChevronRight, Loader2, SkipForward, Moon, CheckCircle2, Trash2, Menu } from "lucide-react";
import { cn } from "../../lib";
import { MarkdownRenderer } from "../markdown/MarkdownRenderer";
import { selectors } from "../../consts/selectors";
import { BACKLOG_KIND_ICONS, BACKLOG_KIND_LABELS } from "../../types";
import type { CrossItemQuestion } from "../../lib/command-post-utils";
import { WorkshopQuestionView, ReviewQuestionView } from "../backlog/question-renderers";
import { ClarifyButton } from "../backlog/clarify-button";
import { useDecisionStreamLogic } from "../../hooks/useDecisionStreamLogic";
import { useClarificationStore } from "../../stores/clarification-store";
import { ScenarioNavigatorPopover } from "./ScenarioNavigatorPopover";
import type { DecisionStreamResults } from "../../hooks/useDecisionStreamLogic";
import { WorkshopTransitionStatus } from "../backlog/workshop-transition-status";
import { backlogService } from "../../services/backlog-service";

export type { DecisionStreamResults };

export interface DecisionStreamViewProps {
  questions: CrossItemQuestion[];
  onComplete: (results: DecisionStreamResults) => void;
  /** Legacy keyboard-dismiss callback; the header deliberately has no back control. */
  onBack?: () => void;
  onOpenSidebar?: () => void;
  onSnoozeItem: (key: string) => void;
  /** Navigate to a backlog item's detail page. */
  onOpenItem?: (kind: string, name: string) => void;
  onQueueComplete?: () => void;
  finalActionLabel?: string;
  currentQuestionId?: string | null;
  onCurrentQuestionChange?: (id: string | null) => void;
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function DecisionStreamView({
  questions,
  onComplete,
  onBack,
  onOpenSidebar,
  onSnoozeItem,
  onOpenItem,
  onQueueComplete,
  finalActionLabel,
  currentQuestionId,
  onCurrentQuestionChange,
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
    completionResults,
    current,
    answer,
    total,
    safeIndex,
    savingId,
    saveError,
    deletingId,
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
    onComplete: finishDecisionStream,
  } = useDecisionStreamLogic({ questions, onComplete, onBack: onBack ?? (() => undefined), onSnoozeItem, navigatorOpenRef, toggleNavigator, onQueueComplete, currentQuestionId, onCurrentQuestionChange });

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

  if (phase === "complete" && completionResults) {
    return (
      <div className="flex h-full items-center justify-center overflow-y-auto px-4 py-6">
        <div className="w-full max-w-2xl space-y-4 rounded-lg border border-slate-800 bg-slate-900/60 p-5">
          <div className="flex items-start gap-3">
            <CheckCircle2 className="mt-0.5 h-6 w-6 shrink-0 text-emerald-400" />
            <div>
              <h3 className="text-base font-semibold text-slate-100">Decision stream complete</h3>
              <p className="mt-1 text-sm text-slate-400">
                Saved {completionResults.answeredCount} answer{completionResults.answeredCount === 1 ? "" : "s"}.
                {completionResults.skippedCount > 0 ? ` Skipped ${completionResults.skippedCount}.` : ""}
                {completionResults.snoozedCount > 0 ? ` Snoozed ${completionResults.snoozedCount} item${completionResults.snoozedCount === 1 ? "" : "s"}.` : ""}
              </p>
            </div>
          </div>

          {completionResults.unlockedItems.length > 0 ? (
            <div className="space-y-2">
              {completionResults.unlockedItems.map((item) => (
                <WorkshopTransitionStatus
                  key={`${item.kind}/${item.name}`}
                  autoAdvance={item.autoAdvance ?? { triggered: false, reason: "ready", nextMode: item.action === "finalize" ? "finalize" : "workshop" }}
                  kind={item.kind}
                  name={item.name}
                  title={item.title}
                  onCancelled={() => finishDecisionStream(completionResults)}
                  onExpired={() => finishDecisionStream(completionResults)}
                  onRunNext={() => {
                    void backlogService.research(item.kind, item.name, {
                      mode: "workshop",
                      prompt: "Run the next workshop round for this backlog item.",
                      confirm: true,
                    }).then(() => finishDecisionStream(completionResults));
                  }}
                  onFinalize={() => {
                    void backlogService.research(item.kind, item.name, {
                      mode: "finalize",
                      prompt: "Finalize the latest workshop answers for this backlog item.",
                      confirm: true,
                    }).then(() => finishDecisionStream(completionResults));
                  }}
                />
              ))}
            </div>
          ) : (
            <div className="rounded-lg border border-slate-800 bg-slate-950/50 px-3 py-2.5 text-sm text-slate-400">
              No workshop item reported a next step from the save response.
            </div>
          )}

          <div className="flex flex-wrap items-center justify-end gap-2">
            <button
              type="button"
              onClick={() => finishDecisionStream(completionResults)}
              className="inline-flex min-h-[40px] items-center justify-center rounded border border-slate-700 px-3 py-2 text-sm text-slate-300 hover:bg-slate-800"
            >
              Back to Command Post
            </button>
          </div>
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
      </div>
    );
  }

  if (!current) return null;

  const isSaving = savingId === current.question.id;
  const isFirst = safeIndex === 0;
  const isLast = safeIndex === total - 1;
  const KindIcon = BACKLOG_KIND_ICONS[current.parentKind];

  return (
    <div ref={containerRef} className="flex h-full flex-col" data-testid={selectors.commandPost.decisionStream.container}>
      {/* Header keeps the decision identity visible and navigates to its full detail page. */}
      <div
        className="relative z-[70] flex shrink-0 items-center gap-2 border-b border-slate-700/50 bg-slate-950 px-3"
        data-testid={selectors.commandPost.decisionStream.header}
      >
        {onOpenSidebar && (
          <button
            type="button"
            onClick={onOpenSidebar}
            className="flex min-h-[44px] shrink-0 items-center rounded-lg p-2 text-slate-400 transition-colors hover:bg-slate-800 hover:text-slate-200 active:bg-slate-700"
            aria-label="Open sidebar"
            data-testid="page-sidebar-button"
          >
            <Menu className="h-4 w-4" />
          </button>
        )}
        {/* Kind icon + title (center, truncated) */}
        <div className="flex min-w-0 flex-1 items-center gap-1.5 overflow-hidden">
          <KindIcon className="h-4 w-4 shrink-0 text-slate-500" aria-label={BACKLOG_KIND_LABELS[current.parentKind]} />
          <button type="button" onClick={() => onOpenItem?.(current.parentKind, current.parentName)} className="truncate text-left text-sm font-medium text-cyan-300 hover:text-cyan-200 hover:underline" data-testid={selectors.commandPost.decisionStream.openItemLink}>
            {current.parentTitle}
          </button>
        </div>

        {/* Counter + navigator */}
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
        </div>
      </div>


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
              <MarkdownRenderer content={current.question.context_note} className="prose-sm-slate mt-1 break-words text-xs text-slate-400 [overflow-wrap:anywhere]" />
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
                {isLast ? (finalActionLabel ?? "Done") : "Next"}
                <ChevronRight className="h-4 w-4" />
              </>
            )}
          </button>
        </div>
      </div>
    </div>
  );
}
