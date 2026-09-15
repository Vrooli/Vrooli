import { useCallback, useEffect, useRef, useState } from "react";

import { blobUrl } from "../../api/client";
import { DiffMode, compare, type DiffResult } from "../../api/diff";

/** Which input slot a chosen file belongs to. */
export type CompareSlot = "base" | "compare";

/** Lifecycle of the comparison surface. */
export type ComparePhase = "idle" | "comparing" | "ready";

/**
 * The injected seam the hook drives. The live impl wraps the diff compare edge
 * + the blob store; tests pass a fake so the whole flow runs without the
 * network.
 */
export interface CompareClient {
  /** Run a comparison of two images. */
  compare: typeof compare;
  /** Resolve a blob key to a URL for the heat-map <img>. */
  blobUrl: (key: string) => string;
}

/** Production client: the diff compare edge + the blob store. */
export const liveCompareClient: CompareClient = {
  compare,
  blobUrl,
};

export interface UseCompareResult {
  phase: ComparePhase;
  base: File | null;
  baseUrl: string | null;
  compareImage: File | null;
  compareUrl: string | null;
  mode: DiffMode;
  tolerance: number;
  result: DiffResult | null;
  heatmapUrl: string | null;
  error: string | null;
  /** True once both slots hold an image (the Compare action is enabled). */
  canCompare: boolean;
  setImage: (slot: CompareSlot, file: File) => void;
  setMode: (mode: DiffMode) => void;
  setTolerance: (value: number) => void;
  runCompare: () => void;
  reset: () => void;
}

/**
 * Drives the visual-comparison pipeline: pick a base + compare image, choose a
 * mode (pixel/perceptual) + tolerance, then run the compare edge and expose the
 * verdict + metrics + heat-map. A run-id + AbortController guard (mirroring the
 * select hook's rigor) means a superseded run never clobbers a newer one: only
 * the latest `runCompare` may commit state, and an in-flight request is aborted
 * when a newer run starts or the surface resets/unmounts.
 */
export function useCompare(client: CompareClient = liveCompareClient): UseCompareResult {
  const [phase, setPhase] = useState<ComparePhase>("idle");
  const [base, setBaseState] = useState<File | null>(null);
  const [baseUrl, setBaseUrl] = useState<string | null>(null);
  const [compareImage, setCompareState] = useState<File | null>(null);
  const [compareUrl, setCompareUrl] = useState<string | null>(null);
  const [mode, setMode] = useState<DiffMode>(DiffMode.PIXEL);
  const [tolerance, setTolerance] = useState(0);
  const [result, setResult] = useState<DiffResult | null>(null);
  const [error, setError] = useState<string | null>(null);

  // Track created object URLs so they're revoked on replace/unmount.
  const baseUrlRef = useRef<string | null>(null);
  const compareUrlRef = useRef<string | null>(null);
  // Monotonic run id + the in-flight controller: a superseded run is ignored.
  const runIdRef = useRef(0);
  const abortRef = useRef<AbortController | null>(null);

  useEffect(
    () => () => {
      if (baseUrlRef.current) {
        URL.revokeObjectURL(baseUrlRef.current);
      }
      if (compareUrlRef.current) {
        URL.revokeObjectURL(compareUrlRef.current);
      }
      abortRef.current?.abort();
    },
    [],
  );

  const heatmapUrl = result?.heatmapRef ? client.blobUrl(result.heatmapRef) : null;

  const setImage = useCallback((slot: CompareSlot, file: File) => {
    const url = URL.createObjectURL(file);
    if (slot === "base") {
      if (baseUrlRef.current) {
        URL.revokeObjectURL(baseUrlRef.current);
      }
      baseUrlRef.current = url;
      setBaseState(file);
      setBaseUrl(url);
    } else {
      if (compareUrlRef.current) {
        URL.revokeObjectURL(compareUrlRef.current);
      }
      compareUrlRef.current = url;
      setCompareState(file);
      setCompareUrl(url);
    }
    // Any new input invalidates the prior verdict.
    setResult(null);
    setError(null);
    setPhase("idle");
  }, []);

  const runCompare = useCallback(() => {
    if (!base || !compareImage) {
      return;
    }
    abortRef.current?.abort();
    const controller = new AbortController();
    abortRef.current = controller;
    const runId = ++runIdRef.current;
    setPhase("comparing");
    setError(null);
    void (async () => {
      try {
        const res = await client.compare({ base, compare: compareImage, mode, tolerance });
        // A newer run (or a reset) superseded this one — drop the result.
        if (runId !== runIdRef.current || controller.signal.aborted) {
          return;
        }
        setResult(res);
        setPhase("ready");
      } catch (err) {
        if (runId !== runIdRef.current || controller.signal.aborted) {
          return;
        }
        setError(err instanceof Error ? err.message : String(err));
        setPhase("idle");
      }
    })();
  }, [base, client, compareImage, mode, tolerance]);

  const reset = useCallback(() => {
    abortRef.current?.abort();
    runIdRef.current += 1;
    setResult(null);
    setError(null);
    setPhase("idle");
  }, []);

  return {
    phase,
    base,
    baseUrl,
    compareImage,
    compareUrl,
    mode,
    tolerance,
    result,
    heatmapUrl,
    error,
    canCompare: base !== null && compareImage !== null,
    setImage,
    setMode,
    setTolerance,
    runCompare,
    reset,
  };
}
