/**
 * GraphNodePopover - Click-anchored detail panel for graph nodes.
 *
 * Shows:
 * - Header: node label + type badge + close button
 * - Health section: overall score (color-coded) + factor breakdown
 * - Connections section: grouped edge counts by kind
 * - Footer: "Go to editor" button
 */

import { cn } from '@/lib/utils'
import type { GraphNode, GraphEdge, HealthScore, EdgeKind } from '@/lib/schemas'

const EDGE_KIND_LABELS: Record<EdgeKind, string> = {
  'cli-read': 'CLI Read',
  'bold-listed': 'Bold Listed',
  'default-scope': 'Default Scope',
  'path-ref': 'Path Reference',
  'membership': 'Membership',
  'code-usage': 'Code Usage',
}

const TYPE_BADGES: Record<string, { label: string; className: string }> = {
  team: { label: 'Team', className: 'bg-blue-500/20 text-blue-300 border-blue-500/40' },
  agent: { label: 'Agent', className: 'bg-emerald-500/20 text-emerald-300 border-emerald-500/40' },
  skill: { label: 'Skill', className: 'bg-violet-500/20 text-violet-300 border-violet-500/40' },
  cli: { label: 'CLI', className: 'bg-orange-500/20 text-orange-300 border-orange-500/40' },
}

interface GraphNodePopoverProps {
  node: GraphNode
  healthScore?: HealthScore | null
  edges: GraphEdge[]
  screenX: number
  screenY: number
  onClose: () => void
  onNavigate: () => void
}

export function GraphNodePopover({
  node,
  healthScore,
  edges,
  screenX,
  screenY,
  onClose,
  onNavigate,
}: GraphNodePopoverProps) {
  const badge = TYPE_BADGES[node.type] ?? { label: node.type, className: 'bg-muted text-muted-foreground border-border' }

  // Group edges by kind with counts
  const edgeCounts = new Map<EdgeKind, { inbound: number; outbound: number }>()
  for (const edge of edges) {
    const entry = edgeCounts.get(edge.kind) ?? { inbound: 0, outbound: 0 }
    if (edge.to === node.id) {
      entry.inbound++
    } else {
      entry.outbound++
    }
    edgeCounts.set(edge.kind, entry)
  }

  return (
    <div
      className="fixed z-[100] w-[280px] bg-popover border border-border rounded-lg shadow-xl text-popover-foreground text-xs"
      style={{ left: screenX + 16, top: screenY - 8 }}
      onClick={(e) => e.stopPropagation()}
    >
      {/* Header */}
      <div className="flex items-start justify-between gap-2 px-3 pt-3 pb-2">
        <div className="min-w-0">
          <p className="font-semibold text-sm break-words">{node.label}</p>
          <span className={cn('inline-block mt-1 px-1.5 py-0.5 text-[10px] border rounded', badge.className)}>
            {badge.label}
          </span>
        </div>
        <button
          type="button"
          onClick={onClose}
          className="shrink-0 p-1 rounded hover:bg-muted text-muted-foreground hover:text-foreground transition-colors"
          aria-label="Close"
        >
          <svg width="14" height="14" viewBox="0 0 14 14" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round">
            <path d="M3 3l8 8M11 3l-8 8" />
          </svg>
        </button>
      </div>

      {/* Health section */}
      <div className="px-3 py-2 border-t border-border">
        {healthScore ? (
          <div className="space-y-1.5">
            <div className="flex items-center justify-between">
              <span className="text-muted-foreground">Health</span>
              <span className={cn(
                'font-semibold',
                healthScore.score < 0.3 && 'text-red-400',
                healthScore.score >= 0.3 && healthScore.score < 0.6 && 'text-yellow-400',
                healthScore.score >= 0.6 && 'text-green-400',
              )}>
                {Math.round(healthScore.score * 100)}%
              </span>
            </div>

            {/* Health bar */}
            <div className="h-1.5 bg-muted rounded-full overflow-hidden">
              <div
                className={cn(
                  'h-full rounded-full transition-all',
                  healthScore.score < 0.3 && 'bg-red-500',
                  healthScore.score >= 0.3 && healthScore.score < 0.6 && 'bg-yellow-500',
                  healthScore.score >= 0.6 && 'bg-green-500',
                )}
                style={{ width: `${Math.round(healthScore.score * 100)}%` }}
              />
            </div>

            {Object.keys(healthScore.factors).length > 0 && (
              <div className="space-y-0.5 pt-1">
                {Object.entries(healthScore.factors).map(([name, value]) => (
                  <div key={name} className="flex items-center justify-between">
                    <span className="text-muted-foreground truncate mr-2">{name}</span>
                    <span className="tabular-nums">{Math.round(value * 100)}%</span>
                  </div>
                ))}
              </div>
            )}
          </div>
        ) : (
          <p className="text-muted-foreground italic">Neutral (not scored)</p>
        )}
      </div>

      {/* Connections section */}
      {edgeCounts.size > 0 && (
        <div className="px-3 py-2 border-t border-border">
          <p className="text-muted-foreground mb-1.5">Connections</p>
          <div className="space-y-0.5">
            {Array.from(edgeCounts.entries()).map(([kind, counts]) => (
              <div key={kind} className="flex items-center justify-between">
                <span className="text-muted-foreground truncate mr-2">
                  {EDGE_KIND_LABELS[kind]}
                </span>
                <span className="tabular-nums whitespace-nowrap">
                  {counts.inbound > 0 && <span title="Inbound">{counts.inbound} in</span>}
                  {counts.inbound > 0 && counts.outbound > 0 && ' / '}
                  {counts.outbound > 0 && <span title="Outbound">{counts.outbound} out</span>}
                </span>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Footer */}
      {(node.type === 'skill' || node.type === 'agent' || node.type === 'team') && (
        <div className="px-3 py-2 border-t border-border">
          <button
            type="button"
            onClick={onNavigate}
            className="w-full px-3 py-1.5 text-xs bg-primary text-primary-foreground rounded-md hover:bg-primary/90 transition-colors"
          >
            Go to editor
          </button>
        </div>
      )}
    </div>
  )
}
