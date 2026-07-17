/**
 * useAgentOpsQueries — react-query hooks over the AgentOperationsService.
 *
 * The workflow-projection query is THE canonical source for a target's
 * workflow state, operation history, and legal actions. Polling follows the
 * existing "poll while active" idiom: 3s while any projected operation is in a
 * non-terminal state (mirroring the executionHistory validating poll), idle
 * otherwise.
 */

import { useQuery } from "@tanstack/react-query";
import { agentOperationsService } from "../services";
import { isWorkflowProjectionActive } from "../lib/agent-ops-utils";
import type { BacklogKind } from "../types";
import type { AgentOpsTarget, WorkflowProjection } from "../types/agent-operations";

// Re-export the target shape for hook consumers that build selectors inline.
export type { AgentOpsTarget } from "../types/agent-operations";

const ACTIVE_POLL_MS = 3_000;

/** Target selector for a backlog item ("kind/name" domain ref). */
export function backlogItemTarget(
  kind: BacklogKind | string | null | undefined,
  name: string | null | undefined,
): AgentOpsTarget | null {
  if (!kind || !name) return null;
  return { kind: "backlog-item", id: `${kind}/${name}` };
}

/** Target selector for an initiative. */
export function initiativeTarget(name: string | null | undefined): AgentOpsTarget | null {
  if (!name) return null;
  return { kind: "initiative", id: name };
}

export const agentOpsKeys = {
  projection: (target: AgentOpsTarget | null) =>
    ["agent-ops", "workflow-projection", target?.kind ?? "", target?.id ?? ""] as const,
  history: (target: AgentOpsTarget | null) =>
    ["agent-ops", "execution-history", target?.kind ?? "", target?.id ?? ""] as const,
  resolvedBindings: (target: AgentOpsTarget | null) =>
    ["agent-ops", "resolved-bindings", target?.kind ?? "", target?.id ?? ""] as const,
  /** Prefix key — invalidates resolved bindings for ALL targets (an initiative override changes item resolution too). */
  allResolvedBindings: ["agent-ops", "resolved-bindings"] as const,
  compatibleModes: (target: AgentOpsTarget | null) =>
    ["agent-ops", "compatible-modes", target?.kind ?? "", target?.id ?? ""] as const,
  bindingOverrides: (owner: AgentOpsTarget | null) =>
    ["agent-ops", "binding-overrides", owner?.kind ?? "", owner?.id ?? ""] as const,
  allBindingOverrides: ["agent-ops", "binding-overrides"] as const,
  migrationStatus: ["agent-ops", "migration-status"] as const,
};

/**
 * Canonical workflow projection for a target. `data.found === false` means no
 * workflow document exists (pre-migration legacy target) — callers fall back
 * to legacy client logic unchanged.
 */
export function useWorkflowProjectionQuery(target: AgentOpsTarget | null) {
  return useQuery({
    queryKey: agentOpsKeys.projection(target),
    queryFn: () => {
      if (!target) throw new Error("Agent-ops target is required");
      return agentOperationsService.getWorkflowProjection(target);
    },
    enabled: !!target,
    refetchInterval: (query) => {
      const data: WorkflowProjection | undefined = query.state.data;
      return isWorkflowProjectionActive(data) ? ACTIVE_POLL_MS : false;
    },
  });
}

/**
 * Canonical execution provenance history (newest first). Only fetched once a
 * workflow document is known to exist; polls alongside the projection while
 * the workflow is active.
 */
export function useExecutionHistoryQuery(
  target: AgentOpsTarget | null,
  options: { enabled: boolean; active: boolean },
) {
  return useQuery({
    queryKey: agentOpsKeys.history(target),
    queryFn: () => {
      if (!target) throw new Error("Agent-ops target is required");
      return agentOperationsService.listExecutionHistory(target);
    },
    enabled: !!target && options.enabled,
    refetchInterval: options.active ? ACTIVE_POLL_MS : false,
  });
}

/**
 * Per-operation resolved bindings (winning binding + contributions) for a
 * target. Fetch-on-demand — the slice-D override UI mounts on this.
 */
export function useResolvedBindingsQuery(target: AgentOpsTarget | null, enabled = true) {
  return useQuery({
    queryKey: agentOpsKeys.resolvedBindings(target),
    queryFn: () => {
      if (!target) throw new Error("Agent-ops target is required");
      return agentOperationsService.getResolvedBindings(target);
    },
    enabled: !!target && enabled,
  });
}

/**
 * Authored modes with server-computed per-operation compatibility verdicts
 * for a target. The override UI derives BOTH its dialog choices (only
 * verdict-compatible modes are offered) and its stale-revision display
 * (equality against the mode's current revision) from this — the server is
 * the only judge of compatibility.
 */
export function useCompatibleModesQuery(target: AgentOpsTarget | null, enabled = true) {
  return useQuery({
    queryKey: agentOpsKeys.compatibleModes(target),
    queryFn: () => {
      if (!target) throw new Error("Agent-ops target is required");
      return agentOperationsService.listCompatibleModes(target);
    },
    enabled: !!target && enabled,
  });
}

/**
 * Raw override documents stored at ONE owner's layer (provenance for the
 * "override set here" affordances: file revision, updated_at, reset action).
 */
export function useBindingOverridesQuery(owner: AgentOpsTarget | null, enabled = true) {
  return useQuery({
    queryKey: agentOpsKeys.bindingOverrides(owner),
    queryFn: () => {
      if (!owner) throw new Error("Agent-ops owner is required");
      return agentOperationsService.listBindingOverrides(owner);
    },
    enabled: !!owner && enabled,
  });
}

/**
 * Persisted-state migration status. Global (no target), changes rarely —
 * cached for a minute, never polled.
 */
export function useMigrationStatusQuery() {
  return useQuery({
    queryKey: agentOpsKeys.migrationStatus,
    queryFn: () => agentOperationsService.getMigrationStatus(),
    staleTime: 60_000,
  });
}
