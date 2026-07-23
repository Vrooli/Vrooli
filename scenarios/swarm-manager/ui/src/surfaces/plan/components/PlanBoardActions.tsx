/**
 * PlanBoardActions — the board's action layer, re-hosting the Command
 * Post's proven wiring (D9): `useCommandPostItemActions` drives per-card
 * run/archive/status mutations, the RunSheet and decision drawer
 * DecisionStreamView. Cards reach the layer through PlanCardActionsContext.
 */

import { useCallback, useMemo, useState, type ReactNode } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { backlogDetailPath } from "../../../app/routes/route-paths";
import { RunSheet, type RunSheetTarget } from "../../../components/backlog/run-sheet";
import { ConfirmDialog } from "../../../components/ui/confirm-dialog";
import { useCommandPostItemActions } from "../../../hooks/useCommandPostItemActions";
import { useBacklogStore } from "../../../stores/backlog-store";
import { useSnoozeStore } from "../../../stores/snooze-store";
import { getPresetExpiry } from "../../../lib/snooze-utils";
import type { BacklogItem } from "../../../types";
import { cardSnoozeKey } from "../lib/plan-presentation";
import { PlanCardActionsContext, type PlanCardActions } from "./plan-card-actions-context";
import { DecisionDrawer } from "./DecisionDrawer";

/** Bulk actions spawning more than this many agents require confirmation. */
const BULK_AGENT_CONFIRM_THRESHOLD = 3;

export interface PlanBoardActionsProps {
  children: ReactNode;
  /** Refetch the board projection (after decision-stream completion). */
  onBoardRefresh: () => void;
}

export function PlanBoardActions({ children, onBoardRefresh }: PlanBoardActionsProps) {
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const backlogItems = useBacklogStore((s) => s.items);
  const snooze = useSnoozeStore((s) => s.snooze);

  const [runModalTargets, setRunModalTargets] = useState<RunSheetTarget[] | undefined>();
  const [pendingBulkTargets, setPendingBulkTargets] = useState<RunSheetTarget[] | null>(null);
  const drawerOpen = searchParams.get("drawer") === "decisions";
  const decisionScope = searchParams.get("decisionScope");
  const decisionQuestionId = searchParams.get("decisionQuestion");

  const itemActions = useCommandPostItemActions({
    onSelectBacklog: (kind, name) => navigate(backlogDetailPath(kind, name)),
    onRunItem: (kind, name, title) => setRunModalTargets([{ kind, name, title }]),
  });

  const itemsByKey = useMemo(() => {
    const map = new Map<string, BacklogItem>();
    for (const item of backlogItems) {
      map.set(`${item.kind}/${item.name}`, item);
    }
    return map;
  }, [backlogItems]);

  const setDrawerParam = useCallback(
    (open: boolean) => {
      setSearchParams((prev) => {
        const next = new URLSearchParams(prev);
        if (open) next.set("drawer", "decisions");
        else {
          next.delete("drawer");
          next.delete("decisionScope");
          next.delete("decisionQuestion");
        }
        return next;
      }, { replace: true });
    },
    [setSearchParams],
  );

  const openDecisions = useCallback(
    (scopeItemKey?: string) => {
      setSearchParams((prev) => {
        const next = new URLSearchParams(prev);
        next.set("drawer", "decisions");
        if (scopeItemKey) next.set("decisionScope", scopeItemKey);
        else next.delete("decisionScope");
        next.delete("decisionQuestion");
        return next;
      }, { replace: true });
    }, [setSearchParams],
  );

  const closeDecisions = useCallback(() => {
    setDrawerParam(false);
  }, [setDrawerParam]);

  const runTargets = useCallback((targets: RunSheetTarget[]) => {
    if (targets.length === 0) return;
    if (targets.length > BULK_AGENT_CONFIRM_THRESHOLD) {
      setPendingBulkTargets(targets);
    } else {
      setRunModalTargets(targets);
    }
  }, []);

  const value = useMemo<PlanCardActions>(
    () => ({
      getCallbacks: (kind, name) => {
        const item = itemsByKey.get(`${kind}/${name}`);
        return item ? itemActions.getItemCallbacks(item) : undefined;
      },
      getBacklogItem: (kind, name) => itemsByKey.get(`${kind}/${name}`),
      runTargets,
      snoozeCard: (card, preset) => {
        const key = cardSnoozeKey(card);
        if (key) snooze(key, getPresetExpiry(preset));
      },
      openDecisions,
    }),
    [itemsByKey, itemActions, runTargets, snooze, openDecisions],
  );

  return (
    <PlanCardActionsContext.Provider value={value}>
      {children}

      <DecisionDrawer
        isOpen={drawerOpen}
        onClose={closeDecisions}
        scopeItemKey={decisionScope}
        currentQuestionId={decisionQuestionId}
        onCurrentQuestionChange={(questionId) => setSearchParams((prev) => {
          const next = new URLSearchParams(prev);
          if (questionId) next.set("decisionQuestion", questionId);
          else next.delete("decisionQuestion");
          return next;
        }, { replace: true })}
        onCompleted={onBoardRefresh}
      />

      {/* Bulk agent confirmation (threshold preserved from the Command Post). */}
      <ConfirmDialog
        isOpen={!!pendingBulkTargets}
        onClose={() => setPendingBulkTargets(null)}
        onConfirm={() => {
          if (pendingBulkTargets) setRunModalTargets(pendingBulkTargets);
          setPendingBulkTargets(null);
        }}
        title={`Run all ready (${pendingBulkTargets?.length ?? 0} items)`}
        description={`This will spawn ${pendingBulkTargets?.length ?? 0} agent sessions. Are you sure you want to proceed?`}
        confirmLabel="Proceed"
      />

      <RunSheet
        isOpen={!!runModalTargets}
        onClose={() => setRunModalTargets(undefined)}
        targets={runModalTargets}
        onSuccess={() => setRunModalTargets(undefined)}
      />
    </PlanCardActionsContext.Provider>
  );
}
