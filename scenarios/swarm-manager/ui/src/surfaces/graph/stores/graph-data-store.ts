/**
 * Graph Data Store
 *
 * Owns the API-backed graph projection, per-lens cached snapshots, and graph
 * metadata. Interaction state such as selection and viewport lives in
 * graph-ui-store. Per-lens display settings live in graph-settings-store.
 */

import { create } from "zustand";
import { backlogService, graphService, type GraphProjectionMeta } from "../../../services";
import { computeNodeAttention, type NodeEnrichment } from "../lib/attention";
import {
  cloneGraphsByLens,
  createEmptyGraphsByLens,
  type GraphLensSnapshot,
} from "../lib/snapshot-utils";
import { useSnoozeStore } from "../../../stores/snooze-store";
import { reconcileEdges, reconcileNodes } from "../lib/structural-sharing";
import {
  getGraphNodeData,
  type GraphEdge,
  type GraphLens,
  type GraphNode,
} from "../types";
import { useGraphSettingsStore } from "./graph-settings-store";

export type { GraphEdge, GraphLens, GraphNode };

export interface FetchGraphOptions {
  silent?: boolean;
  force?: boolean;
}

// ---------------------------------------------------------------------------
// State interface
// ---------------------------------------------------------------------------

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
  setNodes: (nodes: GraphNode[]) => void;
  setEdges: (edges: GraphEdge[]) => void;
  setGraphData: (nodes: GraphNode[], edges: GraphEdge[], meta?: GraphProjectionMeta | null) => void;
  setLens: (lens: GraphLens) => void;
  setFocusNode: (nodeId: string | null) => void;
  setReturnLens: (lens: GraphLens | null) => void;
  fetchGraph: (lens?: GraphLens, options?: FetchGraphOptions) => Promise<void>;
  setNodePulsing: (nodeId: string, pulsing: boolean) => void;
}

// ---------------------------------------------------------------------------
// Module-level request tracking
// ---------------------------------------------------------------------------

const GRAPH_SNAPSHOT_STALE_MS = 30_000;

const graphRequestSequence: Record<GraphLens, number> = {
  plan: 0,
  focus: 0,
  topology: 0,
};

const graphAbortControllers = new Map<GraphLens, AbortController>();
const graphInFlightRequests = new Map<GraphLens, Promise<void>>();

// ---------------------------------------------------------------------------
// Snapshot helpers
// ---------------------------------------------------------------------------

// Structural-sharing reconciliation now lives in lib/structural-sharing.ts so
// unchanged nodes and edges keep their refs across polls. That lets downstream
// useMemo chains in the canvas skip work when the backend reports no-op data.

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

// ---------------------------------------------------------------------------
// Initial state
// ---------------------------------------------------------------------------

export function createGraphDataInitialState() {
  return {
    nodes: [] as GraphNode[],
    edges: [] as GraphEdge[],
    meta: null as GraphProjectionMeta | null,
    loading: false,
    error: null as string | null,
    lens: "plan" as GraphLens,
    focusNodeId: null as string | null,
    returnLens: null as GraphLens | null,
    graphsByLens: createEmptyGraphsByLens(),
  };
}

export const graphDataInitialState = createGraphDataInitialState();

export function resetGraphRequestState(): void {
  for (const controller of graphAbortControllers.values()) {
    controller.abort();
  }
  graphAbortControllers.clear();
  graphInFlightRequests.clear();
  graphRequestSequence.plan = 0;
  graphRequestSequence.focus = 0;
  graphRequestSequence.topology = 0;
}

// ---------------------------------------------------------------------------
// Store
// ---------------------------------------------------------------------------

export const useGraphDataStore = create<GraphDataState>((set, get) => ({
  ...graphDataInitialState,

  setNodes: (nodes) =>
    set((state) =>
      updateLensSnapshot(state, state.lens, (snapshot) => ({
        ...snapshot,
        nodes: reconcileNodes(snapshot.nodes, nodes),
      })),
    ),

  setEdges: (edges) =>
    set((state) =>
      updateLensSnapshot(state, state.lens, (snapshot) => ({
        ...snapshot,
        edges: reconcileEdges(snapshot.edges, edges),
      })),
    ),

  setGraphData: (nodes, edges, meta = null) =>
    set((state) =>
      updateLensSnapshot(state, state.lens, (snapshot) => ({
        ...snapshot,
        nodes: reconcileNodes(snapshot.nodes, nodes),
        edges: reconcileEdges(snapshot.edges, edges),
        meta,
        error: null,
      })),
    ),

  setLens: (lens) =>
    set((state) => {
      // Keep the settings store in sync so it operates on the correct lens.
      useGraphSettingsStore.getState().setActiveLens(lens);
      return syncActiveLensSnapshot(lens, state.graphsByLens[lens]);
    }),

  setFocusNode: (nodeId) => set({ focusNodeId: nodeId }),
  setReturnLens: (lens) => set({ returnLens: lens }),

  fetchGraph: async (lensArg, options) => {
    const lens = lensArg ?? get().lens;

    // Plan lens: the kanban board owns its data (plan-data-store fetching
    // GET /api/v1/plan). Delegating here lets the shared /ws/graph
    // invalidation path ("plan" in the lens payload) refresh the board
    // without a second socket.
    if (lens === "plan") {
      const { usePlanDataStore } = await import("../../plan/stores/plan-data-store");
      await usePlanDataStore.getState().fetchBoard({ silent: true, force: true });
      return;
    }

    // Focus mode: client-side filter from topology data.
    if (lens === "focus") {
      const topoSnapshot = get().graphsByLens.topology;
      // Ensure topology data is fresh first.
      if (!isSnapshotFresh(topoSnapshot, options?.force)) {
        await get().fetchGraph("topology", options);
      }

      const freshTopo = get().graphsByLens.topology;
      const snoozedKeys = useSnoozeStore.getState().snoozedKeys();

      // Fetch enrichment data (feedback + maturity) so we can detect pending
      // decisions, workshop-needed, and maturity-ready states.
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
        // Initiatives and scenarios are structural context — they have no
        // attention state of their own. They're added in a second pass
        // below, pulled in whenever they connect to an attention-worthy item.
        if (data.entityType === "initiative" || data.entityType === "scenario") continue;
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

      // Second pass: pull in structural context nodes (initiatives, scenarios)
      // that connect to any attention-worthy item. Initiatives are targets of
      // member_of edges; scenarios are targets of "targets" edges. Without this,
      // focus lens drops the surrounding context and shows bare backlog items.
      const contextNodeIds = new Set<string>();
      for (const edge of freshTopo.edges) {
        if (!filteredNodeIds.has(edge.source)) continue;
        if (edge.type === "member_of" || edge.type === "targets") {
          contextNodeIds.add(edge.target);
        }
      }
      for (const node of freshTopo.nodes) {
        const entityType = getGraphNodeData(node).entityType;
        if (entityType !== "initiative" && entityType !== "scenario") continue;
        if (!contextNodeIds.has(node.id)) continue;
        filteredNodeIds.add(node.id);
        filteredNodes.push(node);
      }

      const filteredEdges = freshTopo.edges.filter(
        (edge) => filteredNodeIds.has(edge.source) && filteredNodeIds.has(edge.target),
      );

      set((state) =>
        updateLensSnapshot(state, "focus", (current) => ({
          nodes: reconcileNodes(current.nodes, filteredNodes),
          edges: reconcileEdges(current.edges, filteredEdges),
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

    const requestPromise = graphService
      .getGraph(lens, {
        signal: controller.signal,
      })
      .then((graph) => {
        if (graphRequestSequence[lens] !== requestId) {
          return;
        }

        set((state) =>
          updateLensSnapshot(state, lens, (current) => ({
            ...current,
            nodes: reconcileNodes(current.nodes, graph.nodes),
            edges: reconcileEdges(current.edges, graph.edges),
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
  };
}
