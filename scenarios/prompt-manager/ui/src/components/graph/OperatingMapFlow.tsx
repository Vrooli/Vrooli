import { useEffect, useMemo, useState } from 'react'
import { type Edge, type Node } from '@xyflow/react'
import { useNavigate } from 'react-router-dom'
import { api } from '@/lib/api'
import type { OperatingMap } from '@/lib/schemas'
import { teamDetailPath } from '@/app/routes/route-paths'
import { FlowShell, layoutFlowDagre } from './FlowShell'

export function OperatingMapFlow() {
  const navigate = useNavigate(); const [data, setData] = useState<OperatingMap | null>(null)
  useEffect(() => { api.getOperatingMap().then(setData).catch(() => setData({ teams: [], topics: [], edges: [] })) }, [])
  const { nodes, edges } = useMemo(() => {
    if (!data) return { nodes: [] as Node[], edges: [] as Edge[] }
    const teams = data.teams.map((team) => ({
      id: team.id,
      position: { x: 0, y: 0 },
      data: { label: `${team.label}\n${team.goal_linkage}\ncontract: ${team.valid ? 'valid' : 'invalid'}` },
      // React Flow's default node stylesheet has a white background. Supply the
      // full token-based surface so this projection follows both app themes.
      style: {
        background: 'hsl(var(--card))',
        color: 'hsl(var(--card-foreground))',
        border: '1px solid hsl(var(--primary))',
        borderRadius: 8,
        padding: 10,
        whiteSpace: 'pre-line',
      },
    }))
    const topics = data.topics.map((topic) => ({
      id: topic.id,
      position: { x: 0, y: 0 },
      data: { label: topic.label },
      style: {
        background: 'hsl(var(--secondary))',
        color: 'hsl(var(--secondary-foreground))',
        border: '1px solid hsl(var(--border))',
        borderRadius: 999,
        padding: 8,
      },
    }))
    const edges = data.edges.map((edge, index) => ({
      id: `${edge.from}-${edge.to}-${index}`,
      source: edge.from,
      target: edge.to,
      style: { stroke: 'hsl(var(--muted-foreground))', strokeWidth: 1.5 },
    }))
    return layoutFlowDagre([...teams, ...topics], edges, { direction: 'LR', nodeWidth: 220, nodeHeight: 72, nodeSep: 36, rankSep: 150 })
  }, [data])
  return <FlowShell nodes={nodes} edges={edges} fitView onNodeClick={(_, node) => data?.teams.some((team) => team.id === node.id) && navigate(teamDetailPath(node.id, { tab: 'members', subTab: 'topics' }))} />
}
