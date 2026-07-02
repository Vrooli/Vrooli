/**
 * Graph UI Store
 *
 * Owns interaction state: selection, highlight mode, layout preferences,
 * viewport persistence, and panel visibility.
 */

import { create } from "zustand";
import type { ReactFlowInstance } from "@xyflow/react";
import type { GraphLens } from "./graph-data-store";
import type { GraphNode } from "../types";

export type LayoutMode = "hierarchical" | "compact" | "grouped";
export type LayoutDirection = "TB" | "LR";
export type HighlightMode = "normal" | "highlight" | "dim" | "hide";

export interface NodeHighlightState {
  highlighted: Set<string>;
  mode: HighlightMode;
}

/**
 * Semantic viewport intent: "the user was looking at node X at zoom Z on this lens."
 *
 * We persist intent instead of raw {x, y, zoom} because raw pixel coordinates are
 * fragile — they're only valid for the exact container size, layout mode, grouping,
 * and node set that produced them. A viewport captured on desktop restores off-screen
 * on mobile; a viewport captured before a layout change restores out of bounds.
 * An intent survives all of those: on restore, if the node still exists we re-center
 * on it at the saved zoom; if not, we fitView.
 *
 * `nodeId` is null when the user has no selected focus (panned/zoomed without a
 * selection). In that case only zoom is meaningful on restore.
 */
export interface ViewportIntent {
  nodeId: string | null;
  zoom: number;
}

const LAYOUT_STORAGE_KEY = "swarm-manager.graph.layout";
const LAYOUT_DIRECTION_STORAGE_KEY = "swarm-manager.graph.layout-direction";
const VIEWPORT_INTENT_STORAGE_KEY = "swarm-manager.graph.viewport-intent.v1";
const SIDEBAR_STORAGE_KEY = "swarm-manager.graph.sidebar-collapsed";

const LAYOUT_CYCLE: LayoutMode[] = ["hierarchical", "compact", "grouped"];

type ViewportIntentByLens = Record<GraphLens, ViewportIntent | null>;

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

function createEmptyViewportIntentByLens(): ViewportIntentByLens {
  return {
    plan: null,
    focus: null,
    topology: null,
  };
}

function isViewportIntent(value: unknown): value is ViewportIntent {
  if (!value || typeof value !== "object") {
    return false;
  }

  const record = value as Record<string, unknown>;
  const nodeIdValid = record.nodeId === null || typeof record.nodeId === "string";
  const zoomValid = typeof record.zoom === "number" && Number.isFinite(record.zoom) && record.zoom > 0;
  return nodeIdValid && zoomValid;
}

function loadViewportIntentByLens(): ViewportIntentByLens {
  const empty = createEmptyViewportIntentByLens();
  try {
    const raw = window.localStorage.getItem(VIEWPORT_INTENT_STORAGE_KEY);
    if (!raw) return empty;

    const parsed = JSON.parse(raw) as Record<string, unknown>;
    const next = createEmptyViewportIntentByLens();

    if (isViewportIntent(parsed.focus)) {
      next.focus = parsed.focus;
    }
    if (isViewportIntent(parsed.topology)) {
      next.topology = parsed.topology;
    }

    return next;
  } catch {
    return empty;
  }
}

// PERF: Debounce intent persistence. onMoveEnd fires frequently during pan/zoom.
// Synchronous JSON.stringify + localStorage.setItem on every call blocks the
// main thread. We batch writes with a 500ms debounce.
let viewportIntentSaveTimer: ReturnType<typeof setTimeout> | null = null;
function saveViewportIntentByLens(intentByLens: ViewportIntentByLens): void {
  if (viewportIntentSaveTimer) clearTimeout(viewportIntentSaveTimer);
  viewportIntentSaveTimer = setTimeout(() => {
    try {
      window.localStorage.setItem(VIEWPORT_INTENT_STORAGE_KEY, JSON.stringify(intentByLens));
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

export interface GraphUIState {
  selectedNodeId: string | null;
  highlightState: NodeHighlightState;
  layoutMode: LayoutMode;
  layoutPreferences: Record<string, LayoutMode>;
  layoutDirection: LayoutDirection;
  fitViewNonce: number;
  viewportIntentByLens: ViewportIntentByLens;
  sidebarCollapsed: boolean;
  focusNodeLabel: string | null;
  /** Runtime-only ref to the React Flow instance. NOT persisted to localStorage.
   *  Set by GraphCanvas on init; consumed by GraphNavControls for viewport manipulation. */
  flowInstance: ReactFlowInstance<GraphNode> | null;
  selectNode: (nodeId: string | null) => void;
  setHighlightState: (state: NodeHighlightState) => void;
  setLayoutMode: (mode: LayoutMode) => void;
  cycleLayoutMode: (lens: string) => void;
  setLayoutForLens: (lens: string, mode: LayoutMode) => void;
  applyLayoutForLens: (lens: string) => void;
  getLayoutForLens: (lens: string) => LayoutMode;
  setLayoutDirection: (direction: LayoutDirection) => void;
  requestFitView: () => void;
  setViewportIntentForLens: (lens: GraphLens, intent: ViewportIntent | null) => void;
  toggleSidebar: () => void;
  setSidebarCollapsed: (collapsed: boolean) => void;
  setFocusNodeLabel: (label: string | null) => void;
  setFlowInstance: (instance: ReactFlowInstance<GraphNode> | null) => void;
}

const initialPrefs = typeof window !== "undefined" ? loadLayoutPreferences() : {};
const initialViewportIntentByLens = typeof window !== "undefined" ? loadViewportIntentByLens() : createEmptyViewportIntentByLens();
const initialSidebarCollapsed = typeof window !== "undefined" ? loadSidebarCollapsed() : false;
const initialLayoutDirection = typeof window !== "undefined" ? loadLayoutDirection() : "TB";

export const graphUIInitialState = {
  selectedNodeId: null as string | null,
  highlightState: { highlighted: new Set<string>(), mode: "normal" as HighlightMode },
  layoutMode: "hierarchical" as LayoutMode,
  layoutPreferences: initialPrefs,
  layoutDirection: initialLayoutDirection,
  fitViewNonce: 0,
  viewportIntentByLens: initialViewportIntentByLens,
  sidebarCollapsed: initialSidebarCollapsed,
  focusNodeLabel: null as string | null,
  flowInstance: null as ReactFlowInstance<GraphNode> | null,
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

  setViewportIntentForLens: (lens, intent) => {
    // PERF: Skip state update if intent hasn't meaningfully changed. onMoveEnd
    // fires frequently during pan/zoom gestures; each set() triggers Zustand
    // notifications to all subscribers.
    const current = get().viewportIntentByLens[lens];
    if (intent === null && current === null) {
      return;
    }
    if (
      intent
      && current
      && current.nodeId === intent.nodeId
      && Math.abs(current.zoom - intent.zoom) < 0.001
    ) {
      return;
    }
    set((state) => {
      const viewportIntentByLens = {
        ...state.viewportIntentByLens,
        [lens]: intent,
      };
      saveViewportIntentByLens(viewportIntentByLens);
      return { viewportIntentByLens };
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

  setFocusNodeLabel: (label) => set({ focusNodeLabel: label }),

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
    viewportIntentByLens: {
      plan: graphUIInitialState.viewportIntentByLens.plan ? { ...graphUIInitialState.viewportIntentByLens.plan } : null,
      focus: graphUIInitialState.viewportIntentByLens.focus ? { ...graphUIInitialState.viewportIntentByLens.focus } : null,
      topology: graphUIInitialState.viewportIntentByLens.topology ? { ...graphUIInitialState.viewportIntentByLens.topology } : null,
    },
    // Runtime-only — always null in clones (tests, resets).
    flowInstance: null,
  };
}
