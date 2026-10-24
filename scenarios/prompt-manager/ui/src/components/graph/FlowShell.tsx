import type { ReactNode } from 'react'
import dagre from '@dagrejs/dagre'
import {
  Background,
  Controls,
  MiniMap,
  ReactFlow,
  type Edge,
  type MiniMapProps,
  type Node,
  type ReactFlowProps,
} from '@xyflow/react'

import '@xyflow/react/dist/style.css'

/**
 * Shared canvas treatment for the contract and hierarchy Flow projections.
 * Individual projections own their nodes, edges, and layout; this shell owns
 * the common React Flow frame, minimap, controls, and selection plumbing.
 */
export interface FlowShellProps<NodeType extends Node = Node, EdgeType extends Edge = Edge> extends ReactFlowProps<NodeType, EdgeType> {
  children?: ReactNode
  miniMapNodeColor?: MiniMapProps<NodeType>['nodeColor']
}

export function FlowShell<NodeType extends Node = Node, EdgeType extends Edge = Edge>({ children, miniMapNodeColor, className = 'bg-background', ...flowProps }: FlowShellProps<NodeType, EdgeType>) {
  return (
    <ReactFlow<NodeType, EdgeType> {...flowProps} className={className} proOptions={{ hideAttribution: true }}>
      <Background color="hsl(var(--border))" gap={20} />
      <Controls showInteractive={false} />
      <MiniMap<NodeType> nodeColor={miniMapNodeColor} maskColor="rgba(0, 0, 0, 0.5)" />
      {children}
    </ReactFlow>
  )
}

export interface DagreLayoutOptions {
  direction: 'TB' | 'LR'
  nodeWidth: number
  nodeHeight: number
  nodeSep?: number
  rankSep?: number
}

// layoutFlowDagre is shared by the team Topics, team Hierarchy, and swarm
// Flow projections. Projection-specific grouping remains at the caller.
export function layoutFlowDagre<NodeType extends Node, EdgeType extends Edge>(
  nodes: NodeType[],
  edges: EdgeType[],
  { direction, nodeWidth, nodeHeight, nodeSep = 30, rankSep = 80 }: DagreLayoutOptions,
): { nodes: NodeType[]; edges: EdgeType[] } {
  const graph = new dagre.graphlib.Graph()
  graph.setDefaultEdgeLabel(() => ({}))
  graph.setGraph({ rankdir: direction, nodesep: nodeSep, ranksep: rankSep })
  nodes.forEach((node) => graph.setNode(node.id, { width: nodeWidth, height: nodeHeight }))
  edges.forEach((edge) => graph.setEdge(edge.source, edge.target))
  dagre.layout(graph)
  return {
    nodes: nodes.map((node) => {
      const position = graph.node(node.id)
      return { ...node, position: { x: position.x - nodeWidth / 2, y: position.y - nodeHeight / 2 } }
    }),
    edges,
  }
}
