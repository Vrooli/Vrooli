/**
 * Graph Data Store
 *
 * Owns the API-backed graph projection, per-lens cached snapshots, graph
 * metadata, and persisted per-lens display settings. Interaction state such as
 * selection and viewport lives in graph-ui-store.
 */

import { create } from "zustand";
import type { Edge, Node } from "@xyflow/react";
import { graphService, type GraphProjectionMeta } from "../../../services";

export type GraphLens = "topology" | "flow" | "operations";

export type EntityType =
  | "backlog"
  | "scenario"
  | "execution"
  | "capture"
  | "agent-run"
  | "initiative";

export type GraphGroupingMode = "initiative" | "none";

export interface GraphLensSettings {
  entityFilters: Record<EntityType, boolean>;
  statusFilters: Record<string, boolean>;
  groupingMode: GraphGroupingMode;
  showSecondaryEdges: boolean;
  autoFitOnChange: boolean;
}

interface GraphLensSnapshot {
  nodes: Node[];
  edges: Edge[];
  meta: GraphProjectionMeta | null;
  loading: boolean;
  error: string | null;
  fetchedAtMs: number | null;
}

export interface FetchGraphOptions {
  silent?: boolean;
  force?: boolean;
}

export interface GraphDataState {
  nodes: Node[];
  edges: Edge[];
  meta: GraphProjectionMeta | null;
  loading: boolean;
  error: string | null;
  lens: GraphLens;
  graphsByLens: Record<GraphLens, GraphLensSnapshot>;
  settingsByLens: Record<GraphLens, GraphLensSettings>;
  setNodes: (nodes: Node[]) => void;
  setEdges: (edges: Edge[]) => void;
  setGraphData: (nodes: Node[], edges: Edge[], meta?: GraphProjectionMeta | null) => void;
  setLens: (lens: GraphLens) => void;
  fetchGraph: (lens?: GraphLens, options?: FetchGraphOptions) => Promise<void>;
  toggleEntityFilter: (type: EntityType) => void;
  setEntityFilter: (type: EntityType, visible: boolean) => void;
  setStatusVisibility: (status: string, visible: boolean) => void;
  clearStatusFilter: (status: string) => void;
  setGroupingMode: (mode: GraphGroupingMode) => void;
  setShowSecondaryEdges: (visible: boolean) => void;
  setAutoFitOnChange: (enabled: boolean) => void;
  resetLensSettings: (lens?: GraphLens) => void;
  setNodePulsing: (nodeId: string, pulsing: boolean) => void;
}

const GRAPH_SETTINGS_STORAGE_KEY = "swarm-manager.graph.settings.v3";
const LEGACY_GRAPH_SETTINGS_STORAGE_KEYS = ["swarm-manager.graph.settings.v2"];
const GRAPH_SNAPSHOT_STALE_MS = 30_000;

const graphRequestSequence: Record<GraphLens, number> = {
  topology: 0,
  flow: 0,
  operations: 0,
};

const graphAbortControllers = new Map<GraphLens, AbortController>();
const graphInFlightRequests = new Map<GraphLens, Promise<void>>();

const DEFAULT_ENTITY_FILTERS: Record<EntityType, boolean> = {
  backlog: true,
  scenario: true,
  execution: true,
  capture: true,
  "agent-run": true,
  initiative: true,
};

function cloneEntityFilters(): Record<EntityType, boolean> {
  return { ...DEFAULT_ENTITY_FILTERS };
}

export function createDefaultLensSettings(_lens: GraphLens): GraphLensSettings {
  return {
    entityFilters: cloneEntityFilters(),
    statusFilters: {},
    groupingMode: "none",
    showSecondaryEdges: true,
    autoFitOnChange: true,
  };
}

function createDefaultSettingsByLens(): Record<GraphLens, GraphLensSettings> {
  return {
    topology: createDefaultLensSettings("topology"),
    flow: createDefaultLensSettings("flow"),
    operations: createDefaultLensSettings("operations"),
  };
}

function cloneLensSettings(settings: GraphLensSettings): GraphLensSettings {
  return {
    entityFilters: { ...settings.entityFilters },
    statusFilters: { ...settings.statusFilters },
    groupingMode: settings.groupingMode,
    showSecondaryEdges: settings.showSecondaryEdges,
    autoFitOnChange: settings.autoFitOnChange,
  };
}

function cloneSettingsByLens(
  settingsByLens: Record<GraphLens, GraphLensSettings>,
): Record<GraphLens, GraphLensSettings> {
  return {
    topology: cloneLensSettings(settingsByLens.topology),
    flow: cloneLensSettings(settingsByLens.flow),
    operations: cloneLensSettings(settingsByLens.operations),
  };
}

function isEntityType(value: unknown): value is EntityType {
  return (
    value === "backlog" ||
    value === "scenario" ||
    value === "execution" ||
    value === "capture" ||
    value === "agent-run" ||
    value === "initiative"
  );
}

function loadPersistedSettings(): Record<GraphLens, GraphLensSettings> {
  const defaults = createDefaultSettingsByLens();

  if (typeof window === "undefined") {
    return defaults;
  }

  try {
    let raw = window.localStorage.getItem(GRAPH_SETTINGS_STORAGE_KEY);
    let migratingLegacySettings = false;
    if (!raw) {
      for (const key of LEGACY_GRAPH_SETTINGS_STORAGE_KEYS) {
        raw = window.localStorage.getItem(key);
        if (raw) {
          migratingLegacySettings = true;
          break;
        }
      }
    }
    if (!raw) return defaults;

    const parsed = JSON.parse(raw) as Record<string, unknown>;
    const next = createDefaultSettingsByLens();

    for (const lens of ["topology", "flow", "operations"] as const) {
      const lensValue = parsed[lens];
      if (typeof lensValue !== "object" || lensValue === null) {
        continue;
      }

      const record = lensValue as Record<string, unknown>;
      const entityFilters = cloneEntityFilters();
      if (typeof record.entityFilters === "object" && record.entityFilters !== null) {
        const filters = record.entityFilters as Record<string, unknown>;
        for (const [key, value] of Object.entries(filters)) {
          if (isEntityType(key) && typeof value === "boolean") {
            entityFilters[key] = value;
          }
        }
      }

      const statusFilters: Record<string, boolean> = {};
      if (typeof record.statusFilters === "object" && record.statusFilters !== null) {
        const statuses = record.statusFilters as Record<string, unknown>;
        for (const [status, value] of Object.entries(statuses)) {
          if (typeof value === "boolean") {
            statusFilters[status] = value;
          }
        }
      }

      next[lens] = {
        entityFilters,
        statusFilters,
        groupingMode:
          !migratingLegacySettings &&
          (record.groupingMode === "initiative" || record.groupingMode === "none")
            ? record.groupingMode
            : defaults[lens].groupingMode,
        showSecondaryEdges:
          typeof record.showSecondaryEdges === "boolean"
            ? record.showSecondaryEdges
            : defaults[lens].showSecondaryEdges,
        autoFitOnChange:
          typeof record.autoFitOnChange === "boolean"
            ? record.autoFitOnChange
            : defaults[lens].autoFitOnChange,
      };
    }

    return next;
  } catch {
    return defaults;
  }
}

function savePersistedSettings(settingsByLens: Record<GraphLens, GraphLensSettings>): void {
  if (typeof window === "undefined") {
    return;
  }

  try {
    window.localStorage.setItem(GRAPH_SETTINGS_STORAGE_KEY, JSON.stringify(settingsByLens));
  } catch {
    // Ignore persistence failures and continue in-memory.
  }
}

function createEmptyLensSnapshot(): GraphLensSnapshot {
  return {
    nodes: [],
    edges: [],
    meta: null,
    loading: false,
    error: null,
    fetchedAtMs: null,
  };
}

function createEmptyGraphsByLens(): Record<GraphLens, GraphLensSnapshot> {
  return {
    topology: createEmptyLensSnapshot(),
    flow: createEmptyLensSnapshot(),
    operations: createEmptyLensSnapshot(),
  };
}

function cloneLensSnapshot(snapshot: GraphLensSnapshot): GraphLensSnapshot {
  return {
    nodes: [...snapshot.nodes],
    edges: [...snapshot.edges],
    meta: snapshot.meta ? { ...snapshot.meta } : null,
    loading: snapshot.loading,
    error: snapshot.error,
    fetchedAtMs: snapshot.fetchedAtMs,
  };
}

function cloneGraphsByLens(
  graphsByLens: Record<GraphLens, GraphLensSnapshot>,
): Record<GraphLens, GraphLensSnapshot> {
  return {
    topology: cloneLensSnapshot(graphsByLens.topology),
    flow: cloneLensSnapshot(graphsByLens.flow),
    operations: cloneLensSnapshot(graphsByLens.operations),
  };
}

function mergeRuntimeNodeState(currentNodes: Node[], nextNodes: Node[]): Node[] {
  const pulsingById = new Map<string, boolean>();
  for (const node of currentNodes) {
    const pulsing = (node.data as Record<string, unknown> | undefined)?.pulsing;
    if (typeof pulsing === "boolean") {
      pulsingById.set(node.id, pulsing);
    }
  }

  return nextNodes.map((node) => {
    const pulsing = pulsingById.get(node.id);
    if (pulsing === undefined) {
      return node;
    }
    return {
      ...node,
      data: {
        ...(node.data as Record<string, unknown>),
        pulsing,
      },
    };
  });
}

function updateLensSettings(
  state: GraphDataState,
  lens: GraphLens,
  updater: (settings: GraphLensSettings) => GraphLensSettings,
): Pick<GraphDataState, "settingsByLens"> {
  const nextSettings = {
    ...state.settingsByLens,
    [lens]: updater(state.settingsByLens[lens]),
  };
  savePersistedSettings(nextSettings);
  return { settingsByLens: nextSettings };
}

function syncActiveLensSnapshot(
  lens: GraphLens,
  snapshot: GraphLensSnapshot,
): Pick<GraphDataState, "lens" | "nodes" | "edges" | "meta" | "loading" | "error"> {
  return {
    lens,
    nodes: snapshot.nodes,
    edges: snapshot.edges,
    meta: snapshot.meta,
    loading: snapshot.loading,
    error: snapshot.error,
  };
}

function isAbortError(error: unknown): boolean {
  return error instanceof DOMException && error.name === "AbortError";
}

function isSnapshotFresh(snapshot: GraphLensSnapshot, force?: boolean): boolean {
  if (force || snapshot.meta === null || snapshot.fetchedAtMs === null) {
    return false;
  }
  return Date.now() - snapshot.fetchedAtMs < GRAPH_SNAPSHOT_STALE_MS;
}

function updateLensSnapshot(
  state: GraphDataState,
  lens: GraphLens,
  updater: (snapshot: GraphLensSnapshot) => GraphLensSnapshot,
): Partial<GraphDataState> {
  const nextSnapshot = updater(state.graphsByLens[lens]);
  const nextGraphsByLens = {
    ...state.graphsByLens,
    [lens]: nextSnapshot,
  };

  if (state.lens !== lens) {
    return { graphsByLens: nextGraphsByLens };
  }

  return {
    graphsByLens: nextGraphsByLens,
    ...syncActiveLensSnapshot(lens, nextSnapshot),
  };
}

export function createGraphDataInitialState() {
  return {
    nodes: [] as Node[],
    edges: [] as Edge[],
    meta: null as GraphProjectionMeta | null,
    loading: false,
    error: null as string | null,
    lens: "topology" as GraphLens,
    graphsByLens: createEmptyGraphsByLens(),
    settingsByLens:
      typeof window !== "undefined" ? loadPersistedSettings() : createDefaultSettingsByLens(),
  };
}

export const graphDataInitialState = createGraphDataInitialState();

export function resetGraphRequestState(): void {
  for (const controller of graphAbortControllers.values()) {
    controller.abort();
  }
  graphAbortControllers.clear();
  graphInFlightRequests.clear();
  graphRequestSequence.topology = 0;
  graphRequestSequence.flow = 0;
  graphRequestSequence.operations = 0;
}

export const useGraphDataStore = create<GraphDataState>((set, get) => ({
  ...graphDataInitialState,

  setNodes: (nodes) =>
    set((state) =>
      updateLensSnapshot(state, state.lens, (snapshot) => ({
        ...snapshot,
        nodes: mergeRuntimeNodeState(snapshot.nodes, nodes),
      })),
    ),

  setEdges: (edges) =>
    set((state) =>
      updateLensSnapshot(state, state.lens, (snapshot) => ({
        ...snapshot,
        edges,
      })),
    ),

  setGraphData: (nodes, edges, meta = null) =>
    set((state) =>
      updateLensSnapshot(state, state.lens, (snapshot) => ({
        ...snapshot,
        nodes: mergeRuntimeNodeState(snapshot.nodes, nodes),
        edges,
        meta,
        error: null,
      })),
    ),

  setLens: (lens) =>
    set((state) => ({
      ...syncActiveLensSnapshot(lens, state.graphsByLens[lens]),
    })),

  fetchGraph: async (lensArg, options) => {
    const lens = lensArg ?? get().lens;
    const snapshot = get().graphsByLens[lens];

    if (isSnapshotFresh(snapshot, options?.force)) {
      return;
    }

    const existingRequest = graphInFlightRequests.get(lens);
    if (existingRequest && !options?.force) {
      return existingRequest;
    }

    graphAbortControllers.get(lens)?.abort();

    const controller = new AbortController();
    graphAbortControllers.set(lens, controller);

    graphRequestSequence[lens] += 1;
    const requestId = graphRequestSequence[lens];

    set((state) =>
      updateLensSnapshot(state, lens, (current) => ({
        ...current,
        loading: !options?.silent,
        error: options?.silent ? current.error : null,
      })),
    );

    const requestPromise = graphService
      .getGraph(lens, { signal: controller.signal })
      .then((graph) => {
        if (graphRequestSequence[lens] !== requestId) {
          return;
        }

        set((state) =>
          updateLensSnapshot(state, lens, (current) => ({
            ...current,
            nodes: mergeRuntimeNodeState(current.nodes, graph.nodes),
            edges: graph.edges,
            meta: graph.meta,
            loading: false,
            error: null,
            fetchedAtMs: Date.now(),
          })),
        );
      })
      .catch((error) => {
        if (graphRequestSequence[lens] !== requestId) {
          return;
        }

        if (isAbortError(error)) {
          set((state) =>
            updateLensSnapshot(state, lens, (current) => ({
              ...current,
              loading: false,
            })),
          );
          return;
        }

        set((state) =>
          updateLensSnapshot(state, lens, (current) => ({
            ...current,
            loading: false,
            error: error instanceof Error ? error.message : `Failed to load ${lens} graph`,
          })),
        );
      })
      .finally(() => {
        if (graphAbortControllers.get(lens) === controller) {
          graphAbortControllers.delete(lens);
        }
        if (graphInFlightRequests.get(lens) === requestPromise) {
          graphInFlightRequests.delete(lens);
        }
      });

    graphInFlightRequests.set(lens, requestPromise);
    return requestPromise;
  },

  toggleEntityFilter: (type) =>
    set((state) =>
      updateLensSettings(state, state.lens, (settings) => ({
        ...settings,
        entityFilters: {
          ...settings.entityFilters,
          [type]: !settings.entityFilters[type],
        },
      })),
    ),

  setEntityFilter: (type, visible) =>
    set((state) =>
      updateLensSettings(state, state.lens, (settings) => ({
        ...settings,
        entityFilters: {
          ...settings.entityFilters,
          [type]: visible,
        },
      })),
    ),

  setStatusVisibility: (status, visible) =>
    set((state) =>
      updateLensSettings(state, state.lens, (settings) => ({
        ...settings,
        statusFilters: {
          ...settings.statusFilters,
          [status]: visible,
        },
      })),
    ),

  clearStatusFilter: (status) =>
    set((state) =>
      updateLensSettings(state, state.lens, (settings) => {
        const nextFilters = { ...settings.statusFilters };
        delete nextFilters[status];
        return {
          ...settings,
          statusFilters: nextFilters,
        };
      }),
    ),

  setGroupingMode: (mode) =>
    set((state) =>
      updateLensSettings(state, state.lens, (settings) => ({
        ...settings,
        groupingMode: mode,
      })),
    ),

  setShowSecondaryEdges: (visible) =>
    set((state) =>
      updateLensSettings(state, state.lens, (settings) => ({
        ...settings,
        showSecondaryEdges: visible,
      })),
    ),

  setAutoFitOnChange: (enabled) =>
    set((state) =>
      updateLensSettings(state, state.lens, (settings) => ({
        ...settings,
        autoFitOnChange: enabled,
      })),
    ),

  resetLensSettings: (lensArg) =>
    set((state) => {
      const lens = lensArg ?? state.lens;
      const nextSettings = {
        ...state.settingsByLens,
        [lens]: createDefaultLensSettings(lens),
      };
      savePersistedSettings(nextSettings);
      return {
        settingsByLens: nextSettings,
      };
    }),

  setNodePulsing: (nodeId, pulsing) =>
    set((state) =>
      updateLensSnapshot(state, state.lens, (snapshot) => ({
        ...snapshot,
        nodes: snapshot.nodes.map((node) =>
          node.id === nodeId
            ? {
                ...node,
                data: {
                  ...(node.data as Record<string, unknown>),
                  pulsing,
                },
              }
            : node,
        ),
      })),
    ),
}));

export function cloneGraphDataInitialState(): typeof graphDataInitialState {
  const initialState = createGraphDataInitialState();
  return {
    ...initialState,
    graphsByLens: cloneGraphsByLens(initialState.graphsByLens),
    settingsByLens: cloneSettingsByLens(initialState.settingsByLens),
  };
}
