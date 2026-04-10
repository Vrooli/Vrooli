/**
 * useCommandPostItemActions — Shared mutation and action wiring for BacklogCard.
 *
 * Encapsulates workshop/archive/status mutations, active run tracking,
 * stepper completion state, and summary data derivation. Used by both
 * the sidebar BacklogTab and the Command Post SummaryView.
 */

import { useCallback, useMemo, useRef, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useAgentActivitiesStore, useBacklogStore } from "../stores";
import { backlogService } from "../services";
import { defaultApiClient } from "../lib/api-client";
import { API_ENDPOINTS } from "../lib/api-endpoints";
import { getAttentionReasons } from "../lib/feed";
import { buildReadinessData } from "../lib/maturity";
import type { AttentionReason, FeedbackItem, MaturityItem } from "../lib/feed";
import type { ReadinessIndicatorData } from "../lib/maturity";
import type { StepperCompletionResult } from "../components/backlog/inline-question-stepper";
import type { BacklogItem, BacklogKind, BacklogStatus, PendingQuestion } from "../types";

const ACTIVE_REFRESH_MS = 6000;

export interface ItemCallbacks {
  onRun: () => void;
  onArchive: () => void;
  onFollowUp: () => void;
  onFinalize: () => void;
  onWorkshop: () => void;
  archivePending: boolean;
  finalizePending: boolean;
  workshopPending: boolean;
  workshopLabel: string;
  runningLabel?: string;
  onStatusChange: (newStatus: BacklogStatus) => void;
  statusChangePending: boolean;
}

export interface UseCommandPostItemActionsResult {
  /** Per-item callback factory. */
  getItemCallbacks: (item: BacklogItem) => ItemCallbacks;
  /** Active run tracking for agent-running badge logic. */
  activeRunKeys: Set<string>;
  /** Summary-derived maps for BacklogCard props. */
  readinessMap: Map<string, ReadinessIndicatorData>;
  pendingQuestionsMap: Map<string, PendingQuestion[]>;
  attentionReasonsMap: Map<string, AttentionReason[]>;
  /** Stepper completion tracking. */
  completedSteppers: Set<string>;
  transitionItems: Map<string, StepperCompletionResult>;
  handleStepperCompleted: (itemKey: string, item: BacklogItem, result: StepperCompletionResult) => void;
  /** Workshop blocking confirmation state. */
  workshopBlockingConfirm: {
    kind: BacklogKind;
    name: string;
    mode: "workshop" | "finalize";
    blockingDepKeys: string[];
  } | null;
  setWorkshopBlockingConfirm: React.Dispatch<React.SetStateAction<UseCommandPostItemActionsResult["workshopBlockingConfirm"]>>;
  /** Execute the confirmed workshop override. */
  confirmWorkshopOverride: () => void;
  /** Mutation objects for pending checks. */
  workshopMutation: { isPending: boolean; variables?: { kind: string; name: string; mode: string } };
  archiveMutation: { isPending: boolean };
}

export interface UseCommandPostItemActionsOptions {
  /** Called when the user selects a backlog item for detail view. */
  onSelectBacklog?: (kind: string, name: string) => void;
  /** Called when the user wants to run an item (opens RunBacklogModal). */
  onRunItem?: (kind: BacklogKind, name: string, title?: string) => void;
}

export function useCommandPostItemActions(
  options: UseCommandPostItemActionsOptions = {},
): UseCommandPostItemActionsResult {
  const queryClient = useQueryClient();
  const items = useBacklogStore((s) => s.items);
  const blockingMap = useBacklogStore((s) => s.blockingMap);
  const fetchBacklog = useBacklogStore((s) => s.fetchBacklog);
  const agentActivities = useAgentActivitiesStore((s) => s.activities);
  const refreshActivities = useAgentActivitiesStore((s) => s.refreshActivities);

  const [completedSteppers, setCompletedSteppers] = useState<Set<string>>(new Set());
  const [transitionItems, setTransitionItems] = useState<Map<string, StepperCompletionResult>>(new Map());
  const [workshopBlockingConfirm, setWorkshopBlockingConfirm] = useState<UseCommandPostItemActionsResult["workshopBlockingConfirm"]>(null);

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

  const readinessMap = useMemo(() => {
    const map = new Map<string, ReadinessIndicatorData>();
    if (!summaryQuery.data?.maturity?.items) return map;
    for (const item of summaryQuery.data.maturity.items) {
      map.set(`${item.kind}/${item.name}`, buildReadinessData(item));
    }
    return map;
  }, [summaryQuery.data?.maturity]);

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

  const feedbackItems = useMemo<FeedbackItem[]>(
    () => (summaryQuery.data?.feedback?.items ?? []).map((item) => ({
      kind: item.kind,
      name: item.name,
      pendingDecisions: item.pending_decisions ?? 0,
    })),
    [summaryQuery.data?.feedback],
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
    mutationFn: ({ kind, name: itemName }: { kind: BacklogKind; name: string }) =>
      defaultApiClient.patch(API_ENDPOINTS.backlogArchiveItem(kind, itemName), {}),
    onSuccess: () => {
      void fetchBacklog({ force: true });
    },
  });

  const statusMutation = useMutation({
    mutationFn: ({ kind, name: itemName, newStatus }: { kind: BacklogKind; name: string; newStatus: BacklogStatus }) =>
      backlogService.update(kind, itemName, { status: newStatus }),
    onSuccess: () => {
      void fetchBacklog({ force: true });
    },
  });

  const workshopMutation = useMutation({
    mutationFn: ({ kind, name: itemName, mode, prompt, confirm, force }: {
      kind: BacklogKind;
      name: string;
      mode: "workshop" | "finalize";
      prompt: string;
      confirm?: boolean;
      force?: boolean;
    }) => backlogService.research(kind, itemName, { mode, prompt, confirm, force }),
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
    // If there's no auto-advance info, skip the transition — show normal action buttons immediately.
    if (!result.autoAdvance) {
      void queryClient.invalidateQueries({ queryKey: ["backlog-summary"] });
      return;
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

  const confirmWorkshopOverride = useCallback(() => {
    if (!workshopBlockingConfirm) return;
    const { kind, name: itemName, mode } = workshopBlockingConfirm;
    const prompt = mode === "finalize"
      ? "Finalize the latest workshop answers into the primary deliverable for this backlog item."
      : "Run the next workshop round for this backlog item.";
    setWorkshopBlockingConfirm(null);
    workshopMutation.mutate({ kind, name: itemName, mode, prompt, confirm: true, force: true });
  }, [workshopBlockingConfirm, workshopMutation]);

  // ── Per-item callback factory ────────────────────────────────────────
  const getItemCallbacks = useCallback((item: BacklogItem): ItemCallbacks => {
    const itemKey = `${item.kind}/${item.name}`;
    const readiness = readinessMap.get(itemKey);

    return {
      onRun: () => {
        options.onRunItem?.(item.kind as BacklogKind, item.name, item.title);
      },
      onArchive: () => {
        archiveMutation.mutate({ kind: item.kind as BacklogKind, name: item.name });
      },
      onFollowUp: () => {
        options.onSelectBacklog?.(item.kind, item.name);
      },
      onFinalize: () => {
        const info = blockingMap[itemKey];
        if (info?.blocked) {
          setWorkshopBlockingConfirm({
            kind: item.kind as BacklogKind,
            name: item.name,
            mode: "finalize",
            blockingDepKeys: info.blockingDepKeys,
          });
          return;
        }
        workshopMutation.mutate({
          kind: item.kind as BacklogKind,
          name: item.name,
          mode: "finalize",
          prompt: "Finalize the latest workshop answers into the primary deliverable for this backlog item.",
          confirm: true,
        });
      },
      onWorkshop: () => {
        const info = blockingMap[itemKey];
        if (info?.blocked) {
          setWorkshopBlockingConfirm({
            kind: item.kind as BacklogKind,
            name: item.name,
            mode: "workshop",
            blockingDepKeys: info.blockingDepKeys,
          });
          return;
        }
        workshopMutation.mutate({
          kind: item.kind as BacklogKind,
          name: item.name,
          mode: "workshop",
          prompt: "Run the next workshop round for this backlog item.",
          confirm: true,
        });
      },
      archivePending: archiveMutation.isPending,
      finalizePending:
        workshopMutation.isPending &&
        workshopMutation.variables?.kind === item.kind &&
        workshopMutation.variables?.name === item.name &&
        workshopMutation.variables?.mode === "finalize",
      workshopPending:
        workshopMutation.isPending &&
        workshopMutation.variables?.kind === item.kind &&
        workshopMutation.variables?.name === item.name &&
        workshopMutation.variables?.mode === "workshop",
      workshopLabel: (readiness?.roundsCompleted ?? 0) > 0 ? "Next Round" : "Workshop",
      runningLabel: activeRunLabels.get(itemKey),
      onStatusChange: (newStatus: BacklogStatus) => {
        statusMutation.mutate({ kind: item.kind as BacklogKind, name: item.name, newStatus });
      },
      statusChangePending:
        statusMutation.isPending &&
        statusMutation.variables?.kind === item.kind &&
        statusMutation.variables?.name === item.name,
    };
  }, [
    options, blockingMap, readinessMap, activeRunLabels,
    archiveMutation, workshopMutation, statusMutation,
  ]);

  return {
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
    workshopMutation: {
      isPending: workshopMutation.isPending,
      variables: workshopMutation.variables,
    },
    archiveMutation: {
      isPending: archiveMutation.isPending,
    },
  };
}
