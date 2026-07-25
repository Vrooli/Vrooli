import { describe, expect, it } from 'vitest'
import type { Edge, Node } from '@xyflow/react'
import { layoutFlowDagre } from './FlowShell'

describe('layoutFlowDagre', () => {
  it('lays directed nodes in a stable flow direction without changing edges', () => {
    const nodes: Node[] = [
      { id: 'producer', position: { x: 0, y: 0 }, data: {} },
      { id: 'consumer', position: { x: 0, y: 0 }, data: {} },
    ]
    const edges: Edge[] = [{ id: 'edge', source: 'producer', target: 'consumer' }]
    const result = layoutFlowDagre(nodes, edges, { direction: 'LR', nodeWidth: 100, nodeHeight: 40 })
    expect(result.edges).toEqual(edges)
    expect(result.nodes.find((node) => node.id === 'producer')!.position.x)
      .toBeLessThan(result.nodes.find((node) => node.id === 'consumer')!.position.x)
  })
})
