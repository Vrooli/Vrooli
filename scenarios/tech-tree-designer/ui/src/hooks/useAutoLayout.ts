import { useMemo } from 'react'
import dagre from 'dagre'
import type { Node } from 'react-flow-renderer'
import type { DesignerNode, DesignerEdge } from '../types/graph'

interface AutoLayoutOptions {
  enabled: boolean
  direction?: 'LR' | 'TB' | 'RL' | 'BT' // Left-Right, Top-Bottom, Right-Left, Bottom-Top
  nodeWidth?: number
  nodeHeight?: number
  rankSep?: number // Separation between ranks
  nodeSep?: number // Separation between nodes in same rank
}

interface UseAutoLayoutResult {
  layoutedNodes: DesignerNode[]
}

/**
 * Custom hook for automatic graph layout using Dagre hierarchical algorithm.
 *
 * Features:
 * - Hierarchical layout (Dagre algorithm)
 * - Configurable direction (LR, TB, RL, BT)
 * - Customizable spacing and node dimensions
 * - Handles disconnected components
 * - Only recalculates when nodes/edges change
 *
 * @param nodes - Graph nodes to layout
 * @param edges - Graph edges defining connections
 * @param options - Layout configuration options
 * @returns Nodes with computed positions
 */
export const useAutoLayout = (
  nodes: DesignerNode[],
  edges: DesignerEdge[],
  options: AutoLayoutOptions
): UseAutoLayoutResult => {
  const layoutedNodes = useMemo(() => {
    // If auto-layout is disabled, return nodes as-is
    if (!options.enabled || nodes.length === 0) {
      return nodes
    }

    const {
      direction = 'LR',
      nodeWidth = 220,
      nodeHeight = 120,
      rankSep = 150,
      nodeSep = 80
    } = options

    // Create new directed graph
    const dagreGraph = new dagre.graphlib.Graph()
    dagreGraph.setDefaultEdgeLabel(() => ({}))

    // Configure graph layout
    dagreGraph.setGraph({
      rankdir: direction,
      ranksep: rankSep,
      nodesep: nodeSep,
      edgesep: 30,
      marginx: 20,
      marginy: 20
    })

    // Add nodes to dagre graph
    nodes.forEach((node) => {
      // Use actual node dimensions if available, otherwise use defaults
      const width = nodeWidth
      const height = nodeHeight

      dagreGraph.setNode(node.id, { width, height })
    })

    // Add edges to dagre graph
    edges.forEach((edge) => {
      if (edge.source && edge.target) {
        dagreGraph.setEdge(edge.source, edge.target)
      }
    })

    // Run Dagre layout algorithm
    dagre.layout(dagreGraph)

    // Apply calculated positions back to nodes
    const positionedNodes = nodes.map((node) => {
      const dagreNode = dagreGraph.node(node.id)

      if (!dagreNode) {
        // Node not in graph (shouldn't happen, but fallback to original position)
        console.warn(`[useAutoLayout] Node ${node.id} not found in dagre graph`)
        return node
      }

      // Dagre positions are center-based, React Flow positions are top-left
      // So we need to offset by half the width/height
      const x = dagreNode.x - nodeWidth / 2
      const y = dagreNode.y - nodeHeight / 2

      return {
        ...node,
        position: { x, y }
      } as DesignerNode
    })

    console.log(`[useAutoLayout] Layouted ${positionedNodes.length} nodes with direction ${direction}`)
    return positionedNodes
  }, [nodes, edges, options.enabled, options.direction, options.nodeWidth, options.nodeHeight, options.rankSep, options.nodeSep])

  return { layoutedNodes }
}

/**
 * Hook for managing auto-layout state in localStorage.
 * Remembers user's layout preference across sessions.
 */
export const useAutoLayoutPreference = (storageKey: string = 'tech-tree-auto-layout') => {
  const getPreference = (): boolean => {
    try {
      const stored = localStorage.getItem(storageKey)
      return stored === null ? true : stored === 'true' // Default to enabled
    } catch {
      return true // Default to enabled if localStorage fails
    }
  }

  const setPreference = (enabled: boolean): void => {
    try {
      localStorage.setItem(storageKey, String(enabled))
    } catch (error) {
      console.warn('[useAutoLayoutPreference] Failed to save preference:', error)
    }
  }

  return { getPreference, setPreference }
}
