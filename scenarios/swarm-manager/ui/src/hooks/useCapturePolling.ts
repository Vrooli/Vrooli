/**
 * useCapturePolling - Polls for capture classification updates.
 *
 * Refreshes captures every 3s when any are in "classifying" status.
 * Stops after 60s to let the backend auto-fail at 2 min.
 */

import { useMemo } from "react";
import { useCaptureStore } from "../stores/capture-store";
import { useStorePolling } from "./useStorePolling";

const POLL_INTERVAL_MS = 3_000;
const STALENESS_THRESHOLD_MS = 60_000;

export function useCapturePolling(): void {
  const captures = useCaptureStore((s) => s.captures);
  const fetchCaptures = useCaptureStore((s) => s.fetchCaptures);

  const shouldPoll = useMemo(() => {
    const classifying = captures.filter((c) => c.status === "classifying");
    if (classifying.length === 0) return false;

    return !classifying.every(
      (c) => Date.now() - new Date(c.created).getTime() > STALENESS_THRESHOLD_MS,
    );
  }, [captures]);

  useStorePolling({
    enabled: shouldPoll,
    intervalMs: POLL_INTERVAL_MS,
    pollFn: () => void fetchCaptures({ force: true }),
  });
}
