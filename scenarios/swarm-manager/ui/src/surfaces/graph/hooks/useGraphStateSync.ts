/**
 * useGraphStateSync — Data store subscription, auto-refresh polling,
 * and URL-to-store sync for the graph workspace.
 *
 * Extracted from GraphWorkspace.tsx.
 */

import { useCallback, useEffect, useRef } from "react";
import { useSearchParams } from "react-router-dom";
import { useGraphDataStore } from "../stores/graph-data-store";
import { useGraphUIStore } from "../stores/graph-ui-store";
import { clearVisualFocus } from "../lib/visual-focus";
import { getGraphNodeLabel } from "../types";
import type { GraphLens } from "../stores/graph-data-store";

export function isGraphLens(value: string | null): value is GraphLens {
  return value === "focus" || value === "topology" || value === "operations";
}

export interface UseGraphStateSyncResult {
  urlLens: GraphLens;
  handleLensChange: (newLens: GraphLens) => void;
  handleReturnToAtlas: () => void;
  handleDeselectNode: () => void;
}

/**
 * Synchronizes URL search params with graph data store state:
 * - lens, focus, returnLens, select params
 * - Fetches graph data when lens changes
 * - Syncs focus node label
 * - Handles URL ↔ store selection sync
 */
export function useGraphStateSync(): UseGraphStateSyncResult {
  const [searchParams, setSearchParams] = useSearchParams();

  const searchLens = searchParams.get("lens");
  const urlLens: GraphLens = isGraphLens(searchLens) ? searchLens : "topology";
  const urlSelect = searchParams.get("select");
  const urlFocus = searchParams.get("focus");
  const urlReturnLens = searchParams.get("returnLens");

  const fetchGraph = useGraphDataStore((s) => s.fetchGraph);
  const nodes = useGraphDataStore((s) => s.nodes);
  const setLens = useGraphDataStore((s) => s.setLens);
  const focusNodeId = useGraphDataStore((s) => s.focusNodeId);
  const setFocusNode = useGraphDataStore((s) => s.setFocusNode);
  const returnLens = useGraphDataStore((s) => s.returnLens);
  const setReturnLens = useGraphDataStore((s) => s.setReturnLens);

  const setFocusNodeLabel = useGraphUIStore((s) => s.setFocusNodeLabel);
  const selectedNodeId = useGraphUIStore((s) => s.selectedNodeId);
  const selectNode = useGraphUIStore((s) => s.selectNode);
  const setHighlightState = useGraphUIStore((s) => s.setHighlightState);
  const applyLayoutForLens = useGraphUIStore((s) => s.applyLayoutForLens);

  // Sync URL lens → store
  useEffect(() => {
    setLens(urlLens);
    applyLayoutForLens(urlLens);
    void fetchGraph(urlLens);
  }, [applyLayoutForLens, fetchGraph, setLens, urlLens]);

  // Sync URL focus/returnLens → store
  useEffect(() => {
    setFocusNode(urlFocus ?? null);
    setReturnLens(isGraphLens(urlReturnLens) ? urlReturnLens : null);
  }, [urlFocus, urlReturnLens, setFocusNode, setReturnLens]);

  // Sync focus node label
  useEffect(() => {
    if (!focusNodeId) {
      setFocusNodeLabel(null);
      return;
    }
    const node = nodes.find((n) => n.id === focusNodeId);
    if (node) {
      setFocusNodeLabel(getGraphNodeLabel(node));
    }
  }, [focusNodeId, nodes, setFocusNodeLabel]);

  // Sync URL → store only on URL-driven changes.
  const prevUrlSelect = useRef(urlSelect);
  useEffect(() => {
    if (urlSelect === prevUrlSelect.current) {
      return;
    }
    prevUrlSelect.current = urlSelect;

    if (urlSelect) {
      if (urlSelect !== selectedNodeId) {
        selectNode(urlSelect);
      }
    } else {
      if (selectedNodeId) {
        selectNode(null);
      }
    }
  }, [selectedNodeId, selectNode, urlSelect]);

  const handleLensChange = useCallback(
    (newLens: GraphLens) => {
      setSearchParams((prev) => {
        const next = new URLSearchParams(prev);
        next.set("lens", newLens);
        return next;
      });
    },
    [setSearchParams],
  );

  const handleReturnToAtlas = useCallback(() => {
    const target = returnLens ?? "topology";
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev);
      next.set("lens", target);
      next.delete("focus");
      next.delete("returnLens");
      return next;
    });
  }, [returnLens, setSearchParams]);

  const handleDeselectNode = useCallback(() => {
    const cleared = clearVisualFocus();
    selectNode(cleared.selectedNodeId);
    setHighlightState(cleared.highlightState);
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev);
      next.delete("select");
      return next;
    });
  }, [selectNode, setHighlightState, setSearchParams]);

  // Sync store → URL when selection is cleared
  const prevSelectedNodeId = useRef(selectedNodeId);
  useEffect(() => {
    const prev = prevSelectedNodeId.current;
    prevSelectedNodeId.current = selectedNodeId;
    if (prev !== null && selectedNodeId === null) {
      setSearchParams((p) => {
        if (!p.has("select")) return p;
        const next = new URLSearchParams(p);
        next.delete("select");
        return next;
      });
    }
  }, [selectedNodeId, setSearchParams]);

  return {
    urlLens,
    handleLensChange,
    handleReturnToAtlas,
    handleDeselectNode,
  };
}
