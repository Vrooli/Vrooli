/**
 * Detail Selection Store
 *
 * Manages which entity detail page is currently open as a full-page overlay
 * on top of the graph workspace. Selections are mutually exclusive — only
 * one detail page can be open at a time.
 *
 * This is separate from graph-ui-store's selectedNodeId, which controls
 * graph highlighting. Detail selection controls the full-page overlay.
 */

import { create } from "zustand";
import { buildBacklogNodeId, buildExecutionNodeId } from "../surfaces/graph/lib/node-id-parser";

export type DetailEntityType = "backlog" | "scenario" | "execution" | "initiative";

export interface DetailSelection {
  entityType: DetailEntityType;
  /** Backlog kind (execute, research, etc.). Only set for backlog entities. */
  kind?: string;
  /** Entity name. Used by backlog, scenario, and initiative. */
  name?: string;
  /** Execution ID. Only set for execution entities. */
  identifier?: string;
  /** Active tab within the detail page (e.g., "files", "workshop"). */
  tab?: string;
}

export interface DetailSelectionStore {
  selection: DetailSelection | null;

  selectBacklog: (kind: string, name: string, tab?: string) => void;
  selectScenario: (name: string, tab?: string) => void;
  selectExecution: (executionId: string) => void;
  selectInitiative: (name: string, tab?: string) => void;
  setTab: (tab: string | null) => void;
  clearSelection: () => void;

  /**
   * Hydrate from URL params without triggering a URL write-back.
   * Used by useDetailUrlSync on mount to avoid a sync loop.
   */
  _hydrate: (selection: DetailSelection | null) => void;
}

export const useDetailSelectionStore = create<DetailSelectionStore>((set) => ({
  selection: null,

  selectBacklog: (kind, name, tab) =>
    set({ selection: { entityType: "backlog", kind, name, tab } }),

  selectScenario: (name, tab) =>
    set({ selection: { entityType: "scenario", name, tab } }),

  selectExecution: (executionId) =>
    set({ selection: { entityType: "execution", identifier: executionId } }),

  selectInitiative: (name, tab) =>
    set({ selection: { entityType: "initiative", name, tab } }),

  setTab: (tab) =>
    set((state) => {
      if (!state.selection) return state;
      return { selection: { ...state.selection, tab: tab ?? undefined } };
    }),

  clearSelection: () => set({ selection: null }),

  _hydrate: (selection) => set({ selection }),
}));

/** Build the canonical graph node ID for the current detail selection. */
export function selectionToNodeId(selection: DetailSelection | null): string | null {
  if (!selection) return null;
  switch (selection.entityType) {
    case "backlog":
      return selection.kind && selection.name
        ? buildBacklogNodeId(selection.kind, selection.name)
        : null;
    case "scenario":
      return selection.name ? `scenario/${selection.name}` : null;
    case "execution":
      return selection.identifier ? buildExecutionNodeId(selection.identifier) : null;
    case "initiative":
      return selection.name ? `initiative/${selection.name}` : null;
    default:
      return null;
  }
}
