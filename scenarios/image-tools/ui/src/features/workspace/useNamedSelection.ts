import { useCallback, useEffect, useRef, useState } from "react";

import { submitAI } from "../../api/ai";
import { fetchBlob } from "../../api/client";
import { jobsClient, JobState, type Job } from "../../api/jobs";
import type { SegmentOpParams } from "./opParams.generated";

/** The seams the live hook talks through; injected in tests. */
export interface NamedSelectionClient {
  submit: typeof submitAI;
  wait: (id: string, signal: AbortSignal) => Promise<Job>;
  cancel: (id: string) => Promise<void>;
  fetchMask: (key: string) => Promise<Blob>;
}

const liveClient: NamedSelectionClient = {
  submit: submitAI,
  wait: async (id, signal) => {
    const response = await jobsClient.waitJob({ id }, { signal });
    if (!response.job) throw new Error("The selection job returned no record");
    return response.job;
  },
  cancel: async (id) => { await jobsClient.cancelJob({ id }); },
  fetchMask: fetchBlob,
};

export interface NamedSelection {
  /** True while a selection job is running. */
  busy: boolean;
  /** The reason the last selection failed (a phrase the model could not find, for one). */
  error: string;
  /** Segment `phrase` in `image` and resolve with the white-on-black mask, or null when it was cancelled or failed. */
  select: (image: File, phrase: string) => Promise<File | null>;
  cancel: () => void;
}

/**
 * Select part of a picture by name ("hair", "the jacket"). It runs the same
 * `segment` job the CLI does, with the default segmentation model the service
 * resolves, and returns the mask as a PNG file ready to be an edit's mask. The
 * job is cancelled when the caller leaves or asks for another selection.
 */
export function useNamedSelection(client: NamedSelectionClient = liveClient): NamedSelection {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const active = useRef<{ controller: AbortController; jobId: string } | null>(null);
  const mounted = useRef(true);

  const cancelWork = useCallback(() => {
    const work = active.current;
    active.current = null;
    if (!work) return;
    work.controller.abort();
    if (work.jobId) void client.cancel(work.jobId).catch(() => undefined);
  }, [client]);

  useEffect(() => {
    mounted.current = true;
    return () => { mounted.current = false; cancelWork(); };
  }, [cancelWork]);

  const select = useCallback(async (image: File, phrase: string): Promise<File | null> => {
    const text = phrase.trim();
    if (!text) return null;
    cancelWork();
    const work = { controller: new AbortController(), jobId: "" };
    active.current = work;
    setBusy(true);
    setError("");
    try {
      const accepted = await client.submit(
        "segment",
        { opParams: { segmentation: { text } } satisfies SegmentOpParams },
        { image, operationId: crypto.randomUUID(), signal: work.controller.signal },
      );
      work.jobId = accepted.jobId;
      if (work.controller.signal.aborted) {
        await client.cancel(accepted.jobId).catch(() => undefined);
        return null;
      }
      const job = await client.wait(accepted.jobId, work.controller.signal);
      if (active.current !== work) return null;
      if (job.state !== JobState.SUCCEEDED || !job.resultRef) {
        throw new Error(job.error || job.message || `Nothing was selected for "${text}"`);
      }
      const blob = await client.fetchMask(job.resultRef);
      return new File([blob], "mask.png", { type: "image/png" });
    } catch (err) {
      if (mounted.current && !work.controller.signal.aborted) setError(err instanceof Error ? err.message : String(err));
      return null;
    } finally {
      if (active.current === work) {
        active.current = null;
        if (mounted.current) setBusy(false);
      }
    }
  }, [cancelWork, client]);

  return { busy, error, select, cancel: () => { cancelWork(); setBusy(false); } };
}
