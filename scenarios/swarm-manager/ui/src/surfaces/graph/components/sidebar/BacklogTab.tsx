/**
 * BacklogTab - Lists backlog items with rich action cards.
 *
 * Uses the shared BacklogCard component, providing inline
 * decision answering, run/workshop/finalize actions, and follow-up/archive.
 */

import { Profiler, memo, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useVirtualizer } from "@tanstack/react-virtual";
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
import type { StableItemCallbacks } from "../../../../hooks/useCommandPostItemActions";
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
      <VirtualizedBacklogList
        sorted={sorted}
        blockingMap={blockingMap}
        readinessMap={readinessMap}
        attentionReasonsMap={attentionReasonsMap}
        pendingQuestionsMap={pendingQuestionsMap}
        activeRunKeys={activeRunKeys}
        activeRunLabels={activeRunLabels}
        completedSteppers={completedSteppers}
        transitionItems={transitionItems}
        getItemCallbacks={getItemCallbacks}
        pendingArchiveKey={pendingArchiveKey}
        pendingWorkshop={pendingWorkshop}
        pendingStatusKey={pendingStatusKey}
        handleStepperCompleted={handleStepperCompleted}
        onItemClick={onItemClick}
      />

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

// Rough card height before measurement; the virtualizer remeasures each
// rendered row via measureElement, so this only affects scrollbar-position
// estimation for unmeasured rows.
const ROW_HEIGHT_ESTIMATE_PX = 180;
// Buffer rows rendered above/below the viewport. Keeps scrolling smooth
// without negating the wins from virtualization.
const ROW_OVERSCAN = 6;
// Vertical gap between rows (matches the previous space-y-2 layout).
const ROW_GAP_PX = 8;

interface VirtualizedBacklogListProps {
  sorted: BacklogItem[];
  blockingMap: Record<string, ItemBlockingInfo>;
  readinessMap: Map<string, ReadinessIndicatorData>;
  attentionReasonsMap: Map<string, AttentionReason[]>;
  pendingQuestionsMap: Map<string, PendingQuestion[]>;
  activeRunKeys: Set<string>;
  activeRunLabels: Map<string, string>;
  completedSteppers: Set<string>;
  transitionItems: Map<string, StepperCompletionResult>;
  getItemCallbacks: (item: BacklogItem) => StableItemCallbacks;
  pendingArchiveKey: string | null;
  pendingWorkshop: { key: string; mode: "workshop" | "finalize" } | null;
  pendingStatusKey: string | null;
  handleStepperCompleted: (itemKey: string, item: BacklogItem, result: StepperCompletionResult) => void;
  onItemClick: (nodeId: string) => void;
}

function VirtualizedBacklogList({
  sorted,
  blockingMap,
  readinessMap,
  attentionReasonsMap,
  pendingQuestionsMap,
  activeRunKeys,
  activeRunLabels,
  completedSteppers,
  transitionItems,
  getItemCallbacks,
  pendingArchiveKey,
  pendingWorkshop,
  pendingStatusKey,
  handleStepperCompleted,
  onItemClick,
}: VirtualizedBacklogListProps) {
  const sentinelRef = useRef<HTMLDivElement>(null);
  const [scrollEl, setScrollEl] = useState<HTMLElement | null>(null);

  // The sidebar's existing overflow-y-auto container is our scroll parent;
  // walk up from the sentinel to find it. Done in an effect so we measure
  // after mount.
  useEffect(() => {
    let el: HTMLElement | null = sentinelRef.current?.parentElement ?? null;
    while (el && el !== document.body) {
      const cs = window.getComputedStyle(el);
      if (cs.overflowY === "auto" || cs.overflowY === "scroll") {
        setScrollEl(el);
        return;
      }
      el = el.parentElement;
    }
  }, []);

  const rowVirtualizer = useVirtualizer({
    count: sorted.length,
    getScrollElement: () => scrollEl,
    estimateSize: () => ROW_HEIGHT_ESTIMATE_PX,
    overscan: ROW_OVERSCAN,
    gap: ROW_GAP_PX,
  });

  const virtualItems = rowVirtualizer.getVirtualItems();
  const totalHeight = rowVirtualizer.getTotalSize();

  return (
    <div ref={sentinelRef} style={{ height: totalHeight, position: "relative", width: "100%" }}>
      {virtualItems.map((virtualRow) => {
        const item = sorted[virtualRow.index];
        if (!item) return null;
        const itemKey = `${item.kind}/${item.name}`;
        const readiness = readinessMap.get(itemKey);
        return (
          <div
            key={itemKey}
            data-index={virtualRow.index}
            ref={rowVirtualizer.measureElement}
            style={{
              position: "absolute",
              top: 0,
              left: 0,
              width: "100%",
              transform: `translateY(${virtualRow.start}px)`,
            }}
          >
            <BacklogRow
              item={item}
              blockingInfo={blockingMap[itemKey] ?? null}
              readiness={readiness}
              attentionReasons={attentionReasonsMap.get(itemKey) ?? EMPTY_REASONS}
              pendingQuestions={pendingQuestionsMap.get(itemKey)}
              agentRunning={activeRunKeys.has(itemKey)}
              isStepperCompleted={completedSteppers.has(itemKey)}
              transitionResult={transitionItems.get(itemKey)}
              callbacks={getItemCallbacks(item)}
              archivePending={pendingArchiveKey === itemKey}
              finalizePending={pendingWorkshop?.key === itemKey && pendingWorkshop.mode === "finalize"}
              workshopPending={pendingWorkshop?.key === itemKey && pendingWorkshop.mode === "workshop"}
              statusChangePending={pendingStatusKey === itemKey}
              workshopLabel={(readiness?.roundsCompleted ?? 0) > 0 ? "Next Round" : "Workshop"}
              runningLabel={activeRunLabels.get(itemKey)}
              handleStepperCompleted={handleStepperCompleted}
              onItemClick={onItemClick}
            />
          </div>
        );
      })}
    </div>
  );
}

interface BacklogRowProps {
  item: BacklogItem;
  blockingInfo: ItemBlockingInfo | null;
  readiness: ReadinessIndicatorData | undefined;
  attentionReasons: AttentionReason[];
  pendingQuestions: PendingQuestion[] | undefined;
  agentRunning: boolean;
  isStepperCompleted: boolean;
  transitionResult: StepperCompletionResult | undefined;
  /** Stable per-item callbacks. Identity is preserved for items whose
   *  kind/name and blocking info haven't changed. */
  callbacks: StableItemCallbacks;
  /** Per-row pending booleans — derived in the parent loop from primitive
   *  pending keys. Only the actively-mutating row sees these flip, so other
   *  rows preserve memo equality. */
  archivePending: boolean;
  finalizePending: boolean;
  workshopPending: boolean;
  statusChangePending: boolean;
  workshopLabel: string;
  runningLabel: string | undefined;
  handleStepperCompleted: (itemKey: string, item: BacklogItem, result: StepperCompletionResult) => void;
  onItemClick: (nodeId: string) => void;
}

const NOOP_TOGGLE_SELECTION = () => {};

const BacklogRow = memo(function BacklogRow({
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
  const handleClick = useCallback(() => onItemClick(nodeId), [nodeId, onItemClick]);
  const handleStepperCompletedForItem = useCallback(
    (result: StepperCompletionResult) => handleStepperCompleted(itemKey, item, result),
    [handleStepperCompleted, itemKey, item],
  );

  return (
    <button
      type="button"
      onClick={handleClick}
      className="group w-full rounded-lg border border-slate-800/80 bg-slate-900/50 p-2.5 text-left transition-colors hover:border-slate-700/80 hover:bg-slate-800/60"
      data-testid="sidebar-backlog-item"
    >
      <BacklogCard
        item={item}
        readinessData={readiness}
        itemActions={itemActions}
        attentionReasons={attentionReasons}
        pendingQuestions={pendingQuestions}
        isStepperCompleted={isStepperCompleted}
        transitionResult={transitionResult}
        onStepperCompleted={handleStepperCompletedForItem}
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
      />
    </button>
  );
});
