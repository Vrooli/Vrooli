import { useCallback, useEffect, useRef, useState } from "react";

import { analyze, type AnalyzeCaption, type AnalyzeResult } from "../../api/analysis";
import { liveEnhanceClient, type SelectedModel } from "./useEnhance";
import { fetchAICaptionResult, submitAI } from "../../api/ai";

/**
 * Analyze reuses the install-gate vocabulary from Enhance. Most checks return
 * structured results synchronously; caption uses a durable AI job. `needs-install`
 * is the model gate for model-backed operations while pure-Go checks skip it.
 */
export type AnalyzePhase =
  | "idle"
  | "needs-install"
  | "installing"
  | "running"
  | "done"
  | "failed";

const ACTIVE_PHASES: ReadonlySet<AnalyzePhase> = new Set<AnalyzePhase>(["installing", "running"]);

export const isAnalyzeActive = (phase: AnalyzePhase): boolean => ACTIVE_PHASES.has(phase);

/**
 * The injected seam the hook drives. The model methods reuse `liveEnhanceClient`
 * (one production implementation of select/install/wait); `analyze` is the sync
 * analysis edge. Tests pass a fake so the whole flow runs without the network.
 */
export interface AnalyzeClient {
  selectModel: (op: string) => Promise<SelectedModel>;
  analyze: (op: string, file: File) => Promise<AnalyzeResult>;
  caption: (file: File, signal: AbortSignal, onJob: (jobId: string) => void) => Promise<AnalyzeCaption>;
  cancelCaption: (jobId: string) => Promise<void>;
  install: (modelId: string) => Promise<{ jobId: string; alreadyInstalled: boolean }>;
  waitJob: (jobId: string) => Promise<{ ok: boolean; error: string }>;
}

/** Production client: the sync analyze edge + shared Models/Jobs methods. */
export const liveAnalyzeClient: AnalyzeClient = {
  selectModel: liveEnhanceClient.selectModel,
  install: liveEnhanceClient.install,
  waitJob: liveEnhanceClient.waitJob,
  analyze: (op, file) => analyze(op, file),
  async caption(file, signal, onJob) {
    const submitted = await submitAI("caption", { prompt: "Describe the visible contents of this image." }, { image: file, signal });
    onJob(submitted.jobId);
    if (signal.aborted) {
      await liveEnhanceClient.cancel(submitted.jobId).catch(() => undefined);
      throw new DOMException("Caption canceled", "AbortError");
    }
    let canceled = false;
    const markCanceled = () => { canceled = true; };
    signal.addEventListener("abort", markCanceled, { once: true });
    await liveEnhanceClient.watch(submitted.jobId, signal, () => undefined);
    signal.removeEventListener("abort", markCanceled);
    // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition -- this flag is mutated by the AbortSignal listener during the awaited watch.
    if (canceled) throw new DOMException("Caption canceled", "AbortError");
    const terminal = await liveEnhanceClient.result(submitted.jobId);
    if (!terminal.ok) throw new Error(terminal.error || "Caption job failed");
    const payload = await fetchAICaptionResult(terminal.resultRef);
    return { kind: "caption", jobId: submitted.jobId, ...payload };
  },
  cancelCaption: liveEnhanceClient.cancel,
};

export interface UseAnalyzeArgs {
  client?: AnalyzeClient;
}

export interface UseAnalyze {
  phase: AnalyzePhase;
  model: SelectedModel | null;
  result: AnalyzeResult | null;
  error: string | null;
  /** Select the model (if model-backed), then run, else open the install gate. */
  run: (op: string, file: File, modelBacked: boolean) => void;
  /** Install the gated model (durable job), then run. */
  installAndRun: () => void;
  cancel: () => void;
  retry: () => void;
  /** Reset to idle (clears the result/model), e.g. on op or image change. */
  clear: () => void;
}

const errorText = (err: unknown): string => (err instanceof Error ? err.message : String(err));

/**
 * Drives one analysis op: (model-backed) select model → install gate → run, or
 * (pure-Go probe) run directly. The run is a single synchronous POST whose
 * structured result is handed back via `result`. Every entry point is a
 * user-triggered handler guarded by a run-id so a superseding run / unmount can
 * never let a stale result land. The client seam is injected for tests.
 */
export function useAnalyze({ client = liveAnalyzeClient }: UseAnalyzeArgs = {}): UseAnalyze {
  const [phase, setPhase] = useState<AnalyzePhase>("idle");
  const [model, setModel] = useState<SelectedModel | null>(null);
  const [result, setResult] = useState<AnalyzeResult | null>(null);
  const [error, setError] = useState<string | null>(null);

  const mounted = useRef(true);
  const runId = useRef(0);
  const pending = useRef<{ op: string; file: File; modelBacked: boolean } | null>(null);
  const abortRef = useRef<AbortController | null>(null);
  const jobIdRef = useRef("");

  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
      abortRef.current?.abort();
      if (jobIdRef.current) void client.cancelCaption(jobIdRef.current).catch(() => undefined);
    };
  }, [client]);

  const live = useCallback((id: number) => mounted.current && runId.current === id, []);

  const doAnalyze = useCallback(
    async (op: string, file: File, id: number) => {
      setPhase("running");
      try {
        let next: AnalyzeResult;
        if (op === "caption") {
          const controller = new AbortController();
          abortRef.current = controller;
          next = await client.caption(file, controller.signal, (jobId) => { jobIdRef.current = jobId; });
          jobIdRef.current = "";
        } else {
          next = await client.analyze(op, file);
        }
        if (live(id)) {
          setResult(next);
          setPhase("done");
        }
      } catch (err) {
        if (live(id)) {
          setError(errorText(err));
          setPhase("failed");
        }
      }
    },
    [client, live],
  );

  const run = useCallback(
    (op: string, file: File, modelBacked: boolean) => {
      const id = ++runId.current;
      pending.current = { op, file, modelBacked };
      setError(null);
      setResult(null);
      setPhase("running");

      void (async () => {
        if (modelBacked) {
          let selected: SelectedModel;
          try {
            selected = await client.selectModel(op);
          } catch (err) {
            if (live(id)) {
              setError(errorText(err));
              setPhase("failed");
            }
            return;
          }
          if (!live(id)) {
            return;
          }
          setModel(selected);
          if (!selected.installed) {
            setPhase("needs-install");
            return;
          }
        } else {
          setModel(null);
        }
        await doAnalyze(op, file, id);
      })();
    },
    [client, doAnalyze, live],
  );

  const installAndRun = useCallback(() => {
    const job = pending.current;
    const target = model;
    if (!job || !target) {
      return;
    }
    const id = ++runId.current;
    setError(null);
    setPhase("installing");

    void (async () => {
      let installed: { jobId: string; alreadyInstalled: boolean };
      try {
        installed = await client.install(target.id);
      } catch (err) {
        if (live(id)) {
          setError(errorText(err));
          setPhase("failed");
        }
        return;
      }
      if (!live(id)) {
        return;
      }
      if (!installed.alreadyInstalled && installed.jobId) {
        let waited: { ok: boolean; error: string };
        try {
          waited = await client.waitJob(installed.jobId);
        } catch (err) {
          if (live(id)) {
            setError(errorText(err));
            setPhase("failed");
          }
          return;
        }
        if (!live(id)) {
          return;
        }
        if (!waited.ok) {
          setError(waited.error || null);
          setPhase("failed");
          return;
        }
      }
      setModel({ ...target, installed: true });
      await doAnalyze(job.op, job.file, id);
    })();
  }, [client, doAnalyze, live, model]);

  const cancel = useCallback(() => {
    runId.current += 1;
    abortRef.current?.abort();
    if (jobIdRef.current) void client.cancelCaption(jobIdRef.current).catch(() => undefined);
    jobIdRef.current = "";
    setPhase("idle");
  }, [client]);

  const retry = useCallback(() => {
    const job = pending.current;
    if (job) {
      run(job.op, job.file, job.modelBacked);
    }
  }, [run]);

  const clear = useCallback(() => {
    runId.current += 1;
    abortRef.current?.abort();
    if (jobIdRef.current) void client.cancelCaption(jobIdRef.current).catch(() => undefined);
    jobIdRef.current = "";
    setResult(null);
    setError(null);
    setModel(null);
    setPhase("idle");
  }, [client]);

  return { phase, model, result, error, run, installAndRun, cancel, retry, clear };
}
