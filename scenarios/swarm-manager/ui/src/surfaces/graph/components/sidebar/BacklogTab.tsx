/** Backlog collection surface; BacklogCard remains scenario-owned content. */
import { Profiler, memo, useCallback, useMemo, useState, type MouseEvent } from "react";
import { useNavigate } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { Plus } from "lucide-react";
import { CollectionList } from "@vrooli/react-component-library/CollectionList/1";
import { onProfilerRender } from "../../../../lib/profiler";
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
import { Button } from "../../../../components/ui/button";
import { CollectionRow } from "../../../../components/ui/collection-row";
import { backlogService, autoFilerService } from "../../../../services";
import { backlogDetailPath } from "../../../../app/routes/route-paths";
import { nextActionDetailTab } from "../../../../lib/backlog-next-action";
import { SidebarEmptyState } from "./SidebarEmptyState";
import { shouldOpenBacklogRow } from "./backlog-row-interaction";
import { useCommandPostItemActions, type StableItemCallbacks } from "../../../../hooks/useCommandPostItemActions";
import type { BacklogItem, PendingQuestion } from "../../../../types";
import type { BacklogNextAction } from "../../../../services/backlog";
import type { BacklogFilters, SortConfig } from "./types";
import type { AttentionReason } from "../../../../lib/attention";
import type { StepperCompletionResult } from "../../../../components/backlog/inline-question-stepper";

interface BacklogTabProps {
  searchQuery: string;
  filters: BacklogFilters;
  sort: SortConfig;
  onItemClick: (nodeId: string) => void;
  onClearSearch?: () => void;
  onCreateBacklog?: () => void;
  onCreateFromPlan?: () => void;
}

const EMPTY_REASONS: AttentionReason[] = [];
const NO_NEXT_ACTIONS: Record<string, BacklogNextAction> = {};
const itemKey = (item: BacklogItem) => `${item.kind}/${item.name}`;

function applyFilters(items: BacklogItem[], filters: BacklogFilters) {
  return items.filter(
    (item) =>
      (filters.showArchived || item.archivedAt == null) &&
      (filters.statuses.length === 0 || filters.statuses.includes(item.status)) &&
      (filters.kinds.length === 0 || filters.kinds.includes(item.kind)) &&
      (filters.priorityMin === null || item.priority >= filters.priorityMin) &&
      (filters.priorityMax === null || item.priority <= filters.priorityMax),
  );
}

function BacklogTabImpl({
  searchQuery,
  filters,
  sort,
  onItemClick,
  onClearSearch,
  onCreateBacklog,
  onCreateFromPlan,
}: BacklogTabProps) {
  const navigate = useNavigate();
  const items = useBacklogStore((s) => s.items);
  const fetchBacklog = useBacklogStore((s) => s.fetchBacklog);
  const snoozedKeys = useSnoozedKeys();
  const [runModalTarget, setRunModalTarget] = useState<RunSheetTarget>();
  const [pendingDismissKey, setPendingDismissKey] = useState<string | null>(null);

  const unblockingMap = useMemo(() => computeUnblockingMap(items), [items]);
  const sorted = useMemo(
    () =>
      sortBacklogItems(
        filterSnoozed(
          applyFilters(items, filters).filter(
            (item) =>
              !searchQuery ||
              matchesSearch(searchQuery, item.title, item.name, item.description, ...(item.tags ?? [])),
          ),
          (item) => snoozeKeyForBacklog(item.kind, item.name),
          snoozedKeys,
        ),
        buildBacklogCompareFn(sort, unblockingMap),
        items,
      ),
    [filters, items, searchQuery, snoozedKeys, sort, unblockingMap],
  );

  const handleSelect = useCallback(
    (kind: string, name: string) => navigate(backlogDetailPath(kind as BacklogItem["kind"], name)),
    [navigate],
  );
  const handleRun = useCallback(
    (kind: BacklogItem["kind"], name: string, title?: string) => setRunModalTarget({ kind, name, title: title ?? "" }),
    [],
  );
  const callbacks = useCommandPostItemActions({ onSelectBacklog: handleSelect, onRunItem: handleRun });
  // Keyed on the whole backlog rather than the filtered view, so a search or
  // filter change reuses the answer instead of refetching it.
  const itemKeys = useMemo(() => items.map(itemKey), [items]);
  const { data: nextActions = NO_NEXT_ACTIONS } = useQuery({
    queryKey: ["backlog", "next-actions", itemKeys],
    queryFn: () => backlogService.getNextActions(items),
    enabled: items.length > 0,
  });

  const dismiss = useCallback(
    async (item: BacklogItem) => {
      setPendingDismissKey(itemKey(item));
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
    return (
      <SidebarEmptyState
        icon={SIDEBAR_TAB_ICONS.backlog}
        title={searchQuery ? "No backlog items match your filters." : "No backlog items yet."}
        hint="Capture an idea or chore to get started."
        query={searchQuery}
        onClearSearch={onClearSearch}
        action={
          onCreateBacklog || onCreateFromPlan ? (
            <div className="mt-1 flex gap-2">
              {onCreateBacklog ? (
                <Button size="sm" data-testid="backlog-tab-create-item" onClick={onCreateBacklog}>
                  <Plus className="mr-1.5 h-3.5 w-3.5" />
                  Create item
                </Button>
              ) : null}
              {onCreateFromPlan ? (
                <Button size="sm" variant="outline" onClick={onCreateFromPlan}>
                  Create from plan
                </Button>
              ) : null}
            </div>
          ) : undefined
        }
      />
    );
  }

  return (
    <>
      <CollectionList
        items={sorted}
        getKey={itemKey}
        label="Backlog"
        virtualize
        selection={{ mode: "none", enterOn: ["shortcut"] }}
        actions={[
          {
            id: "open",
            label: "Open",
            onSelect: ([item]) => {
              if (item) onItemClick(buildBacklogNodeId(item.kind, item.name));
            },
          },
          {
            id: "archive",
            label: "Archive",
            tone: "destructive",
            bulk: true,
            disabled: (item) => (item.archivedAt == null ? false : "Already archived"),
            onSelect: async (rows) => {
              for (const item of rows) await backlogService.archiveItem(item.kind, item.name);
            },
          },
        ]}
        renderItem={(item) => {
          const key = itemKey(item);
          return (
            <BacklogRow
              item={item}
              nextAction={nextActions[key]}
              callbacks={callbacks.getItemCallbacks(item)}
              attentionReasons={callbacks.attentionReasonsMap.get(key) ?? EMPTY_REASONS}
              pendingQuestions={callbacks.pendingQuestionsMap.get(key)}
              agentRunning={callbacks.activeRunKeys.has(key)}
              isStepperCompleted={callbacks.completedSteppers.has(key)}
              archivePending={callbacks.pendingArchiveKey === key}
              dismissPending={pendingDismissKey === key}
              statusChangePending={callbacks.pendingStatusKey === key}
              runningLabel={callbacks.activeRunLabels.get(key)}
              handleStepperCompleted={callbacks.handleStepperCompleted}
              onDismiss={dismiss}
              onItemClick={onItemClick}
            />
          );
        }}
      />
      <RunSheet
        isOpen={!!runModalTarget}
        onClose={() => setRunModalTarget(undefined)}
        target={runModalTarget}
        onSuccess={() => {
          setRunModalTarget(undefined);
          void fetchBacklog({ force: true });
        }}
      />
    </>
  );
}

interface BacklogRowProps {
  item: BacklogItem;
  nextAction?: BacklogNextAction;
  attentionReasons: AttentionReason[];
  pendingQuestions?: PendingQuestion[];
  agentRunning: boolean;
  isStepperCompleted: boolean;
  callbacks: StableItemCallbacks;
  archivePending: boolean;
  dismissPending: boolean;
  statusChangePending: boolean;
  runningLabel?: string;
  handleStepperCompleted: (key: string, item: BacklogItem, result: StepperCompletionResult) => void;
  onDismiss: (item: BacklogItem) => Promise<void>;
  onItemClick: (id: string) => void;
}

// Memoized with stable handlers so a poll that changes one row, or none,
// re-renders only that row rather than every visible card.
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
  runningLabel,
  handleStepperCompleted,
  onDismiss,
  onItemClick,
}: BacklogRowProps) {
  const navigate = useNavigate();
  const itemActions = useMemo(
    () => itemActionsFromNextAction(item, nextAction, { agentRunning }),
    [agentRunning, item, nextAction],
  );
  const handleNext = useCallback(() => {
    if (nextAction?.id === "run") return callbacks.onRun();
    if (nextAction?.id === "archive") return callbacks.onArchive();
    if (nextAction?.id === "accept_suggestion") return callbacks.onStatusChange("backlog");
    if (nextAction?.id === "retry") {
      void backlogService
        .retry(item.kind, item.name)
        .then(() => useBacklogStore.getState().fetchBacklog({ force: true }));
      return;
    }
    const tab = nextAction && nextActionDetailTab(nextAction);
    navigate(backlogDetailPath(item.kind, item.name, tab ? { tab } : undefined));
  }, [callbacks, item.kind, item.name, navigate, nextAction]);
  const handleStepperDone = useCallback(
    (result: StepperCompletionResult) => handleStepperCompleted(itemKey(item), item, result),
    [handleStepperCompleted, item],
  );
  const acceptSuggestion = useCallback(() => callbacks.onStatusChange("backlog"), [callbacks]);
  const dismissSuggestion = useCallback(() => void onDismiss(item), [item, onDismiss]);

  // BacklogCard contains its own controls (next action, status chip, stepper),
  // so the row opens only when the click lands on content rather than one of
  // them. Keyboard opening is the list's job: Enter runs the "open" action.
  const handleClick = (event: MouseEvent<HTMLDivElement>) => {
    if (shouldOpenBacklogRow(event.target)) onItemClick(buildBacklogNodeId(item.kind, item.name));
  };

  return (
    <CollectionRow className="group" data-testid="sidebar-backlog-item" onClick={handleClick}>
      <BacklogCard
        item={item}
        nextAction={nextAction}
        itemActions={itemActions}
        attentionReasons={attentionReasons}
        pendingQuestions={pendingQuestions}
        isStepperCompleted={isStepperCompleted}
        onStepperCompleted={handleStepperDone}
        onRun={callbacks.onRun}
        onNextAction={handleNext}
        onArchive={callbacks.onArchive}
        onFollowUp={callbacks.onFollowUp}
        onAcceptSuggestion={acceptSuggestion}
        onDismissSuggestion={dismissSuggestion}
        onStatusChange={callbacks.onStatusChange}
        archivePending={archivePending}
        dismissPending={dismissPending}
        statusChangePending={statusChangePending}
        runningLabel={runningLabel}
      />
    </CollectionRow>
  );
});

export const BacklogTab = memo(function BacklogTab(props: BacklogTabProps) {
  return (
    <Profiler id="BacklogTab" onRender={onProfilerRender}>
      <BacklogTabImpl {...props} />
    </Profiler>
  );
});
