import { useCallback, useEffect, useRef, useState } from "react";

import {
  fetchAIResult,
  submitAI,
  type AIImageResult,
  type AIParamsInput,
} from "../../api/ai";
import { ApiError } from "../../api/client";
import { JobState, jobsClient } from "../../api/jobs";
import type { RunOpImageResult } from "../../api/ops";
import { parseVariationKeys } from "./createCatalog";
import {
  isEnhanceActive,
  liveEnhanceClient,
  type EnhancePhase,
  type EnhanceProgress,
  type SelectedModel,
} from "./useEnhance";

/**
 * Create reuses the same durable-job lifecycle phases as Enhance — the only
 * differences are that the input image is optional (text-to-image takes none)
 * and a run yields N variations instead of one. Aliasing keeps the phase
 * vocabulary single-sourced rather than duplicated.
 */
export type CreatePhase = EnhancePhase;
export const isCreateActive = isEnhanceActive;

/** One generated variation, materialized from its result blob. */
export interface CreateVariation {
  /** 0-based position in the requested set. */
  index: number;
  result: RunOpImageResult;
  /** The bytes as a File so "send to canvas" can adopt it as a new document. */
  outputFile: File;
}

/**
 * The injected seam the hook drives. Mirrors `EnhanceClient` but `submit`
 * accepts an optional input image + mask (text-to-image has neither) and
 * `result` also returns the terminal job message (which carries the extra
 * variation blob keys). The shared methods reuse `liveEnhanceClient` so there
 * is one production implementation of each.
 */
export interface CreateClient {
  selectModel: (op: string) => Promise<SelectedModel>;
  submit: (
    op: string,
    params: AIParamsInput,
    input?: File,
    mask?: File,
  ) => Promise<{ jobId: string; tier: string; warnings: string[] }>;
  watch: (
    jobId: string,
    signal: AbortSignal,
    onEvent: (event: EnhanceProgress) => void,
  ) => Promise<void>;
  result: (
    jobId: string,
  ) => Promise<{ ok: boolean; resultRef: string; message: string; error: string }>;
  fetchResult: (resultRef: string) => Promise<AIImageResult>;
  install: (modelId: string) => Promise<{ jobId: string; alreadyInstalled: boolean }>;
  waitJob: (jobId: string) => Promise<{ ok: boolean; error: string }>;
  cancel: (jobId: string) => Promise<void>;
}

/** Production client: the AI submit edge + Jobs/Models clients (shared with Enhance). */
export const liveCreateClient: CreateClient = {
  selectModel: liveEnhanceClient.selectModel,
  watch: liveEnhanceClient.watch,
  install: liveEnhanceClient.install,
  waitJob: liveEnhanceClient.waitJob,
  cancel: liveEnhanceClient.cancel,
  fetchResult: (resultRef) => fetchAIResult(resultRef),
  async submit(op, params, input, mask) {
    const res = await submitAI(op, params, { image: input, mask });
    return { jobId: res.jobId, tier: res.tier, warnings: res.warnings };
  },
  async result(jobId) {
    const resp = await jobsClient.getJob({ id: jobId });
    const job = resp.job;
    return {
      ok: job?.state === JobState.SUCCEEDED,
      resultRef: job?.resultRef ?? "",
      message: job?.message ?? "",
      error: job?.error ?? "",
    };
  },
};

const IDLE_PROGRESS: EnhanceProgress = { percent: 0, message: "", state: "unspecified" };

const errorText = (err: unknown): string =>
  err instanceof Error ? err.message : String(err);

interface PendingRun {
  op: string;
  params: AIParamsInput;
  input?: File;
  mask?: File;
}

export interface UseCreateArgs {
  client?: CreateClient;
}

export interface UseCreate {
  phase: CreatePhase;
  model: SelectedModel | null;
  progress: EnhanceProgress;
  tier: string;
  warnings: string[];
  error: string | null;
  /**
   * True when the last submit was rejected by the public-tier consent gate
   * (HTTP 403). The panel surfaces the server's message and directs the user to
   * the consent checkbox.
   */
  consentBlocked: boolean;
  /** The variations produced by the last successful run (empty until then). */
  results: CreateVariation[];
  /** How many variations the in-flight run requested (drives skeleton slots). */
  requestedCount: number;
  /** Fetch the op's selected-model badge without running (called on op select). */
  preview: (op: string) => void;
  /** Select the model, then run if installed, else open the install gate. */
  start: (op: string, params: AIParamsInput, input?: File, mask?: File) => void;
  installAndRun: () => void;
  cancel: () => void;
  retry: () => void;
  /** Clear results + terminal state back to idle (keeps the model badge). */
  dismiss: () => void;
}

/**
 * Drives one generation op through the durable job lifecycle: select model →
 * (install gate) → submit → watch progress → fetch every variation blob →
 * expose them as `results` for the grid. Like `useEnhance`, every mutating
 * entry point is a user-triggered handler guarded by a run-id + AbortController
 * so a cancel / unmount / superseding run can never let a stale set of
 * variations land. The client seam is injected for tests.
 */
export function useCreate({ client = liveCreateClient }: UseCreateArgs = {}): UseCreate {
  const [phase, setPhase] = useState<CreatePhase>("idle");
  const [model, setModel] = useState<SelectedModel | null>(null);
  const [progress, setProgress] = useState<EnhanceProgress>(IDLE_PROGRESS);
  const [tier, setTier] = useState("");
  const [warnings, setWarnings] = useState<string[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [consentBlocked, setConsentBlocked] = useState(false);
  const [results, setResults] = useState<CreateVariation[]>([]);
  const [requestedCount, setRequestedCount] = useState(1);

  const mounted = useRef(true);
  const runId = useRef(0);
  const previewId = useRef(0);
  const abortRef = useRef<AbortController | null>(null);
  const jobIdRef = useRef("");
  const pending = useRef<PendingRun | null>(null);

  useEffect(
    () => () => {
      mounted.current = false;
      abortRef.current?.abort();
    },
    [],
  );

  const live = useCallback((id: number) => mounted.current && runId.current === id, []);

  const runJob = useCallback(
    async (run: PendingRun, id: number) => {
      setPhase("submitting");
      setProgress(IDLE_PROGRESS);
      setError(null);
      setConsentBlocked(false);
      setWarnings([]);
      setResults([]);

      let submitted: { jobId: string; tier: string; warnings: string[] };
      try {
        submitted = await client.submit(run.op, run.params, run.input, run.mask);
      } catch (err) {
        if (live(id)) {
          setError(errorText(err));
          setConsentBlocked(err instanceof ApiError && err.status === 403);
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

      let terminal: { ok: boolean; resultRef: string; message: string; error: string };
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

      const keys = parseVariationKeys(terminal.message, terminal.resultRef);
      try {
        const images = await Promise.all(keys.map((key) => client.fetchResult(key)));
        if (!live(id)) {
          return;
        }
        const variations: CreateVariation[] = images.map((image, index) => ({
          index,
          result: {
            kind: "image",
            url: image.url,
            width: image.width,
            height: image.height,
            format: image.format,
            jobId: submitted.jobId,
          },
          outputFile: image.outputFile,
        }));
        setResults(variations);
        setPhase("succeeded");
      } catch (err) {
        if (live(id)) {
          setError(errorText(err));
          setPhase("failed");
        }
      }
    },
    [client, live],
  );

  const start = useCallback(
    (op: string, params: AIParamsInput, input?: File, mask?: File) => {
      const id = ++runId.current;
      abortRef.current?.abort();
      const run: PendingRun = { op, params, input, mask };
      pending.current = run;
      setRequestedCount(Math.max(1, params.variations ?? 1));
      setError(null);
      setConsentBlocked(false);
      setResults([]);
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
        await runJob(run, id);
      })();
    },
    [client, live, runJob],
  );

  const installAndRun = useCallback(() => {
    const run = pending.current;
    const target = model;
    if (!run || !target) {
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
      await runJob(run, id);
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
    const run = pending.current;
    if (run) {
      start(run.op, run.params, run.input, run.mask);
    }
  }, [start]);

  const dismiss = useCallback(() => {
    runId.current += 1;
    abortRef.current?.abort();
    setPhase("idle");
    setError(null);
    setConsentBlocked(false);
    setResults([]);
    setProgress(IDLE_PROGRESS);
  }, []);

  return {
    phase,
    model,
    progress,
    tier,
    warnings,
    error,
    consentBlocked,
    results,
    requestedCount,
    preview,
    start,
    installAndRun,
    cancel,
    retry,
    dismiss,
  };
}
