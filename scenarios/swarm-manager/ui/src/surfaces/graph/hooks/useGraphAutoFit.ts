/**
 * useGraphAutoFit — Viewport auto-fitting, fingerprint tracking, and
 * intent-based viewport restore for the graph canvas.
 *
 * On first data arrival per lens, restores the user's saved viewport *intent*
 * (center on a specific node at a specific zoom). Falls back to fitView if
 * the intent is missing or the node no longer exists. Subsequent changes
 * trigger fitView when autoFitOnChange is enabled.
 */

import { useEffect, useMemo, useRef } from "react";
import type { ReactFlowInstance } from "@xyflow/react";
import type { GraphLens } from "../stores/graph-data-store";
import { useGraphUIStore } from "../stores/graph-ui-store";
import type { GraphEdge, GraphNode } from "../types";

export interface UseGraphAutoFitOptions {
  flowRef: React.RefObject<ReactFlowInstance<GraphNode> | null>;
  lens: GraphLens;
  layoutMode: string;
  layoutDirection: string;
  showSecondaryEdges: boolean;
  autoFitOnChange: boolean;
  processedNodes: GraphNode[];
  processedEdges: GraphEdge[];
  styledNodesLength: number;
}

/**
 * Manages auto-fit-on-change behavior and manual fitView nonce triggers.
 *
 * On the first data arrival per lens, attempts to restore the saved viewport
 * intent before falling back to fitView.
 */
export function useGraphAutoFit({
  flowRef,
  lens,
  layoutMode,
  layoutDirection,
  showSecondaryEdges,
  autoFitOnChange,
  processedNodes,
  processedEdges,
  styledNodesLength,
}: UseGraphAutoFitOptions): void {
  const fitViewNonce = useGraphUIStore((s) => s.fitViewNonce);

  // Track whether the initial data load has completed for the current lens.
  const initialLoadCompleteRef = useRef(false);
  const initialLoadLensRef = useRef(lens);

  // Reset initial-load tracking when the lens changes.
  if (lens !== initialLoadLensRef.current) {
    initialLoadCompleteRef.current = false;
    initialLoadLensRef.current = lens;
  }

  // PERF: Use a cheap fingerprint instead of JSON.stringify of all IDs.
  const nodeFingerprint = useMemo(
    () => processedNodes.map((n) => n.id).join("\0"),
    [processedNodes],
  );
  const edgeFingerprint = useMemo(
    () => processedEdges.map((e) => e.id).join("\0"),
    [processedEdges],
  );
  const autoFitFingerprint = `${lens}|${layoutMode}|${layoutDirection}|${showSecondaryEdges}|${nodeFingerprint}|${edgeFingerprint}`;

  useEffect(() => {
    if (!flowRef.current || styledNodesLength === 0) {
      return;
    }

    // Restore-from-intent on first data arrival for this lens. Intent is a
    // semantic pointer ({nodeId, zoom}) — durable across container-size,
    // layout-mode, and data-set changes in a way that raw {x, y, zoom} isn't.
    if (!initialLoadCompleteRef.current) {
      initialLoadCompleteRef.current = true;

      const intent = useGraphUIStore.getState().viewportIntentByLens[lens];

      if (intent?.nodeId) {
        // Look up the node in the just-processed data. If it's still there,
        // center on it at the saved zoom. If not (deleted, filtered out, or
        // collapsed inside a cluster), fall through to fitView.
        const target = processedNodes.find((n) => n.id === intent.nodeId);
        if (target) {
          const raf = window.requestAnimationFrame(() => {
            flowRef.current?.setCenter(target.position.x, target.position.y, {
              zoom: intent.zoom,
              duration: 0,
            });
          });
          return () => window.cancelAnimationFrame(raf);
        }
      }

      // No restorable intent — fall through to fitView below.
    } else if (!autoFitOnChange) {
      // Post-initial-load changes only re-fit if the user has enabled it.
      return;
    }

    const raf = window.requestAnimationFrame(() => {
      flowRef.current?.fitView({ padding: 0.2, maxZoom: 1.2 });
    });

    return () => window.cancelAnimationFrame(raf);
  }, [autoFitFingerprint, autoFitOnChange, styledNodesLength, flowRef, lens, processedNodes]);

  useEffect(() => {
    if (!flowRef.current || fitViewNonce === 0 || styledNodesLength === 0) {
      return;
    }

    const raf = window.requestAnimationFrame(() => {
      flowRef.current?.fitView({ padding: 0.2, maxZoom: 1.2 });
    });

    return () => window.cancelAnimationFrame(raf);
  }, [fitViewNonce, styledNodesLength, flowRef]);
}
