/**
 * Graph UI Store
 *
 * Owns interaction state: selection, highlight mode, layout preferences,
 * viewport persistence, and panel visibility.
 */

import { create } from "zustand";
import type { Viewport } from "@xyflow/react";
import type { GraphLens } from "./graph-data-store";

export type LayoutMode = "hierarchical" | "compact" | "grouped";
export type LayoutDirection = "TB" | "LR";
export type HighlightMode = "normal" | "highlight" | "dim" | "hide";

export interface NodeHighlightState {
  highlighted: Set<string>;
  mode: HighlightMode;
}

const LAYOUT_STORAGE_KEY = "swarm-manager.graph.layout";
const LAYOUT_DIRECTION_STORAGE_KEY = "swarm-manager.graph.layout-direction";
const VIEWPORT_STORAGE_KEY = "swarm-manager.graph.viewport.v2";
const SIDEBAR_STORAGE_KEY = "swarm-manager.graph.sidebar-collapsed";

const LAYOUT_CYCLE: LayoutMode[] = ["hierarchical", "compact", "grouped"];

type ViewportByLens = Record<GraphLens, Viewport | null>;

function loadLayoutPreferences(): Record<string, LayoutMode> {
  try {
    const raw = window.localStorage.getItem(LAYOUT_STORAGE_KEY);
    if (!raw) return {};
    const parsed = JSON.parse(raw) as Record<string, unknown>;
    const next: Record<string, LayoutMode> = {};

    for (const [key, value] of Object.entries(parsed)) {
      if (value === "hierarchical" || value === "compact" || value === "grouped") {
        next[key] = value;
      }
    }

    return next;
  } catch {
    return {};
  }
}

function saveLayoutPreferences(prefs: Record<string, LayoutMode>): void {
  try {
    window.localStorage.setItem(LAYOUT_STORAGE_KEY, JSON.stringify(prefs));
  } catch {
    // Ignore persistence failures.
  }
}

function loadLayoutDirection(): LayoutDirection {
  try {
    return window.localStorage.getItem(LAYOUT_DIRECTION_STORAGE_KEY) === "LR" ? "LR" : "TB";
  } catch {
    return "TB";
  }
}

function saveLayoutDirection(direction: LayoutDirection): void {
  try {
    window.localStorage.setItem(LAYOUT_DIRECTION_STORAGE_KEY, direction);
  } catch {
    // Ignore persistence failures.
  }
}

function createEmptyViewportByLens(): ViewportByLens {
  return {
    topology: null,
    flow: null,
    operations: null,
  };
}

function isViewport(value: unknown): value is Viewport {
  if (!value || typeof value !== "object") {
    return false;
  }

  const record = value as Record<string, unknown>;
  return typeof record.x === "number"
    && typeof record.y === "number"
    && typeof record.zoom === "number";
}

function loadViewportByLens(): ViewportByLens {
  const empty = createEmptyViewportByLens();
  try {
    const raw = window.localStorage.getItem(VIEWPORT_STORAGE_KEY);
    if (!raw) return empty;

    const parsed = JSON.parse(raw) as Record<string, unknown>;
    const next = createEmptyViewportByLens();

    if (isViewport(parsed.topology)) {
      next.topology = parsed.topology;
    }
    if (isViewport(parsed.flow)) {
      next.flow = parsed.flow;
    }
    if (isViewport(parsed.operations)) {
      next.operations = parsed.operations;
    }

    return next;
  } catch {
    return empty;
  }
}

function saveViewportByLens(viewportByLens: ViewportByLens): void {
  try {
    window.localStorage.setItem(VIEWPORT_STORAGE_KEY, JSON.stringify(viewportByLens));
  } catch {
    // Ignore persistence failures.
  }
}

function loadSidebarCollapsed(): boolean {
  try {
    return window.localStorage.getItem(SIDEBAR_STORAGE_KEY) === "true";
  } catch {
    return false;
  }
}

function saveSidebarCollapsed(collapsed: boolean): void {
  try {
    window.localStorage.setItem(SIDEBAR_STORAGE_KEY, String(collapsed));
  } catch {
    // Ignore persistence failures.
  }
}

export interface GraphUIState {
  selectedNodeId: string | null;
  highlightState: NodeHighlightState;
  layoutMode: LayoutMode;
  layoutPreferences: Record<string, LayoutMode>;
  layoutDirection: LayoutDirection;
  fitViewNonce: number;
  viewportByLens: ViewportByLens;
  sidebarCollapsed: boolean;
  inspectorOpen: boolean;
  expandedTopologyClusters: Set<string>;
  selectNode: (nodeId: string | null) => void;
  setHighlightState: (state: NodeHighlightState) => void;
  setLayoutMode: (mode: LayoutMode) => void;
  cycleLayoutMode: (lens: string) => void;
  setLayoutForLens: (lens: string, mode: LayoutMode) => void;
  applyLayoutForLens: (lens: string) => void;
  getLayoutForLens: (lens: string) => LayoutMode;
  setLayoutDirection: (direction: LayoutDirection) => void;
  requestFitView: () => void;
  setViewportForLens: (lens: GraphLens, viewport: Viewport) => void;
  toggleSidebar: () => void;
  setSidebarCollapsed: (collapsed: boolean) => void;
  toggleInspector: () => void;
  setInspectorOpen: (open: boolean) => void;
  toggleTopologyCluster: (clusterId: string) => void;
  collapseAllTopologyClusters: () => void;
  expandTopologyClusters: (clusterIds: string[]) => void;
}

const initialPrefs = typeof window !== "undefined" ? loadLayoutPreferences() : {};
const initialViewportByLens = typeof window !== "undefined" ? loadViewportByLens() : createEmptyViewportByLens();
const initialSidebarCollapsed = typeof window !== "undefined" ? loadSidebarCollapsed() : false;
const initialLayoutDirection = typeof window !== "undefined" ? loadLayoutDirection() : "TB";

export const graphUIInitialState = {
  selectedNodeId: null as string | null,
  highlightState: { highlighted: new Set<string>(), mode: "normal" as HighlightMode },
  layoutMode: "hierarchical" as LayoutMode,
  layoutPreferences: initialPrefs,
  layoutDirection: initialLayoutDirection,
  fitViewNonce: 0,
  viewportByLens: initialViewportByLens,
  sidebarCollapsed: initialSidebarCollapsed,
  inspectorOpen: false,
  expandedTopologyClusters: new Set<string>(),
};

export const useGraphUIStore = create<GraphUIState>((set, get) => ({
  ...graphUIInitialState,

  selectNode: (nodeId) =>
    set({
      selectedNodeId: nodeId,
      inspectorOpen: nodeId !== null,
    }),

  setHighlightState: (highlightState) => set({ highlightState }),

  setLayoutMode: (mode) => set({ layoutMode: mode }),

  cycleLayoutMode: (lens) =>
    set((state) => {
      const currentMode = state.layoutPreferences[lens] ?? state.layoutMode;
      const idx = LAYOUT_CYCLE.indexOf(currentMode);
      const next = LAYOUT_CYCLE[(idx + 1) % LAYOUT_CYCLE.length] as LayoutMode;
      const layoutPreferences: Record<string, LayoutMode> = {
        ...state.layoutPreferences,
        [lens]: next,
      };
      saveLayoutPreferences(layoutPreferences);
      return {
        layoutMode: next,
        layoutPreferences,
      };
    }),

  setLayoutForLens: (lens, mode) =>
    set((state) => {
      const layoutPreferences: Record<string, LayoutMode> = {
        ...state.layoutPreferences,
        [lens]: mode,
      };
      saveLayoutPreferences(layoutPreferences);
      return {
        layoutMode: mode,
        layoutPreferences,
      };
    }),

  applyLayoutForLens: (lens) =>
    set((state) => ({
      layoutMode: state.layoutPreferences[lens] ?? "hierarchical",
    })),

  getLayoutForLens: (lens) => get().layoutPreferences[lens] ?? "hierarchical",

  setLayoutDirection: (direction) => {
    saveLayoutDirection(direction);
    set({ layoutDirection: direction });
  },

  requestFitView: () =>
    set((state) => ({
      fitViewNonce: state.fitViewNonce + 1,
    })),

  setViewportForLens: (lens, viewport) =>
    set((state) => {
      const viewportByLens = {
        ...state.viewportByLens,
        [lens]: viewport,
      };
      saveViewportByLens(viewportByLens);
      return { viewportByLens };
    }),

  toggleSidebar: () =>
    set((state) => {
      const next = !state.sidebarCollapsed;
      saveSidebarCollapsed(next);
      return { sidebarCollapsed: next };
    }),

  setSidebarCollapsed: (collapsed) => {
    saveSidebarCollapsed(collapsed);
    set({ sidebarCollapsed: collapsed });
  },

  toggleInspector: () =>
    set((state) => ({ inspectorOpen: !state.inspectorOpen })),

  setInspectorOpen: (open) => set({ inspectorOpen: open }),

  toggleTopologyCluster: (clusterId) =>
    set((state) => {
      const next = new Set(state.expandedTopologyClusters);
      if (next.has(clusterId)) {
        next.delete(clusterId);
      } else {
        next.add(clusterId);
      }
      return { expandedTopologyClusters: next };
    }),

  collapseAllTopologyClusters: () => set({ expandedTopologyClusters: new Set<string>() }),

  expandTopologyClusters: (clusterIds) => set({ expandedTopologyClusters: new Set(clusterIds) }),
}));

export function cloneGraphUIInitialState(): typeof graphUIInitialState {
  return {
    ...graphUIInitialState,
    highlightState: {
      highlighted: new Set(graphUIInitialState.highlightState.highlighted),
      mode: graphUIInitialState.highlightState.mode,
    },
    layoutPreferences: { ...graphUIInitialState.layoutPreferences },
    viewportByLens: {
      topology: graphUIInitialState.viewportByLens.topology ? { ...graphUIInitialState.viewportByLens.topology } : null,
      flow: graphUIInitialState.viewportByLens.flow ? { ...graphUIInitialState.viewportByLens.flow } : null,
      operations: graphUIInitialState.viewportByLens.operations ? { ...graphUIInitialState.viewportByLens.operations } : null,
    },
    expandedTopologyClusters: new Set(graphUIInitialState.expandedTopologyClusters),
  };
}
