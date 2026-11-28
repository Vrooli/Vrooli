import { useEffect, useRef } from "react";
import type { Run, RunEvent } from "../types";
import type { RunEventReconciliationIntent } from "../lib/runEventStore";

interface ReconciliationJob {
  intent: RunEventReconciliationIntent;
  cancelled: boolean;
  controller: AbortController;
  timer?: ReturnType<typeof setTimeout>;
  failures: number;
}

interface ReconciliationOptions {
  intents: RunEventReconciliationIntent[];
  actions: {
    runSnapshotLoaded: (run: Run) => void;
    eventsGapFilled: (runId: string, events: RunEvent[], intent?: RunEventReconciliationIntent) => void;
  };
  getRun: (id: string, options?: { signal?: AbortSignal }) => Promise<Run>;
  getRunEvents: (id: string, options?: { afterSequence?: bigint; signal?: AbortSignal }) => Promise<RunEvent[]>;
}

function cancel(job: ReconciliationJob) {
  job.cancelled = true;
  job.controller.abort();
  clearTimeout(job.timer);
}

// A connected socket does not guarantee another notification after a failed
// reconnect read. Keep each intent until both durable projections reconcile.
// Requests use the shared API deadline; retry delay grows from 1s to 30s.
export function useRunReconciliation({ intents, actions, getRun, getRunEvents }: ReconciliationOptions) {
  const jobsRef = useRef(new Map<string, ReconciliationJob>());

  useEffect(() => {
    const jobs = jobsRef.current;
    return () => {
      for (const job of jobs.values()) cancel(job);
      jobs.clear();
    };
  }, []);

  useEffect(() => {
    const jobs = jobsRef.current;
    const wanted = new Map(intents.map((intent) => [intent.runId, intent]));
    for (const [runId, job] of jobs) {
      if (wanted.get(runId) !== job.intent) {
        cancel(job);
        jobs.delete(runId);
      }
    }
    for (const intent of intents) {
      if (jobs.has(intent.runId)) continue;
      const job: ReconciliationJob = { intent, cancelled: false, controller: new AbortController(), failures: 0 };
      jobs.set(intent.runId, job);
      const reconcile = async () => {
        if (job.cancelled) return;
        job.controller = new AbortController();
        const signal = job.controller.signal;
        try {
          const [events, run] = await Promise.all([
            getRunEvents(intent.runId, { afterSequence: intent.afterSequence, signal }),
            getRun(intent.runId, { signal }),
          ]);
          if (job.cancelled) return;
          actions.runSnapshotLoaded(run);
          actions.eventsGapFilled(intent.runId, events, intent);
          if (jobs.get(intent.runId) === job) jobs.delete(intent.runId);
        } catch (error) {
          job.controller.abort(); // Cancel the other read if its peer failed.
          if (job.cancelled) return;
          console.error(`Failed to reconcile run ${intent.runId}; retrying:`, error);
          const delay = Math.min(1000 * 2 ** job.failures, 30_000);
          job.failures = Math.min(job.failures + 1, 5);
          job.timer = setTimeout(() => { void reconcile(); }, delay);
        }
      };
      void reconcile();
    }
  }, [intents, actions, getRun, getRunEvents]);
}
