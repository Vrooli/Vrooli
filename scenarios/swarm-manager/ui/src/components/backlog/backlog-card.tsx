/**
 * BacklogCard — shared card interior for backlog items.
 *
 * Renders everything inside `<ResponsiveListItem>`. The wrapper/Link stays
 * in the parent component so routing context is controlled by the caller.
 *
 * Primary action visibility and labels come from the server next-action
 * projection. While it loads, the card shows a truthful loading state.
 */

import { memo } from "react";
import { Archive, ArrowRight, Clock, Lock, Play } from "lucide-react";
import { Button } from "../ui/button";
import { TagList } from "../ui/tag-list";
import { formatRelativeTime } from "../../lib";
import { BACKLOG_KIND_ICONS, BACKLOG_STATUS_COLORS, formatBacklogStatus } from "../../types";
import type { BacklogItem, BacklogStatus, PendingQuestion } from "../../types";
import { StatusChipPopover } from "./status-chip-popover";
import type { ItemActions } from "../../lib/backlog-queue-utils";
import type { BacklogNextAction } from "../../services/backlog";
import type { AttentionReason } from "../../lib/attention";
import type { StepperCompletionResult } from "./inline-question-stepper";
import { InlineQuestionStepper } from "./inline-question-stepper";
import { PendingDecisionBadge } from "./pending-decision-badge";
import { MilestoneBadge } from "./milestone-badge";
import { DependencyIndicator } from "./dependency-indicator";
import { ScenarioBadge } from "./scenario-badge";
import { AgentRunningBadge } from "./agent-running-badge";
import { CircuitBrokenBadge } from "./circuit-broken-badge";
import { NoteIndicator } from "../ui/note-indicator";
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
  /**
   * Picker pick-mode selection contract. When `selection.selectionMode` is
   * true the card renders a compact display-only summary (no action rows,
   * no stepper) wrapped in PickModeRow. All the sidebar action props below
   * are ignored in that mode, so they are optional for picker callers.
   */
  selection?: CardSelection;
  itemActions?: ItemActions;
  /** Server-owned primary action projection for list and graph contexts. */
  nextAction?: BacklogNextAction;
  // Attention / stepper
  attentionReasons?: AttentionReason[];
  pendingQuestions?: PendingQuestion[];
  isStepperCompleted?: boolean;
  onStepperCompleted?: (result: StepperCompletionResult) => void;
  // Batch mode
  batchMode?: boolean;
  isSelected?: boolean;
  onToggleSelection?: () => void;
  // Actions
  onRun?: () => void;
  onNextAction?: () => void;
  onArchive?: () => void;
  onFollowUp?: () => void;
  onAcceptSuggestion?: () => void;
  onDismissSuggestion?: () => void;
  archivePending?: boolean;
  dismissPending?: boolean;
  /** Human-readable label shown when an agent is running. */
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
  selection,
  itemActions = NO_ITEM_ACTIONS,
  nextAction,
  attentionReasons = [],
  pendingQuestions,
  isStepperCompleted = false,
  onStepperCompleted = () => {},
  batchMode = false,
  isSelected = false,
  onToggleSelection = () => {},
  onNextAction = () => {},
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
  const KindIcon = BACKLOG_KIND_ICONS[item.kind];
  const hasPrimaryActionRow = nextAction?.id !== "none";
  const actionProjectionPending = nextAction === undefined;
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

      {/* Body: pending independent review or normal content */}
      {hasActiveStepper ? (
        <InlineQuestionStepper
          questions={pendingQuestions as PendingQuestion[]}
          backlogKind={item.kind}
          backlogName={item.name}
          onAllAnswered={onStepperCompleted}
        />
      ) : (
        <>
          {attentionReasons.length > 0 && (
            <div className="mt-2">
              <PendingDecisionBadge reasons={attentionReasons} />
            </div>
          )}
          {(item.milestone || (item.dependsOn && item.dependsOn.length > 0)) && (
            <div className="mt-2 flex flex-wrap gap-1">
              <MilestoneBadge milestone={item.milestone} />
              <DependencyIndicator dependsOn={item.dependsOn} />
            </div>
          )}
          <TagList
            tags={item.tags}
            maxTags={displayLimitsConfig.backlogCardMaxTags}
            className="mt-3"
          />
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

          {actionProjectionPending && !item.archivedAt && (
            <p className="mt-3 text-xs text-slate-500" data-testid="backlog-card-action-loading">Loading next action…</p>
          )}

          {/* Primary action row */}
          {hasPrimaryActionRow && (
            <div className={actionRowClass} data-testid="backlog-card-actions" onClick={(event) => event.preventDefault()}>
              {nextAction ? (
                <Button
                  size="sm"
                  variant="default"
                  disabled={!nextAction.enabled}
                  title={nextAction.reason}
                  onClick={(event) => {
                    event.preventDefault();
                    event.stopPropagation();
                    onNextAction();
                  }}
                >
                  <Play className="mr-1 h-3 w-3" />
                  {nextAction.expandedLabel}
                </Button>
              ) : null}
              {showSnooze && onSnooze && (
                <SnoozePopover itemKey={snoozeItemKey} onSnooze={onSnooze}>
                  <Clock className="h-3.5 w-3.5" />
                </SnoozePopover>
              )}
            </div>
          )}

          {/* Standalone snooze when no action rows are shown */}
          {showSnooze && onSnooze && !hasPrimaryActionRow && !actionProjectionPending && (
            <div className={actionRowClass} onClick={(event) => event.preventDefault()}>
              <SnoozePopover itemKey={snoozeItemKey} onSnooze={onSnooze}>
                <Clock className="h-3.5 w-3.5" />
              </SnoozePopover>
            </div>
          )}

          {/* Disabled reason for buttons that are shown but not clickable */}
          {itemActions.disabledReason && itemActions.runDisabled && (
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
