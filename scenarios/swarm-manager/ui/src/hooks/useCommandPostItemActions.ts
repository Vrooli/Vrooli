/**
 * useCommandPostItemActions — Shared mutation and action wiring for BacklogCard.
 *
 * Encapsulates archive/status mutations, active run tracking,
 * stepper completion state, and summary data derivation. Used by both
 * the sidebar BacklogTab and the Plan board card menus.
 */

import { useCallback, useMemo, useRef, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useActionMutation } from "./useActionMutation";
import { useAgentActivitiesStore, useBacklogStore } from "../stores";
import { backlogService } from "../services";
import { defaultApiClient } from "../lib/api-client";
import { API_ENDPOINTS } from "../lib/api-endpoints";
import { getAttentionReasons } from "../lib/attention";
import type { AttentionReason, FeedbackItem } from "../lib/attention";
import type { StepperCompletionResult } from "../components/backlog/inline-question-stepper";
import type { BacklogItem, BacklogKind, BacklogStatus, PendingQuestion } from "../types";

const ACTIVE_REFRESH_MS = 6000;

/**
 * The full action contract consumed by `BacklogCard`. We keep this interface
 * for BacklogCard's prop contract (it still receives all of these), but we
 * split how it's *produced*: stable function callbacks come from a memoized
 * per-item map, while the per-row pending booleans / labels are derived in
 * the parent loop from the primitive pending keys exposed by this hook.
 */
/**
 * Stable per-item callbacks. Identity is preserved across renders for items
 * whose `kind/name` hasn't changed, so consumers
 * (e.g. BacklogRow) can pass them as memoized props without breaking
 * `React.memo` on the card.
 */
export interface StableItemCallbacks {
  onRun: () => void;
  onArchive: () => void;
  onFollowUp: () => void;
  onStatusChange: (newStatus: BacklogStatus) => void;
}

export interface UseCommandPostItemActionsResult {
  /** Stable per-item callbacks. Identity is preserved when items / blocking
   *  haven't changed, so passing this through to memoized rows is safe. */
  getItemCallbacks: (item: BacklogItem) => StableItemCallbacks;
  /** Active run tracking for agent-running badge logic. */
  activeRunKeys: Set<string>;
  /** Per-item label shown when an agent is running, e.g. "Running workshop…". */
  activeRunLabels: Map<string, string>;
  /** Summary-derived maps for BacklogCard props. */
  pendingQuestionsMap: Map<string, PendingQuestion[]>;
  attentionReasonsMap: Map<string, AttentionReason[]>;
  /** Stepper completion tracking. */
  completedSteppers: Set<string>;
  handleStepperCompleted: (itemKey: string, item: BacklogItem, result: StepperCompletionResult) => void;
  archiveMutation: { isPending: boolean };
  /** Primitive pending keys — derive per-row booleans in the parent loop so
   *  only the actively-pending row re-renders, not the whole list. */
  pendingArchiveKey: string | null;
  pendingStatusKey: string | null;
}

export interface UseCommandPostItemActionsOptions {
  /** Called when the user selects a backlog item for detail view. */
  onSelectBacklog?: (kind: string, name: string) => void;
  /** Called when the user wants to run an item (opens RunSheet). */
  onRunItem?: (kind: BacklogKind, name: string, title?: string) => void;
}

export function useCommandPostItemActions(
  options: UseCommandPostItemActionsOptions = {},
): UseCommandPostItemActionsResult {
  const queryClient = useQueryClient();
  const items = useBacklogStore((s) => s.items);
  const fetchBacklog = useBacklogStore((s) => s.fetchBacklog);
  const agentActivities = useAgentActivitiesStore((s) => s.activities);

  const [completedSteppers, setCompletedSteppers] = useState<Set<string>>(new Set());

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

  // ── Summary query ───────────────────────────────────────────────────
  const summaryQuery = useQuery({
    queryKey: ["backlog-summary"],
    queryFn: () => backlogService.getBacklogSummary(),
    staleTime: 60_000,
    refetchInterval: activeRunKeys.size > 0 ? ACTIVE_REFRESH_MS : false,
  });

  // Snapshot ref: once an item's pending questions are first seen, preserve
  // them until the stepper completes.  This prevents background summary
  // refreshes from yanking the stepper out from under the user.
  const stableQuestionsRef = useRef<Map<string, PendingQuestion[]>>(new Map());

  const pendingQuestionsMap = useMemo(() => {
    const serverMap = new Map<string, PendingQuestion[]>();
    if (summaryQuery.data?.pending_questions?.items) {
      for (const pqi of summaryQuery.data.pending_questions.items) {
        serverMap.set(`${pqi.kind}/${pqi.name}`, pqi.questions);
      }
    }

    const stable = stableQuestionsRef.current;
    const result = new Map<string, PendingQuestion[]>();

    // Use server data when available; snapshot first-seen questions.
    for (const [key, questions] of serverMap) {
      result.set(key, questions);
      if (!stable.has(key)) {
        stable.set(key, questions);
      }
    }

    // Preserve snapshot for items the server dropped but stepper hasn't finished.
    for (const [key, questions] of stable) {
      if (!result.has(key) && !completedSteppers.has(key)) {
        result.set(key, questions);
      }
    }

    // Clean up completed items from the snapshot.
    for (const key of completedSteppers) {
      stable.delete(key);
    }

    return result;
  }, [summaryQuery.data?.pending_questions, completedSteppers]);

  const feedbackItems = useMemo<FeedbackItem[]>(() => [], []);

  const attentionReasonsMap = useMemo(() => {
    const fbMap = new Map(feedbackItems.map((f) => [`${f.kind}/${f.name}`, f]));
    const map = new Map<string, AttentionReason[]>();
    for (const item of items) {
      const reasons = getAttentionReasons(item, fbMap, new Map());
      if (reasons.length > 0) map.set(`${item.kind}/${item.name}`, reasons);
    }
    return map;
  }, [items, feedbackItems]);

  // ── Mutations ────────────────────────────────────────────────────────
  const archiveMutation = useActionMutation({
    mutationFn: ({ kind, name: itemName }: { kind: BacklogKind; name: string }) =>
      defaultApiClient.patch(API_ENDPOINTS.backlogArchiveItem(kind, itemName), {}),
    errorMessage: "Couldn't archive that item",
    successMessage: (_result, { kind, name: itemName }) => `Archived ${kind}/${itemName}`,
    source: "useCommandPostItemActions.archive",
    onSuccess: () => {
      void fetchBacklog({ force: true });
    },
  });

  const statusMutation = useActionMutation({
    mutationFn: ({ kind, name: itemName, newStatus }: { kind: BacklogKind; name: string; newStatus: BacklogStatus }) =>
      backlogService.update(kind, itemName, { status: newStatus }),
    errorMessage: "Couldn't change that item's status",
    successMessage: (_item, { name: itemName, newStatus }) =>
      `${itemName} set to ${newStatus.replaceAll("_", " ")}`,
    source: "useCommandPostItemActions.status",
    onSuccess: () => {
      void fetchBacklog({ force: true });
    },
  });

  const handleStepperCompleted = useCallback((itemKey: string, _item: BacklogItem, _result: StepperCompletionResult) => {
    setCompletedSteppers((prev) => {
      const next = new Set<string>(prev);
      next.add(itemKey);
      return next;
    });
    void queryClient.invalidateQueries({ queryKey: ["backlog-summary"] });
  }, [queryClient]);

  // ── Stable per-item callbacks ───────────────────────────────────────
  // Pull mutate functions out so the closure depends only on stable refs.
  // (TanStack Query v5 returns stable mutate functions across renders.)
  const archive = archiveMutation.mutate;
  const status = statusMutation.mutate;
  const onRunItem = options.onRunItem;
  const onSelectBacklog = options.onSelectBacklog;

  // Per-item callback map. Rebuilt only when items / option
  // callbacks change. For the common steady-state, this map's identity AND
  // each entry's identity are preserved across renders, so memoized rows
  // skip re-rendering on every parent update.
  const itemCallbackMap = useMemo<Map<string, StableItemCallbacks>>(() => {
    const map = new Map<string, StableItemCallbacks>();
    for (const item of items) {
      const itemKey = `${item.kind}/${item.name}`;
      map.set(itemKey, {
        onRun: () => onRunItem?.(item.kind, item.name, item.title),
        onArchive: () => archive({ kind: item.kind, name: item.name }),
        onFollowUp: () => onSelectBacklog?.(item.kind, item.name),
        onStatusChange: (newStatus: BacklogStatus) => {
          status({ kind: item.kind, name: item.name, newStatus });
        },
      });
    }
    return map;
  }, [items, archive, status, onRunItem, onSelectBacklog]);

  const getItemCallbacks = useCallback(
    (item: BacklogItem): StableItemCallbacks => {
      const itemKey = `${item.kind}/${item.name}`;
      const entry = itemCallbackMap.get(itemKey);
      if (entry) return entry;
      // Item not in the store yet (rare; e.g. optimistic UI). Return a
      // throwaway object — this row will re-render on the next items
      // refresh and pick up the stable entry.
      return {
        onRun: () => onRunItem?.(item.kind, item.name, item.title),
        onArchive: () => archive({ kind: item.kind, name: item.name }),
        onFollowUp: () => onSelectBacklog?.(item.kind, item.name),
        onStatusChange: (newStatus: BacklogStatus) => status({
          kind: item.kind, name: item.name, newStatus,
        }),
      };
    },
    [itemCallbackMap, archive, status, onRunItem, onSelectBacklog],
  );

  // ── Primitive pending keys ──────────────────────────────────────────
  // Per-row pending booleans get derived from these in the parent loop.
  // Only the actively-mutating row sees its boolean flip; all other rows
  // see false === false and skip re-render via memo.
  const pendingArchiveKey = archiveMutation.isPending && archiveMutation.variables
    ? `${archiveMutation.variables.kind}/${archiveMutation.variables.name}`
    : null;
  const pendingStatusKey = statusMutation.isPending && statusMutation.variables
    ? `${statusMutation.variables.kind}/${statusMutation.variables.name}`
    : null;

  return {
    getItemCallbacks,
    activeRunKeys,
    activeRunLabels,
    pendingQuestionsMap,
    attentionReasonsMap,
    completedSteppers,
    handleStepperCompleted,
    archiveMutation: {
      isPending: archiveMutation.isPending,
    },
    pendingArchiveKey,
    pendingStatusKey,
  };
}
