import { useCallback, useEffect, useRef, useState } from "react";

import { ApiError, blobUrl, fetchBlob } from "../../api/client";
import { submitAI } from "../../api/ai";
import { SegmentMode, segment, type SegmentResult, type SuggestedEdit } from "../../api/selection";

/** Lifecycle of the smart-select surface. */
export type SmartSelectPhase = "idle" | "segmenting" | "ready" | "applying" | "submitted";

/** The injected seam the hook drives. The live impl wraps the selection + AI
 * edges + blob store; tests pass a fake so the whole flow runs without the
 * network. */
export interface SmartSelectClient {
  /** Run a segmentation against the image (point/box/auto). */
  segment: typeof segment;
  /** Fetch a stored blob (the produced mask) so it can ride the AI submit. */
  fetchBlob: (key: string) => Promise<Blob>;
  /** Resolve a blob key to a URL for the overlay <img>. */
  blobUrl: (key: string) => string;
  /** Submit a masked (or whole-image) AI op for the chosen edit. */
  submitAI: typeof submitAI;
}

/** Production client: the selection + AI edges + the blob store. */
export const liveSmartSelectClient: SmartSelectClient = {
  segment,
  fetchBlob,
  blobUrl,
  submitAI,
};

/** Outcome of applying a contextual edit. */
export interface ApplyOutcome {
  kind: "submitted" | "gated" | "error";
  jobId?: string;
  tier?: string;
  warnings?: string[];
  message?: string;
}

export interface UseSmartSelectResult {
  phase: SmartSelectPhase;
  image: File | null;
  imageUrl: string | null;
  result: SegmentResult | null;
  maskUrl: string | null;
  error: string | null;
  outcome: ApplyOutcome | null;
  tolerance: number;
  setImage: (file: File) => void;
  setTolerance: (value: number) => void;
  selectPoint: (nx: number, ny: number) => void;
  selectAuto: () => void;
  applyEdit: (edit: SuggestedEdit, promptText: string) => void;
  reset: () => void;
}

/**
 * Drives the smart-select pipeline: load an image, click/auto a region
 * (segment + classify in one round-trip), then apply a contextual edit (a
 * masked AI submit). The mask the edge produces is re-uploaded as the AI op's
 * `mask` part for masked edits; whole-image edits (background_removal) skip it.
 *
 * A 409 from the AI submit (no backend/model) becomes a `gated` outcome with an
 * actionable message rather than an opaque failure.
 */
export function useSmartSelect(client: SmartSelectClient = liveSmartSelectClient): UseSmartSelectResult {
  const [phase, setPhase] = useState<SmartSelectPhase>("idle");
  const [image, setImageState] = useState<File | null>(null);
  const [imageUrl, setImageUrl] = useState<string | null>(null);
  const [result, setResult] = useState<SegmentResult | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [outcome, setOutcome] = useState<ApplyOutcome | null>(null);
  const [tolerance, setTolerance] = useState(0);

  // Track the created object URL so it can be revoked on replace/unmount.
  const objectUrlRef = useRef<string | null>(null);
  useEffect(
    () => () => {
      if (objectUrlRef.current) {
        URL.revokeObjectURL(objectUrlRef.current);
      }
    },
    [],
  );

  const maskUrl = result?.maskRef ? client.blobUrl(result.maskRef) : null;

  const setImage = useCallback((file: File) => {
    if (objectUrlRef.current) {
      URL.revokeObjectURL(objectUrlRef.current);
    }
    const url = URL.createObjectURL(file);
    objectUrlRef.current = url;
    setImageState(file);
    setImageUrl(url);
    setResult(null);
    setError(null);
    setOutcome(null);
    setPhase("idle");
  }, []);

  const runSegment = useCallback(
    async (input: Parameters<typeof segment>[0]) => {
      if (!image) {
        return;
      }
      setPhase("segmenting");
      setError(null);
      setOutcome(null);
      try {
        const res = await client.segment(input);
        setResult(res);
        setPhase("ready");
      } catch (err) {
        setError(err instanceof Error ? err.message : String(err));
        setPhase("idle");
      }
    },
    [client, image],
  );

  const selectPoint = useCallback(
    (nx: number, ny: number) => {
      if (!image) {
        return;
      }
      void runSegment({
        image,
        mode: SegmentMode.POINT,
        points: [{ x: clamp01(nx), y: clamp01(ny) }],
        tolerance,
      });
    },
    [image, runSegment, tolerance],
  );

  const selectAuto = useCallback(() => {
    if (!image) {
      return;
    }
    void runSegment({ image, mode: SegmentMode.AUTO, tolerance });
  }, [image, runSegment, tolerance]);

  const applyEdit = useCallback(
    (edit: SuggestedEdit, promptText: string) => {
      if (!image || !result) {
        return;
      }
      const prompt = edit.requiresPrompt ? `${edit.prompt}${promptText}`.trim() : edit.prompt;
      void (async () => {
        setPhase("applying");
        setOutcome(null);
        try {
          let mask: File | undefined;
          if (edit.requiresMask && result.maskRef) {
            const blob = await client.fetchBlob(result.maskRef);
            mask = new File([blob], "mask.png", { type: "image/png" });
          }
          const submit = await client.submitAI(edit.operation, { prompt }, { image, mask });
          setOutcome({
            kind: "submitted",
            jobId: submit.jobId,
            tier: submit.tier,
            warnings: submit.warnings,
          });
          setPhase("submitted");
        } catch (err) {
          if (err instanceof ApiError && err.status === 409) {
            setOutcome({ kind: "gated", message: err.message });
          } else {
            setOutcome({ kind: "error", message: err instanceof Error ? err.message : String(err) });
          }
          setPhase("ready");
        }
      })();
    },
    [client, image, result],
  );

  const reset = useCallback(() => {
    setResult(null);
    setError(null);
    setOutcome(null);
    setPhase(image ? "idle" : "idle");
  }, [image]);

  return {
    phase,
    image,
    imageUrl,
    result,
    maskUrl,
    error,
    outcome,
    tolerance,
    setImage,
    setTolerance,
    selectPoint,
    selectAuto,
    applyEdit,
    reset,
  };
}

function clamp01(v: number): number {
  if (v < 0) return 0;
  if (v > 1) return 1;
  return v;
}
