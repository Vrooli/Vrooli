/**
 * Graph UI Store
 *
 * Owns interaction state: selected node, highlight/dim/hide query modes,
 * layout mode, viewport persistence, sidebar collapse.
 */

import { create } from "zustand";
import type { Viewport } from "@xyflow/react";

export type LayoutMode = "hierarchical" | "compact" | "grouped";

export type HighlightMode = "normal" | "highlight" | "dim" | "hide";

export interface NodeHighlightState {
  /** Node IDs to highlight (selected + BFS neighbors). */
  highlighted: Set<string>;
  /** Current display mode. */
  mode: HighlightMode;
}

const LAYOUT_STORAGE_KEY = "swarm-manager.graph.layout";
const VIEWPORT_STORAGE_KEY = "swarm-manager.graph.viewport";
const SIDEBAR_STORAGE_KEY = "swarm-manager.graph.sidebar-collapsed";

function loadLayoutPreferences(): Record<string, LayoutMode> {
  try {
    const raw = window.localStorage.getItem(LAYOUT_STORAGE_KEY);
    return raw ? JSON.parse(raw) : {};
  } catch {
    return {};
  }
}

function saveLayoutPreferences(prefs: Record<string, LayoutMode>): void {
  try {
    window.localStorage.setItem(LAYOUT_STORAGE_KEY, JSON.stringify(prefs));
  } catch {
    // Silent failure.
  }
}

function loadViewport(): Viewport | null {
  try {
    const raw = window.localStorage.getItem(VIEWPORT_STORAGE_KEY);
    return raw ? JSON.parse(raw) : null;
  } catch {
    return null;
  }
}

function saveViewport(viewport: Viewport): void {
  try {
    window.localStorage.setItem(VIEWPORT_STORAGE_KEY, JSON.stringify(viewport));
  } catch {
    // Silent failure.
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
    // Silent failure.
  }
}

const LAYOUT_CYCLE: LayoutMode[] = ["hierarchical", "compact", "grouped"];

export interface GraphUIState {
  selectedNodeId: string | null;
  highlightState: NodeHighlightState;
  layoutMode: LayoutMode;
  /** Per-lens layout preferences. */
  layoutPreferences: Record<string, LayoutMode>;
  viewport: Viewport | null;
  sidebarCollapsed: boolean;
  inspectorOpen: boolean;
  /** Set of collapsed cluster IDs for topology initiative clustering. */
  collapsedClusters: Set<string>;

  selectNode: (nodeId: string | null) => void;
  setHighlightState: (state: NodeHighlightState) => void;
  setLayoutMode: (mode: LayoutMode) => void;
  cycleLayoutMode: () => void;
  setLayoutForLens: (lens: string, mode: LayoutMode) => void;
  getLayoutForLens: (lens: string) => LayoutMode;
  setViewport: (viewport: Viewport) => void;
  toggleSidebar: () => void;
  setSidebarCollapsed: (collapsed: boolean) => void;
  toggleInspector: () => void;
  setInspectorOpen: (open: boolean) => void;
  toggleClusterCollapse: (clusterId: string) => void;
  setAllClustersCollapsed: (clusterIds: string[]) => void;
}

const initialPrefs = typeof window !== "undefined" ? loadLayoutPreferences() : {};
const initialViewport = typeof window !== "undefined" ? loadViewport() : null;
const initialSidebarCollapsed = typeof window !== "undefined" ? loadSidebarCollapsed() : false;

export const graphUIInitialState = {
  selectedNodeId: null as string | null,
  highlightState: { highlighted: new Set<string>(), mode: "normal" as HighlightMode },
  layoutMode: "hierarchical" as LayoutMode,
  layoutPreferences: initialPrefs,
  viewport: initialViewport,
  sidebarCollapsed: initialSidebarCollapsed,
  inspectorOpen: false,
  collapsedClusters: new Set<string>(),
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

  cycleLayoutMode: () =>
    set((state) => {
      const idx = LAYOUT_CYCLE.indexOf(state.layoutMode);
      const next = LAYOUT_CYCLE[(idx + 1) % LAYOUT_CYCLE.length];
      return { layoutMode: next };
    }),

  setLayoutForLens: (lens, mode) => {
    const prefs = { ...get().layoutPreferences, [lens]: mode };
    saveLayoutPreferences(prefs);
    set({ layoutPreferences: prefs, layoutMode: mode });
  },

  getLayoutForLens: (lens) => {
    return get().layoutPreferences[lens] ?? "hierarchical";
  },

  setViewport: (viewport) => {
    saveViewport(viewport);
    set({ viewport });
  },

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

  toggleClusterCollapse: (clusterId) =>
    set((state) => {
      const next = new Set(state.collapsedClusters);
      if (next.has(clusterId)) {
        next.delete(clusterId);
      } else {
        next.add(clusterId);
      }
      return { collapsedClusters: next };
    }),

  setAllClustersCollapsed: (clusterIds) =>
    set({ collapsedClusters: new Set(clusterIds) }),
}));
