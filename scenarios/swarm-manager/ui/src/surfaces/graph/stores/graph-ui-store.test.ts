import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cloneGraphUIInitialState, useGraphUIStore } from "./graph-ui-store";

function resetStore() {
  useGraphUIStore.setState(cloneGraphUIInitialState());
  window.localStorage.clear();
}

describe("graphUIStore", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    resetStore();
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  describe("node selection", () => {
    it("starts with no selection", () => {
      expect(useGraphUIStore.getState().selectedNodeId).toBeNull();
    });

    it("selects a node", () => {
      useGraphUIStore.getState().selectNode("node-1");
      expect(useGraphUIStore.getState().selectedNodeId).toBe("node-1");
    });

    it("clears selection on null", () => {
      useGraphUIStore.getState().selectNode("node-1");
      useGraphUIStore.getState().selectNode(null);
      expect(useGraphUIStore.getState().selectedNodeId).toBeNull();
    });
  });

  describe("layout preferences", () => {
    it("defaults to hierarchical before a route lens is applied", () => {
      expect(useGraphUIStore.getState().layoutMode).toBe("hierarchical");
    });

    it("uses grouped as the default topology layout", () => {
      useGraphUIStore.getState().applyLayoutForLens("topology");
      expect(useGraphUIStore.getState().layoutMode).toBe("grouped");
      expect(useGraphUIStore.getState().getLayoutForLens("topology")).toBe("grouped");
    });

    it("persists per-lens layout mode changes", () => {
      useGraphUIStore.getState().setLayoutForLens("topology", "compact");

      expect(useGraphUIStore.getState().layoutMode).toBe("compact");
      expect(useGraphUIStore.getState().getLayoutForLens("topology")).toBe("compact");
      expect(window.localStorage.getItem("swarm-manager.graph.layout")).toContain("\"topology\":\"compact\"");
    });

    it("cycles layout mode for the current lens", () => {
      useGraphUIStore.getState().cycleLayoutMode("topology");
      expect(useGraphUIStore.getState().layoutMode).toBe("hierarchical");

      useGraphUIStore.getState().cycleLayoutMode("topology");
      expect(useGraphUIStore.getState().layoutMode).toBe("compact");
    });

    it("loads a stored lens layout into the active layout mode", () => {
      useGraphUIStore.getState().setLayoutForLens("focus", "grouped");
      useGraphUIStore.getState().setLayoutMode("hierarchical");

      useGraphUIStore.getState().applyLayoutForLens("focus");
      expect(useGraphUIStore.getState().layoutMode).toBe("grouped");
    });

    it("stores layout direction", () => {
      useGraphUIStore.getState().setLayoutDirection("LR");
      expect(useGraphUIStore.getState().layoutDirection).toBe("LR");
      expect(window.localStorage.getItem("swarm-manager.graph.layout-direction")).toBe("LR");
    });
  });

  describe("fit view and viewport intent", () => {
    it("increments the fit-view nonce on request", () => {
      expect(useGraphUIStore.getState().fitViewNonce).toBe(0);
      useGraphUIStore.getState().requestFitView();
      expect(useGraphUIStore.getState().fitViewNonce).toBe(1);
    });

    it("persists viewport intent per lens", () => {
      const intent = { nodeId: "backlog-item/execute/task-a", zoom: 1.2 };
      useGraphUIStore.getState().setViewportIntentForLens("focus", intent);

      // Store state is updated synchronously.
      expect(useGraphUIStore.getState().viewportIntentByLens.focus).toEqual(intent);

      // localStorage write is debounced — flush the timer.
      vi.advanceTimersByTime(600);
      expect(window.localStorage.getItem("swarm-manager.graph.viewport-intent.v1")).toBe(
        JSON.stringify({
          plan: null,
          focus: intent,
          topology: null,
        }),
      );
    });

    it("keeps lens intents isolated from each other", () => {
      const topologyIntent = { nodeId: "scenario/swarm-manager", zoom: 1.2 };
      const focusIntent = { nodeId: "execution-record/exec-1", zoom: 0.75 };

      useGraphUIStore.getState().setViewportIntentForLens("topology", topologyIntent);
      useGraphUIStore.getState().setViewportIntentForLens("focus", focusIntent);

      expect(useGraphUIStore.getState().viewportIntentByLens.topology).toEqual(topologyIntent);
      expect(useGraphUIStore.getState().viewportIntentByLens.focus).toEqual(focusIntent);
    });

    it("accepts intent with null nodeId (pan/zoom without selection)", () => {
      const intent = { nodeId: null, zoom: 0.9 };
      useGraphUIStore.getState().setViewportIntentForLens("focus", intent);
      expect(useGraphUIStore.getState().viewportIntentByLens.focus).toEqual(intent);
    });

    it("can clear a stored intent by passing null", () => {
      useGraphUIStore.getState().setViewportIntentForLens("focus", { nodeId: "x", zoom: 1 });
      useGraphUIStore.getState().setViewportIntentForLens("focus", null);
      expect(useGraphUIStore.getState().viewportIntentByLens.focus).toBeNull();
    });

    it("skips writes when intent has not meaningfully changed", () => {
      const setSpy = vi.spyOn(window.localStorage, "setItem");
      useGraphUIStore.getState().setViewportIntentForLens("focus", { nodeId: "x", zoom: 1.0 });
      vi.advanceTimersByTime(600);
      const writesAfterFirst = setSpy.mock.calls.length;

      // Same nodeId, zoom within 0.001 → no-op.
      useGraphUIStore.getState().setViewportIntentForLens("focus", { nodeId: "x", zoom: 1.00005 });
      vi.advanceTimersByTime(600);
      expect(setSpy.mock.calls.length).toBe(writesAfterFirst);
    });
  });

  describe("sidebar", () => {
    it("toggles the sidebar", () => {
      useGraphUIStore.getState().toggleSidebar();
      expect(useGraphUIStore.getState().sidebarCollapsed).toBe(true);
      expect(window.localStorage.getItem("swarm-manager.graph.sidebar-collapsed")).toBe("true");
    });
  });

});
