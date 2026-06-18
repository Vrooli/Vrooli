import { useCallback, useEffect, useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";

import { selectors } from "../../consts/selectors";
import { listOperations, type OperationInfo } from "../../api/ops";
import { SpatialGroup } from "../../hooks/SpatialGroup";
import { useSpatialNav } from "../../hooks/useSpatialNav";
import { useKeyboardShortcuts } from "../../hooks/useKeyboardShortcuts";
import { CanvasActionBar } from "./CanvasActionBar";
import { CreatePanel } from "./CreatePanel";
import { fullImageRect, type Rect, type Size } from "./cropMath";
import { EnhancePanel } from "./EnhancePanel";
import { HistoryRail } from "./HistoryRail";
import { Inspector } from "./Inspector";
import { ModeSwitcher } from "./ModeSwitcher";
import { OP_SPECS, opSpec } from "./opSpecs";
import {
  isEnhanceActive,
  useEnhance,
  type EnhanceClient,
  type EnhanceResult,
} from "./useEnhance";
import { useCreate, type CreateClient, type CreateVariation } from "./useCreate";
import { useWorkspace, type WorkspaceRunner } from "./useWorkspace";
import { takeWorkspaceIntent } from "./workspaceIntent";
import { WorkspaceCanvas } from "./WorkspaceCanvas";

/** The crop spec's default rect (x0/y0/w100/h100); seeding overrides it. */
const isDefaultCropRect = (rect: Rect): boolean =>
  rect.x === 0 && rect.y === 0 && rect.width === 100 && rect.height === 100;

const OPS_QUERY_KEY = ["operations"] as const;

export interface WorkspaceProps {
  /** Op-execution seam; defaults to the live REST runner. Injected in tests. */
  runner?: WorkspaceRunner;
  /** AI job-lifecycle seam; defaults to the live client. Injected in tests. */
  enhanceClient?: EnhanceClient;
  /** Create (generation) job-lifecycle seam; defaults to the live client. */
  createClient?: CreateClient;
}

/**
 * The unified Workspace surface: one image, one history, four modes. The
 * canvas is the work surface (registered as a spatial `passthrough` group so
 * D-pad/arrow input flows to it), the right Inspector is mode-aware, and the
 * left rail shows the non-destructive history. Replaces the retired two-card
 * editor surface.
 */
export function Workspace({ runner, enhanceClient, createClient }: WorkspaceProps = {}) {
  const spatialNav = useSpatialNav();
  const ws = useWorkspace(runner);
  const { operation, setOperation, canUndo, canRedo, undo, redo } = ws;

  const opsQuery = useQuery({ queryKey: OPS_QUERY_KEY, queryFn: listOperations });

  const operations: OperationInfo[] = useMemo(
    () => (opsQuery.data?.operations ?? []).filter((op) => op.name in OP_SPECS),
    [opsQuery.data],
  );

  // Pick the first known operation once discovery resolves.
  useEffect(() => {
    if (!operation && operations.length > 0) {
      setOperation(operations[0]?.name ?? "");
    }
  }, [operation, operations, setOperation]);

  const onUndo = useCallback(() => {
    if (canUndo) {
      undo();
      return true;
    }
    return false;
  }, [canUndo, undo]);
  const onRedo = useCallback(() => {
    if (canRedo) {
      redo();
      return true;
    }
    return false;
  }, [canRedo, redo]);
  useKeyboardShortcuts(useMemo(() => ({ onUndo, onRedo }), [onUndo, onRedo]));

  const spec = opSpec(operation);
  const hasSteps = ws.entries.length > 0;
  const downloadName = `result.${ws.currentResult?.format || "png"}`;

  // Crop drag-box: active only when editing with the crop op selected. The rect
  // lives in the same params the Advanced numeric fields drive, so the box and
  // the accessible fallback stay in lockstep. `setParams` merges x/y/w/h in one
  // state change so a drag is a single render, not four.
  const cropActive = ws.mode === "edit" && operation === "crop";
  const cropRect: Rect = useMemo(
    () => ({
      x: Number(ws.params.x ?? 0),
      y: Number(ws.params.y ?? 0),
      width: Number(ws.params.width ?? 0),
      height: Number(ws.params.height ?? 0),
    }),
    [ws.params.x, ws.params.y, ws.params.width, ws.params.height],
  );
  const emitCrop = useCallback(
    (rect: Rect) => ws.setParams({ x: rect.x, y: rect.y, width: rect.width, height: rect.height }),
    [ws],
  );
  const seedCrop = useCallback(
    (natural: Size) => {
      if (isDefaultCropRect(cropRect)) {
        const full = fullImageRect(natural);
        ws.setParams({ x: full.x, y: full.y, width: full.width, height: full.height });
      }
    },
    [cropRect, ws],
  );

  // Enhance (AI) lifecycle. The result composes into the same non-destructive
  // history via `applyImageResult`, then the canvas auto-engages before/after.
  const [compareSignal, setCompareSignal] = useState(0);
  const lastEntry = ws.entries.length > 0 ? ws.entries[ws.entries.length - 1] : undefined;
  const currentInput = lastEntry?.outputFile ?? ws.base?.file ?? null;
  const { applyImageResult } = ws;
  const onEnhanceResult = useCallback(
    ({ op, result, outputFile }: EnhanceResult) => {
      applyImageResult(op, {}, result, outputFile);
      setCompareSignal((s) => s + 1);
    },
    [applyImageResult],
  );
  const enhance = useEnhance({ onResult: onEnhanceResult, client: enhanceClient });
  const enhanceProgress = isEnhanceActive(enhance.phase)
    ? { percent: enhance.progress.percent, message: enhance.progress.message }
    : null;

  // Create (generation) lifecycle. A generated variation is a fresh document,
  // so "send to canvas" adopts it as the new base (resetting the edit stack);
  // "Enhance" does the same and switches modes so the user can upscale it.
  const create = useCreate({ client: createClient });
  const { setBase, setMode } = ws;

  // Apply a one-shot Home / Library handoff (mode + op + starting image) once
  // on mount. Token-keyed so a StrictMode re-mount or a manual /workspace visit
  // never re-applies a stale intent. The store methods are stable useCallbacks.
  // For the AI modes the handed-off op pre-selects the panel's action rather
  // than the deterministic op picker.
  const [initialAiAction, setInitialAiAction] = useState("");
  useEffect(() => {
    const intent = takeWorkspaceIntent();
    if (!intent) {
      return;
    }
    if (intent.mode) {
      setMode(intent.mode);
    }
    if (intent.operation) {
      if (intent.mode === "enhance" || intent.mode === "create") {
        setInitialAiAction(intent.operation);
      } else {
        setOperation(intent.operation);
      }
    }
    if (intent.file) {
      setBase(intent.file);
    }
  }, [setBase, setMode, setOperation]);

  const onSendToCanvas = useCallback(
    (variation: CreateVariation) => {
      setBase(variation.outputFile);
      setMode("edit");
    },
    [setBase, setMode],
  );
  const onSendToEnhance = useCallback(
    (variation: CreateVariation) => {
      setBase(variation.outputFile);
      setMode("enhance");
    },
    [setBase, setMode],
  );

  return (
    <div data-testid={selectors.workspace.root} className="flex flex-col gap-4">
      <ModeSwitcher mode={ws.mode} onModeChange={ws.setMode} />
      <div className="grid gap-4 lg:grid-cols-[15rem_minmax(0,1fr)_20rem]">
        <HistoryRail base={ws.base} entries={ws.entries} />
        <SpatialGroup controllerRef={spatialNav} mode="passthrough">
          <div className="flex min-h-0 flex-col gap-2">
            <CanvasActionBar
              canUndo={ws.canUndo}
              canRedo={ws.canRedo}
              hasSteps={hasSteps}
              downloadUrl={hasSteps ? ws.previewUrl : null}
              downloadName={downloadName}
              onUndo={ws.undo}
              onRedo={ws.redo}
              onReset={ws.reset}
            />
            <WorkspaceCanvas
              baseUrl={ws.base?.url ?? null}
              previewUrl={ws.previewUrl}
              hasSteps={hasSteps}
              currentResult={ws.currentResult}
              metadata={ws.metadata}
              onFile={ws.setBase}
              crop={
                cropActive
                  ? { rect: cropRect, onChange: emitCrop, onNatural: seedCrop }
                  : null
              }
              progress={enhanceProgress}
              compareSignal={compareSignal}
            />
          </div>
        </SpatialGroup>
        {ws.mode === "enhance" ? (
          <EnhancePanel
            enhance={enhance}
            input={currentInput}
            inputWidth={ws.currentResult?.width ?? 0}
            inputHeight={ws.currentResult?.height ?? 0}
            initialAction={initialAiAction}
          />
        ) : ws.mode === "create" ? (
          <CreatePanel
            create={create}
            input={currentInput}
            inputUrl={ws.previewUrl}
            onSendToCanvas={onSendToCanvas}
            onSendToEnhance={onSendToEnhance}
            initialAction={initialAiAction}
          />
        ) : (
          <Inspector
            mode={ws.mode}
            opsLoading={opsQuery.isLoading}
            opsError={Boolean(opsQuery.error)}
            operations={operations}
            operation={ws.operation}
            params={ws.params}
            spec={spec}
            applying={ws.applying}
            runError={ws.error}
            hasBase={ws.base !== null}
            hasSteps={hasSteps}
            previewUrl={ws.previewUrl}
            onSelectOperation={ws.setOperation}
            onParam={ws.setParam}
            onApply={(overlay) => void ws.apply(overlay)}
          />
        )}
      </div>
    </div>
  );
}
