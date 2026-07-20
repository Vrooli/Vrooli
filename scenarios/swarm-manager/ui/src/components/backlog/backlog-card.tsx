/**
 * BacklogCard — shared card interior for backlog items.
 *
 * Renders everything inside `<ResponsiveListItem>`. The wrapper/Link stays
 * in the parent component so routing context is controlled by the caller.
 *
 * Action visibility/disabled logic comes from the `itemActions` prop
 * (computed by `getItemActions()` in the parent) — this component only renders.
 */

import { memo } from "react";
import { Archive, ArrowRight, CheckSquare, Clock, Loader2, Lock, MessageSquare, MessageSquareText, Play, Sparkles } from "lucide-react";
import { Button } from "../ui/button";
import { TagList } from "../ui/tag-list";
import { formatRelativeTime } from "../../lib";
import { BACKLOG_KIND_ICONS, BACKLOG_STATUS_COLORS, formatBacklogStatus } from "../../types";
import type { BacklogItem, BacklogStatus, PendingQuestion } from "../../types";
import { StatusChipPopover } from "./status-chip-popover";
import type { ItemActions } from "../../lib/backlog-queue-utils";
import type { AttentionReason } from "../../lib/attention";
import type { ReadinessIndicatorData } from "../../lib/maturity";
import type { StepperCompletionResult } from "./inline-question-stepper";
import { InlineQuestionStepper } from "./inline-question-stepper";
import { PendingDecisionBadge } from "./pending-decision-badge";
import { InitiativeBadge } from "./initiative-badge";
import { DependencyIndicator } from "./dependency-indicator";
import { ScenarioBadge } from "./scenario-badge";
import { AgentRunningBadge } from "./agent-running-badge";
import { CircuitBrokenBadge } from "./circuit-broken-badge";
import { NoteIndicator } from "../ui/note-indicator";
import { ReadinessBar } from "./readiness-bar";
import { AutoAdvanceCountdown } from "./auto-advance-countdown";
import { SnoozePopover } from "../command-post/SnoozePopover";
import { snoozeKeyForBacklog } from "../../lib/snooze-utils";
import { displayLimitsConfig } from "../../config";
import { PickModeRow } from "../session/context/selectable-card";
import type { CardSelection } from "../session/context/selectable";

// All-off ItemActions used when the card renders without a sidebar action
// context (i.e. the SessionContextPicker pick-mode path, which suppresses
// every action row). Keeps the sidebar prop contract non-breaking while
// letting the picker render a BacklogCard with only `item` + `selection`.
const NO_ITEM_ACTIONS: ItemActions = {
  locked: false,
  terminal: false,
  blocked: false,
  blockingDepKeys: [],
  primaryCta: null,
  canRun: false,
  runDisabled: false,
  canWorkshop: false,
  workshopDisabled: false,
  canFinalize: false,
  finalizeDisabled: false,
  canFollowUp: false,
  canRetry: false,
  canArchive: false,
  showDecisionStepper: false,
  agentRunning: false,
  notQueueableReason: null,
  disabledReason: null,
};

export interface BacklogCardProps {
  item: BacklogItem;
  readinessData?: ReadinessIndicatorData;
  /**
   * Picker pick-mode selection contract. When `selection.selectionMode` is
   * true the card renders a compact display-only summary (no action rows,
   * no stepper) wrapped in PickModeRow. All the sidebar action props below
   * are ignored in that mode, so they are optional for picker callers.
   */
  selection?: CardSelection;
  itemActions?: ItemActions;
  // Attention / stepper
  attentionReasons?: AttentionReason[];
  pendingQuestions?: PendingQuestion[];
  isStepperCompleted?: boolean;
  transitionResult?: StepperCompletionResult;
  onStepperCompleted?: (result: StepperCompletionResult) => void;
  // Batch mode
  batchMode?: boolean;
  isSelected?: boolean;
  onToggleSelection?: () => void;
  // Actions
  onRun?: () => void;
  onArchive?: () => void;
  onFollowUp?: () => void;
  onFinalize?: () => void;
  onWorkshop?: () => void;
  onAcceptSuggestion?: () => void;
  onDismissSuggestion?: () => void;
  archivePending?: boolean;
  dismissPending?: boolean;
  finalizePending?: boolean;
  workshopPending?: boolean;
  workshopLabel?: string;
  /** Human-readable label shown when an agent is running (e.g. "Running workshop…"). */
  runningLabel?: string;
  /** Callback for inline status changes via the status chip popover. */
  onStatusChange?: (newStatus: BacklogStatus) => void;
  /** Whether a status change is in flight. */
  statusChangePending?: boolean;
  /** When true, render a snooze popover in the action area. */
  showSnooze?: boolean;
  /** Callback when the user snoozes this item. */
  onSnooze?: (key: string, expiresAt: number) => void;
}

function BacklogCardImpl({
  item,
  readinessData,
  selection,
  itemActions = NO_ITEM_ACTIONS,
  attentionReasons = [],
  pendingQuestions,
  isStepperCompleted = false,
  transitionResult,
  onStepperCompleted = () => {},
  batchMode = false,
  isSelected = false,
  onToggleSelection = () => {},
  onRun = () => {},
  onArchive = () => {},
  onFollowUp = () => {},
  onFinalize = () => {},
  onWorkshop = () => {},
  onAcceptSuggestion = () => {},
  onDismissSuggestion = () => {},
  archivePending = false,
  dismissPending = false,
  finalizePending = false,
  workshopPending = false,
  workshopLabel = "Workshop",
  runningLabel = "Agent running…",
  onStatusChange,
  statusChangePending,
  showSnooze,
  onSnooze,
}: BacklogCardProps) {
  // Picker pick-mode: a compact, display-only summary. No action rows, no
  // stepper, no status popover — just the identifying header + title/desc.
  if (selection?.selectionMode) {
    const PickKindIcon = BACKLOG_KIND_ICONS[item.kind];
    return (
      <PickModeRow selection={selection}>
        {item.archivedAt != null && (
          <div className="mb-2 flex items-center gap-1.5 rounded border border-amber-500/20 bg-amber-500/5 px-2 py-1 text-[11px] text-amber-400/80">
            <Archive className="h-3 w-3 shrink-0" />
            Archived
          </div>
        )}
        <div className="flex items-start justify-between gap-2">
          <div className="flex min-w-0 items-center gap-2">
            {PickKindIcon && <PickKindIcon className="h-3.5 w-3.5 shrink-0 text-slate-400" />}
            <span
              className={`inline-block h-2 w-2 shrink-0 rounded-full ${BACKLOG_STATUS_COLORS[item.status] ?? "bg-slate-500"}`}
            />
            <span className="truncate text-[11px] uppercase tracking-wider text-slate-400">
              {formatBacklogStatus(item.status)}
            </span>
			{item.stale && <span className="rounded bg-amber-500/15 px-1.5 py-0.5 text-[10px] font-medium text-amber-300">Stale</span>}
          </div>
          <span className="shrink-0 rounded-full bg-slate-700 px-2 py-0.5 text-[10px] text-slate-300">
            P{item.priority}
          </span>
        </div>
        <p className="mt-1.5 line-clamp-2 text-[13px] font-medium leading-snug text-slate-100">{item.title}</p>
        {item.description && (
          <p className="mt-1 line-clamp-2 text-[11px] text-slate-400">{item.description}</p>
        )}
      </PickModeRow>
    );
  }

  const hasActiveStepper = itemActions.showDecisionStepper && (pendingQuestions?.length ?? 0) > 0 && !isStepperCompleted;
  const showBatchCheckbox = batchMode;
  const deliverableLabel = item.kind === "research" ? "conclusion" : "plan";
  const KindIcon = BACKLOG_KIND_ICONS[item.kind];
  const hasPrimaryActionRow = (
    (itemActions.canFinalize || itemActions.finalizeDisabled || itemActions.canRun || itemActions.runDisabled || itemActions.canWorkshop || itemActions.workshopDisabled)
    && !itemActions.blocked
  );
  const snoozeItemKey = snoozeKeyForBacklog(item.kind, item.name);
  const actionRowClass = "mt-3 flex flex-nowrap items-center gap-2 overflow-x-auto pb-1";

  return (
    <>
      {/* Archived banner */}
      {item.archivedAt != null && (
        <div className="mb-2 flex items-center gap-1.5 rounded border border-amber-500/20 bg-amber-500/5 px-2 py-1 text-[11px] text-amber-400/80">
          <Archive className="h-3 w-3 shrink-0" />
          Archived
        </div>
      )}

      {/* Header: status + priority */}
      <div className="flex items-start justify-between">
        <div className="flex items-center gap-2">
          {showBatchCheckbox ? (
            <input
              type="checkbox"
              aria-label={`Select backlog item ${item.title}`}
              checked={isSelected}
              aria-checked={isSelected}
              onClick={(event) => event.stopPropagation()}
              onChange={(event) => {
                event.stopPropagation();
                onToggleSelection();
              }}
            />
          ) : null}
          {KindIcon && <KindIcon className="h-3.5 w-3.5 text-slate-400" />}
          {onStatusChange && !itemActions.locked ? (
            <StatusChipPopover
              currentStatus={item.status}
              onStatusChange={onStatusChange}
              pending={statusChangePending}
            />
          ) : (
            <>
              <span
                className={`inline-block h-2 w-2 rounded-full ${BACKLOG_STATUS_COLORS[item.status] ?? "bg-slate-500"}`}
              />
              <span className="text-xs uppercase tracking-wider text-slate-400">
                {formatBacklogStatus(item.status)}
              </span>
            </>
          )}
          <ScenarioBadge acceptanceAllow={item.acceptanceAllow} />
          <AgentRunningBadge backlogKind={item.kind} backlogName={item.name} />
          <CircuitBrokenBadge backlogKind={item.kind} backlogName={item.name} />
          <NoteIndicator note={item.note} />
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
          backlogKind={item.kind}
          backlogName={item.name}
          onAllAnswered={onStepperCompleted}
        />
      ) : transitionResult ? (
        <div className="mt-3">
          {transitionResult.autoAdvance?.pending && transitionResult.autoAdvance?.advanceAt ? (
            <AutoAdvanceCountdown
              advanceAt={transitionResult.autoAdvance.advanceAt}
              delaySeconds={transitionResult.autoAdvance.delaySeconds ?? 10}
              nextMode={(transitionResult.autoAdvance.nextMode ?? "workshop")}
              kind={item.kind}
              name={item.name}
              onCancelled={onStepperCompleted.bind(null, {})}
              onExpired={() => {/* timer expired — server ticker will spawn; UI shows spinner */}}
            />
          ) : (
          <div className="flex items-center gap-2 rounded-lg border border-cyan-500/20 bg-cyan-500/[0.03] px-3 py-2.5 text-sm text-cyan-300">
          {transitionResult.autoAdvance?.triggered && transitionResult.autoAdvance?.nextMode === "finalize" ? (
            <>
              <Loader2 className="h-4 w-4 animate-spin shrink-0" />
              {`Finalizing ${deliverableLabel}...`}
            </>
          ) : transitionResult.autoAdvance?.triggered ? (
            <>
              <Loader2 className="h-4 w-4 animate-spin shrink-0" />
              Generating next workshop round...
            </>
          ) : transitionResult.autoAdvance?.nextMode === "finalize" ? (
            <>
              <CheckSquare className="h-4 w-4 shrink-0 text-emerald-400" />
              <span className="text-emerald-300">{`All decisions answered — ${deliverableLabel} ready to finalize.`}</span>
            </>
          ) : transitionResult.autoAdvance?.nextMode === "workshop" ? (
            <>
              <CheckSquare className="h-4 w-4 shrink-0" />
              All decisions answered — run the next workshop round to continue.
            </>
          ) : (
            <>
              <CheckSquare className="h-4 w-4 shrink-0" />
              All decisions answered
            </>
          )}
          </div>
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
              <DependencyIndicator dependsOn={item.dependsOn} />
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

          {item.status === "suggested" && !item.archivedAt && (
            <div className={actionRowClass} data-testid="backlog-card-actions" onClick={(event) => event.preventDefault()}>
              <Button
                size="sm"
                variant="default"
                disabled={statusChangePending}
                onClick={(event) => {
                  event.preventDefault();
                  event.stopPropagation();
                  onAcceptSuggestion();
                }}
              >
                <CheckSquare className="mr-1 h-3 w-3" />
                Accept
              </Button>
              <Button
                size="sm"
                variant="outline"
                disabled={dismissPending}
                onClick={(event) => {
                  event.preventDefault();
                  event.stopPropagation();
                  onDismissSuggestion();
                }}
              >
                <Archive className="mr-1 h-3 w-3" />
                {dismissPending ? "Dismissing..." : "Dismiss"}
              </Button>
            </div>
          )}

          {/* Blocked state */}
          {itemActions.blocked && !itemActions.locked && (
            <div className="mt-3 space-y-2" onClick={(event) => event.preventDefault()}>
              {(itemActions.finalizeDisabled || itemActions.workshopDisabled || (itemActions.runDisabled && !itemActions.workshopDisabled && !itemActions.finalizeDisabled)) && (
                <div className={actionRowClass} data-testid="backlog-card-actions">
                  {(itemActions.finalizeDisabled) && (
                    <Button
                      size="sm"
                      variant="outline"
                      disabled
                      onClick={(event) => { event.preventDefault(); event.stopPropagation(); }}
                    >
                      <Sparkles className="mr-1 h-3 w-3" />
                      Finalize
                    </Button>
                  )}
                  {(itemActions.workshopDisabled) && (
                    <Button
                      size="sm"
                      variant="outline"
                      disabled
                      onClick={(event) => { event.preventDefault(); event.stopPropagation(); }}
                    >
                      <MessageSquareText className="mr-1 h-3 w-3" />
                      {workshopLabel}
                    </Button>
                  )}
                  {(itemActions.runDisabled && !itemActions.workshopDisabled && !itemActions.finalizeDisabled) && (
                    <Button
                      size="sm"
                      disabled
                      onClick={(event) => { event.preventDefault(); event.stopPropagation(); }}
                    >
                      <Play className="mr-1 h-3 w-3" />
                      Run
                    </Button>
                  )}
                </div>
              )}
              <p className="text-xs text-slate-500">
                Blocked by dependencies
              </p>
            </div>
          )}

          {/* Primary action row */}
          {hasPrimaryActionRow && (
            <div className={actionRowClass} data-testid="backlog-card-actions" onClick={(event) => event.preventDefault()}>
              {(itemActions.canFinalize || itemActions.finalizeDisabled) && (
                <Button
                  size="sm"
                  variant={itemActions.primaryCta === "finalize" ? "default" : "outline"}
                  disabled={itemActions.finalizeDisabled || finalizePending}
                  onClick={(event) => {
                    event.preventDefault();
                    event.stopPropagation();
                    onFinalize();
                  }}
                >
                  <Sparkles className="mr-1 h-3 w-3" />
                  {itemActions.agentExecuting ? runningLabel : finalizePending ? "Starting..." : "Finalize"}
                </Button>
              )}
              {(itemActions.canRun || itemActions.runDisabled) && (
                <Button
                  size="sm"
                  variant={itemActions.primaryCta === "run" ? "default" : "outline"}
                  disabled={itemActions.runDisabled}
                  title={itemActions.runDisabled && itemActions.disabledReason ? itemActions.disabledReason : undefined}
                  onClick={(event) => {
                    event.preventDefault();
                    event.stopPropagation();
                    onRun();
                  }}
                >
                  <Play className="mr-1 h-3 w-3" />
                  {itemActions.agentExecuting ? runningLabel : "Run"}
                </Button>
              )}
              {(itemActions.canWorkshop || itemActions.workshopDisabled) && (
                <Button
                  size="sm"
                  variant={itemActions.primaryCta === "workshop" ? "default" : "outline"}
                  disabled={itemActions.workshopDisabled || workshopPending}
                  title={(itemActions.workshopDisabled || workshopPending) && itemActions.disabledReason ? itemActions.disabledReason : undefined}
                  onClick={(event) => {
                    event.preventDefault();
                    event.stopPropagation();
                    onWorkshop();
                  }}
                >
                  <MessageSquareText className="mr-1 h-3 w-3" />
                  {itemActions.agentExecuting ? runningLabel : workshopPending ? "Starting..." : workshopLabel}
                </Button>
              )}
              {showSnooze && onSnooze && (
                <SnoozePopover itemKey={snoozeItemKey} onSnooze={onSnooze}>
                  <Clock className="h-3.5 w-3.5" />
                </SnoozePopover>
              )}
            </div>
          )}

          {/* Terminal actions: Follow Up + Archive in a single row */}
          {(itemActions.canFollowUp || itemActions.canArchive) && (
            <div className={actionRowClass} data-testid="backlog-card-actions" onClick={(event) => event.preventDefault()}>
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
              {showSnooze && onSnooze && (
                <SnoozePopover itemKey={snoozeItemKey} onSnooze={onSnooze}>
                  <Clock className="h-3.5 w-3.5" />
                </SnoozePopover>
              )}
            </div>
          )}

          {/* Standalone snooze when no action rows are shown */}
          {showSnooze && onSnooze && !hasPrimaryActionRow && !itemActions.canFollowUp && !itemActions.canArchive && !itemActions.blocked && (
            <div className={actionRowClass} onClick={(event) => event.preventDefault()}>
              <SnoozePopover itemKey={snoozeItemKey} onSnooze={onSnooze}>
                <Clock className="h-3.5 w-3.5" />
              </SnoozePopover>
            </div>
          )}

          {/* Disabled reason for buttons that are shown but not clickable */}
          {itemActions.disabledReason && (itemActions.runDisabled || itemActions.workshopDisabled || itemActions.finalizeDisabled) && (
            <p className="mt-3 text-xs text-amber-400/80">
              {itemActions.disabledReason}
            </p>
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

export const BacklogCard = memo(BacklogCardImpl);
