import { describe, it, expect } from 'vitest'
import { buildPreviewHealthScores } from './graphHealthPreview'
import type { GraphData, GraphHealthConfig } from '@/lib/schemas'

const config: GraphHealthConfig = {
  team: { outgoingEdges: 1, incomingEdges: 1, codeUsage: 0.5, recentActivity: 0.5 },
  agent: { outgoingEdges: 1, incomingEdges: 1, codeUsage: 0.5, recentActivity: 0.5 },
  skill: { outgoingEdges: 1, incomingEdges: 1, codeUsage: 0.5, recentActivity: 0.5 },
  cli: { neutralCommands: ['vrooli'], externalToolScore: 0.2, scenarioFallbackScore: 0 },
}

describe('buildPreviewHealthScores', () => {
  it('applies external tool CLI policy from config', () => {
    const graph: GraphData = {
      nodes: [
        { id: 'skill-1', type: 'skill', label: 'Skill 1', description: '', status: '', tags: [] },
        { id: 'cli:grep', type: 'cli', label: 'grep', description: '', status: '', tags: [] },
      ],
      edges: [
        { from: 'skill-1', to: 'cli:grep', kind: 'code-usage', category: 'external-tool', sourceFile: 'SKILL.md', lineNumber: 1 },
      ],
      healthScores: [],
    }

    const out = buildPreviewHealthScores(graph, config, [])
    const cli = out.find((s) => s.nodeId === 'cli:grep')
    expect(cli?.score).toBe(0.2)
  })

  it('removes neutral command CLI rows', () => {
    const graph: GraphData = {
      nodes: [
        { id: 'skill-1', type: 'skill', label: 'Skill 1', description: '', status: '', tags: [] },
        { id: 'cli:vrooli', type: 'cli', label: 'vrooli', description: '', status: '', tags: [] },
      ],
      edges: [
        { from: 'skill-1', to: 'cli:vrooli', kind: 'code-usage', category: 'scenario-cli', command: 'vrooli', sourceFile: 'SKILL.md', lineNumber: 1 },
      ],
      healthScores: [],
    }

    const out = buildPreviewHealthScores(graph, config, [])
    expect(out.some((s) => s.nodeId === 'cli:vrooli')).toBe(false)
  })
})
