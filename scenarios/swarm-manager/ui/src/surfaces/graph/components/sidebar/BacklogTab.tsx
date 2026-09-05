/**
 * BacklogTab - Lists backlog items with rich action cards.
 *
 * Uses the shared BacklogCard component, providing inline
 * decision answering, run actions, and follow-up/archive.
 */

import { Profiler, memo, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useVirtualizer } from "@tanstack/react-virtual";
import { useQuery } from "@tanstack/react-query";
import { onProfilerRender } from "../../../../lib/profiler";
import { Download, Loader2, Plus } from "lucide-react";
import { SIDEBAR_TAB_ICONS } from "../../../../types/constants";
import { useBacklogStore } from "../../../../stores";
import { useSnoozedKeys } from "../../../../stores/snooze-store";
import { itemActionsFromNextAction } from "../../../../lib";
import { buildBacklogCompareFn, sortBacklogItems } from "../../../../lib/backlog-sort";
import { computeUnblockingMap } from "../../../../lib/dependency-sort";
import { filterSnoozed, snoozeKeyForBacklog } from "../../../../lib/snooze-utils";
import { buildBacklogNodeId } from "../../lib/node-id-parser";
import { matchesSearch } from "./useSidebarSearch";
import { BacklogCard } from "../../../../components/backlog/backlog-card";
import { RunSheet, type RunSheetTarget } from "../../../../components/backlog/run-sheet";
import { ConfirmDialog } from "../../../../components/ui/confirm-dialog";
import { Button } from "../../../../components/ui/button";
import { ContextMenu } from "../../../../components/ui/context-menu";
import { useContextMenu } from "../../../../components/ui/use-context-menu";
import type { ActionMenuItem } from "../../../../components/ui/action-menu";
import { backlogGoalTarget } from "../../../../components/goals/goal-target";
import { useSetAsGoalMenu } from "../../../../components/goals/useSetAsGoalMenu";
import { useCommandPostItemActions } from "../../../../hooks/useCommandPostItemActions";
import type { StableItemCallbacks } from "../../../../hooks/useCommandPostItemActions";
import type { BacklogItem, ItemBlockingInfo, PendingQuestion } from "../../../../types";
import type { BacklogNextAction } from "../../../../services/backlog";
import type { BacklogFilters, SortConfig } from "./types";
import type { AttentionReason } from "../../../../lib/attention";
import type { StepperCompletionResult } from "../../../../components/backlog/inline-question-stepper";
import { backlogDetailPath } from "../../../../app/routes/route-paths";
import { nextActionDetailTab } from "../../../../lib/backlog-next-action";
import { SidebarEmptyState } from "./SidebarEmptyState";
import { autoFilerService, backlogService } from "../../../../services";
import { runBulkAction, summarizeBulkOutcomes, failedOutcomeIds, type BulkOutcome } from "./bulk-actions";

interface BacklogTabProps {
  searchQuery: string;
  filters: BacklogFilters;
  sort: SortConfig;
  onItemClick: (nodeId: string) => void;
  onClearSearch?: () => void;
  selectionMode?: boolean;
  selectedIds?: Set<string>;
  onToggleSelection?: (id: string) => void;
  onVisibleIdsChange?: (ids: string[]) => void;
  onCreateBacklog?: () => void;
  onCreateFromPlan?: () => void;
}

function applyFilters(items: BacklogItem[], filters: BacklogFilters): BacklogItem[] {
  return items.filter((item) => {
    // Hide archived items unless showArchived is on
    if (item.archivedAt != null && !filters.showArchived) return false;
    if (filters.statuses.length > 0 && !filters.statuses.includes(item.status)) return false;
    if (filters.kinds.length > 0 && !filters.kinds.includes(item.kind)) return false;
    if (filters.priorityMin !== null && item.priority < filters.priorityMin) return false;
    if (filters.priorityMax !== null && item.priority > filters.priorityMax) return false;
    return true;
  });
}

function hasActiveFilters(filters: BacklogFilters): boolean {
  return filters.statuses.length > 0 || filters.kinds.length > 0 || filters.priorityMin !== null || filters.priorityMax !== null || filters.showArchived;
}

function backlogSelectionId(item: BacklogItem): string {
  return `backlog:${item.kind}/${item.name}`;
}

function BacklogTabImpl({
  searchQuery,
  filters,
  sort,
  onItemClick,
  onClearSearch,
  selectionMode = false,
  selectedIds = new Set<string>(),
  onToggleSelection,
  onVisibleIdsChange,
  onCreateBacklog,
  onCreateFromPlan,
}: BacklogTabProps) {
  const navigate = useNavigate();
  const items = useBacklogStore((s) => s.items);
  const blockingMap = useBacklogStore((s) => s.blockingMap);
  const fetchBacklog = useBacklogStore((s) => s.fetchBacklog);
  const snoozedKeys = useSnoozedKeys();

  const [runModalTarget, setRunModalTarget] = useState<RunSheetTarget | undefined>();
  const [pendingDismissKey, setPendingDismissKey] = useState<string | null>(null);

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
    pendingQuestionsMap,
    attentionReasonsMap,
    completedSteppers,
    handleStepperCompleted,
    pendingArchiveKey,
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

  useEffect(() => {
    onVisibleIdsChange?.(sorted.map(backlogSelectionId));
  }, [onVisibleIdsChange, sorted]);

  const selectedItems = useMemo(
    () => sorted.filter((item) => selectedIds.has(backlogSelectionId(item))),
    [selectedIds, sorted],
  );

  const { data: nextActions = {} } = useQuery({
    queryKey: ["backlog", "next-actions", sorted.map((item) => `${item.kind}/${item.name}`)],
    queryFn: () => backlogService.getNextActions(sorted.map(({ kind, name }) => ({ kind, name }))),
    enabled: sorted.length > 0,
  });

  const handleCloseRunModal = useCallback(() => setRunModalTarget(undefined), []);
  const handleRunModalSuccess = useCallback(() => {
    setRunModalTarget(undefined);
    void fetchBacklog({ force: true });
  }, [fetchBacklog]);
  const handleDismissSuggestion = useCallback(
    async (item: BacklogItem) => {
      const key = `${item.kind}/${item.name}`;
      setPendingDismissKey(key);
      try {
        await autoFilerService.dismissSuggestion(item.kind, item.name);
        await fetchBacklog({ force: true });
      } finally {
        setPendingDismissKey(null);
      }
    },
    [fetchBacklog],
  );
  if (sorted.length === 0) {
    const filtersActive = hasActiveFilters(filters);
    const title = filtersActive
      ? "No backlog items match your filters."
      : "No backlog items yet.";
    return (
      <SidebarEmptyState
        icon={SIDEBAR_TAB_ICONS.backlog}
        title={title}
        hint={filtersActive ? undefined : "Capture an idea or chore to get started."}
        query={searchQuery}
        onClearSearch={onClearSearch}
        action={
          !filtersActive && (onCreateBacklog || onCreateFromPlan) ? (
            <div className="mt-1 flex flex-wrap justify-center gap-2">
              {onCreateBacklog && (
                <Button
                  type="button"
                  size="sm"
                  onClick={onCreateBacklog}
                  data-testid="backlog-tab-create-item"
                >
                  <Plus className="mr-1.5 h-3.5 w-3.5" />
                  Create item
                </Button>
              )}
              {onCreateFromPlan && (
                <Button
                  type="button"
                  size="sm"
                  variant="outline"
                  onClick={onCreateFromPlan}
                  data-testid="backlog-tab-create-from-plan"
                >
                  Create from plan
                </Button>
              )}
            </div>
          ) : undefined
        }
      />
    );
  }

  return (
    <>
      {selectionMode && (
        <BacklogBulkActions
          selectedItems={selectedItems}
        />
      )}

      <Profiler id="VirtualizedBacklogList" onRender={onProfilerRender}>
        <VirtualizedBacklogList
          sorted={sorted}
          nextActions={nextActions}
          blockingMap={blockingMap}
          attentionReasonsMap={attentionReasonsMap}
          pendingQuestionsMap={pendingQuestionsMap}
          activeRunKeys={activeRunKeys}
          activeRunLabels={activeRunLabels}
          completedSteppers={completedSteppers}
          getItemCallbacks={getItemCallbacks}
          pendingArchiveKey={pendingArchiveKey}
          pendingStatusKey={pendingStatusKey}
          pendingDismissKey={pendingDismissKey}
          handleStepperCompleted={handleStepperCompleted}
          onDismissSuggestion={handleDismissSuggestion}
          onItemClick={onItemClick}
          selectionMode={selectionMode}
          selectedIds={selectedIds}
          onToggleSelection={onToggleSelection}
        />
      </Profiler>

      <RunSheet
        isOpen={!!runModalTarget}
        onClose={handleCloseRunModal}
        target={runModalTarget}
        onSuccess={handleRunModalSuccess}
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
  nextActions: Record<string, BacklogNextAction>;
  blockingMap: Record<string, ItemBlockingInfo>;
  attentionReasonsMap: Map<string, AttentionReason[]>;
  pendingQuestionsMap: Map<string, PendingQuestion[]>;
  activeRunKeys: Set<string>;
  activeRunLabels: Map<string, string>;
  completedSteppers: Set<string>;
  getItemCallbacks: (item: BacklogItem) => StableItemCallbacks;
  pendingArchiveKey: string | null;
  pendingStatusKey: string | null;
  pendingDismissKey: string | null;
  handleStepperCompleted: (itemKey: string, item: BacklogItem, result: StepperCompletionResult) => void;
  onDismissSuggestion: (item: BacklogItem) => void;
  onItemClick: (nodeId: string) => void;
  selectionMode: boolean;
  selectedIds: Set<string>;
  onToggleSelection?: (id: string) => void;
}

function VirtualizedBacklogList({
  sorted,
  nextActions,
  blockingMap,
  attentionReasonsMap,
  pendingQuestionsMap,
  activeRunKeys,
  activeRunLabels,
  completedSteppers,
  getItemCallbacks,
  pendingArchiveKey,
  pendingStatusKey,
  pendingDismissKey,
  handleStepperCompleted,
  onDismissSuggestion,
  onItemClick,
  selectionMode,
  selectedIds,
  onToggleSelection,
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
        const callbacks = getItemCallbacks(item);
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
              nextAction={nextActions[itemKey]}
              blockingInfo={blockingMap[itemKey] ?? null}
              attentionReasons={attentionReasonsMap.get(itemKey) ?? EMPTY_REASONS}
              pendingQuestions={pendingQuestionsMap.get(itemKey)}
              agentRunning={activeRunKeys.has(itemKey)}
              isStepperCompleted={completedSteppers.has(itemKey)}
              callbacks={callbacks}
              archivePending={pendingArchiveKey === itemKey}
              dismissPending={pendingDismissKey === itemKey}
              statusChangePending={pendingStatusKey === itemKey}
              runningLabel={activeRunLabels.get(itemKey)}
              handleStepperCompleted={handleStepperCompleted}
              onAcceptSuggestion={() => callbacks.onStatusChange("backlog")}
              onDismissSuggestion={() => onDismissSuggestion(item)}
              onItemClick={onItemClick}
              selectionMode={selectionMode}
              selected={selectedIds.has(backlogSelectionId(item))}
              onToggleSelection={onToggleSelection}
            />
          </div>
        );
      })}
    </div>
  );
}

interface BacklogRowProps {
  item: BacklogItem;
  nextAction?: BacklogNextAction;
  blockingInfo: ItemBlockingInfo | null;
  attentionReasons: AttentionReason[];
  pendingQuestions: PendingQuestion[] | undefined;
  agentRunning: boolean;
  isStepperCompleted: boolean;
  /** Stable per-item callbacks. Identity is preserved for items whose
   *  kind/name and blocking info haven't changed. */
  callbacks: StableItemCallbacks;
  /** Per-row pending booleans — derived in the parent loop from primitive
   *  pending keys. Only the actively-mutating row sees these flip, so other
   *  rows preserve memo equality. */
  archivePending: boolean;
  dismissPending: boolean;
  statusChangePending: boolean;
  onAcceptSuggestion: () => void;
  onDismissSuggestion: () => void;
  runningLabel: string | undefined;
  handleStepperCompleted: (itemKey: string, item: BacklogItem, result: StepperCompletionResult) => void;
  onItemClick: (nodeId: string) => void;
  selectionMode: boolean;
  selected: boolean;
  onToggleSelection?: (id: string) => void;
}

const BacklogRow = memo(function BacklogRow({
  item,
  nextAction,
  attentionReasons,
  pendingQuestions,
  agentRunning,
  isStepperCompleted,
  callbacks,
  archivePending,
  dismissPending,
  statusChangePending,
  onAcceptSuggestion,
  onDismissSuggestion,
  runningLabel,
  handleStepperCompleted,
  onItemClick,
  selectionMode,
  selected,
  onToggleSelection,
}: BacklogRowProps) {
  const navigate = useNavigate();
  const fetchBacklog = useBacklogStore((state) => state.fetchBacklog);
  const itemKey = `${item.kind}/${item.name}`;
  const selectionId = backlogSelectionId(item);
  const nodeId = useMemo(() => buildBacklogNodeId(item.kind, item.name), [item.kind, item.name]);
  const itemActions = useMemo(
    () =>
      itemActionsFromNextAction(
        item,
        nextAction,
        { agentRunning },
      ),
    [item, nextAction, agentRunning],
  );
  const handleClick = useCallback(() => onItemClick(nodeId), [nodeId, onItemClick]);
  const handleStepperCompletedForItem = useCallback(
    (result: StepperCompletionResult) => handleStepperCompleted(itemKey, item, result),
    [handleStepperCompleted, itemKey, item],
  );
  const handleNextAction = useCallback(() => {
    if (!nextAction) return;
    switch (nextAction.id) {
      case "run":
        callbacks.onRun();
        return;
      case "archive":
        callbacks.onArchive();
        return;
      case "accept_suggestion":
        callbacks.onStatusChange("backlog");
        return;
      case "retry":
        void backlogService.retry(item.kind, item.name).then(() => fetchBacklog({ force: true }));
        return;
      default: {
        const tab = nextActionDetailTab(nextAction);
        navigate(backlogDetailPath(item.kind, item.name, tab ? { tab } : undefined));
      }
    }
  }, [callbacks, fetchBacklog, item.kind, item.name, navigate, nextAction]);

  const contextMenu = useContextMenu();
  const goalTarget = useMemo(() => backlogGoalTarget(item), [item]);
  const setAsGoal = useSetAsGoalMenu(goalTarget);
  const contextItems = useMemo<ActionMenuItem[]>(() => {
    const items: ActionMenuItem[] = [
      { label: "Open", onSelect: () => onItemClick(nodeId), testId: "context-menu-open" },
    ];
    if (setAsGoal.item) items.push(setAsGoal.item);
    return items;
  }, [nodeId, onItemClick, setAsGoal.item]);

  return (
    <>
    <button
      type="button"
      onClick={handleClick}
      {...contextMenu.triggerProps}
      className="group w-full rounded-lg border border-slate-800/80 bg-slate-900/50 p-2.5 text-left transition-colors hover:border-slate-700/80 hover:bg-slate-800/60"
      data-testid="sidebar-backlog-item"
    >
      <BacklogCard
        item={item}
        nextAction={nextAction}
        itemActions={itemActions}
        attentionReasons={attentionReasons}
        pendingQuestions={pendingQuestions}
        isStepperCompleted={isStepperCompleted}
        onStepperCompleted={handleStepperCompletedForItem}
        batchMode={selectionMode}
        isSelected={selected}
        onToggleSelection={() => onToggleSelection?.(selectionId)}
        onRun={callbacks.onRun}
        onNextAction={handleNextAction}
        onArchive={callbacks.onArchive}
        onFollowUp={callbacks.onFollowUp}
        onAcceptSuggestion={onAcceptSuggestion}
        onDismissSuggestion={onDismissSuggestion}
        onStatusChange={callbacks.onStatusChange}
        archivePending={archivePending}
        dismissPending={dismissPending}
        statusChangePending={statusChangePending}
        runningLabel={runningLabel}
      />
    </button>
    <ContextMenu
      origin={contextMenu.origin}
      onClose={contextMenu.close}
      items={contextItems}
      testId="sidebar-backlog-context-menu"
    />
    {setAsGoal.dialog}
    </>
  );
});

type BacklogBulkAction =
  | "archive"
  | "unarchive"
  | "status"
  | "priority"
  | "assign-milestone"
  | "detach-milestone"
  | "add-tags"
  | "remove-tags"
  | "run"
  | "export";

const BACKLOG_STATUSES: BacklogItem["status"][] = [
  "backlog",
  "researching",
  "ready",
  "queued",
  "in_progress",
  "in_review",
  "review_pending",
  "completed",
  "failed",
  "needs_followup",
];

function BacklogBulkActions({
  selectedItems,
}: {
  selectedItems: BacklogItem[];
}) {
  const fetchBacklog = useBacklogStore((s) => s.fetchBacklog);
  const [action, setAction] = useState<BacklogBulkAction>("archive");
  const [status, setStatus] = useState<BacklogItem["status"]>("ready");
  const [priority, setPriority] = useState(5);
  const [milestone, setMilestone] = useState("");
  const [tags, setTags] = useState("");
  const [runMode, setRunMode] = useState<"manual" | "yolo">("manual");
  const [operation, setOperation] = useState<"generator" | "improver">("generator");
  const [pendingConfirm, setPendingConfirm] = useState<BacklogBulkAction | null>(null);
  const [running, setRunning] = useState(false);
  const [summary, setSummary] = useState<string | null>(null);
  const [outcomes, setOutcomes] = useState<BulkOutcome[]>([]);

  const eligibleItems = useMemo(() => {
    switch (action) {
      case "archive":
        return selectedItems.filter((item) => item.archivedAt == null);
      case "unarchive":
        return selectedItems.filter((item) => item.archivedAt != null);
      case "run":
        return selectedItems.filter((item) => item.status === "backlog" || item.status === "researching" || item.status === "ready");
      default:
        return selectedItems;
    }
  }, [action, selectedItems]);

  const skippedCount = selectedItems.length - eligibleItems.length;
  const tagList = tags.split(",").map((tag) => tag.trim()).filter(Boolean);

  const actionLabel = {
    archive: "Archive selected",
    unarchive: "Unarchive selected",
    status: "Change status",
    priority: "Set priority",
    "assign-milestone": "Assign milestone",
    "detach-milestone": "Detach milestone",
    "add-tags": "Add tags",
    "remove-tags": "Remove tags",
    run: "Run selected",
    export: "Export selected",
  }[action];

  const execute = useCallback(async () => {
    if (selectedItems.length === 0 || running) return;
    setRunning(true);
    setSummary(null);
    setOutcomes([]);
    try {
      if (action === "export") {
        const blob = await backlogService.exportItems({
          names: selectedItems.map((item) => item.name),
          kinds: Array.from(new Set(selectedItems.map((item) => item.kind))),
          includePrd: true,
          includeRequirements: true,
          includeClarifyQuestions: true,
          includeSuggestions: true,
          includeNotes: true,
        });
        const href = URL.createObjectURL(blob);
        const link = document.createElement("a");
        link.href = href;
        link.download = "swarm-manager-backlog-selection.zip";
        link.click();
        URL.revokeObjectURL(href);
        const next = selectedItems.map((item) => ({
          id: backlogSelectionId(item),
          label: item.title,
          status: "success" as const,
        }));
        setOutcomes(next);
        setSummary(summarizeBulkOutcomes(next));
        return;
      }

      const next = await runBulkAction(eligibleItems, {
        getId: backlogSelectionId,
        getLabel: (item) => item.title,
        run: (item) => {
          switch (action) {
            case "archive":
              return backlogService.archiveItem(item.kind, item.name);
            case "unarchive":
              return backlogService.unarchiveItem(item.kind, item.name);
            case "status":
              return backlogService.update(item.kind, item.name, { status });
            case "priority":
              return backlogService.update(item.kind, item.name, { priority });
            case "assign-milestone":
              return backlogService.update(item.kind, item.name, { milestone: milestone.trim() });
            case "detach-milestone":
              return backlogService.update(item.kind, item.name, { milestone: "" });
            case "add-tags":
              return backlogService.update(item.kind, item.name, { tags: Array.from(new Set([...(item.tags ?? []), ...tagList])) });
            case "remove-tags":
              return backlogService.update(item.kind, item.name, { tags: (item.tags ?? []).filter((tag) => !tagList.includes(tag)) });
            case "run":
              return backlogService.queue(item.kind, item.name, { mode: runMode, operation, confirm: true });
            default:
              return Promise.resolve();
          }
        },
      });
      const skipped: BulkOutcome[] = skippedCount > 0
        ? selectedItems
            .filter((item) => !eligibleItems.includes(item))
            .map((item) => ({
              id: backlogSelectionId(item),
              label: item.title,
              status: "skipped" as const,
              message: "Not eligible for this action",
            }))
        : [];
      const allOutcomes = [...next, ...skipped];
      setOutcomes(allOutcomes);
      setSummary(summarizeBulkOutcomes(allOutcomes));
      await fetchBacklog({ force: true });
    } finally {
      setRunning(false);
      setPendingConfirm(null);
    }
  }, [action, eligibleItems, fetchBacklog, milestone, operation, priority, runMode, running, selectedItems, skippedCount, status, tagList]);

  const needsText = action === "assign-milestone" || action === "add-tags" || action === "remove-tags";
  const disabled = selectedItems.length === 0
    || running
    || eligibleItems.length === 0
    || (needsText && (action === "assign-milestone" ? milestone.trim() === "" : tagList.length === 0));
  const requiresConfirm = action === "archive" || action === "unarchive" || action === "run";
  const failedIds = failedOutcomeIds(outcomes);

  return (
    <div className="mb-2 rounded-lg border border-slate-800 bg-slate-900/70 p-2" data-testid="sidebar-backlog-bulk-actions">
      <div className="flex flex-wrap items-center gap-2">
        <select
          value={action}
          onChange={(event) => setAction(event.target.value as BacklogBulkAction)}
          className="h-8 rounded border border-slate-700 bg-slate-950 px-2 text-xs text-slate-200"
          aria-label="Bulk action"
        >
          <option value="archive">Archive selected</option>
          <option value="unarchive">Unarchive selected</option>
          <option value="status">Change status</option>
          <option value="priority">Set priority</option>
          <option value="assign-milestone">Assign milestone</option>
          <option value="detach-milestone">Detach milestone</option>
          <option value="add-tags">Add tags</option>
          <option value="remove-tags">Remove tags</option>
          <option value="run">Run selected</option>
          <option value="export">Export selected</option>
        </select>
        {action === "status" && (
          <select value={status} onChange={(event) => setStatus(event.target.value as BacklogItem["status"])} className="h-8 rounded border border-slate-700 bg-slate-950 px-2 text-xs text-slate-200" aria-label="Status">
            {BACKLOG_STATUSES.map((option) => <option key={option} value={option}>{option.replace(/_/g, " ")}</option>)}
          </select>
        )}
        {action === "priority" && (
          <input type="number" min={1} max={10} value={priority} onChange={(event) => setPriority(Math.max(1, Math.min(10, Number(event.target.value || 1))))} className="h-8 w-16 rounded border border-slate-700 bg-slate-950 px-2 text-xs text-slate-200" aria-label="Priority" />
        )}
        {action === "assign-milestone" && (
          <input value={milestone} onChange={(event) => setMilestone(event.target.value)} placeholder="milestone-name" className="h-8 min-w-0 flex-1 rounded border border-slate-700 bg-slate-950 px-2 text-xs text-slate-200 placeholder:text-slate-500" aria-label="Milestone name" />
        )}
        {(action === "add-tags" || action === "remove-tags") && (
          <input value={tags} onChange={(event) => setTags(event.target.value)} placeholder="tags, comma separated" className="h-8 min-w-0 flex-1 rounded border border-slate-700 bg-slate-950 px-2 text-xs text-slate-200 placeholder:text-slate-500" aria-label="Tags" />
        )}
        {action === "run" && (
          <>
            <select value={runMode} onChange={(event) => setRunMode(event.target.value as "manual" | "yolo")} className="h-8 rounded border border-slate-700 bg-slate-950 px-2 text-xs text-slate-200" aria-label="Run mode">
              <option value="manual">Manual</option>
              <option value="yolo">YOLO</option>
            </select>
            <select value={operation} onChange={(event) => setOperation(event.target.value as "generator" | "improver")} className="h-8 rounded border border-slate-700 bg-slate-950 px-2 text-xs text-slate-200" aria-label="Operation">
              <option value="generator">Generator</option>
              <option value="improver">Improver</option>
            </select>
          </>
        )}
        <button
          type="button"
          disabled={disabled}
          onClick={() => {
            if (requiresConfirm) setPendingConfirm(action);
            else void execute();
          }}
          className="inline-flex h-8 items-center gap-1.5 rounded border border-cyan-500/40 bg-cyan-500/10 px-2 text-xs font-medium text-cyan-200 hover:bg-cyan-500/20 disabled:cursor-not-allowed disabled:opacity-50"
        >
          {running ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : action === "export" ? <Download className="h-3.5 w-3.5" /> : null}
          Apply
        </button>
      </div>
      <div className="mt-1.5 text-[11px] text-slate-500">
        {eligibleItems.length} eligible{skippedCount > 0 ? `, ${skippedCount} skipped` : ""}
        {summary ? ` - ${summary}` : ""}
      </div>
      {outcomes.some((outcome) => outcome.status === "failed") && (
        <div className="mt-1 max-h-20 overflow-y-auto text-[11px] text-red-300">
          {[...failedIds].map((id) => {
            const outcome = outcomes.find((entry) => entry.id === id);
            return outcome ? <div key={id}>{outcome.label}: {outcome.message}</div> : null;
          })}
        </div>
      )}
      <ConfirmDialog
        isOpen={pendingConfirm !== null}
        onClose={() => setPendingConfirm(null)}
        onConfirm={() => void execute()}
        title={actionLabel}
        description={`${actionLabel} will affect ${eligibleItems.length} selected backlog item${eligibleItems.length === 1 ? "" : "s"}.${skippedCount > 0 ? ` ${skippedCount} item${skippedCount === 1 ? "" : "s"} will be skipped.` : ""}`}
        confirmLabel={actionLabel}
        isLoading={running}
      />
    </div>
  );
}
