import { useEffect, useState } from "react";

import { jobsClient, type ProgressEvent } from "../../api/jobs";

/**
 * Subscribe to live job progress over the WatchJob server stream (SSE-style).
 * Returns the latest ProgressEvent, or null until the first event arrives.
 *
 * The Connect-Web transport surfaces WatchJob as an async iterable; the server
 * replays the latest known event first, then streams updates until the job is
 * terminal. The subscription is torn down (and the request aborted) when
 * `enabled` flips false or the component unmounts, so terminal jobs don't hold
 * an open stream. Stream errors are swallowed — the caller still has the
 * polled ListJobs snapshot to fall back on.
 */
export function useJobProgress(jobId: string, enabled: boolean): ProgressEvent | null {
  const [event, setEvent] = useState<ProgressEvent | null>(null);

  useEffect(() => {
    if (!enabled || !jobId) {
      return;
    }
    const controller = new AbortController();

    void (async () => {
      try {
        for await (const next of jobsClient.watchJob({ id: jobId }, { signal: controller.signal })) {
          if (controller.signal.aborted) {
            break;
          }
          setEvent(next);
        }
      } catch {
        // Stream aborted (unmount / disable) or failed; the polled snapshot
        // from ListJobs remains the source of truth.
      }
    })();

    return () => {
      controller.abort();
    };
  }, [jobId, enabled]);

  return event;
}
