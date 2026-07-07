/**
 * useGraphStateSync — Data store subscription, auto-refresh polling,
 * and URL-to-store sync for the graph workspace.
 *
 * Extracted from GraphWorkspace.tsx.
 */

import { useCallback, useEffect, useRef } from "react";
import { useLocation, useNavigate, useSearchParams } from "react-router-dom";
import { graphPath, isGraphMode, type AppGraphLens } from "../../../app/routes/route-paths";
import { useGraphDataStore, type GraphLens } from "../stores/graph-data-store";
import { useGraphUIStore } from "../stores/graph-ui-store";
import { clearVisualFocus } from "../lib/visual-focus";
import { getGraphNodeLabel } from "../types";

export function isGraphLens(value: string | null): value is AppGraphLens {
  return value === "plan" || value === "graph" || value === "focus";
}

export interface UseGraphStateSyncResult {
  urlLens: AppGraphLens;
  handleLensChange: (newLens: AppGraphLens) => void;
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
  const navigate = useNavigate();
  const location = useLocation();
  const [searchParams, setSearchParams] = useSearchParams();

  const isPlanRoute = location.pathname === "/plan";
  const urlMode = searchParams.get("mode");
  const graphMode: GraphLens = isGraphMode(urlMode ?? undefined) ? (urlMode as GraphLens) : "topology";
  const urlDataLens: GraphLens = isPlanRoute ? "plan" : graphMode;
  const urlLens: AppGraphLens = isPlanRoute ? "plan" : urlDataLens === "focus" ? "focus" : "graph";
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
    setLens(urlDataLens);
    applyLayoutForLens(urlDataLens);
    void fetchGraph(urlDataLens);
  }, [applyLayoutForLens, fetchGraph, setLens, urlDataLens]);

  // Sync URL focus/returnLens → store
  useEffect(() => {
    setFocusNode(urlFocus ?? null);
    setReturnLens(
      urlReturnLens === "plan" || urlReturnLens === "focus" || urlReturnLens === "topology"
        ? urlReturnLens
        : null,
    );
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
    (newLens: AppGraphLens) => {
      navigate(graphPath({
        lens: newLens,
        focus: searchParams.get("focus"),
        returnLens: searchParams.get("returnLens"),
        select: searchParams.get("select"),
      }));
    },
    [navigate, searchParams],
  );

  const handleReturnToAtlas = useCallback(() => {
    const target: AppGraphLens =
      returnLens === "plan" ? "plan" : returnLens === "focus" ? "focus" : "graph";
    navigate(graphPath({ lens: target }));
  }, [navigate, returnLens]);

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
