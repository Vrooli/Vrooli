import { useCallback, useEffect, useState } from "react";

import type { OpParamValues, RunOpImageResult } from "../../api/ops";

/** A single applied, non-destructive edit step. */
export interface AppliedOp {
  operation: string;
  params: OpParamValues;
  /** The image result of running this op on the previous step's output. */
  result: RunOpImageResult;
  /** The op output materialized as a File so the next op can consume it. */
  outputFile: File;
}

/** The base (unedited) source image. */
export interface BaseImage {
  file: File;
  url: string;
}

/**
 * Runs one op against `input` and returns both the image result and the output
 * materialized as a File (so subsequent ops compose headlessly). Injected so
 * the stack logic is pure and unit-testable without the network.
 */
export type OpRunner = (
  operation: string,
  params: OpParamValues,
  input: File,
) => Promise<{ result: RunOpImageResult; outputFile: File }>;

export interface OpStack {
  base: BaseImage | null;
  entries: AppliedOp[];
  applying: boolean;
  error: unknown;
  canUndo: boolean;
  canRedo: boolean;
  /** URL of the current (post-last-op) preview, or the base image. */
  previewUrl: string | null;
  setBase: (file: File) => void;
  apply: (operation: string, params: OpParamValues) => Promise<void>;
  undo: () => void;
  redo: () => void;
  reset: () => void;
}

/**
 * Non-destructive op-stack over the deterministic ops. The base image is never
 * mutated; applying an op runs it against the latest output and pushes a step.
 * Undo/redo move steps between the live stack and a redo stack — they reuse the
 * cached per-step results rather than recomputing, so undo cannot corrupt
 * earlier outputs. The editor merely composes API calls; everything here is
 * expressible headlessly.
 */
export function useOpStack(runner: OpRunner): OpStack {
  const [base, setBaseState] = useState<BaseImage | null>(null);
  const [entries, setEntries] = useState<AppliedOp[]>([]);
  const [redoStack, setRedoStack] = useState<AppliedOp[]>([]);
  const [applying, setApplying] = useState(false);
  const [error, setError] = useState<unknown>(null);

  // Revoke the base object URL when it is replaced or on unmount.
  useEffect(
    () => () => {
      if (base) {
        URL.revokeObjectURL(base.url);
      }
    },
    [base],
  );

  const setBase = useCallback((file: File) => {
    setBaseState((prev) => {
      if (prev) {
        URL.revokeObjectURL(prev.url);
      }
      return { file, url: URL.createObjectURL(file) };
    });
    setEntries([]);
    setRedoStack([]);
    setError(null);
  }, []);

  const apply = useCallback(
    async (operation: string, params: OpParamValues) => {
      const input = entries.length > 0 ? entries[entries.length - 1]?.outputFile : base?.file;
      if (!input) {
        return;
      }
      setApplying(true);
      setError(null);
      try {
        const { result, outputFile } = await runner(operation, params, input);
        setEntries((prev) => [...prev, { operation, params, result, outputFile }]);
        setRedoStack([]);
      } catch (err) {
        setError(err);
      } finally {
        setApplying(false);
      }
    },
    [base, entries, runner],
  );

  const undo = useCallback(() => {
    setEntries((prev) => {
      if (prev.length === 0) {
        return prev;
      }
      const last = prev[prev.length - 1];
      if (last) {
        setRedoStack((r) => [last, ...r]);
      }
      return prev.slice(0, -1);
    });
  }, []);

  const redo = useCallback(() => {
    setRedoStack((prev) => {
      if (prev.length === 0) {
        return prev;
      }
      const [next, ...rest] = prev;
      if (next) {
        setEntries((e) => [...e, next]);
      }
      return rest;
    });
  }, []);

  const reset = useCallback(() => {
    setEntries([]);
    setRedoStack([]);
    setError(null);
  }, []);

  const previewUrl =
    entries.length > 0 ? entries[entries.length - 1]?.result.url ?? null : base?.url ?? null;

  return {
    base,
    entries,
    applying,
    error,
    canUndo: entries.length > 0,
    canRedo: redoStack.length > 0,
    previewUrl,
    setBase,
    apply,
    undo,
    redo,
    reset,
  };
}
