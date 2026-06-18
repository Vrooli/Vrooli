import { useCallback, useEffect, useState } from "react";

import {
  runOp,
  type OpParamValues,
  type RunOpImageResult,
  type RunOpResult,
} from "../../api/ops";
import { defaultParamsFor, toRequestParams } from "./opParams";
import { opSpec } from "./opSpecs";

/**
 * The four things you can do to the loaded image. The Workspace spine treats
 * the image as the document and these as modes it flows across. Stage 0b wires
 * `edit` (deterministic ops); the AI modes arrive in later stages.
 */
export type WorkspaceMode = "edit" | "enhance" | "create" | "analyze";

export const WORKSPACE_MODES = ["edit", "enhance", "create", "analyze"] as const;

/** The base (unedited) source image. */
export interface BaseImage {
  file: File;
  url: string;
}

/** A single applied, non-destructive edit step. */
export interface AppliedOp {
  operation: string;
  params: OpParamValues;
  /** The image result of running this op on the previous step's output. */
  result: RunOpImageResult;
  /** The op output materialized as a File so the next op can consume it. */
  outputFile: File;
}

/**
 * The normalized result of running one op: an image (composable — carries the
 * output as a File so the next op stacks on top) or a metadata read (no image
 * change). Keeping both kinds in one runner seam lets the store stay pure and
 * unit-testable without the network.
 */
export type WorkspaceRunResult =
  | { kind: "image"; result: RunOpImageResult; outputFile: File }
  | { kind: "metadata"; json: string };

export type WorkspaceRunner = (
  operation: string,
  params: OpParamValues,
  input: File,
  opts?: { overlay?: File },
) => Promise<WorkspaceRunResult>;

/**
 * Production runner: execute the op via the REST ops edge, then — for image
 * results — materialize the bytes as a File so the next op composes headlessly.
 * Metadata reads pass straight through.
 */
export const liveRunner: WorkspaceRunner = async (operation, params, input, opts) => {
  const result: RunOpResult = await runOp(operation, input, toRequestParams(params), opts ?? {});
  if (result.kind === "metadata") {
    return { kind: "metadata", json: result.json };
  }
  const blob = await fetch(result.url).then((r) => r.blob());
  const outputFile = new File([blob], `step.${result.format || "png"}`, {
    type: blob.type || "image/png",
  });
  return { kind: "image", result, outputFile };
};

const EMPTY_PARAMS: OpParamValues = {};

export interface Workspace {
  base: BaseImage | null;
  entries: AppliedOp[];
  applying: boolean;
  error: unknown;
  canUndo: boolean;
  canRedo: boolean;
  mode: WorkspaceMode;
  operation: string;
  params: OpParamValues;
  /** URL of the current (post-last-op) image, or the base image, or null. */
  previewUrl: string | null;
  /** JSON of the most recent metadata read, cleared on the next image op. */
  metadata: string | null;
  /** The top-of-stack image result (drives the result meta + download). */
  currentResult: RunOpImageResult | null;
  setBase: (file: File) => void;
  setMode: (mode: WorkspaceMode) => void;
  setOperation: (operation: string) => void;
  setParam: (name: string, value: string | number | boolean) => void;
  /** Run the current operation+params against the latest output. */
  apply: (overlay?: File) => Promise<void>;
  undo: () => void;
  redo: () => void;
  reset: () => void;
}

/**
 * The unified Workspace store: one source image, an ordered non-destructive
 * step stack, the active mode, and the selected operation + params. Applying an
 * image op composes it onto the latest output and pushes a step; a metadata
 * read shows JSON without touching the stack. Undo/redo move cached steps
 * between the live stack and a redo stack — they never recompute, so undo
 * cannot corrupt earlier results. The runner is injected so the store is unit-
 * testable without the network.
 */
export function useWorkspace(runner: WorkspaceRunner = liveRunner): Workspace {
  const [base, setBaseState] = useState<BaseImage | null>(null);
  const [entries, setEntries] = useState<AppliedOp[]>([]);
  const [redoStack, setRedoStack] = useState<AppliedOp[]>([]);
  const [applying, setApplying] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const [mode, setMode] = useState<WorkspaceMode>("edit");
  const [operation, setOperationState] = useState<string>("");
  const [params, setParams] = useState<OpParamValues>(EMPTY_PARAMS);
  const [metadata, setMetadata] = useState<string | null>(null);

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
    setMetadata(null);
  }, []);

  const setOperation = useCallback((next: string) => {
    setOperationState(next);
    setParams(defaultParamsFor(next));
  }, []);

  const setParam = useCallback(
    (name: string, value: string | number | boolean) =>
      setParams((prev) => ({ ...prev, [name]: value })),
    [],
  );

  const apply = useCallback(
    async (overlay?: File) => {
      const input = entries.length > 0 ? entries[entries.length - 1]?.outputFile : base?.file;
      if (!input || !operation) {
        return;
      }
      const spec = opSpec(operation);
      setApplying(true);
      setError(null);
      try {
        const run = await runner(operation, params, input, spec?.acceptsOverlay ? { overlay } : {});
        if (run.kind === "metadata") {
          setMetadata(run.json);
        } else {
          setMetadata(null);
          setEntries((prev) => [
            ...prev,
            { operation, params, result: run.result, outputFile: run.outputFile },
          ]);
          setRedoStack([]);
        }
      } catch (err) {
        setError(err);
      } finally {
        setApplying(false);
      }
    },
    [base, entries, operation, params, runner],
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
    setMetadata(null);
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
    setMetadata(null);
  }, []);

  const currentEntry = entries.length > 0 ? entries[entries.length - 1] ?? null : null;
  const previewUrl = currentEntry ? currentEntry.result.url : base?.url ?? null;

  return {
    base,
    entries,
    applying,
    error,
    canUndo: entries.length > 0,
    canRedo: redoStack.length > 0,
    mode,
    operation,
    params,
    previewUrl,
    metadata,
    currentResult: currentEntry?.result ?? null,
    setBase,
    setMode,
    setOperation,
    setParam,
    apply,
    undo,
    redo,
    reset,
  };
}
