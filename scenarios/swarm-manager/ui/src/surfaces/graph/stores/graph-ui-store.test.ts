import { describe, it, expect, beforeEach, vi } from "vitest";
import { useGraphUIStore, graphUIInitialState } from "./graph-ui-store";

function resetStore() {
  useGraphUIStore.setState({
    ...graphUIInitialState,
    highlightState: { highlighted: new Set(), mode: "normal" },
    layoutPreferences: {},
    viewport: null,
    sidebarCollapsed: false,
    inspectorOpen: false,
    selectedNodeId: null,
  });
}

// Mock localStorage.
const storage = new Map<string, string>();
vi.stubGlobal("localStorage", {
  getItem: (key: string) => storage.get(key) ?? null,
  setItem: (key: string, value: string) => storage.set(key, value),
  removeItem: (key: string) => storage.delete(key),
});

describe("graphUIStore", () => {
  beforeEach(() => {
    storage.clear();
    resetStore();
  });

  describe("node selection", () => {
    it("starts with no selection", () => {
      expect(useGraphUIStore.getState().selectedNodeId).toBeNull();
    });

    it("selects a node and opens inspector", () => {
      useGraphUIStore.getState().selectNode("node-1");
      expect(useGraphUIStore.getState().selectedNodeId).toBe("node-1");
      expect(useGraphUIStore.getState().inspectorOpen).toBe(true);
    });

    it("deselects node on null", () => {
      useGraphUIStore.getState().selectNode("node-1");
      useGraphUIStore.getState().selectNode(null);
      expect(useGraphUIStore.getState().selectedNodeId).toBeNull();
      expect(useGraphUIStore.getState().inspectorOpen).toBe(false);
    });
  });

  describe("highlight state", () => {
    it("defaults to normal mode with empty set", () => {
      const { highlightState } = useGraphUIStore.getState();
      expect(highlightState.mode).toBe("normal");
      expect(highlightState.highlighted.size).toBe(0);
    });

    it("updates highlight state", () => {
      useGraphUIStore.getState().setHighlightState({
        highlighted: new Set(["a", "b"]),
        mode: "dim",
      });
      const { highlightState } = useGraphUIStore.getState();
      expect(highlightState.mode).toBe("dim");
      expect(highlightState.highlighted.has("a")).toBe(true);
      expect(highlightState.highlighted.has("b")).toBe(true);
    });
  });

  describe("layout mode", () => {
    it("defaults to hierarchical", () => {
      expect(useGraphUIStore.getState().layoutMode).toBe("hierarchical");
    });

    it("sets layout mode", () => {
      useGraphUIStore.getState().setLayoutMode("compact");
      expect(useGraphUIStore.getState().layoutMode).toBe("compact");
    });

    it("cycles through layout modes", () => {
      expect(useGraphUIStore.getState().layoutMode).toBe("hierarchical");
      useGraphUIStore.getState().cycleLayoutMode();
      expect(useGraphUIStore.getState().layoutMode).toBe("compact");
      useGraphUIStore.getState().cycleLayoutMode();
      expect(useGraphUIStore.getState().layoutMode).toBe("grouped");
      useGraphUIStore.getState().cycleLayoutMode();
      expect(useGraphUIStore.getState().layoutMode).toBe("hierarchical");
    });
  });

  describe("per-lens layout preferences", () => {
    it("returns hierarchical as default for unknown lens", () => {
      expect(useGraphUIStore.getState().getLayoutForLens("topology")).toBe("hierarchical");
    });

    it("persists and retrieves per-lens layout", () => {
      useGraphUIStore.getState().setLayoutForLens("topology", "compact");
      expect(useGraphUIStore.getState().getLayoutForLens("topology")).toBe("compact");
      expect(useGraphUIStore.getState().layoutMode).toBe("compact");
    });

    it("saves to localStorage", () => {
      useGraphUIStore.getState().setLayoutForLens("flow", "grouped");
      const stored = JSON.parse(storage.get("swarm-manager.graph.layout")!);
      expect(stored.flow).toBe("grouped");
    });
  });

  describe("viewport persistence", () => {
    it("starts with null viewport", () => {
      expect(useGraphUIStore.getState().viewport).toBeNull();
    });

    it("saves viewport to state and localStorage", () => {
      const viewport = { x: 100, y: 200, zoom: 1.5 };
      useGraphUIStore.getState().setViewport(viewport);
      expect(useGraphUIStore.getState().viewport).toEqual(viewport);
      const stored = JSON.parse(storage.get("swarm-manager.graph.viewport")!);
      expect(stored).toEqual(viewport);
    });
  });

  describe("sidebar collapse", () => {
    it("starts expanded", () => {
      expect(useGraphUIStore.getState().sidebarCollapsed).toBe(false);
    });

    it("toggles collapse state", () => {
      useGraphUIStore.getState().toggleSidebar();
      expect(useGraphUIStore.getState().sidebarCollapsed).toBe(true);
      expect(storage.get("swarm-manager.graph.sidebar-collapsed")).toBe("true");
      useGraphUIStore.getState().toggleSidebar();
      expect(useGraphUIStore.getState().sidebarCollapsed).toBe(false);
    });

    it("sets collapse state explicitly", () => {
      useGraphUIStore.getState().setSidebarCollapsed(true);
      expect(useGraphUIStore.getState().sidebarCollapsed).toBe(true);
    });
  });

  describe("inspector", () => {
    it("starts closed", () => {
      expect(useGraphUIStore.getState().inspectorOpen).toBe(false);
    });

    it("toggles inspector", () => {
      useGraphUIStore.getState().toggleInspector();
      expect(useGraphUIStore.getState().inspectorOpen).toBe(true);
      useGraphUIStore.getState().toggleInspector();
      expect(useGraphUIStore.getState().inspectorOpen).toBe(false);
    });
  });

  describe("cluster collapse", () => {
    it("starts with empty collapsed clusters set", () => {
      expect(useGraphUIStore.getState().collapsedClusters.size).toBe(0);
    });

    it("toggles cluster collapse", () => {
      useGraphUIStore.getState().toggleClusterCollapse("initiative/init-1");
      expect(useGraphUIStore.getState().collapsedClusters.has("initiative/init-1")).toBe(true);

      useGraphUIStore.getState().toggleClusterCollapse("initiative/init-1");
      expect(useGraphUIStore.getState().collapsedClusters.has("initiative/init-1")).toBe(false);
    });

    it("sets all clusters collapsed", () => {
      useGraphUIStore.getState().setAllClustersCollapsed([
        "initiative/init-1",
        "initiative/init-2",
        "__unassigned__",
      ]);
      const { collapsedClusters } = useGraphUIStore.getState();
      expect(collapsedClusters.size).toBe(3);
      expect(collapsedClusters.has("initiative/init-1")).toBe(true);
      expect(collapsedClusters.has("initiative/init-2")).toBe(true);
      expect(collapsedClusters.has("__unassigned__")).toBe(true);
    });
  });
});
