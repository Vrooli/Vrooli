import { describe, it, expect } from 'vitest'
import { buildPreviewHealthScores } from './graphHealthPreview'
import type { GraphData, GraphHealthConfig } from '@/lib/schemas'

const config: GraphHealthConfig = {
  team: { outgoingEdges: 1, incomingEdges: 1, codeUsage: 0.5, recentActivity: 0.5, skillContentLength: 0.75, agentContextLoad: 0.75, teamMemberCountBalance: 0.75, teamRoleCoverage: 0.75 },
  agent: { outgoingEdges: 1, incomingEdges: 1, codeUsage: 0.5, recentActivity: 0.5, skillContentLength: 0.75, agentContextLoad: 0.75, teamMemberCountBalance: 0.75, teamRoleCoverage: 0.75 },
  skill: { outgoingEdges: 1, incomingEdges: 1, codeUsage: 0.5, recentActivity: 0.5, skillContentLength: 0.75, agentContextLoad: 0.75, teamMemberCountBalance: 0.75, teamRoleCoverage: 0.75 },
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

  it('does not emit recommendations for zero-weight factors', () => {
    const weightedConfig: GraphHealthConfig = {
      ...config,
      team: {
        ...config.team,
        incomingEdges: 0,
        outgoingEdges: 1,
        codeUsage: 0,
        recentActivity: 0,
        skillContentLength: 0,
        agentContextLoad: 0,
        teamMemberCountBalance: 0,
        teamRoleCoverage: 0,
      },
    }
    const graph: GraphData = {
      nodes: [{ id: 'team-1', type: 'team', label: 'Team 1', description: '', status: '', tags: [] }],
      edges: [],
      healthScores: [],
    }

    const out = buildPreviewHealthScores(graph, weightedConfig, [])
    const team = out.find((s) => s.nodeId === 'team-1')
    expect(team).toBeDefined()
    expect(team?.messages.some((m) => m.factor === 'incoming-edges')).toBe(false)
  })

  it('orders recommendations by weighted impact', () => {
    const weightedConfig: GraphHealthConfig = {
      ...config,
      team: {
        ...config.team,
        outgoingEdges: 2,
        incomingEdges: 0.2,
        codeUsage: 0,
        recentActivity: 0,
        skillContentLength: 0,
        agentContextLoad: 0,
        teamMemberCountBalance: 0,
        teamRoleCoverage: 0,
      },
    }
    const graph: GraphData = {
      nodes: [{ id: 'team-1', type: 'team', label: 'Team 1', description: '', status: '', tags: [] }],
      edges: [],
      healthScores: [],
    }

    const out = buildPreviewHealthScores(graph, weightedConfig, [])
    const team = out.find((s) => s.nodeId === 'team-1')
    expect(team).toBeDefined()
    expect(team?.messages[0]?.factor).toBe('outgoing-edges')
  })
})
