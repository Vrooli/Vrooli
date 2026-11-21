import { useCallback, useEffect, useRef, useState } from "react";
import { cancelPrecommitCandidateCheck, getPrecommitCandidateCheck, precommitChecksEnabled, startPrecommitCandidateCheck, type PrecommitCandidateCheck } from "./precommitCandidateApi";

type CheckState = "idle" | "loading" | "disabled" | "queued" | "running" | "cancelling" | "cancelled" | "passed" | "failed" | "timeout" | "incomplete" | "error";
interface ViewState {
  state: CheckState;
  check?: PrecommitCandidateCheck;
  error?: string;
  stale?: boolean;
}

const storageKey = (repositoryId: string) => `gct.precommitCandidate.${repositoryId}`;
const terminal = new Set(["passed", "failed", "timeout", "incomplete", "cancelled"]);

function readSavedJob(repositoryId: string): string | undefined {
  try {
    return window.localStorage.getItem(storageKey(repositoryId)) || undefined;
  } catch {
    return undefined;
  }
}

function saveJob(repositoryId: string, jobId: string): void {
  try {
    window.localStorage.setItem(storageKey(repositoryId), jobId);
  } catch {
    // Server-side identity and deterministic Start remain authoritative.
  }
}

export function usePrecommitCandidateCheck(repositoryId: string, enabled: boolean, statusVersion: number, updatingIndex: boolean, hasCandidate: boolean) {
  const [view, setView] = useState<ViewState>({ state: "idle" });
  const sequence = useRef(0);
  const retrying = useRef(false);

  const refresh = useCallback(async (signal?: AbortSignal, retryTerminal = false) => {
    const current = ++sequence.current;
    const stillCurrent = () => !signal?.aborted && current === sequence.current;
    if (!enabled || !hasCandidate) {
      setView({ state: "idle" });
      return;
    }
    setView(previous => ({ ...previous, state: "loading", stale: previous.state === "passed" }));
    try {
      if (!await precommitChecksEnabled(repositoryId)) {
        if (stillCurrent()) setView({ state: "disabled" });
        return;
      }
      let check: PrecommitCandidateCheck;
      let showingPrevious = false;
      try {
        // Start is idempotent for a repository/candidate/config identity. It
        // also reattaches after reload and starts a fresh job after a stage change.
        check = await startPrecommitCandidateCheck(repositoryId, retryTerminal);
      } catch (startError) {
        const saved = readSavedJob(repositoryId);
        if (!saved) throw startError;
        check = await getPrecommitCandidateCheck(repositoryId, saved);
        showingPrevious = true;
      }
      if (!stillCurrent()) return;
      if (check.jobId) saveJob(repositoryId, check.jobId);
      if (showingPrevious) {
        setView({ state: "error", check, error: "The current staged candidate could not be confirmed; this receipt is only for a previous candidate." });
        return;
      }
      setView({ state: check.state as CheckState, check });
      while (check.jobId && !terminal.has(check.state)) {
        await new Promise<void>(resolve => window.setTimeout(resolve, 900));
        if (!stillCurrent()) return;
        check = await getPrecommitCandidateCheck(repositoryId, check.jobId);
        if (!stillCurrent()) return;
        setView({ state: check.state as CheckState, check });
      }
    } catch (error) {
      if (stillCurrent()) setView({ state: "error", error: error instanceof Error ? error.message : String(error) });
    }
  }, [enabled, hasCandidate, repositoryId]);

  useEffect(() => {
    if (!enabled || !hasCandidate || updatingIndex) {
      if (!hasCandidate) setView({ state: "idle" });
      return;
    }
    const controller = new AbortController();
    const sequenceCounter = sequence;
    void refresh(controller.signal);
    return () => {
      controller.abort();
      sequenceCounter.current++;
    };
  }, [enabled, hasCandidate, refresh, statusVersion, updatingIndex]);

  const retry = useCallback(async () => {
    if (retrying.current) return;
    retrying.current = true;
    setView(previous => ({ state: "loading", check: previous.check, stale: false }));
    try {
      await refresh(undefined, true);
    } finally {
      retrying.current = false;
    }
  }, [refresh]);

  const cancel = useCallback(async () => {
    const jobId = view.check?.jobId;
    if (!jobId) return;
    sequence.current++;
    setView(previous => ({ ...previous, state: "cancelling" }));
    try {
      const check = await cancelPrecommitCandidateCheck(repositoryId, jobId);
      setView({ state: check.state as CheckState, check });
    } catch (error) {
      setView(previous => ({ ...previous, state: "error", error: error instanceof Error ? error.message : String(error) }));
    }
  }, [repositoryId, view.check?.jobId]);

  return { ...view, refresh: () => refresh(), retry, cancel };
}

