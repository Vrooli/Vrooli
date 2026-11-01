import { useCallback, useEffect, useRef, useState, type Dispatch, type SetStateAction } from "react";
import { Code, ConnectError } from "@connectrpc/connect";
import type { AgentUpdateResponse } from "@vrooli/proto-types/web-console/v1/capabilities/capabilities_pb";
import { capabilitiesClient } from "../../api/capabilities";

export interface AgentUpdateCheck { id: string; passed: boolean; message: string }

export interface AgentUpdate {
  installed: string;
  latest: string;
  /** The version the permission policy was reviewed against, not an install target. */
  pinned: string;
  status: string;
  state: string;
  primaryAction: string;
  operatorCommand: string;
  reason: string;
  checks: AgentUpdateCheck[];
  installKind: string;
  rollbackVersion: string;
  newerThanReviewed: boolean;
  checkedAt: string;
  sourceKind: string;
  sourceId: string;
  probeVerified: boolean;
  note: string;
}

/** An action a harness's control plane can run. Retry drops the server's cache and looks again. */
export type AgentUpdateAction = "install" | "use_managed_copy" | "link_managed_copy" | "update" | "rollback" | "retry" | "refresh_catalog";

export type AgentActionRun =
  | { phase: "running"; action: AgentUpdateAction }
  | { phase: "failed"; action: AgentUpdateAction; message: string; unsupported: boolean; outcomeUnknown: boolean }
  | { phase: "done"; action: AgentUpdateAction; message: string; previous: string; current: string };

/**
 * A control-plane update can take minutes, and the server reports no progress,
 * so the call simply waits. The bound only keeps a dead connection from
 * spinning forever; the run itself belongs to the control plane. It stays above
 * the server's own deadline of 8m30s.
 */
const ACTION_TIMEOUT_MS = 20 * 60_000;
/** A tunnel can drop a long response. Repeating rejoins the running action. */
const TRANSPORT_ATTEMPTS = 3;
const TRANSPORT_CODES = new Set([Code.Unknown, Code.Unavailable, Code.DeadlineExceeded, Code.Canceled, Code.Internal]);

function decode(value: AgentUpdateResponse): AgentUpdate {
  return {
    installed: value.installed, latest: value.latest, pinned: value.pinned, status: value.status,
    state: value.state, primaryAction: value.primaryAction, operatorCommand: value.operatorCommand, reason: value.reason,
    checks: value.checks.map((check) => ({ id: check.id, passed: check.passed, message: check.message })),
    installKind: value.installKind, rollbackVersion: value.rollbackVersion, newerThanReviewed: value.newerThanReviewed,
    checkedAt: value.checkedAt, sourceKind: value.sourceKind, sourceId: value.sourceId, probeVerified: value.probeVerified, note: value.note,
  };
}

type ByAgent<T> = Record<string, T>;

function setFor<T>(setter: Dispatch<SetStateAction<Record<string, ByAgent<T>>>>, targetId: string, change: (current: ByAgent<T>) => ByAgent<T>) {
  setter((all) => ({ ...all, [targetId]: change(all[targetId] ?? {}) }));
}

const EMPTY_UPDATES: ByAgent<AgentUpdate | null> = {};
const EMPTY_RUNS: ByAgent<AgentActionRun> = {};

/**
 * Read checks for this target while the launcher is open, and run the actions they
 * offer. Failure is unknown. Readings and runs are held per machine, because an
 * action can outlast the operator's look at it: a run on one machine neither blocks
 * the same agent on another nor reports into its panel, and returning shows it.
 */
export function useAgentUpdates(open: boolean, targetId: string, agentKey: string) {
  const [updatesByTarget, setUpdatesByTarget] = useState<Record<string, ByAgent<AgentUpdate | null>>>({});
  const [loading, setLoading] = useState(false);
  const [runsByTarget, setRunsByTarget] = useState<Record<string, ByAgent<AgentActionRun>>>({});
  const running = useRef(new Set<string>());
  const updatesRef = useRef(updatesByTarget);
  updatesRef.current = updatesByTarget;
  // A finished run's note answers "did that work?" for this visit, not the next.
  useEffect(() => {
    if (!open) return;
    setRunsByTarget((all) => Object.fromEntries(Object.entries(all).map(([id, runs]) => [id, Object.fromEntries(Object.entries(runs).filter(([, item]) => item.phase === "running"))])));
  }, [open]);

  const check = useCallback(async (checkTarget: string, agentId: string, signal?: AbortSignal) => {
    let update: AgentUpdate | null = null;
    try {
      const value = await capabilitiesClient.agentUpdate({ targetId: checkTarget, agentId }, { signal });
      if (value.targetId === checkTarget && value.agentId === agentId) update = decode(value);
    } catch { /* Unavailable is not an update. */ }
    if (!signal?.aborted) setFor(setUpdatesByTarget, checkTarget, (current) => ({ ...current, [agentId]: update }));
  }, []);

  useEffect(() => {
    if (!open || !agentKey) { setLoading(false); return; }
    const abort = new AbortController();
    setLoading(true);
    void Promise.all(agentKey.split(",").filter(Boolean).map((agentId) => check(targetId, agentId, abort.signal))).then(() => {
      if (!abort.signal.aborted) setLoading(false);
    });
    return () => { abort.abort(); };
  }, [agentKey, check, open, targetId]);

  /** Runs one action. A second press on the same agent and machine does nothing, and it never rejects. */
  const run = useCallback(async (agentId: string, action: AgentUpdateAction) => {
    // Captured: this run belongs to the machine it started on, wherever the view goes next.
    const runTarget = targetId;
    const key = JSON.stringify([runTarget, agentId]);
    if (running.current.has(key)) return;
    running.current.add(key);
    const previous = updatesRef.current[runTarget]?.[agentId]?.installed ?? "";
    setFor<AgentActionRun>(setRunsByTarget, runTarget, (current) => ({ ...current, [agentId]: { phase: "running", action } }));
    let next: AgentActionRun | null = null;
    for (let attempt = 1; next === null; attempt += 1) {
      try {
        const response = await capabilitiesClient.runAgentUpdateAction({ targetId: runTarget, agentId, action }, { timeoutMs: ACTION_TIMEOUT_MS });
        const observed = response.update ? decode(response.update) : null;
        if (observed) setFor(setUpdatesByTarget, runTarget, (current) => ({ ...current, [agentId]: observed }));
        // A check that came back is simply the new state; nothing to announce.
        next = response.success
          ? { phase: "done", action, message: response.message, previous, current: observed?.installed ?? "" }
          : { phase: "failed", action, message: response.message, unsupported: false, outcomeUnknown: false };
      } catch (error) {
        const connect = ConnectError.from(error);
        const transport = TRANSPORT_CODES.has(connect.code);
        // A rollback is not idempotent, so a dropped one is read back, never repeated.
        if (transport && action === "rollback") {
          await check(runTarget, agentId);
          next = { phase: "failed", action, message: "", unsupported: false, outcomeUnknown: true };
        } else if (transport && attempt < TRANSPORT_ATTEMPTS) {
          continue;
        } else {
          next = { phase: "failed", action, message: connect.rawMessage, unsupported: connect.code === Code.Unimplemented, outcomeUnknown: false };
        }
      }
    }
    running.current.delete(key);
    const finished = next;
    setFor(setRunsByTarget, runTarget, (current) => (finished.phase === "done" && action === "retry"
      ? Object.fromEntries(Object.entries(current).filter(([id]) => id !== agentId))
      : { ...current, [agentId]: finished }));
  }, [check, targetId]);

  /** Looks again from the source. Failure stays on the harness's page. */
  const retryCheck = useCallback((agentId: string) => { void run(agentId, "retry"); }, [run]);

  return { updates: updatesByTarget[targetId] ?? EMPTY_UPDATES, loading, runs: runsByTarget[targetId] ?? EMPTY_RUNS, run, retryCheck };
}

export type AgentUpdatesController = ReturnType<typeof useAgentUpdates>;
