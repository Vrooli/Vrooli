/**
 * GraphNode - Custom React Flow node for the dependency graph.
 *
 * Shape varies by node type:
 * - Rectangle: Team
 * - Circle: Agent
 * - Diamond: Skill
 * - Hexagon: CLI
 *
 * Health is encoded by both fill and border color:
 * - Critical (0-0.3): Red
 * - Warning (0.3-0.6): Yellow
 * - Healthy (0.6-1.0): Green
 */

import { memo } from 'react'
import { Handle, Position, type NodeProps, type Node } from '@xyflow/react'
import { cn } from '@/lib/utils'
import type { NodeType } from '@/lib/schemas'

export interface GraphNodeData extends Record<string, unknown> {
  label: string
  nodeType: NodeType
  healthScore: number | null
  queryState: 'normal' | 'selected' | 'dimmed'
}

type GraphNodeProps = NodeProps<Node<GraphNodeData, 'graphNode'>>

const TYPE_SHAPES: Record<NodeType, string> = {
  team: 'rounded-lg',        // Rectangle
  agent: 'rounded-full',     // Circle
  skill: 'rotate-45',        // Diamond (rotated square)
  action: 'rounded-md',       // Operational command contract
  cli: 'clip-hexagon',       // Hexagon
}

function getHealthClasses(score: number | null): { background: string; border: string; text: string } {
  if (score === null) {
    return {
      background: 'bg-muted/20',
      border: 'border-border/70',
      text: 'text-foreground',
    }
  }

  if (score < 0.3) {
    return {
      background: 'bg-red-500/20',
      border: 'border-red-400/90',
      text: 'text-red-900 dark:text-red-100',
    }
  }

  if (score < 0.6) {
    return {
      background: 'bg-yellow-500/20',
      border: 'border-yellow-300/90',
      text: 'text-yellow-900 dark:text-yellow-100',
    }
  }

  return {
    background: 'bg-emerald-500/20',
    border: 'border-emerald-300/80',
    text: 'text-emerald-900 dark:text-emerald-100',
  }
}

function GraphNodeComponent({ data }: GraphNodeProps) {
  const { label, nodeType, healthScore, queryState } = data
  const shape = TYPE_SHAPES[nodeType]
  const health = getHealthClasses(healthScore)
  const isDiamond = nodeType === 'skill'
  const isQuerySelected = queryState === 'selected'
  const isQueryDimmed = queryState === 'dimmed'

  const appearance = isQueryDimmed
    ? {
      background: 'bg-muted/30',
      border: 'border-border/40',
      text: 'text-muted-foreground',
    }
    : health

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
          'px-4 py-2.5 border-2 cursor-pointer transition-all',
          'hover:brightness-110',
          appearance.background,
          appearance.border,
          shape,
          isQuerySelected && 'ring-2 ring-white/40 shadow-lg',
          isQueryDimmed && 'opacity-45 saturate-50',
          isDiamond && 'w-[100px] h-[100px] flex items-center justify-center',
          !isDiamond && 'min-w-[120px] max-w-[180px]',
        )}
      >
        <div className={cn(isDiamond && '-rotate-45')}>
          <p className={cn(
            'text-xs font-medium text-center truncate',
            appearance.text,
          )}>
            {label}
          </p>
          {healthScore !== null && (
            <p className={cn('text-[10px] text-center mt-0.5', isQueryDimmed ? 'text-muted-foreground/80' : 'text-foreground/75')}>
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
