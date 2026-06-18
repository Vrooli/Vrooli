import { useCallback, useEffect, useRef, useState } from "react";

import { analyze, type AnalyzeResult } from "../../api/analysis";
import { liveEnhanceClient, type SelectedModel } from "./useEnhance";

/**
 * Analyze reuses the install-gate vocabulary from Enhance but runs
 * *synchronously*: one POST returns the structured result (no durable job to
 * watch). `needs-install` is the model gate for the model-backed ops
 * (ocr / nsfw_classify); the pure-Go `probe` skips it.
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
  install: (modelId: string) => Promise<{ jobId: string; alreadyInstalled: boolean }>;
  waitJob: (jobId: string) => Promise<{ ok: boolean; error: string }>;
}

/** Production client: the sync analyze edge + shared Models/Jobs methods. */
export const liveAnalyzeClient: AnalyzeClient = {
  selectModel: liveEnhanceClient.selectModel,
  install: liveEnhanceClient.install,
  waitJob: liveEnhanceClient.waitJob,
  analyze: (op, file) => analyze(op, file),
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

  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
    };
  }, []);

  const live = useCallback((id: number) => mounted.current && runId.current === id, []);

  const doAnalyze = useCallback(
    async (op: string, file: File, id: number) => {
      setPhase("running");
      try {
        const next = await client.analyze(op, file);
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
    setPhase("idle");
  }, []);

  const retry = useCallback(() => {
    const job = pending.current;
    if (job) {
      run(job.op, job.file, job.modelBacked);
    }
  }, [run]);

  const clear = useCallback(() => {
    runId.current += 1;
    setResult(null);
    setError(null);
    setModel(null);
    setPhase("idle");
  }, []);

  return { phase, model, result, error, run, installAndRun, cancel, retry, clear };
}
