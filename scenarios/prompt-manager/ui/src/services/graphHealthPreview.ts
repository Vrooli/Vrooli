import type {
  GraphData,
  GraphHealthConfig,
  HealthScore,
  GraphEdge,
  GraphNode,
  HealthMessage,
} from '@/lib/schemas'

type FactorKey =
  | 'outgoing-edges'
  | 'incoming-edges'
  | 'code-usage'
  | 'recent-activity'
  | 'skill-content-length'
  | 'agent-context-load'
  | 'team-member-count-balance'
  | 'team-role-coverage'
  | 'action-contract'
  | 'action-command'
  | 'action-examples'
  | 'action-owner'

type FactorWeights = Record<FactorKey, number>

type NodeMetrics = {
  skillContentTokens: number
  agentContextTokens: number
  teamMemberCount: number
  teamDistinctRoleCount: number
  teamMembersWithRoleCount: number
}

export function buildPreviewHealthScores(
  graph: GraphData,
  config: GraphHealthConfig,
  baseline: HealthScore[],
): HealthScore[] {
  const metricsByNodeId = collectNodeMetrics(graph)
  const baselineByNodeId = new Map<string, HealthScore>()
  for (const hs of baseline) baselineByNodeId.set(hs.nodeId, hs)
  const baseScores = scoreAllWithConfig(graph, config, metricsByNodeId, baselineByNodeId)
  return applyCLIHealthPolicy(graph, baseScores, baseline, config)
}

function scoreAllWithConfig(
  graph: GraphData,
  cfg: GraphHealthConfig,
  metricsByNodeId: Map<string, NodeMetrics>,
  baselineByNodeId: Map<string, HealthScore>,
): HealthScore[] {
  return graph.nodes.map((node) => {
    const weights = weightsForNode(node, cfg)
    const metrics = metricsByNodeId.get(node.id) ?? emptyMetrics()
    const factors: Record<FactorKey, number> = {
      'outgoing-edges': outgoingEdgesScore(node.id, graph.edges),
      'incoming-edges': incomingEdgesScore(node.id, graph.edges),
      'code-usage': codeUsageScore(node.id, graph.edges),
      'recent-activity': recentActivityScore(node.id),
      'skill-content-length': skillContentLengthScore(node, metrics),
      'agent-context-load': agentContextLoadScore(node, metrics),
      'team-member-count-balance': teamMemberCountBalanceScore(node, metrics),
      'team-role-coverage': teamRoleCoverageScore(node, metrics),
      'action-contract': node.type === 'action' ? 1 : 0,
      'action-command': node.type === 'action' ? 1 : 0,
      'action-examples': node.type === 'action' ? 1 : 0,
      'action-owner': node.type === 'action' ? 1 : 0,
    }
    const baseline = baselineByNodeId.get(node.id)
    if (baseline) {
      for (const key of Object.keys(factors) as FactorKey[]) {
        const hasMetricSignal = (
          key === 'skill-content-length'
            ? metrics.skillContentTokens > 0
            : key === 'agent-context-load'
              ? metrics.agentContextTokens > 0
              : key === 'team-member-count-balance'
                ? metrics.teamMemberCount > 0
                : key === 'team-role-coverage'
                  ? metrics.teamDistinctRoleCount > 0 || metrics.teamMembersWithRoleCount > 0
                  : true
        )
        if (!hasMetricSignal && baseline.factors[key] !== undefined) {
          factors[key] = baseline.factors[key]
        }
      }
    }

    const weightedTotal = (
      factors['outgoing-edges'] * weights.outgoingEdges +
      factors['incoming-edges'] * weights.incomingEdges +
      factors['code-usage'] * weights.codeUsage +
      factors['recent-activity'] * weights.recentActivity +
      factors['skill-content-length'] * weights.skillContentLength +
      factors['agent-context-load'] * weights.agentContextLoad +
      factors['team-member-count-balance'] * weights.teamMemberCountBalance +
      factors['team-role-coverage'] * weights.teamRoleCoverage +
      factors['action-contract'] * (weights.actionContract ?? 0) +
      factors['action-command'] * (weights.actionCommand ?? 0) +
      factors['action-examples'] * (weights.actionExamples ?? 0) +
      factors['action-owner'] * (weights.actionOwner ?? 0)
    )
    const weightSum = (
      weights.outgoingEdges +
      weights.incomingEdges +
      weights.codeUsage +
      weights.recentActivity +
      weights.skillContentLength +
      weights.agentContextLoad +
      weights.teamMemberCountBalance +
      weights.teamRoleCoverage +
      (weights.actionContract ?? 0) +
      (weights.actionCommand ?? 0) +
      (weights.actionExamples ?? 0) +
      (weights.actionOwner ?? 0)
    )

    const factorWeights = toFactorWeights(weights)
    const messages = buildFactorMessages(node, factors, metrics, factorWeights)

    return {
      nodeId: node.id,
      score: weightSum > 0 ? weightedTotal / weightSum : 0,
      factors,
      messages,
    }
  })
}

function applyCLIHealthPolicy(
  graph: GraphData,
  scores: HealthScore[],
  baseline: HealthScore[],
  cfg: GraphHealthConfig,
): HealthScore[] {
  const nodeById = new Map<string, GraphNode>()
  for (const n of graph.nodes) nodeById.set(n.id, n)

  const scoreByNodeId = new Map<string, HealthScore>()
  for (const hs of scores) scoreByNodeId.set(hs.nodeId, hs)

  const baselineByNodeId = new Map<string, HealthScore>()
  for (const hs of baseline) baselineByNodeId.set(hs.nodeId, hs)

  const usageByNode = new Map<string, { command: string; isScenarioCLI: boolean }>()
  for (const e of graph.edges) {
    if (e.kind !== 'code-usage') continue
    const target = nodeById.get(e.to)
    if (!target || target.type !== 'cli') continue

    const existing = usageByNode.get(e.to) ?? { command: stripCLI(e.to), isScenarioCLI: false }
    const command = e.command?.trim() || existing.command
    const isScenario = existing.isScenarioCLI || e.category === 'scenario-cli'
    usageByNode.set(e.to, { command, isScenarioCLI: isScenario })
  }

  const neutralCommands = new Set(cfg.cli.neutralCommands.map((c) => c.trim()).filter(Boolean))

  for (const node of graph.nodes) {
    if (node.type !== 'cli') continue

    const usage = usageByNode.get(node.id)
    const command = (usage?.command || stripCLI(node.id)).trim()

    if (neutralCommands.has(command)) {
      scoreByNodeId.delete(node.id)
      continue
    }

    if (!usage?.isScenarioCLI) {
      scoreByNodeId.set(node.id, {
        nodeId: node.id,
        score: cfg.cli.externalToolScore,
        factors: { 'cli-portability': cfg.cli.externalToolScore },
        messages: [
          {
            key: 'cli.external-tool',
            severity: 'warning',
            factor: 'cli-portability',
            summary: 'External CLI detected',
            detail: `Command "${command}" is treated as non-scenario tooling.`,
            recommendation: 'Wrap this workflow in a scenario CLI to improve portability.',
            metricValue: cfg.cli.externalToolScore,
            target: 'Scenario CLI usage',
          },
        ],
      })
      continue
    }

    const baseScenarioScore = baselineByNodeId.get(node.id)?.score
    const scenarioScore = baseScenarioScore ?? cfg.cli.scenarioFallbackScore
    scoreByNodeId.set(node.id, {
      nodeId: node.id,
      score: scenarioScore,
      factors: { 'scenario-completeness': scenarioScore },
      messages: [
        {
          key: 'cli.scenario-completeness',
          severity: 'info',
          factor: 'scenario-completeness',
          summary: 'Scenario CLI health applied',
          detail: `CLI "${command}" uses scenario completeness score.`,
          recommendation: 'Improve scenario completeness to raise this CLI node score.',
          metricValue: scenarioScore,
          target: '>= 0.80',
        },
      ],
    })
  }

  const merged: HealthScore[] = []
  for (const n of graph.nodes) {
    const hs = scoreByNodeId.get(n.id)
    if (hs) merged.push(hs)
  }
  return merged
}

function weightsForNode(node: GraphNode, cfg: GraphHealthConfig) {
  if (node.type === 'team') return cfg.team
  if (node.type === 'agent') return cfg.agent
  if (node.type === 'action') return cfg.action ?? cfg.skill
  return cfg.skill
}

function outgoingEdgesScore(nodeId: string, edges: GraphEdge[]): number {
  let count = 0
  for (const e of edges) {
    if (e.from === nodeId) count++
  }
  return Math.min(count / 5, 1)
}

function incomingEdgesScore(nodeId: string, edges: GraphEdge[]): number {
  let count = 0
  for (const e of edges) {
    if (e.to === nodeId) count++
  }
  return Math.min(count / 5, 1)
}

function codeUsageScore(nodeId: string, edges: GraphEdge[]): number {
  let hasScenarioCLI = false
  let hasExternal = false
  for (const e of edges) {
    if (e.from !== nodeId || e.kind !== 'code-usage') continue
    if (e.category === 'scenario-cli') hasScenarioCLI = true
    if (e.category === 'external-tool' || e.category === 'script') hasExternal = true
  }
  if (hasExternal) return 0.1
  if (hasScenarioCLI) return 1
  return 0.5
}

function recentActivityScore(_nodeId: string): number {
  return 0.5
}

function skillContentLengthScore(node: GraphNode, metrics: NodeMetrics): number {
  if (node.type !== 'skill') return 0.5
  const tokens = metrics.skillContentTokens
  if (tokens <= 0) return 0.5
  if (tokens < 150) return 0.5 + (tokens / 150) * 0.5
  if (tokens <= 1800) return 1
  if (tokens >= 4000) return 0
  return 1 - ((tokens - 1800) / (4000 - 1800))
}

function agentContextLoadScore(node: GraphNode, metrics: NodeMetrics): number {
  if (node.type !== 'agent') return 0.5
  const tokens = metrics.agentContextTokens
  if (tokens <= 0) return 0.5
  if (tokens <= 3500) return 1
  if (tokens >= 12000) return 0
  return 1 - ((tokens - 3500) / (12000 - 3500))
}

function teamMemberCountBalanceScore(node: GraphNode, metrics: NodeMetrics): number {
  if (node.type !== 'team') return 0.5
  const members = metrics.teamMemberCount
  if (members <= 0) return 0
  if (members === 1) return 0.25
  if (members === 2) return 0.6
  if (members <= 8) return 1
  if (members <= 14) return 1 - ((members - 8) / (14 - 8)) * (1 - 0.4)
  if (members >= 20) return 0
  return 0.4 - ((members - 14) / (20 - 14)) * (0.4 - 0)
}

function teamRoleCoverageScore(node: GraphNode, metrics: NodeMetrics): number {
  if (node.type !== 'team') return 0.5
  const members = metrics.teamMemberCount
  if (members <= 0) return 0

  let roleVarietyScore = 0
  const distinctRoles = metrics.teamDistinctRoleCount
  if (distinctRoles <= 0) roleVarietyScore = 0
  else if (distinctRoles === 1) roleVarietyScore = 0.3
  else if (distinctRoles === 2) roleVarietyScore = 0.7
  else if (distinctRoles <= 6) roleVarietyScore = 1
  else if (distinctRoles >= 12) roleVarietyScore = 0.5
  else roleVarietyScore = 1 - ((distinctRoles - 6) / (12 - 6)) * (1 - 0.5)

  const coverage = Math.max(0, Math.min(1, metrics.teamMembersWithRoleCount / members))
  return roleVarietyScore * 0.5 + coverage * 0.5
}

function collectNodeMetrics(graph: GraphData): Map<string, NodeMetrics> {
  const metrics = new Map<string, NodeMetrics>()
  for (const node of graph.nodes) {
    metrics.set(node.id, emptyMetrics())
  }

  for (const edge of graph.edges) {
    if (edge.kind !== 'membership') continue
    const m = metrics.get(edge.from)
    if (m) {
      m.teamMemberCount += 1
    }
  }

  return metrics
}

function emptyMetrics(): NodeMetrics {
  return {
    skillContentTokens: 0,
    agentContextTokens: 0,
    teamMemberCount: 0,
    teamDistinctRoleCount: 0,
    teamMembersWithRoleCount: 0,
  }
}

function buildFactorMessages(
  node: GraphNode,
  factors: Record<FactorKey, number>,
  metrics: NodeMetrics,
  weights: FactorWeights,
): HealthMessage[] {
  const ranked: Array<{ message: HealthMessage; impact: number }> = []

  const maybePush = (factor: FactorKey, message: HealthMessage) => {
    const weight = weights[factor]
    if (weight <= 0) return
    ranked.push({
      message,
      impact: weight * (1 - factors[factor]),
    })
  }

  if (factors['outgoing-edges'] < 0.2) {
    maybePush('outgoing-edges', {
      key: 'factor.outgoing-edges.low',
      severity: 'warning',
      factor: 'outgoing-edges',
      summary: 'Low outbound connectivity',
      detail: 'This node has very few outgoing references.',
      recommendation: 'Add explicit links to dependent skills, agents, or tools.',
      metricValue: factors['outgoing-edges'],
      target: '>= 0.60',
    })
  }

  if (factors['incoming-edges'] < 0.2) {
    maybePush('incoming-edges', {
      key: 'factor.incoming-edges.low',
      severity: 'warning',
      factor: 'incoming-edges',
      summary: 'Low inbound discoverability',
      detail: 'Few nodes reference this node.',
      recommendation: 'Cross-reference this node from related teams, agents, or skills.',
      metricValue: factors['incoming-edges'],
      target: '>= 0.60',
    })
  }

  if (factors['code-usage'] <= 0.1) {
    maybePush('code-usage', {
      key: 'factor.code-usage.external',
      severity: 'warning',
      factor: 'code-usage',
      summary: 'External tooling dependency',
      detail: 'Detected external CLI or script usage.',
      recommendation: 'Prefer Vrooli scenario CLIs for reproducibility and orchestration.',
      metricValue: factors['code-usage'],
      target: '1.00',
    })
  }

  if (node.type === 'skill' && factors['skill-content-length'] < 0.5) {
    maybePush('skill-content-length', {
      key: 'skill.content-length.high',
      severity: 'warning',
      factor: 'skill-content-length',
      summary: 'Skill content is oversized',
      detail: `Skill content is about ${Math.round(metrics.skillContentTokens)} tokens.`,
      recommendation: 'Split the skill into smaller focused skills and keep task-critical instructions in the primary file.',
      metricValue: metrics.skillContentTokens,
      target: '150-1800 tokens',
    })
  }

  if (node.type === 'agent' && factors['agent-context-load'] < 0.5) {
    maybePush('agent-context-load', {
      key: 'agent.context-load.high',
      severity: 'warning',
      factor: 'agent-context-load',
      summary: 'Agent context load is high',
      detail: `Agent markdown payload is about ${Math.round(metrics.agentContextTokens)} tokens.`,
      recommendation: 'Move reference-heavy content to shared docs and keep per-agent files concise.',
      metricValue: metrics.agentContextTokens,
      target: '<= 3500 tokens',
    })
  }

  if (node.type === 'team' && factors['team-member-count-balance'] < 0.5) {
    maybePush('team-member-count-balance', {
      key: 'team.member-count.imbalanced',
      severity: 'warning',
      factor: 'team-member-count-balance',
      summary: 'Team size is imbalanced',
      detail: `Team currently has ${Math.round(metrics.teamMemberCount)} members.`,
      recommendation: 'Aim for a balanced team size where collaboration and specialization are both practical.',
      metricValue: metrics.teamMemberCount,
      target: '3-8 members',
    })
  }

  if (node.type === 'team' && factors['team-role-coverage'] < 0.5) {
    const members = Math.max(metrics.teamMemberCount, 1)
    const coverage = metrics.teamMembersWithRoleCount / members
    maybePush('team-role-coverage', {
      key: 'team.role-coverage.low',
      severity: 'warning',
      factor: 'team-role-coverage',
      summary: 'Role coverage is weak',
      detail: `Distinct roles: ${Math.round(metrics.teamDistinctRoleCount)}, members with role assignments: ${Math.round(metrics.teamMembersWithRoleCount)} (${Math.round(coverage * 100)}%).`,
      recommendation: 'Define clearer role assignments and ensure each member has explicit role coverage.',
      metricValue: coverage,
      target: '>= 0.80 assignment coverage and >= 2 distinct roles',
    })
  }

  ranked.sort((a, b) => b.impact - a.impact)
  return ranked.map((entry) => entry.message)
}

function stripCLI(nodeId: string): string {
  return nodeId.startsWith('cli:') ? nodeId.slice(4) : nodeId
}

function toFactorWeights(weights: GraphHealthConfig['team']): FactorWeights {
  return {
    'outgoing-edges': weights.outgoingEdges,
    'incoming-edges': weights.incomingEdges,
    'code-usage': weights.codeUsage,
    'recent-activity': weights.recentActivity,
    'skill-content-length': weights.skillContentLength,
    'agent-context-load': weights.agentContextLoad,
    'team-member-count-balance': weights.teamMemberCountBalance,
    'team-role-coverage': weights.teamRoleCoverage,
    'action-contract': weights.actionContract ?? 0,
    'action-command': weights.actionCommand ?? 0,
    'action-examples': weights.actionExamples ?? 0,
    'action-owner': weights.actionOwner ?? 0,
  }
}
