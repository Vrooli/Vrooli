/**
 * GraphNode - Custom React Flow node for the dependency graph.
 *
 * Shape varies by node type:
 * - Rectangle: Team
 * - Circle: Agent
 * - Diamond: Skill
 * - Hexagon: CLI
 *
 * Color tinted by health score:
 * - Red tint (0-0.3)
 * - Yellow tint (0.3-0.6)
 * - Full color (0.6-1.0)
 */

import { memo } from 'react'
import { Handle, Position, type NodeProps, type Node } from '@xyflow/react'
import { cn } from '@/lib/utils'
import type { NodeType } from '@/lib/schemas'

export interface GraphNodeData extends Record<string, unknown> {
  label: string
  nodeType: NodeType
  healthScore: number | null
  isHighlighted: boolean
}

type GraphNodeProps = NodeProps<Node<GraphNodeData, 'graphNode'>>

const TYPE_COLORS: Record<NodeType, { bg: string; border: string; text: string }> = {
  team: { bg: 'bg-blue-500/20', border: 'border-blue-500/60', text: 'text-blue-300' },
  agent: { bg: 'bg-emerald-500/20', border: 'border-emerald-500/60', text: 'text-emerald-300' },
  skill: { bg: 'bg-violet-500/20', border: 'border-violet-500/60', text: 'text-violet-300' },
  cli: { bg: 'bg-orange-500/20', border: 'border-orange-500/60', text: 'text-orange-300' },
}

const TYPE_SHAPES: Record<NodeType, string> = {
  team: 'rounded-lg',        // Rectangle
  agent: 'rounded-full',     // Circle
  skill: 'rotate-45',        // Diamond (rotated square)
  cli: 'clip-hexagon',       // Hexagon
}

function getHealthTint(score: number | null): string {
  if (score === null) return ''
  if (score < 0.3) return 'ring-2 ring-red-500/50'
  if (score < 0.6) return 'ring-2 ring-yellow-500/50'
  return ''
}

function GraphNodeComponent({ data }: GraphNodeProps) {
  const { label, nodeType, healthScore, isHighlighted } = data
  const colors = TYPE_COLORS[nodeType]
  const shape = TYPE_SHAPES[nodeType]
  const healthTint = getHealthTint(healthScore)
  const isDiamond = nodeType === 'skill'

  return (
    <div className="relative">
      {/* Handles - positioned outside the shape */}
      <Handle
        type="target"
        position={Position.Top}
        className="!w-2.5 !h-2.5 !bg-primary/50 !border-2 !border-background"
      />

      <div
        className={cn(
          'px-4 py-2.5 border cursor-pointer transition-all',
          'hover:brightness-110',
          colors.bg,
          colors.border,
          shape,
          healthTint,
          isHighlighted && 'ring-2 ring-primary shadow-lg shadow-primary/20',
          isDiamond && 'w-[100px] h-[100px] flex items-center justify-center',
          !isDiamond && 'min-w-[120px] max-w-[180px]',
        )}
      >
        <div className={cn(isDiamond && '-rotate-45')}>
          <p className={cn(
            'text-xs font-medium text-center truncate',
            colors.text,
          )}>
            {label}
          </p>
          {healthScore !== null && (
            <p className="text-[10px] text-center text-muted-foreground mt-0.5">
              {Math.round(healthScore * 100)}%
            </p>
          )}
        </div>
      </div>

      <Handle
        type="source"
        position={Position.Bottom}
        className="!w-2.5 !h-2.5 !bg-primary/50 !border-2 !border-background"
      />
    </div>
  )
}

export const GraphFlowNode = memo(GraphNodeComponent)
