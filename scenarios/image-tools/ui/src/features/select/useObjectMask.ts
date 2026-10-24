import { useCallback, useEffect, useRef, useState } from "react";
import { submitAI } from "../../api/ai";
import { blobUrl } from "../../api/client";
import { jobsClient, JobState, type Job } from "../../api/jobs";

export interface MaskPoint { x: number; y: number; negative: boolean }
export interface ObjectMaskClient {
  submit: typeof submitAI;
  wait: (id: string, signal: AbortSignal) => Promise<Job>;
  cancel: (id: string) => Promise<void>;
  blobUrl: typeof blobUrl;
}
const liveClient: ObjectMaskClient = {
  submit: submitAI,
  wait: async (id, signal) => {
    const response = await jobsClient.waitJob({ id }, { signal });
    if (!response.job) throw new Error("The mask job returned no record");
    return response.job;
  },
  cancel: async (id) => { await jobsClient.cancelJob({ id }); },
  blobUrl,
};

/** Each changed prompt has a new durable identity; a failed response can retry
 * the same identity. WaitJob blocks once. Cancelling observation also asks the
 * job owner to stop inference, including an admission that arrives late. */
export function useObjectMask(image: File, client: ObjectMaskClient = liveClient) {
  const [points, setPoints] = useState<MaskPoint[]>([]);
  const [model, setModel] = useState("");
  const [maskRef, setMaskRef] = useState("");
  const [jobId, setJobId] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [cancelError, setCancelError] = useState("");
  const [servingModel, setServingModel] = useState("");
  const identity = useRef<string | null>(null);
  const active = useRef<{ controller: AbortController; jobId: string } | null>(null);
  const mounted = useRef(true);

  const cancelWork = useCallback(() => {
    const work = active.current;
    active.current = null;
    if (!work) return;
    work.controller.abort();
    if (work.jobId) void client.cancel(work.jobId).catch((err: unknown) => {
      if (mounted.current) setCancelError(err instanceof Error ? err.message : String(err));
    });
  }, [client]);
  useEffect(() => {
    mounted.current = true;
    return () => { mounted.current = false; cancelWork(); };
  }, [cancelWork]);
  const change = useCallback((next: MaskPoint[]) => {
    cancelWork();
    identity.current = null;
    setPoints(next);
    setMaskRef("");
    setJobId("");
    setBusy(false);
    setError("");
  }, [cancelWork]);
  const chooseModel = (next: string) => { change(points); setModel(next); };
  const run = async () => {
    if (busy || !points.some((p) => !p.negative)) return;
    cancelWork();
    const work = { controller: new AbortController(), jobId: "" };
    active.current = work;
    const operationId = identity.current ?? crypto.randomUUID();
    identity.current = operationId;
    setBusy(true); setError(""); setCancelError(""); setMaskRef("");
    try {
      // Keep admission observable even if the user cancels while it is in
      // flight: once its job identity arrives, cancel that actual owner job.
      const accepted = await client.submit("segment", { segmentation: { points }, modelOverride: model }, { image, operationId });
      work.jobId = accepted.jobId;
      if (work.controller.signal.aborted) {
        try { await client.cancel(accepted.jobId); } catch (err) { if (mounted.current) setCancelError(err instanceof Error ? err.message : String(err)); }
        return;
      }
      setJobId(accepted.jobId); setServingModel(accepted.modelId);
      const job = await client.wait(accepted.jobId, work.controller.signal);
      if (active.current !== work) return;
      if (job.state !== JobState.SUCCEEDED || !job.resultRef) {
        identity.current = null;
        throw new Error(job.error || job.message || "The mask job did not produce a mask");
      }
      setMaskRef(job.resultRef);
    } catch (err) {
      if (mounted.current && !work.controller.signal.aborted) setError(err instanceof Error ? err.message : String(err));
    } finally {
      if (active.current === work) { active.current = null; if (mounted.current) setBusy(false); }
    }
  };
  return {
    points, model, chooseModel, run, busy, error, cancelError, jobId, servingModel,
    maskUrl: maskRef ? client.blobUrl(maskRef) : "",
    addPoint: (point: MaskPoint) => change([...points, point]),
    undo: () => change(points.slice(0,-1)), clear: () => change([]),
    cancel: () => { cancelWork(); identity.current = null; setBusy(false); },
  };
}
