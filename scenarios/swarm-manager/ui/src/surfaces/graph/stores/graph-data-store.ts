/**
 * Graph Data Store
 *
 * Owns the API-backed graph projection, per-lens cached snapshots, graph
 * metadata, and persisted per-lens display settings. Interaction state such as
 * selection and viewport lives in graph-ui-store.
 */

import { create } from "zustand";
import { backlogService, graphService, type GraphProjectionMeta } from "../../../services";
import { GRAPH_ENTITY_TYPES } from "../lib/entity-shapes";
import { computeNodeAttention, type NodeEnrichment } from "../lib/attention";
import { useSnoozeStore } from "../../../stores/snooze-store";
import {
  ENTITY_STATUS_REGISTRY,
  getGraphNodeData,
  type GraphEdge,
  type GraphEntityType,
  type GraphGroupingMode,
  type GraphLens,
  type GraphNode,
} from "../types";

export type { GraphEdge, GraphGroupingMode, GraphLens, GraphNode };
export type EntityType = GraphEntityType;

export interface GraphLensSettings {
  entityFilters: Record<EntityType, boolean>;
  /** Per-entity-type status visibility. Outer key = entity type, inner key = status string. */
  statusFilters: Record<string, Record<string, boolean>>;
  groupingMode: GraphGroupingMode;
  showSecondaryEdges: boolean;
  autoFitOnChange: boolean;
  showMiniMap: boolean;
  /** Show on-screen pan/zoom controls for TV and accessibility. */
  showNavControls: boolean;
  /** Pulse nodes that need attention (via computeNodeAttention). Not applicable to focus lens. */
  highlightActionableNodes: boolean;
}

interface GraphLensSnapshot {
  nodes: GraphNode[];
  edges: GraphEdge[];
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
  nodes: GraphNode[];
  edges: GraphEdge[];
  meta: GraphProjectionMeta | null;
  loading: boolean;
  error: string | null;
  lens: GraphLens;
  focusNodeId: string | null;
  returnLens: GraphLens | null;
  graphsByLens: Record<GraphLens, GraphLensSnapshot>;
  settingsByLens: Record<GraphLens, GraphLensSettings>;
  setNodes: (nodes: GraphNode[]) => void;
  setEdges: (edges: GraphEdge[]) => void;
  setGraphData: (nodes: GraphNode[], edges: GraphEdge[], meta?: GraphProjectionMeta | null) => void;
  setLens: (lens: GraphLens) => void;
  setFocusNode: (nodeId: string | null) => void;
  setReturnLens: (lens: GraphLens | null) => void;
  fetchGraph: (lens?: GraphLens, options?: FetchGraphOptions) => Promise<void>;
  toggleEntityFilter: (type: EntityType) => void;
  setEntityFilter: (type: EntityType, visible: boolean) => void;
  setStatusVisibility: (entityType: string, status: string, visible: boolean) => void;
  clearStatusFilter: (entityType: string, status: string) => void;
  setEntityStatusGroupVisibility: (entityType: string, statuses: readonly string[], visible: boolean) => void;
  setGroupingMode: (mode: GraphGroupingMode) => void;
  setShowSecondaryEdges: (visible: boolean) => void;
  setShowMiniMap: (visible: boolean) => void;
  setShowNavControls: (visible: boolean) => void;
  setAutoFitOnChange: (enabled: boolean) => void;
  setHighlightActionableNodes: (enabled: boolean) => void;
  resetLensSettings: (lens?: GraphLens) => void;
  setNodePulsing: (nodeId: string, pulsing: boolean) => void;
}

const GRAPH_SETTINGS_STORAGE_KEY = "swarm-manager.graph.settings.v5";
const LEGACY_GRAPH_SETTINGS_STORAGE_KEYS = ["swarm-manager.graph.settings.v4", "swarm-manager.graph.settings.v3", "swarm-manager.graph.settings.v2"];
const GRAPH_SNAPSHOT_STALE_MS = 30_000;

const graphRequestSequence: Record<GraphLens, number> = {
  focus: 0,
  topology: 0,
  operations: 0,
};

const graphAbortControllers = new Map<GraphLens, AbortController>();
const graphInFlightRequests = new Map<GraphLens, Promise<void>>();

const DEFAULT_ENTITY_FILTERS: Record<EntityType, boolean> = Object.fromEntries(
  GRAPH_ENTITY_TYPES.map((et) => [et, true]),
) as Record<EntityType, boolean>;

function cloneEntityFilters(): Record<EntityType, boolean> {
  return { ...DEFAULT_ENTITY_FILTERS };
}

export function createDefaultLensSettings(lens: GraphLens): GraphLensSettings {
  return {
    entityFilters: cloneEntityFilters(),
    statusFilters: {},
    groupingMode: lens === "topology" ? "initiative" : "none",
    showSecondaryEdges: lens !== "focus",
    autoFitOnChange: true,
    showMiniMap: false,
    showNavControls: false,
    highlightActionableNodes: lens !== "focus",
  };
}

function createDefaultSettingsByLens(): Record<GraphLens, GraphLensSettings> {
  return {
    focus: createDefaultLensSettings("focus"),
    topology: createDefaultLensSettings("topology"),
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
    showMiniMap: settings.showMiniMap,
    showNavControls: settings.showNavControls,
    highlightActionableNodes: settings.highlightActionableNodes,
  };
}

function cloneSettingsByLens(
  settingsByLens: Record<GraphLens, GraphLensSettings>,
): Record<GraphLens, GraphLensSettings> {
  return {
    focus: cloneLensSettings(settingsByLens.focus),
    topology: cloneLensSettings(settingsByLens.topology),
    operations: cloneLensSettings(settingsByLens.operations),
  };
}

const ENTITY_TYPE_SET: ReadonlySet<string> = new Set(GRAPH_ENTITY_TYPES);

function isEntityType(value: unknown): value is EntityType {
  return typeof value === "string" && ENTITY_TYPE_SET.has(value);
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

    for (const lens of ["focus", "topology", "operations"] as const) {
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

      const statusFilters: Record<string, Record<string, boolean>> = {};
      if (typeof record.statusFilters === "object" && record.statusFilters !== null) {
        const rawStatuses = record.statusFilters as Record<string, unknown>;
        const firstValue = Object.values(rawStatuses)[0];

        if (typeof firstValue === "boolean") {
          // v3 flat format — broadcast each status to all entity types that include it
          for (const [status, value] of Object.entries(rawStatuses)) {
            if (typeof value !== "boolean") continue;
            for (const [et, knownStatuses] of Object.entries(ENTITY_STATUS_REGISTRY)) {
              if (knownStatuses && (knownStatuses as readonly string[]).includes(status)) {
                statusFilters[et] = statusFilters[et] || {};
                statusFilters[et][status] = value;
              }
            }
          }
        } else {
          // v4 grouped format — parse directly
          for (const [entityType, group] of Object.entries(rawStatuses)) {
            if (typeof group === "object" && group !== null) {
              const parsed: Record<string, boolean> = {};
              for (const [status, value] of Object.entries(group as Record<string, unknown>)) {
                if (typeof value === "boolean") parsed[status] = value;
              }
              if (Object.keys(parsed).length > 0) statusFilters[entityType] = parsed;
            }
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
        showMiniMap:
          typeof record.showMiniMap === "boolean"
            ? record.showMiniMap
            : defaults[lens].showMiniMap,
        showNavControls:
          typeof record.showNavControls === "boolean"
            ? record.showNavControls
            : defaults[lens].showNavControls,
        highlightActionableNodes:
          typeof record.highlightActionableNodes === "boolean"
            ? record.highlightActionableNodes
            : defaults[lens].highlightActionableNodes,
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
    focus: createEmptyLensSnapshot(),
    topology: createEmptyLensSnapshot(),
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
    focus: cloneLensSnapshot(graphsByLens.focus),
    topology: cloneLensSnapshot(graphsByLens.topology),
    operations: cloneLensSnapshot(graphsByLens.operations),
  };
}

function mergeRuntimeNodeState(currentNodes: GraphNode[], nextNodes: GraphNode[]): GraphNode[] {
  const pulsingById = new Map<string, boolean>();
  for (const node of currentNodes) {
    const pulsing = getGraphNodeData(node).pulsing;
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
        ...getGraphNodeData(node),
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
    nodes: [] as GraphNode[],
    edges: [] as GraphEdge[],
    meta: null as GraphProjectionMeta | null,
    loading: false,
    error: null as string | null,
    lens: "topology" as GraphLens,
    focusNodeId: null as string | null,
    returnLens: null as GraphLens | null,
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
  graphRequestSequence.focus = 0;
  graphRequestSequence.topology = 0;
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

  setFocusNode: (nodeId) => set({ focusNodeId: nodeId }),
  setReturnLens: (lens) => set({ returnLens: lens }),

  fetchGraph: async (lensArg, options) => {
    const lens = lensArg ?? get().lens;

    // Focus lens: client-side filter from topology data.
    if (lens === "focus") {
      const topoSnapshot = get().graphsByLens.topology;
      // Ensure topology data is fresh first.
      if (!isSnapshotFresh(topoSnapshot, options?.force)) {
        await get().fetchGraph("topology", options);
      }

      const freshTopo = get().graphsByLens.topology;
      const snoozedKeys = useSnoozeStore.getState().snoozedKeys();

      // Fetch enrichment data (feedback + maturity) so we can detect pending
      // decisions, workshop-needed, and maturity-ready states — matching the
      // same conditions the Command Post uses.
      const enrichmentMap = new Map<string, NodeEnrichment>();
      try {
        const summary = await backlogService.getBacklogSummary();
        const feedbackByKey = new Map(
          (summary.feedback?.items ?? []).map((f) => [`${f.kind}/${f.name}`, f]),
        );
        const maturityByKey = new Map(
          (summary.maturity?.items ?? []).map((m) => [`${m.kind}/${m.name}`, m]),
        );
        for (const key of new Set([...feedbackByKey.keys(), ...maturityByKey.keys()])) {
          const fb = feedbackByKey.get(key);
          const mat = maturityByKey.get(key);
          enrichmentMap.set(key, {
            pendingDecisions: fb?.pending_decisions ?? 0,
            maturityReady: mat ? (mat.ready ?? null) : null,
            pendingSynthesis: mat?.pending_synthesis ?? false,
          });
        }
      } catch {
        // If summary fetch fails, fall back to status-only filtering.
      }

      const filteredNodeIds = new Set<string>();
      const filteredNodes: GraphNode[] = [];

      for (const node of freshTopo.nodes) {
        const data = getGraphNodeData(node);
        // Build enrichment for backlog nodes from the summary data.
        let enrichment: NodeEnrichment | undefined;
        if (data.entityType === "backlog" && "kind" in data && "name" in data) {
          enrichment = enrichmentMap.get(`${data.kind}/${data.name}`);
        }
        const result = computeNodeAttention(data, enrichment, snoozedKeys);
        if (result.needsAttention) {
          filteredNodeIds.add(node.id);
          filteredNodes.push({
            ...node,
            data: {
              ...data,
              pulsing: true,
              pulseMode: "persistent" as const,
            },
          });
        }
      }

      const filteredEdges = freshTopo.edges.filter(
        (edge) => filteredNodeIds.has(edge.source) && filteredNodeIds.has(edge.target),
      );

      set((state) =>
        updateLensSnapshot(state, "focus", () => ({
          nodes: filteredNodes,
          edges: filteredEdges,
          meta: freshTopo.meta,
          loading: false,
          error: null,
          fetchedAtMs: Date.now(),
        })),
      );
      return;
    }

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

    const focusNodeId = get().focusNodeId;
    const requestPromise = graphService
      .getGraph(lens, {
        signal: controller.signal,
        focusNodeId: lens === "operations" ? (focusNodeId ?? undefined) : undefined,
      })
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

  setStatusVisibility: (entityType, status, visible) =>
    set((state) =>
      updateLensSettings(state, state.lens, (settings) => ({
        ...settings,
        statusFilters: {
          ...settings.statusFilters,
          [entityType]: {
            ...settings.statusFilters[entityType],
            [status]: visible,
          },
        },
      })),
    ),

  clearStatusFilter: (entityType, status) =>
    set((state) =>
      updateLensSettings(state, state.lens, (settings) => {
        const entityGroup = { ...settings.statusFilters[entityType] };
        delete entityGroup[status];
        const nextFilters = { ...settings.statusFilters };
        if (Object.keys(entityGroup).length === 0) {
          delete nextFilters[entityType];
        } else {
          nextFilters[entityType] = entityGroup;
        }
        return { ...settings, statusFilters: nextFilters };
      }),
    ),

  setEntityStatusGroupVisibility: (entityType, statuses, visible) =>
    set((state) =>
      updateLensSettings(state, state.lens, (settings) => {
        const entityGroup = { ...settings.statusFilters[entityType] };
        for (const status of statuses) {
          entityGroup[status] = visible;
        }
        return {
          ...settings,
          statusFilters: {
            ...settings.statusFilters,
            [entityType]: entityGroup,
          },
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

  setShowMiniMap: (visible) =>
    set((state) =>
      updateLensSettings(state, state.lens, (settings) => ({
        ...settings,
        showMiniMap: visible,
      })),
    ),

  setShowNavControls: (visible) =>
    set((state) =>
      updateLensSettings(state, state.lens, (settings) => ({
        ...settings,
        showNavControls: visible,
      })),
    ),

  setAutoFitOnChange: (enabled) =>
    set((state) =>
      updateLensSettings(state, state.lens, (settings) => ({
        ...settings,
        autoFitOnChange: enabled,
      })),
    ),

  setHighlightActionableNodes: (enabled) =>
    set((state) =>
      updateLensSettings(state, state.lens, (settings) => ({
        ...settings,
        highlightActionableNodes: enabled,
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
                  ...getGraphNodeData(node),
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
