/**
 * PhaseGraph
 *
 * Interactive DAG of an operating mode's phases. xyflow + dagre top-down
 * layout, rendered inside a fixed-height surface above the PhaseList.
 *
 * Edges are colored by transition kind (slate=always, amber=payload_bool,
 * cyan=progress_decision) and labeled server-side so the CLI and UI emit
 * identical strings (e.g. "on payload.replan_needed=true", "on continue").
 *
 * Clicking a node calls onSelectPhase(phase) so the page can highlight the
 * matching PhaseCard and scroll it into view.
 */

import { useEffect, useMemo, useState } from "react";
import {
  ReactFlow,
  ReactFlowProvider,
  Background,
  Controls,
  MiniMap,
  MarkerType,
  type Edge,
  useNodesState,
  useEdgesState,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import dagre from "dagre";
import { selectors } from "../../../consts/selectors";
import type {
  OperatingModeCatalogEntry,
  OperatingModeCatalogPhase,
  OperatingModePhaseTransition,
  OperatingModeRound,
} from "../../../types/operating-mode";
import { PhaseNode, type PhaseNodeData, type PhaseNodeType } from "./phase-node";
import { PhaseGraphGlossaryDialog } from "./phase-graph-glossary-dialog";

export interface PhaseStateMap {
  [phase: string]: { startable?: boolean; reason?: string; isNext?: boolean };
}

const NODE_WIDTH = 180;
const NODE_HEIGHT = 76;

const NODE_TYPES = { phase: PhaseNode } as const;

const EDGE_KIND_COLORS = {
  always: "var(--graph-edge-always)",
  payload_bool: "var(--graph-edge-payload-bool)",
  progress_decision: "var(--graph-edge-progress-decision)",
} as const;

interface PhaseGraphProps {
  entry: OperatingModeCatalogEntry;
  selectedPhaseId?: string | null;
  onSelectPhase?: (phase: string) => void;
  /** Composer mode adds startable/reason/runCount affordances and disables click on non-startable phases. */
  mode?: "details" | "composer";
  /** Round counts per phase. Renders a small badge on each node when provided. */
  rounds?: OperatingModeRound[];
  /** Per-phase startable/reason/next data, sourced from the workspace endpoint in composer mode. */
  phaseStates?: PhaseStateMap;
}

interface BuildArgs {
  phases: OperatingModeCatalogPhase[];
  transitions: OperatingModePhaseTransition[];
  selectedPhaseId?: string | null;
  mode?: "details" | "composer";
  roundsByPhase?: Map<string, number>;
  phaseStates?: PhaseStateMap;
}

function buildGraph({ phases, transitions, selectedPhaseId, mode, roundsByPhase, phaseStates }: BuildArgs): { nodes: PhaseNodeType[]; edges: Edge[] } {
  const nodes: PhaseNodeType[] = phases.map((phase) => ({
    id: phase.phase,
    type: "phase",
    position: { x: 0, y: 0 },
    data: {
      phase: phase.phase,
      title: phase.label || phase.title || phase.phase,
      isStart: !!phase.isStart,
      isTerminal: !!phase.isTerminal,
      writesRepo: phase.writesRepo,
      selected: selectedPhaseId === phase.phase,
      mode,
      startable: phaseStates?.[phase.phase]?.startable,
      reason: phaseStates?.[phase.phase]?.reason,
      isNext: phaseStates?.[phase.phase]?.isNext,
      runCount: roundsByPhase?.get(phase.phase),
    },
  }));

  const edges: Edge[] = transitions.map((edge, idx) => {
    const color = EDGE_KIND_COLORS[edge.conditionKind] ?? EDGE_KIND_COLORS.always;
    return {
      id: `${edge.from}->${edge.to}-${idx}`,
      source: edge.from,
      target: edge.to,
      label: edge.label,
      labelBgPadding: [4, 2],
      labelBgBorderRadius: 4,
      labelBgStyle: { fill: "var(--graph-edge-label-bg)" },
      labelStyle: { fill: "var(--graph-edge-label-color)", fontSize: 11 },
      style: { stroke: color, strokeWidth: 1.5 },
      markerEnd: { type: MarkerType.ArrowClosed, color },
    };
  });

  if (nodes.length === 0) return { nodes, edges };

  const g = new dagre.graphlib.Graph();
  g.setDefaultEdgeLabel(() => ({}));
  g.setGraph({ rankdir: "TB", ranksep: 70, nodesep: 40 });
  for (const node of nodes) {
    g.setNode(node.id, { width: NODE_WIDTH, height: NODE_HEIGHT });
  }
  for (const edge of edges) {
    g.setEdge(edge.source, edge.target);
  }
  dagre.layout(g);

  const positioned = nodes.map((node) => {
    const pos = g.node(node.id) as { x: number; y: number } | undefined;
    if (!pos) return node;
    return {
      ...node,
      position: { x: pos.x - NODE_WIDTH / 2, y: pos.y - NODE_HEIGHT / 2 },
    };
  });

  return { nodes: positioned, edges };
}

const LEGEND_ITEMS: Array<{ dot: string; label: string }> = [
  { dot: "bg-emerald-400", label: "start" },
  { dot: "bg-violet-400", label: "terminal" },
  { dot: "bg-cyan-400", label: "selected" },
];

const EDGE_LEGEND: Array<{ color: string; label: string }> = [
  { color: "var(--graph-edge-always)", label: "always" },
  { color: "var(--graph-edge-payload-bool)", label: "payload bool" },
  { color: "var(--graph-edge-progress-decision)", label: "progress decision" },
];

export function PhaseGraph({ entry, selectedPhaseId, onSelectPhase, mode, rounds, phaseStates }: PhaseGraphProps) {
  const transitions = useMemo(() => entry.phaseGraph?.transitions ?? [], [entry.phaseGraph]);
  const roundsByPhase = useMemo(() => {
    const map = new Map<string, number>();
    for (const round of rounds ?? []) {
      map.set(round.phase, (map.get(round.phase) ?? 0) + 1);
    }
    return map;
  }, [rounds]);
  const { nodes: layoutedNodes, edges: builtEdges } = useMemo(
    () => buildGraph({ phases: entry.phases, transitions, selectedPhaseId, mode, roundsByPhase, phaseStates }),
    [entry.phases, transitions, selectedPhaseId, mode, roundsByPhase, phaseStates],
  );
  const [nodes, setNodes, onNodesChange] = useNodesState<PhaseNodeType>(layoutedNodes);
  const [edges, setEdges, onEdgesChange] = useEdgesState<Edge>(builtEdges);
  const [glossaryOpen, setGlossaryOpen] = useState(false);

  useEffect(() => {
    setNodes(layoutedNodes);
  }, [layoutedNodes, setNodes]);
  useEffect(() => {
    setEdges(builtEdges);
  }, [builtEdges, setEdges]);

  return (
    <div className="space-y-2" data-testid="phase-graph">
      <button
        type="button"
        onClick={() => setGlossaryOpen(true)}
        aria-label="Open phase-graph glossary"
        data-testid={selectors.initiativeDetails.phaseGraphLegend}
        className="flex w-full flex-wrap items-center gap-x-4 gap-y-1 rounded px-2 py-1 text-left text-[11px] text-slate-400 transition-colors hover:bg-slate-800/40 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cyan-500/50"
      >
        <span className="font-medium text-slate-300">Legend:</span>
        <span className="flex items-center gap-3">
          {LEGEND_ITEMS.map((item) => (
            <span key={item.label} className="flex items-center gap-1.5">
              <span className={`inline-block h-2 w-2 rounded-full ${item.dot}`} />
              {item.label}
            </span>
          ))}
        </span>
        <span className="text-slate-600">·</span>
        <span className="flex items-center gap-3">
          {EDGE_LEGEND.map((item) => (
            <span key={item.label} className="flex items-center gap-1.5">
              <span
                className="inline-block h-0.5 w-5 rounded-sm"
                style={{ background: item.color }}
              />
              {item.label}
            </span>
          ))}
        </span>
        <span className="ml-auto text-[11px] text-slate-500">click for glossary</span>
      </button>
      <PhaseGraphGlossaryDialog isOpen={glossaryOpen} onClose={() => setGlossaryOpen(false)} />
      <div className="h-[300px] min-h-[240px] overflow-hidden rounded-lg border border-slate-800 bg-slate-950 sm:h-[440px] md:h-[480px]">
        <ReactFlowProvider>
          <ReactFlow
            nodes={nodes}
            edges={edges}
            nodeTypes={NODE_TYPES}
            onNodesChange={onNodesChange}
            onEdgesChange={onEdgesChange}
            onNodeClick={(_, node) => {
              if (mode === "composer" && (node.data as PhaseNodeData | undefined)?.startable === false) return;
              onSelectPhase?.(node.id);
            }}
            nodesDraggable={false}
            nodesConnectable={false}
            elementsSelectable
            fitView
            fitViewOptions={{ padding: 0.2 }}
            proOptions={{ hideAttribution: true }}
          >
            <Background gap={24} size={1} color="var(--graph-grid)" />
            <Controls showInteractive={false} />
            <MiniMap
              pannable
              zoomable
              maskColor="var(--graph-mini-mask)"
              nodeColor={(node) => {
                const data = node.data as PhaseNodeData | undefined;
                if (data?.isStart) return "var(--graph-mini-node-start)";
                if (data?.isTerminal) return "var(--graph-mini-node-terminal)";
                return "var(--graph-mini-node-default)";
              }}
              className="!bg-slate-900 !border !border-slate-700"
            />
          </ReactFlow>
        </ReactFlowProvider>
      </div>
    </div>
  );
}
