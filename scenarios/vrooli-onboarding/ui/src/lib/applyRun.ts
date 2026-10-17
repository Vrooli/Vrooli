import { ApplyRunState, type GetApplyRunResponse } from "@vrooli/proto-types/vrooli-onboarding/v1/apply/apply_pb";

/**
 * Applying a selection starts scenarios, and starting a scenario can restart
 * the onboarding API this page is talking to. The run itself is executed by a
 * separate process and survives that restart; only the connection breaks.
 *
 * Treating the first failed poll as failure told the operator "the selection
 * could not be applied" while the apply was in fact still running, which is
 * both false and the most alarming thing the page could say at that moment.
 */
export const APPLY_RECONNECT_WINDOW_MS = 5 * 60 * 1000;
export const APPLY_POLL_INTERVAL_MS = 500;

export function isApplyRunSettled(status: ApplyRunState): boolean {
  return status !== ApplyRunState.PENDING && status !== ApplyRunState.APPLYING;
}

export interface PollApplyRunOptions {
  /** Fetches the current server-owned run state. */
  fetchStatus: (runID: string) => Promise<GetApplyRunResponse>;
  /** Called on every observed state, including the first. */
  onUpdate: (run: GetApplyRunResponse) => void;
  /** Called when the API becomes unreachable, and again when it returns. */
  onConnectionChange?: (connected: boolean) => void;
  wait: (ms: number) => Promise<void>;
  now: () => number;
  reconnectWindowMs?: number;
  pollIntervalMs?: number;
}

/**
 * Follows a server-owned apply run to a settled state, tolerating the API
 * disappearing underneath it.
 *
 * Throws only when the run cannot be observed for the whole reconnect window --
 * that is the one case where this client genuinely cannot say what happened.
 */
export async function pollApplyRun(
  accepted: GetApplyRunResponse,
  options: PollApplyRunOptions,
): Promise<GetApplyRunResponse> {
  const {
    fetchStatus,
    onUpdate,
    onConnectionChange,
    wait,
    now,
    reconnectWindowMs = APPLY_RECONNECT_WINDOW_MS,
    pollIntervalMs = APPLY_POLL_INTERVAL_MS,
  } = options;

  onUpdate(accepted);
  let current = accepted;
  let unreachableSince: number | null = null;

  while (!isApplyRunSettled(current.status)) {
    await wait(pollIntervalMs);
    try {
      current = await fetchStatus(accepted.runId);
      if (unreachableSince !== null) {
        unreachableSince = null;
        onConnectionChange?.(true);
      }
      onUpdate(current);
    } catch (error) {
      if (unreachableSince === null) {
        unreachableSince = now();
        onConnectionChange?.(false);
      }
      if (now() - unreachableSince > reconnectWindowMs) {
        throw error;
      }
    }
  }
  return current;
}
