import { useEffect, useMemo, useState } from "react";
import ELK, { type ElkExtendedEdge, type ElkNode } from "elkjs/lib/elk.bundled.js";
import {
  Background,
  Controls,
  MarkerType,
  MiniMap,
  ReactFlow,
  ReactFlowProvider,
  type Edge,
  type Node,
  type NodeProps,
} from "@xyflow/react";
import { Network, Workflow } from "lucide-react";
import { NodeKind, type TechEdge, type TechNode, type TechTreeGraph } from "@vrooli/proto-types/tech-tree-designer/v1/graph/graph_pb";

import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";

type ScenarioNodeData = {
  label: string;
  scenario: string;
  kind: NodeKind;
  stability: string;
  transportWorld: string;
  sector: string;
  tier: string;
};

const elk = new ELK();
const NODE_WIDTH = 230;
const NODE_HEIGHT = 104;

const stabilityTone = (stability: string) => {
  if (stability.includes("stable")) return "border-emerald-400/70 bg-emerald-950/35";
  if (stability.includes("beta")) return "border-sky-400/70 bg-sky-950/35";
  if (stability.includes("experimental")) return "border-amber-400/75 bg-amber-950/35";
  return "border-app-border bg-app-surface";
};

function ScenarioNode({ data, selected }: NodeProps<Node<ScenarioNodeData>>) {
  const { t } = useTranslation();
  const isPlanned = data.kind === NodeKind.PLANNED;
  return (
    <div
      className={[
        "h-[104px] w-[230px] rounded-lg border p-3 shadow-lg shadow-black/15 transition",
        stabilityTone(data.stability),
        selected ? "ring-2 ring-app-primary" : "",
      ].join(" ")}
    >
      <div className="flex items-start gap-2">
        <span className="mt-0.5 rounded-md bg-black/20 p-1 text-app-foreground">
          {isPlanned ? <Workflow aria-hidden className="h-4 w-4" /> : <Network aria-hidden className="h-4 w-4" />}
        </span>
        <div className="min-w-0">
          <p className="truncate text-sm font-semibold text-app-foreground">{data.label}</p>
          <p className="truncate text-xs text-app-muted-foreground">{data.scenario}</p>
        </div>
      </div>
      <div className="mt-3 flex flex-wrap gap-1 text-[11px]">
        <span className="rounded-full bg-black/20 px-2 py-1 uppercase">
          {isPlanned ? t(strings.graph.node.planned) : t(strings.graph.node.live)}
        </span>
        <span className="rounded-full bg-black/20 px-2 py-1">{data.transportWorld || t(strings.graph.node.none)}</span>
        <span className="rounded-full bg-black/20 px-2 py-1">{data.stability || t(strings.graph.node.unknown)}</span>
      </div>
      <p className="mt-2 truncate text-[11px] text-app-muted-foreground">
        {[data.sector || t(strings.graph.node.unassigned), data.tier || t(strings.graph.node.untiered)].join(" / ")}
      </p>
    </div>
  );
}

const nodeTypes = { scenario: ScenarioNode };

const edgeLabel = (edge: TechEdge) => {
  const evidence = edge.evidence[0];
  return evidence?.importPath || evidence?.path || edge.stability.join(", ") || "dependency";
};

const toFlowEdge = (edge: TechEdge): Edge => ({
  id: `${edge.fromScenario}->${edge.toScenario}`,
  source: edge.fromScenario,
  target: edge.toScenario,
  label: edgeLabel(edge),
  markerEnd: { type: MarkerType.ArrowClosed },
  className: edge.stability.includes("experimental") ? "ttd-flow-edge-planned" : "ttd-flow-edge",
  style: { strokeWidth: 1.5 },
});

const toFlowNode = (node: TechNode): Node<ScenarioNodeData> => ({
  id: node.scenario,
  type: "scenario",
  position: { x: 0, y: 0 },
  data: {
    label: node.displayName || node.scenario,
    scenario: node.scenario,
    kind: node.kind,
    stability: node.stability.join(", "),
    transportWorld: node.transportWorld,
    sector: node.sector,
    tier: node.tier,
  },
});

async function layoutGraph(nodes: Node<ScenarioNodeData>[], edges: Edge[]): Promise<Node<ScenarioNodeData>[]> {
  const graph: ElkNode = {
    id: "root",
    layoutOptions: {
      "elk.algorithm": "layered",
      "elk.direction": "RIGHT",
      "elk.spacing.nodeNode": "44",
      "elk.layered.spacing.nodeNodeBetweenLayers": "76",
      "elk.edgeRouting": "ORTHOGONAL",
    },
    children: nodes.map((node) => ({
      id: node.id,
      width: NODE_WIDTH,
      height: NODE_HEIGHT,
    })),
    edges: edges.map<ElkExtendedEdge>((edge) => ({
      id: edge.id,
      sources: [edge.source],
      targets: [edge.target],
    })),
  };
  const result = await elk.layout(graph);
  const positions = new Map((result.children ?? []).map((child) => [child.id, child]));
  return nodes.map((node) => {
    const position = positions.get(node.id);
    return {
      ...node,
      position: { x: position?.x ?? 0, y: position?.y ?? 0 },
    };
  });
}

export function GraphCanvas({ graph }: { graph: TechTreeGraph }) {
  const { t } = useTranslation();
  const [nodes, setNodes] = useState<Node<ScenarioNodeData>[]>([]);

  const edges = useMemo(() => graph.edges.map(toFlowEdge), [graph.edges]);
  const baseNodes = useMemo(() => graph.nodes.map(toFlowNode), [graph.nodes]);

  useEffect(() => {
    let active = true;
    void layoutGraph(baseNodes, edges).then((nextNodes) => {
      if (active) setNodes(nextNodes);
    });
    return () => {
      active = false;
    };
  }, [baseNodes, edges]);

  if (graph.nodes.length === 0) {
    return (
      <div className="flex min-h-[360px] items-center justify-center rounded-lg border border-dashed border-app-border bg-app-surface p-8 text-center text-app-muted-foreground">
        {t(strings.graph.states.empty)}
      </div>
    );
  }

  return (
    <div data-testid={selectors.graph.canvas} className="h-[640px] overflow-hidden rounded-lg border border-app-border bg-app-surface">
      <ReactFlowProvider>
        <ReactFlow
          nodes={nodes}
          edges={edges}
          nodeTypes={nodeTypes}
          fitView
          fitViewOptions={{ padding: 0.24 }}
          minZoom={0.2}
          maxZoom={1.8}
          nodesDraggable
          nodesConnectable={false}
        >
          <Background gap={24} color="rgba(148, 163, 184, 0.18)" />
          <Controls position="bottom-left" />
          <MiniMap
            pannable
            zoomable
            nodeColor={(node) =>
              node.data.kind === NodeKind.PLANNED ? "#f59e0b" : "#38bdf8"
            }
          />
        </ReactFlow>
      </ReactFlowProvider>
    </div>
  );
}
