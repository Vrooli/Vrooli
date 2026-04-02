/**
 * Graph UI Store
 *
 * Owns interaction state: selection, highlight mode, layout preferences,
 * viewport persistence, and panel visibility.
 */

import { create } from "zustand";
import type { ReactFlowInstance, Viewport } from "@xyflow/react";
import type { GraphLens } from "./graph-data-store";
import type { GraphNode, GraphEdge } from "../types";

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
const SIDEBAR_WAS_OPEN_KEY = "swarm-manager.graph.sidebar-was-open-before-detail";

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
    focus: null,
    topology: null,
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

    if (isViewport(parsed.focus)) {
      next.focus = parsed.focus;
    }
    if (isViewport(parsed.topology)) {
      next.topology = parsed.topology;
    }
    if (isViewport(parsed.operations)) {
      next.operations = parsed.operations;
    }

    return next;
  } catch {
    return empty;
  }
}

// PERF: Debounce viewport persistence. onMoveEnd fires frequently during
// pan/zoom gestures. Synchronous JSON.stringify + localStorage.setItem on
// every call blocks the main thread. We batch writes with a 500ms debounce
// so only the final viewport position is persisted.
let viewportSaveTimer: ReturnType<typeof setTimeout> | null = null;
function saveViewportByLens(viewportByLens: ViewportByLens): void {
  if (viewportSaveTimer) clearTimeout(viewportSaveTimer);
  viewportSaveTimer = setTimeout(() => {
    try {
      window.localStorage.setItem(VIEWPORT_STORAGE_KEY, JSON.stringify(viewportByLens));
    } catch {
      // Ignore persistence failures.
    }
  }, 500);
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

function loadSidebarWasOpenBeforeDetail(): boolean {
  try {
    return window.localStorage.getItem(SIDEBAR_WAS_OPEN_KEY) === "true";
  } catch {
    return false;
  }
}

function saveSidebarWasOpenBeforeDetail(wasOpen: boolean): void {
  try {
    window.localStorage.setItem(SIDEBAR_WAS_OPEN_KEY, String(wasOpen));
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
  expandedTopologyClusters: Set<string>;
  focusNodeLabel: string | null;
  sidebarWasOpenBeforeDetail: boolean;
  /** Runtime-only ref to the React Flow instance. NOT persisted to localStorage.
   *  Set by GraphCanvas on init; consumed by GraphNavControls for viewport manipulation. */
  flowInstance: ReactFlowInstance<GraphNode, GraphEdge> | null;
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
  toggleTopologyCluster: (clusterId: string) => void;
  collapseAllTopologyClusters: () => void;
  expandTopologyClusters: (clusterIds: string[]) => void;
  setFocusNodeLabel: (label: string | null) => void;
  saveSidebarStateBeforeDetail: () => void;
  restoreSidebarStateAfterDetail: () => void;
  setFlowInstance: (instance: ReactFlowInstance<GraphNode, GraphEdge> | null) => void;
}

const initialPrefs = typeof window !== "undefined" ? loadLayoutPreferences() : {};
const initialViewportByLens = typeof window !== "undefined" ? loadViewportByLens() : createEmptyViewportByLens();
const initialSidebarCollapsed = typeof window !== "undefined" ? loadSidebarCollapsed() : false;
const initialSidebarWasOpen = typeof window !== "undefined" ? loadSidebarWasOpenBeforeDetail() : false;
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
  sidebarWasOpenBeforeDetail: initialSidebarWasOpen,
  expandedTopologyClusters: new Set<string>(),
  focusNodeLabel: null as string | null,
  flowInstance: null as ReactFlowInstance<GraphNode, GraphEdge> | null,
};

export const useGraphUIStore = create<GraphUIState>((set, get) => ({
  ...graphUIInitialState,

  selectNode: (nodeId) =>
    set({
      selectedNodeId: nodeId,
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

  setViewportForLens: (lens, viewport) => {
    // PERF: Skip state update if viewport hasn't meaningfully changed.
    // onMoveEnd fires frequently; each set() triggers Zustand notifications
    // to all subscribers. We threshold to avoid unnecessary re-renders.
    const current = get().viewportByLens[lens];
    if (
      current
      && Math.abs(current.x - viewport.x) < 0.5
      && Math.abs(current.y - viewport.y) < 0.5
      && Math.abs(current.zoom - viewport.zoom) < 0.001
    ) {
      return;
    }
    set((state) => {
      const viewportByLens = {
        ...state.viewportByLens,
        [lens]: viewport,
      };
      saveViewportByLens(viewportByLens);
      return { viewportByLens };
    });
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

  setFocusNodeLabel: (label) => set({ focusNodeLabel: label }),

  saveSidebarStateBeforeDetail: () => {
    const wasOpen = !get().sidebarCollapsed;
    saveSidebarWasOpenBeforeDetail(wasOpen);
    set({ sidebarWasOpenBeforeDetail: wasOpen });
  },

  restoreSidebarStateAfterDetail: () => {
    const wasOpen = get().sidebarWasOpenBeforeDetail;
    if (wasOpen) {
      saveSidebarCollapsed(false);
      set({ sidebarCollapsed: false });
    }
    saveSidebarWasOpenBeforeDetail(false);
    set({ sidebarWasOpenBeforeDetail: false });
  },

  setFlowInstance: (instance) => set({ flowInstance: instance }),
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
      focus: graphUIInitialState.viewportByLens.focus ? { ...graphUIInitialState.viewportByLens.focus } : null,
      topology: graphUIInitialState.viewportByLens.topology ? { ...graphUIInitialState.viewportByLens.topology } : null,
      operations: graphUIInitialState.viewportByLens.operations ? { ...graphUIInitialState.viewportByLens.operations } : null,
    },
    sidebarWasOpenBeforeDetail: graphUIInitialState.sidebarWasOpenBeforeDetail,
    expandedTopologyClusters: new Set(graphUIInitialState.expandedTopologyClusters),
    // Runtime-only — always null in clones (tests, resets).
    flowInstance: null,
  };
}
