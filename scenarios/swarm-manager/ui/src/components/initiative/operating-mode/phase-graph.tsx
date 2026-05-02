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

import { useEffect, useMemo } from "react";
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
import type {
  OperatingModeCatalogEntry,
  OperatingModeCatalogPhase,
  OperatingModePhaseTransition,
} from "../../../types/operating-mode";
import { PhaseNode, type PhaseNodeData, type PhaseNodeType } from "./phase-node";

const NODE_WIDTH = 180;
const NODE_HEIGHT = 76;

const NODE_TYPES = { phase: PhaseNode } as const;

const EDGE_KIND_COLORS = {
  always: "#94a3b8",
  payload_bool: "#fbbf24",
  progress_decision: "#22d3ee",
} as const;

interface PhaseGraphProps {
  entry: OperatingModeCatalogEntry;
  selectedPhaseId?: string | null;
  onSelectPhase?: (phase: string) => void;
}

interface BuildArgs {
  phases: OperatingModeCatalogPhase[];
  transitions: OperatingModePhaseTransition[];
  selectedPhaseId?: string | null;
}

function buildGraph({ phases, transitions, selectedPhaseId }: BuildArgs): { nodes: PhaseNodeType[]; edges: Edge[] } {
  const nodes: PhaseNodeType[] = phases.map((phase) => ({
    id: phase.phase,
    type: "phase",
    position: { x: 0, y: 0 },
    data: {
      phase: phase.phase,
      title: phase.title || phase.phase,
      isStart: !!phase.isStart,
      isTerminal: !!phase.isTerminal,
      writesRepo: phase.writesRepo,
      selected: selectedPhaseId === phase.phase,
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
      labelBgStyle: { fill: "#0f172a", fillOpacity: 0.85 },
      labelStyle: { fill: "#e2e8f0", fontSize: 11 },
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
  { color: "#94a3b8", label: "always" },
  { color: "#fbbf24", label: "payload bool" },
  { color: "#22d3ee", label: "progress decision" },
];

export function PhaseGraph({ entry, selectedPhaseId, onSelectPhase }: PhaseGraphProps) {
  const transitions = useMemo(() => entry.phaseGraph?.transitions ?? [], [entry.phaseGraph]);
  const { nodes: layoutedNodes, edges: builtEdges } = useMemo(
    () => buildGraph({ phases: entry.phases, transitions, selectedPhaseId }),
    [entry.phases, transitions, selectedPhaseId],
  );
  const [nodes, setNodes, onNodesChange] = useNodesState<PhaseNodeType>(layoutedNodes);
  const [edges, setEdges, onEdgesChange] = useEdgesState<Edge>(builtEdges);

  useEffect(() => {
    setNodes(layoutedNodes);
  }, [layoutedNodes, setNodes]);
  useEffect(() => {
    setEdges(builtEdges);
  }, [builtEdges, setEdges]);

  return (
    <div className="space-y-2" data-testid="phase-graph">
      <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-[11px] text-slate-400">
        <span className="font-medium text-slate-300">Legend:</span>
        <div className="flex items-center gap-3">
          {LEGEND_ITEMS.map((item) => (
            <span key={item.label} className="flex items-center gap-1.5">
              <span className={`inline-block h-2 w-2 rounded-full ${item.dot}`} />
              {item.label}
            </span>
          ))}
        </div>
        <span className="text-slate-600">·</span>
        <div className="flex items-center gap-3">
          {EDGE_LEGEND.map((item) => (
            <span key={item.label} className="flex items-center gap-1.5">
              <span
                className="inline-block h-0.5 w-5 rounded-sm"
                style={{ background: item.color }}
              />
              {item.label}
            </span>
          ))}
        </div>
      </div>
      <div className="h-[420px] overflow-hidden rounded-lg border border-slate-800 bg-slate-950 sm:h-[480px]">
        <ReactFlowProvider>
          <ReactFlow
            nodes={nodes}
            edges={edges}
            nodeTypes={NODE_TYPES}
            onNodesChange={onNodesChange}
            onEdgesChange={onEdgesChange}
            onNodeClick={(_, node) => onSelectPhase?.(node.id)}
            nodesDraggable={false}
            nodesConnectable={false}
            elementsSelectable
            fitView
            fitViewOptions={{ padding: 0.2 }}
            proOptions={{ hideAttribution: true }}
          >
            <Background gap={24} size={1} color="#1e293b" />
            <Controls showInteractive={false} className="!bg-slate-900 !border-slate-700" />
            <MiniMap
              pannable
              zoomable
              maskColor="rgb(15, 23, 42, 0.8)"
              nodeColor={(node) => {
                const data = node.data as PhaseNodeData | undefined;
                if (data?.isStart) return "#34d399";
                if (data?.isTerminal) return "#a78bfa";
                return "#475569";
              }}
              className="!bg-slate-900 !border !border-slate-700"
            />
          </ReactFlow>
        </ReactFlowProvider>
      </div>
    </div>
  );
}
