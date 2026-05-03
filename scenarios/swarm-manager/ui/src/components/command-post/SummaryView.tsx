/**
 * SummaryView — The landing/triage view inside the Command Post overlay.
 *
 * Consumes stores directly, computes action groups via sortedGroupActionItems(),
 * and renders ActionGroupCards, a prioritized BacklogCard feed,
 * snoozed section, and recent activity section.
 *
 * Cards act as filters: tapping a card body filters the "Needs Attention"
 * feed to show only that group's items. Tapping again clears the filter.
 */

import { Profiler, memo, useCallback, useMemo, useState } from "react";
import { onProfilerRender } from "../../lib/profiler";
import { useBacklogStore } from "../../stores/backlog-store";
import { useExecutionStore } from "../../stores/execution-store";
import { useCaptureStore } from "../../stores/capture-store";
import { useSnoozeStore, useSnoozedKeys } from "../../stores/snooze-store";
import { sortedGroupActionItems, type ActionGroup, type ActionGroupId, type ActionableItem } from "../../lib/command-post-utils";
import { getItemActions } from "../../lib/backlog-queue-utils";
import { useCommandPostItemActions } from "../../hooks/useCommandPostItemActions";
import type { RunBacklogTarget } from "../backlog/run-backlog-modal";
import { RunBacklogModal } from "../backlog/run-backlog-modal";
import { ConfirmDialog } from "../ui/confirm-dialog";
import { ActionGroupCard } from "./ActionGroupCard";
import { BacklogCard } from "../backlog/backlog-card";
import { ExecutionCaptureCard } from "./ExecutionCaptureCard";
import { SnoozedSection } from "./SnoozedSection";
import { RecentSection } from "./RecentSection";
import { EmptyState } from "./EmptyState";
import type { DetailRouteTarget } from "../../app/routes/route-paths";

/** Bulk actions spawning more than this many agents require confirmation. */
const BULK_AGENT_CONFIRM_THRESHOLD = 3;

interface SummaryViewProps {
  onEnterDecisionStream: () => void;
  onNavigateToDetail: (selection: DetailRouteTarget) => void;
  onSwitchLens: (lens: string) => void;
}

export function SummaryView(props: SummaryViewProps) {
  return (
    <Profiler id="SummaryView" onRender={onProfilerRender}>
      <SummaryViewImpl {...props} />
    </Profiler>
  );
}

function SummaryViewImpl({
  onEnterDecisionStream,
  onNavigateToDetail,
  onSwitchLens,
}: SummaryViewProps) {
  const backlogItems = useBacklogStore((s) => s.items);
  const blockingMap = useBacklogStore((s) => s.blockingMap);
  const executions = useExecutionStore((s) => s.items);
  const captures = useCaptureStore((s) => s.captures);
  const snoozeEntries = useSnoozeStore((s) => s.entries);
  const snooze = useSnoozeStore((s) => s.snooze);
  const unsnooze = useSnoozeStore((s) => s.unsnooze);
  const snoozedKeys = useSnoozedKeys();

  const [runModalTargets, setRunModalTargets] = useState<RunBacklogTarget[] | undefined>();
  const [activeFilter, setActiveFilter] = useState<ActionGroupId | null>(null);
  const [pendingBulkAction, setPendingBulkAction] = useState<{ group: ActionGroup; targets: RunBacklogTarget[] } | null>(null);

  // ── Shared item action wiring ──────────────────────────────────────
  const itemActionsHook = useCommandPostItemActions({
    onSelectBacklog: (kind, name) => {
      onNavigateToDetail({ entityType: "backlog", kind, name });
    },
    onRunItem: (kind, name, title) => {
      setRunModalTargets([{ kind, name, title }]);
    },
  });

  const {
    getItemCallbacks,
    activeRunKeys,
    activeRunLabels,
    readinessMap,
    pendingQuestionsMap,
    attentionReasonsMap,
    completedSteppers,
    transitionItems,
    handleStepperCompleted,
    workshopBlockingConfirm,
    setWorkshopBlockingConfirm,
    confirmWorkshopOverride,
    pendingArchiveKey,
    pendingWorkshop,
    pendingStatusKey,
  } = itemActionsHook;

  // ── Feed/feedback/maturity maps for groupActionItems ───────────────
  // These maps are derived from the same summary query inside the hook,
  // but groupActionItems needs FeedbackItem/MaturityItem maps. We build
  // them from the hook's attentionReasonsMap source data via a separate
  // lightweight derivation from the summary query.
  const feedbackMap = useMemo(() => {
    // Build from pendingQuestionsMap — if an item has pending questions,
    // it has pending decisions
    const map = new Map<string, { kind: string; name: string; pendingDecisions: number }>();
    for (const [key, questions] of pendingQuestionsMap) {
      const [kind, name] = key.split("/");
      if (kind && name) {
        map.set(key, { kind, name, pendingDecisions: questions.length });
      }
    }
    return map;
  }, [pendingQuestionsMap]);

  const maturityMap = useMemo(() => {
    const map = new Map<string, { kind: string; name: string; ready: boolean; pendingItems: number }>();
    for (const [key, data] of readinessMap) {
      const [kind, name] = key.split("/");
      if (kind && name) {
        map.set(key, { kind, name, ready: data.ready, pendingItems: data.pendingItems });
      }
    }
    return map;
  }, [readinessMap]);

  const groups = useMemo(
    () => sortedGroupActionItems(backlogItems, executions, captures, feedbackMap, maturityMap, snoozedKeys),
    [backlogItems, executions, captures, feedbackMap, maturityMap, snoozedKeys],
  );

  const allItems = useMemo(() => groups.flatMap((g) => g.items), [groups]);
  const totalCount = useMemo(() => groups.reduce((sum, g) => sum + g.count, 0), [groups]);

  // Filter feed items when a card is active
  const visibleItems = useMemo(() => {
    if (!activeFilter) return allItems;
    const activeGroup = groups.find((g) => g.id === activeFilter);
    return activeGroup?.items ?? [];
  }, [allItems, groups, activeFilter]);

  const snoozedEntries = useMemo(
    () => Array.from(snoozeEntries.values()),
    [snoozeEntries],
  );

  const handleSnooze = useCallback(
    (key: string, expiresAt: number) => {
      snooze(key, expiresAt);
    },
    [snooze],
  );

  const navigateToItem = useCallback(
    (item: { type: string; kind?: string; name?: string; executionId?: string }) => {
      if (item.type === "backlog" && item.kind && item.name) {
        onNavigateToDetail({ entityType: "backlog", kind: item.kind, name: item.name });
      } else if (item.type === "execution" && item.executionId) {
        onNavigateToDetail({ entityType: "execution", identifier: item.executionId });
      }
    },
    [onNavigateToDetail],
  );

  const executeBulkAgentAction = useCallback(
    (targets: RunBacklogTarget[]) => {
      if (targets.length > 0) {
        setRunModalTargets(targets);
      }
    },
    [],
  );

  const handleBulkAction = useCallback(
    (group: ActionGroup) => {
      switch (group.id) {
        case "needs-workshop":
        case "ready-to-run": {
          const targets: RunBacklogTarget[] = group.items
            .filter((i) => i.kind && i.name)
            .map((i) => ({ kind: i.kind as RunBacklogTarget["kind"], name: i.name ?? "", title: i.title }));
          if (targets.length > BULK_AGENT_CONFIRM_THRESHOLD) {
            setPendingBulkAction({ group, targets });
          } else {
            executeBulkAgentAction(targets);
          }
          break;
        }
        case "pending-decisions":
          onEnterDecisionStream();
          break;
        case "needs-review":
        case "needs-classification": {
          const first = group.items[0];
          if (first) navigateToItem(first);
          break;
        }
      }
    },
    [onEnterDecisionStream, navigateToItem, executeBulkAgentAction],
  );

  const handleFilter = useCallback(
    (groupId: ActionGroupId) => {
      setActiveFilter((prev) => (prev === groupId ? null : groupId));
    },
    [],
  );

  if (totalCount === 0) {
    return <EmptyState onSwitchLens={onSwitchLens} />;
  }

  return (
    <div className="space-y-4" data-testid="command-post-summary">
      {/* Action group cards — 2-column grid, full width */}
      <div className="grid grid-cols-2 gap-3">
        {groups.map((group) => (
          <ActionGroupCard
            key={group.id}
            group={group}
            isActive={activeFilter === group.id}
            onBulkAction={() => handleBulkAction(group)}
            onFilter={() => handleFilter(group.id)}
          />
        ))}
      </div>

      {/* Prioritized feed — filtered when a card is active */}
      <div className="space-y-2">
        <div className="flex items-center gap-2">
          <h3 className="text-sm font-medium text-slate-400">
            {activeFilter ? groups.find((g) => g.id === activeFilter)?.label ?? "Needs Attention" : "Needs Attention"}
          </h3>
          {activeFilter && (
            <button
              type="button"
              onClick={() => setActiveFilter(null)}
              className="rounded px-1.5 py-0.5 text-[10px] text-slate-500 transition-colors hover:bg-slate-800 hover:text-slate-300"
            >
              Clear filter
            </button>
          )}
        </div>
        {visibleItems.map((item) => {
          const bi = item.type === "backlog" ? item.backlogItem : undefined;
          const itemKey = bi ? `${bi.kind}/${bi.name}` : "";
          const readiness = bi ? readinessMap.get(itemKey) : undefined;
          return (
            <FeedItem
              key={item.key}
              item={item}
              blockingInfo={bi ? blockingMap[itemKey] ?? null : null}
              readiness={readiness}
              attentionReasons={bi ? attentionReasonsMap.get(itemKey) ?? EMPTY_ATTENTION_REASONS : EMPTY_ATTENTION_REASONS}
              pendingQuestions={bi ? pendingQuestionsMap.get(itemKey) : undefined}
              agentRunning={bi ? activeRunKeys.has(itemKey) : false}
              isStepperCompleted={bi ? completedSteppers.has(itemKey) : false}
              transitionResult={bi ? transitionItems.get(itemKey) : undefined}
              callbacks={bi ? getItemCallbacks(bi) : undefined}
              archivePending={pendingArchiveKey === itemKey}
              finalizePending={pendingWorkshop?.key === itemKey && pendingWorkshop.mode === "finalize"}
              workshopPending={pendingWorkshop?.key === itemKey && pendingWorkshop.mode === "workshop"}
              statusChangePending={pendingStatusKey === itemKey}
              workshopLabel={(readiness?.roundsCompleted ?? 0) > 0 ? "Next Round" : "Workshop"}
              runningLabel={bi ? activeRunLabels.get(itemKey) : undefined}
              handleStepperCompleted={handleStepperCompleted}
              handleSnooze={handleSnooze}
              navigateToItem={navigateToItem}
            />
          );
        })}
        {visibleItems.length === 0 && activeFilter && (
          <p className="py-4 text-center text-xs text-slate-500">No items in this group</p>
        )}
      </div>

      {/* Snoozed section */}
      <SnoozedSection
        entries={snoozedEntries}
        onUnsnooze={unsnooze}
      />

      {/* Recent activity */}
      <RecentSection />

      {/* Bulk agent confirmation */}
      <ConfirmDialog
        isOpen={!!pendingBulkAction}
        onClose={() => setPendingBulkAction(null)}
        onConfirm={() => {
          if (pendingBulkAction) {
            executeBulkAgentAction(pendingBulkAction.targets);
          }
          setPendingBulkAction(null);
        }}
        title={`${pendingBulkAction?.group.label ?? "Bulk action"} (${pendingBulkAction?.targets.length ?? 0} items)`}
        description={`This will spawn ${pendingBulkAction?.targets.length ?? 0} agent sessions. Are you sure you want to proceed?`}
        confirmLabel="Proceed"
      />

      {/* Workshop blocking override confirmation */}
      <ConfirmDialog
        isOpen={!!workshopBlockingConfirm}
        onClose={() => setWorkshopBlockingConfirm(null)}
        onConfirm={confirmWorkshopOverride}
        title="Dependencies Not Ready"
        description={
          workshopBlockingConfirm?.blockingDepKeys.length
            ? `This item is blocked by incomplete dependencies: ${workshopBlockingConfirm.blockingDepKeys.join(", ")}. Do you want to proceed anyway?`
            : "This item has incomplete dependencies. Do you want to proceed anyway?"
        }
        confirmLabel="Override and Proceed"
      />

      {/* Run modal */}
      <RunBacklogModal
        isOpen={!!runModalTargets}
        onClose={() => setRunModalTargets(undefined)}
        targets={runModalTargets}
        onSuccess={() => setRunModalTargets(undefined)}
      />
    </div>
  );
}

// ---------------------------------------------------------------------------
// FeedItem — Renders BacklogCard for backlog items, ExecutionCaptureCard for others
// ---------------------------------------------------------------------------

// Stable empty-array reference so non-attention-reason rows don't see a
// fresh `[]` literal each render and break BacklogCard memo equality.
const EMPTY_ATTENTION_REASONS: import("../../lib/feed").AttentionReason[] = [];
const NOOP_TOGGLE_SELECTION = () => {};

interface FeedItemProps {
  item: ActionableItem;
  blockingInfo: import("../../types").ItemBlockingInfo | null;
  readiness: import("../../lib/maturity").ReadinessIndicatorData | undefined;
  attentionReasons: import("../../lib/feed").AttentionReason[];
  pendingQuestions: import("../../types").PendingQuestion[] | undefined;
  agentRunning: boolean;
  isStepperCompleted: boolean;
  transitionResult: import("../backlog/inline-question-stepper").StepperCompletionResult | undefined;
  /** Stable per-item callbacks — undefined for non-backlog rows. */
  callbacks: import("../../hooks/useCommandPostItemActions").StableItemCallbacks | undefined;
  archivePending: boolean;
  finalizePending: boolean;
  workshopPending: boolean;
  statusChangePending: boolean;
  workshopLabel: string;
  runningLabel: string | undefined;
  handleStepperCompleted: (itemKey: string, item: import("../../types").BacklogItem, result: import("../backlog/inline-question-stepper").StepperCompletionResult) => void;
  handleSnooze: (key: string, expiresAt: number) => void;
  navigateToItem: (item: { type: string; kind?: string; name?: string; executionId?: string }) => void;
}

const FeedItem = memo(function FeedItem({
  item,
  blockingInfo,
  readiness,
  attentionReasons,
  pendingQuestions,
  agentRunning,
  isStepperCompleted,
  transitionResult,
  callbacks,
  archivePending,
  finalizePending,
  workshopPending,
  statusChangePending,
  workshopLabel,
  runningLabel,
  handleStepperCompleted,
  handleSnooze,
  navigateToItem,
}: FeedItemProps) {
  const handleNavigate = useCallback(() => navigateToItem(item), [navigateToItem, item]);

  if (item.type === "backlog" && item.backlogItem && callbacks) {
    const bi = item.backlogItem;
    const itemKey = `${bi.kind}/${bi.name}`;
    const itemActions = getItemActions({
      item: bi,
      blockingInfo,
      readinessReady: readiness ? readiness.ready : null,
      pendingSynthesis: readiness?.pendingSynthesis ?? false,
      agentRunning,
      hasPendingDecisions: (pendingQuestions?.length ?? 0) > 0,
      hasExecutionHistory: bi.status === "completed" || bi.status === "failed",
    });
    const onStepperCompleted = (result: import("../backlog/inline-question-stepper").StepperCompletionResult) =>
      handleStepperCompleted(itemKey, bi, result);

    return (
      <button
        type="button"
        onClick={handleNavigate}
        className="group w-full rounded-lg border border-slate-800/80 bg-slate-900/50 p-2.5 text-left transition-colors hover:border-slate-700/80 hover:bg-slate-800/60"
        data-testid={`command-post-feed-item-${item.key}`}
      >
        <BacklogCard
          item={bi}
          readinessData={readiness}
          itemActions={itemActions}
          attentionReasons={attentionReasons}
          pendingQuestions={pendingQuestions}
          isStepperCompleted={isStepperCompleted}
          transitionResult={transitionResult}
          onStepperCompleted={onStepperCompleted}
          batchMode={false}
          isSelected={false}
          onToggleSelection={NOOP_TOGGLE_SELECTION}
          onRun={callbacks.onRun}
          onArchive={callbacks.onArchive}
          onFollowUp={callbacks.onFollowUp}
          onFinalize={callbacks.onFinalize}
          onWorkshop={callbacks.onWorkshop}
          onStatusChange={callbacks.onStatusChange}
          archivePending={archivePending}
          finalizePending={finalizePending}
          workshopPending={workshopPending}
          statusChangePending={statusChangePending}
          workshopLabel={workshopLabel}
          runningLabel={runningLabel}
          showSnooze
          onSnooze={handleSnooze}
        />
      </button>
    );
  }

  // Execution/capture items: minimal card
  return (
    <ExecutionCaptureCard
      item={item}
      onNavigate={handleNavigate}
      onSnooze={handleSnooze}
    />
  );
});
