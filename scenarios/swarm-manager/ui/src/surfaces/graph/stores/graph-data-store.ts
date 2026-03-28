/**
 * Graph Data Store
 *
 * Owns graph nodes, edges, active lens, and entity-type filters.
 * Data-only — interaction state lives in graph-ui-store.
 */

import { create } from "zustand";
import type { Node, Edge } from "@xyflow/react";

export type GraphLens = "topology" | "flow" | "operations";

export type EntityType = "backlog" | "scenario" | "execution" | "capture" | "agent-run" | "initiative";

export interface GraphDataState {
  nodes: Node[];
  edges: Edge[];
  lens: GraphLens;
  /** Which entity types are visible in the graph and sidebar feed. */
  entityFilters: Record<EntityType, boolean>;
  setNodes: (nodes: Node[]) => void;
  setEdges: (edges: Edge[]) => void;
  setGraphData: (nodes: Node[], edges: Edge[]) => void;
  setLens: (lens: GraphLens) => void;
  toggleEntityFilter: (type: EntityType) => void;
  setEntityFilter: (type: EntityType, visible: boolean) => void;
  resetFilters: () => void;
}

const DEFAULT_FILTERS: Record<EntityType, boolean> = {
  backlog: true,
  scenario: true,
  execution: true,
  capture: true,
  "agent-run": true,
  initiative: true,
};

export const graphDataInitialState = {
  nodes: [] as Node[],
  edges: [] as Edge[],
  lens: "topology" as GraphLens,
  entityFilters: { ...DEFAULT_FILTERS },
};

export const useGraphDataStore = create<GraphDataState>((set) => ({
  ...graphDataInitialState,

  setNodes: (nodes) => set({ nodes }),

  setEdges: (edges) => set({ edges }),

  setGraphData: (nodes, edges) => set({ nodes, edges }),

  setLens: (lens) => set({ lens }),

  toggleEntityFilter: (type) =>
    set((state) => ({
      entityFilters: {
        ...state.entityFilters,
        [type]: !state.entityFilters[type],
      },
    })),

  setEntityFilter: (type, visible) =>
    set((state) => ({
      entityFilters: {
        ...state.entityFilters,
        [type]: visible,
      },
    })),

  resetFilters: () => set({ entityFilters: { ...DEFAULT_FILTERS } }),
}));
