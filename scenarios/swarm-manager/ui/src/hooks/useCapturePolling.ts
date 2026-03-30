/**
 * useCapturePolling - Polls for capture classification updates.
 *
 * Refreshes captures every 3s when any are in "classifying" status.
 * Stops after 60s to let the backend auto-fail at 2 min.
 */

import { useEffect } from "react";
import { useCaptureStore } from "../stores/capture-store";

const POLL_INTERVAL_MS = 3_000;
const STALENESS_THRESHOLD_MS = 60_000;

export function useCapturePolling(): void {
  const captures = useCaptureStore((s) => s.captures);
  const fetchCaptures = useCaptureStore((s) => s.fetchCaptures);

  useEffect(() => {
    const classifying = captures.filter((c) => c.status === "classifying");
    if (classifying.length === 0) return;

    const allStale = classifying.every(
      (c) => Date.now() - new Date(c.created).getTime() > STALENESS_THRESHOLD_MS,
    );
    if (allStale) return;

    const interval = setInterval(() => void fetchCaptures({ force: true }), POLL_INTERVAL_MS);
    return () => clearInterval(interval);
  }, [captures, fetchCaptures]);
}
