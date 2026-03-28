import { beforeEach, describe, expect, it } from "vitest";
import { cloneGraphUIInitialState, useGraphUIStore } from "./graph-ui-store";

function resetStore() {
  useGraphUIStore.setState(cloneGraphUIInitialState());
  window.localStorage.clear();
}

describe("graphUIStore", () => {
  beforeEach(resetStore);

  describe("node selection", () => {
    it("starts with no selection", () => {
      expect(useGraphUIStore.getState().selectedNodeId).toBeNull();
    });

    it("selects a node and opens the inspector", () => {
      useGraphUIStore.getState().selectNode("node-1");
      expect(useGraphUIStore.getState().selectedNodeId).toBe("node-1");
      expect(useGraphUIStore.getState().inspectorOpen).toBe(true);
    });

    it("clears selection on null", () => {
      useGraphUIStore.getState().selectNode("node-1");
      useGraphUIStore.getState().selectNode(null);
      expect(useGraphUIStore.getState().selectedNodeId).toBeNull();
      expect(useGraphUIStore.getState().inspectorOpen).toBe(false);
    });
  });

  describe("layout preferences", () => {
    it("defaults to hierarchical", () => {
      expect(useGraphUIStore.getState().layoutMode).toBe("hierarchical");
    });

    it("persists per-lens layout mode changes", () => {
      useGraphUIStore.getState().setLayoutForLens("topology", "compact");

      expect(useGraphUIStore.getState().layoutMode).toBe("compact");
      expect(useGraphUIStore.getState().getLayoutForLens("topology")).toBe("compact");
      expect(window.localStorage.getItem("swarm-manager.graph.layout")).toContain("\"topology\":\"compact\"");
    });

    it("cycles layout mode for the current lens", () => {
      useGraphUIStore.getState().cycleLayoutMode("topology");
      expect(useGraphUIStore.getState().layoutMode).toBe("compact");

      useGraphUIStore.getState().cycleLayoutMode("topology");
      expect(useGraphUIStore.getState().layoutMode).toBe("grouped");
    });

    it("loads a stored lens layout into the active layout mode", () => {
      useGraphUIStore.getState().setLayoutForLens("operations", "grouped");
      useGraphUIStore.getState().setLayoutMode("hierarchical");

      useGraphUIStore.getState().applyLayoutForLens("operations");
      expect(useGraphUIStore.getState().layoutMode).toBe("grouped");
    });

    it("stores layout direction", () => {
      useGraphUIStore.getState().setLayoutDirection("LR");
      expect(useGraphUIStore.getState().layoutDirection).toBe("LR");
      expect(window.localStorage.getItem("swarm-manager.graph.layout-direction")).toBe("LR");
    });
  });

  describe("fit view and viewport", () => {
    it("increments the fit-view nonce on request", () => {
      expect(useGraphUIStore.getState().fitViewNonce).toBe(0);
      useGraphUIStore.getState().requestFitView();
      expect(useGraphUIStore.getState().fitViewNonce).toBe(1);
    });

    it("persists viewport changes", () => {
      const viewport = { x: 100, y: 200, zoom: 1.2 };
      useGraphUIStore.getState().setViewport(viewport);

      expect(useGraphUIStore.getState().viewport).toEqual(viewport);
      expect(window.localStorage.getItem("swarm-manager.graph.viewport")).toBe(
        JSON.stringify(viewport),
      );
    });
  });

  describe("sidebar and inspector", () => {
    it("toggles the sidebar", () => {
      useGraphUIStore.getState().toggleSidebar();
      expect(useGraphUIStore.getState().sidebarCollapsed).toBe(true);
      expect(window.localStorage.getItem("swarm-manager.graph.sidebar-collapsed")).toBe("true");
    });

    it("toggles the inspector", () => {
      useGraphUIStore.getState().toggleInspector();
      expect(useGraphUIStore.getState().inspectorOpen).toBe(true);
    });
  });

  describe("topology cluster expansion", () => {
    it("starts with no expanded topology clusters", () => {
      expect(useGraphUIStore.getState().expandedTopologyClusters.size).toBe(0);
    });

    it("toggles a topology cluster", () => {
      useGraphUIStore.getState().toggleTopologyCluster("initiative/graph");
      expect(useGraphUIStore.getState().expandedTopologyClusters.has("initiative/graph")).toBe(true);

      useGraphUIStore.getState().toggleTopologyCluster("initiative/graph");
      expect(useGraphUIStore.getState().expandedTopologyClusters.has("initiative/graph")).toBe(false);
    });

    it("can collapse and expand all topology clusters", () => {
      useGraphUIStore.getState().expandTopologyClusters(["initiative/one", "initiative/two"]);
      expect(useGraphUIStore.getState().expandedTopologyClusters.size).toBe(2);

      useGraphUIStore.getState().collapseAllTopologyClusters();
      expect(useGraphUIStore.getState().expandedTopologyClusters.size).toBe(0);
    });
  });
});
