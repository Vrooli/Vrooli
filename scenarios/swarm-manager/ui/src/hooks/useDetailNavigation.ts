/**
 * useDetailNavigation
 *
 * Single entry point for all detail page navigation. Encapsulates:
 * - Opening a detail page (with sidebar state preservation)
 * - Closing a detail page (with sidebar state restoration)
 * - Opening the sidebar from within a detail page
 * - Drilling to a graph lens from a detail page
 *
 * This hook coordinates between the detail-selection-store and graph-ui-store
 * so that sidebar state is preserved across detail open/close cycles.
 *
 * DOC: docs/plans/navigation-header-unification-plan.md#phase-1
 */

import { useCallback } from "react";
import { useSearchParams } from "react-router-dom";
import { useDetailSelectionStore, type DetailSelection } from "../stores/detail-selection-store";
import { useGraphUIStore } from "../surfaces/graph/stores/graph-ui-store";
import { useGraphDataStore, type GraphLens } from "../surfaces/graph/stores/graph-data-store";
import { getGraphNodeLabel } from "../surfaces/graph/types";

interface OpenDetailOptions {
  /** Whether the detail was opened from the sidebar (affects restore behavior). */
  fromSidebar?: boolean;
}

export interface DetailNavigation {
  openDetail: (selection: DetailSelection, opts?: OpenDetailOptions) => void;
  closeDetail: () => void;
  openSidebar: () => void;
  drillToLens: (nodeId: string, targetLens: GraphLens) => void;
}

export function useDetailNavigation(): DetailNavigation {
  const [, setSearchParams] = useSearchParams();

  const selectBacklog = useDetailSelectionStore((s) => s.selectBacklog);
  const selectScenario = useDetailSelectionStore((s) => s.selectScenario);
  const selectExecution = useDetailSelectionStore((s) => s.selectExecution);
  const selectInitiative = useDetailSelectionStore((s) => s.selectInitiative);
  const clearSelection = useDetailSelectionStore((s) => s.clearSelection);

  const saveSidebarState = useGraphUIStore((s) => s.saveSidebarStateBeforeDetail);
  const restoreSidebarState = useGraphUIStore((s) => s.restoreSidebarStateAfterDetail);
  const setSidebarCollapsed = useGraphUIStore((s) => s.setSidebarCollapsed);

  const lens = useGraphDataStore((s) => s.lens);
  const nodes = useGraphDataStore((s) => s.nodes);
  const setFocusNodeLabel = useGraphUIStore((s) => s.setFocusNodeLabel);

  const openDetail = useCallback(
    (selection: DetailSelection, opts?: OpenDetailOptions) => {
      // Save current sidebar state before opening detail page.
      if (opts?.fromSidebar) {
        saveSidebarState();
      }

      // Collapse sidebar on mobile so detail page is visible.
      if (window.innerWidth < 768) {
        setSidebarCollapsed(true);
      }

      switch (selection.entityType) {
        case "backlog":
          if (selection.kind && selection.name) {
            selectBacklog(selection.kind, selection.name, selection.tab);
          }
          break;
        case "scenario":
          if (selection.name) selectScenario(selection.name, selection.tab);
          break;
        case "execution":
          if (selection.identifier) selectExecution(selection.identifier);
          break;
        case "initiative":
          if (selection.name) selectInitiative(selection.name, selection.tab);
          break;
      }
    },
    [saveSidebarState, setSidebarCollapsed, selectBacklog, selectScenario, selectExecution, selectInitiative],
  );

  const closeDetail = useCallback(() => {
    clearSelection();
    restoreSidebarState();
  }, [clearSelection, restoreSidebarState]);

  const openSidebar = useCallback(() => {
    setSidebarCollapsed(false);
  }, [setSidebarCollapsed]);

  const drillToLens = useCallback(
    (nodeId: string, targetLens: GraphLens) => {
      const node = nodes.find((n) => n.id === nodeId);
      if (node) setFocusNodeLabel(getGraphNodeLabel(node));

      clearSelection();
      // Drilling to lens is an explicit graph navigation — don't restore sidebar.
      setSidebarCollapsed(true);

      setSearchParams((prev) => {
        const next = new URLSearchParams(prev);
        next.set("lens", targetLens);
        next.set("focus", nodeId);
        next.set("returnLens", lens);
        next.delete("select");
        // Clear detail params only (preserve sidebar params).
        next.delete("detail");
        next.delete("kind");
        next.delete("name");
        next.delete("execId");
        next.delete("tab");
        return next;
      });
    },
    [clearSelection, lens, nodes, setFocusNodeLabel, setSidebarCollapsed, setSearchParams],
  );

  return { openDetail, closeDetail, openSidebar, drillToLens };
}
