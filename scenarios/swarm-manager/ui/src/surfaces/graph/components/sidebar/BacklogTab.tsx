/**
 * BacklogTab - Lists backlog items with rich action cards.
 *
 * Uses the shared BacklogCard component, providing inline
 * decision answering, run/workshop/finalize actions, and follow-up/archive.
 */

import { Profiler, memo, useCallback, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { onProfilerRender } from "../../../../lib/profiler";
import { ListTodo } from "lucide-react";
import { useBacklogStore } from "../../../../stores";
import { useSnoozedKeys } from "../../../../stores/snooze-store";
import { getItemActions } from "../../../../lib";
import { buildBacklogCompareFn, sortBacklogItems } from "../../../../lib/backlog-sort";
import { computeUnblockingMap } from "../../../../lib/dependency-sort";
import { filterSnoozed, snoozeKeyForBacklog } from "../../../../lib/snooze-utils";
import { buildBacklogNodeId } from "../../lib/node-id-parser";
import { matchesSearch } from "./useSidebarSearch";
import { BacklogCard } from "../../../../components/backlog/backlog-card";
import { RunBacklogModal } from "../../../../components/backlog/run-backlog-modal";
import type { RunBacklogTarget } from "../../../../components/backlog/run-backlog-modal";
import { ConfirmDialog } from "../../../../components/ui/confirm-dialog";
import { useCommandPostItemActions } from "../../../../hooks/useCommandPostItemActions";
import type { ItemCallbacks } from "../../../../hooks/useCommandPostItemActions";
import type { BacklogItem, ItemBlockingInfo, PendingQuestion } from "../../../../types";
import type { BacklogFilters, SortConfig } from "./types";
import type { AttentionReason } from "../../../../lib/feed";
import type { ReadinessIndicatorData } from "../../../../lib/maturity";
import type { StepperCompletionResult } from "../../../../components/backlog/inline-question-stepper";
import { backlogDetailPath } from "../../../../app/routes/route-paths";
import { SidebarEmptyState } from "./SidebarEmptyState";

interface PlanValidationSummary {
  passed: boolean;
}

function parsePlanValidationSummary(validationJson: string): PlanValidationSummary | null {
  try {
    const parsed: unknown = JSON.parse(validationJson);
    if (
      typeof parsed === "object"
      && parsed !== null
      && "passed" in parsed
      && typeof parsed.passed === "boolean"
    ) {
      return { passed: parsed.passed };
    }
  } catch {
    return null;
  }
  return null;
}

interface BacklogTabProps {
  searchQuery: string;
  filters: BacklogFilters;
  sort: SortConfig;
  onItemClick: (nodeId: string) => void;
  onClearSearch?: () => void;
}

function applyFilters(items: BacklogItem[], filters: BacklogFilters): BacklogItem[] {
  return items.filter((item) => {
    // Hide archived items unless showArchived is on
    if (item.archivedAt != null && !filters.showArchived) return false;
    if (filters.statuses.length > 0 && !filters.statuses.includes(item.status)) return false;
    if (filters.kinds.length > 0 && !filters.kinds.includes(item.kind)) return false;
    if (filters.priorityMin !== null && item.priority < filters.priorityMin) return false;
    if (filters.priorityMax !== null && item.priority > filters.priorityMax) return false;
    if (filters.validationStatus) {
      const json = item.planValidationJson;
      if (filters.validationStatus === "none") {
        if (json) return false;
      } else if (filters.validationStatus === "passed") {
        if (!json) return false;
        const validation = parsePlanValidationSummary(json);
        if (!validation?.passed) return false;
      } else if (filters.validationStatus === "failed") {
        if (!json) return false;
        const validation = parsePlanValidationSummary(json);
        if (validation?.passed !== false) return false;
      }
    }
    return true;
  });
}

function hasActiveFilters(filters: BacklogFilters): boolean {
  return filters.statuses.length > 0 || filters.kinds.length > 0 || filters.priorityMin !== null || filters.priorityMax !== null || filters.showArchived || filters.validationStatus !== "";
}

function BacklogTabImpl({ searchQuery, filters, sort, onItemClick, onClearSearch }: BacklogTabProps) {
  const navigate = useNavigate();
  const items = useBacklogStore((s) => s.items);
  const blockingMap = useBacklogStore((s) => s.blockingMap);
  const fetchBacklog = useBacklogStore((s) => s.fetchBacklog);
  const snoozedKeys = useSnoozedKeys();

  const [runModalTarget, setRunModalTarget] = useState<RunBacklogTarget | undefined>();

  // Stable callbacks for the action hook so its internal memos don't churn.
  const handleSelectBacklog = useCallback(
    (kind: string, name: string) =>
      navigate(backlogDetailPath(kind as BacklogItem["kind"], name)),
    [navigate],
  );
  const handleRunItem = useCallback(
    (kind: BacklogItem["kind"], name: string, title?: string) =>
      setRunModalTarget({ kind, name, title: title ?? "" }),
    [],
  );

  // ── Shared action wiring ────────────────────────────────────────────
  const {
    getItemCallbacks,
    activeRunKeys,
    readinessMap,
    pendingQuestionsMap,
    attentionReasonsMap,
    completedSteppers,
    transitionItems,
    handleStepperCompleted,
    workshopBlockingConfirm,
    setWorkshopBlockingConfirm,
    confirmWorkshopOverride,
  } = useCommandPostItemActions({
    onSelectBacklog: handleSelectBacklog,
    onRunItem: handleRunItem,
  });

  // ── Filter, search, snooze, sort ─────────────────────────────────────
  // Unblocking map is O(n²)-ish on the full backlog; memoize on items alone
  // so it doesn't recompute when filters or search change.
  const unblockingMap = useMemo(() => computeUnblockingMap(items), [items]);

  const sorted = useMemo(() => {
    let next = applyFilters(items, filters);
    if (searchQuery) {
      next = next.filter((item) =>
        matchesSearch(searchQuery, item.title, item.name, item.description, ...(item.tags ?? [])),
      );
    }
    next = filterSnoozed(next, (item) => snoozeKeyForBacklog(item.kind, item.name), snoozedKeys);
    return sortBacklogItems(next, buildBacklogCompareFn(sort, unblockingMap), items);
  }, [items, filters, searchQuery, sort, snoozedKeys, unblockingMap]);

  const handleCloseRunModal = useCallback(() => setRunModalTarget(undefined), []);
  const handleRunModalSuccess = useCallback(() => {
    setRunModalTarget(undefined);
    void fetchBacklog({ force: true });
  }, [fetchBacklog]);
  const handleCloseWorkshopConfirm = useCallback(
    () => setWorkshopBlockingConfirm(null),
    [setWorkshopBlockingConfirm],
  );

  if (sorted.length === 0) {
    const filtersActive = hasActiveFilters(filters);
    const title = filtersActive
      ? "No backlog items match your filters."
      : "No backlog items yet.";
    return (
      <SidebarEmptyState
        icon={ListTodo}
        title={title}
        hint={filtersActive ? undefined : "Capture an idea or chore to get started."}
        query={searchQuery}
        onClearSearch={onClearSearch}
      />
    );
  }

  return (
    <>
      <div className="space-y-2">
        {sorted.map((item) => {
          const itemKey = `${item.kind}/${item.name}`;
          return (
            <BacklogRow
              key={itemKey}
              item={item}
              allItems={items}
              blockingInfo={blockingMap[itemKey] ?? null}
              readiness={readinessMap.get(itemKey)}
              attentionReasons={attentionReasonsMap.get(itemKey) ?? EMPTY_REASONS}
              pendingQuestions={pendingQuestionsMap.get(itemKey)}
              agentRunning={activeRunKeys.has(itemKey)}
              isStepperCompleted={completedSteppers.has(itemKey)}
              transitionResult={transitionItems.get(itemKey)}
              getItemCallbacks={getItemCallbacks}
              handleStepperCompleted={handleStepperCompleted}
              onItemClick={onItemClick}
            />
          );
        })}
      </div>

      {/* Run modal */}
      <RunBacklogModal
        isOpen={!!runModalTarget}
        onClose={handleCloseRunModal}
        target={runModalTarget}
        onSuccess={handleRunModalSuccess}
      />

      {/* Workshop blocking override confirmation */}
      <ConfirmDialog
        isOpen={!!workshopBlockingConfirm}
        onClose={handleCloseWorkshopConfirm}
        onConfirm={confirmWorkshopOverride}
        title="Dependencies Not Ready"
        description={
          workshopBlockingConfirm?.blockingDepKeys.length
            ? `This item is blocked by incomplete dependencies: ${workshopBlockingConfirm.blockingDepKeys.join(", ")}. Do you want to proceed anyway?`
            : "This item has incomplete dependencies. Do you want to proceed anyway?"
        }
        confirmLabel="Override and Proceed"
      />
    </>
  );
}

function BacklogTabWrapped(props: BacklogTabProps) {
  return (
    <Profiler id="BacklogTab" onRender={onProfilerRender}>
      <BacklogTabImpl {...props} />
    </Profiler>
  );
}

export const BacklogTab = memo(BacklogTabWrapped);

// Stable empty array, returned when a key has no attention reasons. Keeping
// the reference constant lets the memoized row skip re-renders that would
// otherwise be triggered by a fresh `[]` literal each render.
const EMPTY_REASONS: AttentionReason[] = [];

interface BacklogRowProps {
  item: BacklogItem;
  allItems: BacklogItem[];
  blockingInfo: ItemBlockingInfo | null;
  readiness: ReadinessIndicatorData | undefined;
  attentionReasons: AttentionReason[];
  pendingQuestions: PendingQuestion[] | undefined;
  agentRunning: boolean;
  isStepperCompleted: boolean;
  transitionResult: StepperCompletionResult | undefined;
  getItemCallbacks: (item: BacklogItem) => ItemCallbacks;
  handleStepperCompleted: (itemKey: string, item: BacklogItem, result: StepperCompletionResult) => void;
  onItemClick: (nodeId: string) => void;
}

const BacklogRow = memo(function BacklogRow({
  item,
  allItems,
  blockingInfo,
  readiness,
  attentionReasons,
  pendingQuestions,
  agentRunning,
  isStepperCompleted,
  transitionResult,
  getItemCallbacks,
  handleStepperCompleted,
  onItemClick,
}: BacklogRowProps) {
  const itemKey = `${item.kind}/${item.name}`;
  const nodeId = useMemo(() => buildBacklogNodeId(item.kind, item.name), [item.kind, item.name]);
  const hasPendingDecisions = (pendingQuestions?.length ?? 0) > 0;
  const itemActions = useMemo(
    () =>
      getItemActions({
        item,
        blockingInfo,
        readinessReady: readiness ? readiness.ready : null,
        pendingSynthesis: readiness?.pendingSynthesis ?? false,
        agentRunning,
        hasPendingDecisions,
        hasExecutionHistory: item.status === "completed" || item.status === "failed",
      }),
    [item, blockingInfo, readiness, agentRunning, hasPendingDecisions],
  );
  const callbacks = getItemCallbacks(item);
  const handleClick = useCallback(() => onItemClick(nodeId), [nodeId, onItemClick]);
  const handleStepperCompletedForItem = useCallback(
    (result: StepperCompletionResult) => handleStepperCompleted(itemKey, item, result),
    [handleStepperCompleted, itemKey, item],
  );
  const noopToggleSelection = useCallback(() => {}, []);

  return (
    <button
      type="button"
      onClick={handleClick}
      className="group w-full rounded-lg border border-slate-800/80 bg-slate-900/50 p-2.5 text-left transition-colors hover:border-slate-700/80 hover:bg-slate-800/60"
      data-testid="sidebar-backlog-item"
    >
      <BacklogCard
        item={item}
        allItems={allItems}
        readinessData={readiness}
        itemActions={itemActions}
        attentionReasons={attentionReasons}
        pendingQuestions={pendingQuestions}
        isStepperCompleted={isStepperCompleted}
        transitionResult={transitionResult}
        onStepperCompleted={handleStepperCompletedForItem}
        batchMode={false}
        isSelected={false}
        onToggleSelection={noopToggleSelection}
        {...callbacks}
      />
    </button>
  );
});
