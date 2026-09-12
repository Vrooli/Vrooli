/**
 * Audits query + mutation hooks. RunSnapshotAudit is async on the server, so
 * the run mutation starts the audit and then polls getAudit until the record
 * reaches a terminal state, returning the final verdict to the caller. Listing
 * is invalidated on completion so a new audit shows up in history immediately.
 */
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { getAudit, isTerminalAudit, listAudits, runSnapshotAudit, type Audit, type RunAuditInput } from "../api/audits";
import { queryKeys } from "./keys";

export function useAudits(targetId = "") {
  return useQuery({
    queryKey: queryKeys.audits(targetId),
    queryFn: () => listAudits(targetId),
  });
}

const POLL_INTERVAL_MS = 1500;
const MAX_POLLS = 400; // generous ceiling; a wedged backend is reconciled server-side

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

/**
 * Starts an audit and resolves with the terminal record. The poll loop is a
 * sequence of fast getAudit calls — the heavy restore+capture runs server-side
 * on a background worker.
 */
async function runAndAwait(input: RunAuditInput): Promise<Audit> {
  const started = await runSnapshotAudit(input);
  if (!started) {
    throw new Error("server returned no audit");
  }
  if (isTerminalAudit(started.status)) {
    return started;
  }
  for (let i = 0; i < MAX_POLLS; i += 1) {
    await sleep(POLL_INTERVAL_MS);
    const current = await getAudit(started.id);
    if (current && isTerminalAudit(current.status)) {
      return current;
    }
  }
  throw new Error("audit did not reach a terminal state in time");
}

export function useRunAudit() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: RunAuditInput) => runAndAwait(input),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["audits"] });
    },
  });
}
