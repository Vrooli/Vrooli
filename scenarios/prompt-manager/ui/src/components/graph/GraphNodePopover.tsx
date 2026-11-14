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
  'action-use': 'Action Use',
  'action-command': 'Action Command',
}

const TYPE_BADGES: Record<string, { label: string; className: string }> = {
  team: { label: 'Team', className: 'bg-blue-500/20 text-blue-700 dark:text-blue-300 border-blue-500/40' },
  agent: { label: 'Agent', className: 'bg-emerald-500/20 text-emerald-700 dark:text-emerald-300 border-emerald-500/40' },
  skill: { label: 'Skill', className: 'bg-violet-500/20 text-violet-700 dark:text-violet-300 border-violet-500/40' },
  action: { label: 'Action', className: 'bg-cyan-500/20 text-cyan-700 dark:text-cyan-300 border-cyan-500/40' },
  cli: { label: 'CLI', className: 'bg-orange-500/20 text-orange-700 dark:text-orange-300 border-orange-500/40' },
}

interface GraphNodePopoverProps {
  node: GraphNode
  healthScore?: HealthScore | null
  edges: GraphEdge[]
  screenX: number
  screenY: number
  onClose: () => void
  onNavigate: () => void
  variant?: 'desktop' | 'mobile'
}

function GraphNodeDetails({
  node,
  healthScore,
  edges,
  onClose,
  onNavigate,
  variant,
}: {
  node: GraphNode
  healthScore?: HealthScore | null
  edges: GraphEdge[]
  onClose: () => void
  onNavigate: () => void
  variant: 'desktop' | 'mobile'
}) {
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
    <>
      {variant === 'mobile' && <div className="mx-auto mb-2 h-1.5 w-10 rounded-full bg-border" />}

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

            {healthScore.messages.length > 0 && (
              <div className="space-y-1.5 pt-2 border-t border-border/70">
                <p className="text-muted-foreground">Recommendations</p>
                {healthScore.messages.slice(0, 3).map((message) => (
                  <div key={message.key} className="space-y-0.5">
                    <p className={cn(
                      'font-medium',
                      message.severity === 'critical' && 'text-red-300',
                      message.severity === 'warning' && 'text-yellow-300',
                      message.severity === 'info' && 'text-sky-300',
                    )}>
                      {message.summary}
                    </p>
                    {message.recommendation && (
                      <p className="text-muted-foreground">{message.recommendation}</p>
                    )}
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
    </>
  )
}

export function GraphNodePopover({
  node,
  healthScore,
  edges,
  screenX,
  screenY,
  onClose,
  onNavigate,
  variant = 'desktop',
}: GraphNodePopoverProps) {
  if (variant === 'mobile') {
    return (
      <>
        <div data-testid="graph-node-popover-backdrop" className="fixed inset-0 z-[100] bg-black/50 motion-safe:animate-in motion-safe:fade-in-0 motion-safe:duration-200" onClick={onClose} />
        <div
          data-testid="graph-node-popover-mobile"
          className="fixed inset-x-0 bottom-0 z-[101] max-h-[80vh] overflow-y-auto rounded-t-2xl border border-border bg-popover text-popover-foreground text-xs shadow-2xl motion-safe:animate-in motion-safe:slide-in-from-bottom-6 motion-safe:fade-in-0 motion-safe:duration-200"
          role="dialog"
          aria-modal="true"
          aria-label={`Graph node ${node.label}`}
          onClick={(e) => e.stopPropagation()}
        >
          <GraphNodeDetails
            node={node}
            healthScore={healthScore}
            edges={edges}
            onClose={onClose}
            onNavigate={onNavigate}
            variant={variant}
          />
        </div>
      </>
    )
  }

  return (
    <div
      data-testid="graph-node-popover-desktop"
      className="fixed z-[100] w-[280px] bg-popover border border-border rounded-lg shadow-xl text-popover-foreground text-xs motion-safe:animate-in motion-safe:fade-in-0 motion-safe:zoom-in-95 motion-safe:duration-150"
      style={{ left: screenX, top: screenY }}
      onClick={(e) => e.stopPropagation()}
    >
      <GraphNodeDetails
        node={node}
        healthScore={healthScore}
        edges={edges}
        onClose={onClose}
        onNavigate={onNavigate}
        variant={variant}
      />
    </div>
  )
}
