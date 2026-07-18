/**
 * useCapturePolling - Polls for capture classification updates.
 *
 * Refreshes captures every 3s when any are in "classifying" status.
 * For workflow-backed captures, each refresh also asks Swarm to apply a
 * matching terminal typed result. Agent Manager owns timeout and retry policy.
 */

import { useMemo } from "react";
import { useCaptureStore } from "../stores/capture-store";
import { captureService } from "../services/capture-service";
import { useStorePolling } from "./useStorePolling";

const POLL_INTERVAL_MS = 3_000;

export function useCapturePolling(): void {
  const captures = useCaptureStore((s) => s.captures);
  const fetchCaptures = useCaptureStore((s) => s.fetchCaptures);
  const classifying = captures.filter((capture) => capture.status === "classifying");

  const shouldPoll = useMemo(() => {
    if (classifying.length === 0) return false;

    return classifying.some((capture) =>
      Boolean(capture.workflowExecutionId) || Date.now() - new Date(capture.created).getTime() <= 60_000,
    );
  }, [captures]);

  useStorePolling({
    enabled: shouldPoll,
    intervalMs: POLL_INTERVAL_MS,
    pollFn: () => {
      void Promise.all(classifying
        .filter((capture) => capture.workflowExecutionId)
        .map((capture) => captureService.applyClassification(capture.id, capture.workflowExecutionId!).catch(() => undefined)));
      void fetchCaptures({ force: true });
    },
  });
}
