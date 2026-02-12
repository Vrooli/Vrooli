/**
 * GraphToolbar - Filter toggles and controls for the graph view.
 *
 * Includes:
 * - Type filter toggles (Teams/Agents/Skills/CLIs)
 * - Health threshold slider
 * - Layout direction toggle
 * - Zoom-to-fit button (callback)
 * - Regenerate button
 */

import { Users, Bot, Sparkles, Terminal, LayoutGrid, RefreshCw, Maximize2 } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useGraphStore } from '@/stores/graphStore'

type LayoutDirection = 'TB' | 'LR'

interface GraphToolbarProps {
  layoutDirection: LayoutDirection
  onToggleLayout: () => void
  onFitView: () => void
  className?: string
}

export function GraphToolbar({ layoutDirection, onToggleLayout, onFitView, className }: GraphToolbarProps) {
  const filters = useGraphStore((s) => s.filters)
  const setFilter = useGraphStore((s) => s.setFilter)
  const regenerateGraph = useGraphStore((s) => s.regenerateGraph)
  const loading = useGraphStore((s) => s.loading)

  const toggleButtons = [
    { key: 'showTeams' as const, label: 'Teams', icon: Users, active: filters.showTeams },
    { key: 'showAgents' as const, label: 'Agents', icon: Bot, active: filters.showAgents },
    { key: 'showSkills' as const, label: 'Skills', icon: Sparkles, active: filters.showSkills },
    { key: 'showCLIs' as const, label: 'CLIs', icon: Terminal, active: filters.showCLIs },
  ]

  return (
    <div className={cn('flex items-center gap-2 flex-wrap', className)}>
      {/* Type filter toggles */}
      {toggleButtons.map((btn) => {
        const Icon = btn.icon
        return (
          <button
            key={btn.key}
            type="button"
            onClick={() => setFilter(btn.key, !btn.active)}
            className={cn(
              'flex items-center gap-1 px-2 py-1 text-xs font-medium rounded-md border transition-colors',
              btn.active
                ? 'bg-primary/20 border-primary/50 text-primary'
                : 'bg-card border-border text-muted-foreground hover:bg-muted',
            )}
            title={`${btn.active ? 'Hide' : 'Show'} ${btn.label}`}
          >
            <Icon className="h-3 w-3" />
            {btn.label}
          </button>
        )
      })}

      <div className="w-px h-5 bg-border" />

      {/* Layout toggle */}
      <button
        type="button"
        onClick={onToggleLayout}
        className={cn(
          'flex items-center gap-1 px-2 py-1 text-xs font-medium rounded-md',
          'bg-card border border-border text-foreground hover:bg-muted transition-colors',
        )}
        title={`Layout: ${layoutDirection === 'TB' ? 'Vertical' : 'Horizontal'}`}
      >
        <LayoutGrid className="h-3 w-3" />
        {layoutDirection === 'TB' ? 'Vertical' : 'Horizontal'}
      </button>

      {/* Fit view */}
      <button
        type="button"
        onClick={onFitView}
        className={cn(
          'flex items-center gap-1 px-2 py-1 text-xs font-medium rounded-md',
          'bg-card border border-border text-foreground hover:bg-muted transition-colors',
        )}
        title="Fit to view"
      >
        <Maximize2 className="h-3 w-3" />
      </button>

      {/* Regenerate */}
      <button
        type="button"
        onClick={() => void regenerateGraph()}
        disabled={loading}
        className={cn(
          'flex items-center gap-1 px-2 py-1 text-xs font-medium rounded-md',
          'bg-card border border-border text-foreground hover:bg-muted transition-colors',
          loading && 'opacity-50 cursor-not-allowed',
        )}
        title="Regenerate graph"
      >
        <RefreshCw className={cn('h-3 w-3', loading && 'animate-spin')} />
      </button>

      {/* Health threshold slider */}
      <div className="flex items-center gap-1.5 ml-1">
        <span className="text-[10px] text-muted-foreground whitespace-nowrap">
          Health &ge; {Math.round(filters.healthThreshold * 100)}%
        </span>
        <input
          type="range"
          min={0}
          max={1}
          step={0.05}
          value={filters.healthThreshold}
          onChange={(e) => setFilter('healthThreshold', parseFloat(e.target.value))}
          className="w-16 h-1 accent-primary"
        />
      </div>
    </div>
  )
}
