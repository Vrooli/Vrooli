/**
 * PlanBoardActions — the board's action layer, re-hosting the Command
 * Post's proven wiring (D9): `useCommandPostItemActions` drives per-card
 * run/workshop/archive/status mutations, the existing RunBacklogModal and
 * confirm dialogs are invoked unchanged, and the decision drawer hosts
 * DecisionStreamView. Cards reach the layer through PlanCardActionsContext.
 */

import { useCallback, useMemo, useState, type ReactNode } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { backlogDetailPath } from "../../../app/routes/route-paths";
import { RunBacklogModal, type RunBacklogTarget } from "../../../components/backlog/run-backlog-modal";
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

  const [runModalTargets, setRunModalTargets] = useState<RunBacklogTarget[] | undefined>();
  const [pendingBulkTargets, setPendingBulkTargets] = useState<RunBacklogTarget[] | null>(null);
  const [decisionScope, setDecisionScope] = useState<string | null>(null);

  const drawerOpen = searchParams.get("drawer") === "decisions";

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
        else next.delete("drawer");
        return next;
      }, { replace: true });
    },
    [setSearchParams],
  );

  const openDecisions = useCallback(
    (scopeItemKey?: string) => {
      setDecisionScope(scopeItemKey ?? null);
      setDrawerParam(true);
    },
    [setDrawerParam],
  );

  const closeDecisions = useCallback(() => {
    setDecisionScope(null);
    setDrawerParam(false);
  }, [setDrawerParam]);

  const runTargets = useCallback((targets: RunBacklogTarget[]) => {
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

      {/* Workshop blocking override — "Dependencies Not Ready" guard. */}
      <ConfirmDialog
        isOpen={!!itemActions.workshopBlockingConfirm}
        onClose={() => itemActions.setWorkshopBlockingConfirm(null)}
        onConfirm={itemActions.confirmWorkshopOverride}
        title="Dependencies Not Ready"
        description={
          itemActions.workshopBlockingConfirm?.blockingDepKeys.length
            ? `This item is blocked by incomplete dependencies: ${itemActions.workshopBlockingConfirm.blockingDepKeys.join(", ")}. Do you want to proceed anyway?`
            : "This item has incomplete dependencies. Do you want to proceed anyway?"
        }
        confirmLabel="Override and Proceed"
      />

      <RunBacklogModal
        isOpen={!!runModalTargets}
        onClose={() => setRunModalTargets(undefined)}
        targets={runModalTargets}
        onSuccess={() => setRunModalTargets(undefined)}
      />
    </PlanCardActionsContext.Provider>
  );
}



