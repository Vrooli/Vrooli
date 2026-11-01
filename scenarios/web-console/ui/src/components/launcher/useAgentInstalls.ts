import { useCallback, useEffect, useRef, useState } from "react";
import type { InstallOutcome } from "../../api/capabilities";

/** How long the transient "Installed" confirmation stays. */
const CONFIRMATION_MS = 6000;

function without(source: Record<string, InstallOutcome>, keys: string[]): Record<string, InstallOutcome> {
  const drop = new Set(keys);
  return Object.fromEntries(Object.entries(source).filter(([key]) => !drop.has(key)));
}

/**
 * Governed installs started from this launcher, with the machine's own verdict.
 * Success is never inferred from the call resolving: "the installer ran" and
 * "the agent is there" are different claims, and only the second is an install
 * the operator can act on.
 */
export function useAgentInstalls(targetId: string, onInstall: ((agentId: string) => Promise<InstallOutcome>) | undefined, readyAgentKey: string) {
  const [installing, setInstalling] = useState<string[]>([]);
  const [outcomes, setOutcomes] = useState<Record<string, InstallOutcome>>({});
  const timers = useRef<number[]>([]);

  useEffect(() => () => { for (const timer of timers.current) window.clearTimeout(timer); }, []);
  useEffect(() => { setInstalling([]); setOutcomes({}); }, [targetId]);

  const startInstall = useCallback(async (agentId: string) => {
    if (!onInstall) return;
    // Clear any previous verdict first: a retry that still showed the last
    // failure underneath its own progress would be reporting two answers.
    setOutcomes((current) => (agentId in current ? without(current, [agentId]) : current));
    setInstalling((current) => (current.includes(agentId) ? current : [...current, agentId]));
    let outcome: InstallOutcome;
    try {
      outcome = await onInstall(agentId);
    } catch (error) {
      // A transport failure is a failed install with no message from the
      // machine, which is still more than the operator had before.
      console.error("capability install failed", agentId, error);
      outcome = { status: "failed" };
    }
    setInstalling((current) => current.filter((id) => id !== agentId));
    setOutcomes((current) => ({ ...current, [agentId]: outcome }));
    // Only the success is transient. A failure or an unconfirmed install carries
    // the only explanation the operator will get, so it stays until a retry.
    if (outcome.status === "installed") {
      timers.current.push(window.setTimeout(() => {
        setOutcomes((current) => (current[agentId]?.status === "installed" ? without(current, [agentId]) : current));
      }, CONFIRMATION_MS));
    }
  }, [onInstall]);

  // An outcome describes a machine state at a moment. Once the machine reports
  // the agent as present, a stored verdict is stale by definition.
  useEffect(() => {
    const ready = new Set(readyAgentKey ? readyAgentKey.split(",") : []);
    setOutcomes((current) => {
      const stale = Object.keys(current).filter((id) => ready.has(id) && current[id]?.status !== "installed");
      return stale.length === 0 ? current : without(current, stale);
    });
  }, [readyAgentKey]);

  return { installing, outcomes, startInstall };
}

export type AgentInstallsController = ReturnType<typeof useAgentInstalls>;
