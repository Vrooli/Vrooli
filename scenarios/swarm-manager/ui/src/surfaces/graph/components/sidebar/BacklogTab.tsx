/**
 * BacklogTab - Lists backlog items with rich action cards.
 *
 * Uses the shared BacklogCard component, providing inline
 * decision answering, run/workshop/finalize actions, and follow-up/archive.
 */

import { useCallback, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ListTodo } from "lucide-react";
import { useAgentActivitiesStore, useBacklogStore } from "../../../../stores";
import { useDetailSelectionStore } from "../../../../stores/detail-selection-store";
import { backlogService } from "../../../../services";
import { getItemActions } from "../../../../lib";
import { dependencyAwareSort } from "../../../../lib/dependency-sort";
import { buildReadinessData } from "../../../../lib/maturity";
import { getAttentionReasons } from "../../../../lib/feed";
import { buildBacklogNodeId } from "../../lib/node-id-parser";
import { matchesSearch } from "./useSidebarSearch";
import { BacklogCard } from "../../../../components/backlog/backlog-card";
import { RunBacklogModal } from "../../../../components/backlog/run-backlog-modal";
import type { RunBacklogTarget } from "../../../../components/backlog/run-backlog-modal";
import type { StepperCompletionResult } from "../../../../components/backlog/inline-question-stepper";
import type { ReadinessIndicatorData } from "../../../../lib/maturity";
import type { AttentionReason, FeedbackItem, MaturityItem } from "../../../../lib/feed";
import type { BacklogItem, BacklogKind, PendingQuestion } from "../../../../types";
import type { BacklogFilters, SortConfig } from "./types";

const ACTIVE_REFRESH_MS = 6000;

interface BacklogTabProps {
  searchQuery: string;
  filters: BacklogFilters;
  sort: SortConfig;
  onItemClick: (nodeId: string) => void;
}

function applyFilters(items: BacklogItem[], filters: BacklogFilters): BacklogItem[] {
  return items.filter((item) => {
    if (filters.statuses.length > 0 && !filters.statuses.includes(item.status)) return false;
    if (filters.kinds.length > 0 && !filters.kinds.includes(item.kind)) return false;
    if (filters.priorityMin !== null && item.priority < filters.priorityMin) return false;
    if (filters.priorityMax !== null && item.priority > filters.priorityMax) return false;
    return true;
  });
}

function applySort(items: BacklogItem[], sort: SortConfig, allItems: BacklogItem[]): BacklogItem[] {
  const dir = sort.direction === "asc" ? 1 : -1;
  const compareFn = (a: BacklogItem, b: BacklogItem): number => {
    switch (sort.field) {
      case "priority":
        return (a.priority - b.priority) * dir;
      case "recency":
        return (new Date(b.updated).getTime() - new Date(a.updated).getTime()) * dir;
      case "status":
        return a.status.localeCompare(b.status) * dir;
      case "alphabetical":
        return ((a.title || a.name).localeCompare(b.title || b.name)) * dir;
    }
  };
  // Dependency-aware sort: deps always appear before dependents.
  // allItems provides the full list for depth resolution when items is filtered.
  return dependencyAwareSort(items, compareFn, allItems);
}

function hasActiveFilters(filters: BacklogFilters): boolean {
  return filters.statuses.length > 0 || filters.kinds.length > 0 || filters.priorityMin !== null || filters.priorityMax !== null;
}

export function BacklogTab({ searchQuery, filters, sort, onItemClick }: BacklogTabProps) {
  const queryClient = useQueryClient();
  const items = useBacklogStore((s) => s.items);
  const fetchBacklog = useBacklogStore((s) => s.fetchBacklog);
  const agentActivities = useAgentActivitiesStore((s) => s.activities);
  const refreshActivities = useAgentActivitiesStore((s) => s.refreshActivities);
  const selectBacklog = useDetailSelectionStore((s) => s.selectBacklog);

  const [runModalTarget, setRunModalTarget] = useState<RunBacklogTarget | undefined>();
  const [completedSteppers, setCompletedSteppers] = useState<Set<string>>(new Set());
  const [transitionItems, setTransitionItems] = useState<Map<string, StepperCompletionResult>>(new Map());

  // ── Active run tracking ──────────────────────────────────────────────
  const activeRunKeys = useMemo(() => {
    const keys = new Set<string>();
    for (const activity of agentActivities) {
      if (activity.ownerType === "backlog" && activity.ownerKind && activity.ownerName) {
        keys.add(`${activity.ownerKind}/${activity.ownerName}`);
      }
    }
    return keys;
  }, [agentActivities]);

  const activeRunLabels = useMemo(() => {
    const labels = new Map<string, string>();
    for (const activity of agentActivities) {
      if (activity.ownerType === "backlog" && activity.ownerKind && activity.ownerName) {
        const key = `${activity.ownerKind}/${activity.ownerName}`;
        switch (activity.purpose) {
          case "workshop": labels.set(key, "Running workshop\u2026"); break;
          case "finalize": labels.set(key, "Running finalize\u2026"); break;
          case "research": labels.set(key, "Running research\u2026"); break;
          case "initialize": labels.set(key, "Initializing workshop\u2026"); break;
          case "process": labels.set(key, "Processing\u2026"); break;
          default: labels.set(key, "Agent running\u2026"); break;
        }
      }
    }
    return labels;
  }, [agentActivities]);

  // ── Summary query (readiness, pending questions, feedback) ───────────
  const summaryQuery = useQuery({
    queryKey: ["backlog-summary"],
    queryFn: () => backlogService.getBacklogSummary(),
    staleTime: 60_000,
    refetchInterval: activeRunKeys.size > 0 ? ACTIVE_REFRESH_MS : false,
  });

  const readinessMap = useMemo(() => {
    const map = new Map<string, ReadinessIndicatorData>();
    if (!summaryQuery.data?.maturity?.items) return map;
    for (const item of summaryQuery.data.maturity.items) {
      map.set(`${item.kind}/${item.name}`, buildReadinessData(item));
    }
    return map;
  }, [summaryQuery.data?.maturity]);

  const pendingQuestionsMap = useMemo(() => {
    const map = new Map<string, PendingQuestion[]>();
    if (!summaryQuery.data?.pending_questions?.items) return map;
    for (const pqi of summaryQuery.data.pending_questions.items) {
      map.set(`${pqi.kind}/${pqi.name}`, pqi.questions);
    }
    return map;
  }, [summaryQuery.data?.pending_questions]);

  const feedbackSummary = summaryQuery.data?.feedback;
  const feedbackItems = useMemo<FeedbackItem[]>(
    () => (feedbackSummary?.items ?? []).map((item) => ({
      kind: item.kind,
      name: item.name,
      pendingDecisions: item.pending_decisions ?? 0,
    })),
    [feedbackSummary],
  );

  const maturityItems = useMemo<MaturityItem[]>(
    () => (summaryQuery.data?.maturity?.items ?? []).map((item) => ({
      kind: item.kind,
      name: item.name,
      ready: item.ready ?? false,
      pendingItems: item.pending_items ?? 0,
    })),
    [summaryQuery.data?.maturity],
  );

  const attentionReasonsMap = useMemo(() => {
    const fbMap = new Map(feedbackItems.map((f) => [`${f.kind}/${f.name}`, f]));
    const matMap = new Map(maturityItems.map((m) => [`${m.kind}/${m.name}`, m]));
    const map = new Map<string, AttentionReason[]>();
    for (const item of items) {
      const reasons = getAttentionReasons(item, fbMap, matMap);
      if (reasons.length > 0) map.set(`${item.kind}/${item.name}`, reasons);
    }
    return map;
  }, [items, feedbackItems, maturityItems]);

  // ── Mutations ────────────────────────────────────────────────────────
  const archiveMutation = useMutation({
    mutationFn: ({ kind, name: itemName, item }: { kind: BacklogKind; name: string; item: BacklogItem }) =>
      backlogService.update(kind, itemName, { ...item, status: "archived" }),
    onSuccess: () => {
      void fetchBacklog({ force: true });
    },
  });

  const workshopMutation = useMutation({
    mutationFn: ({ kind, name: itemName, mode, prompt }: {
      kind: BacklogKind;
      name: string;
      mode: "workshop" | "finalize";
      prompt: string;
    }) => backlogService.research(kind, itemName, { mode, prompt }),
    onSuccess: (result) => {
      if (result.runId) {
        void refreshActivities(true);
      }
      void queryClient.invalidateQueries({ queryKey: ["backlog-summary"] });
    },
  });

  const handleStepperCompleted = useCallback((itemKey: string, _item: BacklogItem, result: StepperCompletionResult) => {
    setCompletedSteppers((prev) => {
      const next = new Set(prev);
      next.add(itemKey);
      return next;
    });
    if (result.autoAdvance?.triggered && result.autoAdvance.runId) {
      void refreshActivities(true);
    }
    setTransitionItems((prev) => {
      const next = new Map(prev);
      next.set(itemKey, result);
      return next;
    });
    setTimeout(() => {
      setTransitionItems((prev) => {
        const next = new Map(prev);
        next.delete(itemKey);
        return next;
      });
      void queryClient.invalidateQueries({ queryKey: ["backlog-summary"] });
    }, 4000);
  }, [queryClient, refreshActivities]);

  // ── Filter, search, sort ─────────────────────────────────────────────
  let filtered = applyFilters(items, filters);
  if (searchQuery) {
    filtered = filtered.filter((item) =>
      matchesSearch(searchQuery, item.title, item.name, item.description, ...(item.tags ?? [])),
    );
  }
  const sorted = applySort(filtered, sort, items);

  if (sorted.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-12 text-slate-500">
        <ListTodo className="mb-2 h-8 w-8" />
        <p className="text-sm">{searchQuery || hasActiveFilters(filters) ? "No backlog items match your filters." : "No backlog items."}</p>
      </div>
    );
  }

  return (
    <>
      <div className="space-y-2">
        {sorted.map((item) => {
          const nodeId = buildBacklogNodeId(item.kind, item.name);
          const itemKey = `${item.kind}/${item.name}`;
          const readiness = readinessMap.get(itemKey);
          const reasons = attentionReasonsMap.get(itemKey) ?? [];

          return (
            <button
              key={nodeId}
              type="button"
              onClick={() => onItemClick(nodeId)}
              className="group w-full rounded-lg border border-slate-800/80 bg-slate-900/50 p-2.5 text-left transition-colors hover:border-slate-700/80 hover:bg-slate-800/60"
              data-testid="sidebar-backlog-item"
            >
              <BacklogCard
                item={item}
                allItems={items}
                readinessData={readiness}
                itemActions={getItemActions({
                  item,
                  allItems: items,
                  readinessReady: readiness ? readiness.ready : null,
                  pendingSynthesis: readiness?.pendingSynthesis ?? false,
                  agentRunning: activeRunKeys.has(itemKey),
                  hasPendingDecisions: (pendingQuestionsMap.get(itemKey)?.length ?? 0) > 0,
                  hasExecutionHistory: item.status === "completed" || item.status === "failed",
                })}
                attentionReasons={reasons}
                pendingQuestions={pendingQuestionsMap.get(itemKey)}
                isStepperCompleted={completedSteppers.has(itemKey)}
                transitionResult={transitionItems.get(itemKey)}
                onStepperCompleted={(result) => handleStepperCompleted(itemKey, item, result)}
                batchMode={false}
                isSelected={false}
                onToggleSelection={() => {}}
                onRun={() => setRunModalTarget({ kind: item.kind, name: item.name, title: item.title })}
                onArchive={() => archiveMutation.mutate({ kind: item.kind as BacklogKind, name: item.name, item })}
                onFollowUp={() => selectBacklog(item.kind, item.name)}
                onFinalize={() => workshopMutation.mutate({
                  kind: item.kind as BacklogKind,
                  name: item.name,
                  mode: "finalize",
                  prompt: "Finalize the latest workshop answers into the primary deliverable for this backlog item.",
                })}
                onWorkshop={() => workshopMutation.mutate({
                  kind: item.kind as BacklogKind,
                  name: item.name,
                  mode: "workshop",
                  prompt: "Run the next workshop round for this backlog item.",
                })}
                runningLabel={activeRunLabels.get(itemKey)}
                archivePending={archiveMutation.isPending}
                finalizePending={
                  workshopMutation.isPending &&
                  workshopMutation.variables?.kind === item.kind &&
                  workshopMutation.variables?.name === item.name &&
                  workshopMutation.variables?.mode === "finalize"
                }
                workshopPending={
                  workshopMutation.isPending &&
                  workshopMutation.variables?.kind === item.kind &&
                  workshopMutation.variables?.name === item.name &&
                  workshopMutation.variables?.mode === "workshop"
                }
                workshopLabel={(readiness?.roundsCompleted ?? 0) > 0 ? "Next Round" : "Workshop"}
              />
            </button>
          );
        })}
      </div>

      {/* Run modal */}
      <RunBacklogModal
        isOpen={!!runModalTarget}
        onClose={() => setRunModalTarget(undefined)}
        target={runModalTarget}
        onSuccess={() => {
          setRunModalTarget(undefined);
          void fetchBacklog({ force: true });
          void refreshActivities(true);
        }}
      />
    </>
  );
}
