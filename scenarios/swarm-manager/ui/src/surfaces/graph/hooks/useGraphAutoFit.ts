/**
 * useGraphAutoFit — Viewport auto-fitting, fingerprint tracking, and
 * debounced layout for the graph canvas.
 *
 * Extracted from GraphCanvas.tsx.
 */

import { useEffect, useMemo, useRef } from "react";
import type { ReactFlowInstance } from "@xyflow/react";
import { useGraphDataStore } from "../stores/graph-data-store";
import { useGraphUIStore } from "../stores/graph-ui-store";
import type { GraphEdge, GraphNode } from "../types";

export interface UseGraphAutoFitOptions {
  flowRef: React.RefObject<ReactFlowInstance<GraphNode, GraphEdge> | null>;
  lens: string;
  layoutMode: string;
  layoutDirection: string;
  groupingMode: string;
  showSecondaryEdges: boolean;
  autoFitOnChange: boolean;
  processedNodes: GraphNode[];
  processedEdges: GraphEdge[];
  styledNodesLength: number;
}

/**
 * Manages auto-fit-on-change behavior and manual fitView nonce triggers.
 *
 * Suppresses fitView on the first data arrival when a stored viewport exists,
 * so the user's last viewport is preserved on page refresh.
 */
export function useGraphAutoFit({
  flowRef,
  lens,
  layoutMode,
  layoutDirection,
  groupingMode,
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
  const autoFitFingerprint = `${lens}|${layoutMode}|${layoutDirection}|${groupingMode}|${showSecondaryEdges}|${nodeFingerprint}|${edgeFingerprint}`;

  useEffect(() => {
    if (!autoFitOnChange || !flowRef.current || styledNodesLength === 0) {
      return;
    }

    // On the first data arrival after mount or lens switch, skip fitView if
    // a stored viewport exists.
    if (!initialLoadCompleteRef.current) {
      initialLoadCompleteRef.current = true;
      const currentLens = useGraphDataStore.getState().lens;
      const savedViewport = useGraphUIStore.getState().viewportByLens[currentLens];
      if (savedViewport) {
        return;
      }
    }

    const raf = window.requestAnimationFrame(() => {
      flowRef.current?.fitView({ padding: 0.2, maxZoom: 1.2 });
    });

    return () => window.cancelAnimationFrame(raf);
  }, [autoFitFingerprint, autoFitOnChange, styledNodesLength, flowRef]);

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
