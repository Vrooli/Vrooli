import type { GraphData, GraphHealthConfig, HealthScore, GraphEdge, GraphNode } from '@/lib/schemas'

type FactorKey = 'outgoing-edges' | 'incoming-edges' | 'code-usage' | 'recent-activity'

export function buildPreviewHealthScores(
  graph: GraphData,
  config: GraphHealthConfig,
  baseline: HealthScore[],
): HealthScore[] {
  const baseScores = scoreAllWithConfig(graph, config)
  return applyCLIHealthPolicy(graph, baseScores, baseline, config)
}

function scoreAllWithConfig(graph: GraphData, cfg: GraphHealthConfig): HealthScore[] {
  return graph.nodes.map((node) => {
    const weights = weightsForNode(node, cfg)
    const factors: Record<FactorKey, number> = {
      'outgoing-edges': outgoingEdgesScore(node.id, graph.edges),
      'incoming-edges': incomingEdgesScore(node.id, graph.edges),
      'code-usage': codeUsageScore(node.id, graph.edges),
      'recent-activity': recentActivityScore(node.id),
    }

    const weightedTotal = (
      factors['outgoing-edges'] * weights.outgoingEdges +
      factors['incoming-edges'] * weights.incomingEdges +
      factors['code-usage'] * weights.codeUsage +
      factors['recent-activity'] * weights.recentActivity
    )
    const weightSum = (
      weights.outgoingEdges +
      weights.incomingEdges +
      weights.codeUsage +
      weights.recentActivity
    )

    return {
      nodeId: node.id,
      score: weightSum > 0 ? weightedTotal / weightSum : 0,
      factors,
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
      })
      continue
    }

    const baseScenarioScore = baselineByNodeId.get(node.id)?.score
    const scenarioScore = baseScenarioScore ?? cfg.cli.scenarioFallbackScore
    scoreByNodeId.set(node.id, {
      nodeId: node.id,
      score: scenarioScore,
      factors: { 'scenario-completeness': scenarioScore },
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

function stripCLI(nodeId: string): string {
  return nodeId.startsWith('cli:') ? nodeId.slice(4) : nodeId
}
