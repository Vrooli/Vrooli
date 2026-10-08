import { useMemo } from "react";
import {
  Background,
  Position,
  ReactFlow,
  type Edge,
  type Node,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import dagre from "dagre";

import type { FlowState, FlowTransition } from "../../api/inventory";
import { useTranslation } from "../../i18n";

const NODE_WIDTH = 160;
const NODE_HEIGHT = 48;

export interface StateGraphProps {
  states: FlowState[];
  events: { id: string }[];
  transitions: FlowTransition[];
  initialState: string;
  /** Optional state to render as the active step (drives the trace player). */
  activeState?: string;
}

interface LayoutResult {
  nodes: Node[];
  edges: Edge[];
}

function layoutGraph(
  states: FlowState[],
  transitions: FlowTransition[],
  initialState: string,
  activeState: string | undefined,
): LayoutResult {
  const g = new dagre.graphlib.Graph();
  g.setGraph({ rankdir: "LR", nodesep: 32, ranksep: 64 });
  g.setDefaultEdgeLabel(() => ({}));
  for (const state of states) {
    g.setNode(state.id, { width: NODE_WIDTH, height: NODE_HEIGHT });
  }

  // De-dup edges: many (from,event) pairs converge on the same (from→to).
  // The graph shows topology, so we collapse parallel edges into one with
  // a joined event label. wantError edges are excluded to keep the happy-
  // path graph readable — the transition table elsewhere shows the full
  // matrix.
  const pairs = new Map<string, { from: string; to: string; events: string[] }>();
  for (const t of transitions) {
    if (t.wantError) continue;
    const key = `${t.from}->${t.to}`;
    const existing = pairs.get(key);
    if (existing) existing.events.push(t.event);
    else pairs.set(key, { from: t.from, to: t.to, events: [t.event] });
  }
  for (const { from, to } of pairs.values()) {
    g.setEdge(from, to);
  }
  dagre.layout(g);

  const nodes: Node[] = states.map((state) => {
    const node = g.node(state.id) as { x: number; y: number } | undefined;
    const pos = node ?? { x: 0, y: 0 };
    const isInitial = state.id === initialState;
    const isActive = state.id === activeState;
    const isTerminal = !!state.terminal;
    return {
      id: state.id,
      data: { label: state.id },
      position: { x: pos.x - NODE_WIDTH / 2, y: pos.y - NODE_HEIGHT / 2 },
      style: {
        width: NODE_WIDTH,
        height: NODE_HEIGHT,
        borderRadius: 8,
        border: isActive
          ? "2px solid var(--color-primary)"
          : isInitial
            ? "2px solid var(--color-success)"
            : "1px solid var(--color-border)",
        background: isActive
          ? "color-mix(in srgb, var(--color-primary) 18%, transparent)"
          : isTerminal
            ? "color-mix(in srgb, var(--color-warning) 18%, transparent)"
            : "var(--color-surface)",
        color: "var(--color-foreground)",
        fontSize: 13,
        fontFamily: "ui-monospace, SFMono-Regular, Menlo, monospace",
      },
      sourcePosition: Position.Right,
      targetPosition: Position.Left,
    };
  });

  const edges: Edge[] = Array.from(pairs.values()).map((pair, idx) => ({
    id: `e-${pair.from}-${pair.to}-${idx}`,
    source: pair.from,
    target: pair.to,
    label: pair.events.join(", "),
    labelStyle: { fill: "var(--color-muted-foreground)", fontSize: 11 },
    labelBgStyle: { fill: "var(--color-surface)" },
    style: { stroke: "var(--color-muted-foreground)" },
  }));

  return { nodes, edges };
}

/**
 * StateGraph renders a flow's transition matrix as a directed graph.
 *
 * Layout is deterministic (dagre, left-to-right). The `activeState` prop
 * lets the trace player highlight the current step; without it, the
 * initial state is the only highlighted node.
 */
export function StateGraph({
  states,
  events,
  transitions,
  initialState,
  activeState,
}: StateGraphProps) {
  const { t } = useTranslation();
  const { nodes, edges } = useMemo(
    () => layoutGraph(states, transitions, initialState, activeState),
    [states, transitions, initialState, activeState],
  );

  if (states.length === 0) {
    return (
      <p
        data-testid="state-graph-empty"
        className="rounded-panel border border-app-border bg-app-surface p-4 text-app-foreground"
      >
        {t("stateGraph.empty", { defaultValue: "No states declared." })}
      </p>
    );
  }

  return (
    <div
      data-testid="state-graph"
      role="img"
      aria-label={t("stateGraph.aria", {
        defaultValue: "State machine graph",
      })}
      className="rounded-panel border border-app-border bg-app-surface"
      style={{ height: 420 }}
    >
      <ReactFlow
        nodes={nodes}
        edges={edges}
        fitView
        nodesDraggable={false}
        nodesConnectable={false}
        edgesFocusable={false}
        proOptions={{ hideAttribution: true }}
      >
        <Background gap={16} />
      </ReactFlow>
      <p data-testid="state-graph-summary" className="px-3 py-2 text-xs text-app-muted-foreground">
        {t("stateGraph.summary", {
          defaultValue: `${states.length} states · ${events.length} events · ${edges.length} edges`,
        })}
      </p>
    </div>
  );
}
