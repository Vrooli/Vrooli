/**
 * GraphSettingsContent - Settings body for graph visualization controls.
 *
 * Rendered inside OverlayModal by ViewOverlay. Contains no modal chrome.
 * Adapted from the former GraphToolbar for modal layout.
 */

import { Users, Bot, Sparkles, Terminal, LayoutGrid, RefreshCw, Maximize2 } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useGraphStore } from '@/stores/graphStore'

export function GraphSettingsContent() {
  const filters = useGraphStore((s) => s.filters)
  const setFilter = useGraphStore((s) => s.setFilter)
  const layoutDirection = useGraphStore((s) => s.layoutDirection)
  const setLayoutDirection = useGraphStore((s) => s.setLayoutDirection)
  const regenerateGraph = useGraphStore((s) => s.regenerateGraph)
  const requestFitView = useGraphStore((s) => s.requestFitView)
  const loading = useGraphStore((s) => s.loading)

  const toggleButtons = [
    { key: 'showTeams' as const, label: 'Teams', icon: Users, active: filters.showTeams },
    { key: 'showAgents' as const, label: 'Agents', icon: Bot, active: filters.showAgents },
    { key: 'showSkills' as const, label: 'Skills', icon: Sparkles, active: filters.showSkills },
    { key: 'showCLIs' as const, label: 'CLIs', icon: Terminal, active: filters.showCLIs },
  ]

  return (
    <div className="space-y-6">
      {/* Visibility section */}
      <section>
        <h3 className="text-sm font-medium text-indigo-400 mb-3">Visibility</h3>
        <div className="grid grid-cols-2 gap-2 mb-4">
          {toggleButtons.map((btn) => {
            const Icon = btn.icon
            return (
              <button
                key={btn.key}
                type="button"
                onClick={() => setFilter(btn.key, !btn.active)}
                className={cn(
                  'flex items-center gap-2 px-3 py-2 text-sm font-medium rounded-lg border transition-colors',
                  btn.active
                    ? 'bg-indigo-500/30 border-indigo-500/50 text-indigo-300'
                    : 'bg-slate-800/50 border-slate-700 text-slate-400 hover:text-slate-200 hover:border-slate-600',
                )}
                title={`${btn.active ? 'Hide' : 'Show'} ${btn.label}`}
              >
                <Icon className="h-4 w-4" />
                {btn.label}
              </button>
            )
          })}
        </div>

        <div className="flex items-center gap-3">
          <span className="text-xs text-slate-400 whitespace-nowrap">
            Health &ge; {Math.round(filters.healthThreshold * 100)}%
          </span>
          <input
            type="range"
            min={0}
            max={1}
            step={0.05}
            value={filters.healthThreshold}
            onChange={(e) => setFilter('healthThreshold', parseFloat(e.target.value))}
            className="flex-1 h-1 accent-indigo-500"
          />
        </div>
      </section>

      <div className="border-t border-white/10" />

      {/* Layout section */}
      <section>
        <h3 className="text-sm font-medium text-indigo-400 mb-3">Layout</h3>
        <div className="flex gap-2 mb-3">
          <button
            type="button"
            onClick={() => setLayoutDirection('TB')}
            className={cn(
              'flex-1 flex items-center justify-center gap-2 px-3 py-2 text-sm font-medium rounded-lg border transition-all',
              layoutDirection === 'TB'
                ? 'bg-indigo-500/30 border-indigo-500/50 text-indigo-300'
                : 'bg-slate-800/50 border-slate-700 text-slate-400 hover:text-slate-200 hover:border-slate-600',
            )}
          >
            <LayoutGrid className="h-4 w-4" />
            Vertical
          </button>
          <button
            type="button"
            onClick={() => setLayoutDirection('LR')}
            className={cn(
              'flex-1 flex items-center justify-center gap-2 px-3 py-2 text-sm font-medium rounded-lg border transition-all',
              layoutDirection === 'LR'
                ? 'bg-indigo-500/30 border-indigo-500/50 text-indigo-300'
                : 'bg-slate-800/50 border-slate-700 text-slate-400 hover:text-slate-200 hover:border-slate-600',
            )}
          >
            <LayoutGrid className="h-4 w-4" />
            Horizontal
          </button>
        </div>

        <div className="flex gap-2">
          <button
            type="button"
            onClick={requestFitView}
            className={cn(
              'flex-1 flex items-center justify-center gap-2 px-3 py-2 text-sm font-medium rounded-lg',
              'bg-slate-800/50 border border-slate-700 text-slate-300 hover:text-white hover:bg-slate-700 transition-colors',
            )}
            title="Fit to view"
          >
            <Maximize2 className="h-4 w-4" />
            Fit to View
          </button>
          <button
            type="button"
            onClick={() => void regenerateGraph()}
            disabled={loading}
            className={cn(
              'flex-1 flex items-center justify-center gap-2 px-3 py-2 text-sm font-medium rounded-lg',
              'bg-slate-800/50 border border-slate-700 text-slate-300 hover:text-white hover:bg-slate-700 transition-colors',
              loading && 'opacity-50 cursor-not-allowed',
            )}
            title="Regenerate graph"
          >
            <RefreshCw className={cn('h-4 w-4', loading && 'animate-spin')} />
            Regenerate
          </button>
        </div>
      </section>
    </div>
  )
}
