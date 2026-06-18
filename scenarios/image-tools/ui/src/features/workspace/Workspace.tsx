import { useCallback, useEffect, useMemo } from "react";
import { useQuery } from "@tanstack/react-query";

import { selectors } from "../../consts/selectors";
import { listOperations, type OperationInfo } from "../../api/ops";
import { SpatialGroup } from "../../hooks/SpatialGroup";
import { useSpatialNav } from "../../hooks/useSpatialNav";
import { useKeyboardShortcuts } from "../../hooks/useKeyboardShortcuts";
import { CanvasActionBar } from "./CanvasActionBar";
import { HistoryRail } from "./HistoryRail";
import { Inspector } from "./Inspector";
import { ModeSwitcher } from "./ModeSwitcher";
import { OP_SPECS, opSpec } from "./opSpecs";
import { useWorkspace, type WorkspaceRunner } from "./useWorkspace";
import { WorkspaceCanvas } from "./WorkspaceCanvas";

const OPS_QUERY_KEY = ["operations"] as const;

export interface WorkspaceProps {
  /** Op-execution seam; defaults to the live REST runner. Injected in tests. */
  runner?: WorkspaceRunner;
}

/**
 * The unified Workspace surface: one image, one history, four modes. The
 * canvas is the work surface (registered as a spatial `passthrough` group so
 * D-pad/arrow input flows to it), the right Inspector is mode-aware, and the
 * left rail shows the non-destructive history. Replaces the retired two-card
 * editor surface.
 */
export function Workspace({ runner }: WorkspaceProps = {}) {
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
            />
          </div>
        </SpatialGroup>
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
          onSelectOperation={ws.setOperation}
          onParam={ws.setParam}
          onApply={(overlay) => void ws.apply(overlay)}
        />
      </div>
    </div>
  );
}
