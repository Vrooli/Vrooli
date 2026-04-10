/**
 * Tests for collectNeighborhood — type-constrained BFS.
 *
 * Rule: each hop must move to a DIFFERENT entity type than any type already
 * visited on the current path. With 4 types (team/agent/skill/cli), this
 * naturally caps at 3 hops and prevents lateral spread within the same type.
 */

import { describe, it, expect } from 'vitest'
import { collectNeighborhood } from './graphNeighborhood'
import type { GraphEdge, GraphNode, NodeType, EdgeKind } from '@/lib/schemas'

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

function makeNode(id: string, type: NodeType = 'skill'): GraphNode {
  return { id, type, label: id, description: '', status: '', tags: [] }
}

function makeEdge(from: string, to: string, kind: EdgeKind = 'membership'): GraphEdge {
  return { from, to, kind, category: '', sourceFile: '', lineNumber: 0 }
}

function nodeMap(nodes: GraphNode[]): Map<string, GraphNode> {
  return new Map(nodes.map((n) => [n.id, n]))
}

function edgeMap(edges: GraphEdge[]): Map<string, GraphEdge[]> {
  const map = new Map<string, GraphEdge[]>()
  for (const edge of edges) {
    const fromList = map.get(edge.from) ?? []
    fromList.push(edge)
    map.set(edge.from, fromList)
    if (edge.from !== edge.to) {
      const toList = map.get(edge.to) ?? []
      toList.push(edge)
      map.set(edge.to, toList)
    }
  }
  return map
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe('collectNeighborhood', () => {
  describe('basic cases', () => {
    it('should return only the start node when it has no edges', () => {
      const nodes = [makeNode('A', 'skill')]
      const result = collectNeighborhood('A', edgeMap([]), nodeMap(nodes))
      expect(result).toEqual(new Set(['A']))
    })

    it('should return empty set when start node does not exist in nodeMap', () => {
      const result = collectNeighborhood('missing', edgeMap([]), nodeMap([]))
      expect(result).toEqual(new Set())
    })

    it('should find direct neighbor of a different type', () => {
      const nodes = [makeNode('s1', 'skill'), makeNode('a1', 'agent')]
      const edges = [makeEdge('a1', 's1')]
      const result = collectNeighborhood('s1', edgeMap(edges), nodeMap(nodes))
      expect(result).toEqual(new Set(['s1', 'a1']))
    })

    it('should find neighbor via incoming edge (bidirectional)', () => {
      const nodes = [makeNode('a1', 'agent'), makeNode('t1', 'team')]
      const edges = [makeEdge('t1', 'a1')]
      const result = collectNeighborhood('a1', edgeMap(edges), nodeMap(nodes))
      expect(result).toEqual(new Set(['a1', 't1']))
    })
  })

  describe('type constraint — no same-type lateral traversal', () => {
    it('should NOT include a second skill when starting from a skill', () => {
      // skill-1 → agent-1 → skill-2  (skill-2 should be excluded: skill already visited)
      const nodes = [
        makeNode('skill-1', 'skill'),
        makeNode('agent-1', 'agent'),
        makeNode('skill-2', 'skill'),
      ]
      const edges = [
        makeEdge('agent-1', 'skill-1'),
        makeEdge('agent-1', 'skill-2'),
      ]
      const result = collectNeighborhood('skill-1', edgeMap(edges), nodeMap(nodes))
      expect(result).toEqual(new Set(['skill-1', 'agent-1']))
      expect(result.has('skill-2')).toBe(false)
    })

    it('should NOT include a second agent when starting from an agent', () => {
      // agent-1 → team-1 → agent-2  (agent-2 excluded: agent already visited)
      const nodes = [
        makeNode('agent-1', 'agent'),
        makeNode('team-1', 'team'),
        makeNode('agent-2', 'agent'),
      ]
      const edges = [
        makeEdge('team-1', 'agent-1'),
        makeEdge('team-1', 'agent-2'),
      ]
      const result = collectNeighborhood('agent-1', edgeMap(edges), nodeMap(nodes))
      expect(result).toEqual(new Set(['agent-1', 'team-1']))
      expect(result.has('agent-2')).toBe(false)
    })

    it('should NOT include a second team when starting from a team', () => {
      // team-1 → agent → team-2  (weird edge, but team-2 excluded: team already visited)
      const nodes = [
        makeNode('team-1', 'team'),
        makeNode('agent-1', 'agent'),
        makeNode('team-2', 'team'),
      ]
      const edges = [
        makeEdge('team-1', 'agent-1'),
        makeEdge('team-2', 'agent-1'),
      ]
      const result = collectNeighborhood('team-1', edgeMap(edges), nodeMap(nodes))
      expect(result).toEqual(new Set(['team-1', 'agent-1']))
      expect(result.has('team-2')).toBe(false)
    })
  })

  describe('full hierarchy traversal', () => {
    //       team-1
    //         |
    //       agent-1
    //         |
    //       skill-1
    //         |
    //       cli:tool
    const nodes = [
      makeNode('team-1', 'team'),
      makeNode('agent-1', 'agent'),
      makeNode('skill-1', 'skill'),
      makeNode('cli:tool', 'cli'),
    ]
    const edges = [
      makeEdge('team-1', 'agent-1'),
      makeEdge('agent-1', 'skill-1'),
      makeEdge('skill-1', 'cli:tool'),
    ]

    it('should find full hierarchy from team (3 hops down)', () => {
      const result = collectNeighborhood('team-1', edgeMap(edges), nodeMap(nodes))
      expect(result).toEqual(new Set(['team-1', 'agent-1', 'skill-1', 'cli:tool']))
    })

    it('should find full hierarchy from cli (3 hops up)', () => {
      const result = collectNeighborhood('cli:tool', edgeMap(edges), nodeMap(nodes))
      expect(result).toEqual(new Set(['team-1', 'agent-1', 'skill-1', 'cli:tool']))
    })

    it('should find full hierarchy from middle (agent)', () => {
      const result = collectNeighborhood('agent-1', edgeMap(edges), nodeMap(nodes))
      expect(result).toEqual(new Set(['team-1', 'agent-1', 'skill-1', 'cli:tool']))
    })

    it('should find full hierarchy from middle (skill)', () => {
      const result = collectNeighborhood('skill-1', edgeMap(edges), nodeMap(nodes))
      expect(result).toEqual(new Set(['team-1', 'agent-1', 'skill-1', 'cli:tool']))
    })
  })

  describe('branching hierarchy — type constraint limits fan-out', () => {
    //       team-1
    //      /      \
    //   agent-1  agent-2
    //     |         |
    //   skill-1   skill-2
    //     |
    //   cli:tool
    const nodes = [
      makeNode('team-1', 'team'),
      makeNode('agent-1', 'agent'),
      makeNode('agent-2', 'agent'),
      makeNode('skill-1', 'skill'),
      makeNode('skill-2', 'skill'),
      makeNode('cli:tool', 'cli'),
    ]
    const edges = [
      makeEdge('team-1', 'agent-1'),
      makeEdge('team-1', 'agent-2'),
      makeEdge('agent-1', 'skill-1'),
      makeEdge('agent-2', 'skill-2'),
      makeEdge('skill-1', 'cli:tool'),
    ]

    it('should NOT cross to peer agents or their subtrees when clicking a skill', () => {
      // From skill-1: up to agent-1, up to team-1 (type agent+team visited),
      // team-1→agent-2 blocked (agent type already visited)
      const result = collectNeighborhood('skill-1', edgeMap(edges), nodeMap(nodes))
      expect(result).toEqual(new Set(['skill-1', 'agent-1', 'team-1', 'cli:tool']))
      expect(result.has('agent-2')).toBe(false)
      expect(result.has('skill-2')).toBe(false)
    })

    it('should NOT cross to peer skills when clicking an agent', () => {
      // From agent-1: up to team-1, down to skill-1, skill-1→cli:tool
      // team-1→agent-2 blocked (agent type already visited)
      const result = collectNeighborhood('agent-1', edgeMap(edges), nodeMap(nodes))
      expect(result).toEqual(new Set(['agent-1', 'team-1', 'skill-1', 'cli:tool']))
      expect(result.has('agent-2')).toBe(false)
      expect(result.has('skill-2')).toBe(false)
    })

    it('should include both agents and their subtrees when clicking team', () => {
      // From team-1: down to agent-1 + agent-2 (both are type agent, but
      // they're direct neighbors — each path is separate).
      // Path 1: team→agent-1→skill-1→cli:tool
      // Path 2: team→agent-2→skill-2 (no cli under skill-2)
      const result = collectNeighborhood('team-1', edgeMap(edges), nodeMap(nodes))
      expect(result).toEqual(new Set([
        'team-1', 'agent-1', 'agent-2', 'skill-1', 'skill-2', 'cli:tool',
      ]))
    })
  })

  describe('lateral edges should not spread to peers', () => {
    it('should NOT follow path-ref edge between two skills', () => {
      // skill-1 has a path-ref to skill-2 (same type → blocked)
      const nodes = [
        makeNode('skill-1', 'skill'),
        makeNode('skill-2', 'skill'),
        makeNode('agent-1', 'agent'),
      ]
      const edges = [
        makeEdge('agent-1', 'skill-1'),
        makeEdge('skill-1', 'skill-2', 'path-ref'),
      ]
      const result = collectNeighborhood('skill-1', edgeMap(edges), nodeMap(nodes))
      expect(result).toEqual(new Set(['skill-1', 'agent-1']))
      expect(result.has('skill-2')).toBe(false)
    })

    it('should NOT follow cross-team edge between agents', () => {
      const nodes = [
        makeNode('agent-1', 'agent'),
        makeNode('agent-2', 'agent'),
        makeNode('team-1', 'team'),
      ]
      const edges = [
        makeEdge('team-1', 'agent-1'),
        makeEdge('agent-1', 'agent-2', 'code-usage'),
      ]
      const result = collectNeighborhood('agent-1', edgeMap(edges), nodeMap(nodes))
      expect(result).toEqual(new Set(['agent-1', 'team-1']))
      expect(result.has('agent-2')).toBe(false)
    })
  })

  describe('cycles and self-loops', () => {
    it('should handle a cycle without infinite loop', () => {
      const nodes = [
        makeNode('t1', 'team'),
        makeNode('a1', 'agent'),
        makeNode('s1', 'skill'),
      ]
      const edges = [
        makeEdge('t1', 'a1'),
        makeEdge('a1', 's1'),
        makeEdge('s1', 't1'),  // cycle back
      ]
      const result = collectNeighborhood('t1', edgeMap(edges), nodeMap(nodes))
      expect(result).toEqual(new Set(['t1', 'a1', 's1']))
    })

    it('should handle self-loop gracefully', () => {
      const nodes = [makeNode('a1', 'agent'), makeNode('t1', 'team')]
      const edges = [
        makeEdge('a1', 'a1'),  // self-loop (same type → blocked)
        makeEdge('t1', 'a1'),
      ]
      const result = collectNeighborhood('a1', edgeMap(edges), nodeMap(nodes))
      expect(result).toEqual(new Set(['a1', 't1']))
    })
  })

  describe('edge cases', () => {
    it('should skip neighbors not in nodeMap (filtered out)', () => {
      const allNodes = [
        makeNode('s1', 'skill'),
        makeNode('a1', 'agent'),
        makeNode('c1', 'cli'),
      ]
      const edges = [makeEdge('a1', 's1'), makeEdge('a1', 'c1')]
      // Only s1 and a1 in nodeMap — c1 is "filtered out"
      const filteredMap = nodeMap(allNodes.slice(0, 2))
      const result = collectNeighborhood('s1', edgeMap(edges), filteredMap)
      expect(result).toEqual(new Set(['s1', 'a1']))
    })

    it('should handle node with edges but no valid neighbors', () => {
      const nodes = [makeNode('s1', 'skill')]
      const edges = [makeEdge('s1', 'ghost')]
      const result = collectNeighborhood('s1', edgeMap(edges), nodeMap(nodes))
      expect(result).toEqual(new Set(['s1']))
    })

    it('should handle multiple edges between same pair', () => {
      const nodes = [makeNode('a1', 'agent'), makeNode('s1', 'skill')]
      const edges = [
        makeEdge('a1', 's1', 'bold-listed'),
        makeEdge('a1', 's1', 'code-usage'),
        makeEdge('s1', 'a1', 'path-ref'),
      ]
      const result = collectNeighborhood('a1', edgeMap(edges), nodeMap(nodes))
      expect(result).toEqual(new Set(['a1', 's1']))
    })

    it('should handle disconnected graph components', () => {
      const nodes = [
        makeNode('a1', 'agent'),
        makeNode('s1', 'skill'),
        makeNode('a2', 'agent'),
        makeNode('s2', 'skill'),
      ]
      const edges = [
        makeEdge('a1', 's1'),
        makeEdge('a2', 's2'),  // separate component
      ]
      const result = collectNeighborhood('a1', edgeMap(edges), nodeMap(nodes))
      expect(result).toEqual(new Set(['a1', 's1']))
      expect(result.has('a2')).toBe(false)
      expect(result.has('s2')).toBe(false)
    })

    it('should handle empty adjacentEdgesMap', () => {
      const nodes = [makeNode('s1', 'skill')]
      const result = collectNeighborhood('s1', new Map(), nodeMap(nodes))
      expect(result).toEqual(new Set(['s1']))
    })
  })
})
