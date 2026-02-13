/**
 * GraphSettingsContent - Settings body for graph visualization controls.
 *
 * Rendered inside shared floating panel by ViewOverlay. Contains no panel chrome.
 * Adapted from the former GraphToolbar for panel layout.
 */

import { Users, Bot, Sparkles, Terminal, LayoutGrid, RefreshCw, Maximize2, Link2Off, FoldVertical } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useGraphStore } from '@/stores/graphStore'

export function GraphSettingsContent() {
  const filters = useGraphStore((s) => s.filters)
  const setFilter = useGraphStore((s) => s.setFilter)
  const layoutDirection = useGraphStore((s) => s.layoutDirection)
  const setLayoutDirection = useGraphStore((s) => s.setLayoutDirection)
  const layoutMode = useGraphStore((s) => s.layoutMode)
  const setLayoutMode = useGraphStore((s) => s.setLayoutMode)
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

        <div className="grid grid-cols-1 gap-2 mt-3">
          <button
            type="button"
            onClick={() => setFilter('collapseCLIs', !filters.collapseCLIs)}
            className={cn(
              'flex items-center justify-between gap-2 px-3 py-2 text-xs font-medium rounded-lg border transition-colors',
              filters.collapseCLIs
                ? 'bg-indigo-500/30 border-indigo-500/50 text-indigo-300'
                : 'bg-slate-800/50 border-slate-700 text-slate-400 hover:text-slate-200 hover:border-slate-600',
            )}
            title={filters.collapseCLIs ? 'Expand individual CLI nodes' : 'Collapse CLI nodes into one cluster'}
          >
            <span className="flex items-center gap-2">
              <FoldVertical className="h-3.5 w-3.5" />
              Collapse CLI Nodes
            </span>
          </button>

          <button
            type="button"
            onClick={() => setFilter('showLowSignalEdges', !filters.showLowSignalEdges)}
            className={cn(
              'flex items-center justify-between gap-2 px-3 py-2 text-xs font-medium rounded-lg border transition-colors',
              filters.showLowSignalEdges
                ? 'bg-indigo-500/30 border-indigo-500/50 text-indigo-300'
                : 'bg-slate-800/50 border-slate-700 text-slate-400 hover:text-slate-200 hover:border-slate-600',
            )}
            title={filters.showLowSignalEdges ? 'Hide low-signal edges' : 'Show low-signal edges'}
          >
            <span className="flex items-center gap-2">
              <Link2Off className="h-3.5 w-3.5" />
              Low-Signal Edges
            </span>
          </button>

          <button
            type="button"
            onClick={() => setFilter('autoFitOnChange', !filters.autoFitOnChange)}
            className={cn(
              'flex items-center justify-between gap-2 px-3 py-2 text-xs font-medium rounded-lg border transition-colors',
              filters.autoFitOnChange
                ? 'bg-indigo-500/30 border-indigo-500/50 text-indigo-300'
                : 'bg-slate-800/50 border-slate-700 text-slate-400 hover:text-slate-200 hover:border-slate-600',
            )}
            title={filters.autoFitOnChange ? 'Disable auto-fit after changes' : 'Enable auto-fit after changes'}
          >
            Auto-fit after changes
          </button>
        </div>
      </section>

      <div className="border-t border-white/10" />

      {/* Layout section */}
      <section>
        <h3 className="text-sm font-medium text-indigo-400 mb-3">Layout</h3>
        <div className="grid grid-cols-3 gap-2 mb-3">
          {[
            { id: 'hierarchical' as const, label: 'Hier' },
            { id: 'compact' as const, label: 'Compact' },
            { id: 'grouped' as const, label: 'Grouped' },
          ].map((option) => (
            <button
              key={option.id}
              type="button"
              onClick={() => setLayoutMode(option.id)}
              className={cn(
                'px-2 py-1.5 text-xs font-medium rounded-lg border transition-colors',
                layoutMode === option.id
                  ? 'bg-indigo-500/30 border-indigo-500/50 text-indigo-300'
                  : 'bg-slate-800/50 border-slate-700 text-slate-400 hover:text-slate-200 hover:border-slate-600',
              )}
            >
              {option.label}
            </button>
          ))}
        </div>

        <div className="flex gap-2 mb-3">
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
            Vertical
          </button>
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
