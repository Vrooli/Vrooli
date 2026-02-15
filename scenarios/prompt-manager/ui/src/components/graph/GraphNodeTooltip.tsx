/**
 * GraphNodeTooltip - Hover details with health breakdown for graph nodes.
 *
 * Health scores are stored separately in the graph's healthScores array,
 * so the tooltip receives the score as an optional prop.
 */

import { cn } from '@/lib/utils'
import type { GraphNode, HealthScore } from '@/lib/schemas'

interface GraphNodeTooltipProps {
  node: GraphNode
  healthScore?: HealthScore | null
  x: number
  y: number
  className?: string
}

export function GraphNodeTooltip({ node, healthScore, x, y, className }: GraphNodeTooltipProps) {
  return (
    <div
      className={cn(
        'absolute z-50 px-3 py-2 bg-popover border border-border rounded-lg shadow-lg',
        'text-xs text-popover-foreground pointer-events-none',
        'max-w-[240px]',
        className,
      )}
      style={{ left: x + 12, top: y - 8 }}
    >
      <p className="font-medium mb-1">{node.label}</p>
      <p className="text-muted-foreground mb-1.5">
        Type: <span className="capitalize">{node.type}</span>
      </p>

      {healthScore && (
        <div className="space-y-1">
          <div className="flex items-center justify-between">
            <span className="text-muted-foreground">Health</span>
            <span className={cn(
              'font-medium',
              healthScore.score < 0.3 && 'text-red-400',
              healthScore.score >= 0.3 && healthScore.score < 0.6 && 'text-yellow-400',
              healthScore.score >= 0.6 && 'text-green-400',
            )}>
              {Math.round(healthScore.score * 100)}%
            </span>
          </div>

          {Object.keys(healthScore.factors).length > 0 && (
            <div className="border-t border-border pt-1 space-y-0.5">
              {Object.entries(healthScore.factors).map(([name, value]) => (
                <div key={name} className="flex items-center justify-between">
                  <span className="text-muted-foreground">{name}</span>
                  <span>{Math.round(value * 100)}%</span>
                </div>
              ))}
            </div>
          )}

          {healthScore.messages.length > 0 && healthScore.messages[0] && (
            <div className="border-t border-border pt-1">
              <p className="text-muted-foreground">{healthScore.messages[0].summary}</p>
            </div>
          )}
        </div>
      )}

      {!healthScore && (
        <p className="text-muted-foreground italic">No health data</p>
      )}
    </div>
  )
}
