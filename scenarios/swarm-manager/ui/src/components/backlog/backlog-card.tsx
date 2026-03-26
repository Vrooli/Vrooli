/**
 * BacklogCard — shared card interior for backlog items.
 *
 * Renders everything inside `<ResponsiveListItem>`. The wrapper/Link stays
 * in BacklogPage so routing context is controlled by the page.
 *
 * Action visibility/disabled logic comes from the `itemActions` prop
 * (computed by `getItemActions()` in the parent) — this component only renders.
 */

import { Archive, ArrowRight, CheckSquare, Loader2, Lock, MessageSquare, MessageSquareText, Play } from "lucide-react";
import { Button } from "../ui/button";
import { TagList } from "../ui/tag-list";
import { formatRelativeTime } from "../../lib";
import { BACKLOG_STATUS_COLORS, formatBacklogStatus, type BacklogKind } from "../../types";
import type { BacklogItem, PendingQuestion } from "../../types";
import type { ItemActions } from "../../lib/backlog-queue-utils";
import type { AttentionReason } from "../../lib/feed";
import type { ReadinessIndicatorData } from "../../lib/maturity";
import type { StepperCompletionResult } from "./inline-question-stepper";
import { InlineQuestionStepper } from "./inline-question-stepper";
import { PendingDecisionBadge } from "./pending-decision-badge";
import { InitiativeBadge } from "./initiative-badge";
import { DependencyIndicator } from "./dependency-indicator";
import { ScenarioBadge } from "./scenario-badge";
import { AgentRunningBadge } from "./agent-running-badge";
import { ReadinessBar } from "./readiness-bar";
import { displayLimitsConfig } from "../../config";

export interface BacklogCardProps {
  item: BacklogItem;
  allItems: BacklogItem[];
  readinessData?: ReadinessIndicatorData;
  itemActions: ItemActions;
  // Attention / stepper
  attentionReasons: AttentionReason[];
  pendingQuestions?: PendingQuestion[];
  isStepperCompleted: boolean;
  transitionResult?: StepperCompletionResult;
  onStepperCompleted: (result: StepperCompletionResult) => void;
  // Batch mode
  batchMode: boolean;
  isSelected: boolean;
  onToggleSelection: () => void;
  // Actions
  onRun: () => void;
  onArchive: () => void;
  onFollowUp: () => void;
  onWorkshop: () => void;
  archivePending: boolean;
  workshopPending: boolean;
}

export function BacklogCard({
  item,
  allItems,
  readinessData,
  itemActions,
  attentionReasons,
  pendingQuestions,
  isStepperCompleted,
  transitionResult,
  onStepperCompleted,
  batchMode,
  isSelected,
  onToggleSelection,
  onRun,
  onArchive,
  onFollowUp,
  onWorkshop,
  archivePending,
  workshopPending,
}: BacklogCardProps) {
  const hasActiveStepper = itemActions.showDecisionStepper && (pendingQuestions?.length ?? 0) > 0 && !isStepperCompleted;
  const showBatchCheckbox = batchMode && (itemActions.canRun || itemActions.runDisabled || itemActions.canWorkshop || itemActions.workshopDisabled) && !itemActions.blocked;

  return (
    <>
      {/* Header: status + priority */}
      <div className="flex items-start justify-between">
        <div className="flex items-center gap-2">
          {showBatchCheckbox ? (
            <input
              type="checkbox"
              aria-label={`Select backlog item ${item.title}`}
              checked={isSelected}
              onClick={(event) => event.stopPropagation()}
              onChange={(event) => {
                event.stopPropagation();
                onToggleSelection();
              }}
            />
          ) : null}
          <span
            className={`inline-block h-2 w-2 rounded-full ${BACKLOG_STATUS_COLORS[item.status] ?? "bg-slate-500"}`}
          />
          <span className="text-xs uppercase tracking-wider text-slate-400">
            {formatBacklogStatus(item.status)}
          </span>
          <ScenarioBadge acceptanceAllow={item.acceptanceAllow} />
          <AgentRunningBadge backlogKind={item.kind as BacklogKind} backlogName={item.name} />
        </div>
        <span className="rounded-full bg-slate-700 px-2 py-0.5 text-xs text-slate-300">
          P{item.priority}
        </span>
      </div>

      {/* Title + description */}
      <h3 className="mt-3 font-medium text-slate-100">{item.title}</h3>
      <p className="mt-1 line-clamp-2 text-sm text-slate-400">{item.description}</p>

      {/* Body: stepper, transition, or normal content */}
      {hasActiveStepper ? (
        <InlineQuestionStepper
          questions={pendingQuestions as PendingQuestion[]}
          backlogKind={item.kind as BacklogKind}
          backlogName={item.name}
          onAllAnswered={onStepperCompleted}
        />
      ) : transitionResult ? (
        <div className="mt-3 flex items-center gap-2 rounded-lg border border-cyan-500/20 bg-cyan-500/[0.03] px-3 py-2.5 text-sm text-cyan-300">
          {transitionResult.autoAdvance?.triggered ? (
            <>
              <Loader2 className="h-4 w-4 animate-spin shrink-0" />
              Generating next workshop round...
            </>
          ) : transitionResult.autoAdvance?.reason === "ready" ? (
            <>
              <CheckSquare className="h-4 w-4 shrink-0 text-emerald-400" />
              <span className="text-emerald-300">All decisions answered — item is ready!</span>
            </>
          ) : (
            <>
              <CheckSquare className="h-4 w-4 shrink-0" />
              All decisions answered
            </>
          )}
        </div>
      ) : (
        <>
          {attentionReasons.length > 0 && (
            <div className="mt-2">
              <PendingDecisionBadge reasons={attentionReasons} />
            </div>
          )}
          {(item.initiative || (item.dependsOn && item.dependsOn.length > 0)) && (
            <div className="mt-2 flex flex-wrap gap-1">
              <InitiativeBadge initiative={item.initiative} />
              <DependencyIndicator dependsOn={item.dependsOn} allItems={allItems} />
            </div>
          )}
          <TagList
            tags={item.tags}
            maxTags={displayLimitsConfig.backlogCardMaxTags}
            className="mt-3"
          />
          {readinessData && item.kind === "idea" ? (
            <ReadinessBar data={readinessData} className="mt-3" />
          ) : null}
          <div className="mt-4 flex items-center justify-between text-xs text-slate-400">
            <span title={new Date(item.updated).toLocaleString()}>{formatRelativeTime(item.updated)}</span>
            <ArrowRight className="h-4 w-4 opacity-0 transition group-hover:opacity-100" />
          </div>

          {/* Locked state */}
          {itemActions.locked && (
            <div className="mt-3 flex items-center gap-1.5 text-xs text-slate-500">
              <Lock className="h-3 w-3" />
              {formatBacklogStatus(item.status)} — check Execution for progress.
            </div>
          )}

          {/* Blocked state */}
          {itemActions.blocked && !itemActions.locked && (
            <div className="mt-3 space-y-2" onClick={(event) => event.preventDefault()}>
              {(itemActions.workshopDisabled) && (
                <Button
                  size="sm"
                  variant="outline"
                  disabled
                  onClick={(event) => { event.preventDefault(); event.stopPropagation(); }}
                >
                  <MessageSquareText className="mr-1 h-3 w-3" />
                  Workshop
                </Button>
              )}
              {(itemActions.runDisabled && !itemActions.workshopDisabled) && (
                <Button
                  size="sm"
                  disabled
                  onClick={(event) => { event.preventDefault(); event.stopPropagation(); }}
                >
                  <Play className="mr-1 h-3 w-3" />
                  Run
                </Button>
              )}
              <p className="text-xs text-slate-500">
                Blocked by dependencies
              </p>
            </div>
          )}

          {/* Workshop button */}
          {(itemActions.canWorkshop || itemActions.workshopDisabled) && !itemActions.blocked && (
            <div className="mt-3" onClick={(event) => event.preventDefault()}>
              <Button
                size="sm"
                variant="outline"
                disabled={itemActions.workshopDisabled || workshopPending}
                onClick={(event) => {
                  event.preventDefault();
                  event.stopPropagation();
                  onWorkshop();
                }}
              >
                <MessageSquareText className="mr-1 h-3 w-3" />
                {itemActions.agentRunning ? "Agent running..." : workshopPending ? "Starting..." : "Workshop"}
              </Button>
            </div>
          )}

          {/* Run button */}
          {(itemActions.canRun || itemActions.runDisabled) && !itemActions.blocked && (
            <div className="mt-3" onClick={(event) => event.preventDefault()}>
              <Button
                size="sm"
                disabled={itemActions.runDisabled}
                onClick={(event) => {
                  event.preventDefault();
                  event.stopPropagation();
                  onRun();
                }}
              >
                <Play className="mr-1 h-3 w-3" />
                {itemActions.agentRunning ? "Agent running..." : "Run"}
              </Button>
            </div>
          )}

          {/* Terminal actions: Follow Up + Archive in a single row */}
          {(itemActions.canFollowUp || itemActions.canArchive) && (
            <div className="mt-3 flex items-center gap-2" onClick={(event) => event.preventDefault()}>
              {itemActions.canFollowUp && (
                <Button
                  size="sm"
                  variant="outline"
                  onClick={(event) => {
                    event.preventDefault();
                    event.stopPropagation();
                    onFollowUp();
                  }}
                >
                  <MessageSquare className="mr-1 h-3 w-3" />
                  Follow Up
                </Button>
              )}
              {itemActions.canArchive && (
                <Button
                  size="sm"
                  variant="outline"
                  disabled={archivePending}
                  onClick={(event) => {
                    event.preventDefault();
                    event.stopPropagation();
                    onArchive();
                  }}
                >
                  <Archive className="mr-1 h-3 w-3" />
                  {archivePending ? "Archiving..." : "Archive"}
                </Button>
              )}
            </div>
          )}

          {/* Not-queueable reason (non-locked, non-terminal items that can't run) */}
          {itemActions.notQueueableReason && !itemActions.locked && !itemActions.terminal && !itemActions.blocked && (
            <p className="mt-3 text-xs text-slate-500">
              {itemActions.notQueueableReason}
            </p>
          )}
        </>
      )}
    </>
  );
}
