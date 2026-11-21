import { useCallback, useEffect, useRef, useState } from "react";

import {
  fetchAIResult,
  submitAI,
  type AIImageResult,
  type AIParamsInput,
} from "../../api/ai";
import { JobState, jobsClient } from "../../api/jobs";
import { modelsClient } from "../../api/models";
import type { RunOpImageResult } from "../../api/ops";

/** Normalized job state name, decoupled from the proto enum's numeric values. */
export type JobStateName =
  | "queued"
  | "running"
  | "succeeded"
  | "failed"
  | "canceled"
  | "unspecified";

const jobStateName = (state: JobState): JobStateName => {
  switch (state) {
    case JobState.QUEUED:
      return "queued";
    case JobState.RUNNING:
      return "running";
    case JobState.SUCCEEDED:
      return "succeeded";
    case JobState.FAILED:
      return "failed";
    case JobState.CANCELED:
      return "canceled";
    default:
      return "unspecified";
  }
};

/** The model resolved for an op, with the facts the install gate surfaces. */
export interface SelectedModel {
  id: string;
  name: string;
  installed: boolean;
  sizeMb: number;
  cpuCapable: boolean;
  gpuRequired: boolean;
  minVramGb: number;
  speedNote: string;
  gpuViable: boolean;
  reason: string;
  warnings: string[];
}

/** A single live progress tick from the durable job. */
export interface EnhanceProgress {
  percent: number;
  message: string;
  state: JobStateName;
}

/**
 * The injected seam the hook drives. The live implementation wraps the AI
 * submit edge + the Jobs and Models Connect clients; tests pass a fake so the
 * whole lifecycle runs without the network.
 */
export interface EnhanceClient {
  selectModel: (op: string) => Promise<SelectedModel>;
  submit: (op: string, params: AIParamsInput, input: File) => Promise<{
    jobId: string;
    tier: string;
    warnings: string[];
  }>;
  watch: (
    jobId: string,
    signal: AbortSignal,
    onEvent: (event: EnhanceProgress) => void,
  ) => Promise<void>;
  result: (jobId: string) => Promise<{ ok: boolean; resultRef: string; error: string }>;
  fetchResult: (resultRef: string) => Promise<AIImageResult>;
  install: (modelId: string) => Promise<{ jobId: string; alreadyInstalled: boolean }>;
  waitJob: (jobId: string) => Promise<{ ok: boolean; error: string }>;
  cancel: (jobId: string) => Promise<void>;
}

/** Production client: AI submit edge + Jobs/Models Connect clients. */
export const liveEnhanceClient: EnhanceClient = {
  async selectModel(op) {
    const resp = await modelsClient.selectModel({ operation: op });
    const m = resp.model;
    const hw = m?.hardware;
    return {
      id: m?.id ?? "",
      name: m?.name ?? "",
      installed: m?.install?.installed ?? false,
      sizeMb: m?.sizeMbApprox ?? 0,
      cpuCapable: hw?.cpuCapable ?? false,
      gpuRequired: hw?.gpuRequired ?? false,
      minVramGb: hw?.minVramGb ?? 0,
      speedNote: hw?.speedNote ?? "",
      gpuViable: resp.gpuViable,
      reason: resp.reason,
      warnings: resp.warnings,
    };
  },
  async submit(op, params, input) {
    const res = await submitAI(op, params, { image: input });
    return { jobId: res.jobId, tier: res.tier, warnings: res.warnings };
  },
  async watch(jobId, signal, onEvent) {
    try {
      for await (const ev of jobsClient.watchJob({ id: jobId }, { signal })) {
        if (signal.aborted) {
          break;
        }
        onEvent({ percent: ev.progress, message: ev.message, state: jobStateName(ev.state) });
      }
    } catch {
      // Stream aborted (cancel/unmount) or failed; the terminal state is read
      // authoritatively via `result` (GetJob) once the watch resolves.
    }
  },
  async result(jobId) {
    const resp = await jobsClient.getJob({ id: jobId });
    const job = resp.job;
    return {
      ok: job?.state === JobState.SUCCEEDED,
      resultRef: job?.resultRef ?? "",
      error: job?.error ?? "",
    };
  },
  fetchResult: (resultRef) => fetchAIResult(resultRef),
  async install(modelId) {
    const resp = await modelsClient.installModel({ id: modelId });
    return { jobId: resp.jobId, alreadyInstalled: resp.alreadyInstalled };
  },
  async waitJob(jobId) {
    const resp = await jobsClient.waitJob({ id: jobId });
    const job = resp.job;
    return { ok: job?.state === JobState.SUCCEEDED, error: job?.error ?? "" };
  },
  async cancel(jobId) {
    await jobsClient.cancelJob({ id: jobId });
  },
};

/**
 * The Enhance lifecycle phases. `needs-install` is the model-install gate; the
 * spinner phases (`submitting`/`installing`/`running`) drive both the inspector
 * progress card and the in-canvas overlay.
 */
export type EnhancePhase =
  | "idle"
  | "needs-install"
  | "installing"
  | "submitting"
  | "running"
  | "succeeded"
  | "failed";

const ACTIVE_PHASES: ReadonlySet<EnhancePhase> = new Set<EnhancePhase>([
  "submitting",
  "installing",
  "running",
]);

export const isEnhanceActive = (phase: EnhancePhase): boolean => ACTIVE_PHASES.has(phase);

export interface EnhanceResult {
  op: string;
  result: RunOpImageResult;
  outputFile: File;
}

export interface UseEnhanceArgs {
  onResult: (result: EnhanceResult) => void;
  client?: EnhanceClient;
}

export interface UseEnhance {
  phase: EnhancePhase;
  model: SelectedModel | null;
  progress: EnhanceProgress;
  tier: string;
  warnings: string[];
  error: string | null;
  /** Fetch the op's selected-model badge without running (called on op select). */
  preview: (op: string) => void;
  /** Select the model, then run if installed, else open the install gate. */
  start: (op: string, params: AIParamsInput, input: File) => void;
  /** Install the gated model (durable job), then run. */
  installAndRun: () => void;
  cancel: () => void;
  retry: () => void;
  /** Clear a terminal/idle state back to idle (keeps the model badge). */
  dismiss: () => void;
}

const IDLE_PROGRESS: EnhanceProgress = { percent: 0, message: "", state: "unspecified" };

const errorText = (err: unknown): string =>
  err instanceof Error ? err.message : String(err);

/**
 * Drives one AI enhancement op through the durable job lifecycle: select model
 * → (install gate) → submit → watch progress → fetch the result blob → hand it
 * back via `onResult` (which composes it as the next Workspace step). All
 * mutating entry points are user-triggered handlers (not effects), each guarded
 * by a run-id + an AbortController so a cancel / unmount / superseding run can
 * never let a stale completion land. The client seam is injected for tests.
 */
export function useEnhance({ onResult, client = liveEnhanceClient }: UseEnhanceArgs): UseEnhance {
  const [phase, setPhase] = useState<EnhancePhase>("idle");
  const [model, setModel] = useState<SelectedModel | null>(null);
  const [progress, setProgress] = useState<EnhanceProgress>(IDLE_PROGRESS);
  const [tier, setTier] = useState("");
  const [warnings, setWarnings] = useState<string[]>([]);
  const [error, setError] = useState<string | null>(null);

  const mounted = useRef(true);
  const runId = useRef(0);
  const previewId = useRef(0);
  const abortRef = useRef<AbortController | null>(null);
  const jobIdRef = useRef("");
  const pending = useRef<{ op: string; params: AIParamsInput; input: File } | null>(null);

  useEffect(
    () => () => {
      mounted.current = false;
      abortRef.current?.abort();
    },
    [],
  );

  const live = useCallback((id: number) => mounted.current && runId.current === id, []);

  const runJob = useCallback(
    async (op: string, params: AIParamsInput, input: File, id: number) => {
      setPhase("submitting");
      setProgress(IDLE_PROGRESS);
      setError(null);
      setWarnings([]);

      let submitted: { jobId: string; tier: string; warnings: string[] };
      try {
        submitted = await client.submit(op, params, input);
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
      jobIdRef.current = submitted.jobId;
      setTier(submitted.tier);
      setWarnings(submitted.warnings);
      setPhase("running");

      const controller = new AbortController();
      abortRef.current = controller;
      await client.watch(submitted.jobId, controller.signal, (event) => {
        if (live(id)) {
          setProgress(event);
        }
      });
      if (!live(id) || controller.signal.aborted) {
        return;
      }

      let terminal: { ok: boolean; resultRef: string; error: string };
      try {
        terminal = await client.result(submitted.jobId);
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
      if (!terminal.ok) {
        setError(terminal.error || null);
        setPhase("failed");
        return;
      }

      try {
        const image = await client.fetchResult(terminal.resultRef);
        if (!live(id)) {
          return;
        }
        onResult({
          op,
          result: {
            kind: "image",
            url: image.url,
            width: image.width,
            height: image.height,
            format: image.format,
            jobId: submitted.jobId,
          },
          outputFile: image.outputFile,
        });
        setPhase("succeeded");
      } catch (err) {
        if (live(id)) {
          setError(errorText(err));
          setPhase("failed");
        }
      }
    },
    [client, live, onResult],
  );

  const start = useCallback(
    (op: string, params: AIParamsInput, input: File) => {
      const id = ++runId.current;
      abortRef.current?.abort();
      pending.current = { op, params, input };
      setError(null);
      setPhase("submitting");

      void (async () => {
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
        await runJob(op, params, input, id);
      })();
    },
    [client, live, runJob],
  );

  const installAndRun = useCallback(() => {
    const job = pending.current;
    const target = model;
    if (!job || !target) {
      return;
    }
    const id = ++runId.current;
    abortRef.current?.abort();
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
      await runJob(job.op, job.params, job.input, id);
    })();
  }, [client, live, model, runJob]);

  const preview = useCallback(
    (op: string) => {
      const id = ++previewId.current;
      void (async () => {
        try {
          const selected = await client.selectModel(op);
          if (mounted.current && previewId.current === id) {
            setModel(selected);
          }
        } catch {
          // The badge is optional; surface failures only when the user runs.
        }
      })();
    },
    [client],
  );

  const cancel = useCallback(() => {
    const jobId = jobIdRef.current;
    runId.current += 1;
    abortRef.current?.abort();
    if (jobId) {
      void client.cancel(jobId).catch(() => {
        // Best-effort; the server-owned job is durable regardless.
      });
    }
    setPhase("idle");
    setProgress(IDLE_PROGRESS);
  }, [client]);

  const retry = useCallback(() => {
    const job = pending.current;
    if (job) {
      start(job.op, job.params, job.input);
    }
  }, [start]);

  const dismiss = useCallback(() => {
    runId.current += 1;
    abortRef.current?.abort();
    setPhase("idle");
    setError(null);
    setProgress(IDLE_PROGRESS);
  }, []);

  return {
    phase,
    model,
    progress,
    tier,
    warnings,
    error,
    preview,
    start,
    installAndRun,
    cancel,
    retry,
    dismiss,
  };
}
