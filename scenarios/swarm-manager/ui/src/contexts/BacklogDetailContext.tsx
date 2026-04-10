/**
 * BacklogDetailContext
 *
 * Lightweight React context providing read-only computed values about the
 * current backlog detail item. Eliminates prop drilling for derived data
 * that multiple child components need (labels, flags, agent state).
 *
 * Mutations and UI state live elsewhere (useBacklogDetailData, backlog-detail-ui-store).
 */

import { createContext, useContext, useMemo, type ReactNode } from "react";
import type { BacklogKind } from "../types";
import type { BacklogItem } from "../types/domain";
import type { ItemActions } from "../lib/backlog-queue-utils";
import type { AgentActivityRecord } from "../stores/agent-activities-store";

export interface BacklogDetailContextValue {
  backlogKind: BacklogKind;
  name: string;
  item: BacklogItem | undefined;
  itemActions: ItemActions | null;
  isLocked: boolean;
  isTerminal: boolean;
  agentRunIsActive: boolean;
  latestAgentActivity: AgentActivityRecord | null;
  deliverableLabel: string;
  workshopActionLabel: string;
  agentRunningLabel: string;
  agentLabel: string;
  isWorkshopFinalized: boolean;
  workshopBlockedDeps: string[];
  isRunningAgent: boolean;
}

const BacklogDetailCtx = createContext<BacklogDetailContextValue | null>(null);

export function BacklogDetailProvider({
  value,
  children,
}: {
  value: BacklogDetailContextValue;
  children: ReactNode;
}) {
  // Memoize by value identity — callers should ensure stable references for
  // unchanged sub-fields to avoid unnecessary re-renders.
  // eslint-disable-next-line react-hooks/exhaustive-deps
  const memoized = useMemo(() => value, [
    value.backlogKind,
    value.name,
    value.item,
    value.itemActions,
    value.isLocked,
    value.isTerminal,
    value.agentRunIsActive,
    value.latestAgentActivity,
    value.deliverableLabel,
    value.workshopActionLabel,
    value.agentRunningLabel,
    value.agentLabel,
    value.isWorkshopFinalized,
    value.workshopBlockedDeps,
    value.isRunningAgent,
  ]);

  return (
    <BacklogDetailCtx.Provider value={memoized}>
      {children}
    </BacklogDetailCtx.Provider>
  );
}

// eslint-disable-next-line react-refresh/only-export-components
export function useBacklogDetail(): BacklogDetailContextValue {
  const ctx = useContext(BacklogDetailCtx);
  if (!ctx) {
    throw new Error("useBacklogDetail must be used within a BacklogDetailProvider");
  }
  return ctx;
}
